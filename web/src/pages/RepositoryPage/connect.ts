import { useLatestRelease } from "@/api/releases";
import { useEffect, useState, type ChangeEvent, type ToggleEvent } from "react";
import { useNavigate, useParams } from "@tanstack/react-router";
import {
  useGetRepo,
  useGetRepositoryTree,
  useGetRepositoryBlob,
  useGetRepositoryBlame,
  useListImages,
  useListReleases,
  useListRepositoryFiles,
  useListTags,
} from "@/api/generated/lyra";
import { splitSource } from "@/routing/paths";

export type RepositoryPageProps = {
  view: "tree" | "blob" | "blame";
  target: string;
  plain?: boolean;
};

export function useConnect({ view, target }: RepositoryPageProps) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const navigate = useNavigate();

  const name = `${owner}/${repo}`;
  const metadata = useGetRepo(owner, repo);
  const hasBranches = Boolean(metadata.data?.branches.length);
  const tags = useListTags(owner, repo);
  const source = splitSource(
    target,
    metadata.data?.branches.map((branch) => branch.name) || [],
    tags.data?.map((tag) => tag.name) || [],
  );
  const ref = source.ref || metadata.data?.default_branch || "main";
  const params = { ref, path: source.path };
  const tree = useGetRepositoryTree(owner, repo, params, {
    query: { enabled: !!metadata.data && view === "tree" },
  });
  const blobParams = params;
  const blob = useGetRepositoryBlob(owner, repo, blobParams, {
    query: { enabled: !!metadata.data && view === "blob" },
  });
  const blame = useGetRepositoryBlame(owner, repo, params, {
    query: { enabled: !!metadata.data && view === "blame" },
  });
  const latest = useLatestRelease(owner, repo);
  const images = useListImages(owner, repo);
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
    void navigate({
      to: "/$owner/$repo/tree/$",
      params: { owner, repo, _splat: event.target.value },
    });
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
    if (view === "blob") {
      void blob.refetch();
    } else if (view === "blame") {
      void blame.refetch();
    } else {
      void tree.refetch();
    }
  }

  function retryFiles() {
    void files.refetch();
  }

  const failure =
    metadata.error || { tree: tree.error, blob: blob.error, blame: blame.error }[view];
  const archiveURL = `/${owner}/${repo}/archive/${ref.split("/").map(encodeURIComponent).join("/")}`;

  return {
    owner,
    repo,
    view,
    path: source.path,
    name,
    metadata,
    hasBranches,
    ref,
    tree,
    blobParams,
    blob,
    blame,
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
    archiveURL,
    failure,
  };
}
