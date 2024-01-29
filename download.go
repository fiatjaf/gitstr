package gitstr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var download = &cli.Command{
	Name:        "download",
	Usage:       "",
	Description: "",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "relay",
			Aliases: []string{"r"},
		},
		&cli.IntFlag{
			Name:    "limit",
			Aliases: []string{"l"},
			Value:   15,
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		id := getRepositoryID()
		if id == "" {
			return fmt.Errorf("no repository id found in `config str.id`")
		}

		pk := getRepositoryPublicKey()
		if pk == "" {
			return fmt.Errorf("no repository pubkey found in `git config str.publickey`")
		}

		limit := c.Int("limit")
		relays := append(getPatchRelays(), c.StringSlice("relay")...)

		// patches we will try to browse -- if given an author we try to get all their patches targeting this repo,
		// if given an event pointer we will try to fetch that patch specifically and so on, if given nothing we will
		// list the latest patches available to this repository
		items := c.Args().Slice()
		if len(items) == 0 {
			items = []string{""}
		}

		for _, arg := range items {
			filter := nostr.Filter{
				Limit: int(limit),
				Kinds: []int{PatchKind},
				Tags:  nostr.TagMap{},
			}

			relays := slices.Clone(relays)

			if arg != "" {
				prefix, data, err := nip19.Decode(arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid argument '%s': %s\n", arg, err)
					continue
				}

				switch prefix {
				case "npub":
					filter.Authors = append(filter.Authors, data.(string))
					filter.Tags["a"] = []string{fmt.Sprintf("%d:%s:%s", RepoAnnouncementKind, pk, id)}
				case "nprofile":
					pp := data.(nostr.ProfilePointer)
					filter.Authors = append(filter.Authors, pp.PublicKey)
					filter.Tags["a"] = []string{fmt.Sprintf("%d:%s:%s", RepoAnnouncementKind, pk, id)}
					relays = append(relays, pp.Relays...)
				case "nevent":
					ep := data.(nostr.EventPointer)
					if ep.Kind != 0 && ep.Kind != PatchKind {
						fmt.Fprintf(os.Stderr, "invalid argument %s: expected an encoded kind %d or nothing\n", arg, PatchKind)
						continue
					}
					filter.IDs = append(filter.IDs, ep.ID)
					relays = append(relays, ep.Relays...)
				case "naddr":
					ep := data.(nostr.EntityPointer)
					if ep.Kind != RepoAnnouncementKind {
						fmt.Fprintf(os.Stderr, "invalid argument %s: expected an encoded kind %d\n", arg, RepoAnnouncementKind)
						continue
					}

					filter.Tags["a"] = []string{fmt.Sprintf("%d:%s:%s", RepoAnnouncementKind, ep.PublicKey, ep.Identifier)}
					filter.Authors = append(filter.Authors, ep.PublicKey)
					relays = append(relays, ep.Relays...)
				}
			}

			gitRoot, err := git("rev-parse", "--show-toplevel")
			base := filepath.Join(gitRoot, ".git/str/patches")
			if err != nil {
				return fmt.Errorf("failed to find git root: %w", err)
			} else if err := os.MkdirAll(base, 0755); err != nil {
				return fmt.Errorf("failed to create .git/str directory")
			}

			for ie := range pool.SubManyEose(ctx, relays, nostr.Filters{filter}) {
				nevent, _ := nip19.EncodeEvent(ie.ID, nil, "")
				npub, _ := nip19.EncodePublicKey(ie.PubKey)
				subjectMatch := subjectRegex.FindStringSubmatch(ie.Event.Content)
				if len(subjectMatch) == 0 {
					continue
				}
				subject := subjectMatch[1]
				subject = strings.ReplaceAll(strings.ReplaceAll(subject, "/", "_"), "'", "")
				fileName := base + "/" + fmt.Sprintf("%s [%s] %s",
					ie.CreatedAt.Time().Format(time.DateOnly), nevent[65:], subject)
				if _, err := os.Stat(fileName); os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "- downloaded patch %s from %s, saved as '%s'\n",
						ie.Event.ID, npub, color.New(color.Underline).Sprint(fileName))
					if err := os.WriteFile(fileName, []byte(ie.Event.Content), 0644); err != nil {
						return fmt.Errorf("failed to write '%s': %w", fileName, err)
					}
					os.Chtimes(fileName, time.Time{}, ie.Event.CreatedAt.Time())
				}
			}
		}

		return nil
	},
}
