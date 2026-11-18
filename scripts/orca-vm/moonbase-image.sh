#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/lib.sh"

repo_root="$(cd "$orca_vm_dir/../.." && pwd)"
repository="${ORCA_MOONBASE_IMAGE_REPOSITORY:-$(state_value imageRepository)}"
tag="$(date -u +%Y%m%d%H%M%S)"
context="$(mktemp -d)"
trap 'rm -rf "$context"' EXIT

cp "$orca_vm_dir"/{Dockerfile,entrypoint.sh,gitconfig,profile.sh,sshd_config} "$context/"
mkdir -p "$context/repo/web"
cp "$repo_root"/{mise.toml,go.mod,go.sum} "$context/repo/"
cp "$repo_root"/web/{package.json,pnpm-lock.yaml,pnpm-workspace.yaml} "$context/repo/web/"

docker buildx build --platform linux/amd64 --tag "$repository:$tag" \
  --metadata-file "$context/metadata.json" --push "$context" >&2

digest="$(jq -r '."containerimage.digest" // empty' "$context/metadata.json")"
[ -n "$digest" ] || { echo "could not read the pushed image digest" >&2; exit 1; }

merge_state --arg image "$repository@$digest" '.image = $image'
cat "$state_file"
