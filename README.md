# go-git-nostr

Send and receive git patches over nostr.

## Install

Download latest binaries from the releases page. https://github.com/npub1zenn0/go-git-nostr/releases

```sh
$ # You'll have to fix the version
$ VERSION=v0.0.0 wget "https://github.com/npub1zenn0/go-git-nostr/releases/download/$VERSION/git-nostr-{send,show}-$VERSION-linux-amd64.tar.gz"
$ tar -xzf <file>.tar.gz
```

Note that there are _two_ binaries in the release: `git-send-nostr`, and `git-show-nostr`. You'll want both, but they are [not in the same zip](https://github.com/wangyoucao577/go-release-action/pull/107).

```sh
$ git config --global nostr.relays "wss://nos.lol wss://relay.damus.io" # can have multiple, split by space
$ git config --global nostr.secretkey <hex_key>
```

## Usage

If you then have the binaries in your `$PATH`, you can then use them like so.

```sh
$ git config nostr.hashtag my-repo-name
$ git show-nostr -h
$ # outputs all patches for project "nostr-git-cli".
$ git show-nostr -t nostr-git-cli -r wss://nos.lol # override relays to just wss://nos.lol

$ # Send a new patch in
$ git send-nostr --dry-run HEAD -t nostr-git-cli -r wss://nos.lol -r wss://example.com

$ # Apply a specific patch.
$ git show-nostr -e "<nostr_event_id>" -t nostr-git-cli -r wss://nos.lol | git am
```

See `git {show,send}-nostr -h` for more.

```
Usage: git-send-nostr <commit>

Arguments:
  <commit>    Commit hash

Flags:
  -h, --help               Show context-sensitive help.
  -r, --relay=RELAY,...    Relay to broadcast to. Will use 'git config
                           nostr.relays' by default.You can specify multiple
                           times '-r wss://... -r wss://...'
  -d, --dry-run            Dry run. Just print event to stdout instead of
                           relaying.
  -s, --sec=STRING         Secret key
```

```
Usage: git-show-nostr

Flags:
  -h, --help               Show context-sensitive help.
  -r, --relay=RELAY,...    Relay to broadcast to. Will use 'git config
                           nostr.relays' by default.You can specify multiple
                           times '-r wss://... -r wss://...'
  -t, --hashtag=STRING     Hashtag (e.g. repo name) to search for. Will use 'git
                           config nostr.hashtag' by default.
  -p, --user=STRING        Show patches from particular user.
                           nprofile/pubkey/npub.
  -e, --event-id=STRING    Show patch from particular event.
```

## Prior art

http://git.jb55.com/git-nostr-tools/file/README.txt.html
