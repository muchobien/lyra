# Development

```sh
mise install          # Go, sqlc, golangci-lint, GoReleaser, packslip, workflow linters
mise tasks            # every task CI runs
mise run fmt
mise run lint
mise run test         # end-to-end tests in e2e/
```

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
  outputs. The CLI calls the API through `internal/cli/lyraapi`, a client that
  [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) generates from that document,
  so a server change that breaks the CLI fails to compile. After changing the API, run
  `mise run openapi` and commit `api/openapi.yaml` and `internal/cli/lyraapi/client.gen.go`.
  `mise run openapi:check` fails when either is out of date.

## Image

The `Dockerfile` unpacks an archive from `dist/` instead of compiling, so build the image
with `mise run image`, not a bare `docker build`.

## Releases

Pushing a `v*` tag runs the release workflow. GoReleaser builds the archives and the
changelog, the container image is built from those archives and pushed, packslip signs and
attests the archives with the workflow's identity, and the last job creates the GitHub
release with the archives, `checksums.txt` and `packslip.sigstore.json`. Pushes to `main`
and pull requests run the same build as a snapshot and publish nothing.
