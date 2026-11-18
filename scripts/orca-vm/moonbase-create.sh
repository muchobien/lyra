#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"

image="${ORCA_MOONBASE_IMAGE:-$(state_value image)}"
[ -n "$image" ] || { echo "no image in $state_file; run scripts/orca-vm/moonbase-image.sh first" >&2; exit 1; }
repo_url="${ORCA_MOONBASE_REPO_URL:-$(state_value repoUrl)}"
repo_ref="${ORCA_MOONBASE_REPO_REF:-$(state_value repoRef)}"
project_root="$(state_value projectRoot)"
ssh_key="${ORCA_MOONBASE_SSH_KEY:-$(state_value sshKey)}"
instance="${ORCA_VM_INSTANCE_ID:?ORCA_VM_INSTANCE_ID is required}"
name="ws-$(printf '%s' "$instance" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9-' '-' | cut -c1-50 | sed 's/-*$//')"

token="${GH_TOKEN:-${GITHUB_TOKEN:-$(gh auth token)}}"
[ -n "$token" ] || { echo "no GitHub token: set GH_TOKEN or sign in with gh auth login" >&2; exit 1; }

if [ ! -f "$ssh_key" ]; then
  ssh-keygen -q -t ed25519 -N '' -C orca-lyra-moonbase -f "$ssh_key" >&2
fi

cleanup() {
  kube delete pod "$name" --ignore-not-found --wait=false >&2 || true
  kube delete secret "$name-git" --ignore-not-found >&2 || true
}
trap cleanup ERR

ensure_namespace
printf '%s' "$token" | kube create secret generic "$name-git" --from-file=token=/dev/stdin >&2
workspace_pod "$image" "$name" "$(cat "$ssh_key.pub")" | kube apply -f - >&2
kube wait --for=condition=Ready "pod/$name" --timeout=10m >&2

pod_uid="$(kube get pod "$name" -o jsonpath='{.metadata.uid}')"
kube patch secret "$name-git" --type=merge -p "$(jq -n --arg name "$name" --arg uid "$pod_uid" \
  '{metadata: {ownerReferences: [{apiVersion: "v1", kind: "Pod", name: $name, uid: $uid}]}}')" >&2

remote_sync='set -euo pipefail
export GIT_TERMINAL_PROMPT=0
[ -d "$project_root/.git" ] || git clone --quiet "$repo_url" "$project_root"
cd "$project_root"
git fetch --quiet origin "$repo_ref"
git checkout --quiet -B "$repo_ref" FETCH_HEAD
git branch --quiet --set-upstream-to "origin/$repo_ref"
mise trust --quiet
[ -z "$git_name" ] || git config --global user.name "$git_name"
[ -z "$git_email" ] || git config --global user.email "$git_email"'
kube exec "$name" -- env project_root="$project_root" repo_url="$repo_url" repo_ref="$repo_ref" \
  git_name="$(git config user.name || true)" git_email="$(git config user.email || true)" \
  bash -lc "$remote_sync" >&2

host_key="$(kube exec "$name" -- cat /home/dev/.ssh/host/ssh_host_ed25519_key.pub | cut -d' ' -f1,2)"
known_hosts="$HOME/.ssh/known_hosts"
touch "$known_hosts"
ssh-keygen -q -R "$name" -f "$known_hosts" >/dev/null 2>&1 || true
printf '%s %s\n' "$name" "$host_key" >>"$known_hosts"

proxy_command="$(printf '%q ' "$kubectl_bin" --kubeconfig "$kubeconfig" --namespace "$namespace" \
  exec -i "$name" -- nc 127.0.0.1 2222)"
proxy_command="${proxy_command% }"
ssh -i "$ssh_key" -o IdentitiesOnly=yes -o BatchMode=yes -o StrictHostKeyChecking=yes \
  -o "ProxyCommand=$proxy_command" "dev@$name" "test -d $(printf '%q' "$project_root/.git")" >&2

jq -n --arg name "$name" --arg key "$ssh_key" --arg proxy "$proxy_command" --arg root "$project_root" \
  --arg namespace "$namespace" '{
  schemaVersion: 1,
  connection: {
    type: "ssh",
    projectRoot: $root,
    target: {
      label: ("moonbase " + $name),
      host: $name,
      port: 22,
      username: "dev",
      identityFile: $key,
      identitiesOnly: true,
      proxyCommand: $proxy
    }
  },
  userData: { provider: "moonbase", resourceId: $name, namespace: $namespace }
}'
