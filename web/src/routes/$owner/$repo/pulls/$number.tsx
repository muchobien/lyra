import { createFileRoute } from "@tanstack/react-router";
import { ItemDetail } from "@/pages/ItemDetail";
import { searchValues } from "@/routing/search";
import { numberParams } from "@/routing/paths";
import {
  getGetPullPatchQueryOptions,
  getGetPullWorkspaceQueryOptions,
  getListPullCommitsQueryOptions,
} from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/pulls/$number")({
  staticData: { section: "pulls" },
  params: numberParams,
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["tab", "file", "layout"]),
  loaderDeps: ({ search }) => ({ tab: search.tab || "overview" }),
  loader: async ({ context: { queryClient }, params: { owner, repo, number }, deps }) => {
    const workspace = loadQuery(queryClient, getGetPullWorkspaceQueryOptions(owner, repo, number));
    if (deps.tab === "commits") {
      void queryClient.query(getListPullCommitsQueryOptions(owner, repo, number)).catch(() => {});
    } else if (deps.tab === "files") {
      void queryClient.query(getGetPullPatchQueryOptions(owner, repo, number)).catch(() => {});
    }
    await workspace;
  },
  remountDeps: ({ params }) => params,
  component: () => <ItemDetail kind="pulls" number={Route.useParams().number} />,
});
