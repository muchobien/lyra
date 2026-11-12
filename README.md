# Lyra

A self-hosted Git forge for one person and their agents. One Go binary, SQLite, bare
repositories on disk. Agents use the `lyra` CLI; the web UI is a read-only browser for
the user.

## Run

```sh
go build -o bin/lyra ./cmd/lyra
lyra serve --listen :8080 --data /var/lib/lyra \
  --trusted-proxy 10.42.0.0/16 --allow-login you@example.com
```

Flags also read `LYRA_LISTEN`, `LYRA_DATA`, `LYRA_TRUSTED_PROXY` and `LYRA_ALLOW_LOGIN`.
The server needs `git` and `ssh-keygen` on `PATH`.

The data directory holds `lyra.db`, `repos/<name>.git`, `hooks/post-receive` (written
at startup, applied through `core.hooksPath`) and `allowed_signers` (rebuilt when
signing keys change).

## Authentication

- Requests from a `--trusted-proxy` address carrying `Tailscale-User-Login` act as
  that login with full access, if it is listed in `--allow-login`. The header is
  ignored from any other address.
- Otherwise a token is required: `Authorization: Bearer <token>`, or HTTP Basic with
  the token as the password (any username) for Git.
- Tokens have a scope (`read` < `write` < `admin`), an optional list of repositories
  and an optional expiry. Only their SHA-256 hash is stored. Managing tokens and keys
  needs an admin token without a repository list.
- `Lyra-Agent` and `Lyra-Session` headers are recorded with every write and push.

## Git

```sh
git clone https://lyra.example/<name>.git
git -c http.extraHeader="Lyra-Agent: claude" push
```

Pushing needs `write`; cloning and fetching need `read`.

## Commit and tag signatures

Register SSH signing keys with `lyra key add --signing [--agent NAME] key.pub`. Commits
and tags are reported as:

- `verified`: signed by a registered key (`git verify-commit` / `git verify-tag`
  succeed against `allowed_signers`);
- `unverified`: signed, but not by a registered key, or the signature is bad;
- `unsigned`: no signature.

Commit lists use `git log` signature placeholders, which run the same check; single
commits and tags call `git verify-commit` and `git verify-tag`.

## CLI

Output is JSON on stdout. Errors are `{"error": "..."}` on stderr with exit code 1
(2 for usage errors).

```
LYRA_URL, LYRA_TOKEN, LYRA_AGENT, LYRA_SESSION

lyra me
lyra repo create <name> [--description TEXT]
lyra repo list
lyra repo get <name>
lyra repo delete <name> --yes
lyra commit list <repo> [--ref REF] [--limit N]
lyra commit show <repo> <rev>
lyra tag list <repo>
lyra token create --name NAME --scope read|write|admin [--repo NAME]... [--expires 720h]
lyra token list
lyra token revoke <id>
lyra key add --signing [--agent NAME] [--title TEXT] <public-key-file|->
lyra key list
lyra key delete <id>
lyra event list [--repo NAME] [--limit N]
```

## API

| Method and path | Scope |
| --- | --- |
| `GET /api/me` | any |
| `GET /api/repos`, `POST /api/repos` | read, admin |
| `GET /api/repos/{name}`, `DELETE /api/repos/{name}` | read, admin |
| `GET /api/repos/{name}/commits?ref=&limit=` | read |
| `GET /api/repos/{name}/commits/{rev}` | read |
| `GET /api/repos/{name}/tags` | read |
| `GET /api/tokens`, `POST /api/tokens`, `DELETE /api/tokens/{id}` | admin |
| `GET /api/keys`, `POST /api/keys`, `DELETE /api/keys/{id}` | read, admin |
| `GET /api/events?repo=&limit=` | read |

## Web UI

`/` lists repositories; `/{name}` browses files at the default branch,
`/{name}/tree/{ref}/{path}`, `/{name}/blob/{ref}/{path}` (`?raw` for the raw file),
`/{name}/commits/{ref}`, `/{name}/commit/{sha}` and `/{name}/tags`.
