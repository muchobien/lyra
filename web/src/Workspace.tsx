import { useRenderMarkdown } from "./api/generated/lyra";
import { useState } from "react";
import type { ChangeEvent, FormEvent, ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import {
  Check,
  CircleCheck,
  CircleDot,
  CircleX,
  Clock3,
  GitMerge,
  GitPullRequest,
  MessageSquare,
  Minus,
  X,
} from "lucide-react";
import { type Actor, type Checks, type Commit } from "./api";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

type StatusProps = { value: string; children?: ReactNode };
export function Status({ value, children }: StatusProps) {
  const icons = {
    success: CircleCheck,
    failure: CircleX,
    cancelled: CircleX,
    merged: GitMerge,
    approved: Check,
    changes_requested: CircleX,
    commented: MessageSquare,
    open: GitPullRequest,
    closed: CircleX,
    waiting: Clock3,
    blocked: Clock3,
    running: Clock3,
    pending: Clock3,
    skipped: Minus,
  };
  const Icon = icons[value as keyof typeof icons] || CircleDot;
  return (
    <span className={styles.status} data-state={value}>
      <Icon size={14} />
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
  function submit(event: FormEvent<HTMLFormElement>) {
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
      <X size={18} />
    </button>
  );
}

type CommitRowsProps = { commits: Commit[]; owner: string; repo: string };
export function CommitRows({ commits, owner, repo }: CommitRowsProps) {
  return (
    <div className={styles.panel}>
      {commits.map((commit) => (
        <article className={styles.row} key={commit.sha}>
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
              {commit.author.name} · <DateTime value={commit.committer.date} />
            </p>
          </div>
          <Status value={commit.signature.status} />
          <code>{commit.sha.slice(0, 7)}</code>
        </article>
      ))}
      {commits.length === 0 && <p className={styles.empty}>No commits.</p>}
    </div>
  );
}

type ChecksProps = { checks: Checks; owner: string; repo: string };
export function ChecksPanel({ checks, owner, repo }: ChecksProps) {
  return (
    <section className={styles.panel} aria-label="Checks">
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
          <div className={styles.grow}>
            <strong>{check.name}</strong>
            <p className={styles.meta}>{check.workflow}</p>
          </div>
          <Status value={check.status} />
          <span className={styles.link}>View logs</span>
        </Link>
      ))}
      {checks.jobs.length === 0 && (
        <p className={styles.empty}>No checks have run for this commit.</p>
      )}
    </section>
  );
}
