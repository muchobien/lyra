import { Link } from "@tanstack/react-router";
import {
  CheckCircleIcon,
  ChecklistIcon,
  CommentIcon,
  EyeIcon,
  GitMergeIcon,
  GitPullRequestIcon,
  IssueClosedIcon,
  IssueOpenedIcon,
  IssueReopenedIcon,
  LockIcon,
  PackageIcon,
  PencilIcon,
  PlayIcon,
  PulseIcon,
  RepoIcon,
  RepoPushIcon,
  ShieldIcon,
  SlidersIcon,
  StopIcon,
  TagIcon,
  VersionsIcon,
  XCircleIcon,
  type Icon,
} from "@primer/octicons-react";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ActorName } from "@/components/metadata/ActorName";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "@/components/code/CommitRows/styles.css";
import "@/components/status/Status/styles.css";
import "./styles.css";
import { itemDetailPaths } from "@/routing/paths";
import { useConnect } from "./connect";

const itemActions: Record<string, Description> = {
  opened: { icon: IssueOpenedIcon, verb: "opened" },
  edited: { icon: PencilIcon, verb: "edited" },
  closed: { icon: IssueClosedIcon, verb: "closed" },
  reopened: { icon: IssueReopenedIcon, verb: "reopened" },
  comment: { icon: CommentIcon, verb: "commented on" },
  merged: { icon: GitMergeIcon, verb: "merged" },
  review: { icon: EyeIcon, verb: "reviewed" },
  synchronized: { icon: GitPullRequestIcon, verb: "updated" },
  checks: { icon: ChecklistIcon, verb: "finished checks on" },
};

const actions: Record<string, Description> = {
  push: { icon: RepoPushIcon, verb: "pushed" },
  "repo.create": { icon: RepoIcon, verb: "created the repository" },
  "run.queued": { icon: PlayIcon, verb: "queued run" },
  "run.completed": { icon: CheckCircleIcon, verb: "completed run" },
  "run.cancelled": { icon: StopIcon, verb: "cancelled run" },
  "release.created": { icon: VersionsIcon, verb: "created release" },
  "release.published": { icon: VersionsIcon, verb: "published release" },
  "release.updated": { icon: VersionsIcon, verb: "updated release" },
  "release.deleted": { icon: VersionsIcon, verb: "deleted release" },
  "release.asset": { icon: PackageIcon, verb: "changed" },
  "image.pushed": { icon: PackageIcon, verb: "pushed container" },
  "image.deleted": { icon: PackageIcon, verb: "deleted container" },
  "label.create": { icon: TagIcon, verb: "created label" },
  "label.delete": { icon: TagIcon, verb: "deleted label" },
  "rule.set": { icon: ShieldIcon, verb: "set branch rule" },
  "rule.delete": { icon: ShieldIcon, verb: "deleted branch rule" },
  "variable.set": { icon: SlidersIcon, verb: "set variable" },
  "variable.delete": { icon: SlidersIcon, verb: "deleted variable" },
  "secret.set": { icon: LockIcon, verb: "set secret" },
  "secret.delete": { icon: LockIcon, verb: "deleted secret" },
};

type Description = { icon: Icon; verb: string };

export function Activity() {
  const {
    connection,
    owner,
    repo,
    search,
    limit,
    events,
    retry,
    filterChanged,
    visible,
    dayFormat,
    timeFormat,
  } = useConnect();
  return (
    <>
      <div className="lyra-workspace-toolbar">
        <input
          className="lyra-workspace-search"
          type="search"
          name="q"
          aria-label="Search activity"
          placeholder="Search activity…"
          value={search.q || ""}
          onChange={filterChanged}
        />
        <select
          className="lyra-controls-select"
          name="action"
          aria-label="Activity type"
          value={search.action || ""}
          onChange={filterChanged}
        >
          <option value="">All activity</option>
          <option value="push">Pushes</option>
          <option value="pr.">Pull requests</option>
          <option value="issue.">Issues</option>
          <option value="run.">Actions</option>
          <option value="release.">Releases</option>
          <option value="image.">Containers</option>
          <option value="rule.">Branch rules</option>
          <option value="variable.">Variables</option>
          <option value="secret.">Secrets</option>
        </select>
        <span className="lyra-workspace-live" data-connected={connection === "live"}>
          {connection}
        </span>
      </div>
      {events.error && <ErrorNotice error={events.error} retry={retry} />}
      {events.isPending && <output className="lyra-workspace-empty">Loading activity…</output>}
      {events.data && (
        <div className="lyra-workspace-panel">
          {visible.map((event, index) => {
            const data = (event.data && typeof event.data === "object" ? event.data : {}) as Record<
              string,
              unknown
            >;
            const day = dayFormat.format(new Date(event.at));
            const previous = visible[index - 1];
            const [kind, suffix = ""] = event.action.split(/\.(.*)/);
            const isItem = kind === "issue" || kind === "pr";
            const description = (isItem ? itemActions[suffix] : actions[event.action]) || {
              icon: PulseIcon,
              verb: event.action.replaceAll(".", " "),
            };
            const ActionIcon =
              event.action === "run.completed" && data.result !== "success"
                ? XCircleIcon
                : description.icon;
            const title = typeof data.title === "string" ? data.title : "";
            const tag = typeof data.tag === "string" ? data.tag : "";
            const label = typeof data.name === "string" ? data.name : "";
            const refs = Array.isArray(data.refs)
              ? data.refs.flatMap((entry: unknown) =>
                  entry &&
                  typeof entry === "object" &&
                  "ref" in entry &&
                  typeof entry.ref === "string"
                    ? [entry.ref.replace(/^refs\/(heads|tags)\//, "")]
                    : [],
                )
              : [];
            return (
              <div key={event.id}>
                {(!previous || dayFormat.format(new Date(previous.at)) !== day) && (
                  <h2 className="lyra-workspace-dayHeading">{day}</h2>
                )}
                <details className="lyra-activity-event">
                  <summary>
                    <span
                      className="lyra-workspace-status lyra-activity-icon"
                      data-state={event.action === "run.completed" ? data.result : undefined}
                    >
                      <ActionIcon size={16} />
                    </span>
                    <p className="lyra-activity-text">
                      <ActorName actor={event.actor} />{" "}
                      <span className="lyra-workspace-muted">
                        {event.action === "release.asset" && typeof data.change === "string"
                          ? data.change
                          : description.verb}
                        {isItem && (kind === "pr" ? " pull request" : " issue")}
                      </span>{" "}
                      {isItem && event.item && (
                        <Link
                          className="lyra-workspace-link"
                          to={itemDetailPaths[kind === "pr" ? "pulls" : "issues"]}
                          params={{
                            owner,
                            repo,
                            number: String(event.item),
                          }}
                          search={{}}
                        >
                          #{event.item}
                        </Link>
                      )}
                      {kind === "run" && typeof data.number === "number" && (
                        <Link
                          className="lyra-workspace-link"
                          to="/$owner/$repo/runs/$number"
                          params={{ owner, repo, number: String(data.number) }}
                          search={{}}
                        >
                          #{data.number}
                        </Link>
                      )}
                      {kind === "run" && typeof data.workflow === "string" && (
                        <span className="lyra-workspace-muted">
                          {" "}
                          · {data.workflow.split("/").at(-1)}
                        </span>
                      )}
                      {event.action === "release.asset" && (
                        <>
                          <code>{label}</code>{" "}
                          <span className="lyra-workspace-muted">
                            {data.change === "deleted" ? "from" : "to"}
                          </span>{" "}
                        </>
                      )}
                      {kind === "release" && tag && (
                        <Link
                          className="lyra-workspace-link"
                          to="/$owner/$repo/releases/tag/$"
                          params={{ owner, repo, _splat: `${tag}` }}
                          search={{}}
                        >
                          {tag}
                        </Link>
                      )}
                      {kind === "image" && (
                        <code>
                          {label}:{tag}
                        </code>
                      )}
                      {["label", "variable", "secret"].includes(kind) && <code>{label}</code>}
                      {kind === "rule" && typeof data.pattern === "string" && (
                        <code>{data.pattern}</code>
                      )}
                      {refs.length > 0 && (
                        <code>
                          {refs.slice(0, 3).join(", ")}
                          {refs.length > 3 && ` and ${refs.length - 3} more`}
                        </code>
                      )}
                      {title && title !== tag && (
                        <span className="lyra-workspace-muted"> · {title}</span>
                      )}
                    </p>
                    <time dateTime={event.at} title={new Date(event.at).toLocaleString()}>
                      {timeFormat.format(new Date(event.at))}
                    </time>
                  </summary>
                  <pre>{JSON.stringify(event.data, null, 2)}</pre>
                </details>
              </div>
            );
          })}
          {visible.length === 0 && <p className="lyra-workspace-empty">No matching activity.</p>}
        </div>
      )}
      {events.data?.length === limit && limit < 1000 && (
        <div className="lyra-workspace-actions">
          <Link
            className="lyra-controls-button"
            to="/$owner/$repo/activity"
            params={{ owner, repo }}
            search={{ ...search, limit: String(limit + 100) }}
          >
            Load older activity
          </Link>
        </div>
      )}
    </>
  );
}
