# go-git-nostr

Send and receive git patches over nostr.

## Install

Download latest binaries from the releases page. https://github.com/npub1zenn0/go-git-nostr/releases

```sh
$ # You'll have to fix the version                                                    or show
$ wget https://github.com/npub1zenn0/go-git-nostr/releases/download/v<version>/git-nostr-send-v<version>-linux-amd64.tar.gz
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
$ # outputs all patches for project "nostr-git-cli".
$ git show-nostr -t nostr-git-cli -r wss://nos.lol # override relays to just wss://nos.lol

$ # Send a new patch in
$ git send-nostr --dry-run HEAD -t nostr-git-cli -r wss://nos.lol -r wss://example.com

$ # Apply a specific patch.
$ git show-nostr -e "<nostr_event_id>" -t nostr-git-cli -r wss://nos.lol | git am
```

See `git {show,send}-nostr --help` for more.

## Prior art

http://git.jb55.com/git-nostr-tools/file/README.txt.html
