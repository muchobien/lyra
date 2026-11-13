import { useContext, useEffect, useState } from "react";
import type { ChangeEvent, FormEvent, MouseEvent, ToggleEvent } from "react";
import { Link, useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  GitBranch,
  KeyRound,
  LockKeyhole,
  Palette,
  Plus,
  Server,
  Trash2,
  Variable as VariableIcon,
} from "lucide-react";
import {
  api,
  IdentityContext,
  repositoryAPI,
  write,
  type AccessToken,
  type Label,
  type Rule,
  type Runner,
  type RunnerToken,
  type Secret,
  type SigningKey,
  type Variable,
} from "./api";
import { CopyButton, ErrorNotice, formValue } from "./Controls";
import { ConfirmDialog, DateTime, DialogClose, FormError } from "./Workspace";
import controls from "./Controls.module.css";
import common from "./Workspace.module.css";
import styles from "./Settings.module.css";

type ConfigChange = { path: string; method: string; body?: unknown; dialog: string };
type Deletion = { path: string; title: string; description: string };
type MintedSecret = { secret: string; title: string; url?: string };
type TokenCreation = { name: string; scope: string; repos: string[]; expires_in: string };
type ServerSigningKey = { fingerprint: string; public_key: string; allowed_signers: string };
export function Settings() {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const identity = useContext(IdentityContext);
  const queryClient = useQueryClient();
  const name = `${owner}/${repo}`;
  const base = repositoryAPI(name);
  const canWrite = identity?.scope !== "read";
  const admin = identity?.scope === "admin";
  const instanceAdmin = admin && !identity?.repos.length;
  const area = search.section || (admin ? "secrets" : "variables");
  const [editingName, setEditingName] = useState("");
  const [editingValue, setEditingValue] = useState("");
  const [editorVersion, setEditorVersion] = useState(0);
  const [editingRule, setEditingRule] = useState<Rule | null>(null);
  const [deletion, setDeletion] = useState<Deletion | null>(null);
  const [oneTime, setOneTime] = useState<MintedSecret | null>(null);
  const [validationError, setValidationError] = useState<Error | null>(null);
  const variables = useQuery({
    queryKey: [`${base}/variables`],
    queryFn: ({ signal }) => api<Variable[]>(`${base}/variables`, signal),
    enabled: area === "variables",
  });
  const secrets = useQuery({
    queryKey: [`${base}/secrets`],
    queryFn: ({ signal }) => api<Secret[]>(`${base}/secrets`, signal),
    enabled: area === "secrets" && admin,
  });
  const rules = useQuery({
    queryKey: [`${base}/rules`],
    queryFn: ({ signal }) => api<Rule[]>(`${base}/rules`, signal),
    enabled: area === "rules",
  });
  const labels = useQuery({
    queryKey: [`${base}/labels`],
    queryFn: ({ signal }) => api<Label[]>(`${base}/labels`, signal),
    enabled: area === "labels",
  });
  const tokens = useQuery({
    queryKey: ["/api/tokens"],
    queryFn: ({ signal }) => api<AccessToken[]>("/api/tokens", signal),
    enabled: area === "access" && instanceAdmin,
  });
  const keys = useQuery({
    queryKey: ["/api/keys"],
    queryFn: ({ signal }) => api<SigningKey[]>("/api/keys", signal),
    enabled: area === "access" && instanceAdmin,
  });
  const serverKey = useQuery({
    queryKey: ["/api/signing-key"],
    queryFn: ({ signal }) => api<ServerSigningKey>("/api/signing-key", signal),
    enabled: area === "access" && instanceAdmin,
  });
  const runners = useQuery({
    queryKey: ["/api/runners"],
    queryFn: ({ signal }) => api<Runner[]>("/api/runners", signal),
    enabled: area === "runners" && instanceAdmin,
    refetchInterval: area === "runners" && instanceAdmin ? 15000 : false,
  });
  const runnerTokens = useQuery({
    queryKey: ["/api/runners/tokens"],
    queryFn: ({ signal }) => api<RunnerToken[]>("/api/runners/tokens", signal),
    enabled: area === "runners" && instanceAdmin,
  });
  const changing = useMutation({
    gcTime: 0,
    mutationFn: (change: ConfigChange) => write(change.path, change.method, change.body),
    onSuccess: async (_data, change) => {
      document.querySelector<HTMLDialogElement>(`#${change.dialog}`)?.close();
      await queryClient.invalidateQueries();
    },
  });
  const removing = useMutation({
    mutationFn: () => write(deletion?.path || "", "DELETE"),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#delete-setting")?.close();
      await queryClient.invalidateQueries();
    },
  });
  const mintToken = useMutation({
    gcTime: 0,
    mutationFn: (body: TokenCreation) =>
      write<{ secret: string; token: AccessToken }>("/api/tokens", "POST", body),
    onSuccess: (created) => {
      document.querySelector<HTMLDialogElement>("#new-token")?.close();
      setOneTime({ title: created.token.name, secret: created.secret });
      document.querySelector<HTMLDialogElement>("#one-time-secret")?.showModal();
      void queryClient.invalidateQueries();
    },
  });
  const mintRunnerToken = useMutation({
    gcTime: 0,
    mutationFn: (name: string) =>
      write<{ secret: string; url: string }>("/api/runners/tokens", "POST", { name }),
    onSuccess: (created) => {
      document.querySelector<HTMLDialogElement>("#new-runner-token")?.close();
      setOneTime({ title: "Runner registration token", secret: created.secret, url: created.url });
      document.querySelector<HTMLDialogElement>("#one-time-secret")?.showModal();
      void queryClient.invalidateQueries();
    },
  });
  useEffect(() => {
    document.title = `${area[0]?.toUpperCase() || ""}${area.slice(1)} · ${name} · Lyra`;
  }, [area, name]);
  function prepareValue(event: MouseEvent<HTMLButtonElement>) {
    setEditorVersion(editorVersion + 1);
    const selected = event.currentTarget.dataset.name || "";
    setEditingName(selected);
    setEditingValue(variables.data?.find((variable) => variable.name === selected)?.value || "");
    changing.reset();
  }
  function valueChanged(event: ChangeEvent<HTMLTextAreaElement>) {
    setEditingValue(event.target.value);
  }
  function valueSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    changing.mutate({
      path: `${base}/${area}/${encodeURIComponent(formValue(form, "name"))}`,
      method: "PUT",
      body: { value: formValue(form, "value") },
      dialog: "value-editor",
    });
  }
  function prepareRule(event: MouseEvent<HTMLButtonElement>) {
    setEditorVersion(editorVersion + 1);
    setEditingRule(
      rules.data?.find((rule) => rule.id === Number(event.currentTarget.dataset.id)) || null,
    );
    changing.reset();
    setValidationError(null);
  }
  function ruleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const methods = form
      .getAll("method")
      .filter((value): value is string => typeof value === "string");
    if (!methods.length) {
      setValidationError(new Error("Choose at least one merge method."));
      return;
    }
    setValidationError(null);
    changing.mutate({
      path: `${base}/rules`,
      method: "POST",
      dialog: "rule-editor",
      body: {
        pattern: formValue(form, "pattern"),
        require_signed: form.has("require_signed"),
        require_approval: form.has("require_approval"),
        require_checks: form.has("require_checks"),
        allow_force_push: form.has("allow_force_push"),
        allow_deletion: form.has("allow_deletion"),
        merge_methods: methods,
      },
    });
  }
  function labelSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    changing.mutate({
      path: `${base}/labels`,
      method: "POST",
      dialog: "label-editor",
      body: {
        name: formValue(form, "name"),
        color: formValue(form, "color"),
        description: formValue(form, "description"),
      },
    });
  }
  function keySubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    changing.mutate({
      path: "/api/keys",
      method: "POST",
      dialog: "key-editor",
      body: {
        title: formValue(form, "title"),
        agent: formValue(form, "agent"),
        public_key: formValue(form, "public_key"),
        signing: true,
      },
    });
  }
  function tokenSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    mintToken.mutate({
      name: formValue(form, "name"),
      scope: formValue(form, "scope"),
      expires_in: formValue(form, "expiry"),
      repos: formValue(form, "repos")
        .split(/[\n,]/)
        .map((repo) => repo.trim())
        .filter(Boolean),
    });
  }
  function runnerTokenSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    mintRunnerToken.mutate(formValue(form, "name"));
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
      path = "",
      title = "Delete item?",
      description = "This item will be removed.",
    } = event.currentTarget.dataset;
    setDeletion({ path, title, description });
    removing.reset();
  }
  function deleteSetting() {
    removing.mutate();
  }
  function clearValue(event: ToggleEvent<HTMLDialogElement>) {
    if (event.newState === "closed") {
      event.currentTarget.querySelector("form")?.reset();
      changing.reset();
      setEditingValue("");
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
    void queryClient.invalidateQueries();
  }
  const areas = [
    { id: "secrets", name: "Secrets", icon: LockKeyhole, visible: admin },
    { id: "variables", name: "Variables", icon: VariableIcon, visible: true },
    { id: "rules", name: "Branch rules", icon: GitBranch, visible: true },
    { id: "labels", name: "Labels", icon: Palette, visible: true },
    { id: "access", name: "Access", icon: KeyRound, visible: instanceAdmin },
    { id: "runners", name: "Runners", icon: Server, visible: instanceAdmin },
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
  let content;
  if (!selectedArea) {
    content = (
      <div className={common.panel}>
        <p className={common.empty}>You don’t have access to this settings section.</p>
      </div>
    );
  } else if (loadError) {
    content = <ErrorNotice error={loadError} retry={retry} />;
  } else if (area === "variables" || area === "secrets") {
    const isSecret = area === "secrets";
    const records = isSecret ? secrets.data : variables.data;
    content = (
      <>
        <div className={common.heading}>
          <div>
            <h1 className={common.title}>{isSecret ? "Secrets" : "Variables"}</h1>
            <p className={common.meta}>
              {isSecret
                ? "Values are encrypted and cannot be read back."
                : "Values are visible to repository readers and available to workflows."}
            </p>
          </div>
          {canWrite && (
            <button
              className={common.primary}
              type="button"
              commandfor="value-editor"
              command="show-modal"
              onClick={prepareValue}
            >
              <Plus size={15} />
              {isSecret ? "New secret" : "New variable"}
            </button>
          )}
        </div>
        <div className={common.panel}>
          {records?.map((record) => (
            <article className={common.row} key={record.name}>
              {isSecret ? <LockKeyhole size={17} /> : <VariableIcon size={17} />}
              <div className={common.grow}>
                <strong>{record.name}</strong>
                <p className={common.meta}>
                  Updated by {record.user} · <DateTime value={record.updated_at} />
                </p>
                {!isSecret && "value" in record && typeof record.value === "string" && (
                  <pre className={styles.variableValue}>{record.value}</pre>
                )}
              </div>
              {canWrite && (
                <>
                  <button
                    className={controls.button}
                    type="button"
                    data-name={record.name}
                    onClick={prepareValue}
                    commandfor="value-editor"
                    command="show-modal"
                  >
                    {isSecret ? "Replace" : "Edit"}
                  </button>
                  <button
                    className={controls.button}
                    type="button"
                    data-path={`${base}/${area}/${encodeURIComponent(record.name)}`}
                    data-title={`Delete ${record.name}?`}
                    data-description={`${record.name} will no longer be available to workflows.`}
                    onClick={prepareDeletion}
                    commandfor="delete-setting"
                    command="show-modal"
                    aria-label={`Delete ${record.name}`}
                  >
                    <Trash2 size={14} />
                  </button>
                </>
              )}
            </article>
          ))}
          {!records && <output className={common.empty}>Loading {area}…</output>}
          {records?.length === 0 && <p className={common.empty}>No {area} configured.</p>}
        </div>
        {canWrite && (
          <dialog
            id="value-editor"
            className={common.dialog}
            closedby="any"
            aria-labelledby="value-title"
            onToggle={clearValue}
          >
            <form key={`${area}/${editingName}/${editorVersion}`} onSubmit={valueSubmit}>
              <h2 id="value-title">
                {editingName ? "Update" : "New"} {isSecret ? "secret" : "variable"}
              </h2>
              <fieldset disabled={changing.isPending}>
                <label className={common.field}>
                  <span>Name</span>
                  <input
                    name="name"
                    defaultValue={editingName}
                    readOnly={!!editingName}
                    required
                    pattern="[A-Za-z_][A-Za-z0-9_]*"
                    maxLength={100}
                    autoComplete="off"
                    placeholder="DEPLOY_ENVIRONMENT"
                  />
                </label>
                {isSecret ? (
                  <label className={common.field}>
                    <span>New value</span>
                    <input type="password" name="value" required autoComplete="new-password" />
                  </label>
                ) : (
                  <label className={common.field}>
                    <span>Value</span>
                    <textarea name="value" value={editingValue} onChange={valueChanged} />
                  </label>
                )}
                <FormError error={changing.error} />
                <div className={common.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="value-editor"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={common.primary} type="submit">
                    {changing.isPending ? "Saving…" : "Save"}
                  </button>
                </div>
              </fieldset>
              <DialogClose target="value-editor" />
            </form>
          </dialog>
        )}
      </>
    );
  } else if (area === "rules") {
    content = (
      <>
        <div className={common.heading}>
          <div>
            <h1 className={common.title}>Branch rules</h1>
            <p className={common.meta}>Matching rules combine. All requirements apply.</p>
          </div>
          {admin && (
            <button
              className={common.primary}
              type="button"
              commandfor="rule-editor"
              command="show-modal"
              onClick={prepareRule}
            >
              <Plus size={15} />
              New rule
            </button>
          )}
        </div>
        <div className={common.panel}>
          {rules.data?.map((rule) => (
            <article className={common.row} key={rule.id}>
              <GitBranch size={18} />
              <div className={common.grow}>
                <strong>
                  <code>{rule.pattern}</code>
                </strong>
                <div className={common.labels}>
                  {rule.require_signed && <span className={common.label}>Verified commits</span>}
                  {rule.require_approval && (
                    <span className={common.label}>Current-head approval</span>
                  )}
                  {rule.require_checks && <span className={common.label}>Successful checks</span>}
                  {!rule.allow_force_push && <span className={common.count}>No force pushes</span>}
                  {!rule.allow_deletion && <span className={common.count}>No deletion</span>}
                </div>
                <p className={common.meta}>Merge methods: {rule.merge_methods.join(", ")}</p>
              </div>
              {admin && (
                <>
                  <button
                    className={controls.button}
                    type="button"
                    data-id={rule.id}
                    onClick={prepareRule}
                    commandfor="rule-editor"
                    command="show-modal"
                  >
                    Edit
                  </button>
                  <button
                    className={controls.button}
                    type="button"
                    data-path={`${base}/rules?pattern=${encodeURIComponent(rule.pattern)}`}
                    data-title={`Delete rule ${rule.pattern}?`}
                    data-description="This rule will stop protecting matching branches."
                    onClick={prepareDeletion}
                    commandfor="delete-setting"
                    command="show-modal"
                    aria-label={`Delete rule ${rule.pattern}`}
                  >
                    <Trash2 size={14} />
                  </button>
                </>
              )}
            </article>
          ))}
          {rules.isPending && <output className={common.empty}>Loading branch rules…</output>}
          {rules.data?.length === 0 && <p className={common.empty}>No branch rules configured.</p>}
        </div>
        {admin && (
          <dialog
            className={common.dialog}
            id="rule-editor"
            closedby="any"
            aria-labelledby="rule-title"
          >
            <form key={`${editingRule?.id || "new"}/${editorVersion}`} onSubmit={ruleSubmit}>
              <h2 id="rule-title">{editingRule ? "Edit branch rule" : "New branch rule"}</h2>
              <fieldset disabled={changing.isPending}>
                <label className={common.field}>
                  <span>Branch name or glob</span>
                  <input
                    name="pattern"
                    defaultValue={editingRule?.pattern || ""}
                    readOnly={!!editingRule}
                    placeholder="main or release/*"
                    required
                  />
                </label>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="require_signed"
                    defaultChecked={editingRule?.require_signed}
                  />
                  Require verified commit signatures
                </label>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="require_approval"
                    defaultChecked={editingRule?.require_approval}
                  />
                  Require approval of the current head
                </label>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="require_checks"
                    defaultChecked={editingRule?.require_checks}
                  />
                  Require successful checks
                </label>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="allow_force_push"
                    defaultChecked={editingRule?.allow_force_push}
                  />
                  Allow force pushes
                </label>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="allow_deletion"
                    defaultChecked={editingRule?.allow_deletion}
                  />
                  Allow branch deletion
                </label>
                <p className={common.meta}>Allowed merge methods</p>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="method"
                    value="squash"
                    defaultChecked={!editingRule || editingRule.merge_methods.includes("squash")}
                  />
                  Squash
                </label>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    name="method"
                    value="ff"
                    defaultChecked={!editingRule || editingRule.merge_methods.includes("ff")}
                  />
                  Fast-forward
                </label>
                <FormError error={validationError || changing.error} />
                <div className={common.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="rule-editor"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={common.primary} type="submit">
                    Save rule
                  </button>
                </div>
              </fieldset>
              <DialogClose target="rule-editor" />
            </form>
          </dialog>
        )}
      </>
    );
  } else if (area === "labels") {
    content = (
      <>
        <div className={common.heading}>
          <h1 className={common.title}>Labels</h1>
          {canWrite && (
            <button
              className={common.primary}
              type="button"
              commandfor="label-editor"
              command="show-modal"
              onClick={resetChange}
            >
              <Plus size={15} />
              New label
            </button>
          )}
        </div>
        <div className={common.panel}>
          {labels.data?.map((label) => (
            <article className={common.row} key={label.name}>
              <span className={styles.labelColor} style={{ backgroundColor: label.color }} />
              <div className={common.grow}>
                <strong>{label.name}</strong>
                <p className={common.meta}>{label.description}</p>
              </div>
              {canWrite && (
                <button
                  className={controls.button}
                  type="button"
                  data-path={`${base}/labels/${encodeURIComponent(label.name)}`}
                  data-title={`Delete label ${label.name}?`}
                  data-description="The label will be removed from this repository and its issues and pull requests."
                  onClick={prepareDeletion}
                  commandfor="delete-setting"
                  command="show-modal"
                  aria-label={`Delete label ${label.name}`}
                >
                  <Trash2 size={14} />
                </button>
              )}
            </article>
          ))}
          {labels.isPending && <output className={common.empty}>Loading labels…</output>}
          {labels.data?.length === 0 && <p className={common.empty}>No labels configured.</p>}
        </div>
        {canWrite && (
          <dialog
            className={common.dialog}
            id="label-editor"
            closedby="any"
            aria-labelledby="label-title"
          >
            <form onSubmit={labelSubmit}>
              <h2 id="label-title">New label</h2>
              <fieldset disabled={changing.isPending}>
                <label className={common.field}>
                  <span>Name</span>
                  <input name="name" required />
                </label>
                <label className={common.field}>
                  <span>Description</span>
                  <input name="description" />
                </label>
                <label className={common.field}>
                  <span>Color</span>
                  <input type="color" name="color" defaultValue="#8b5ec8" />
                </label>
                <FormError error={changing.error} />
                <div className={common.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="label-editor"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={common.primary} type="submit">
                    Create label
                  </button>
                </div>
              </fieldset>
              <DialogClose target="label-editor" />
            </form>
          </dialog>
        )}
      </>
    );
  } else if (area === "access") {
    content = (
      <>
        <div className={common.heading}>
          <div>
            <h1 className={common.title}>Access</h1>
            <p className={common.meta}>Tokens and signing keys apply to your Lyra instance.</p>
          </div>
          <button
            className={common.primary}
            type="button"
            commandfor="new-token"
            command="show-modal"
            onClick={resetToken}
          >
            <Plus size={15} />
            New token
          </button>
        </div>
        <section className={common.panel}>
          <div className={common.panelHeading}>
            <h2>Access tokens</h2>
          </div>
          {tokens.data?.map((token) => (
            <article className={common.row} key={token.id}>
              <KeyRound size={18} />
              <div className={common.grow}>
                <strong>{token.name}</strong>
                <p className={common.meta}>
                  {token.scope} · {token.repos.join(", ") || "All repositories"}
                </p>
                <p className={common.meta}>
                  Last used <DateTime value={token.last_used_at} /> · Expires{" "}
                  <DateTime value={token.expires_at} />
                </p>
              </div>
              <button
                className={controls.button}
                type="button"
                data-path={`/api/tokens/${token.id}`}
                data-title={`Revoke ${token.name}?`}
                data-description="This token will stop authenticating immediately."
                onClick={prepareDeletion}
                commandfor="delete-setting"
                command="show-modal"
              >
                Revoke
              </button>
            </article>
          ))}
          {tokens.isPending && <output className={common.empty}>Loading tokens…</output>}
          {tokens.data?.length === 0 && <p className={common.empty}>No access tokens.</p>}
        </section>
        <section className={common.panel}>
          <div className={common.panelHeading}>
            <h2>Commit signing keys</h2>
            <button
              className={controls.button}
              type="button"
              commandfor="key-editor"
              command="show-modal"
              onClick={resetChange}
            >
              Add signing key
            </button>
          </div>
          {keys.data?.map((key) => (
            <article className={common.row} key={key.id}>
              <KeyRound size={18} />
              <div className={common.grow}>
                <strong>{key.title || "Signing key"}</strong>
                <p className={common.meta}>
                  {key.agent || "Human"} · <code>{key.fingerprint}</code>
                </p>
              </div>
              <button
                className={controls.button}
                type="button"
                data-path={`/api/keys/${key.id}`}
                data-title={`Remove ${key.title || "signing key"}?`}
                data-description="Future commits will no longer be verified against this registered key."
                onClick={prepareDeletion}
                commandfor="delete-setting"
                command="show-modal"
                aria-label={`Remove signing key ${key.title}`}
              >
                <Trash2 size={14} />
              </button>
            </article>
          ))}
          {keys.isPending && <output className={common.empty}>Loading signing keys…</output>}
          {keys.data?.length === 0 && <p className={common.empty}>No registered signing keys.</p>}
        </section>
        {serverKey.data && (
          <section className={common.panel}>
            <div className={common.panelHeading}>
              <h2>Lyra merge signing key</h2>
              <CopyButton
                value={serverKey.data.allowed_signers}
                label="Copy allowed signers entry"
                compact
              />
            </div>
            <p className={styles.fingerprint}>
              <code>{serverKey.data.fingerprint}</code>
            </p>
          </section>
        )}
        <dialog
          id="new-token"
          className={common.dialog}
          closedby="any"
          aria-labelledby="new-token-title"
          onToggle={clearToken}
        >
          <form onSubmit={tokenSubmit}>
            <h2 id="new-token-title">New access token</h2>
            <fieldset disabled={mintToken.isPending}>
              <label className={common.field}>
                <span>Name</span>
                <input name="name" required placeholder="Agent or device name" autoComplete="off" />
              </label>
              <label className={common.field}>
                <span>Scope</span>
                <select name="scope" className={controls.select} defaultValue="write">
                  <option value="read">Read</option>
                  <option value="write">Write</option>
                  <option value="admin">Admin</option>
                </select>
                <small>
                  Read browses and pulls. Write pushes and collaborates. Admin manages
                  configuration.
                </small>
              </label>
              <label className={common.field}>
                <span>Repositories</span>
                <textarea name="repos" defaultValue={name} />
                <small>
                  One owner/repository or owner/* per line. Leave empty for all repositories.
                </small>
              </label>
              <label className={common.field}>
                <span>Expiry</span>
                <select name="expiry" className={controls.select} defaultValue="720h">
                  <option value="24h">24 hours</option>
                  <option value="168h">7 days</option>
                  <option value="720h">30 days</option>
                  <option value="2160h">90 days</option>
                  <option value="">No expiry</option>
                </select>
              </label>
              <FormError error={mintToken.error} />
              <div className={common.dialogActions}>
                <button
                  className={controls.button}
                  type="button"
                  commandfor="new-token"
                  command="close"
                >
                  Cancel
                </button>
                <button className={common.primary} type="submit">
                  Create token
                </button>
              </div>
            </fieldset>
            <DialogClose target="new-token" />
          </form>
        </dialog>
        <dialog
          id="key-editor"
          className={common.dialog}
          closedby="any"
          aria-labelledby="key-title"
        >
          <form onSubmit={keySubmit}>
            <h2 id="key-title">Add signing key</h2>
            <fieldset disabled={changing.isPending}>
              <label className={common.field}>
                <span>Title</span>
                <input name="title" required />
              </label>
              <label className={common.field}>
                <span>Agent</span>
                <input name="agent" placeholder="Leave empty for your own key" />
              </label>
              <label className={common.field}>
                <span>OpenSSH public key</span>
                <textarea name="public_key" required placeholder="ssh-ed25519 …" />
              </label>
              <FormError error={changing.error} />
              <div className={common.dialogActions}>
                <button
                  className={controls.button}
                  type="button"
                  commandfor="key-editor"
                  command="close"
                >
                  Cancel
                </button>
                <button className={common.primary} type="submit">
                  Add key
                </button>
              </div>
            </fieldset>
            <DialogClose target="key-editor" />
          </form>
        </dialog>
      </>
    );
  } else if (area === "runners") {
    content = (
      <>
        <div className={common.heading}>
          <div>
            <h1 className={common.title}>Runners</h1>
            <p className={common.meta}>Runners are shared across repositories in this instance.</p>
          </div>
          <button
            className={common.primary}
            type="button"
            commandfor="new-runner-token"
            command="show-modal"
            onClick={resetRunnerToken}
          >
            <Plus size={15} />
            Registration token
          </button>
        </div>
        <section className={common.panel}>
          {runners.data?.map((runner) => (
            <article className={common.row} key={runner.id}>
              <Server size={18} />
              <div className={common.grow}>
                <strong>{runner.name}</strong>
                <p className={common.meta}>
                  {runner.version || "Not connected yet"} · Last seen{" "}
                  <DateTime value={runner.last_seen_at} />
                </p>
                <div className={common.labels}>
                  {runner.labels.map((label) => (
                    <span className={common.label} key={label}>
                      {label}
                    </span>
                  ))}
                </div>
              </div>
              <span className={common.live} data-connected={runner.online}>
                {runner.online ? "Online" : "Offline"}
              </span>
              <button
                className={controls.button}
                type="button"
                data-path={`/api/runners/${runner.id}`}
                data-title={`Remove runner ${runner.name}?`}
                data-description="The runner registration will be removed."
                onClick={prepareDeletion}
                commandfor="delete-setting"
                command="show-modal"
                aria-label={`Remove runner ${runner.name}`}
              >
                <Trash2 size={14} />
              </button>
            </article>
          ))}
          {runners.isPending && <output className={common.empty}>Loading runners…</output>}
          {runners.data?.length === 0 && <p className={common.empty}>No runners registered.</p>}
        </section>
        <section className={common.panel}>
          <div className={common.panelHeading}>
            <h2>Registration tokens</h2>
          </div>
          {runnerTokens.data?.map((token) => (
            <article className={common.row} key={token.id}>
              <div className={common.grow}>
                <strong>{token.name}</strong>
                <p className={common.meta}>
                  Created <DateTime value={token.created_at} />
                </p>
              </div>
              <button
                className={controls.button}
                type="button"
                data-path={`/api/runners/tokens/${token.id}`}
                data-title={`Revoke registration token ${token.name}?`}
                data-description="This token will stop registering new runners. Existing runners are kept."
                onClick={prepareDeletion}
                commandfor="delete-setting"
                command="show-modal"
              >
                Revoke
              </button>
            </article>
          ))}
          {runnerTokens.data?.length === 0 && (
            <p className={common.empty}>No registration tokens.</p>
          )}
        </section>
        <dialog
          id="new-runner-token"
          className={common.dialog}
          closedby="any"
          aria-labelledby="runner-token-title"
          onToggle={clearToken}
        >
          <form onSubmit={runnerTokenSubmit}>
            <h2 id="runner-token-title">New registration token</h2>
            <fieldset disabled={mintRunnerToken.isPending}>
              <label className={common.field}>
                <span>Name</span>
                <input name="name" required maxLength={100} />
              </label>
              <FormError error={mintRunnerToken.error} />
              <div className={common.dialogActions}>
                <button
                  className={controls.button}
                  type="button"
                  commandfor="new-runner-token"
                  command="close"
                >
                  Cancel
                </button>
                <button className={common.primary} type="submit">
                  Create token
                </button>
              </div>
            </fieldset>
            <DialogClose target="new-runner-token" />
          </form>
        </dialog>
      </>
    );
  }
  return (
    <div className={styles.workspace}>
      <nav className={styles.navigation} aria-label="Settings sections">
        {areas
          .filter((entry) => entry.visible)
          .map((entry) => {
            const Icon = entry.icon;
            return (
              <Link
                key={entry.id}
                to="/$owner/$repo/$section"
                params={{ owner, repo, section: "settings" }}
                search={{ section: entry.id }}
                aria-current={area === entry.id ? "page" : undefined}
              >
                <Icon size={16} />
                {entry.name}
              </Link>
            );
          })}
      </nav>
      <div className={common.grow}>{content}</div>
      <ConfirmDialog
        id="delete-setting"
        title={deletion?.title || "Delete item?"}
        confirm="Delete"
        onConfirm={deleteSetting}
        pending={removing.isPending}
        error={removing.error}
        danger
      >
        <p>{deletion?.description}</p>
      </ConfirmDialog>
      <dialog
        className={common.dialog}
        id="one-time-secret"
        closedby="any"
        aria-labelledby="one-time-title"
        onToggle={clearOneTime}
      >
        <h2 id="one-time-title">{oneTime?.title || "Token created"}</h2>
        <p>Copy this token now. It cannot be read again after closing.</p>
        <label className={common.field}>
          <span>Token</span>
          <input type="password" readOnly value={oneTime?.secret || ""} autoComplete="off" />
        </label>
        {oneTime && <CopyButton value={oneTime.secret} label="Copy token" />}
        {oneTime?.url && (
          <label className={common.field}>
            <span>Instance URL</span>
            <input readOnly value={oneTime.url} />
          </label>
        )}
        <div className={common.dialogActions}>
          <button
            type="button"
            className={common.primary}
            commandfor="one-time-secret"
            command="close"
          >
            Done
          </button>
        </div>
        <DialogClose target="one-time-secret" />
      </dialog>
    </div>
  );
}
