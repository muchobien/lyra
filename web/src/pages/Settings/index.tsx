import { Field } from "@/components/forms/Field";
import { FormDialog } from "@/components/forms/FormDialog";
import { Link } from "@tanstack/react-router";
import {
  GitBranchIcon,
  KeyIcon,
  LockIcon,
  PlusIcon,
  ServerIcon,
  SlidersIcon,
  TrashIcon,
} from "@primer/octicons-react";
import { CopyButton } from "@/components/controls/CopyButton";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ScrollNavigation } from "@/components/controls/ScrollNavigation";
import { DateTime } from "@/components/metadata/DateTime";
import { ConfirmDialog } from "@/components/forms/ConfirmDialog";
import { DialogClose } from "@/components/forms/DialogClose";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "@/components/forms/FormDialog/styles.css";
import "./styles.css";
import { useConnect, type SettingsProps } from "./connect";

export function Settings({ scope }: SettingsProps) {
  const {
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
  } = useConnect({ scope });
  let content;
  if (!selectedArea) {
    content = (
      <div className="lyra-workspace-panel">
        <p className="lyra-workspace-empty">You don’t have access to this settings section.</p>
      </div>
    );
  } else if (loadError) {
    content = <ErrorNotice error={loadError} retry={retry} />;
  } else if (area === "variables" || area === "secrets") {
    const isSecret = area === "secrets";
    const records = isSecret ? secrets.data : variables.data;
    content = (
      <>
        <div className="lyra-workspace-heading">
          <div>
            <h1 className="lyra-workspace-title">{isSecret ? "Secrets" : "Variables"}</h1>
            <p className="lyra-workspace-meta">
              {isSecret
                ? "Values are encrypted and cannot be read back."
                : "Values are visible to repository readers and available to workflows."}
            </p>
          </div>
          {canWrite && (
            <button
              className="lyra-workspace-primary"
              type="button"
              commandfor="value-editor"
              command="show-modal"
              onClick={prepareValue}
            >
              <PlusIcon size={16} />
              {isSecret ? "New secret" : "New variable"}
            </button>
          )}
        </div>
        <div className="lyra-workspace-panel">
          {records?.map((record) => (
            <article className="lyra-workspace-row" key={record.name}>
              {isSecret ? <LockIcon size={16} /> : <SlidersIcon size={16} />}
              <div className="lyra-workspace-grow">
                <strong>{record.name}</strong>
                <p className="lyra-workspace-meta">
                  Updated by {record.user} · <DateTime value={record.updated_at} />
                </p>
                {!isSecret && "value" in record && typeof record.value === "string" && (
                  <pre className="lyra-settings-variableValue">{record.value}</pre>
                )}
              </div>
              {canWrite && (
                <>
                  <button
                    className="lyra-controls-button"
                    type="button"
                    data-name={record.name}
                    onClick={prepareValue}
                    commandfor="value-editor"
                    command="show-modal"
                  >
                    {isSecret ? "Replace" : "Edit"}
                  </button>
                  <button
                    className="lyra-controls-button"
                    type="button"
                    data-kind={area}
                    data-name={record.name}
                    data-title={`Delete ${record.name}?`}
                    data-description={`${record.name} will no longer be available to workflows.`}
                    onClick={prepareDeletion}
                    commandfor="delete-setting"
                    command="show-modal"
                    aria-label={`Delete ${record.name}`}
                  >
                    <TrashIcon size={16} />
                  </button>
                </>
              )}
            </article>
          ))}
          {!records && <output className="lyra-workspace-empty">Loading {area}…</output>}
          {records?.length === 0 && <p className="lyra-workspace-empty">No {area} configured.</p>}
        </div>
        {canWrite && (
          <FormDialog
            id="value-editor"
            title={
              <>
                {editor.name ? "Update" : "New"} {isSecret ? "secret" : "variable"}
              </>
            }
            onSubmit={valueSubmit}
            pending={changing.isPending}
            error={changing.error}
            submitLabel={changing.isPending ? "Saving…" : "Save"}
            formKey={`${area}/${editor.name}/${editor.version}`}
            onToggle={clearValue}
          >
            <Field label="Name">
              <input
                name="name"
                defaultValue={editor.name}
                readOnly={!!editor.name}
                required
                pattern="[A-Za-z_][A-Za-z0-9_]*"
                maxLength={100}
                autoComplete="off"
                placeholder="DEPLOY_ENVIRONMENT"
              />
            </Field>
            {isSecret ? (
              <Field label="New value">
                <input type="password" name="value" required autoComplete="new-password" />
              </Field>
            ) : (
              <Field label="Value">
                <textarea name="value" defaultValue={editor.value} />
              </Field>
            )}
          </FormDialog>
        )}
      </>
    );
  } else if (area === "rules") {
    content = (
      <>
        <div className="lyra-workspace-heading">
          <div>
            <h1 className="lyra-workspace-title">Branch rules</h1>
            <p className="lyra-workspace-meta">Matching rules combine. All requirements apply.</p>
          </div>
          {admin && (
            <button
              className="lyra-workspace-primary"
              type="button"
              commandfor="rule-editor"
              command="show-modal"
              onClick={prepareRule}
            >
              <PlusIcon size={16} />
              New rule
            </button>
          )}
        </div>
        <div className="lyra-workspace-panel">
          {rules.data?.map((rule) => (
            <article className="lyra-workspace-row" key={rule.id}>
              <GitBranchIcon size={16} />
              <div className="lyra-workspace-grow">
                <strong>
                  <code>{rule.pattern}</code>
                </strong>
                <div className="lyra-workspace-labels">
                  {rule.require_signed && (
                    <span className="lyra-workspace-label">Verified commits</span>
                  )}
                  {rule.require_approval && (
                    <span className="lyra-workspace-label">Current-head approval</span>
                  )}
                  {rule.require_checks && (
                    <span className="lyra-workspace-label">Successful checks</span>
                  )}
                  {!rule.allow_force_push && (
                    <span className="lyra-workspace-count">No force pushes</span>
                  )}
                  {!rule.allow_deletion && (
                    <span className="lyra-workspace-count">No deletion</span>
                  )}
                </div>
                <p className="lyra-workspace-meta">
                  Merge methods: {rule.merge_methods.join(", ")}
                </p>
              </div>
              {admin && (
                <>
                  <button
                    className="lyra-controls-button"
                    type="button"
                    data-id={rule.id}
                    onClick={prepareRule}
                    commandfor="rule-editor"
                    command="show-modal"
                  >
                    Edit
                  </button>
                  <button
                    className="lyra-controls-button"
                    type="button"
                    data-kind="rules"
                    data-name={rule.pattern}
                    data-title={`Delete rule ${rule.pattern}?`}
                    data-description="This rule will stop protecting matching branches."
                    onClick={prepareDeletion}
                    commandfor="delete-setting"
                    command="show-modal"
                    aria-label={`Delete rule ${rule.pattern}`}
                  >
                    <TrashIcon size={16} />
                  </button>
                </>
              )}
            </article>
          ))}
          {rules.isPending && (
            <output className="lyra-workspace-empty">Loading branch rules…</output>
          )}
          {rules.data?.length === 0 && (
            <p className="lyra-workspace-empty">No branch rules configured.</p>
          )}
        </div>
        {admin && (
          <FormDialog
            id="rule-editor"
            title={editor.rule ? "Edit branch rule" : "New branch rule"}
            onSubmit={ruleSubmit}
            pending={changing.isPending}
            error={validationError || changing.error}
            submitLabel="Save rule"
            formKey={`${editor.rule?.id || "new"}/${editor.version}`}
          >
            <Field label="Branch name or glob">
              <input
                name="pattern"
                defaultValue={editor.rule?.pattern || ""}
                readOnly={!!editor.rule}
                placeholder="main or release/*"
                required
              />
            </Field>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="require_signed"
                defaultChecked={editor.rule?.require_signed}
              />
              Require verified commit signatures
            </label>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="require_approval"
                defaultChecked={editor.rule?.require_approval}
              />
              Require approval of the current head
            </label>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="require_checks"
                defaultChecked={editor.rule?.require_checks}
              />
              Require successful checks
            </label>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="allow_force_push"
                defaultChecked={editor.rule?.allow_force_push}
              />
              Allow force pushes
            </label>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="allow_deletion"
                defaultChecked={editor.rule?.allow_deletion}
              />
              Allow branch deletion
            </label>
            <p className="lyra-workspace-meta">Allowed merge methods</p>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="method"
                value="squash"
                defaultChecked={!editor.rule || editor.rule.merge_methods.includes("squash")}
              />
              Squash
            </label>
            <label className="lyra-workspace-checkbox">
              <input
                type="checkbox"
                name="method"
                value="ff"
                defaultChecked={!editor.rule || editor.rule.merge_methods.includes("ff")}
              />
              Fast-forward
            </label>
          </FormDialog>
        )}
      </>
    );
  } else if (area === "labels") {
    content = (
      <>
        <div className="lyra-workspace-heading">
          <h1 className="lyra-workspace-title">Labels</h1>
          {canWrite && (
            <button
              className="lyra-workspace-primary"
              type="button"
              commandfor="label-editor"
              command="show-modal"
              onClick={resetChange}
            >
              <PlusIcon size={16} />
              New label
            </button>
          )}
        </div>
        <div className="lyra-workspace-panel">
          {labels.data?.map((label) => (
            <article className="lyra-workspace-row" key={label.name}>
              <span
                className="lyra-settings-labelColor"
                style={{ backgroundColor: `#${label.color}` }}
              />
              <div className="lyra-workspace-grow">
                <strong>{label.name}</strong>
                <p className="lyra-workspace-meta">{label.description}</p>
              </div>
              {canWrite && (
                <button
                  className="lyra-controls-button"
                  type="button"
                  data-kind="labels"
                  data-name={label.name}
                  data-title={`Delete label ${label.name}?`}
                  data-description="The label will be removed from this repository and its issues and pull requests."
                  onClick={prepareDeletion}
                  commandfor="delete-setting"
                  command="show-modal"
                  aria-label={`Delete label ${label.name}`}
                >
                  <TrashIcon size={16} />
                </button>
              )}
            </article>
          ))}
          {labels.isPending && <output className="lyra-workspace-empty">Loading labels…</output>}
          {labels.data?.length === 0 && (
            <p className="lyra-workspace-empty">No labels configured.</p>
          )}
        </div>
        {canWrite && (
          <FormDialog
            id="label-editor"
            title="New label"
            onSubmit={labelSubmit}
            pending={changing.isPending}
            error={changing.error}
            submitLabel="Create label"
          >
            <Field label="Name">
              <input name="name" required />
            </Field>
            <Field label="Description">
              <input name="description" />
            </Field>
            <Field label="Color">
              <input type="color" name="color" defaultValue="#8b5ec8" />
            </Field>
          </FormDialog>
        )}
      </>
    );
  } else if (area === "access") {
    content = (
      <>
        <div className="lyra-workspace-heading">
          <div>
            <h1 className="lyra-workspace-title">Access</h1>
            <p className="lyra-workspace-meta">
              Tokens and signing keys apply to your Lyra instance.
            </p>
          </div>
          <button
            className="lyra-workspace-primary"
            type="button"
            commandfor="new-token"
            command="show-modal"
            onClick={resetToken}
          >
            <PlusIcon size={16} />
            New token
          </button>
        </div>
        <section className="lyra-workspace-panel">
          <div className="lyra-workspace-panelHeading">
            <h2>Access tokens</h2>
          </div>
          {tokens.data?.map((token) => (
            <article className="lyra-workspace-row" key={token.id}>
              <KeyIcon size={16} />
              <div className="lyra-workspace-grow">
                <strong>{token.name}</strong>
                <p className="lyra-workspace-meta">
                  {token.scope} · {token.repos.join(", ") || "All repositories"}
                </p>
                <p className="lyra-workspace-meta">
                  Last used <DateTime value={token.last_used_at} /> · Expires{" "}
                  <DateTime value={token.expires_at} />
                </p>
              </div>
              <button
                className="lyra-controls-button"
                type="button"
                data-kind="tokens"
                data-id={token.id}
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
          {tokens.isPending && <output className="lyra-workspace-empty">Loading tokens…</output>}
          {tokens.data?.length === 0 && <p className="lyra-workspace-empty">No access tokens.</p>}
        </section>
        <section className="lyra-workspace-panel">
          <div className="lyra-workspace-panelHeading">
            <h2>Commit signing keys</h2>
            <button
              className="lyra-controls-button"
              type="button"
              commandfor="key-editor"
              command="show-modal"
              onClick={resetChange}
            >
              Add signing key
            </button>
          </div>
          {keys.data?.map((key) => (
            <article className="lyra-workspace-row" key={key.id}>
              <KeyIcon size={16} />
              <div className="lyra-workspace-grow">
                <strong>{key.title || "Signing key"}</strong>
                <p className="lyra-workspace-meta">
                  {key.agent || "Human"} · <code>{key.fingerprint}</code>
                </p>
              </div>
              <button
                className="lyra-controls-button"
                type="button"
                data-kind="keys"
                data-id={key.id}
                data-title={`Remove ${key.title || "signing key"}?`}
                data-description="Future commits will no longer be verified against this registered key."
                onClick={prepareDeletion}
                commandfor="delete-setting"
                command="show-modal"
                aria-label={`Remove signing key ${key.title}`}
              >
                <TrashIcon size={16} />
              </button>
            </article>
          ))}
          {keys.isPending && (
            <output className="lyra-workspace-empty">Loading signing keys…</output>
          )}
          {keys.data?.length === 0 && (
            <p className="lyra-workspace-empty">No registered signing keys.</p>
          )}
        </section>
        {serverKey.data && (
          <section className="lyra-workspace-panel">
            <div className="lyra-workspace-panelHeading">
              <h2>Lyra merge signing key</h2>
              <CopyButton
                value={serverKey.data.allowed_signers}
                label="Copy allowed signers entry"
                compact
              />
            </div>
            <p className="lyra-settings-fingerprint">
              <code>{serverKey.data.fingerprint}</code>
            </p>
          </section>
        )}
        <FormDialog
          id="new-token"
          title="New access token"
          onSubmit={tokenSubmit}
          pending={mintToken.isPending}
          error={mintToken.error}
          submitLabel="Create token"
          onToggle={clearToken}
        >
          <Field label="Name">
            <input name="name" required placeholder="Agent or device name" autoComplete="off" />
          </Field>
          <Field label="Scope">
            <select name="scope" className="lyra-controls-select" defaultValue="write">
              <option value="read">Read</option>
              <option value="write">Write</option>
              <option value="admin">Admin</option>
            </select>
            <small>
              Read browses and pulls. Write pushes and collaborates. Admin manages configuration.
            </small>
          </Field>
          <Field label="Repositories">
            <textarea name="repos" defaultValue={name} />
            <small>
              One owner/repository or owner/* per line. Leave empty for all repositories.
            </small>
          </Field>
          <Field label="Expiry">
            <select name="expiry" className="lyra-controls-select" defaultValue="720h">
              <option value="24h">24 hours</option>
              <option value="168h">7 days</option>
              <option value="720h">30 days</option>
              <option value="2160h">90 days</option>
              <option value="">No expiry</option>
            </select>
          </Field>
        </FormDialog>
        <FormDialog
          id="key-editor"
          title="Add signing key"
          onSubmit={keySubmit}
          pending={changing.isPending}
          error={changing.error}
          submitLabel="Add key"
        >
          <Field label="Title">
            <input name="title" required />
          </Field>
          <Field label="Agent">
            <input name="agent" placeholder="Leave empty for your own key" />
          </Field>
          <Field label="OpenSSH public key">
            <textarea name="public_key" required placeholder="ssh-ed25519 …" />
          </Field>
        </FormDialog>
      </>
    );
  } else if (area === "runners") {
    content = (
      <>
        <div className="lyra-workspace-heading">
          <div>
            <h1 className="lyra-workspace-title">Runners</h1>
            <p className="lyra-workspace-meta">
              Runners are shared across repositories in this instance.
            </p>
          </div>
          <button
            className="lyra-workspace-primary"
            type="button"
            commandfor="new-runner-token"
            command="show-modal"
            onClick={resetRunnerToken}
          >
            <PlusIcon size={16} />
            Registration token
          </button>
        </div>
        <section className="lyra-workspace-panel">
          {runners.data?.map((runner) => (
            <article className="lyra-workspace-row" key={runner.id}>
              <ServerIcon size={16} />
              <div className="lyra-workspace-grow">
                <strong>{runner.name}</strong>
                <p className="lyra-workspace-meta">
                  {runner.version || "Not connected yet"} · Last seen{" "}
                  <DateTime value={runner.last_seen_at} />
                </p>
                <div className="lyra-workspace-labels">
                  {runner.labels.map((label) => (
                    <span className="lyra-workspace-label" key={label}>
                      {label}
                    </span>
                  ))}
                </div>
              </div>
              <span className="lyra-workspace-live" data-connected={runner.online}>
                {runner.online ? "Online" : "Offline"}
              </span>
              <button
                className="lyra-controls-button"
                type="button"
                data-kind="runners"
                data-id={runner.id}
                data-title={`Remove runner ${runner.name}?`}
                data-description="The runner registration will be removed."
                onClick={prepareDeletion}
                commandfor="delete-setting"
                command="show-modal"
                aria-label={`Remove runner ${runner.name}`}
              >
                <TrashIcon size={16} />
              </button>
            </article>
          ))}
          {runners.isPending && <output className="lyra-workspace-empty">Loading runners…</output>}
          {runners.data?.length === 0 && (
            <p className="lyra-workspace-empty">No runners registered.</p>
          )}
        </section>
        <section className="lyra-workspace-panel">
          <div className="lyra-workspace-panelHeading">
            <h2>Registration tokens</h2>
          </div>
          {runnerTokens.data?.map((token) => (
            <article className="lyra-workspace-row" key={token.id}>
              <div className="lyra-workspace-grow">
                <strong>{token.name}</strong>
                <p className="lyra-workspace-meta">
                  Created <DateTime value={token.created_at} />
                </p>
              </div>
              <button
                className="lyra-controls-button"
                type="button"
                data-kind="runner-tokens"
                data-id={token.id}
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
            <p className="lyra-workspace-empty">No registration tokens.</p>
          )}
        </section>
        <FormDialog
          id="new-runner-token"
          title="New registration token"
          onSubmit={runnerTokenSubmit}
          pending={mintRunnerToken.isPending}
          error={mintRunnerToken.error}
          submitLabel="Create token"
          onToggle={clearToken}
        >
          <Field label="Name">
            <input name="name" required maxLength={100} />
          </Field>
        </FormDialog>
      </>
    );
  }

  return (
    <div className="lyra-settings-workspace">
      <title>
        {[selectedArea?.name || "Settings", instance ? "Instance" : name, "Lyra"].join(" · ")}
      </title>

      <ScrollNavigation
        className="lyra-settings-navigation"
        label="Settings sections"
        active={area}
      >
        {areas
          .filter((entry) => entry.visible)
          .map((entry) => {
            const Icon = entry.icon;
            const current = area === entry.id ? "page" : undefined;
            return instance ? (
              <Link
                key={entry.id}
                to="/settings"
                search={{ section: entry.id }}
                aria-current={current}
              >
                <Icon size={16} />
                {entry.name}
              </Link>
            ) : (
              <Link
                key={entry.id}
                to="/$owner/$repo/settings"
                params={{ owner, repo }}
                search={{ section: entry.id }}
                aria-current={current}
              >
                <Icon size={16} />
                {entry.name}
              </Link>
            );
          })}
      </ScrollNavigation>

      <div className="lyra-workspace-grow">{content}</div>

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
        className="lyra-workspace-dialog"
        id="one-time-secret"
        closedby="any"
        aria-labelledby="one-time-title"
        onToggle={clearOneTime}
      >
        <h2 id="one-time-title">{oneTime?.title || "Token created"}</h2>
        <p>Copy this token now. It cannot be read again after closing.</p>
        <Field label="Token">
          <input type="password" readOnly value={oneTime?.secret || ""} autoComplete="off" />
        </Field>
        {oneTime && <CopyButton value={oneTime.secret} label="Copy token" />}
        {oneTime?.url && (
          <Field label="Instance URL">
            <input readOnly value={oneTime.url} />
          </Field>
        )}
        <div className="lyra-workspace-dialogActions">
          <button
            type="button"
            className="lyra-workspace-primary"
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
