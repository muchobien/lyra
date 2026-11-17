import {
  useGetRepo,
  useListCommits,
  useListTags,
  useGetCommitWorkspace,
} from "@/api/generated/lyra";
import { type ChangeEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { searchLimit } from "@/routing/search";

export type CodeScreenProps = { section: "commits" | "commit" | "tags"; target: string };

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
    void metadata.refetch();
  }
  const error = metadata.error || commits.error || tags.error || detail.error;

  return {
    owner,
    repo,
    metadata,
    reference,
    limit,
    commits,
    tags,
    detail,
    branchChanged,
    retry,
    error,
  };
}
