import { Field } from "@/components/forms/Field";
import { FormDialog } from "@/components/forms/FormDialog";
import { Link } from "@tanstack/react-router";
import { CalendarIcon, GitBranchIcon, PlayIcon, StopwatchIcon } from "@primer/octicons-react";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ActorName } from "@/components/metadata/ActorName";
import { DateTime } from "@/components/metadata/DateTime";
import { duration } from "@/utils/duration";
import { RunState } from "@/components/status/RunState";
import { Status } from "@/components/status/Status";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "@/components/forms/FormError/styles.css";
import "@/components/status/Status/styles.css";
import "./styles.css";
import { useConnect } from "./connect";

export function ActionsList() {
  const {
    owner,
    repo,
    search,
    identity,
    limit,
    metadata,
    reference,
    workflows,
    tags,
    runs,
    selected,
    visible,
    dispatch,
    retry,
    filtersChanged,
    dispatchRun,
    resetDispatch,
    triggerLabels,
  } = useConnect();

  return (
    <div className="lyra-actions-listWorkspace">
      <aside className="lyra-actions-workflows">
        <Link
          to="/$owner/$repo/runs"
          params={{ owner, repo }}
          search={{ ref: search.ref }}
          aria-current={!search.workflow ? "page" : undefined}
        >
          <PlayIcon size={16} />
          All workflows
        </Link>
        {workflows.data?.map((workflow) => (
          <Link
            key={workflow.path}
            to="/$owner/$repo/runs"
            params={{ owner, repo }}
            search={{ ref: search.ref, workflow: workflow.path }}
            aria-current={workflow.path === search.workflow ? "page" : undefined}
          >
            <span>{workflow.name}</span>
            {workflow.dispatch && <span className="lyra-workspace-count">Manual</span>}
            {workflow.error && <Status value="failure">Invalid</Status>}
          </Link>
        ))}
        {workflows.data?.length === 0 && (
          <p className="lyra-workspace-meta">No workflow files at this ref.</p>
        )}
      </aside>

      <div className="lyra-workspace-grow">
        <div className="lyra-workspace-toolbar">
          <input
            type="search"
            className="lyra-workspace-search"
            name="q"
            aria-label="Search recent runs"
            placeholder="Search recent runs…"
            value={search.q || ""}
            onChange={filtersChanged}
          />
          <select
            className="lyra-controls-select"
            name="state"
            aria-label="Run status"
            value={search.state || ""}
            onChange={filtersChanged}
          >
            <option value="">All statuses</option>
            {["waiting", "blocked", "running", "success", "failure", "cancelled", "skipped"].map(
              (status) => (
                <option key={status}>{status}</option>
              ),
            )}
          </select>
          <select
            className="lyra-controls-select"
            name="ref"
            aria-label="Workflow ref"
            value={reference}
            onChange={filtersChanged}
          >
            {!metadata.data?.branches.some((branch) => branch.name === reference) &&
              !tags.data?.some((tag) => `refs/tags/${tag.name}` === reference) && (
                <option>{reference}</option>
              )}
            <optgroup label="Branches">
              {metadata.data?.branches.map((branch) => (
                <option key={branch.name}>{branch.name}</option>
              ))}
            </optgroup>
            <optgroup label="Tags">
              {tags.data?.map((tag) => (
                <option key={tag.name} value={`refs/tags/${tag.name}`}>
                  {tag.name}
                </option>
              ))}
            </optgroup>
          </select>
          {selected?.dispatch && identity?.scope !== "read" && (
            <button
              className="lyra-workspace-primary"
              type="button"
              commandfor="dispatch-workflow"
              command="show-modal"
              onClick={resetDispatch}
            >
              <PlayIcon size={16} />
              Run workflow
            </button>
          )}
        </div>
        {selected && (
          <div className="lyra-workspace-toolbar">
            <strong>{selected.name}</strong>
            <Link
              className="lyra-workspace-link"
              to="/$owner/$repo/blob/$"
              params={{ owner, repo, _splat: `${reference}/${selected.path}` }}
            >
              View workflow file
            </Link>
            {selected.error && <p className="lyra-workspace-formError">{selected.error}</p>}
          </div>
        )}
        {runs.error && <ErrorNotice error={runs.error} retry={retry} />}
        {workflows.error && <ErrorNotice error={workflows.error} retry={retry} />}
        {runs.isPending && <output className="lyra-workspace-empty">Loading runs…</output>}
        {runs.data && (
          <div className="lyra-workspace-panel lyra-actions-runs">
            {visible.map((run) => {
              return (
                <Link
                  className="lyra-workspace-row"
                  key={run.id}
                  to="/$owner/$repo/runs/$number"
                  params={{ owner, repo, number: run.number }}
                  search={{}}
                >
                  <RunState status={run.status} />
                  <div className="lyra-workspace-grow">
                    <strong className="lyra-workspace-rowTitle">{run.title || run.name}</strong>
                    <p className="lyra-workspace-meta">
                      {run.name} #{run.number} · {triggerLabels[run.event] || run.event} ·{" "}
                      <ActorName actor={run.author} />{" "}
                      <span className="lyra-actions-ref">
                        <GitBranchIcon size={16} />
                        {run.head_ref || run.ref.replace(/^refs\/(heads|tags)\//, "")}
                      </span>{" "}
                      <code>{run.sha.slice(0, 7)}</code>
                    </p>
                  </div>
                  <div className="lyra-actions-runTimes">
                    <span>
                      <CalendarIcon size={16} />
                      <DateTime value={run.created_at} />
                    </span>
                    {run.started_at ? (
                      <span>
                        <StopwatchIcon size={16} />
                        {duration(run.started_at, run.stopped_at)}
                      </span>
                    ) : (
                      <span className="lyra-workspace-status" data-state={run.status}>
                        {run.status}
                      </span>
                    )}
                  </div>
                </Link>
              );
            })}
            {visible.length === 0 && <p className="lyra-workspace-empty">No matching runs.</p>}
          </div>
        )}
        {runs.data?.length === limit && limit < 1000 && (
          <div className="lyra-workspace-actions">
            <Link
              className="lyra-controls-button"
              to="/$owner/$repo/runs"
              params={{ owner, repo }}
              search={{ ...search, limit: String(limit + 50) }}
            >
              Load older runs
            </Link>
          </div>
        )}
        {selected?.dispatch && (
          <FormDialog
            id="dispatch-workflow"
            title={<>Run {selected.name}</>}
            onSubmit={dispatchRun}
            pending={dispatch.isPending}
            error={dispatch.error}
            submitLabel={dispatch.isPending ? "Queuing…" : "Run workflow"}
            fieldsKey={`${selected.path}/${reference}`}
            description={
              <>
                <p className="lyra-workspace-meta">
                  Using <code>{reference}</code>
                </p>
              </>
            }
          >
            {Object.entries(selected.inputs).map(([name, input]) => (
              <div key={name}>
                {input.type === "boolean" && (
                  <label className="lyra-workspace-checkbox">
                    <input
                      name={`input:${name}`}
                      type="checkbox"
                      defaultChecked={input.default === "true"}
                    />
                    {name}
                  </label>
                )}
                {input.type === "choice" && (
                  <Field
                    label={
                      <>
                        {name}
                        {input.required && " *"}
                      </>
                    }
                  >
                    <select
                      className="lyra-controls-select"
                      name={`input:${name}`}
                      defaultValue={input.default}
                      required={input.required}
                    >
                      <option value="">Select…</option>
                      {input.options?.map((option) => (
                        <option key={option}>{option}</option>
                      ))}
                    </select>
                  </Field>
                )}
                {!["choice", "boolean"].includes(input.type) && (
                  <Field
                    label={
                      <>
                        {name}
                        {input.required && " *"}
                      </>
                    }
                  >
                    <input
                      name={`input:${name}`}
                      type={input.type === "number" ? "number" : "text"}
                      step={input.type === "number" ? "any" : undefined}
                      defaultValue={input.default}
                      required={input.required}
                    />
                  </Field>
                )}
                {input.description && <p className="lyra-workspace-meta">{input.description}</p>}
              </div>
            ))}
          </FormDialog>
        )}
      </div>
    </div>
  );
}
