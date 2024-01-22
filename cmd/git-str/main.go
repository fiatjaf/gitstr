package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nbd-wtf/go-nostr"
	"github.com/urfave/cli/v3"
)

var pool *nostr.SimplePool

const (
	RepoAnnouncementKind = 30617
	PatchKind            = 1617
	IssueKind            = 1621
	ReplyKind            = 1622
)

var app = &cli.Command{
	Name:                   "git str",
	Suggest:                true,
	UseShortOptionHandling: true,
	Before: func(ctx context.Context, c *cli.Command) error {
		pool = nostr.NewSimplePool(ctx)
		return nil
	},
	Commands: []*cli.Command{
		initRepo,
		download,
		send,
	},
}

func main() {
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
