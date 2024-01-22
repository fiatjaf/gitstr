package gitstr

import (
	"context"

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

var App = &cli.Command{
	Name:                   "git str",
	Description:            "NIP-34 git nostr helper",
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
