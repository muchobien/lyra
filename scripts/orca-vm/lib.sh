orca_vm_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
state_file="${ORCA_MOONBASE_STATE:-$orca_vm_dir/moonbase-state.json}"

state_value() {
  local value
  value="$(jq -r --arg key "$1" '.[$key] // empty' "$state_file")"
  case "$value" in
  "~/"*) value="$HOME/${value#"~/"}" ;;
  esac
  printf '%s' "$value"
}

merge_state() {
  local merged
  merged="$(jq "$@" "$state_file")"
  printf '%s\n' "$merged" >"$state_file"
}

kubeconfig="${ORCA_MOONBASE_KUBECONFIG:-$(state_value kubeconfig)}"
namespace="${ORCA_MOONBASE_NAMESPACE:-$(state_value namespace)}"
kubectl_bin="$(command -v kubectl)"

kube() {
  "$kubectl_bin" --kubeconfig "$kubeconfig" --namespace "$namespace" "$@"
}

ensure_namespace() {
  "$kubectl_bin" --kubeconfig "$kubeconfig" create namespace "$namespace" --dry-run=client -o yaml |
    "$kubectl_bin" --kubeconfig "$kubeconfig" apply -f - >&2
}

workspace_pod() {
  local image="$1" name="$2" public_key="$3"
  jq -n --arg image "$image" --arg name "$name" --arg key "$public_key" '{
    apiVersion: "v1",
    kind: "Pod",
    metadata: { name: $name, labels: { "app.kubernetes.io/name": "orca-workspace" } },
    spec: {
      automountServiceAccountToken: false,
      enableServiceLinks: false,
      securityContext: {
        runAsNonRoot: true,
        runAsUser: 1000,
        runAsGroup: 1000,
        fsGroup: 1000,
        seccompProfile: { type: "RuntimeDefault" }
      },
      containers: [{
        name: "workspace",
        image: $image,
        env: [{ name: "ORCA_SSH_PUBLIC_KEY", value: $key }],
        securityContext: { allowPrivilegeEscalation: false, capabilities: { drop: ["ALL"] } },
        resources: {
          requests: { cpu: "500m", memory: "1Gi" },
          limits: { memory: "6Gi", "ephemeral-storage": "30Gi" }
        },
        readinessProbe: { exec: { command: ["nc", "-z", "127.0.0.1", "2222"] }, periodSeconds: 2 },
        volumeMounts: [
          { name: "git", mountPath: "/run/orca/git", readOnly: true },
          { name: "agent-auth", mountPath: "/run/orca/agent-auth", readOnly: true }
        ]
      }],
      volumes: [
        { name: "git", secret: { secretName: ($name + "-git"), defaultMode: 288 } },
        { name: "agent-auth", secret: { secretName: "agent-auth", optional: true, defaultMode: 288 } }
      ]
    }
  }'
}
