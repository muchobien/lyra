import { createFileRoute } from "@tanstack/react-router";
import { RepositoryPage } from "@/pages/RepositoryPage";
import { searchValues } from "@/routing/search";
import {
  getGetRepoQueryOptions,
  getGetRepositoryBlobQueryOptions,
  getGetRepositoryTreeQueryOptions,
} from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/")({
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["ref", "path", "file"]),
  loaderDeps: ({ search }) => search,
  loader: async ({ context: { queryClient }, params: { owner, repo }, deps }) => {
    const metadata = await loadQuery(queryClient, getGetRepoQueryOptions(owner, repo));
    const ref = deps.ref || metadata.default_branch || "main";
    if (deps.file) {
      await loadQuery(
        queryClient,
        getGetRepositoryBlobQueryOptions(owner, repo, { ref, path: deps.file }),
      );
    } else {
      await loadQuery(
        queryClient,
        getGetRepositoryTreeQueryOptions(owner, repo, { ref, path: deps.path || "" }),
      );
    }
  },
  head: ({ params, match }) => ({
    meta: [
      {
        title: `${match.loaderDeps.file || match.loaderDeps.path || `${params.owner}/${params.repo}`} · Lyra`,
      },
    ],
  }),
  component: RepositoryPage,
});
