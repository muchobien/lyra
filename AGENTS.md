# AGENTS.md

Lyra is a self-hosted Git forge for one person and their agents: one Go binary
(`cmd/lyra`), SQLite, and bare repositories on disk. User docs are in
[`docs/`](docs/README.md).

## Layout

| Path | What it holds |
| --- | --- |
| `cmd/lyra` | `main`: `serve`, `restore`, `version`, `openapi` and the git hooks |
| `internal/cli` | The `lyra` client commands (cobra) |
| `internal/server` | HTTP server: huma API, web UI, Git, registry, Actions runner protocol |
| `internal/server/openapi.go` | Every API operation, grouped by area |
| `internal/store` | SQLite access; migrations in `migrations/`, sqlc queries in `queries.sql` |
| `internal/repos` | Git operations (shells out to `git`) |
| `internal/backup` | Backup and restore |
| `e2e` | End-to-end tests: real servers driven with `git`, the CLI and HTTP |
| `api/openapi.yaml` | Generated OpenAPI document; don't edit by hand |
| `docs` | User documentation |

## Commands

```sh
mise run fmt                 # gofmt + goimports
mise run lint                # format check, go vet, golangci-lint (funlen 80, lll 120)
mise run build
go test -count=1 ./e2e/      # -count=1: the tests build the binary, so the cache misses server changes
mise run sqlc:check          # after changing migrations or queries.sql: sqlc generate
mise run openapi:check       # after changing the API: mise run openapi
```

Run lint and the e2e tests before committing. `mise tasks` lists everything CI runs.

## Code

- Match how the codebase already does it; find a sibling handler, command or test and copy
  its shape.
- Minimal diffs. Don't refactor nearby code, and don't add helpers with a single caller.
- No comments explaining what code does; rationale goes in the commit or PR description.
- JSON is `encoding/json/v2`. Nil slices encode as `[]`.
- No single-letter variable names.
- Report unrelated bugs instead of fixing them in the same change.

### API handlers

- Register each operation in `internal/server/openapi.go` with `register(api, huma.Operation{…}, handler)`.
  Its description starts with the scope it needs (`Scope: read.`).
- Handlers have the shape `func (s *Server) handleX(r *http.Request, input *xInput) (*body[T], error)`
  and end with `return reply(value)`. Return `badRequest`, `forbidden`, `notFound`,
  `conflict` or store errors; `register` turns them into `{"error": "..."}`.
- Path parameters come from embedded exported structs (`RepoPath`, `ItemPath`,
  `ReleasePath`, `RunPath`, `SecretPath`); huma ignores unexported embedded fields.
- Optional request fields use `omitzero`; fields without it are required in the schema.
  Use `enum`, `minimum` and `doc` tags so the spec documents them.
- Non-JSON responses (streams, files, tar) use `stream[Input](handler)` with an explicit
  `Responses` entry.
- After changing the API, run `mise run openapi` and commit `api/openapi.yaml`.

## Docs

- `README.md` stays short: what Lyra is, install, run, first steps.
- Behaviour that spans commands goes in `docs/<topic>.md`, linked from `docs/README.md`.
- Per-command behaviour belongs in the command's cobra `Long` help, not in the docs.
- State what the code does, as fact. Don't mention how a change was tested.

## Git and pull requests

The repository is `muchobien/lyra` on GitHub; use `gh`. `main` is the default branch.

- **Branches:** semantic names: `feat/…`, `fix/…`, `refactor/…`, `chore/…`, `docs/…`.
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org). Types used here:
  `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `build`, `chore`, `style`. A scope is
  optional (`refactor(server): …`). The subject is lowercase, says what the change does,
  and has no trailing period:
  - `feat: registry garbage collection`
  - `fix: resync a pull request's head when it's reopened`
  - `refactor: serve the API through huma and generate its OpenAPI document`
- **Body:** what changed and why, for a reader who hasn't seen the work.
- **Pull requests:** the title is the main commit's Conventional Commit subject. Open them
  as drafts, assigned to the owner, with existing labels only (`gh label list`). Body:

  ```markdown
  ## Summary
  <1–3 sentences: what changes and why>

  ## Changes
  - <one bullet per meaningful change>

  Closes #<issue>   ← only when there is one
  ```

  Don't mention local tooling or how you tested it, what you couldn't do, comparisons to
  other PRs or projects, or "should not" phrasing. After new commits, update the title
  and body to match the whole branch.
- **Merging:** squash merge into `main`.
- **Safety:** never force-push unless asked, and then with `--force-with-lease`. Never skip
  hooks with `--no-verify`; fix the cause.
- **Workflows:** the `gh` token can't push changes under `.github/workflows`; push those
  over SSH.
- Before committing, remove any comments, test pages and debug code you added.
