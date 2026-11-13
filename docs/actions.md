# Actions

Lyra serves the Forgejo runner protocol (ConnectRPC, `runner.v1`) at `/api/actions/`, for
`forgejo-runner` 13.

## Runners

Runners authenticate with their own uuid and token, not with the tailnet header or Lyra
tokens. Managing runners needs an unrestricted admin token.

- **`lyra runner create`** prints a uuid and token, shown once, for a `server.connections`
  entry in the runner's config file. Many processes can share that runner, such as
  `forgejo-runner one-job` in a fresh pod per job.
- **Registration tokens** (`lyra runner token create`) work with `forgejo-runner register`,
  including `--ephemeral`, and can register any number of runners until they're deleted.
- **Labels and version** come from the runner's `Declare` call when it starts.
- `--actions-url` is the URL runners reach Lyra at, such as an in-cluster service address.

## Triggers

Workflows live in `.forgejo/workflows/`, or `.github/workflows/` when the first doesn't
exist.

- **`push`:** with `branches`, `tags`, `paths` and their `-ignore` forms. The pusher sees
  `lyra: queued run #N (<workflow>)`.
- **`pull_request`:** opening, reopening and pushing to a pull request, with `types`
  (default `opened`, `synchronize`, `reopened`), `branches` (the base) and `paths` (the
  files the pull request changes). The workflow file comes from the head commit, and
  `github.ref` is `refs/pull/<n>/head`, with `head_ref`, `base_ref` and
  `github.event.pull_request`.
- **`workflow_dispatch`:** `lyra workflow run`. Jobs read `${{ inputs.name }}`.
- **`release`:** publishing a release queues `on: release` workflows from the tag's commit.
  `published`, `released` (not a prerelease) and `prereleased` are understood.
- **`schedule`:** every minute, Lyra checks the default branch's workflows for `cron`
  entries due that minute (UTC, five fields) and queues them as actor `lyra`. A restart
  within the same minute can queue a schedule twice.

## Jobs

- **Scheduling:** jobs are split and matrices expanded with the runner's own parser. A
  runner gets the oldest waiting job whose `runs-on` labels it all has.
- **Needs:** a job waits for the jobs it needs and gets their outputs and results. If one
  didn't succeed, the job is skipped without reaching a runner, unless it has an `if:`.
- **Job token:** `github.token` (also `secrets.GITHUB_TOKEN`) works while the job runs, with
  `write` access to its own repository only: for Git over HTTPS, the API and the `lyra` CLI.
  Writes are recorded as `job:<repo>#<run>`.
- **Secrets and variables:** jobs get the repository's secrets as `secrets.NAME` (masked in
  logs by the runner) and variables as `vars.NAME`. Secrets are encrypted with AES-256-GCM
  using the secrets key; their values are never returned, and managing them needs `admin`.
  Names are upper-cased and can't start with `GITHUB_`, `FORGEJO_` or `GITEA_`. Changing the
  key makes existing secrets unreadable, and jobs that use them fail with
  `lyra: the job couldn't start`.
- **Short `uses:` names** resolve against `--default-actions-url`
  (default `https://data.forgejo.org`).
- **Logs** are kept in the database while a job runs, then stored as one object per job
  attempt.
- **Dead runners:** a running job whose runner hasn't reported for `--job-stale-after`
  (default 10m) fails with a `lyra: the runner stopped reporting` log line.
- **Retention:** finished runs and their logs are deleted after `--actions-retention`
  (default 90 days; 0 keeps them).
- **Cancel** stops waiting jobs at once; runners stop running ones on their next report.
  `rerun --failed` reruns jobs that didn't succeed and the jobs that need them.
- **Events:** `run.queued`, `job.completed`, `run.completed` (with the result) and
  `run.cancelled`.

## Checks

`lyra pr view` includes `checks`: the jobs of the latest run of each workflow and event for
the pull request's head commit, with a `state` of `none`, `pending`, `success` or `failure`.

- When a run finishes and none is left pending for that commit, Lyra records `pr.checks` on
  the pull request, so `lyra pr wait --for checks` returns.
- When no workflow ran for the head at all, `lyra pr wait --for checks` returns at once with
  state `none`.
- The branch rule `--require-checks` blocks merging until the state is `success`.
