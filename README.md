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
The server needs `git` on `PATH`.

The data directory holds `lyra.db`, `repos/<name>.git` and `hooks/post-receive`
(written at startup, applied through `core.hooksPath`).

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

- `verified`: an SSH signature in the `git` namespace, made by a registered signing
  key over the exact signed content;
- `unverified`: signed, but by an unregistered or deleted key, in another namespace,
  over different content, or not an SSH signature;
- `unsigned`: no signature.

Lyra reads the raw commit and tag objects with `git cat-file --batch`, splits off the
`gpgsig` header or the trailing tag signature the way git does, and verifies the
SSHSIG in Go. Status is computed when read, so registering or deleting a key changes
it for existing commits.

## CLI

Output is JSON on stdout. Errors are `{"error": "..."}` on stderr with exit code 1
(2 for usage errors). Every command takes `--help` and `--jq <expression>`, which
prints strings raw and other values as JSON. `lyra completion <shell>` prints shell
completion.

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

## Development

`mise install` provides Go and sqlc. The schema lives in goose migrations under
`internal/store/migrations`, applied by the server at startup. Queries live in
`internal/store/queries.sql`; run `sqlc generate` after changing either, and commit the
generated `internal/store/queries` package.
