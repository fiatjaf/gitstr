package gitstr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var initRepo = &cli.Command{
	Name:        "init",
	Usage:       "",
	Description: "",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "relay",
			Aliases: []string{"r"},
		},
		&cli.StringFlag{
			Name:    "sec",
			Usage:   "secret key to sign the repository announcement, as hex or nsec, or bunker:// URL, or a NIP-46-powered name@domain",
			Aliases: []string{"connect"},
		},
		&cli.StringFlag{
			Name:  "id",
			Usage: "repository id",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "repository name",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "repository brief description",
		},
		&cli.StringFlag{
			Name:  "patches-relay",
			Usage: "relay that will be used to read patches",
		},
		&cli.StringFlag{
			Name:  "clone-url",
			Usage: "URL through which this repository can cloned",
		},
		&cli.StringFlag{
			Name:  "web-url",
			Usage: "URL through which this repository can be browsed on the web",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		evt := nostr.Event{
			CreatedAt: nostr.Now(),
			Kind:      RepoAnnouncementKind,
			Content:   "",
			Tags:      nostr.Tags{},
		}

		defaultId, _ := os.Getwd()
		defaultId = filepath.Base(defaultId)
		defaultClone, _ := git("remote", "get-url", "origin")
		defaultName := defaultId
		defaultWeb := ""
		if strings.HasPrefix(defaultClone, "http") {
			defaultWeb = defaultClone
		} else if strings.HasPrefix(defaultClone, "git@") {
			defaultWeb = "https://" + strings.Replace(defaultClone[4:], ":", "/", 1)
		}

		for _, prop := range []struct {
			name     string
			tag      string
			prompt   string
			deflt    string
			optional bool
			multi    bool
		}{
			{"id", "d", "specify the repository unique id (for this keypair)", defaultId, false, false},
			{"patches-relay", "relays", "specify relay URLs to watch for patches", "wss://relay.nostr.bg wss://nostr21.com wss://nostr.fmt.wiz.biz", false, true},
			{"clone-url", "clone", "specify the repository URL for git clone", defaultClone, false, true},
			{"name", "name", "specify the repository name", defaultName, true, false},
			{"description", "description", "specify the repository description", "", true, false},
			{"web-url", "web", "specify the repository URL for browsing on the web", defaultWeb, true, true},
		} {
			v := c.String(prop.name)
			if v == "" {
				v, _ = git("config", "--local", "str."+prop.name)
				if v == "" {
					v = prop.deflt
				}

				prompt := prop.prompt
				if prop.optional {
					prompt += " (optional)"
				}
				if prop.multi {
					prompt += "*"
				}

				var err error
				v, err = ask(prompt+": ", v, func(answer string) bool {
					if prop.optional {
						return false
					}
					return answer == ""
				})
				if err != nil {
					return err
				}
			}

			if v != "" {
				git("config", "--local", "str."+prop.name, v)
				tag := nostr.Tag{prop.tag}
				if prop.multi {
					manyV := split(v)
					tag = append(tag, manyV...)
				} else {
					tag = append(tag, v)
				}
				evt.Tags = append(evt.Tags, tag)
			} else if v == "" && !prop.optional {
				return fmt.Errorf("'%s' is mandatory", prop.name)
			}
		}

		bunker, sec, isEncrypted, err := gatherSecretKeyOrBunker(ctx, c)
		if err != nil {
			return fmt.Errorf("failed to get authentication data: %w", err)
		}

		if isEncrypted {
			sec, err = promptDecrypt(sec)
			if err != nil {
				return err
			}
		}

		if bunker != nil {
			err = bunker.SignEvent(ctx, &evt)
			if err != nil {
				return fmt.Errorf("error signing event with bunker: %w", err)
			}
		} else {
			err = evt.Sign(sec)
			if err != nil {
				return fmt.Errorf("error signing event with key: %w", err)
			}
		}

		git("config", "--local", "str.publickey", evt.PubKey)

		relays := c.StringSlice("relay")
		successRelays := make([]string, 0, len(relays))
		for _, r := range relays {
			logf("publishing to %s...", r)
			if relay, err := pool.EnsureRelay(r); err == nil {
				if err := relay.Publish(ctx, evt); err != nil {
					logf(" failed: %s\n", err)
				} else {
					logf("done\n")
					successRelays = append(successRelays, r)
				}
			} else {
				logf(" failed: %s\n", err)
			}
		}

		if len(successRelays) > 0 {
			tag := evt.Tags.GetFirst([]string{"d", ""})
			naddr, _ := nip19.EncodeEntity(evt.PubKey, RepoAnnouncementKind, (*tag)[1], successRelays)
			fmt.Println(naddr)
			return nil
		} else {
			fmt.Println(evt)
			return fmt.Errorf("couldn't publish the event to any relays, use -r or --relay to specify some relays")
		}
	},
}
