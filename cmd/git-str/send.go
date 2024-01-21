package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fiatjaf/gitstr/git"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var send = &cli.Command{
	Name:        "send",
	Usage:       "",
	Description: "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "sec",
			Usage:       "secret key to sign the event, as hex or nsec",
			DefaultText: "the key '1'",
			Value:       "0000000000000000000000000000000000000000000000000000000000000001",
		},
		&cli.BoolFlag{
			Name:  "prompt-sec",
			Usage: "prompt the user to paste a hex or nsec with which to sign the event",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		// gather the secret key
		sec, err := gatherSecretKeyFromArguments(c)
		if err != nil {
			return err
		}

		patch, err := git.Run("format-patch", "--stdout", c.Args().First())
		if err != nil {
			return fmt.Errorf("error getting patch: %w", err)
		}

		relays, err := git.GetRelays(nil)
		if err != nil {
			return err
		}

		evt := nostr.Event{
			CreatedAt: nostr.Now(),
			Kind:      PatchKind,
			Tags:      nostr.Tags{},
			Content:   patch,
		}

		err = evt.Sign(sec)
		if err != nil {
			return fmt.Errorf("error signing message: %w", err)
		}
		fmt.Fprintf(os.Stderr, evt.String())

		goodRelays := make([]string, 0, len(relays))
		for _, r := range relays {
			relay, err := pool.EnsureRelay(r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to connect to '%s': %s", r, err)
				continue
			}
			if err := relay.Publish(ctx, evt); err != nil {
				fmt.Fprintf(os.Stderr, "failed to publish to '%s': %s", r, err)
				continue
			}
			goodRelays = append(goodRelays, relay.URL)
		}
		if len(goodRelays) == 0 {
			return fmt.Errorf("event not published")
		}

		code, _ := nip19.EncodeEvent(evt.GetID(), goodRelays, evt.PubKey)
		fmt.Println(code)

		return nil
	},
}
