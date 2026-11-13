import { useContext, useEffect, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { Link, useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  GitBranch,
  GitCommitHorizontal,
  MessageSquare,
  Files,
  ListChecks,
  Pencil,
} from "lucide-react";
import {
  api,
  IdentityContext,
  repositoryAPI,
  write,
  type Commit,
  type DiffFile,
  type DiscussionComment,
  type ItemWorkspace,
  type Review,
} from "./api";
import { ErrorNotice, CopyButton, formValue } from "./Controls";
import {
  ActorName,
  ChecksPanel,
  CommitRows,
  DateTime,
  DialogClose,
  FormError,
  Markdown,
  MarkdownEditor,
  RichText,
  Status,
} from "./Workspace";
import { DiffViewer, type CommentAnchor } from "./DiffViewer";
import { useDraft } from "./drafts";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

type DetailProps = { kind: "pulls" | "issues"; number: string };
type TimelineEntry = {
  kind: "comment" | "review";
  comment?: DiscussionComment;
  review?: Review;
  at: string;
  id: string;
};
type EditRequest = {
  title?: string;
  body?: string;
  state?: string;
  add_labels?: string[];
  remove_labels?: string[];
  add_assignees?: string[];
  remove_assignees?: string[];
};
type ReviewRequest = { state: string; body: string; head_sha: string };
type MergeRequest = {
  method: string;
  head_sha: string;
  delete_branch: boolean;
  title: string;
  message: string;
};
export function ItemDetail({ kind, number }: DetailProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const endpoint = repositoryAPI(name, `/${kind}/${encodeURIComponent(number)}`);
  const workspace = useQuery({
    queryKey: [endpoint, "view"],
    queryFn: ({ signal }) => api<ItemWorkspace>(`${endpoint}/view`, signal),
  });
  const tab = search.tab || "overview";
  const commits = useQuery({
    queryKey: [`${endpoint}/commits`],
    queryFn: ({ signal }) => api<Commit[]>(`${endpoint}/commits`, signal),
    enabled: kind === "pulls" && tab === "commits",
  });
  const patch = useQuery({
    queryKey: [`${endpoint}/patch`],
    queryFn: ({ signal }) => api<DiffFile[]>(`${endpoint}/patch`, signal),
    enabled: kind === "pulls" && tab === "files",
  });
  const draftKey = `${identity?.actor.user}/${name}/${kind}/${number}`;
  const comment = useDraft(`${draftKey}/comment`);
  const editTitle = useDraft(`${draftKey}/edit/title`, workspace.data?.item.title);
  const editBody = useDraft(`${draftKey}/edit/body`, workspace.data?.item.body);
  const reviewBody = useDraft(`${draftKey}/review`);
  const mergeTitle = useDraft(
    `${draftKey}/merge/title`,
    workspace.data ? `${workspace.data.item.title} (#${number})` : "",
  );
  const mergeBody = useDraft(`${draftKey}/merge/body`, workspace.data?.item.body);
  const [reviewHead, setReviewHead] = useState("");
  const [mergeHead, setMergeHead] = useState("");
  const [reviewDecision, setReviewDecision] = useState("comment");
  const [mergeMethod, setMergeMethod] = useState("squash");
  const [anchor, setAnchor] = useState<CommentAnchor>({
    path: "",
    line: 0,
    side: "new",
    head_sha: "",
  });
  const lineBody = useDraft(
    `${draftKey}/line/${anchor.head_sha}/${anchor.side}/${anchor.path}/${anchor.line}`,
  );
  async function refresh() {
    await queryClient.invalidateQueries();
  }
  const commenting = useMutation({
    mutationFn: (body: string) => write(`${endpoint}/comments`, "POST", { body }),
    onSuccess: async () => {
      comment.clear();
      await refresh();
    },
  });
  const editing = useMutation({
    mutationFn: (body: EditRequest) => write(endpoint, "PATCH", body),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#edit-item")?.close();
      editTitle.clear();
      editBody.clear();
      await refresh();
    },
  });
  const reviewing = useMutation({
    mutationFn: (body: ReviewRequest) => write(`${endpoint}/reviews`, "POST", body),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#review-pull")?.close();
      reviewBody.clear();
      await refresh();
    },
  });
  const merging = useMutation({
    mutationFn: (body: MergeRequest) => write(`${endpoint}/merge`, "POST", body),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#merge-pull")?.close();
      mergeTitle.clear();
      mergeBody.clear();
      await refresh();
    },
  });
  const lineCommenting = useMutation({
    mutationFn: (body: CommentAnchor & { body: string }) =>
      write(`${endpoint}/comments`, "POST", body),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#line-comment")?.close();
      lineBody.clear();
      await refresh();
    },
  });
  useEffect(() => {
    document.title = `${workspace.data?.item.title || `#${number}`} · ${name} · Lyra`;
  }, [workspace.data?.item.title, number, name]);
  function retry() {
    void workspace.refetch();
  }
  function retryCommits() {
    void commits.refetch();
  }
  function retryPatch() {
    void patch.refetch();
  }
  function submitComment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    commenting.mutate(comment.value);
  }
  function changeState() {
    editing.mutate({ state: workspace.data?.item.state === "open" ? "closed" : "open" });
  }
  function titleChanged(event: ChangeEvent<HTMLInputElement>) {
    editTitle.setValue(event.target.value);
  }
  function submitEdit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspace.data) {
      return;
    }
    const data = new FormData(event.currentTarget);
    const labels = formValue(data, "labels")
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean);
    const assignees = formValue(data, "assignees")
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean);
    editing.mutate({
      title: editTitle.value,
      body: editBody.value,
      add_labels: labels.filter((label) => !workspace.data.item.labels.includes(label)),
      remove_labels: workspace.data.item.labels.filter((label) => !labels.includes(label)),
      add_assignees: assignees.filter(
        (assignee) => !workspace.data.item.assignees.includes(assignee),
      ),
      remove_assignees: workspace.data.item.assignees.filter(
        (assignee) => !assignees.includes(assignee),
      ),
    });
  }
  function openReview() {
    setReviewHead(workspace.data?.pull?.head_sha || "");
    reviewing.reset();
  }
  function decisionChanged(event: ChangeEvent<HTMLSelectElement>) {
    setReviewDecision(event.target.value);
  }
  function submitReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    reviewing.mutate({ state: reviewDecision, body: reviewBody.value, head_sha: reviewHead });
  }
  function openMerge() {
    setMergeHead(workspace.data?.pull?.head_sha || "");
    const plan = workspace.data?.merge;
    setMergeMethod(
      plan?.methods.find((method) => method === "squash" && !plan.conflicts.length) ||
        plan?.methods.find((method) => method === "ff" && plan.fast_forward) ||
        "squash",
    );
    merging.reset();
  }
  function methodChanged(event: ChangeEvent<HTMLSelectElement>) {
    setMergeMethod(event.target.value);
  }
  function mergeTitleChanged(event: ChangeEvent<HTMLInputElement>) {
    mergeTitle.setValue(event.target.value);
  }
  function submitMerge(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    merging.mutate({
      method: mergeMethod,
      head_sha: mergeHead,
      delete_branch: data.get("delete_branch") === "on",
      title: mergeTitle.value,
      message: mergeBody.value,
    });
  }
  function openLineComment(next: CommentAnchor) {
    setAnchor(next);
    lineCommenting.reset();
    document.querySelector<HTMLDialogElement>("#line-comment")?.showModal();
  }
  function submitLine(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    lineCommenting.mutate({ ...anchor, body: lineBody.value });
  }
  if (workspace.error) {
    return <ErrorNotice error={workspace.error} retry={retry} />;
  }
  if (!workspace.data) {
    return (
      <output className={styles.empty}>
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
          key={pull.head_sha}
          files={patch.data}
          owner={owner}
          repo={repo}
          section={kind}
          target={number}
          head={pull.head_sha}
          canComment={canWrite && item.state === "open"}
          onComment={openLineComment}
          comments={workspace.data.comments}
        />
      );
    } else {
      detailContent = <output className={styles.empty}>Loading diff…</output>;
    }
  } else if (pull && tab === "commits") {
    if (commits.error) {
      detailContent = <ErrorNotice error={commits.error} retry={retryCommits} />;
    } else if (commits.data) {
      detailContent = <CommitRows commits={commits.data} owner={owner} repo={repo} />;
    } else {
      detailContent = <output className={styles.empty}>Loading commits…</output>;
    }
  } else if (pull && tab === "checks" && checks) {
    detailContent = <ChecksPanel checks={checks} owner={owner} repo={repo} />;
  } else {
    detailContent = (
      <div className={styles.columns}>
        <div>
          <article className={styles.panel}>
            <div className={styles.panelHeading}>
              <ActorName actor={item.author} />
              <DateTime value={item.created_at} />
            </div>
            {item.body ? (
              <Markdown html={workspace.data.html} />
            ) : (
              <p className={styles.empty}>No description.</p>
            )}
          </article>
          {entries.map((entry) => (
            <article className={styles.panel} key={entry.id} id={entry.id}>
              {entry.comment && (
                <>
                  <div className={styles.panelHeading}>
                    <ActorName actor={entry.comment.author} />
                    <DateTime value={entry.at} />
                  </div>
                  {entry.comment.path && (
                    <div className={styles.panelHeading}>
                      <Link
                        className={styles.link}
                        to="/$owner/$repo/$section/$"
                        params={{ owner, repo, section: kind, _splat: number }}
                        search={{ tab: "files", file: entry.comment.path }}
                      >
                        {entry.comment.path}:{entry.comment.line} ({entry.comment.side})
                      </Link>
                      {entry.comment.outdated && <span className={styles.count}>Outdated</span>}
                    </div>
                  )}
                  <Markdown html={entry.comment.html} />
                </>
              )}
              {entry.review && (
                <>
                  <div className={styles.panelHeading}>
                    <ActorName actor={entry.review.author} />
                    <Status value={entry.review.state} />
                  </div>
                  <p className={styles.meta} style={{ padding: ".5rem 1.25rem 0" }}>
                    <DateTime value={entry.at} /> ·{" "}
                    <code>{entry.review.commit_sha.slice(0, 7)}</code>
                    {entry.review.state === "approved" &&
                      entry.review.commit_sha !== pull?.head_sha &&
                      " · Previous head"}
                  </p>
                  {entry.review.body && <Markdown html={entry.review.html} />}
                  {entry.review.comments.map((reviewComment) => (
                    <section key={reviewComment.id}>
                      <p className={styles.meta} style={{ paddingInline: "1.25rem" }}>
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
              <div className={styles.actions}>
                {item.state !== "merged" && (
                  <button
                    className={controls.button}
                    type="button"
                    onClick={changeState}
                    disabled={editing.isPending}
                  >
                    {item.state === "open" ? "Close" : "Reopen"}{" "}
                    {kind === "pulls" ? "pull request" : "issue"}
                  </button>
                )}
                <button
                  className={styles.primary}
                  type="submit"
                  disabled={commenting.isPending || !comment.value.trim()}
                >
                  {commenting.isPending ? "Posting…" : "Comment"}
                </button>
              </div>
            </form>
          )}
        </div>
        <aside className={styles.sidebar}>
          <section>
            <h2>Assignees</h2>
            <p className={styles.muted}>{item.assignees.join(", ") || "Unassigned"}</p>
          </section>
          <section>
            <h2>Labels</h2>
            <div className={styles.labels}>
              {item.labels.map((label) => (
                <span className={styles.label} key={label}>
                  {label}
                </span>
              ))}
              {item.labels.length === 0 && <span className={styles.muted}>None</span>}
            </div>
          </section>
          {pull && (
            <section>
              <h2>Reviewers</h2>
              {[...reviewers.values()].map((review) => (
                <p className={styles.meta} key={`${review.author.user}/${review.author.agent}`}>
                  <ActorName actor={review.author} /> <Status value={review.state} />
                  {review.state === "approved" && review.commit_sha !== pull.head_sha && (
                    <small> · Previous head</small>
                  )}
                </p>
              ))}
              {reviewers.size === 0 && <p className={styles.muted}>No reviews yet.</p>}
            </section>
          )}
        </aside>
      </div>
    );
  }
  return (
    <>
      <div className={styles.heading}>
        <div className={styles.grow}>
          <h1 className={styles.title}>
            {item.title}
            <span>#{item.number}</span>
          </h1>
          <p className={styles.meta}>
            <Status value={item.state} /> · <ActorName actor={item.author} /> opened{" "}
            <DateTime value={item.created_at} />
          </p>
          {pull && (
            <p className={styles.meta}>
              <GitBranch size={13} /> <code>{pull.head}</code> into <code>{pull.base}</code> ·{" "}
              <code>{pull.head_sha.slice(0, 7)}</code>
              {pull.merge_sha && (
                <>
                  {" "}
                  · merged as{" "}
                  <Link
                    className={styles.link}
                    to="/$owner/$repo/$section/$"
                    params={{ owner, repo, section: "commit", _splat: pull.merge_sha }}
                    search={{}}
                  >
                    {pull.merge_sha.slice(0, 7)}
                  </Link>
                </>
              )}
            </p>
          )}
        </div>
        <div className={styles.headingActions}>
          <CopyButton
            value={location.origin + `/${name}/${kind}/${number}`}
            label="Copy link"
            compact
          />
          {canWrite && (
            <button
              className={controls.button}
              type="button"
              commandfor="edit-item"
              command="show-modal"
            >
              <Pencil size={14} />
              Edit
            </button>
          )}
          {pull && canWrite && item.state === "open" && (
            <button
              className={styles.primary}
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
        <nav className={styles.tabs} aria-label="Pull request sections">
          <Link
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section: kind, _splat: number }}
            search={{ tab: "overview" }}
            aria-current={tab === "overview" ? "page" : undefined}
          >
            <MessageSquare size={15} />
            Overview
          </Link>
          <Link
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section: kind, _splat: number }}
            search={{ tab: "commits" }}
            aria-current={tab === "commits" ? "page" : undefined}
          >
            <GitCommitHorizontal size={15} />
            Commits
          </Link>
          <Link
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section: kind, _splat: number }}
            search={{ tab: "checks" }}
            aria-current={tab === "checks" ? "page" : undefined}
          >
            <ListChecks size={15} />
            Checks <span className={styles.count}>{checks?.jobs.length || 0}</span>
          </Link>
          <Link
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section: kind, _splat: number }}
            search={{ tab: "files" }}
            aria-current={tab === "files" ? "page" : undefined}
          >
            <Files size={15} />
            Files changed
          </Link>
        </nav>
      )}
      {pull && item.state === "open" && (
        <section className={styles.panel} aria-label="Merge status">
          <div className={styles.panelHeading}>
            <div>
              <strong>{canMerge ? "Ready to merge" : "Merge requirements"}</strong>
              <p className={styles.meta}>
                {reasons.length > 0
                  ? reasons[0]
                  : "The current head meets the repository’s merge rules."}
              </p>
            </div>
            {canWrite && (
              <button
                className={canMerge ? styles.primary : controls.button}
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
          {tab === "overview" && (
            <details className={styles.disclosure}>
              <summary>Merge details</summary>
              {reasons.length > 0 ? (
                <ul>
                  {reasons.map((reason) => (
                    <li key={reason}>{reason}</li>
                  ))}
                </ul>
              ) : (
                <p className={styles.meta}>
                  Available methods:{" "}
                  {availableMethods
                    .map((method) => (method === "ff" ? "Fast-forward" : "Squash"))
                    .join(", ")}
                </p>
              )}
              <p className={styles.meta}>
                Head <code>{pull.head_sha.slice(0, 7)}</code>
                {checks && (
                  <>
                    {" "}
                    · Checks: <Status value={checks.state} />
                  </>
                )}
              </p>
            </details>
          )}
        </section>
      )}
      {detailContent}
      {canWrite && (
        <dialog
          className={styles.dialog}
          id="edit-item"
          closedby="any"
          aria-labelledby="edit-item-title"
        >
          <form onSubmit={submitEdit}>
            <h2 id="edit-item-title">Edit {kind === "pulls" ? "pull request" : "issue"}</h2>
            <fieldset disabled={editing.isPending}>
              <label className={styles.field}>
                <span>Title</span>
                <input value={editTitle.value} onChange={titleChanged} required maxLength={256} />
              </label>
              <MarkdownEditor
                id="edit-body"
                label="Description"
                value={editBody.value}
                onChange={editBody.setValue}
              />
              <label className={styles.field}>
                <span>Labels</span>
                <input name="labels" defaultValue={item.labels.join(", ")} />
              </label>
              <label className={styles.field}>
                <span>Assignees</span>
                <input name="assignees" defaultValue={item.assignees.join(", ")} />
              </label>
              <FormError error={editing.error} />
              <div className={styles.dialogActions}>
                <button
                  className={controls.button}
                  type="button"
                  commandfor="edit-item"
                  command="close"
                >
                  Cancel
                </button>
                <button className={styles.primary} type="submit">
                  Save changes
                </button>
              </div>
            </fieldset>
            <DialogClose target="edit-item" />
          </form>
        </dialog>
      )}
      {pull && canWrite && (
        <>
          <dialog
            className={styles.dialog}
            id="review-pull"
            closedby="any"
            aria-labelledby="review-pull-title"
          >
            <form onSubmit={submitReview}>
              <h2 id="review-pull-title">Review pull request</h2>
              <p className={styles.meta}>
                Reviewing <code>{reviewHead.slice(0, 7)}</code>
              </p>
              <fieldset disabled={reviewing.isPending}>
                <label className={styles.field}>
                  <span>Decision</span>
                  <select
                    className={controls.select}
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
                </label>
                {workspace.data.is_author && (
                  <p className={styles.meta}>Authors can comment on their own pull requests.</p>
                )}
                <MarkdownEditor
                  id="review-body"
                  label="Review"
                  value={reviewBody.value}
                  onChange={reviewBody.setValue}
                  required={reviewDecision !== "approve"}
                />
                <FormError error={reviewing.error} />
                <div className={styles.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="review-pull"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={styles.primary} type="submit">
                    {reviewing.isPending ? "Submitting…" : "Submit review"}
                  </button>
                </div>
              </fieldset>
              <DialogClose target="review-pull" />
            </form>
          </dialog>
          <dialog
            className={styles.dialog}
            id="merge-pull"
            closedby="any"
            aria-labelledby="merge-pull-title"
          >
            <form onSubmit={submitMerge}>
              <h2 id="merge-pull-title">Merge into {pull.base}</h2>
              <p className={styles.meta}>
                Head <code>{mergeHead.slice(0, 7)}</code>
              </p>
              <fieldset disabled={merging.isPending}>
                <label className={styles.field}>
                  <span>Merge method</span>
                  <select className={controls.select} value={mergeMethod} onChange={methodChanged}>
                    {availableMethods.map((method) => (
                      <option key={method} value={method}>
                        {method === "ff" ? "Fast-forward" : "Squash and merge"}
                      </option>
                    ))}
                  </select>
                </label>
                {mergeMethod === "squash" && (
                  <>
                    <label className={styles.field}>
                      <span>Commit title</span>
                      <input value={mergeTitle.value} onChange={mergeTitleChanged} required />
                    </label>
                    <MarkdownEditor
                      id="merge-body"
                      label="Commit message"
                      value={mergeBody.value}
                      onChange={mergeBody.setValue}
                    />
                  </>
                )}
                <label className={styles.checkbox}>
                  <input type="checkbox" name="delete_branch" />
                  Delete {pull.head} after merging
                </label>
                <FormError error={merging.error} />
                <div className={styles.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="merge-pull"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={styles.primary} type="submit" disabled={!canMerge}>
                    {merging.isPending ? "Merging…" : "Confirm merge"}
                  </button>
                </div>
              </fieldset>
              <DialogClose target="merge-pull" />
            </form>
          </dialog>
          <dialog
            className={styles.dialog}
            id="line-comment"
            closedby="any"
            aria-labelledby="line-comment-title"
          >
            <form onSubmit={submitLine}>
              <h2 id="line-comment-title">Comment on line</h2>
              <p className={styles.meta}>
                <code>
                  {anchor.path}:{anchor.line}
                </code>{" "}
                · {anchor.side} · {anchor.head_sha.slice(0, 7)}
              </p>
              <fieldset disabled={lineCommenting.isPending}>
                <MarkdownEditor
                  id="line-comment-body"
                  label="Comment"
                  value={lineBody.value}
                  onChange={lineBody.setValue}
                  required
                />
                <FormError error={lineCommenting.error} />
                <div className={styles.dialogActions}>
                  <button
                    className={controls.button}
                    type="button"
                    commandfor="line-comment"
                    command="close"
                  >
                    Cancel
                  </button>
                  <button className={styles.primary} type="submit">
                    {lineCommenting.isPending ? "Posting…" : "Comment on line"}
                  </button>
                </div>
              </fieldset>
              <DialogClose target="line-comment" />
            </form>
          </dialog>
        </>
      )}
    </>
  );
}
