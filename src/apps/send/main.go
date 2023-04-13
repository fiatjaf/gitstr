package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/npub1zenn0/nostr-git-cli/src/apps/send/cmd"
)

var CLI struct {
	Relay []string `short:"r" help:"Relay to broadcast to. Will use 'git config nostr.relays' by default.You can specify multiple times '-r wss://... -r wss://...'"`

	DryRun bool `short:"d" help:"Dry run. Just print event to stdout instead of relaying."`

	// tag?
	SecretKey string `short:"s" name:"sec" help:"Secret key" type:"string"`

	// type: can autocast?
	Commit string `arg:"" help:"Commit hash" type:"string"`
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "<commit>":
		id, err := cmd.Send(CLI.Commit, CLI.Relay, CLI.SecretKey, CLI.DryRun)
		if err != nil {
			log.Fatal(err)
		} else if !CLI.DryRun {
			fmt.Println(id)
		}
	default:
		log.Fatal("no such command")
	}
}
