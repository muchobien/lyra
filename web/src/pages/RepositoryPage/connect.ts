import { useLatestRelease } from "@/api/releases";
import { useEffect, useState, type ChangeEvent, type ToggleEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import {
  useGetRepo,
  useGetRepositoryTree,
  useGetRepositoryBlob,
  useListImages,
  useListReleases,
  useListRepositoryFiles,
  useListTags,
} from "@/api/generated/lyra";

export function useConnect() {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: "/$owner/$repo/" });
  const navigate = useNavigate({ from: "/$owner/$repo/" });
  const name = `${owner}/${repo}`;
  const metadata = useGetRepo(owner, repo);
  const hasBranches = Boolean(metadata.data?.branches.length);
  const ref = search.ref || metadata.data?.default_branch || "main";
  const params = { ref, path: search.path || "" };
  const tree = useGetRepositoryTree(owner, repo, params, {
    query: { enabled: !!metadata.data && !search.file },
  });
  const blobParams = { ref, path: search.file || "" };
  const blob = useGetRepositoryBlob(owner, repo, blobParams, {
    query: { enabled: !!metadata.data && !!search.file },
  });
  const latest = useLatestRelease(owner, repo);
  const images = useListImages(owner, repo);
  const tags = useListTags(owner, repo);
  const releases = useListReleases(owner, repo);
  const [finderOpen, setFinderOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const files = useListRepositoryFiles(owner, repo, { ref }, { query: { enabled: finderOpen } });
  const visibleFiles =
    files.data?.filter((path) => path.toLowerCase().includes(filter.toLowerCase())) || [];

  useEffect(() => {
    function shortcut(event: KeyboardEvent) {
      if (
        !hasBranches ||
        event.key.toLowerCase() !== "t" ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey
      ) {
        return;
      }
      if (
        event.target instanceof HTMLElement &&
        event.target.closest("input, textarea, select, [contenteditable=true], dialog")
      ) {
        return;
      }
      const dialog = document.getElementById("file-finder");
      if (dialog instanceof HTMLDialogElement) {
        event.preventDefault();
        dialog.showModal();
      }
    }
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, [hasBranches]);

  function branchChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({ search: { ref: event.target.value } });
  }
  function searchChanged(event: ChangeEvent<HTMLInputElement>) {
    setFilter(event.target.value);
  }
  function finderToggled(event: ToggleEvent<HTMLDialogElement>) {
    setFinderOpen(event.newState === "open");
  }
  function closeFinder() {
    const dialog = document.getElementById("file-finder");
    if (dialog instanceof HTMLDialogElement) {
      dialog.close();
    }
    setFilter("");
  }
  function retry() {
    void metadata.refetch();
    if (search.file) {
      void blob.refetch();
    } else {
      void tree.refetch();
    }
  }
  function retryFiles() {
    void files.refetch();
  }

  const failure = metadata.error || (search.file ? blob.error : tree.error);

  return {
    owner,
    repo,
    search,
    name,
    metadata,
    hasBranches,
    ref,
    tree,
    blobParams,
    blob,
    latest,
    images,
    tags,
    releases,
    filter,
    files,
    visibleFiles,
    branchChanged,
    searchChanged,
    finderToggled,
    closeFinder,
    retry,
    retryFiles,
    failure,
  };
}
