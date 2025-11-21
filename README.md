**deprecated**: see https://github.com/fiatjaf/nak instead (the `git` subcommand).

---

# gitstr

Send and receive git patches over Nostr, using [NIP-34](https://github.com/nostr-protocol/nips/pull/997).

## How to install

Do `go install github.com/fiatjaf/gitstr/cmd/git-str@latest` if you have Go or [download a binary](https://github.com/fiatjaf/gitstr/releases).

### Confirm the Installation Location

```bash
ls $HOME/go/bin
```

If git-str is in $HOME/go/bin, but it’s still not found globally, add $HOME/go/bin to your PATH:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
```

```bash
source ~/.bashrc
```

Now try again:

```bash
git-str --help
```

or

```bash
git str
```

## How to receive patches

If you want to receive patches in our repo, call `git str init -r <relay> [-r <relay>...]`, this will ask you a bunch of questions (you can also answer them using flags and not be asked, see `git str init --help`) and then it will announce your repository to the relays specified with `-r`.

After someone has sent you a patch you'll be able to call `git str download` and fetch all patches. They will be stored in the `.git/str/patches/` directory. You can also pass arguments to `git str download`, like an `nevent1...` code or a `npub1...` code, to download only patches narrowed by these arguments.

After that you can call `git am -i <patch-file>` to apply the patch.

## How to send patches

First you need to know the `naddr1...` code that corresponds to the target upstream repository you're sending the patch to. Until someone makes an explorer of git repositories or something like that, you'll have to get that manually from the repository owner.

Then call `git send <commit>` (you can use `HEAD^` for the last commit and other git tricks here). You'll be asked some questions (which you can also answer with flags, see `git str send --help`) and the patch will be sent. You can also give a path to a patch file generated with `git format-patch` too instead.

### Sending patches to repositories that haven't announced themselves

You can pass `--dangling` to `git str send` and that will happen. Later anyone can download that patch by specifying its `nevent1` code on `git str download <nevent1...>`.

## Contributing to this repository

Send your patches to `naddr1qqrxw6t5wd68yqg5waehxw309aex2mrp0yhxgctdw4eju6t0qyt8wumn8ghj7un9d3shjtnwdaehgu3wvfskueqpzemhxue69uhhyetvv9ujuurjd9kkzmpwdejhgq3q80cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsxpqqqpmejeaalw2`.
