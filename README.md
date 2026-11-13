# Lyra

A self-hosted Git forge for one person and their agents. One Go binary, SQLite, bare
repositories on disk. Agents use the `lyra` CLI; the user browses code and works on
issues in the web UI.

## Run

```sh
go build -o bin/lyra ./cmd/lyra
lyra serve --listen :8080 --data /var/lib/lyra \
  --trusted-proxy 10.42.0.0/16 --allow-login you@example.com
```

Flags also read `LYRA_LISTEN`, `LYRA_DATA`, `LYRA_TRUSTED_PROXY`, `LYRA_ALLOW_LOGIN` and
`LYRA_SIGNING_KEY_FILE`. The server needs `git` on `PATH`.

Release assets are stored in `<data>/blobs` by default. To store them in S3 (Garage, for
example), pass `--storage s3 --s3-endpoint http://garage:3900 --s3-bucket lyra` (or
`LYRA_STORAGE`, `LYRA_S3_ENDPOINT`, `LYRA_S3_BUCKET`; `--s3-region` defaults to `garage`), with
credentials in `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`. Lyra uses path-style
requests and serves every download itself, so the bucket doesn't need to be reachable by
clients.

### Container image

`ghcr.io/muchobien/lyra` is built for `linux/amd64` and `linux/arm64`, tagged `main` and
`sha-<short>` from `main`, and `X.Y.Z`, `X.Y` and `latest` from version tags. It runs
`lyra serve` as a non-root user with data in the `/data` volume:

```sh
docker run -p 8080:8080 -v lyra-data:/data \
  -e LYRA_TRUSTED_PROXY=10.42.0.0/16 -e LYRA_ALLOW_LOGIN=you@example.com \
  ghcr.io/muchobien/lyra:main
```

The data directory holds `lyra.db`, `repos/<name>.git`, `hooks/pre-receive` and `hooks/post-receive`
(written at startup, applied through `core.hooksPath`), `blobs` with the default storage and,
unless `--signing-key-file` is set, the generated `lyra_signing_key`.

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

Lyra signs the squash commits it creates with its own SSH key:

- **The key:** `--signing-key-file` (an OpenSSH private key, for example from a
  Secret). Without it, Lyra generates an ed25519 key in the data directory on first
  start.
- **Registration:** at startup the public key is registered as a signing key of the
  user `lyra`, so those commits show as `verified`. Keys used before stay registered,
  so changing the key keeps earlier merges verified.
- **Checking locally:** `lyra signing-key` prints the key and an `allowed_signers`
  line for `git verify-commit`.

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
lyra event wait [--repo NAME] [--action PATTERN]... [--assignee NAME|me] [--since ID] [--timeout 30m]
lyra issue wait <repo> <number> [--for comment,closed,...] [--since ID] [--timeout 30m]
lyra pr wait <repo> <number> [--for review,comment,merged,closed,push,...] [--since ID] [--timeout 30m]
lyra issue create <repo> --title TEXT [--body TEXT|--body-file F|-] [--label L]... [--assignee NAME|@me]...
lyra issue list <repo> [--state open|closed|all] [--label L]... [--assignee NAME|@me]... [--limit N]
lyra issue view <repo> <number>
lyra issue comment <repo> <number> --body TEXT|--body-file F|-
lyra issue edit <repo> <number> [--title TEXT] [--body ...] [--add-label L] [--remove-label L] [--add-assignee A] [--remove-assignee A]
lyra issue close|reopen <repo> <number>
lyra pr create <repo> --head BRANCH [--base BRANCH] --title TEXT [--body ...] [--label L]... [--assignee NAME|@me]...
lyra pr list <repo> [--state open|closed|merged|all] [--label L]... [--assignee NAME|@me]... [--limit N]
lyra pr view|commits <repo> <number>
lyra pr diff <repo> <number> [--files]
lyra pr merge <repo> <number> --squash|--ff [--head SHA] [--delete-branch] [--title TEXT] [--body ...]
lyra signing-key
lyra pr review <repo> <number> --approve|--request-changes|--comment [--body ...] [--comments-file F|-]
lyra pr comment <repo> <number> --body ... [--path FILE --line N [--side new|old]]
lyra rule set <repo> <pattern> [--require-signed] [--require-approval] [--merge-methods ff,squash]
lyra rule list <repo>
lyra rule delete <repo> <pattern>
lyra pr comment|edit|close|reopen <repo> <number> ...   (same flags as for issues)
lyra label create <repo> <name> [--color d73a4a] [--description TEXT]
lyra label list <repo>
lyra label delete <repo> <name>
```

```
lyra release create <repo> <tag> [--target REF] [--title T] [--notes ...|--notes-file F|-]
                    [--draft] [--prerelease] [--asset FILE]...
lyra release list <repo>
lyra release view <repo> <tag|latest>
lyra release edit <repo> <tag> [--title T] [--notes ...] [--prerelease=true|false] [--target REF]
lyra release publish <repo> <tag>
lyra release upload <repo> <tag> FILE... [--clobber]
lyra release download <repo> <tag|latest> [--pattern GLOB] [--dir D]
lyra release delete-asset <repo> <tag> <name>
lyra release delete <repo> <tag> --yes [--cleanup-tag]
```

A release belongs to a tag:

- **Tags:** if the tag doesn't exist, publishing creates it as a lightweight tag at
  `--target` (default: the default branch). A draft doesn't create its tag until it's
  published. An existing tag is never moved: `--target` must match it.
- **Drafts** are visible only with `write` access.
- **Assets:** `--asset` creates the release as a draft, uploads the files, then publishes
  it, so `release.published` fires once with every asset present. Uploading a name that
  exists needs `--clobber`. `download` checks each file's sha256 before keeping it.
- **Latest** is the most recently published release that isn't a prerelease.
- **Stable URLs:** `/{repo}/releases/download/{tag}/{name}` and
  `/{repo}/releases/latest/download/{name}` serve assets with their sha256 as the `ETag`.
- **Events:** `release.created` (drafts), `release.published`, `release.updated`,
  `release.deleted`, and `release.asset` for each upload or deletion.

Issues and pull requests share one number sequence per repository. Assignees are
logins or agent names; `@me` is the `Lyra-Agent` name when one is set, otherwise the
login. Labels must exist before they are applied. Every write returns the `event_id` it
recorded.

A pull request proposes a branch (`--head`) for another branch in the same repository
(`--base`, by default the default branch). Lyra keeps `refs/pull/<number>/head` at its
latest commit, so its commits stay available after the branch is deleted. `pr diff`
prints the unified diff against the merge base, and `--files` lists changed files as
JSON.

Pushes update pull requests:

- A push to an open pull request's head branch updates it (`pr.synchronized`).
- When an open pull request's head commit becomes part of its base branch, it is marked
  merged (`pr.merged`, method `push`).
- Commits pushed to the default branch close the issues they reference with
  `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves` or
  `resolved` followed by `#<number>`. So does merging a pull request whose title or
  body references them.

Git prints what changed as `remote: lyra: …` lines.

`lyra pr merge` merges from Lyra:

- **`--ff`:** moves the base branch to the head commit. Only possible when the base
  hasn't moved since the branch was made.
- **`--squash`:** writes one commit on top of the base, signed by Lyra.
  - Its tree comes from `git merge-tree`, and conflicts are refused.
  - The author is the author of the pull request's first commit. Other authors get
    `Co-authored-by` trailers.
  - The committer is `Lyra <lyra@<host>>`.
  - The message is the title with `(#<number>)`, then the body.
- **Safety:** the base branch is updated only if it hasn't moved since Lyra read it.
  `--head SHA` refuses the merge if the head branch is no longer at the commit you
  reviewed.
- **Conflicts and moved branches:** these return HTTP 409.
- **Afterwards:** a merge records the same events as a push, and closes issues
  referenced in the pull request or the squash message.

`lyra pr view` includes `merge`, with:

- the current head and base commits;
- `fast_forward`;
- the `conflicts`;
- the `methods` the branch rules allow;
- the reasons the merge is `blocked`.

Branch rules apply to branches whose name matches a glob (`main`, `release/*`). When
several match, the strictest combination applies. Rules apply to admins and the
tailnet identity too.

- `--require-signed`: every commit a push adds to a matching branch must have a
  `verified` signature, or the push is refused by the `pre-receive` hook.
  - On an existing branch, the added commits are those not reachable from the old tip.
  - A push that creates a matching branch checks its whole history.
  - Merges into a matching branch are refused while the pull request has an
    unverified commit, with squash too, so Lyra's signature can't stand in for an
    unsigned commit.
- `--require-approval`: a pull request into a matching branch can be merged only
  after someone other than its author approves its current head commit. A new push
  needs a new approval, and no reviewer's latest review may request changes. The
  author is the same login and agent name, so the human user approving an agent's
  pull request counts.
- `--merge-methods`: limits how pull requests into a matching branch can be merged.

**Reviews and line comments:**

- **What a review is:** `lyra pr review` records `approved`, `changes_requested` or
  `commented` against the pull request's current head commit, with optional line
  comments.
- **Pointing at a line:** a line comment names a file and a line number in the
  head version (`side` `new`) or the base version (`old`). The line must be in the
  pull request's diff.
- **Outdated comments:** when a later push changes the file, its `new`-side comments
  are marked `outdated`.
- **Where they show up:** `lyra pr view` lists `reviews`, each with its line
  comments, and every comment with its `path`, `line`, `side` and `commit_sha`.
- **Events:** each review is recorded as `pr.review`, and each line comment as
  `pr.comment` with its position.

Managing rules needs `admin`. Each change is recorded as `rule.set` or `rule.delete`.

`lyra pr wait` and `lyra issue wait` block until someone other than the caller acts on
that pull request or issue, then print the event, with the review or comment it
carries.

- **Default actions:** pull requests wait for a review, comment, merge or close; issues
  wait for anything but their opening. `--for` narrows it.
- **Who "the caller" is:** the token, or the agent and session when `LYRA_SESSION` is
  set, so another session of the same agent counts as someone else.
- **Where the wait starts:** without `--since`, after the later of the caller's last
  action on the item and the last event a wait returned to the caller. So:
  - a reply that arrived between pushing a fix and starting to wait returns at once;
  - waiting again doesn't return the same event twice.

Lyra keeps that last returned event per caller and item. With `--since ID`, the wait
starts after that event instead.

`lyra event wait --assignee me` returns when an issue or pull request is assigned to
the caller (the agent name, or the login without one). This is a work queue; check
`lyra issue list --assignee @me` first for work assigned earlier.

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
| `GET /api/events/stream?repo=&item=&action=&assignee=&exclude_self=&since=` | read |
| `GET /api/repos/{name}/issues?state=&label=&assignee=&limit=` | read |
| `POST /api/repos/{name}/issues` | write |
| `GET /api/repos/{name}/issues/{number}` | read |
| `PATCH /api/repos/{name}/issues/{number}` | write |
| `POST /api/repos/{name}/issues/{number}/comments` | write |
| `GET /api/repos/{name}/labels`, `POST /api/repos/{name}/labels` | read, write |
| `DELETE /api/repos/{name}/labels/{label}` | write |
| `GET /api/repos/{name}/pulls?state=&label=&assignee=&limit=` | read |
| `POST /api/repos/{name}/pulls` (`title`, `body`, `head`, `base`, `labels`, `assignees`) | write |
| `GET /api/repos/{name}/pulls/{number}` | read |
| `PATCH /api/repos/{name}/pulls/{number}` | write |
| `POST /api/repos/{name}/pulls/{number}/comments` | write |
| `GET /api/repos/{name}/pulls/{number}/commits` | read |
| `GET /api/repos/{name}/pulls/{number}/diff` (`text/x-diff`) | read |
| `GET /api/repos/{name}/pulls/{number}/files` | read |
| `POST /api/repos/{name}/pulls/{number}/merge` (`method`, `head_sha`, `title`, `message`, `delete_branch`) | write |
| `GET /api/signing-key` | any |
| `GET /api/repos/{name}/rules` | read |
| `POST /api/repos/{name}/rules` (`pattern`, `require_signed`, `require_approval`, `merge_methods`) | admin |
| `POST /api/repos/{name}/pulls/{number}/reviews` (`state`: `approve`, `request_changes` or `comment`; `body`; `comments`) | write |

Comments on pull requests also take `path`, `line` and `side`.
| `DELETE /api/repos/{name}/rules?pattern=` | admin |
| `GET /api/repos/{name}/releases`, `GET /api/repos/{name}/releases/latest` | read |
| `GET /api/repos/{name}/releases/tags/{tag}`, `GET /api/repos/{name}/releases/{id}` | read |
| `POST /api/repos/{name}/releases` (`tag`, `target`, `title`, `notes`, `draft`, `prerelease`) | write |
| `PATCH /api/repos/{name}/releases/{id}` (`title`, `notes`, `prerelease`, `target`; `draft: false` publishes) | write |
| `DELETE /api/repos/{name}/releases/{id}?cleanup_tag=` | write |
| `POST /api/repos/{name}/releases/{id}/assets?name=&clobber=` (the file is the body) | write |
| `GET /api/repos/{name}/releases/{id}/assets/{asset}`, `DELETE …/assets/{asset}` | read, write |

`PATCH` on an issue or pull request takes any of `title`, `body`, `state` (`open` or `closed`), `add_labels`,
`remove_labels`, `add_assignees` and `remove_assignees`. Each change is recorded as
its own event (`issue.edited`, `issue.closed`, `issue.labeled`, `issue.assigned`, …),
and `event_id` is the last one.

`/api/events/stream` is a server-sent event stream. Each event's `id` is its event id,
and the stream accepts `Last-Event-ID` to resume. It starts with an `id` line holding
the current position, and sends a keepalive every 25 seconds. Tokens see only events
for the repositories they can read; account events (tokens, keys) need a token without
a repository list.

The stream's filters:

- `item`: one issue or pull request number, together with `repo`.
- `exclude_self=true`: leaves out the caller's own events.
- `assignee`: only `*.assigned` events for that name, where `me` is the caller.
- `since=mine`, with `item`: starts where `lyra pr wait` does. Every event delivered on
  an `item` stream moves that position forward.

## Web UI

`/` lists repositories; `/{name}` browses files at the default branch,
`/{name}/tree/{ref}/{path}`, `/{name}/blob/{ref}/{path}` (`?raw` for the raw file),
`/{name}/commits/{ref}`, `/{name}/commit/{sha}` and `/{name}/tags`.

`/{name}/issues` lists open issues (`?state=closed` for closed ones) and has a form to
open one. `/{name}/issues/{number}` shows the issue and its comments, rendered as
Markdown with raw HTML removed, with forms to comment, close or reopen, and set labels
and assignees. `/{name}/pulls` and `/{name}/pulls/{number}` do the same for pull
requests, with `?tab=commits` (signature status) and `?tab=files` (changed files and
the diff). An open pull request's page has:

- a merge form: squash or fast-forward, optionally deleting the branch. If the
  branches conflict, or a branch rule blocks the merge, it shows why instead;
- a review form: approve, request changes or comment;
- a timeline of comments and reviews.

The files tab shows each diff line with its old and new numbers, and line comments
under their lines. Clicking a line number opens a form to comment on that line, which
is the only script on the page.

Requests that change data and come from another site are rejected, using Go's
`http.CrossOriginProtection` (`Sec-Fetch-Site` and `Origin`). Without this, any page
could post forms as the user, because the proxy adds the tailnet login header to every
request. The CLI and git send neither header and aren't affected.

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
- Waiting for other people:
  - A reply that arrived before the wait.
  - The caller's own actions ignored, and another session's not.
  - Not returning the same event twice.
  - `--for`, issue waits, and the `--assignee me` work queue.
- Issues, labels and comments from the CLI and the UI forms, including rejected
  cross-site posts, Markdown sanitizing, and cleanup when a repository is deleted.
- Pull requests:
  - Creating them, with validation; commits, diff and changed files, including a
    rename.
  - Updates and merges detected from pushes, and closing keywords.
  - Closed and reopened pull requests, deleted head branches, and the UI tabs.
- Merging:
  - Squash merges by several authors, checked with `git verify-commit`.
  - Fast-forward, conflicts, a stale `--head`, and merging from the UI.
  - The signing key surviving a restart, and a key set with `--signing-key-file`.
- Branch rules:
  - Pushes refused for unsigned or unregistered-key commits, including with the
    tailnet identity, and a new branch whose history has unsigned commits.
  - Merges blocked by unsigned commits or a disallowed method.
  - Setting, listing and deleting rules, and the permissions to do so.
- Reviews:
  - Approving or requesting changes, refused for the author, and validation.
  - Line comments on both sides of the diff, refused outside it, and outdated after a
    push.
  - `--require-approval`, including an approval that a new push makes stale, and
    reviewing and commenting from the UI.
