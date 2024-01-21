package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fiatjaf/gitstr"
	"github.com/nbd-wtf/go-nostr"
	"github.com/urfave/cli/v3"
)

var pool *nostr.SimplePool

var app = &cli.Command{
	Name: "git str show",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "relay",
			Aliases: []string{"r"},
			Usage:   "Relay to broadcast to. Will use 'git config nostr.relays' by default.You can specify multiple times '-r wss://... -r wss://...'",
		},
		&cli.StringFlag{
			Name:    "author",
			Aliases: []string{"p"},
			Usage:   "Show patches from particular user. nprofile/hex/npub.",
		},
		&cli.StringFlag{
			Name:    "event",
			Aliases: []string{"e"},
			Usage:   "Show patch from particular event. nevent/hex",
		},
	},
	Before: func(ctx context.Context, c *cli.Command) error {
		pool = nostr.NewSimplePool(ctx)
		return nil
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		return gitstr.Show(ctx, pool, c.StringSlice("relay"), c.String("hashtag"), c.String("user"), c.String("event"))
	},
}

func main() {
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
