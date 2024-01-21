package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bgentry/speakeasy"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

func isPiped() bool {
	stat, _ := os.Stdin.Stat()
	return stat.Mode()&os.ModeCharDevice == 0
}

func gatherSecretKeyFromArguments(c *cli.Command) (string, error) {
	sec := c.String("sec")
	if c.Bool("prompt-sec") {
		if isPiped() {
			return "", fmt.Errorf("can't prompt for a secret key when processing data from a pipe, try again without --prompt-sec")
		}
		var err error
		sec, err = speakeasy.FAsk(os.Stderr, "type your secret key as nsec or hex: ")
		if err != nil {
			return "", fmt.Errorf("failed to get secret key: %w", err)
		}
	}
	if strings.HasPrefix(sec, "nsec1") {
		_, hex, err := nip19.Decode(sec)
		if err != nil {
			return "", fmt.Errorf("invalid nsec: %w", err)
		}
		sec = hex.(string)
	}
	if len(sec) > 64 {
		return "", fmt.Errorf("invalid secret key: too large")
	}
	sec = strings.Repeat("0", 64-len(sec)) + sec // left-pad
	if ok := nostr.IsValid32ByteHex(sec); !ok {
		return "", fmt.Errorf("invalid secret key")
	}

	return sec, nil
}
