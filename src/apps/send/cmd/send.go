package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/npub1zenn0/nostr-git-cli/src/internal/git"
)

// Send a git patch to nostr relays.
func Send(hash string, relays []string, sec string, dryRun bool) (string, error) {
	patch, err := git.Run("format-patch", "--stdout", hash)
	if err != nil {
		return "", fmt.Errorf("error getting patch: %w", err)
	}
	author, subject, err := git.ExtractAuthorSubject(patch)
	if err != nil {
		return "", err
	}

	relays, err = git.GetRelays(relays)
	if err != nil {
		return "", err
	}

	sec, err = git.GetSecKey(sec)
	if err != nil {
		return "", err
	}

	evt := mkEvent(patch, author, subject)

	err = evt.Sign(sec)
	if err != nil {
		return "", fmt.Errorf("error signing message: %w", err)
	}

	if dryRun {
		evtJson, _ := evt.MarshalJSON()
		fmt.Printf("%v\n", string(evtJson))
		log.Println("this was a dry run")
		return "", nil
	}

	for _, relay := range relays {
		err = publish(relay, evt)
		if err != nil {
			log.Println(fmt.Errorf("warning: %w", err))
		}
	}

	return evt.ID, nil
}

func publish(relay string, evt nostr.Event) error {
	const connTimeout = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
	defer cancel()

	conn, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		return fmt.Errorf("error connecting to relay %v: %w", relay, err)
	}
	status, err := conn.Publish(conn.ConnectionContext, evt)
	if err != nil {
		return fmt.Errorf("error publishing (relay=%v;status=%v): %w", relay, status, err)
	}
	return nil
}

func mkEvent(content string, author string, subject string) nostr.Event {
	const kind = 19691228
	hashtag, _ := git.Run("config", "nostr.hashtag")

	tags := nostr.Tags{
		nostr.Tag{"author", author},
		nostr.Tag{"subject", subject},
	}
	if hashtag != "" {
		tags = append(tags, nostr.Tag{"t", hashtag})
	}
	return nostr.Event{
		CreatedAt: time.Now().UTC(),
		Kind:      kind,
		Tags:      tags,
		Content:   content,
	}
}
