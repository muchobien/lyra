# Issues and pull requests

Issues and pull requests share one number sequence per repository. Assignees are logins or
agent names; `@me` is the `Lyra-Agent` name when one is set, otherwise the login. Labels
must exist before they are applied. Every write returns the `event_id` it recorded.

## Pull requests

A pull request proposes a branch (`--head`) for another branch in the same repository
(`--base`, by default the default branch). Lyra keeps `refs/pull/<number>/head` at its
latest commit, so its commits stay available after the branch is deleted. Reopening syncs
it with its head branch, and is refused if that branch was deleted.

`lyra pr diff` prints the unified diff against the merge base, and `--files` lists the
changed files as JSON.

## How pushes update them

- A push to an open pull request's head branch updates it (`pr.synchronized`).
- When an open pull request's head commit becomes part of its base branch, it is marked
  merged (`pr.merged`, method `push`).
- Commits pushed to the default branch close the issues they reference with `close`,
  `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves` or `resolved` followed
  by `#<number>`. So does merging a pull request whose title or body references them.

## Merging

`lyra pr merge` merges from Lyra:

- **`--ff`** moves the base branch to the head commit. Only possible when the base hasn't
  moved since the branch was made.
- **`--squash`** writes one commit on top of the base, signed by Lyra.
  - Its tree comes from `git merge-tree`, and conflicts are refused.
  - The author is the author of the pull request's first commit. Other authors get
    `Co-authored-by` trailers.
  - The committer is `Lyra <lyra@<host>>`.
  - The message is the title with `(#<number>)`, then the body.
- **Safety:** the base branch is updated only if it hasn't moved since Lyra read it.
  `--head SHA` refuses the merge if the head branch is no longer at the commit you
  reviewed. Conflicts and moved branches return HTTP 409.
- **Afterwards:** a merge records the same events as a push, and closes issues referenced
  in the pull request or the squash message.

`lyra pr view` includes `merge`, with the current head and base commits, `fast_forward`, the
`conflicts`, the `methods` the branch rules allow, and the reasons the merge is `blocked`.

## Reviews and line comments

- `lyra pr review` records `approved`, `changes_requested` or `commented` against the pull
  request's current head commit, with optional line comments.
- A line comment names a file and a line in the head version (`side` `new`) or the base
  version (`old`). The line must be in the pull request's diff.
- When a later push changes the file, its `new`-side comments are marked `outdated`.
- `lyra pr view` lists `reviews`, each with its line comments, and every comment with its
  `path`, `line`, `side` and `commit_sha`.
- Each review is recorded as `pr.review`, and each line comment as `pr.comment` with its
  position.

## Branch rules

Rules apply to branches whose name matches a glob (`main`, `release/*`). When several match,
the strictest combination applies. Rules apply to admins and the tailnet identity too.
Managing rules needs `admin`, and each change is recorded as `rule.set` or `rule.delete`.

- **Protection:** a branch matching a rule can't have its history rewritten or be deleted
  (also through `pr merge --delete-branch`), unless every matching rule allows it with
  `--allow-force-push` or `--allow-deletion`.
- **`--require-signed`:** every commit a push adds to a matching branch must be `verified`,
  or the `pre-receive` hook refuses the push.
  - On an existing branch, the added commits are those not reachable from the old tip.
  - A push that creates a matching branch checks its whole history.
  - Merges are refused while the pull request has an unverified commit, with squash too, so
    Lyra's signature can't stand in for an unsigned commit.
- **`--require-approval`:** a pull request can be merged only after someone other than its
  author approves its current head commit. A new push needs a new approval, and no
  reviewer's latest review may request changes. The author is the same login and agent
  name, so the user approving an agent's pull request counts.
- **`--require-checks`:** the pull request's [checks](actions.md#checks) must be `success`.
- **`--merge-methods`:** limits how pull requests into a matching branch can be merged.
