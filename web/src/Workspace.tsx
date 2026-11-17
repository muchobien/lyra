import { useRenderMarkdown } from "./api/generated/lyra";
import { useState } from "react";
import type { ChangeEvent, ReactNode, SubmitEvent } from "react";
import { Link } from "@tanstack/react-router";
import {
  BlockedIcon,
  CheckCircleFillIcon,
  CheckCircleIcon,
  CheckIcon,
  ClockIcon,
  CommentIcon,
  FileDirectoryIcon,
  GitMergeIcon,
  GitPullRequestIcon,
  IssueOpenedIcon,
  SkipIcon,
  StopIcon,
  VerifiedIcon,
  XCircleFillIcon,
  XCircleIcon,
  XIcon,
} from "@primer/octicons-react";
import { type Actor, type Checks, type Commit } from "./api";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

const runIcons = {
  success: CheckCircleFillIcon,
  failure: XCircleFillIcon,
  waiting: ClockIcon,
  blocked: BlockedIcon,
  cancelled: StopIcon,
  skipped: SkipIcon,
};

type RunStateProps = { status: string };
export function RunState({ status }: RunStateProps) {
  const RunIcon = runIcons[status as keyof typeof runIcons] || ClockIcon;
  return (
    <span className={`${styles.status} ${styles.runState}`} data-state={status} title={status}>
      {status === "running" ? (
        <svg className={styles.spinner} viewBox="0 0 16 16" width="16" height="16">
          <circle cx="8" cy="8" r="6.25" />
          <path d="M8 1.75a6.25 6.25 0 0 1 6.25 6.25" />
        </svg>
      ) : (
        <RunIcon size={16} />
      )}
      <span className={controls.srOnly}>{status}</span>
    </span>
  );
}

type StatusProps = { value: string; issue?: boolean; children?: ReactNode };
export function Status({ value, issue, children }: StatusProps) {
  const icons = {
    success: CheckCircleIcon,
    failure: XCircleIcon,
    cancelled: XCircleIcon,
    merged: GitMergeIcon,
    approved: CheckIcon,
    changes_requested: XCircleIcon,
    commented: CommentIcon,
    open: GitPullRequestIcon,
    closed: XCircleIcon,
    waiting: ClockIcon,
    blocked: ClockIcon,
    running: ClockIcon,
    pending: ClockIcon,
    skipped: SkipIcon,
  };
  const Icon =
    (issue && value === "open" ? IssueOpenedIcon : icons[value as keyof typeof icons]) ||
    IssueOpenedIcon;
  return (
    <span className={styles.status} data-state={value}>
      <Icon size={16} />
      {children || value.replaceAll("_", " ")}
    </span>
  );
}

type ActorProps = { actor: Actor };
export function ActorName({ actor }: ActorProps) {
  return (
    <span title={actor.agent ? `${actor.agent} for ${actor.user}` : actor.user}>
      {actor.agent ? (
        <>
          <strong>{actor.agent}</strong>
          <span className={styles.muted}> for </span>
          {actor.user}
        </>
      ) : (
        actor.user
      )}
    </span>
  );
}

type DateProps = { value?: string | null };
export function DateTime({ value }: DateProps) {
  if (!value) {
    return <span>—</span>;
  }
  return (
    <time dateTime={value} title={new Date(value).toLocaleString()}>
      {new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(value))}
    </time>
  );
}

export function duration(start?: string | null, stop?: string | null) {
  if (!start) {
    return "Not started";
  }
  const seconds = Math.max(
    0,
    Math.floor(((stop ? Date.parse(stop) : Date.now()) - Date.parse(start)) / 1000),
  );
  return new Intl.DurationFormat(undefined, { style: "narrow" }).format({
    hours: Math.floor(seconds / 3600),
    minutes: Math.floor((seconds % 3600) / 60),
    seconds: seconds % 60,
  });
}

type MarkdownProps = { html: string };
export function Markdown({ html }: MarkdownProps) {
  return <div className={styles.markdown} dangerouslySetInnerHTML={{ __html: html }} />;
}

type RichTextProps = { source: string };
export function RichText({ source }: RichTextProps) {
  const rendered = useRenderMarkdown(
    { body: source },
    { query: { staleTime: Infinity, enabled: !!source } },
  );
  if (!source) {
    return null;
  }
  if (rendered.data) {
    return <Markdown html={rendered.data.html} />;
  }
  return <div className={styles.plainText}>{source}</div>;
}

type EditorProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
  placeholder?: string;
};
export function MarkdownEditor({
  id,
  label,
  value,
  onChange,
  required = false,
  placeholder = "Write with Markdown…",
}: EditorProps) {
  const [preview, setPreview] = useState(false);
  function change(event: ChangeEvent<HTMLTextAreaElement>) {
    onChange(event.target.value);
  }
  function edit() {
    setPreview(false);
  }
  function showPreview() {
    setPreview(true);
  }
  return (
    <div className={styles.editor}>
      <div className={styles.editorTabs}>
        <label htmlFor={id}>{label}</label>
        <button type="button" onClick={edit} aria-pressed={!preview}>
          Write
        </button>
        <button type="button" onClick={showPreview} aria-pressed={preview}>
          Preview
        </button>
      </div>
      <textarea
        id={id}
        name="body"
        value={value}
        onChange={change}
        placeholder={placeholder}
        required={required}
        maxLength={65536}
        hidden={preview}
      />
      {preview && (
        <div className={styles.preview}>
          {value ? (
            <RichText source={value} />
          ) : (
            <p className={styles.muted}>Nothing to preview.</p>
          )}
        </div>
      )}
    </div>
  );
}

type FormErrorProps = { error: Error | null };
export function FormError({ error }: FormErrorProps) {
  return error ? (
    <p className={styles.formError} role="alert">
      {error.message}
    </p>
  ) : null;
}

type DialogProps = {
  id: string;
  title: string;
  children: ReactNode;
  confirm: string;
  onConfirm: () => void;
  pending?: boolean;
  error?: Error | null;
  danger?: boolean;
};
export function ConfirmDialog({
  id,
  title,
  children,
  confirm,
  onConfirm,
  pending = false,
  error = null,
  danger = false,
}: DialogProps) {
  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    onConfirm();
  }
  return (
    <dialog id={id} className={styles.dialog} closedby="any" aria-labelledby={`${id}-title`}>
      <form onSubmit={submit}>
        <h2 id={`${id}-title`}>{title}</h2>
        {children}
        <FormError error={error} />
        <div className={styles.dialogActions}>
          <button
            type="button"
            className={controls.button}
            commandfor={id}
            command="close"
            disabled={pending}
          >
            Cancel
          </button>
          <button
            type="submit"
            className={danger ? styles.danger : styles.primary}
            disabled={pending}
          >
            {pending ? "Working…" : confirm}
          </button>
        </div>
      </form>
    </dialog>
  );
}

type DialogCloseProps = { target: string };
export function DialogClose({ target }: DialogCloseProps) {
  return (
    <button
      type="button"
      className={styles.close}
      commandfor={target}
      command="close"
      aria-label="Close dialog"
    >
      <XIcon size={16} />
    </button>
  );
}

type CommitRowsProps = { commits: Commit[]; owner: string; repo: string };
export function CommitRows({ commits, owner, repo }: CommitRowsProps) {
  const dayFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "full" });
  const timeFormat = new Intl.DateTimeFormat(undefined, { timeStyle: "short" });
  return (
    <div className={`${styles.panel} ${styles.commits}`}>
      {commits.map((commit, index) => {
        const day = dayFormat.format(new Date(commit.committer.date));
        const previous = commits[index - 1];
        return (
          <div key={commit.sha}>
            {(!previous || dayFormat.format(new Date(previous.committer.date)) !== day) && (
              <h2 className={styles.dayHeading}>{day}</h2>
            )}
            <article className={styles.row}>
              <div className={styles.grow}>
                <Link
                  className={styles.rowTitle}
                  to="/$owner/$repo/$section/$"
                  params={{ owner, repo, section: "commit", _splat: commit.sha }}
                  search={{}}
                >
                  {commit.subject}
                </Link>
                <p className={styles.meta}>
                  <strong>{commit.author.name}</strong> committed at{" "}
                  <time
                    dateTime={commit.committer.date}
                    title={new Date(commit.committer.date).toLocaleString()}
                  >
                    {timeFormat.format(new Date(commit.committer.date))}
                  </time>
                </p>
              </div>
              {commit.signature.status === "verified" && (
                <span className={styles.verified} title="Verified">
                  <VerifiedIcon size={16} />
                  <span className={controls.srOnly}>Verified</span>
                </span>
              )}
              {!["verified", "unverified"].includes(commit.signature.status) && (
                <Status value={commit.signature.status} />
              )}
              <Link
                className={styles.sha}
                to="/$owner/$repo/$section/$"
                params={{ owner, repo, section: "commit", _splat: commit.sha }}
                search={{}}
              >
                {commit.sha.slice(0, 7)}
              </Link>
              <Link
                className={styles.browse}
                to="/$owner/$repo"
                params={{ owner, repo }}
                search={{ ref: commit.sha }}
                aria-label={`Browse files at ${commit.sha.slice(0, 7)}`}
                title="Browse files"
              >
                <FileDirectoryIcon size={16} />
              </Link>
            </article>
          </div>
        );
      })}
      {commits.length === 0 && <p className={styles.empty}>No commits.</p>}
    </div>
  );
}

type ChecksProps = { checks: Checks; owner: string; repo: string };
export function ChecksPanel({ checks, owner, repo }: ChecksProps) {
  return (
    <section className={`${styles.panel} ${styles.checks}`} aria-label="Checks">
      <div className={styles.panelHeading}>
        <h2>Checks</h2>
        <Status value={checks.state} />
      </div>
      {checks.jobs.map((check) => (
        <Link
          className={styles.row}
          key={`${check.run}-${check.job_id}`}
          to="/$owner/$repo/$section/$"
          params={{ owner, repo, section: "runs", _splat: String(check.run) }}
          search={{ job: String(check.job_id) }}
        >
          <RunState status={check.status} />
          <span className={styles.grow}>
            <strong>{check.name}</strong>{" "}
            <span className={styles.muted}>· {check.workflow.split("/").at(-1)}</span>
          </span>
          <span className={styles.link}>View logs</span>
        </Link>
      ))}
      {checks.jobs.length === 0 && (
        <p className={styles.empty}>No checks have run for this commit.</p>
      )}
    </section>
  );
}
