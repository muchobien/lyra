# Operations

## Releases and the container image

Each version tag publishes a GitHub release with static binaries for `linux-amd64`,
`linux-arm64` and `darwin-arm64`, signed with [packslip](https://packslip.dev).
`packslip install github.com/muchobien/lyra` checks the signature; to check a download
yourself, run `packslip verify packslip.sigstore.json --artifact <archive>`.

`ghcr.io/muchobien/lyra` is built for `linux/amd64` and `linux/arm64`, tagged `main` and
`sha-<short>` from `main`, and `X.Y.Z`, `X.Y` and `latest` from version tags. It runs
`lyra serve` as a non-root user with data in the `/data` volume:

```sh
docker run -p 8080:8080 -v lyra-data:/data \
  -e LYRA_TRUSTED_PROXY=10.42.0.0/16 -e LYRA_ALLOW_LOGIN=you@example.com \
  ghcr.io/muchobien/lyra:main
```

## Data directory

The data directory (`--data`, `/data` in the image) holds:

- `lyra.db`, the SQLite database;
- `repos/<owner>/<repo>.git`, the bare repositories;
- `hooks/pre-receive` and `hooks/post-receive`, written at startup and applied through
  `core.hooksPath`;
- `blobs`, with the default storage;
- the generated `lyra_signing_key` and `secrets_key`, unless `--signing-key-file` and
  `--secrets-key-file` are set. Lyra warns when it generates them.

## Storage

Release assets, job logs and registry layers are stored in `<data>/blobs` by default. To
store them in S3 (Garage, for example), pass
`--storage s3 --s3-endpoint http://garage:3900 --s3-bucket lyra`, with credentials in
`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` (`--s3-region` defaults to `garage`). Lyra
uses path-style requests and serves every download itself, so the bucket doesn't need to be
reachable by clients.

## Backups

```sh
lyra admin backup --output lyra.tar
lyra restore lyra.tar --data /new/data   # offline, into an empty directory
lyra admin storage-check                 # stored files the database refers to but storage lacks
lyra admin status                        # read-only mode and the last backup
```

- **A backup holds** `lyra.db` (a consistent copy made with SQLite's `VACUUM INTO`), every
  bare repository, and `manifest.json`: each repository's refs, and the stored files and
  images the database refers to.
- **Consistency:** writes (pushes, API and UI writes, registry pushes, runner reports, the
  scheduler) wait while the database is copied and the repositories are hard-linked into a
  staging directory; then the tar is streamed. `admin status` reports the pause.
- **Not included:** the signing and secrets keys, and the files in storage. Back up the S3
  bucket or `blobs` directory separately.
- **Checks:** `admin backup` fails if the tar ends without its manifest. `restore` refuses a
  non-empty directory, checks the database's integrity and schema version, runs `git fsck`
  on every repository, and compares their refs with the manifest. After starting
  `lyra serve` on the restored directory, with the same keys and storage,
  `admin storage-check` exits 1 if storage lacks any file the database refers to.
- These commands need an unrestricted admin token. Each backup is recorded as
  `backup.created`.

## Read-only mode

`lyra admin readonly on` refuses every write with 503 while reads and clones keep working.
It lives in memory, so a restart turns it off. `lyra admin registry-gc` turns it on while it
runs.

## Metrics

`--metrics-listen :9090` serves Prometheus metrics at `/metrics` on a separate listener, so
they aren't reachable through the main one:

- `lyra_repos`, `lyra_runs{status}`, `lyra_jobs{status}`, `lyra_runners_online` and
  `lyra_read_only`, read from the database at each scrape;
- `lyra_backup_last_success_timestamp_seconds`, `lyra_backup_last_bytes` and
  `lyra_backup_last_pause_seconds`, for alerting on stale backups;
- `lyra_http_requests_total{method,code}` and `lyra_http_request_duration_seconds{method}`;
- Go runtime and process metrics.

## Web UI

The [web UI](web-ui.md) is embedded in the binary. API, Git transport, runner protocol,
registry, raw-file and release-download requests retain their own routes. Missing assets
are never served an application shell.

Requests that change data and come from another site are rejected with Go's
`http.CrossOriginProtection` (`Sec-Fetch-Site` and `Origin`). Without it, any page could
post forms as you, because the proxy adds the tailnet login header to every request. The
CLI and git send neither header and aren't affected.
