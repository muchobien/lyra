# Repositories and Git

## Owners and names

Every repository belongs to an owner, a namespace such as `krosf`, and is named
`owner/repo` everywhere: `krosf/lyra` in the CLI, `/krosf/lyra` in the web UI,
`/api/repos/krosf/lyra` in the API, and `https://lyra.example/krosf/lyra.git` for Git.

- Owners are created with `lyra owner create` before their repositories, and can only be
  deleted once they have none. Managing owners needs an unrestricted admin token.
- Owners have no members: access comes from the tailnet login or a token, and a token can
  cover a whole owner with `owner/*`.
- Names are 1 to 100 letters, digits, `.`, `_` or `-`, starting with a letter or digit.
  Owners can't be `api`, `static`, `healthz`, `settings` or `v2`, and repositories can't end in `.git`.

## Choosing the repository in the CLI

Commands that act on one repository (`pr`, `issue`, `run`, `release`, `label`, `rule`,
`secret`, `variable`, `commit`, `tag`, `workflow`, `image list`) take it from, in order:

1. `-R owner/repo` or `--repo owner/repo`;
2. `LYRA_REPO`;
3. the git remotes of the working directory whose host is the Lyra server's. HTTPS, `ssh://` and
   `git@host:owner/repo` URLs all count. When they point at different repositories, `origin` wins;
   without an `origin` among them, the command asks for `--repo`.

## Git

```sh
git clone https://lyra.example/<owner>/<repo>.git
git -c http.extraHeader="Lyra-Agent: claude" push
```

Pushing needs `write`; cloning and fetching need `read`. Over HTTPS, use a token as the
password with any user name. Git prints what a push changed as `remote: lyra: …` lines.

## Go modules

`go get lyra.example/<owner>/<repo>` works: requests with `?go-get=1` for a repository or
any path inside it get a `go-import` tag pointing at the clone URL. Every request needs a
token, so keep the module out of the public proxy and checksum database and give Go and Git
the token through `~/.netrc`:

```sh
go env -w GOPRIVATE=lyra.example
printf 'machine lyra.example login token password %s\n' "$LYRA_TOKEN" >> ~/.netrc
```

## Archives

`https://lyra.example/<owner>/<repo>/archive/<ref>.tar.gz` and `…/<ref>.zip` download the
files at a branch, tag or commit, without history, inside a `<repo>-<ref>` directory (slashes
in the ref become `-`). They need `read`; the repository's Clone menu links both for the
ref on screen.

## Signatures

Register SSH signing keys with `lyra key add --signing [--agent NAME] key.pub`. Commits and
tags are reported as:

- `verified`: an SSH signature in the `git` namespace, made by a registered signing key over
  the exact signed content;
- `unverified`: signed, but by an unregistered or deleted key, in another namespace, over
  different content, or not an SSH signature;
- `unsigned`: no signature.

Lyra reads the raw commit and tag objects, splits off the signature the way git does, and
verifies the SSHSIG in Go. Status is computed when read, so registering or deleting a key
changes it for existing commits.

## Lyra's signing key

Lyra signs the squash commits it creates with its own SSH key:

- **The key:** `--signing-key-file` (an OpenSSH private key, for example from a Secret).
  Without it, Lyra generates an ed25519 key in the data directory on first start.
- **Registration:** at startup the public key is registered as a signing key of the user
  `lyra`, so those commits show as `verified`. Keys used before stay registered, so changing
  the key keeps earlier merges verified.
- **Checking locally:** `lyra signing-key` prints the key and an `allowed_signers` line for
  `git verify-commit`.
