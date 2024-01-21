package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/nbd-wtf/go-nostr"
)

func Run(cmd ...string) (string, error) {
	v, err := exec.Command("git", cmd...).Output()
	return strings.TrimSpace(string(v)), err
}

var (
	subjectRegex = regexp.MustCompile(`(?m)^Subject: (.*)$`)
	authorRegex  = regexp.MustCompile(`(?m)^From: (.*)$`)
)

func GetSecretKey(sec string) (string, error) {
	if sec == "" {
		_sec, err := Run("config", "str.secretkey")
		if err != nil {
			return "", fmt.Errorf("secret key not set. Use one of\n\t-s <key>\n\tgit config --global nostr.secretkey <key>\n%w", err)
		}
		sec = _sec
	}
	return strings.TrimSpace(sec), nil
}

func GetRelays(relays []string) ([]string, error) {
	if len(relays) == 0 {
		_relays, err := Run("config", "str.relays")
		relays = strings.Split(_relays, " ")
		if err != nil || len(relays) == 0 {
			return nil, fmt.Errorf("relay not set, not relaying. Use one of\n\t-r wss://relay.damus.io\n\tgit config --global nostr.relays wss://relay.damus.io\n%w", err)
		}
	}
	return relays, nil
}

func GetRepositoryID() string {
	id, err := Run("config", "str.id")
	if err != nil {
		return ""
	}
	return id
}

func GetRepositoryPublicKey() string {
	pk, _ := Run("config", "str.publickey")
	if nostr.IsValidPublicKey(pk) {
		return pk
	}
	return ""
}
