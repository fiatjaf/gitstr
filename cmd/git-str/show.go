package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fiatjaf/gitstr/git"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var show = &cli.Command{
	Name:        "show",
	Usage:       "",
	Description: "",
	Action: func(ctx context.Context, c *cli.Command) error {
		relays, err := git.GetRelays(c.StringSlice("relay"))
		if err != nil {
			return fmt.Errorf("error in relays: %w", err)
		}

		id := git.GetRepositoryID()
		if id == "" {
			return fmt.Errorf("no repository id given: %w", err)
		}

		pk := git.GetRepositoryPublicKey()
		if pk == "" {
			return fmt.Errorf("no repository id given: %w", err)
		}

		for _, arg := range c.Args().Slice() {
			fmt.Println("arg", arg)

			filter := nostr.Filter{
				Tags: nostr.TagMap{
					"a": []string{fmt.Sprintf("%d:%s:%s", RepoAnnouncementKind, pk, id)},
				},
			}

			prefix, data, err := nip19.Decode(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid argument '%s': %s", arg, err)
				continue
			}

			switch prefix {
			case "npub":
				filter.Authors = append(filter.Authors, data.(string))
			case "nprofile":
				pp := data.(nostr.ProfilePointer)
				filter.Authors = append(filter.Authors, pp.PublicKey)
				relays = append(relays, pp.Relays...)
			case "nevent":
				ep := data.(nostr.EventPointer)
				if ep.Kind != 0 && ep.Kind != PatchKind {
					fmt.Fprintf(os.Stderr, "invalid argument %s: expected an encoded kind %d or nothing", arg, PatchKind)
					continue
				}
				filter.IDs = append(filter.IDs, ep.ID)
				relays = append(relays, ep.Relays...)
			case "naddr":
				ep := data.(nostr.EntityPointer)
				if ep.Kind != RepoAnnouncementKind {
					fmt.Fprintf(os.Stderr, "invalid argument %s: expected an encoded kind %d", arg, RepoAnnouncementKind)
					continue
				}

				filter.Tags["a"] = []string{fmt.Sprintf("%d:%s:%s", RepoAnnouncementKind, ep.PublicKey, ep.Identifier)}
				filter.Authors = append(filter.Authors, ep.PublicKey)
				relays = append(relays, ep.Relays...)
			}

			for ie := range pool.SubManyEose(ctx, relays, nostr.Filters{filter}) {
				fmt.Println(ie.Event.Content)
			}
		}

		return nil
	},
}

func query(
	relay string,
	hashtag string,
	userPubkey string,
	eventID string,
) ([]*nostr.Event, error) {
	const limit = 20
	const kinds = 19691228
	const connTimeout = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
	defer cancel()
	conn, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		return nil, err
	}

	var authors []string
	if userPubkey != "" {
		authors = append(authors, userPubkey)
	}
	var ids []string
	if eventID != "" {
		ids = append(ids, eventID)
	}
	ctx, cancel = context.WithTimeout(context.Background(), connTimeout)
	defer cancel()
	evts, err := conn.QuerySync(ctx, nostr.Filter{
		Kinds:   []int{kinds},
		Authors: authors,
		Limit:   limit,
		IDs:     ids,
		Tags: nostr.TagMap{
			"t": []string{hashtag},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error in query: %w", err)
	}
	return evts, nil
}
