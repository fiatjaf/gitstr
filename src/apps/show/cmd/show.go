package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/npub1zenn0/nostr-git-cli/src/internal/git"
)

func Show(relays []string, hashtag string, userPubkey string, eventID string) (string, error) {
	relays, err := git.GetRelays(relays)
	if err != nil {
		return "", fmt.Errorf("error in relays: %w", err)
	}

	hashtag, err = git.GetHashtag(hashtag)
	if err != nil {
		return "", fmt.Errorf("error in hashtag: %w", err)
	}

	evts := queryAll(relays, hashtag, userPubkey, eventID)

	patches := make([]string, 0)
	for _, e := range evts {
		patches = append(patches, e.Content)
	}

	return strings.Join(patches, "\n\n"), nil
}

func queryAll(
	relays []string,
	hashtag string,
	userPubkey string,
	eventID string,
) []*nostr.Event {
	allEvts := make([]*nostr.Event, 0)
	for _, r := range relays {
		evts, err := query(r, hashtag, userPubkey, eventID)
		if err != nil {
			log.Printf("failed query %v: %v\n", r, err)
		}
		allEvts = append(allEvts, evts...)
	}
	return allEvts
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
