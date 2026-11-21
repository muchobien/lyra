import { useContext, useState, type ChangeEvent, type SubmitEvent } from "react";
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
  listPullCommits,
} from "@/api/generated/lyra";
import { formValue } from "@/utils/forms";
import { type CommentAnchor } from "@/components/code/DiffViewer";
import { useDraft } from "@/hooks/useDraft";
import { itemDetailPaths } from "@/routing/paths";
import { repositoryQueries } from "@/api/repositoryQueries";
import { type Actor } from "@/api/generated/models";
import { englishText, useOnDeviceModel } from "@/hooks/useOnDeviceModel";

export type DetailProps = { kind: "pulls" | "issues"; number: number };

export function useConnect({ kind, number }: Pick<DetailProps, "kind" | "number">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: itemDetailPaths[kind] });
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);

  const name = `${owner}/${repo}`;
  const issueWorkspace = useGetIssueWorkspace(owner, repo, number, {
    query: { enabled: kind === "issues" },
  });
  const pullWorkspace = useGetPullWorkspace(owner, repo, number, {
    query: { enabled: kind === "pulls" },
  });
  const workspace = kind === "pulls" ? pullWorkspace : issueWorkspace;
  const labels = useListLabels(owner, repo);
  const tab = search.tab || "overview";
  const commits = useListPullCommits(owner, repo, number, {
    query: { enabled: kind === "pulls" && tab === "commits" },
  });
  const patch = useGetPullPatch(owner, repo, number, {
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

  const summary = useOnDeviceModel("summarizer", async () =>
    "Summarizer" in self
      ? Summarizer.availability({ expectedInputLanguages: ["en"], outputLanguage: "en" })
      : "unavailable",
  );

  const commitMessage = useOnDeviceModel("language-model", async () =>
    "LanguageModel" in self ? LanguageModel.availability(englishText) : "unavailable",
  );

  const lineBody = useDraft(
    `${draftKey}/line/${anchor.head_sha}/${anchor.side}/${anchor.path}/${anchor.line}`,
  );

  async function refresh() {
    await queryClient.invalidateQueries(repositoryQueries(owner, repo));
  }

  const commenting = useMutation({
    mutationFn: (body: string) =>
      kind === "pulls"
        ? commentPull(owner, repo, number, { body })
        : commentIssue(owner, repo, number, { body }),
    onSuccess: async () => {
      comment.clear();
      await refresh();
    },
  });

  const editing = useMutation({
    mutationFn: (body: EditItemRequest) =>
      kind === "pulls" ? editPull(owner, repo, number, body) : editIssue(owner, repo, number, body),
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

  function retry() {
    void workspace.refetch();
  }

  function retryCommits() {
    void commits.refetch();
  }

  function retryPatch() {
    void patch.refetch();
  }

  function summarize() {
    const data = workspace.data;
    if (!data) {
      return;
    }
    const author = (actor: Actor) =>
      actor.agent ? `${actor.agent} for ${actor.user}` : actor.user;
    const replies = [
      ...data.comments
        .filter((entry) => !entry.review_id)
        .map((entry) => ({
          at: entry.created_at,
          text: `${author(entry.author)} commented:\n${entry.body}`,
        })),
      ...data.reviews.map((entry) => ({
        at: entry.created_at,
        text: [
          `${author(entry.author)} reviewed (${entry.state}):\n${entry.body}`,
          ...entry.comments.map((line) => `On ${line.path}:${line.line}: ${line.body}`),
        ].join("\n"),
      })),
    ].toSorted((first, second) => Date.parse(first.at) - Date.parse(second.at));
    const thread = [
      `${author(data.item.author)} opened it:\n${data.item.body || "No description."}`,
      ...replies.map((entry) => entry.text),
    ];
    void summary.run(async (signal, monitor) => {
      const summarizer = await Summarizer.create({
        type: "key-points",
        format: "markdown",
        length: "medium",
        expectedInputLanguages: ["en"],
        outputLanguage: "en",
        sharedContext: `${kind === "pulls" ? "A pull request" : "An issue"} titled "${data.item.title}" in the Git repository ${name}.`,
        signal,
        monitor,
      });
      let input = thread.join("\n\n");
      let usage = await summarizer.measureInputUsage(input, { signal });
      while (input && usage > summarizer.inputQuota) {
        input = input.slice(0, Math.floor((input.length * summarizer.inputQuota) / usage) - 200);
        usage = await summarizer.measureInputUsage(input, { signal });
      }
      return summarizer.summarizeStreaming(input, { signal });
    });
  }

  function submitComment(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    commenting.mutate(comment.value);
  }

  function changeState() {
    editing.mutate({ state: workspace.data?.item.state === "open" ? "closed" : "open" });
  }

  function titleChanged(event: ChangeEvent<HTMLInputElement>) {
    editTitle.setValue(event.target.value);
  }

  function submitEdit(event: SubmitEvent<HTMLFormElement>) {
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

  function submitReview(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    reviewing.mutate({
      owner,
      repo,
      number,
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

  function draftCommitMessage() {
    const item = workspace.data?.item;
    if (!item) {
      return;
    }
    void commitMessage.run(async (signal, monitor) => {
      const commits = await listPullCommits(owner, repo, number, { signal });
      const session = await LanguageModel.create({
        ...englishText,
        initialPrompts: [
          {
            role: "system",
            content:
              "You write squash commit messages in the Conventional Commits format. The first line is the subject: a type such as feat, fix, refactor, docs, test, ci, build or chore, an optional scope in parentheses, a colon, then a lowercase summary of what the change does, with no trailing period and at most 72 characters. After a blank line, the body explains what changed and why in a few short lines, for a reader who hasn't seen the pull request. Only state what the pull request supports. Output only the message, without code fences.",
          },
        ],
        signal,
        monitor,
      });
      const log = commits
        .map((commit) => [`- ${commit.subject}`, commit.body.trim()].filter(Boolean).join("\n\n"))
        .join("\n");
      return session.promptStreaming(
        `Pull request title: ${item.title}\n\nDescription:\n${item.body || "None."}\n\nCommits, newest first:\n${log}`,
        { signal },
      );
    });
  }

  function acceptCommitMessage() {
    const [subject = "", ...body] = commitMessage.output.trim().split("\n");
    const suffix = ` (#${number})`;
    mergeTitle.setValue(subject.endsWith(suffix) ? subject : subject + suffix);
    mergeBody.setValue(body.join("\n").trim());
    commitMessage.reset();
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

  function submitMerge(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    merging.mutate({
      owner,
      repo,
      number,
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

  function submitLine(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    lineCommenting.mutate({
      owner,
      repo,
      number,
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
    summary,
    commitMessage,
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
    summarize,
    submitComment,
    changeState,
    titleChanged,
    submitEdit,
    openReview,
    decisionChanged,
    submitReview,
    openMerge,
    draftCommitMessage,
    acceptCommitMessage,
    methodChanged,
    mergeTitleChanged,
    submitMerge,
    openLineComment,
    submitLine,
  };
}
