# Lyra

A self-hosted Git forge for one person and their agents. One Go binary, SQLite, bare
repositories on disk. Agents use the `lyra` CLI; the user browses code and works on
issues in the web UI.

- **CLI:** `lyra --help` and `lyra <command> --help` list every command and flag.
- **Server flags:** `lyra serve --help`. Every flag also reads a `LYRA_*` variable.
- **API:** [`api/openapi.yaml`](api/openapi.yaml) (OpenAPI 3.1), generated from the code.
  The server also serves it at `/api/openapi.yaml` and `/api/openapi.json`, and
  `lyra openapi` prints it. Render it with
  `npx @redocly/cli build-docs api/openapi.yaml -o lyra-api.html`.

## Run

```sh
lyra serve --listen :8080 --data /var/lib/lyra \
  --trusted-proxy 10.42.0.0/16 --allow-login you@example.com
```

The server needs `git` on `PATH`. The data directory holds `lyra.db`,
`repos/<owner>/<repo>.git`, `hooks/` (written at startup, applied through `core.hooksPath`),
`blobs` with the default storage, and the generated `lyra_signing_key` and `secrets_key`
unless `--signing-key-file` and `--secrets-key-file` are set.

Release assets, job logs and container images are stored in `<data>/blobs`, or in S3 with
`--storage s3` (Garage, for example), with credentials in `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`. Lyra uses path-style requests and serves every download itself, so
the bucket doesn't need to be reachable by clients.

### Install

Each version tag publishes a GitHub release with static binaries for `linux-amd64`,
`linux-arm64` and `darwin-arm64`, plus `checksums.txt` and `packslip.sigstore.json`.
[packslip](https://packslip.dev) picks the archive for the host and checks that this
repository's release workflow signed it:

```sh
packslip install github.com/muchobien/lyra
packslip verify packslip.sigstore.json --artifact lyra-<version>-linux-amd64.tar.gz
```

To build from source, run `mise install`, `mise run ui:build`, then `go build -o bin/lyra ./cmd/lyra`.

### Container image

`ghcr.io/muchobien/lyra` is built for `linux/amd64` and `linux/arm64`, tagged `main` and
`sha-<short>` from `main`, and `X.Y.Z`, `X.Y` and `latest` from version tags. It runs
`lyra serve` as a non-root user with data in the `/data` volume:

```sh
docker run -p 8080:8080 -v lyra-data:/data \
  -e LYRA_TRUSTED_PROXY=10.42.0.0/16 -e LYRA_ALLOW_LOGIN=you@example.com \
  ghcr.io/muchobien/lyra:main
```

## Authentication

- Requests from a `--trusted-proxy` address carrying `Tailscale-User-Login` act as
  that login with full access, if it is listed in `--allow-login`. The header is
  ignored from any other address.
- Otherwise a token is required: `Authorization: Bearer <token>`, or HTTP Basic with
  the token as the password (any username) for Git and the registry.
- Tokens have a scope (`read` < `write` < `admin`), an optional list of repositories
  (`owner/repo`, or `owner/*`) and an optional expiry. Only their SHA-256 hash is stored.
- `Lyra-Agent` and `Lyra-Session` headers are recorded with every write and push.

## Owners and repositories

Every repository belongs to an owner, a namespace such as `krosf`, and is named
`owner/repo` everywhere: `krosf/lyra` in the CLI, `/krosf/lyra` in the web UI,
`/api/repos/krosf/lyra` in the API, and `https://lyra.example/krosf/lyra.git` for Git.

- Owners are created with `lyra owner create` before their repositories, and can only be
  deleted once they have none. They have no members: access comes from the tailnet login
  or a token.
- Names are 1 to 100 letters, digits, `.`, `_` or `-`, starting with a letter or digit.
  Owners can't be `api`, `static`, `healthz` or `v2`, and repositories can't end in `.git`.

```sh
git clone https://lyra.example/<owner>/<repo>.git
git -c http.extraHeader="Lyra-Agent: claude" push
```

Pushing needs `write`; cloning and fetching need `read`. Git prints what a push changed as
`remote: lyra: …` lines.

## Signatures

Register SSH signing keys with `lyra key add --signing`. Commits and tags are reported as
`verified` (an SSH signature in the `git` namespace by a registered key over the exact
signed content), `unverified` (any other signature) or `unsigned`. Status is computed when
read, so registering or deleting a key changes it for existing commits.

Lyra signs the squash commits it creates with its own SSH key (`--signing-key-file`, or
generated on first start), registered at startup as a signing key of the user `lyra`. Keys
used before stay registered, so changing the key keeps earlier merges verified.
`lyra signing-key` prints an `allowed_signers` line for `git verify-commit`.

## Issues and pull requests

Issues and pull requests share one number sequence per repository. Assignees are logins or
agent names; `@me` is the `Lyra-Agent` name when one is set, otherwise the login. Labels
must exist before they are applied. Every write returns the `event_id` it recorded.

A pull request proposes a branch (`--head`) for another branch in the same repository
(`--base`, by default the default branch). Lyra keeps `refs/pull/<number>/head` at its
latest commit, so its commits stay available after the branch is deleted. Reopening syncs
it with its head branch, and is refused if that branch was deleted.

Pushes update pull requests:

- A push to an open pull request's head branch updates it (`pr.synchronized`).
- When its head commit becomes part of its base branch, it is marked merged (`pr.merged`,
  method `push`).
- Commits pushed to the default branch close the issues they reference with `close`,
  `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves` or `resolved`
  followed by `#<number>`. So does merging a pull request whose title or body references
  them.

`lyra pr merge` merges from Lyra:

- **`--ff`** moves the base branch to the head commit, if the base hasn't moved since the
  branch was made.
- **`--squash`** writes one commit on top of the base, signed by Lyra, with its tree from
  `git merge-tree`. The author is the author of the first commit, other authors get
  `Co-authored-by` trailers, and the message is the title with `(#<number>)`, then the body.
- **Safety:** the base branch is updated only if it hasn't moved since Lyra read it, and
  `--head SHA` refuses the merge if the head branch is no longer at the commit you
  reviewed. Conflicts and moved branches return 409.

`lyra pr view` includes `merge`: the head and base commits, `fast_forward`, the
`conflicts`, the `methods` the branch rules allow, and the reasons the merge is `blocked`.

### Reviews

`lyra pr review` records `approved`, `changes_requested` or `commented` against the
current head commit, with optional line comments. A line comment names a file and a line
in the head version (`side` `new`) or the base version (`old`), and the line must be in the
diff. When a later push changes the file, its `new`-side comments are marked `outdated`.

### Branch rules

Rules apply to branches whose name matches a glob (`main`, `release/*`). When several
match, the strictest combination applies. Rules apply to admins and the tailnet identity
too, and managing them needs `admin`.

- **Protection:** pushes that rewrite a matching branch's history and its deletion are
  refused, unless every matching rule allows them with `--allow-force-push` or
  `--allow-deletion`.
- **`--require-signed`:** every commit a push adds must be `verified`, or the
  `pre-receive` hook refuses it; a push that creates the branch checks its whole history.
  Merges are refused while the pull request has an unverified commit, with squash too, so
  Lyra's signature can't stand in for an unsigned commit.
- **`--require-approval`:** someone other than the author (the same login and agent name)
  must approve the current head commit, and no reviewer's latest review may request
  changes. So the user approving an agent's pull request counts.
- **`--require-checks`:** the pull request's checks must be `success`.
- **`--merge-methods`:** limits how pull requests can be merged.

## Waiting for events

`lyra event wait` blocks until a matching event happens and prints it. Actions are glob
patterns (`push`, `repo.*`). With `--since ID` it also returns events since that one, so
an agent that saved the last event id it saw can't miss one. It reconnects if the server
restarts. `--assignee me` turns it into a work queue; check
`lyra issue list --assignee @me` first for work assigned earlier.

`lyra pr wait` and `lyra issue wait` block until someone other than the caller acts on
that pull request or issue, then print the event with the review or comment it carries.

- **Default actions:** pull requests wait for a review, comment, finished checks, merge or
  close; issues for anything but their opening. `--for` narrows it.
- **The caller** is the token, or the agent and session when `LYRA_SESSION` is set, so
  another session of the same agent counts as someone else.
- **Where the wait starts:** without `--since`, after the later of the caller's last action
  on the item and the last event a wait returned to the caller. A reply that arrived
  between pushing a fix and starting to wait returns at once, and waiting again doesn't
  return the same event twice.

## Releases

A release belongs to a tag. Publishing creates the tag at `--target` if it doesn't exist;
an existing tag is never moved. Drafts are visible only with `write` access, and don't
create their tag until published.

- **Assets:** `--asset` creates the release as a draft, uploads the files, then publishes
  it, so `release.published` fires once with every asset present. `download` checks each
  file's sha256.
- **Latest** is the most recently published release that isn't a prerelease.
- **Generated notes** list the pull requests whose merge commit is reachable from the
  release's commit but not from the previous release (the most recently published one
  whose commit is an ancestor; prereleases count only for another prerelease).
- **Stable URLs:** `/{owner}/{repo}/releases/download/{tag}/{name}` and
  `/{owner}/{repo}/releases/latest/download/{name}`, with the sha256 as `ETag` and range
  requests, so interrupted downloads can resume.

## Container registry

Lyra serves the OCI distribution API at `/v2/`. An image is named after a repository,
optionally with more path components: `registry.example.com/krosf/lyra:v1` or
`registry.example.com/krosf/lyra/runner:v1`. Pushing to a repository that doesn't exist is
refused with `NAME_UNKNOWN`.

```sh
docker login registry.example.com -u lyra --password-stdin <<< "$LYRA_TOKEN"
docker push registry.example.com/krosf/lyra:v1
```

- **Access:** pulling needs `read`, pushing `write`, deleting `admin`. Mounting a layer from
  another repository needs `read` there. The catalog needs an unrestricted token.
- **Tracking:** each tag pushed is recorded with its digest and pusher. Manifests pushed
  only by digest, such as the per-platform manifests of a multi-arch image, aren't listed.
- **Space** is freed by `lyra admin registry-gc`, which deletes layers and manifests no
  tag refers to. Lyra is read-only while it runs.
- **Releases** list the images tagged with their tag, with or without a leading `v`.

## Actions

Lyra serves the Forgejo runner protocol at `/api/actions/` for `forgejo-runner` 13.
Runners authenticate with their own uuid and token. `lyra runner create` prints those for
a `server.connections` entry in the runner's config, which many processes can share (such
as `forgejo-runner one-job` in a fresh pod per job). `lyra runner token create` makes a
registration token for `forgejo-runner register`, including `--ephemeral`.

Workflows live in `.forgejo/workflows/`, or `.github/workflows/` when the first doesn't
exist. Triggers:

- **`push`**, with `branches`, `tags`, `paths` and their `-ignore` forms.
- **`pull_request`** on opening, reopening and pushing, with `types`, `branches` (the base)
  and `paths`, using the workflow file from the head commit. `github.ref` is
  `refs/pull/<n>/head`.
- **`workflow_dispatch`** with `lyra workflow run`. Inputs are checked against the workflow.
- **`release`** when a release is published: `published`, `released` or `prereleased`.
- **`schedule`**: each minute, `cron` entries (UTC) on the default branch's workflows.

How jobs run:

- **Jobs** are split and matrices expanded with the runner's own parser. A runner gets the
  oldest waiting job whose `runs-on` labels it all has. A job whose `needs` didn't succeed
  is skipped, unless it has an `if:`.
- **Job token:** `github.token` has `write` access to its own repository only, for Git, the
  API and the CLI. Writes are recorded as `job:<repo>#<run>`.
- **Secrets** are encrypted with AES-256-GCM using the secrets key. Their values are never
  returned. Changing the key makes existing secrets unreadable, and jobs that use them fail
  with `lyra: the job couldn't start`.
- **Dead runners:** a job whose runner stops reporting for `--job-stale-after` fails.
- **Retention:** finished runs and their logs are deleted after `--actions-retention`.

**Checks:** `lyra pr view` includes `checks`, the jobs of the latest run of each workflow
and event for the head commit. When they finish, Lyra records `pr.checks` on the pull
request, so `lyra pr wait --for checks` returns. When no workflow ran for the head, it
returns at once with state `none`.

## Backups

```sh
lyra admin backup --output lyra.tar
lyra restore lyra.tar --data /new/data   # offline, into an empty directory
lyra admin storage-check                 # stored files the database refers to but storage lacks
```

- **A backup holds** `lyra.db` (copied with `VACUUM INTO`), every bare repository, and
  `manifest.json` with each repository's refs and the stored files and images the database
  refers to. Writes wait while the database and repositories are copied; `admin status`
  reports the pause.
- **Not included:** the signing and secrets keys, and the files in storage (back up the S3
  bucket or `blobs` directory separately).
- **Checks:** `restore` checks the database's integrity and schema version, runs
  `git fsck` on every repository, and compares their refs with the manifest. After starting
  on the restored directory, `admin storage-check` exits 1 if storage lacks any file.
- **Read-only mode** (`lyra admin readonly on`) refuses every write with 503 while reads
  and clones keep working, until it's turned off or the server restarts.

## Metrics

`--metrics-listen :9090` serves Prometheus metrics at `/metrics` on a separate listener:
counts of repositories, runs, jobs and online runners, read-only mode, the last backup's
time, size and pause, HTTP requests, and Go runtime metrics.

## Web UI

The web interface is a React application built with Vite+, TanStack Router and Query,
and CSS Modules. Its assets are embedded in the Go binary; Node is needed only at build
 time. It targets current stable Chrome and uses native dialogs, customizable selects,
popovers, and light/dark appearance.

Browse repositories, branches, files, commit history, and tags. The repository sidebar
shows the latest stable release and its tag-linked containers. File and branch selection
are kept in the URL; `T` opens the file finder. Go renders Markdown, highlights source,
and parses structured diffs.

Issue and pull request screens support creation, edits, comments, labels, and assignees.
Pull requests include commits, checks, file diffs, line comments, reviews, and merging.
Review and merge submissions identify the head that was displayed. Comment drafts and
viewed-file progress survive navigation within the application and reset on reload.

Actions includes workflow discovery, typed manual inputs, job/step views, incremental
logs, cancellation, and reruns. Release screens support drafts, publishing, generated
notes, asset uploads/downloads and checksums. Containers link back to matching releases.
Activity updates through the existing event stream.

Repository settings expose variables and branch rules to readers, variable/label edits
to writers, and secrets to administrators. Unrestricted administrators can manage access
tokens, signing keys, and runners. Secret values are never read back or stored in drafts;
new credentials are shown once and cleared when their dialog closes.

The API, Git transport, runner service, registry, raw files, and release downloads keep
their own routes. Missing assets are never served an application shell. Source tree and
blob URLs resolve to the repository workspace, with raw blob downloads still served directly.

## Development

Run `mise run build` to install the frozen frontend dependencies, build the embedded
assets in `internal/webui/dist`, and compile Go. A clean checkout needs `mise run ui:build`
before direct Go build/test commands. `mise run ui:check` runs Vite+ formatting, lint,
and type checking; frontend versions are exact and locked with pnpm.

For UI development, start a local Lyra server and run `pnpm --dir web dev`.
`LYRA_DEV_URL` chooses the backend (default `http://127.0.0.1:8080`);
`LYRA_DEV_LOGIN` optionally supplies the trusted-proxy login for a locally configured
backend. These development-server variables are never included in the browser bundle.

GoReleaser builds the frontend before compiling the release binaries, so archives and
container images contain the same embedded UI.


`mise install` provides the pinned Go, Node, pnpm, sqlc, golangci-lint, GoReleaser, packslip and the workflow
linters. The schema lives in goose migrations under `internal/store/migrations`, applied
at startup. Queries live in `internal/store/queries.sql`; run `sqlc generate` after
changing either, and commit the generated `internal/store/queries` package.

The API is served with [huma](https://huma.rocks), which builds the OpenAPI document from
the operations in `internal/server/openapi.go` and the Go types of their inputs and
outputs. Run `mise run openapi` after changing the API, and commit `api/openapi.yaml`.

`mise tasks` lists the tasks CI runs: `fmt`, `lint`, `build`, `test`, `sqlc:check`,
`openapi:check`, `lint:actions`, `actions:lock`, `actions:verify`, `dist`, `dist:snapshot`
and `image`. The
`Dockerfile` unpacks an archive from `dist/`, so build the image with `mise run image`, not
a bare `docker build`.

`go test ./...` runs the end-to-end tests in `e2e/`. They need `git` and `ssh-keygen` on
`PATH`, build the binary, start servers on free ports, and drive them with `git`, the CLI
and HTTP. The S3 test needs Docker and is skipped without it.

### Releases

Pushing a `v*` tag runs the release workflow. GoReleaser builds the archives and the
changelog, the container image is built from those archives and pushed, packslip signs and
attests the archives with the workflow's identity, and the last job creates the GitHub
release. Pushes to `main` and pull requests run the same build as a snapshot.
