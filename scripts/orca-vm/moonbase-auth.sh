#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"

name=agent-auth-login
image="${ORCA_MOONBASE_IMAGE:-$(state_value image)}"
[ -n "$image" ] || { echo "no image in $state_file; run scripts/orca-vm/moonbase-image.sh first" >&2; exit 1; }

case "${1:-}" in
start)
  ensure_namespace
  kube create secret generic "$name-git" --from-literal=token= --dry-run=client -o yaml | kube apply -f - >&2
  workspace_pod "$image" "$name" "unused" | kube apply -f - >&2
  kube wait --for=condition=Ready "pod/$name" --timeout=10m >&2
  login="$(printf '%q ' "$kubectl_bin" --kubeconfig "$kubeconfig" --namespace "$namespace" exec -it "$name" --)"
  cat >&2 <<EOF
Sign in to both agents in the login pod, then run: $0 save
  ${login}bash -lc 'claude auth login'
  ${login}bash -lc 'codex login --device-auth'
EOF
  ;;
save)
  kube exec "$name" -- bash -lc 'claude auth status >/dev/null' ||
    { echo "claude is not signed in in $name" >&2; exit 1; }
  kube exec "$name" -- bash -lc 'codex login status >/dev/null' ||
    { echo "codex is not signed in in $name" >&2; exit 1; }
  kube create secret generic agent-auth \
    --from-file=claude-credentials.json=<(kube exec "$name" -- cat /home/dev/.claude/.credentials.json) \
    --from-file=claude.json=<(kube exec "$name" -- cat /home/dev/.claude.json) \
    --from-file=codex-auth.json=<(kube exec "$name" -- cat /home/dev/.codex/auth.json) \
    --dry-run=client -o yaml | kube apply -f - >&2
  kube delete pod "$name" --ignore-not-found >&2
  kube delete secret "$name-git" --ignore-not-found >&2
  echo "saved agent credentials to secret $namespace/agent-auth" >&2
  ;;
*)
  echo "usage: $0 start|save" >&2
  exit 2
  ;;
esac
