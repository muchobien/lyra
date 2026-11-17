import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useContext, useState, type ChangeEvent, type FormEvent } from "react";
import { useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { IdentityContext } from "@/api";
import {
  type EditItemRequest,
  type ReviewRequest,
  type MergeRequest,
} from "@/api/generated/models";
import {
  useGetIssueWorkspace,
  useGetPullWorkspace,
  useListPullCommits,
  useGetPullPatch,
  commentIssue,
  commentPull,
  editIssue,
  editPull,
  useReviewPull,
  useMergePull,
  useCommentPull,
  useListLabels,
} from "@/api/generated/lyra";
import { formValue } from "@/utils/forms";
import { type CommentAnchor } from "@/components/code/DiffViewer";
import { useDraft } from "@/hooks/useDraft";
import { itemDetailPaths } from "@/routing/paths";

export type DetailProps = { kind: "pulls" | "issues"; number: string };

export function useConnect({ kind, number }: Pick<DetailProps, "kind" | "number">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: itemDetailPaths[kind] });
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const issueWorkspace = useGetIssueWorkspace(owner, repo, Number(number), {
    query: { enabled: kind === "issues" },
  });
  const pullWorkspace = useGetPullWorkspace(owner, repo, Number(number), {
    query: { enabled: kind === "pulls" },
  });
  const workspace = kind === "pulls" ? pullWorkspace : issueWorkspace;
  const labels = useListLabels(owner, repo);
  const tab = search.tab || "overview";
  const commits = useListPullCommits(owner, repo, Number(number), {
    query: { enabled: kind === "pulls" && tab === "commits" },
  });
  const patch = useGetPullPatch(owner, repo, Number(number), {
    query: { enabled: kind === "pulls" && tab === "files" },
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
  const [reviewDecision, setReviewDecision] = useState<ReviewRequest["state"]>("comment");
  const [mergeMethod, setMergeMethod] = useState<MergeRequest["method"]>("squash");
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
    mutationFn: (body: string) =>
      kind === "pulls"
        ? commentPull(owner, repo, Number(number), { body })
        : commentIssue(owner, repo, Number(number), { body }),
    onSuccess: async () => {
      comment.clear();
      await refresh();
    },
  });
  const editing = useMutation({
    mutationFn: (body: EditItemRequest) =>
      kind === "pulls"
        ? editPull(owner, repo, Number(number), body)
        : editIssue(owner, repo, Number(number), body),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#edit-item")?.close();
      editTitle.clear();
      editBody.clear();
      await refresh();
    },
  });
  const reviewing = useReviewPull({
    mutation: {
      onSuccess: async () => {
        document.querySelector<HTMLDialogElement>("#review-pull")?.close();
        reviewBody.clear();
        await refresh();
      },
    },
  });
  const merging = useMergePull({
    mutation: {
      onSuccess: async () => {
        document.querySelector<HTMLDialogElement>("#merge-pull")?.close();
        mergeTitle.clear();
        mergeBody.clear();
        await refresh();
      },
    },
  });
  const lineCommenting = useCommentPull({
    mutation: {
      onSuccess: async () => {
        document.querySelector<HTMLDialogElement>("#line-comment")?.close();
        lineBody.clear();
        await refresh();
      },
    },
  });
  useDocumentTitle(`${workspace.data?.item.title || `#${number}`} · ${name}`);
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
    const value = event.target.value;
    if (value === "comment" || value === "approve" || value === "request_changes") {
      setReviewDecision(value);
    }
  }
  function submitReview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    reviewing.mutate({
      owner,
      repo,
      number: Number(number),
      data: { state: reviewDecision, body: reviewBody.value, head_sha: reviewHead },
    });
  }
  function openMerge() {
    setMergeHead(workspace.data?.pull?.head_sha || "");
    const plan = workspace.data?.merge;
    if (plan?.methods.includes("squash") && !plan.conflicts.length) {
      setMergeMethod("squash");
    } else if (plan?.methods.includes("ff") && plan.fast_forward) {
      setMergeMethod("ff");
    } else {
      setMergeMethod("squash");
    }
    merging.reset();
  }
  function methodChanged(event: ChangeEvent<HTMLSelectElement>) {
    const value = event.target.value;
    if (value === "squash" || value === "ff") {
      setMergeMethod(value);
    }
  }
  function mergeTitleChanged(event: ChangeEvent<HTMLInputElement>) {
    mergeTitle.setValue(event.target.value);
  }
  function submitMerge(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    merging.mutate({
      owner,
      repo,
      number: Number(number),
      data: {
        method: mergeMethod,
        head_sha: mergeHead,
        delete_branch: data.get("delete_branch") === "on",
        title: mergeTitle.value,
        message: mergeBody.value,
      },
    });
  }
  function openLineComment(next: CommentAnchor) {
    setAnchor(next);
    lineCommenting.reset();
    document.querySelector<HTMLDialogElement>("#line-comment")?.showModal();
    document.getElementById("line-comment-body")?.focus();
  }
  function submitLine(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    lineCommenting.mutate({
      owner,
      repo,
      number: Number(number),
      data: { ...anchor, body: lineBody.value },
    });
  }

  return {
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
  };
}
