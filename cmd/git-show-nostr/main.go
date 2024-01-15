package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/fiatjaf/gitstr"
)

var CLI struct {
	Relay []string `short:"r" help:"Relay to broadcast to. Will use 'git config nostr.relays' by default.You can specify multiple times '-r wss://... -r wss://...'"`

	Hashtag string `short:"t" help:"Hashtag (e.g. repo name) to search for. Will use 'git config nostr.hashtag' by default."`

	User string `short:"p" help:"Show patches from particular user. nprofile/pubkey/npub."`

	EventID string `short:"e" help:"Show patch from particular event."`
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	default:
		patches, err := gitstr.Show(CLI.Relay, CLI.Hashtag, CLI.User, CLI.EventID)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(patches)
	}
}
