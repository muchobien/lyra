import { type ChangeEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useListReleases, useGetRepo, useListImages } from "@/api/generated/lyra";

export function useConnect() {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: "/$owner/$repo/images" });
  const navigate = useNavigate({ from: "/$owner/$repo/images" });
  const images = useListImages(owner, repo);
  const releases = useListReleases(owner, repo);
  const metadata = useGetRepo(owner, repo);
  function retry() {
    void images.refetch();
  }
  function searchChanged(event: ChangeEvent<HTMLInputElement>) {
    void navigate({ search: { q: event.target.value }, replace: true, resetScroll: false });
  }
  const visible =
    images.data
      ?.filter(
        (image) =>
          !search.q ||
          `${image.name}:${image.tag} ${image.digest}`
            .toLowerCase()
            .includes(search.q.toLowerCase()),
      )
      .toSorted((first, second) => Date.parse(second.pushed_at) - Date.parse(first.pushed_at)) ||
    [];

  return { owner, repo, search, images, releases, metadata, retry, searchChanged, visible };
}
