import {
  useDeleteBranch,
  useGetRepo,
  useListBranches,
  useListCommits,
  useListTags,
  useGetCommitWorkspace,
} from "@/api/generated/lyra";
import { useContext, useState, type ChangeEvent, type MouseEvent } from "react";
import { IdentityContext } from "@/api";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { searchLimit } from "@/routing/search";

export type CodeScreenProps = {
  section: "commits" | "commit" | "tags" | "branches";
  target: string;
};

export function useConnect({ section, target }: Pick<CodeScreenProps, "section" | "target">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const limitSearch = useSearch({ strict: false, select: (search) => search.limit });
  const metadata = useGetRepo(owner, repo);
  const navigate = useNavigate();
  const reference = target || metadata.data?.default_branch || "main";
  const limit = searchLimit(limitSearch, 30, 500);
  const commits = useListCommits(
    owner,
    repo,
    { ref: reference, limit },
    { query: { enabled: section === "commits" && !!metadata.data?.branches.length } },
  );
  const tags = useListTags(owner, repo, { query: { enabled: section === "tags" } });
  const branches = useListBranches(owner, repo, { query: { enabled: section === "branches" } });
  const identity = useContext(IdentityContext);
  const canWrite = identity?.scope !== "read";
  const [deleting, setDeleting] = useState("");
  const removing = useDeleteBranch({
    mutation: {
      onSuccess: () => {
        document.querySelector<HTMLDialogElement>("#delete-branch")?.close();
        void branches.refetch();
        void metadata.refetch();
      },
    },
  });
  const detail = useGetCommitWorkspace(owner, repo, target, {
    query: { enabled: section === "commit" && !!target },
  });

  function branchChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({
      to: "/$owner/$repo/commits/$",
      params: { owner, repo, _splat: event.target.value },
      search: {},
    });
  }

  function prepareDelete(event: MouseEvent<HTMLButtonElement>) {
    setDeleting(event.currentTarget.dataset.name || "");
    removing.reset();
  }

  function deleteBranch() {
    removing.mutate({ owner, repo, params: { name: deleting } });
  }

  function retry() {
    if (section === "commits") {
      void commits.refetch();
    }
    if (section === "commit") {
      void detail.refetch();
    }
    if (section === "tags") {
      void tags.refetch();
    }
    if (section === "branches") {
      void branches.refetch();
    }
    void metadata.refetch();
  }

  const error = metadata.error || commits.error || tags.error || branches.error || detail.error;

  return {
    owner,
    repo,
    metadata,
    reference,
    limit,
    commits,
    tags,
    branches,
    canWrite,
    deleting,
    removing,
    detail,
    branchChanged,
    prepareDelete,
    deleteBranch,
    retry,
    error,
  };
}
