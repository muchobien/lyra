import { Field } from "@/components/forms/Field";
import { FormDialog } from "@/components/forms/FormDialog";
import { type CSSProperties } from "react";
import { Link } from "@tanstack/react-router";
import {
  ChecklistIcon,
  CommentIcon,
  FileDiffIcon,
  GitBranchIcon,
  GitCommitIcon,
  PencilIcon,
} from "@primer/octicons-react";
import { type DiscussionComment, type Review } from "@/api";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { CopyButton } from "@/components/controls/CopyButton";
import { ScrollNavigation } from "@/components/controls/ScrollNavigation";
import { ActorName } from "@/components/metadata/ActorName";
import { DateTime } from "@/components/metadata/DateTime";
import { Status } from "@/components/status/Status";
import { Markdown } from "@/components/markdown/Markdown";
import { MarkdownEditor } from "@/components/markdown/MarkdownEditor";
import { RichText } from "@/components/markdown/RichText";
import { FormError } from "@/components/forms/FormError";
import { ChecksPanel } from "@/components/code/ChecksPanel";
import { CommitRows } from "@/components/code/CommitRows";
import { DiffViewer } from "@/components/code/DiffViewer";
import { itemDetailPaths } from "@/routing/paths";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import { useConnect, type DetailProps } from "./connect";

type TimelineEntry = {
  kind: "comment" | "review";
  comment?: DiscussionComment;
  review?: Review;
  at: string;
  id: string;
};

export function ItemDetail({ kind, number }: DetailProps) {
  const {
    owner,
    repo,
    name,
    workspace,
    labels,
    tab,
    commits,
    patch,
    comment,
    editTitle,
    editBody,
    reviewBody,
    mergeTitle,
    mergeBody,
    reviewHead,
    mergeHead,
    reviewDecision,
    mergeMethod,
    anchor,
    lineBody,
    commenting,
    editing,
    reviewing,
    merging,
    lineCommenting,
    retry,
    retryCommits,
    retryPatch,
    submitComment,
    changeState,
    titleChanged,
    submitEdit,
    openReview,
    decisionChanged,
    submitReview,
    openMerge,
    methodChanged,
    mergeTitleChanged,
    submitMerge,
    openLineComment,
    submitLine,
  } = useConnect({ kind, number });
  if (workspace.error) {
    return <ErrorNotice error={workspace.error} retry={retry} />;
  }
  if (!workspace.data) {
    return (
      <output className="lyra-workspace-empty">
        Loading {kind === "pulls" ? "pull request" : "issue"}…
      </output>
    );
  }
  const { item, pull, merge, checks, can_write: canWrite } = workspace.data;
  const reasons = [...(merge?.blocked || [])];
  if (workspace.data.merge_error) {
    reasons.push(workspace.data.merge_error);
  }
  if (merge?.conflicts.length) {
    reasons.push(`Conflicts in ${merge.conflicts.join(", ")}`);
  }
  const availableMethods =
    merge?.methods.filter(
      (method) =>
        (method === "squash" && !merge.conflicts.length) || (method === "ff" && merge.fast_forward),
    ) || [];
  const canMerge =
    canWrite &&
    item.state === "open" &&
    !!merge &&
    reasons.length === 0 &&
    availableMethods.length > 0;
  const entries: TimelineEntry[] = [
    ...workspace.data.comments
      .filter((entry) => !entry.review_id)
      .map((entry) => ({
        kind: "comment" as const,
        comment: entry,
        at: entry.created_at,
        id: `comment-${entry.id}`,
      })),
    ...workspace.data.reviews.map((entry) => ({
      kind: "review" as const,
      review: entry,
      at: entry.created_at,
      id: `review-${entry.id}`,
    })),
  ].toSorted((first, second) => Date.parse(first.at) - Date.parse(second.at));
  const reviewers = new Map<string, Review>();
  for (const review of workspace.data.reviews) {
    if (review.state !== "commented") {
      reviewers.set(`${review.author.user}/${review.author.agent || ""}`, review);
    }
  }
  let detailContent;
  if (pull && tab === "files") {
    if (patch.error) {
      detailContent = <ErrorNotice error={patch.error} retry={retryPatch} />;
    } else if (patch.data) {
      detailContent = (
        <DiffViewer
          key={patch.data.head_sha}
          files={patch.data.files}
          owner={owner}
          repo={repo}
          section={kind}
          target={number}
          head={patch.data.head_sha}
          canComment={canWrite && item.state === "open"}
          onComment={openLineComment}
          comments={workspace.data.comments}
        />
      );
    } else {
      detailContent = <output className="lyra-workspace-empty">Loading diff…</output>;
    }
  } else if (pull && tab === "commits") {
    if (commits.error) {
      detailContent = <ErrorNotice error={commits.error} retry={retryCommits} />;
    } else if (commits.data) {
      detailContent = <CommitRows commits={commits.data} owner={owner} repo={repo} />;
    } else {
      detailContent = <output className="lyra-workspace-empty">Loading commits…</output>;
    }
  } else if (pull && tab === "checks" && checks) {
    detailContent = <ChecksPanel checks={checks} owner={owner} repo={repo} />;
  } else {
    detailContent = (
      <div className="lyra-workspace-columns">
        <div>
          {pull && item.state === "open" && (
            <section className="lyra-workspace-panel" aria-label="Merge status">
              <div className="lyra-workspace-panelHeading">
                <div>
                  <strong>{canMerge ? "Ready to merge" : "Merge requirements"}</strong>
                  <p className="lyra-workspace-meta">
                    {reasons.length > 0
                      ? reasons[0]
                      : "The current head meets the repository’s merge rules."}
                  </p>
                </div>
                {canWrite && (
                  <button
                    className={canMerge ? "lyra-workspace-primary" : "lyra-controls-button"}
                    type="button"
                    disabled={!canMerge}
                    commandfor="merge-pull"
                    command="show-modal"
                    onClick={openMerge}
                  >
                    Merge
                  </button>
                )}
              </div>
              <details className="lyra-workspace-disclosure">
                <summary>Merge details</summary>
                {reasons.length > 0 ? (
                  <ul>
                    {reasons.map((reason) => (
                      <li key={reason}>{reason}</li>
                    ))}
                  </ul>
                ) : (
                  <p className="lyra-workspace-meta">
                    Available methods:{" "}
                    {availableMethods
                      .map((method) => (method === "ff" ? "Fast-forward" : "Squash"))
                      .join(", ")}
                  </p>
                )}
                <p className="lyra-workspace-meta">
                  Head <code>{pull.head_sha.slice(0, 7)}</code>
                  {checks && (
                    <>
                      {" "}
                      · Checks: <Status value={checks.state} />
                    </>
                  )}
                </p>
              </details>
            </section>
          )}
          <article className="lyra-workspace-panel">
            <div className="lyra-workspace-panelHeading">
              <ActorName actor={item.author} />
              <DateTime value={item.created_at} />
            </div>
            {item.body ? (
              <Markdown html={workspace.data.html} />
            ) : (
              <p className="lyra-workspace-empty">No description.</p>
            )}
          </article>
          {entries.map((entry) => (
            <article className="lyra-workspace-panel" key={entry.id} id={entry.id}>
              {entry.comment && (
                <>
                  <div className="lyra-workspace-panelHeading">
                    <ActorName actor={entry.comment.author} />
                    <DateTime value={entry.at} />
                  </div>
                  {entry.comment.path && (
                    <div className="lyra-workspace-panelHeading">
                      <Link
                        className="lyra-workspace-link"
                        to={itemDetailPaths[kind]}
                        params={{ owner, repo, number: number }}
                        search={{ tab: "files", file: entry.comment.path }}
                      >
                        {entry.comment.path}:{entry.comment.line} ({entry.comment.side})
                      </Link>
                      {entry.comment.outdated && (
                        <span className="lyra-workspace-count">Outdated</span>
                      )}
                    </div>
                  )}
                  <Markdown html={entry.comment.html} />
                </>
              )}
              {entry.review && (
                <>
                  <div className="lyra-workspace-panelHeading">
                    <ActorName actor={entry.review.author} />
                    <Status value={entry.review.state} />
                  </div>
                  <p className="lyra-workspace-meta" style={{ padding: ".5rem 1.25rem 0" }}>
                    <DateTime value={entry.at} /> ·{" "}
                    <code>{entry.review.commit_sha.slice(0, 7)}</code>
                    {entry.review.state === "approved" &&
                      entry.review.commit_sha !== pull?.head_sha &&
                      " · Previous head"}
                  </p>
                  {entry.review.body && <Markdown html={entry.review.html} />}
                  {entry.review.comments.map((reviewComment) => (
                    <section key={reviewComment.id}>
                      <p className="lyra-workspace-meta" style={{ paddingInline: "1.25rem" }}>
                        {reviewComment.path}:{reviewComment.line}
                        {reviewComment.outdated && " · Outdated"}
                      </p>
                      <RichText source={reviewComment.body} />
                    </section>
                  ))}
                </>
              )}
            </article>
          ))}
          {canWrite && (
            <form onSubmit={submitComment}>
              <MarkdownEditor
                id="discussion-comment"
                label="Add a comment"
                value={comment.value}
                onChange={comment.setValue}
                required
              />
              <FormError error={commenting.error || editing.error} />
              <div className="lyra-workspace-actions">
                {item.state !== "merged" && (
                  <button
                    className="lyra-controls-button"
                    type="button"
                    onClick={changeState}
                    disabled={editing.isPending}
                  >
                    {item.state === "open" ? "Close" : "Reopen"}{" "}
                    {kind === "pulls" ? "pull request" : "issue"}
                  </button>
                )}
                <button
                  className="lyra-workspace-primary"
                  type="submit"
                  disabled={commenting.isPending || !comment.value.trim()}
                >
                  {commenting.isPending ? "Posting…" : "Comment"}
                </button>
              </div>
            </form>
          )}
        </div>
        <aside className="lyra-workspace-sidebar">
          <section>
            <h2>Assignees</h2>
            <p className="lyra-workspace-muted">{item.assignees.join(", ") || "Unassigned"}</p>
          </section>
          <section>
            <h2>Labels</h2>
            <div className="lyra-workspace-labels">
              {item.labels.map((label) => {
                const color = labels.data?.find((entry) => entry.name === label)?.color;
                return (
                  <span
                    className="lyra-workspace-label"
                    key={label}
                    style={color ? ({ "--label": `#${color}` } as CSSProperties) : undefined}
                  >
                    {label}
                  </span>
                );
              })}
              {item.labels.length === 0 && <span className="lyra-workspace-muted">None</span>}
            </div>
          </section>
          {pull && (
            <section>
              <h2>Reviewers</h2>
              {[...reviewers.values()].map((review) => (
                <p
                  className="lyra-workspace-meta"
                  key={`${review.author.user}/${review.author.agent}`}
                >
                  <ActorName actor={review.author} /> <Status value={review.state} />
                  {review.state === "approved" && review.commit_sha !== pull.head_sha && (
                    <small> · Previous head</small>
                  )}
                </p>
              ))}
              {reviewers.size === 0 && <p className="lyra-workspace-muted">No reviews yet.</p>}
            </section>
          )}
        </aside>
      </div>
    );
  }
  return (
    <>
      <div className="lyra-workspace-heading">
        <div className="lyra-workspace-grow">
          <h1 className="lyra-workspace-title">
            {item.title}
            <span>#{item.number}</span>
          </h1>
          <p className="lyra-workspace-meta">
            <Status value={item.state} issue={kind === "issues"} /> ·{" "}
            <ActorName actor={item.author} /> opened <DateTime value={item.created_at} />
          </p>
          {pull && (
            <p className="lyra-workspace-meta">
              <GitBranchIcon size={16} /> <code>{pull.head}</code> into <code>{pull.base}</code> ·{" "}
              <code>{pull.head_sha.slice(0, 7)}</code>
              {pull.merge_sha && (
                <>
                  {" "}
                  · merged as{" "}
                  <Link
                    className="lyra-workspace-link"
                    to="/$owner/$repo/commit/$sha"
                    params={{ owner, repo, sha: pull.merge_sha }}
                    search={{}}
                  >
                    {pull.merge_sha.slice(0, 7)}
                  </Link>
                </>
              )}
            </p>
          )}
        </div>
        <div className="lyra-workspace-headingActions">
          <CopyButton
            value={location.origin + `/${name}/${kind}/${number}`}
            label="Copy link"
            compact
          />
          {canWrite && (
            <button
              className="lyra-controls-button"
              type="button"
              commandfor="edit-item"
              command="show-modal"
            >
              <PencilIcon size={16} />
              Edit
            </button>
          )}
          {pull && canWrite && item.state === "open" && (
            <button
              className="lyra-workspace-primary"
              type="button"
              commandfor="review-pull"
              command="show-modal"
              onClick={openReview}
            >
              Submit review
            </button>
          )}
        </div>
      </div>
      {pull && (
        <ScrollNavigation
          className="lyra-workspace-tabs"
          label="Pull request sections"
          active={tab}
        >
          <Link
            to={itemDetailPaths[kind]}
            params={{ owner, repo, number: number }}
            search={{ tab: "overview" }}
            aria-current={tab === "overview" ? "page" : undefined}
          >
            <CommentIcon size={16} />
            Overview
          </Link>
          <Link
            to={itemDetailPaths[kind]}
            params={{ owner, repo, number: number }}
            search={{ tab: "commits" }}
            aria-current={tab === "commits" ? "page" : undefined}
          >
            <GitCommitIcon size={16} />
            Commits
          </Link>
          <Link
            to={itemDetailPaths[kind]}
            params={{ owner, repo, number: number }}
            search={{ tab: "checks" }}
            aria-current={tab === "checks" ? "page" : undefined}
          >
            <ChecklistIcon size={16} />
            Checks <span className="lyra-workspace-count">{checks?.jobs.length || 0}</span>
          </Link>
          <Link
            to={itemDetailPaths[kind]}
            params={{ owner, repo, number: number }}
            search={{ tab: "files" }}
            aria-current={tab === "files" ? "page" : undefined}
          >
            <FileDiffIcon size={16} />
            Files changed
          </Link>
        </ScrollNavigation>
      )}
      {detailContent}
      {canWrite && (
        <FormDialog
          id="edit-item"
          title={<>Edit {kind === "pulls" ? "pull request" : "issue"}</>}
          onSubmit={submitEdit}
          pending={editing.isPending}
          error={editing.error}
          submitLabel="Save changes"
        >
          <Field label="Title">
            <input value={editTitle.value} onChange={titleChanged} required maxLength={256} />
          </Field>
          <MarkdownEditor
            id="edit-body"
            label="Description"
            value={editBody.value}
            onChange={editBody.setValue}
          />
          <Field label="Labels">
            <input name="labels" defaultValue={item.labels.join(", ")} />
          </Field>
          <Field label="Assignees">
            <input name="assignees" defaultValue={item.assignees.join(", ")} />
          </Field>
        </FormDialog>
      )}
      {pull && canWrite && (
        <>
          <FormDialog
            id="review-pull"
            title="Review pull request"
            onSubmit={submitReview}
            pending={reviewing.isPending}
            error={reviewing.error}
            submitLabel={reviewing.isPending ? "Submitting…" : "Submit review"}
            submitDisabled={reviewDecision !== "approve" && !reviewBody.value.trim()}
            description={
              <>
                <p className="lyra-workspace-meta">
                  Reviewing <code>{reviewHead.slice(0, 7)}</code>
                </p>
              </>
            }
          >
            <Field label="Decision">
              <select
                className="lyra-controls-select"
                value={reviewDecision}
                onChange={decisionChanged}
              >
                <option value="comment">Comment</option>
                <option value="approve" disabled={workspace.data.is_author}>
                  Approve
                </option>
                <option value="request_changes" disabled={workspace.data.is_author}>
                  Request changes
                </option>
              </select>
            </Field>
            {workspace.data.is_author && (
              <p className="lyra-workspace-meta">Authors can comment on their own pull requests.</p>
            )}
            <MarkdownEditor
              id="review-body"
              label="Review"
              value={reviewBody.value}
              onChange={reviewBody.setValue}
              required={reviewDecision !== "approve"}
            />
          </FormDialog>
          <FormDialog
            id="merge-pull"
            title={<>Merge into {pull.base}</>}
            onSubmit={submitMerge}
            pending={merging.isPending}
            error={merging.error}
            submitLabel={merging.isPending ? "Merging…" : "Confirm merge"}
            submitDisabled={!canMerge}
            description={
              <>
                <p className="lyra-workspace-meta">
                  Head <code>{mergeHead.slice(0, 7)}</code>
                </p>
              </>
            }
          >
            <Field label="Merge method">
              <select className="lyra-controls-select" value={mergeMethod} onChange={methodChanged}>
                {availableMethods.map((method) => (
                  <option key={method} value={method}>
                    {method === "ff" ? "Fast-forward" : "Squash and merge"}
                  </option>
                ))}
              </select>
            </Field>
            {mergeMethod === "squash" && (
              <>
                <Field label="Commit title">
                  <input value={mergeTitle.value} onChange={mergeTitleChanged} required />
                </Field>
                <MarkdownEditor
                  id="merge-body"
                  label="Commit message"
                  value={mergeBody.value}
                  onChange={mergeBody.setValue}
                />
              </>
            )}
            <label className="lyra-workspace-checkbox">
              <input type="checkbox" name="delete_branch" />
              Delete {pull.head} after merging
            </label>
          </FormDialog>
          <FormDialog
            id="line-comment"
            title="Comment on line"
            onSubmit={submitLine}
            pending={lineCommenting.isPending}
            error={lineCommenting.error}
            submitLabel={lineCommenting.isPending ? "Posting…" : "Comment on line"}
            submitDisabled={!lineBody.value.trim()}
            description={
              <>
                <p className="lyra-workspace-meta">
                  <code>
                    {anchor.path}:{anchor.line}
                  </code>{" "}
                  · {anchor.side} · {anchor.head_sha.slice(0, 7)}
                </p>
              </>
            }
          >
            <MarkdownEditor
              id="line-comment-body"
              label="Comment"
              value={lineBody.value}
              onChange={lineBody.setValue}
              required
            />
          </FormDialog>
        </>
      )}
    </>
  );
}
