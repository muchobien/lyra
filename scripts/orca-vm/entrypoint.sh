#!/usr/bin/env bash
set -euo pipefail

mkdir -p "$HOME/.ssh/host" "$HOME/.claude" "$HOME/.codex"
chmod 700 "$HOME/.ssh" "$HOME/.ssh/host"

if [ ! -f "$HOME/.ssh/host/ssh_host_ed25519_key" ]; then
  ssh-keygen -q -t ed25519 -N '' -f "$HOME/.ssh/host/ssh_host_ed25519_key"
fi

printf '%s\n' "${ORCA_SSH_PUBLIC_KEY:?ORCA_SSH_PUBLIC_KEY is required}" >"$HOME/.ssh/authorized_keys"
chmod 600 "$HOME/.ssh/authorized_keys"
printf 'PATH=%s\nMISE_DATA_DIR=/opt/mise\nMISE_CACHE_DIR=/opt/mise/cache\nMISE_YES=1\n' "$PATH" >"$HOME/.ssh/environment"

auth=/run/orca/agent-auth
if [ -f "$auth/claude-credentials.json" ] && [ ! -f "$HOME/.claude/.credentials.json" ]; then
  install -m 600 "$auth/claude-credentials.json" "$HOME/.claude/.credentials.json"
fi
if [ -f "$auth/claude.json" ] && [ ! -f "$HOME/.claude.json" ]; then
  install -m 600 "$auth/claude.json" "$HOME/.claude.json"
fi
if [ -f "$auth/codex-auth.json" ] && [ ! -f "$HOME/.codex/auth.json" ]; then
  install -m 600 "$auth/codex-auth.json" "$HOME/.codex/auth.json"
fi

exec /usr/sbin/sshd -D -e
