package nostrgit

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fiatjaf/nostr-git-cli/git"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/samber/lo"
	"github.com/samber/lo/parallel"
)

func Show(relays []string, hashtag string, user string, eventID string) (string, error) {
	relays, err := git.GetRelays(relays)
	if err != nil {
		return "", fmt.Errorf("error in relays: %w", err)
	}

	hashtag, err = git.GetHashtag(hashtag)
	if err != nil {
		return "", fmt.Errorf("error in hashtag: %w", err)
	}

	pubkey, autoRelays, err := decodeUser(user)
	if err != nil {
		return "", err
	}

	evtID, evtRelays, err := decodeEventID(eventID)
	if err != nil {
		return "", err
	}

	// The nprofile/nevent included relays will probably always be useful enough.
	allRelays := append(relays, evtRelays...)
	allRelays = append(allRelays, autoRelays...)
	allRelays = lo.Uniq(allRelays)

	evts := queryAll(allRelays, hashtag, pubkey, evtID)

	patches := lo.Map(evts, func(e *nostr.Event, _ int) string {
		return e.Content
	})

	return strings.Join(patches, "\n\n"), nil
}

func decodeEventID(eventID string) (string, []string, error) {
	if !strings.HasPrefix(eventID, "nevent") {
		return eventID, nil, nil
	}
	prefix, nevent, err := nip19.Decode(eventID)
	if err != nil {
		return "", nil, fmt.Errorf("error decoding eventID: %w", err)
	}
	if prefix != "nevent" {
		return "", nil, fmt.Errorf("received event with unexpected prefix: %v", prefix)
	}
	evt := nevent.(nostr.EventPointer)
	return evt.ID, unsplit(evt.Relays), nil
}

func decodeUser(user string) (string, []string, error) {
	if !strings.HasPrefix(user, "npub") && !strings.HasPrefix(user, "nprofile") {
		// Assume it's already in pubkey hex format.
		return user, nil, nil
	}
	prefix, profile, err := nip19.Decode(user)
	if err != nil {
		return "", nil, fmt.Errorf("error decoding user: %w", err)
	}
	switch prefix {
	case "npub":
		return profile.(string), nil, nil

	case "nprofile":
		p := profile.(nostr.ProfilePointer)
		return p.PublicKey, unsplit(p.Relays), nil
	}
	return "", nil, fmt.Errorf("received pubkey with unexpected prefix: %v", prefix)
}

func queryAll(
	relays []string,
	hashtag string,
	userPubkey string,
	eventID string,
) []*nostr.Event {
	evts := parallel.Map(relays, func(r string, _ int) []*nostr.Event {
		evts, err := query(r, hashtag, userPubkey, eventID)
		if err != nil {
			log.Printf("failed query %v: %v\n", r, err)
			return nil
		}
		return evts
	})
	flatEvts := lo.Flatten(evts)
	return lo.UniqBy(flatEvts, func(e *nostr.Event) string {
		return e.ID
	})
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
		Limit:   limit,
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

func unsplit(arr []string) []string {
	return lo.FlatMap(arr, func(a string, _ int) []string {
		return strings.Split(a, ",")
	})
}
