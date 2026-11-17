import { useContext, useState, type SubmitEvent, type MouseEvent, type ToggleEvent } from "react";
import { useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  GitBranchIcon,
  KeyIcon,
  LockIcon,
  PaintbrushIcon,
  ServerIcon,
  SlidersIcon,
} from "@primer/octicons-react";
import { IdentityContext, type Rule } from "@/api";
import {
  useListVariables,
  useListSecrets,
  useListRules,
  useListLabels,
  useListTokens,
  useListKeys,
  useGetSigningKey,
  useListRunners,
  useListRunnerTokens,
  getListTokensQueryKey,
  getListKeysQueryKey,
  getGetSigningKeyQueryKey,
  getListRunnersQueryKey,
  getListRunnerTokensQueryKey,
  useCreateToken,
  useCreateRunnerToken,
  setSecret,
  setVariable,
  setRule,
  createLabel,
  createKey,
  deleteSecret,
  deleteVariable,
  deleteRule,
  deleteLabel,
  revokeToken,
  deleteKey,
  deleteRunner,
  deleteRunnerToken,
} from "@/api/generated/lyra";
import { formValue } from "@/utils/forms";
import { repositoryQueries } from "@/api/repositoryQueries";

const instanceSettingsQueries = new Set<string>([
  getListTokensQueryKey()[0],
  getListKeysQueryKey()[0],
  getGetSigningKeyQueryKey()[0],
  getListRunnersQueryKey()[0],
  getListRunnerTokensQueryKey()[0],
]);

type ConfigChange = { save: () => Promise<unknown>; dialog: string };

type Deletion = { remove: () => Promise<unknown>; title: string; description: string };

type MintedSecret = { secret: string; title: string; url?: string };

type SettingsEditor = { name: string; value: string; version: number; rule: Rule | null };

export type SettingsProps = { scope: "instance" | "repository" };

export function useConnect({ scope }: Pick<SettingsProps, "scope">) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ from: scope === "instance" ? "/settings" : "/$owner/$repo/settings" });
  const identity = useContext(IdentityContext);
  const queryClient = useQueryClient();
  const name = `${owner}/${repo}`;
  const canWrite = identity?.scope !== "read";
  const admin = identity?.scope === "admin";
  const instanceAdmin = admin && !identity?.repos.length;
  const instance = scope === "instance";
  let fallbackArea = admin ? "secrets" : "variables";
  if (instance) {
    fallbackArea = "access";
  }
  const area = search.section || fallbackArea;
  const [editor, setEditor] = useState<SettingsEditor>({
    name: "",
    value: "",
    version: 0,
    rule: null,
  });
  const [deletion, setDeletion] = useState<Deletion | null>(null);
  const [oneTime, setOneTime] = useState<MintedSecret | null>(null);
  const [validationError, setValidationError] = useState<Error | null>(null);
  const variables = useListVariables(owner, repo, { query: { enabled: area === "variables" } });
  const secrets = useListSecrets(owner, repo, { query: { enabled: area === "secrets" && admin } });
  const rules = useListRules(owner, repo, { query: { enabled: area === "rules" } });
  const labels = useListLabels(owner, repo, { query: { enabled: area === "labels" } });
  const tokens = useListTokens({ query: { enabled: area === "access" && instanceAdmin } });
  const keys = useListKeys({ query: { enabled: area === "access" && instanceAdmin } });
  const serverKey = useGetSigningKey({ query: { enabled: area === "access" && instanceAdmin } });
  const runners = useListRunners({
    query: {
      enabled: area === "runners" && instanceAdmin,
      refetchInterval: area === "runners" && instanceAdmin ? 15000 : false,
    },
  });
  const runnerTokens = useListRunnerTokens({
    query: { enabled: area === "runners" && instanceAdmin },
  });
  const changing = useMutation({
    gcTime: 0,
    mutationFn: (change: ConfigChange) => change.save(),
    onSuccess: async (_data, change) => {
      const dialog = document.querySelector<HTMLDialogElement>(`#${change.dialog}`);
      dialog?.close();
      dialog?.querySelector("form")?.reset();
      await refresh();
    },
  });
  const removing = useMutation({
    mutationFn: () => {
      if (!deletion) {
        throw new Error("Choose an item to delete.");
      }
      return deletion.remove();
    },
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#delete-setting")?.close();
      await refresh();
    },
  });
  const mintToken = useCreateToken({
    mutation: {
      gcTime: 0,
      onSuccess: (created) => {
        document.querySelector<HTMLDialogElement>("#new-token")?.close();
        setOneTime({ title: created.token.name, secret: created.secret });
        document.querySelector<HTMLDialogElement>("#one-time-secret")?.showModal();
        void refresh();
      },
    },
  });
  const mintRunnerToken = useCreateRunnerToken({
    mutation: {
      gcTime: 0,
      onSuccess: (created) => {
        document.querySelector<HTMLDialogElement>("#new-runner-token")?.close();
        setOneTime({
          title: "Runner registration token",
          secret: created.secret,
          url: created.url,
        });
        document.querySelector<HTMLDialogElement>("#one-time-secret")?.showModal();
        void refresh();
      },
    },
  });
  function prepareValue(event: MouseEvent<HTMLButtonElement>) {
    const selected = event.currentTarget.dataset.name || "";
    setEditor({
      name: selected,
      value: variables.data?.find((variable) => variable.name === selected)?.value || "",
      rule: null,
      version: editor.version + 1,
    });
    changing.reset();
  }
  function valueSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    changing.mutate({
      save: () =>
        area === "secrets"
          ? setSecret(owner, repo, formValue(form, "name"), { value: formValue(form, "value") })
          : setVariable(owner, repo, formValue(form, "name"), { value: formValue(form, "value") }),
      dialog: "value-editor",
    });
  }
  function prepareRule(event: MouseEvent<HTMLButtonElement>) {
    setEditor({
      name: "",
      value: "",
      version: editor.version + 1,
      rule: rules.data?.find((rule) => rule.id === Number(event.currentTarget.dataset.id)) || null,
    });
    changing.reset();
    setValidationError(null);
  }
  function ruleSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const methods = form
      .getAll("method")
      .filter((value): value is "squash" | "ff" => value === "squash" || value === "ff");
    if (!methods.length) {
      setValidationError(new Error("Choose at least one merge method."));
      return;
    }
    setValidationError(null);
    changing.mutate({
      dialog: "rule-editor",
      save: () =>
        setRule(owner, repo, {
          pattern: formValue(form, "pattern"),
          require_signed: form.has("require_signed"),
          require_approval: form.has("require_approval"),
          require_checks: form.has("require_checks"),
          allow_force_push: form.has("allow_force_push"),
          allow_deletion: form.has("allow_deletion"),
          merge_methods: methods,
        }),
    });
  }
  function labelSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    changing.mutate({
      dialog: "label-editor",
      save: () =>
        createLabel(owner, repo, {
          name: formValue(form, "name"),
          color: formValue(form, "color"),
          description: formValue(form, "description"),
        }),
    });
  }
  function keySubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    changing.mutate({
      dialog: "key-editor",
      save: () =>
        createKey({
          title: formValue(form, "title"),
          agent: formValue(form, "agent"),
          public_key: formValue(form, "public_key"),
          signing: true,
        }),
    });
  }
  function tokenSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const scope = formValue(form, "scope");
    if (scope !== "read" && scope !== "write" && scope !== "admin") {
      return;
    }
    mintToken.mutate({
      data: {
        name: formValue(form, "name"),
        scope,
        expires_in: formValue(form, "expiry"),
        repos: formValue(form, "repos")
          .split(/[\n,]/)
          .map((repo) => repo.trim())
          .filter(Boolean),
      },
    });
  }
  function runnerTokenSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    mintRunnerToken.mutate({ data: { name: formValue(form, "name") } });
  }
  function resetChange() {
    changing.reset();
  }
  function resetToken() {
    mintToken.reset();
  }
  function resetRunnerToken() {
    mintRunnerToken.reset();
  }
  function prepareDeletion(event: MouseEvent<HTMLButtonElement>) {
    const {
      kind = "",
      name = "",
      id = "",
      title = "Delete item?",
      description = "This item will be removed.",
    } = event.currentTarget.dataset;
    let remove: () => Promise<unknown>;
    switch (kind) {
      case "secrets":
        remove = () => deleteSecret(owner, repo, name);
        break;
      case "variables":
        remove = () => deleteVariable(owner, repo, name);
        break;
      case "rules":
        remove = () => deleteRule(owner, repo, { pattern: name });
        break;
      case "labels":
        remove = () => deleteLabel(owner, repo, name);
        break;
      case "tokens":
        remove = () => revokeToken(Number(id));
        break;
      case "keys":
        remove = () => deleteKey(Number(id));
        break;
      case "runners":
        remove = () => deleteRunner(Number(id));
        break;
      case "runner-tokens":
        remove = () => deleteRunnerToken(Number(id));
        break;
      default:
        return;
    }
    setDeletion({ remove, title, description });
    removing.reset();
  }
  function deleteSetting() {
    removing.mutate();
  }
  function clearValue(event: ToggleEvent<HTMLDialogElement>) {
    if (event.newState === "closed") {
      event.currentTarget.querySelector("form")?.reset();
      changing.reset();
    }
  }
  function clearToken(event: ToggleEvent<HTMLDialogElement>) {
    if (event.newState === "closed") {
      event.currentTarget.querySelector("form")?.reset();
      mintToken.reset();
      mintRunnerToken.reset();
    }
  }
  function clearOneTime(event: ToggleEvent<HTMLDialogElement>) {
    if (event.newState === "closed") {
      setOneTime(null);
      mintToken.reset();
      mintRunnerToken.reset();
    }
  }
  function retry() {
    void refresh();
  }
  function refresh() {
    return queryClient.invalidateQueries(
      instance
        ? {
            predicate: ({ queryKey: [key] }) =>
              typeof key === "string" && instanceSettingsQueries.has(key),
          }
        : repositoryQueries(owner, repo),
    );
  }
  const areas = [
    { id: "secrets", name: "Secrets", icon: LockIcon, visible: !instance && admin },
    { id: "variables", name: "Variables", icon: SlidersIcon, visible: !instance },
    { id: "rules", name: "Branch rules", icon: GitBranchIcon, visible: !instance },
    { id: "labels", name: "Labels", icon: PaintbrushIcon, visible: !instance },
    { id: "access", name: "Access", icon: KeyIcon, visible: instance && instanceAdmin },
    { id: "runners", name: "Runners", icon: ServerIcon, visible: instance && instanceAdmin },
  ];
  const selectedArea = areas.find((entry) => entry.id === area && entry.visible);
  const loadError = {
    variables: variables.error,
    secrets: secrets.error,
    rules: rules.error,
    labels: labels.error,
    access: tokens.error || keys.error || serverKey.error,
    runners: runners.error || runnerTokens.error,
  }[area];

  return {
    owner,
    repo,
    name,
    canWrite,
    admin,
    instance,
    area,
    editor,
    deletion,
    oneTime,
    validationError,
    variables,
    secrets,
    rules,
    labels,
    tokens,
    keys,
    serverKey,
    runners,
    runnerTokens,
    changing,
    removing,
    mintToken,
    mintRunnerToken,
    prepareValue,
    valueSubmit,
    prepareRule,
    ruleSubmit,
    labelSubmit,
    keySubmit,
    tokenSubmit,
    runnerTokenSubmit,
    resetChange,
    resetToken,
    resetRunnerToken,
    prepareDeletion,
    deleteSetting,
    clearValue,
    clearToken,
    clearOneTime,
    retry,
    areas,
    selectedArea,
    loadError,
  };
}
