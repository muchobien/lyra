# Lyra

A self-hosted Git forge for one person and their agents: one Go binary, SQLite and bare
repositories on disk. Agents work through the `lyra` CLI; you use the web UI. It has issues,
pull requests with reviews and branch rules, releases, a container registry, Actions and
backups.

## Documentation

- [`docs/`](docs/README.md): how Lyra works, how to run it, and how to develop it.
- `lyra --help`, `lyra <command> --help` and `lyra serve --help`: every command and flag.
- [`api/openapi.yaml`](api/openapi.yaml): the HTTP API, also served at `/api/openapi.yaml`.

## Install

```sh
packslip install github.com/muchobien/lyra   # verified release binary
go build -o bin/lyra ./cmd/lyra              # or from source
```

The container image is `ghcr.io/muchobien/lyra`.

## Run

```sh
lyra serve --data /var/lib/lyra \
  --trusted-proxy 10.42.0.0/16 --allow-login you@example.com
```

The server needs `git` on `PATH`. You sign in through a Tailscale proxy; agents use tokens
([authentication](docs/authentication.md)).

## First steps

```sh
lyra owner create krosf
lyra repo create krosf/demo
lyra token create --name claude --scope write --repo 'krosf/*'
git clone https://lyra.example/krosf/demo.git
```
