package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func Run(cmd ...string) (string, error) {
	v, err := exec.Command("git", cmd...).Output()
	return strings.TrimSpace(string(v)), err
}

var subjectRegex = regexp.MustCompile(`(?m)^Subject: (.*)$`)
var authorRegex = regexp.MustCompile(`(?m)^From: (.*)$`)

// ExtractAuthorSubject from a git patch.
func ExtractAuthorSubject(patch string) (string, string, error) {
	subjectMatch := subjectRegex.FindStringSubmatch(patch)
	if len(subjectMatch) == 0 {
		return "", "", fmt.Errorf("error getting subject")
	}
	subject := subjectMatch[1]

	authorMatch := authorRegex.FindStringSubmatch(patch)
	if len(authorMatch) == 0 {
		return "", "", fmt.Errorf("error getting author")
	}
	author := authorMatch[1]
	return author, subject, nil
}

func GetSecKey(sec string) (string, error) {
	if sec == "" {
		_sec, err := Run("config", "nostr.secretkey")
		if err != nil {
			return "", fmt.Errorf("secret key not set. Use one of\n\t-s <key>\n\tgit config --global nostr.secretkey <key>\n%w", err)
		}
		sec = _sec
	}
	return strings.TrimSpace(sec), nil
}

func GetRelays(relays []string) ([]string, error) {
	if len(relays) == 0 {
		_relays, err := Run("config", "nostr.relays")
		relays = strings.Split(_relays, " ")
		if err != nil || len(relays) == 0 {
			return nil, fmt.Errorf("relay not set, not relaying. Use one of\n\t-r wss://relay.damus.io\n\tgit config --global nostr.relays wss://relay.damus.io\n%w", err)
		}
	}
	return relays, nil
}

func GetHashtag(hashtag string) (string, error) {
	if hashtag == "" {
		_hashtag, err := Run("config", "nostr.hashtag")
		if err != nil || _hashtag == "" {
			return "", fmt.Errorf("error getting hashtag: %w", err)
		}
		hashtag = _hashtag
	}
	return hashtag, nil
}
