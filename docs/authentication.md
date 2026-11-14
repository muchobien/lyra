# Authentication

## The tailnet login

Requests from a `--trusted-proxy` address carrying `Tailscale-User-Login` act as that login
with full access, if it is listed in `--allow-login`. The header is ignored from any other
address. This is how you use the web UI, behind a Tailscale proxy.

## Tokens

Agents and scripts use tokens:

- as `Authorization: Bearer <token>` for the API and the CLI (`LYRA_TOKEN`);
- as the password, with any user name, for Git over HTTPS and `docker login`.

A token has:

- a **scope**: `read` < `write` < `admin`;
- an optional **list of repositories**: `owner/repo`, or `owner/*` for all of an owner's
  repositories. A token without a list is *unrestricted*;
- an optional **expiry**.

Only a token's SHA-256 hash is stored. Managing tokens, keys, owners and runners needs an
unrestricted admin token.

```sh
lyra token create --name claude --scope write --repo 'krosf/*' --expires 720h
```

## Anonymous clones

Every request needs one of the above, except cloning and fetching repositories of the
owners in `--actions-owners`, which works without credentials so runners can fetch actions
from Lyra. See [hosting actions](actions.md#hosting-actions).

## Agents and sessions

`Lyra-Agent` and `Lyra-Session` headers (`LYRA_AGENT` and `LYRA_SESSION` in the CLI) are
recorded with every write and push, so you can tell agents apart. They also decide who
"someone else" is when [waiting for events](events.md), and `@me` resolves to the agent
name when one is set.
