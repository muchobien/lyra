# Development

```sh
mise install          # Go, Node, pnpm, sqlc, GoReleaser and linters
mise tasks            # every task CI runs
mise run fmt
mise run lint
mise run test         # end-to-end tests in e2e/
```

## Frontend

The UI uses React, TanStack Router and Query, CSS Modules and Vite+, with exact frontend
versions and a frozen pnpm lockfile. Go retains authorization, Git operations, Markdown,
syntax highlighting, diff parsing, signatures and merge rules.

`mise run build`, `lint` and `test` build the frontend assets embedded in
`internal/webui/dist` before compiling Go. A clean checkout needs `mise run ui:build`
before direct Go build/test commands. Node is a build dependency only.

`mise run ui:check` runs Vite+ formatting and lint checks plus TypeScript checking.
`mise run ui:fmt` formats the frontend. Current stable Chrome is the browser target;
native dialogs, invoker commands, customizable selects and anchored popovers need no
browser polyfills.

For UI development, start a local Lyra server and run `pnpm --dir web dev`.
`LYRA_DEV_URL` chooses the backend (default `http://127.0.0.1:8080`). `LYRA_DEV_LOGIN`
optionally supplies a trusted-proxy login for a locally configured backend. These
development-server variables are never included in the browser bundle.

## Tests

`go test ./...` runs the end-to-end tests in `e2e/`. They need `git` and `ssh-keygen` on
`PATH`, build the `lyra` binary, start servers on free ports with temporary data
directories, and drive them with `git`, the CLI and HTTP. The S3 test needs Docker and is
skipped without it.

The tests build the binary at run time, so Go's test cache doesn't see server changes: use
`go test -count=1 ./e2e/` after changing the code.

## Generated code

- **Database:** the schema lives in goose migrations under `internal/store/migrations`,
  applied at startup, and queries in `internal/store/queries.sql`. After changing either,
  run `sqlc generate` and commit `internal/store/queries`. `mise run sqlc:check` fails when
  it's out of date.
- **API:** routes are [huma](https://huma.rocks) operations in `internal/server/openapi.go`,
  and huma builds the OpenAPI document from them and the Go types of their inputs and
  outputs. After changing the API, run `mise run openapi` and commit `api/openapi.yaml`.
  `mise run openapi:check` fails when it's out of date.

## Image

The `Dockerfile` unpacks an archive from `dist/` instead of compiling, so build the image
with `mise run image`, not a bare `docker build`.

## Releases

Pushing a `v*` tag runs the release workflow. GoReleaser builds the frontend before
compiling the embedded binaries, then builds the archives and the
changelog, the container image is built from those archives and pushed, packslip signs and
attests the archives with the workflow's identity, and the last job creates the GitHub
release with the archives, `checksums.txt` and `packslip.sigstore.json`. Pushes to `main`
and pull requests run the same build as a snapshot and publish nothing.
