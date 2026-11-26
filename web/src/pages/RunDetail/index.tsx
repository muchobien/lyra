import { Link } from "@tanstack/react-router";
import {
  ChevronDownIcon,
  ChevronUpIcon,
  AlertIcon,
  DownloadIcon,
  GitBranchIcon,
  InfoIcon,
  PauseIcon,
  PlayIcon,
  SyncIcon,
  XCircleIcon,
} from "@primer/octicons-react";
import { CopyButton } from "@/components/controls/CopyButton";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ResponsiveDetails } from "@/components/controls/ResponsiveDetails";
import { ActorName } from "@/components/metadata/ActorName";
import { DateTime } from "@/components/metadata/DateTime";
import { duration } from "@/utils/duration";
import { RunState } from "@/components/status/RunState";
import { Status } from "@/components/status/Status";
import { ConfirmDialog } from "@/components/forms/ConfirmDialog";
import { FormError } from "@/components/forms/FormError";
import { LogText } from "@/components/code/LogText";
import { matchesQuery } from "@/components/code/LogText/logic";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "./styles.css";
import { useConnect, type RunDetailProps } from "./connect";

export function RunDetail({ number }: RunDetailProps) {
  const {
    owner,
    repo,
    search,
    identity,
    name,
    run,
    jobs,
    selected,
    active,
    logsURL,
    logs,
    targetLine,
    annotations,
    following,
    timestamps,
    fullLogOpen,
    logPanel,
    matches,
    matchIndex,
    cancel,
    rerun,
    retry,
    retryLogs,
    rerunClicked,
    cancelRun,
    annotationClicked,
    toggleFollow,
    logScrolled,
    timestampsChanged,
    fullLogToggled,
    jobChanged,
    matchClicked,
    searchKeyDown,
    searchChanged,
  } = useConnect({ number });

  if (run.error) {
    return <ErrorNotice error={run.error} retry={retry} />;
  }
  if (!run.data) {
    return <output className="lyra-workspace-empty">Loading run…</output>;
  }

  const data = run.data;
  const canWrite = identity?.scope !== "read";
  const logLines = logs.data?.lines || [];
  const logTimes = timestamps ? logs.data?.times : undefined;
  const highlightedStep = selected?.started_at
    ? selected.steps.find((step) => step.result === "failure") ||
      selected.steps.find((step) => !!step.started_at && !step.stopped_at)
    : undefined;
  const targetStep = selected?.steps.find(
    (step) => targetLine > step.log_index && targetLine <= step.log_index + step.log_length,
  );
  const showFullLog = fullLogOpen || (!!targetLine && !targetStep);
  const triggerLabels: Record<string, string> = {
    workflow_dispatch: "Manual",
    pull_request: "Pull request",
    schedule: "Scheduled",
    release: "Release",
    push: "Push",
  };
  const annotationIcons = { error: XCircleIcon, warning: AlertIcon, notice: InfoIcon };
  let logContent;
  if (selected && !selected.started_at) {
    let waiting = "This job did not start.";
    if (selected.status === "blocked") {
      waiting = `Waiting for ${selected.needs.join(", ")} to finish.`;
    } else if (selected.status === "waiting") {
      waiting = `Waiting for a runner with the labels ${selected.runs_on.join(", ")}.`;
    }
    logContent = <p className="lyra-workspace-empty">{waiting}</p>;
  } else if (logs.error) {
    logContent = <ErrorNotice error={logs.error} retry={retryLogs} />;
  } else if (logs.isPending && selected) {
    logContent = <output className="lyra-workspace-empty">Loading logs…</output>;
  } else if (!logLines.length) {
    logContent = (
      <p className="lyra-workspace-empty">
        {active ? "Waiting for log output…" : "No log output for this job."}
      </p>
    );
  } else if (!selected?.steps.length) {
    logContent = (
      <LogText
        lines={logLines}
        times={logTimes}
        start={0}
        end={logLines.length}
        query={search.q}
        target={targetLine}
        empty="No log lines."
      />
    );
  } else if (active) {
    logContent = (
      <div>
        {selected.steps.map((step) =>
          step.started_at ? (
            <details
              className="lyra-actions-step"
              key={step.id}
              open={!step.stopped_at || step.result === "failure"}
              data-running={!step.stopped_at || undefined}
            >
              <summary>
                <Status value={step.result || "running"} />
                {step.name}
                <span className="lyra-workspace-meta lyra-actions-stepDuration">
                  {duration(step.started_at, step.stopped_at)}
                </span>
              </summary>
              <LogText
                lines={logLines}
                times={logTimes}
                start={step.log_index}
                end={step.stopped_at ? step.log_index + step.log_length : logLines.length}
                query={search.q}
                target={targetLine}
                empty="No log lines for this step."
                live={!step.stopped_at}
              />
            </details>
          ) : (
            <p className="lyra-actions-pendingStep" key={step.id}>
              <Status value="pending" />
              {step.name}
            </p>
          ),
        )}
        <details className="lyra-actions-step" open={showFullLog} onToggle={fullLogToggled}>
          <summary>Full log</summary>
          {showFullLog && (
            <LogText
              lines={logLines}
              times={logTimes}
              start={0}
              end={logLines.length}
              query={search.q}
              target={targetStep ? undefined : targetLine}
              empty="No log lines."
            />
          )}
        </details>
      </div>
    );
  } else {
    const query = search.q;
    const matchedSteps = selected.steps.filter(
      (step) =>
        step.id !== highlightedStep?.id &&
        !!query &&
        matchesQuery(logLines, step.log_index, step.log_index + step.log_length, query),
    );
    logContent = (
      <div>
        {highlightedStep && (
          <details open className="lyra-actions-step" data-step={highlightedStep.id}>
            <summary>
              <Status value={highlightedStep.result || "running"} />
              {highlightedStep.name}
            </summary>
            <LogText
              lines={logLines}
              times={logTimes}
              start={highlightedStep.log_index}
              end={highlightedStep.log_index + highlightedStep.log_length}
              query={search.q}
              target={targetLine}
              empty="No log lines for this step."
            />
          </details>
        )}
        <details
          className="lyra-actions-completedSteps"
          open={!highlightedStep || matchedSteps.length > 0}
        >
          <summary>All steps · {selected.steps.length}</summary>
          {selected.steps
            .filter((step) => step.id !== highlightedStep?.id)
            .map((step) => (
              <details
                className="lyra-actions-step"
                key={step.id}
                data-step={step.id}
                open={matchedSteps.includes(step)}
              >
                <summary>
                  <Status value={step.result || "skipped"} />
                  {step.name}
                  <span className="lyra-workspace-meta lyra-actions-stepDuration">
                    {duration(step.started_at, step.stopped_at)}
                  </span>
                </summary>
                <LogText
                  lines={logLines}
                  times={logTimes}
                  start={step.log_index}
                  end={step.log_index + step.log_length}
                  query={search.q}
                  target={targetLine}
                  empty="No log lines for this step."
                />
              </details>
            ))}
        </details>
        <details className="lyra-actions-step" open={showFullLog} onToggle={fullLogToggled}>
          <summary>Full log</summary>
          {showFullLog && (
            <LogText
              lines={logLines}
              times={logTimes}
              start={0}
              end={logLines.length}
              query={search.q}
              target={targetStep ? undefined : targetLine}
              empty="No log lines."
            />
          )}
        </details>
      </div>
    );
  }

  return (
    <>
      <title>{`${data.title || "Run"} #${number} · ${name} · Lyra`}</title>

      <div className="lyra-workspace-heading">
        <div className="lyra-workspace-grow">
          <h1 className="lyra-workspace-title">
            {data.title || data.name}
            <span>#{data.number}</span>
          </h1>
          <p className="lyra-workspace-meta">
            {data.name} · {triggerLabels[data.event] || data.event} ·{" "}
            <ActorName actor={data.author} />
          </p>
        </div>
        <div className="lyra-workspace-headingActions">
          <Status value={data.status} />
          <CopyButton
            value={location.origin + `/${name}/runs/${number}`}
            label="Copy run link"
            compact
          />
          {canWrite && active && (
            <button
              className="lyra-controls-button"
              type="button"
              commandfor="cancel-run"
              command="show-modal"
            >
              Cancel run
            </button>
          )}
          {canWrite && !active && (
            <button
              className="lyra-controls-button lyra-actions-rerunTrigger"
              type="button"
              popoverTarget="rerun-options"
            >
              <SyncIcon size={16} />
              Re-run
            </button>
          )}
        </div>
      </div>

      <ResponsiveDetails className="lyra-actions-metadata" query="(width > 600px)">
        <summary>Run details</summary>
        <div>
          <span>
            <GitBranchIcon size={16} />{" "}
            {data.head_ref || data.ref.replace(/^refs\/(heads|tags)\//, "")}
          </span>
          <Link
            to="/$owner/$repo/commit/$sha"
            params={{ owner, repo, sha: data.sha }}
            search={{}}
            className="lyra-workspace-link"
          >
            <code>{data.sha.slice(0, 7)}</code>
          </Link>
          <span>
            <DateTime value={data.created_at} />
          </span>
          <span>{duration(data.started_at, data.stopped_at)}</span>
          {data.pull && (
            <Link
              to="/$owner/$repo/pulls/$number"
              params={{ owner, repo, number: data.pull }}
              search={{}}
              className="lyra-workspace-link"
            >
              Pull request #{data.pull}
            </Link>
          )}
          <Link
            to="/$owner/$repo/blob/$"
            params={{ owner, repo, _splat: `${data.sha}/${data.workflow}` }}
            className="lyra-workspace-link"
          >
            Workflow file
          </Link>
        </div>
      </ResponsiveDetails>

      <FormError error={rerun.error} />

      <div className="lyra-actions-runWorkspace">
        <nav className="lyra-actions-jobs" aria-label="Run jobs">
          {jobs.map((job) => (
            <Link
              key={job.id}
              to="/$owner/$repo/runs/$number"
              params={{ owner, repo, number: number }}
              search={{ job: String(job.id) }}
              aria-current={selected?.id === job.id ? "page" : undefined}
            >
              <RunState status={job.status} />
              <span>{job.name}</span>
            </Link>
          ))}
        </nav>
        <section className="lyra-actions-logSection" aria-label="Job logs">
          <div className="lyra-actions-logToolbar">
            <strong>{selected?.name || "Jobs"}</strong>
            {selected && (
              <span className="lyra-workspace-meta">
                {selected.attempt > 0 && `Attempt ${selected.attempt} · `}
                {selected.runs_on.join(", ")}
              </span>
            )}
            <span className="lyra-workspace-grow" />
            {active && (
              <button type="button" className="lyra-controls-button" onClick={toggleFollow}>
                {following ? <PauseIcon size={16} /> : <PlayIcon size={16} />}
                {following ? "Pause following" : "Resume following"}
              </button>
            )}
            {selected && (
              <a
                href={logsURL}
                download={`run-${number}-${selected.key}.log`}
                className="lyra-controls-button"
              >
                <DownloadIcon size={16} />
                Download log
              </a>
            )}
          </div>
          <div className="lyra-actions-logFilters">
            <select
              className="lyra-controls-select lyra-actions-mobileJobs"
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
              className="lyra-workspace-search"
              type="search"
              aria-label="Search logs"
              placeholder="Search logs…"
              value={search.q || ""}
              onChange={searchChanged}
              onKeyDown={searchKeyDown}
            />
            {search.q && (
              <span className="lyra-actions-logMatches">
                <output className="lyra-workspace-meta" aria-live="polite">
                  {matches.length ? `${matchIndex + 1} of ${matches.length}` : "No matches"}
                </output>
                <button
                  type="button"
                  className="lyra-controls-button"
                  aria-label="Previous match"
                  title="Previous match (Shift+Enter)"
                  data-offset="-1"
                  disabled={!matches.length}
                  onClick={matchClicked}
                >
                  <ChevronUpIcon size={16} />
                </button>
                <button
                  type="button"
                  className="lyra-controls-button"
                  aria-label="Next match"
                  title="Next match (Enter)"
                  data-offset="1"
                  disabled={!matches.length}
                  onClick={matchClicked}
                >
                  <ChevronDownIcon size={16} />
                </button>
              </span>
            )}
            <label className="lyra-workspace-checkbox">
              <input type="checkbox" checked={timestamps} onChange={timestampsChanged} />
              Show timestamps
            </label>
          </div>
          {active && highlightedStep && (
            <p className="lyra-actions-activeStep">
              <Status value="running" />
              {highlightedStep.name}
            </p>
          )}
          {annotations.length > 0 && (
            <details open className="lyra-actions-annotations">
              <summary>Annotations · {annotations.length}</summary>
              <ul>
                {annotations.map((annotation) => {
                  const Icon = annotationIcons[annotation.level];
                  return (
                    <li key={annotation.line}>
                      <button
                        type="button"
                        data-level={annotation.level}
                        data-step={annotation.step?.id}
                        data-line={annotation.line}
                        onClick={annotationClicked}
                      >
                        <Icon size={16} />
                        <span>
                          <strong>
                            {annotation.level[0].toUpperCase() + annotation.level.slice(1)}
                            {annotation.title && ` · ${annotation.title}`}
                          </strong>
                          <span>{annotation.text}</span>
                          <span className="lyra-workspace-meta">
                            {annotation.location && `${annotation.location} · `}
                            {annotation.step ? annotation.step.name : "Full log"} · line{" "}
                            {annotation.line}
                          </span>
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </details>
          )}
          <div className="lyra-actions-logPanel" ref={logPanel} onScroll={logScrolled}>
            {logContent}
          </div>
          {jobs.length === 0 && <p className="lyra-workspace-empty">This run has no jobs.</p>}
        </section>
      </div>

      <div id="rerun-options" popover="auto" className="lyra-actions-rerunMenu">
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
