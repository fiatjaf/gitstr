package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var send = &cli.Command{
	Name:        "send",
	UsageText:   "git str send <commit>",
	Description: "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "sec",
			Usage: "secret key to sign the patch, as hex or nsec",
		},
		&cli.BoolFlag{
			Name:  "store-sec",
			Usage: "if we should save the secret key to git config --local",
		},
		&cli.StringFlag{
			Name:    "repository",
			Aliases: []string{"a"},
			Usage:   "repository reference, as an naddr1... code",
		},
		&cli.StringFlag{
			Name:    "in-reply-to",
			Aliases: []string{"e"},
			Usage:   "reply to another git event, as an nevent1... or hex code",
		},
		&cli.StringSliceFlag{
			Name:    "relay",
			Aliases: []string{"r"},
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "do not ask for confirmation before publishing",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		// commit
		commit := c.Args().First()
		if commit == "" {
			return fmt.Errorf("no commit specified")
		}

		patch, err := git("format-patch", "--stdout", commit)
		if err != nil {
			return fmt.Errorf("error getting patch: %w", err)
		}
		if patch == "" {
			return fmt.Errorf("the patch for '%s' is empty", commit)
		}

		evt := nostr.Event{
			CreatedAt: nostr.Now(),
			Kind:      PatchKind,
			Tags: nostr.Tags{
				nostr.Tag{"alt", "a git patch"},
			},
			Content: patch,
		}

		// target repository
		patchRelays, err := getAndApplyTargetRepository(ctx, c, &evt, c.StringSlice("relay"))
		if err != nil {
			return err
		}

		threadRelays, err := getAndApplyTargetThread(ctx, c, &evt)
		if err != nil {
			return err
		}

		// gather the secret key
		sec, err := gatherSecretKey(c)
		if err != nil {
			return err
		}

		err = evt.Sign(sec)
		if err != nil {
			return fmt.Errorf("error signing message: %w", err)
		}

		targetRelays := append(append(patchRelays, threadRelays...), c.StringSlice("relay")...)
		goodRelays := make([]string, 0, len(targetRelays))

		fmt.Fprintf(os.Stderr, "\nwill publish event\n%s", sprintPatch(evt))
		if confirm("proceed to publish the event? ") {
			for _, r := range targetRelays {
				relay, err := pool.EnsureRelay(r)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to connect to '%s': %s\n", r, err)
					continue
				}
				if err := relay.Publish(ctx, evt); err != nil {
					fmt.Fprintf(os.Stderr, "failed to publish to '%s': %s\n", r, err)
					continue
				}
				goodRelays = append(goodRelays, relay.URL)
			}
		}
		if len(goodRelays) == 0 {
			fmt.Println(evt)
			return fmt.Errorf("didn't publish the event")
		}

		code, _ := nip19.EncodeEvent(evt.GetID(), goodRelays, evt.PubKey)
		fmt.Println(code)

		return nil
	},
}

func getAndApplyTargetRepository(
	ctx context.Context,
	c *cli.Command,
	evt *nostr.Event,
	extraRelays []string,
) (patchRelays []string, err error) {
	target := c.String("repository")
	var stored string
	if target == "" {
		target, _ = git("config", "--local", "str.upstream")
		stored = target
	}

	if target == "" {
		var err error
		target, err = ask("repository to target with this (naddr1...): ", "", func(answer string) bool {
			prefix, _, err := nip19.Decode(answer)
			if err != nil {
				return true
			}
			if prefix != "naddr" {
				return true
			}
			return false
		})
		if err != nil {
			return nil, err
		}
	}

	_, data, _ := nip19.Decode(target)
	ep, ok := data.(nostr.EntityPointer)
	if !ok {
		return nil, fmt.Errorf("invalid target '%s'", target)
	}
	if ep.Kind != RepoAnnouncementKind {
		return nil, fmt.Errorf("invalid kind %d, expected %d", ep.Kind, RepoAnnouncementKind)
	}

	filter := nostr.Filter{
		Tags:    nostr.TagMap{"d": {ep.Identifier}},
		Authors: []string{ep.PublicKey},
		Kinds:   []int{ep.Kind},
	}

	repo := pool.QuerySingle(ctx, append(ep.Relays, extraRelays...), filter)
	if repo == nil {
		return nil, fmt.Errorf("couldn't find event for %s", filter)
	}

	fmt.Fprintf(os.Stderr, "found upstream repository %s\n%s\n", target, sprintRepository(repo.Event))

	if stored != target {
		if confirm("store it as your main upstream target? ") {
			git("config", "--local", "str.upstream", target)
		}
	}

	for _, tag := range repo.Event.Tags.GetAll([]string{"patches", ""}) {
		patchRelays = append(patchRelays, tag[1])
	}

	evt.Tags = append(evt.Tags, nostr.Tag{
		"a",
		fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier),
		repo.Relay.URL,
	})

	return patchRelays, nil
}

func getAndApplyTargetThread(
	ctx context.Context,
	c *cli.Command,
	evt *nostr.Event,
) (patchRelays []string, err error) {
	target := c.String("in-reply-to")
	if target == "" {
		var err error
		target, err = ask("reference a thread? (nevent or hex) (leave blank if not): ", "", func(answer string) bool {
			if answer == "" {
				return false
			}
			prefix, _, _ := nip19.Decode(answer)
			if prefix != "nevent" {
				if !nostr.IsValid32ByteHex(answer) {
					return true
				}
			}
			return false
		})
		if err != nil {
			return nil, err
		}
	}

	if target != "" {
		_, data, _ := nip19.Decode(target)
		ep, ok := data.(nostr.EventPointer)
		if ok {
			target = ep.ID
		}
	}

	target = strings.TrimSpace(target)

	if target != "" {
		if nostr.IsValid32ByteHex(target) {
			return nil, fmt.Errorf("invalid target thread id")
		}
		evt.Tags = append(evt.Tags, nostr.Tag{"e", target})
	}

	// TODO: fetch user relays, fetch thread root, return related relays so we can submit the patch to those too
	return nil, nil
}
