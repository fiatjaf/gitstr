package gitstr

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip49"
	"github.com/urfave/cli/v3"
)

var subjectRegex = regexp.MustCompile(`(?m)^Subject: (.*)$`)

func isPiped() bool {
	stat, _ := os.Stdin.Stat()
	return stat.Mode()&os.ModeCharDevice == 0
}

func gatherSecretKey(c *cli.Command) (key string, encrypted bool, err error) {
	sec := c.String("sec")

	if sec == "" && !c.IsSet("sec") {
		sec, _ = git("config", "--local", "str.secretkey")
	}

	askToStore := false
	if sec == "" {
		sec, _ = ask("input secret key (hex, nsec or ncryptsec): ", "", func(answer string) bool {
			switch {
			case nostr.IsValid32ByteHex(answer):
				askToStore = true
				return false
			case strings.HasPrefix(answer, "nsec1"):
				askToStore = true
				return false
			case strings.HasPrefix(answer, "ncryptsec1"):
				// will always store encrypted keys
				return false
			default:
				return true
			}
		})
		if sec == "" {
			return "", false, fmt.Errorf("couldn't gather secret key")
		}
	}

	if strings.HasPrefix(sec, "ncryptsec1") {
		git("config", "--local", "str.secretkey", sec)
		return sec, true, nil
	} else {
		if strings.HasPrefix(sec, "nsec1") {
			_, hex, err := nip19.Decode(sec)
			if err != nil {
				return "", false, fmt.Errorf("invalid nsec: %w", err)
			}
			sec = hex.(string)
		}
		if ok := nostr.IsValid32ByteHex(sec); !ok {
			return "", false, fmt.Errorf("invalid secret key")
		}
	}

	if (askToStore && confirm("store the secret key on git config? ")) ||
		c.Bool("store-sec") {
		git("config", "--local", "str.secretkey", sec)
	}

	return sec, false, nil
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

func promptDecrypt(ncryptsec1 string) (string, error) {
	for i := 1; i < 4; i++ {
		var attemptStr string
		if i > 1 {
			attemptStr = fmt.Sprintf(" [%d/3]", i)
		}
		password, err := ask("type the password to decrypt your secret key"+attemptStr+": ", "", nil)
		if err != nil {
			return "", err
		}
		sec, err := nip49.Decrypt(ncryptsec1, password)
		if err != nil {
			continue
		}
		return sec, nil
	}
	return "", fmt.Errorf("couldn't decrypt private key")
}

func ask(msg string, defaultValue string, shouldAskAgain func(answer string) bool) (string, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 color.YellowString(msg),
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

func concatSlices[V any](slices ...[]V) []V {
	size := 0
	for _, ss := range slices {
		size += len(ss)
	}
	newSlice := make([]V, size)
	pos := 0
	for _, ss := range slices {
		copy(newSlice[pos:], ss)
		pos += len(ss)
	}
	return newSlice
}

func edit(initial string) (string, error) {
	editor := "vim"
	if s := os.Getenv("EDITOR"); s != "" {
		editor = s
	}
	// tmpfile
	f, err := os.CreateTemp("", "go-editor")
	if err != nil {
		return "", fmt.Errorf("creating tmpfile: %w", err)
	}
	defer os.Remove(f.Name())

	// write initial string to it
	if err := os.WriteFile(f.Name(), []byte(initial), 0644); err != nil {
		return "", fmt.Errorf("error writing to tmpfile '%s': %w", f.Name(), err)
	}

	// open editor
	cmd := exec.Command("sh", "-c", editor+" "+f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("executing '%s %s': %w", editor, f.Name(), err)
	}

	// read tmpfile
	b, err := os.ReadFile(f.Name())
	if err != nil {
		return "", fmt.Errorf("reading tmpfile '%s': %w", f.Name(), err)
	}

	return string(b), nil
}

func split(str string) []string {
	res := make([]string, 0, 5)
	for _, v := range strings.Split(str, " ") {
		for _, v := range strings.Split(v, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				res = append(res, v)
			}
		}
	}
	return res
}
