import { createFileRoute } from "@tanstack/react-router";
import { ActionsList } from "@/pages/ActionsList";
import { searchLimit, searchValues } from "@/routing/search";
import { getListRunsQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/runs/")({
  staticData: { section: "runs" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["state", "q", "workflow", "ref", "limit"]),
  loaderDeps: ({ search }) => ({ limit: searchLimit(search.limit, 50) }),
  loader: async ({ context, params: { owner, repo }, deps }) => {
    await loadQuery(context.queryClient, getListRunsQueryOptions(owner, repo, deps));
  },
  head: ({ params }) => ({ meta: [{ title: `Actions · ${params.owner}/${params.repo} · Lyra` }] }),
  component: ActionsList,
});
