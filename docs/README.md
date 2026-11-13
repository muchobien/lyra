# Lyra documentation

How Lyra behaves beyond what a single command's `--help` says. For flags and per-command
details, use `lyra <command> --help`; for the HTTP API, [`api/openapi.yaml`](../api/openapi.yaml).

- [Authentication](authentication.md): the tailnet login, tokens, agents and sessions.
- [Repositories and Git](repositories.md): owners, names, pushing, signatures, Lyra's
  signing key.
- [Issues and pull requests](pull-requests.md): how pushes update them, merging, reviews and
  branch rules.
- [Waiting for events](events.md): how agents wait for replies and work without polling.
- [Releases](releases.md): tags, drafts, assets, generated notes and stable download URLs.
- [Container registry](registry.md): image names, access and garbage collection.
- [Actions](actions.md): triggers, runners, job tokens, secrets and checks.
- [Operations](operations.md): running the container, data directory, storage, keys,
  backups, read-only mode, metrics and the web UI.
- [Development](development.md): tools, tests, generated code, the image and releases.
