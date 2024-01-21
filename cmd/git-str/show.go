package main

import (
	"context"

	"github.com/fiatjaf/gitstr"
	"github.com/urfave/cli/v3"
)

var show = &cli.Command{
	Name:        "show",
	Usage:       "",
	Description: "",
	Flags: []cli.Flag{
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
	Action: func(ctx context.Context, c *cli.Command) error {
		return gitstr.Show(ctx, pool, c.StringSlice("relay"), c.String("hashtag"), c.String("user"), c.String("event"))
	},
}
