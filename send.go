package gitstr

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var send = &cli.Command{
	Name:        "send",
	UsageText:   "git str send <commit or patch-file>",
	Description: "",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:    "sec",
			Usage:   "secret key to sign the patch, as hex or nsec, or bunker:// URL, or a NIP-46-powered name@domain",
			Aliases: []string{"connect"},
		},
		&cli.StringFlag{
			Name:    "to",
			Aliases: []string{"a", "repository"},
			Usage:   "repository reference, as an naddr1... code",
		},
		&cli.StringSliceFlag{
			Name:  "cc",
			Usage: "npub, hex or nprofile to mention in the event",
		},
		&cli.BoolFlag{
			Name:  "annotate",
			Usage: "specify this to submit patches without having a target repository -- anyone can fetch those later and apply wherever they want",
		},
		&cli.BoolFlag{
			Name:  "dangling",
			Usage: "specify this to submit patches without having a target repository -- anyone can fetch those later and apply wherever they want",
		},
		&cli.StringFlag{
			Name:    "in-reply-to",
			Aliases: []string{"e"},
			Usage:   "reply to another git event, as an nevent1... or hex code",
		},
		&cli.StringSliceFlag{
			Name:    "relay",
			Aliases: []string{"r"},
			Usage:   "extra relays to search for the target repository in and to publish the patch to",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "do not ask for confirmation before publishing",
		},
	}, gitFormatPatchFlags...),
	Action: func(ctx context.Context, c *cli.Command) error {
		// git-format-patch extra flags that will be handled directly to it
		gitFormatPatchArgs := []string{"format-patch", "--stdout"}
		for _, fd := range gitFormatPatchFlags {
			if fd.IsSet() {
				switch flag := fd.(type) {
				case *cli.StringFlag:
					gitFormatPatchArgs = append(gitFormatPatchArgs, "--"+flag.Name+"="+c.String(flag.Name))
				case *cli.BoolFlag:
					gitFormatPatchArgs = append(gitFormatPatchArgs, "--"+flag.Name)
				}
			}
		}

		// commit or file
		patches := make([]string, 0, 10)
		for _, arg := range c.Args().Slice() {
			if arg == "" {
				return fmt.Errorf("no commit or patch file specified")
			}
			if contents, err := os.ReadFile(arg); err != nil && !os.IsNotExist(err) {
				// it's a file
				return fmt.Errorf("error reading file '%s': %w", arg, err)
			} else if os.IsNotExist(err) {
				// it's a git reference
				args := make([]string, len(gitFormatPatchArgs)+1)
				copy(args, gitFormatPatchArgs)
				args[len(gitFormatPatchArgs)] = arg
				out, err := git(args...)
				if err != nil {
					return fmt.Errorf("error getting patch: %w", err)
				}

				// split multiple patches into separate strings
				for _, patch := range strings.Split(out, "\n\nFrom ") {
					patches = append(patches, "From "+patch)
				}
			} else {
				patches = append(patches, string(contents))
			}
		}

		patches = filterSlice(patches, func(v string) bool { return v != "" })
		if len(patches) == 0 {
			return fmt.Errorf("couldn't get any patches for %v", c.Args().Slice())
		}

		// create the events
		events := make([]*nostr.Event, len(patches))
		for i := range patches {
			events[i] = &nostr.Event{
				CreatedAt: nostr.Now(),
				Kind:      PatchKind,
				Tags: nostr.Tags{
					nostr.Tag{"alt", "a git patch"},
					nostr.Tag{"t", "root"},
				},
			}
		}

		// get metadata and apply it to events
		patchRelays, err := getAndApplyTargetRepository(ctx, c, events, c.StringSlice("relay"))
		if err != nil {
			return err
		}
		threadRelays, err := getAndApplyTargetThread(ctx, c, events)
		if err != nil {
			return err
		}
		mentionRelays, err := getAndApplyTargetMentions(ctx, c, events)
		if err != nil {
			return err
		}

		// check if there are relays available
		targetRelays := concatSlices(patchRelays, threadRelays, mentionRelays, c.StringSlice("relay"))
		if len(targetRelays) == 0 {
			return fmt.Errorf("got no relays to publish to, you can specify one with --relay/-r")
		}

		// possibly annotate and assign patch content to events
		for i, patch := range patches {
			if c.Bool("annotate") {
				var err error
				events[i].Content, err = edit(patch)
				if err != nil {
					return fmt.Errorf("error annotating patch: %w", err)
				}
			} else {
				events[i].Content = patch
			}
		}

		// gather the secret key
		bunker, sec, isEncrypted, err := gatherSecretKeyOrBunker(ctx, c)
		if err != nil {
			return err
		}
		if isEncrypted {
			sec, err = promptDecrypt(sec)
			if err != nil {
				return err
			}
		}

		// publish all the patches
		for _, evt := range events {
			if bunker != nil {
				logf(color.YellowString("signing event with bunker..."))
				err = bunker.SignEvent(ctx, evt)
				if err != nil {
					return fmt.Errorf("error signing event with bunker: %w", err)
				}
			} else {
				err = evt.Sign(sec)
				if err != nil {
					return fmt.Errorf("error signing event with key: %w", err)
				}
			}

			goodRelays := make([]string, 0, len(targetRelays))
			logf("\n%s", sprintPatch(evt))
			if confirm("proceed to publish the event? ") {
				for _, r := range targetRelays {
					relay, err := pool.EnsureRelay(r)
					if err != nil {
						logf("failed to connect to '%s': %s\n", r, err)
						continue
					}
					if err := relay.Publish(ctx, *evt); err != nil {
						logf("failed to publish to '%s': %s\n", r, err)
						continue
					}
					goodRelays = append(goodRelays, relay.URL)
				}
			}
			if len(goodRelays) == 0 {
				fmt.Println(evt)
				logf(color.RedString("didn't publish the event\n"))
				continue
			}

			code, _ := nip19.EncodeEvent(evt.GetID(), goodRelays, evt.PubKey)
			fmt.Println(code)
		}

		return nil
	},
}

func getAndApplyTargetRepository(
	ctx context.Context,
	c *cli.Command,
	evts []*nostr.Event,
	extraRelays []string,
) (patchRelays []string, err error) {
	if c.Bool("dangling") {
		logf("this patch won't target any specific repository")
		return nil, nil
	}

	target := c.String("to")
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

	logf("%s %s\n%s\n", color.YellowString("found upstream repository"),
		target, sprintRepository(repo.Event))

	if stored != target {
		if confirm("store it as your main upstream target? ") {
			git("config", "--local", "str.upstream", target)
		}
	}

	for _, tag := range repo.Event.Tags.GetAll([]string{"patches", ""}) {
		patchRelays = append(patchRelays, tag[1:]...)
	}
	for _, tag := range repo.Event.Tags.GetAll([]string{"relays", ""}) {
		patchRelays = append(patchRelays, tag[1:]...)
	}

	for _, evt := range evts {
		evt.Tags = append(evt.Tags,
			nostr.Tag{
				"a",
				fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier),
				repo.Relay.URL,
			},
			nostr.Tag{"p", ep.PublicKey},
		)
	}

	return patchRelays, nil
}

func getAndApplyTargetThread(
	ctx context.Context,
	c *cli.Command,
	evts []*nostr.Event,
) (mentionRelays []string, err error) {
	target := c.String("in-reply-to")
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
		for _, evt := range evts {
			evt.Tags = append(evt.Tags, nostr.Tag{"e", target})
			evt.Tags = slices.DeleteFunc(evt.Tags, func(tag nostr.Tag) bool {
				return len(tag) >= 2 && tag[0] == "t" && tag[1] == "root"
			})
		}
	}

	// TODO: fetch user relays, fetch thread root, return related relays so we can submit the patch to those too
	return nil, nil
}

func getAndApplyTargetMentions(
	ctx context.Context,
	c *cli.Command,
	evts []*nostr.Event,
) (mentionRelays []string, err error) {
	for _, target := range c.StringSlice("cc") {
		prefix, data, err := nip19.Decode(target)
		if err == nil {
			switch v := data.(type) {
			case string:
				if prefix == "npub" {
					target = v
				}
			case nostr.ProfilePointer:
				target = v.PublicKey
				mentionRelays = append(mentionRelays, v.Relays...)
			}
		}
		target = strings.TrimSpace(target)

		if nostr.IsValid32ByteHex(target) {
			for _, evt := range evts {
				evt.Tags = append(evt.Tags, nostr.Tag{"p", target})
			}
		} else {
			return nil, fmt.Errorf("invalid mention '%s'", target)
		}
	}

	// TODO: fetch user relays, fetch thread root, return related relays so we can submit the patch to those too
	return nil, nil
}

var gitFormatPatchFlags = []cli.Flag{
	&cli.StringFlag{Name: "base", Hidden: true},
}
