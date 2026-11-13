import { useContext, useEffect, useRef, useState } from "react";
import type { ChangeEvent, FormEvent, MouseEvent, UIEvent } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CirclePlay, Download, GitBranch, Pause, Play, RotateCw } from "lucide-react";
import {
  api,
  IdentityContext,
  repositoryAPI,
  request,
  write,
  type Repository,
  type Run,
  type Tag,
  type Workflow,
} from "./api";
import { CopyButton, ErrorNotice, formValue } from "./Controls";
import {
  ActorName,
  ConfirmDialog,
  DateTime,
  DialogClose,
  duration,
  FormError,
  Status,
} from "./Workspace";
import controls from "./Controls.module.css";
import common from "./Workspace.module.css";
import styles from "./Actions.module.css";

type DispatchRequest = { workflow: string; ref: string; inputs: Record<string, string> };
export function ActionsList() {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate({ from: "/$owner/$repo/$section" });
  const identity = useContext(IdentityContext);
  const queryClient = useQueryClient();
  const name = `${owner}/${repo}`;
  const limit = Math.max(50, Math.min(1000, Number(search.limit) || 50));
  const metadata = useQuery({
    queryKey: ["repository", name],
    queryFn: ({ signal }) => api<Repository>(repositoryAPI(name), signal),
  });
  const reference = search.ref || metadata.data?.default_branch || "main";
  const workflowsPath = repositoryAPI(name, `/workflows?ref=${encodeURIComponent(reference)}`);
  const workflows = useQuery({
    queryKey: [workflowsPath],
    queryFn: ({ signal }) => api<Workflow[]>(workflowsPath, signal),
    enabled: !!metadata.data,
  });
  const tags = useQuery({
    queryKey: [repositoryAPI(name, "/tags")],
    queryFn: ({ signal }) => api<Tag[]>(repositoryAPI(name, "/tags"), signal),
  });
  const runsPath = repositoryAPI(name, `/runs?limit=${limit}`);
  const runs = useQuery({
    queryKey: [runsPath],
    queryFn: ({ signal }) => api<Run[]>(runsPath, signal),
    refetchInterval: (query) =>
      query.state.data?.some((run) => ["waiting", "running", "blocked"].includes(run.status))
        ? 3000
        : false,
  });
  const selected = workflows.data?.find((workflow) => workflow.path === search.workflow);
  const visible =
    runs.data?.filter(
      (run) =>
        (!search.workflow || run.workflow === search.workflow) &&
        (!search.state || run.status === search.state) &&
        (!search.q ||
          `${run.title} ${run.name} ${run.number}`.toLowerCase().includes(search.q.toLowerCase())),
    ) || [];
  const dispatch = useMutation({
    mutationFn: (body: DispatchRequest) =>
      write<Run>(repositoryAPI(name, "/workflows/dispatch"), "POST", body),
    onSuccess: (run) => {
      document.querySelector<HTMLDialogElement>("#dispatch-workflow")?.close();
      void queryClient.invalidateQueries();
      void navigate({
        to: "/$owner/$repo/$section/$",
        params: { owner, repo, section: "runs", _splat: String(run.number) },
        search: {},
      });
    },
  });
  useEffect(() => {
    document.title = `Actions · ${name} · Lyra`;
  }, [name]);
  function retry() {
    void runs.refetch();
    void workflows.refetch();
  }
  function filtersChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    void navigate({
      search: { ...search, [event.target.name]: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }
  function dispatchRun(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) {
      return;
    }
    const form = new FormData(event.currentTarget);
    const inputs: Record<string, string> = {};
    for (const [name, input] of Object.entries(selected.inputs)) {
      inputs[name] =
        input.type === "boolean"
          ? String(form.has(`input:${name}`))
          : formValue(form, `input:${name}`);
    }
    dispatch.mutate({ workflow: selected.path, ref: reference, inputs });
  }
  function resetDispatch() {
    dispatch.reset();
  }
  const triggerLabels: Record<string, string> = {
    push: "Push",
    pull_request: "Pull request",
    workflow_dispatch: "Manual",
    schedule: "Scheduled",
    release: "Release",
  };
  return (
    <div className={styles.listWorkspace}>
      <aside className={styles.workflows}>
        <Link
          to="/$owner/$repo/$section"
          params={{ owner, repo, section: "runs" }}
          search={{ ref: search.ref }}
          aria-current={!search.workflow ? "page" : undefined}
        >
          <CirclePlay size={16} />
          All workflows
        </Link>
        {workflows.data?.map((workflow) => (
          <Link
            key={workflow.path}
            to="/$owner/$repo/$section"
            params={{ owner, repo, section: "runs" }}
            search={{ ref: search.ref, workflow: workflow.path }}
            aria-current={workflow.path === search.workflow ? "page" : undefined}
          >
            <span>{workflow.name}</span>
            {workflow.dispatch && <span className={common.count}>Manual</span>}
            {workflow.error && <Status value="failure">Invalid</Status>}
          </Link>
        ))}
        {workflows.data?.length === 0 && (
          <p className={common.meta}>No workflow files at this ref.</p>
        )}
      </aside>
      <div className={common.grow}>
        <div className={common.toolbar}>
          <input
            type="search"
            className={common.search}
            name="q"
            aria-label="Search recent runs"
            placeholder="Search recent runs…"
            value={search.q || ""}
            onChange={filtersChanged}
          />
          <select
            className={controls.select}
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
            className={controls.select}
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
              className={common.primary}
              type="button"
              commandfor="dispatch-workflow"
              command="show-modal"
              onClick={resetDispatch}
            >
              <Play size={15} />
              Run workflow
            </button>
          )}
        </div>
        {selected && (
          <div className={common.toolbar}>
            <strong>{selected.name}</strong>
            <Link
              className={common.link}
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{ ref: reference, file: selected.path }}
            >
              View workflow file
            </Link>
            {selected.error && <p className={common.formError}>{selected.error}</p>}
          </div>
        )}
        {runs.error && <ErrorNotice error={runs.error} retry={retry} />}
        {workflows.error && <ErrorNotice error={workflows.error} retry={retry} />}
        {runs.isPending && <output className={common.empty}>Loading runs…</output>}
        {runs.data && (
          <div className={common.panel}>
            {visible.map((run) => (
              <Link
                className={common.row}
                key={run.id}
                to="/$owner/$repo/$section/$"
                params={{ owner, repo, section: "runs", _splat: String(run.number) }}
                search={{}}
              >
                <Status value={run.status} />
                <div className={common.grow}>
                  <strong className={common.rowTitle}>{run.title || run.name}</strong>
                  <p className={common.meta}>
                    {run.name} #{run.number} · {triggerLabels[run.event] || run.event} ·{" "}
                    <ActorName actor={run.author} />
                  </p>
                  <p className={common.meta}>
                    <GitBranch size={12} />{" "}
                    {run.head_ref || run.ref.replace(/^refs\/(heads|tags)\//, "")} ·{" "}
                    <code>{run.sha.slice(0, 7)}</code>
                  </p>
                </div>
                <div className={styles.runTimes}>
                  <DateTime value={run.created_at} />
                  <span>{duration(run.started_at, run.stopped_at)}</span>
                </div>
              </Link>
            ))}
            {visible.length === 0 && <p className={common.empty}>No matching runs.</p>}
          </div>
        )}
        {runs.data?.length === limit && limit < 1000 && (
          <div className={common.actions}>
            <Link
              className={controls.button}
              to="/$owner/$repo/$section"
              params={{ owner, repo, section: "runs" }}
              search={{ ...search, limit: String(limit + 50) }}
            >
              Load older runs
            </Link>
          </div>
        )}
        {selected?.dispatch && (
          <dialog
            className={common.dialog}
            id="dispatch-workflow"
            closedby="any"
            aria-labelledby="dispatch-title"
          >
            <form onSubmit={dispatchRun}>
              <h2 id="dispatch-title">Run {selected.name}</h2>
              <p className={common.meta}>
                Using <code>{reference}</code>
              </p>
              <fieldset key={`${selected.path}/${reference}`} disabled={dispatch.isPending}>
                {Object.entries(selected.inputs).map(([name, input]) => (
                  <div key={name}>
                    {input.type === "boolean" && (
                      <label className={common.checkbox}>
                        <input
                          name={`input:${name}`}
                          type="checkbox"
                          defaultChecked={input.default === "true"}
                        />
                        {name}
                      </label>
                    )}
                    {input.type === "choice" && (
                      <label className={common.field}>
                        <span>
                          {name}
                          {input.required && " *"}
                        </span>
                        <select
                          className={controls.select}
                          name={`input:${name}`}
                          defaultValue={input.default}
                          required={input.required}
                        >
                          <option value="">Select…</option>
                          {input.options?.map((option) => (
                            <option key={option}>{option}</option>
                          ))}
                        </select>
                      </label>
                    )}
                    {!["choice", "boolean"].includes(input.type) && (
                      <label className={common.field}>
                        <span>
                          {name}
                          {input.required && " *"}
                        </span>
                        <input
                          name={`input:${name}`}
                          type={input.type === "number" ? "number" : "text"}
                          defaultValue={input.default}
                          required={input.required}
                        />
                      </label>
                    )}
                    {input.description && <p className={common.meta}>{input.description}</p>}
                  </div>
                ))}
                <FormError error={dispatch.error} />
                <div className={common.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="dispatch-workflow"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={common.primary} type="submit">
                    {dispatch.isPending ? "Queuing…" : "Run workflow"}
                  </button>
                </div>
              </fieldset>
              <DialogClose target="dispatch-workflow" />
            </form>
          </dialog>
        )}
      </div>
    </div>
  );
}

type Logs = { lines: string[]; offset: number; done: boolean };
type RunDetailProps = { number: string };
export function RunDetail({ number }: RunDetailProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate({ from: "/$owner/$repo/$section/$" });
  const identity = useContext(IdentityContext);
  const queryClient = useQueryClient();
  const name = `${owner}/${repo}`;
  const endpoint = repositoryAPI(name, `/runs/${encodeURIComponent(number)}`);
  const run = useQuery({
    queryKey: [endpoint],
    queryFn: ({ signal }) => api<Run>(endpoint, signal),
    refetchInterval: (query) =>
      query.state.data && ["waiting", "blocked", "running"].includes(query.state.data.status)
        ? 2000
        : false,
  });
  const jobs = run.data?.jobs || [];
  const selected =
    jobs.find((job) => String(job.id) === search.job) ||
    jobs.find((job) => job.status === "failure") ||
    jobs.find((job) => job.status === "running") ||
    jobs[0];
  const active = !!run.data && ["waiting", "blocked", "running"].includes(run.data.status);
  const logsURL = selected ? `${endpoint}/jobs/${selected.id}/logs` : "";
  const logKey = [logsURL, selected?.attempt || 0];
  const logs = useQuery({
    queryKey: logKey,
    enabled: !!selected,
    queryFn: async ({ signal }) => {
      const previous = queryClient.getQueryData<Logs>(logKey);
      const response = await request(`${logsURL}?offset=${previous?.offset || 0}`, signal);
      const text = await response.text();
      const lines = text ? text.replace(/\n$/, "").split("\n") : [];
      return {
        lines: [...(previous?.lines || []), ...lines],
        offset: Number(response.headers.get("Lyra-Log-Next") || 0),
        done: response.headers.get("Lyra-Log-Done") === "true",
      };
    },
    refetchInterval: (query) => {
      if (query.state.error || query.state.data?.done) {
        return false;
      }
      return 1000;
    },
  });
  const [following, setFollowing] = useState(true);
  const logPanel = useRef<HTMLDivElement>(null);
  const [metadataOpen] = useState(() => matchMedia("(width > 600px)").matches);
  const cancel = useMutation({
    mutationFn: () => write(`${endpoint}/cancel`, "POST"),
    onSuccess: () => {
      document.querySelector<HTMLDialogElement>("#cancel-run")?.close();
      void queryClient.invalidateQueries();
    },
  });
  const rerun = useMutation({
    mutationFn: (failed: boolean) =>
      write(`${endpoint}/rerun${failed ? "?failed=true" : ""}`, "POST"),
    onSuccess: () => {
      document.getElementById("rerun-options")?.hidePopover();
      void queryClient.invalidateQueries();
    },
  });
  useEffect(() => {
    document.title = `${run.data?.title || "Run"} #${number} · ${name} · Lyra`;
  }, [run.data?.title, number, name]);
  useEffect(() => {
    if (following && active && logPanel.current) {
      logPanel.current.scrollTop = logPanel.current.scrollHeight;
    }
  }, [logs.data?.offset, following, active]);
  function retry() {
    void run.refetch();
  }
  function retryLogs() {
    void logs.refetch();
  }
  function rerunClicked(event: MouseEvent<HTMLButtonElement>) {
    rerun.mutate(event.currentTarget.dataset.failed === "true");
  }
  function cancelRun() {
    cancel.mutate();
  }
  function toggleFollow() {
    setFollowing(!following);
  }
  function logScrolled(event: UIEvent<HTMLDivElement>) {
    const element = event.currentTarget;
    if (element.scrollHeight - element.scrollTop - element.clientHeight > 40) {
      setFollowing(false);
    }
  }
  function jobChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({ search: { ...search, job: event.target.value }, resetScroll: false });
    setFollowing(true);
  }
  function searchChanged(event: ChangeEvent<HTMLInputElement>) {
    void navigate({
      search: { ...search, q: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }
  if (run.error) {
    return <ErrorNotice error={run.error} retry={retry} />;
  }
  if (!run.data) {
    return <output className={common.empty}>Loading run…</output>;
  }
  const data = run.data;
  const canWrite = identity?.scope !== "read";
  const logLines = logs.data?.lines || [];
  const highlightedStep =
    selected?.steps.find((step) => step.result === "failure") ||
    selected?.steps.find((step) => !!step.started_at && !step.stopped_at);
  const filteredLines = logLines.filter(
    (line) => !search.q || line.toLowerCase().includes(search.q.toLowerCase()),
  );
  const triggerLabels: Record<string, string> = {
    workflow_dispatch: "Manual",
    pull_request: "Pull request",
    schedule: "Scheduled",
    release: "Release",
    push: "Push",
  };
  let logContent;
  if (logs.error) {
    logContent = <ErrorNotice error={logs.error} retry={retryLogs} />;
  } else if (logs.isPending && selected) {
    logContent = <output className={common.empty}>Loading logs…</output>;
  } else if (!logLines.length) {
    logContent = (
      <p className={common.empty}>
        {active ? "Waiting for log output…" : "No log output for this job."}
      </p>
    );
  } else if (active || !selected?.steps.length) {
    logContent = (
      <pre className={styles.logText}>{filteredLines.join("\n") || "No matching log lines."}</pre>
    );
  } else {
    logContent = (
      <div className={styles.steps}>
        {highlightedStep && (
          <details open className={styles.step}>
            <summary>
              <Status value={highlightedStep.result || "running"} />
              {highlightedStep.name}
            </summary>
            <pre className={styles.logText}>
              {logLines
                .slice(
                  highlightedStep.log_index,
                  highlightedStep.log_index + highlightedStep.log_length,
                )
                .filter((line) => !search.q || line.toLowerCase().includes(search.q.toLowerCase()))
                .join("\n") || "No log lines for this step."}
            </pre>
          </details>
        )}
        <details className={styles.completedSteps} open={!highlightedStep}>
          <summary>All steps · {selected.steps.length}</summary>
          {selected.steps
            .filter((step) => step.id !== highlightedStep?.id)
            .map((step) => (
              <details className={styles.step} key={step.id}>
                <summary>
                  <Status value={step.result || "skipped"} />
                  {step.name}
                  <span className={common.meta}>{duration(step.started_at, step.stopped_at)}</span>
                </summary>
                <pre className={styles.logText}>
                  {logLines
                    .slice(step.log_index, step.log_index + step.log_length)
                    .filter(
                      (line) => !search.q || line.toLowerCase().includes(search.q.toLowerCase()),
                    )
                    .join("\n") || "No log lines for this step."}
                </pre>
              </details>
            ))}
        </details>
        <details className={styles.step}>
          <summary>Full log</summary>
          <pre className={styles.logText}>
            {filteredLines.join("\n") || "No matching log lines."}
          </pre>
        </details>
      </div>
    );
  }
  return (
    <>
      <div className={common.heading}>
        <div className={common.grow}>
          <h1 className={common.title}>
            {data.title || data.name}
            <span>#{data.number}</span>
          </h1>
          <p className={common.meta}>
            {data.name} · {triggerLabels[data.event] || data.event} ·{" "}
            <ActorName actor={data.author} />
          </p>
        </div>
        <div className={common.headingActions}>
          <Status value={data.status} />
          <CopyButton
            value={location.origin + `/${name}/runs/${number}`}
            label="Copy run link"
            compact
          />
          {canWrite && active && (
            <button
              className={controls.button}
              type="button"
              commandfor="cancel-run"
              command="show-modal"
            >
              Cancel run
            </button>
          )}
          {canWrite && !active && (
            <button
              className={`${controls.button} ${styles.rerunTrigger}`}
              type="button"
              popoverTarget="rerun-options"
            >
              <RotateCw size={15} />
              Re-run
            </button>
          )}
        </div>
      </div>
      <details className={styles.metadata} open={metadataOpen}>
        <summary>Run details</summary>
        <div>
          <span>
            <GitBranch size={13} /> {data.head_ref || data.ref.replace(/^refs\/(heads|tags)\//, "")}
          </span>
          <Link
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section: "commit", _splat: data.sha }}
            search={{}}
            className={common.link}
          >
            <code>{data.sha.slice(0, 7)}</code>
          </Link>
          <span>
            <DateTime value={data.created_at} />
          </span>
          <span>{duration(data.started_at, data.stopped_at)}</span>
          {data.pull && (
            <Link
              to="/$owner/$repo/$section/$"
              params={{ owner, repo, section: "pulls", _splat: String(data.pull) }}
              search={{}}
              className={common.link}
            >
              Pull request #{data.pull}
            </Link>
          )}
          <Link
            to="/$owner/$repo"
            params={{ owner, repo }}
            search={{ ref: data.sha, file: data.workflow }}
            className={common.link}
          >
            Workflow file
          </Link>
        </div>
      </details>
      <FormError error={rerun.error} />
      <div className={styles.runWorkspace}>
        <nav className={styles.jobs} aria-label="Run jobs">
          {jobs.map((job) => (
            <Link
              key={job.id}
              to="/$owner/$repo/$section/$"
              params={{ owner, repo, section: "runs", _splat: number }}
              search={{ job: String(job.id) }}
              aria-current={selected?.id === job.id ? "page" : undefined}
            >
              <Status value={job.status} />
              <span>{job.name}</span>
            </Link>
          ))}
        </nav>
        <section className={styles.logSection} aria-label="Job logs">
          <div className={styles.logToolbar}>
            <strong>{selected?.name || "Jobs"}</strong>
            {selected && (
              <span className={common.meta}>
                Attempt {selected.attempt} · {selected.runs_on.join(", ")}
              </span>
            )}
            <span className={common.grow} />
            {active && (
              <button type="button" className={controls.button} onClick={toggleFollow}>
                {following ? <Pause size={14} /> : <Play size={14} />}
                {following ? "Pause following" : "Resume following"}
              </button>
            )}
            {selected && (
              <a
                href={logsURL}
                download={`run-${number}-${selected.key}.log`}
                className={controls.button}
              >
                <Download size={14} />
                Download log
              </a>
            )}
          </div>
          <div className={styles.logFilters}>
            <select
              className={`${controls.select} ${styles.mobileJobs}`}
              aria-label="Selected job"
              value={String(selected?.id || "")}
              onChange={jobChanged}
            >
              {jobs.map((job) => (
                <option key={job.id} value={job.id}>
                  {job.name} · {job.status}
                </option>
              ))}
            </select>
            <input
              className={common.search}
              type="search"
              aria-label="Filter log lines"
              placeholder="Filter log lines…"
              value={search.q || ""}
              onChange={searchChanged}
            />
          </div>
          {active && highlightedStep && (
            <p className={styles.activeStep}>
              <Status value="running" />
              {highlightedStep.name}
            </p>
          )}
          <div className={styles.logPanel} ref={logPanel} onScroll={logScrolled}>
            {logContent}
          </div>
          {jobs.length === 0 && <p className={common.empty}>This run has no jobs.</p>}
        </section>
      </div>
      <div id="rerun-options" popover="auto" className={styles.rerunMenu}>
        <button type="button" onClick={rerunClicked} data-failed="false" disabled={rerun.isPending}>
          Re-run all jobs
        </button>
        <button
          type="button"
          onClick={rerunClicked}
          data-failed="true"
          disabled={
            rerun.isPending ||
            !jobs.some((job) => job.status === "failure" || job.status === "cancelled")
          }
        >
          Re-run failed jobs
        </button>
      </div>
      <ConfirmDialog
        id="cancel-run"
        title={`Cancel run #${number}?`}
        confirm="Cancel run"
        onConfirm={cancelRun}
        pending={cancel.isPending}
        error={cancel.error}
        danger
      >
        <p>The current jobs will stop and queued jobs will be cancelled.</p>
      </ConfirmDialog>
    </>
  );
}
