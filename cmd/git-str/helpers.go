package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/urfave/cli/v3"
)

var subjectRegex = regexp.MustCompile(`(?m)^Subject: (.*)$`)

func isPiped() bool {
	stat, _ := os.Stdin.Stat()
	return stat.Mode()&os.ModeCharDevice == 0
}

func gatherSecretKey(c *cli.Command) (string, error) {
	sec := c.String("sec")

	if sec == "" && !c.IsSet("sec") {
		sec, _ = git("config", "--local", "str.secretkey")
	}

	askToStore := false
	if sec == "" {
		askToStore = true
		sec, _ = ask("input secret key: ", "", func(answer string) bool {
			return !nostr.IsValid32ByteHex(answer)
		})
		if sec == "" {
			return "", fmt.Errorf("couldn't gather secret key")
		}
	}

	if strings.HasPrefix(sec, "nsec1") {
		_, hex, err := nip19.Decode(sec)
		if err != nil {
			return "", fmt.Errorf("invalid nsec: %w", err)
		}
		sec = hex.(string)
	}

	if ok := nostr.IsValid32ByteHex(sec); !ok {
		return "", fmt.Errorf("invalid secret key")
	}

	if (askToStore && confirm("store the secret key on git config? ")) ||
		c.Bool("store-sec") {
		git("config", "--local", "str.secretkey", sec)
	}

	return sec, nil
}

func getPatchRelays() []string {
	str, _ := git("config", "str.patches-relay")
	spl := strings.Split(str, " ")
	res := make([]string, 0, len(spl))
	for _, url := range spl {
		if url != "" {
			res = append(res, url)
		}
	}
	return res
}

func getRepositoryID() string {
	id, err := git("config", "--local", "str.id")
	if err != nil {
		return ""
	}
	return id
}

func getRepositoryPublicKey() string {
	pk, _ := git("config", "str.publickey")
	if nostr.IsValidPublicKey(pk) {
		return pk
	}
	return ""
}

func git(cmd ...string) (string, error) {
	v, err := exec.Command("git", cmd...).Output()
	return strings.TrimSpace(string(v)), err
}

func gitWithStdin(stdin string, cmd ...string) (string, error) {
	command := exec.Command("git", cmd...)
	command.Stdin = strings.NewReader(stdin)
	v, err := command.Output()
	return strings.TrimSpace(string(v)), err
}

func sprintRepository(repo *nostr.Event) string {
	res := ""
	npub, _ := nip19.EncodePublicKey(repo.PubKey)
	res += "\nauthor: " + npub
	res += "\nid: " + (*repo.Tags.GetFirst([]string{"d", ""}))[1]
	res += "\n"
	// TODO: more stuff
	return res
}

func sprintPatch(patch nostr.Event) string {
	res := ""
	npub, _ := nip19.EncodePublicKey(patch.PubKey)
	target := strings.Split((*patch.Tags.GetFirst([]string{"a", ""}))[1], ":")
	targetId := target[2]
	targetNpub, _ := nip19.EncodePublicKey(target[1])

	res += "\nid: " + patch.ID
	res += "\nauthor: " + npub
	res += "\ntarget repo: " + targetId
	res += "\ntarget author: " + targetNpub
	res += "\n\n" + patch.Content
	// TODO: colors
	return res
}

func humanDate(createdAt nostr.Timestamp) string {
	ts := createdAt.Time()
	now := time.Now()
	if ts.Before(now.AddDate(0, -9, 0)) {
		return ts.UTC().Format("02 Jan 2006")
	} else if ts.Before(now.AddDate(0, 0, -6)) {
		return ts.UTC().Format("Jan _2")
	} else {
		return ts.UTC().Format("Mon, Jan _2 15:04 UTC")
	}
}

func confirm(msg string) bool {
	var res bool
	ask(msg+"(y/n) ", "", func(answer string) bool {
		switch answer {
		case "y", "yes":
			res = true
			return false
		case "n", "no":
			res = false
			return false
		default:
			return true
		}
	})
	return res
}

func ask(msg string, defaultValue string, shouldAskAgain func(answer string) bool) (string, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 msg,
		InterruptPrompt:        "^C",
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		return "", err
	}

	rl.WriteStdin([]byte(defaultValue))
	for {
		answer, err := rl.Readline()
		if err != nil {
			return "", err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if shouldAskAgain != nil && shouldAskAgain(answer) {
			continue
		}
		return answer, err
	}
}
