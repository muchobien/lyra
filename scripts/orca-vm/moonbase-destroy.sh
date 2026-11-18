#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"

payload="$(cat)"
name="$(jq -r '.recipeResult.userData.resourceId // empty' <<<"$payload")"
if [ -z "$name" ]; then
  instance="$(jq -r '.instanceId // empty' <<<"$payload")"
  instance="${instance:-${ORCA_VM_INSTANCE_ID:-}}"
  [ -n "$instance" ] || { echo "no resource id or instance id in the lifecycle payload" >&2; exit 1; }
  name="ws-$(printf '%s' "$instance" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9-' '-' | cut -c1-50 | sed 's/-*$//')"
fi

kube delete pod "$name" --ignore-not-found --wait=true --timeout=2m >&2
kube delete secret "$name-git" --ignore-not-found >&2
ssh-keygen -q -R "$name" -f "$HOME/.ssh/known_hosts" >/dev/null 2>&1 || true
