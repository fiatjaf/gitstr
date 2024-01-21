package main

import (
	"context"
	"fmt"

	"github.com/fiatjaf/gitstr"
	"github.com/urfave/cli/v3"
)

var send = &cli.Command{
	Name:        "send",
	Usage:       "",
	Description: "",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "secret-key",
			Aliases: []string{"sec"},
			Usage:   "Nostr secret key",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		id, err := gitstr.Send(c.Args().First(), c.StringSlice("relay"), c.String("secret-key"), false)
		if err != nil {
			return err
		}
		fmt.Println(id)
		return nil
	},
}
