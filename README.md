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

### Container image

`ghcr.io/muchobien/lyra` is built for `linux/amd64` and `linux/arm64`, tagged `main` and
`sha-<short>` from `main`, and `X.Y.Z`, `X.Y` and `latest` from version tags. It runs
`lyra serve` as a non-root user with data in the `/data` volume:

```sh
docker run -p 8080:8080 -v lyra-data:/data \
  -e LYRA_TRUSTED_PROXY=10.42.0.0/16 -e LYRA_ALLOW_LOGIN=you@example.com \
  ghcr.io/muchobien/lyra:main
```

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
(2 for usage errors, 3 when a wait times out). Every command takes `--help` and `--jq <expression>`, which
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
lyra event wait [--repo NAME] [--action PATTERN]... [--since ID] [--timeout 30m]
```

`lyra event wait` blocks until a matching event happens and prints it. Actions are glob
patterns (`push`, `repo.*`). Without `--since` it waits for events after it starts; with
`--since ID` it also returns events since that one, so an agent that saved the id of the
last event it saw can't miss one that happened in between. It reconnects on its own if
the server restarts.

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
| `GET /api/events/stream?repo=&action=&since=` | read |

`/api/events/stream` is a server-sent event stream. Each event's `id` is its event id,
and the stream accepts `Last-Event-ID` to resume. It starts with an `id` line holding
the current position, and sends a keepalive every 25 seconds. Tokens see only events
for the repositories they can read; account events (tokens, keys) need a token without
a repository list.

## Web UI

`/` lists repositories; `/{name}` browses files at the default branch,
`/{name}/tree/{ref}/{path}`, `/{name}/blob/{ref}/{path}` (`?raw` for the raw file),
`/{name}/commits/{ref}`, `/{name}/commit/{sha}` and `/{name}/tags`.

## Development

`mise install` provides Go and sqlc. The schema lives in goose migrations under
`internal/store/migrations`, applied by the server at startup. Queries live in
`internal/store/queries.sql`; run `sqlc generate` after changing either, and commit the
generated `internal/store/queries` package.

`go test ./...` runs the end-to-end tests in `e2e/`, which CI also runs. They need
`git` and `ssh-keygen` on `PATH`. The tests build the `lyra` binary, start servers on
free ports with temporary data directories and generated SSH keys, and drive them
with `git` and the CLI. They cover:

- Auth through the Tailscale header and through tokens.
- Clone and push, including chunked request bodies.
- Scoped, read-only, revoked and expired tokens, and recorded actors.
- Signature status of commits and tags, cross-checked against `git verify-commit`
  and `git verify-tag`, including forged signatures.
- Waiting for events: filters, `--since`, timeouts, scoped tokens, and resuming across
  a server restart.
