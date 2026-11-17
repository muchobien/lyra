import { createFileRoute } from "@tanstack/react-router";
import { Activity } from "@/pages/Activity";
import { searchLimit, searchValues } from "@/routing/search";
import { getListEventsQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/activity")({
  staticData: { section: "activity" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["q", "action", "limit"]),
  loaderDeps: ({ search }) => ({ limit: searchLimit(search.limit, 100) }),
  loader: async ({ context, params: { owner, repo }, deps }) => {
    await loadQuery(
      context.queryClient,
      getListEventsQueryOptions({ repo: `${owner}/${repo}`, limit: deps.limit }),
    );
  },
  head: ({ params }) => ({ meta: [{ title: `Activity · ${params.owner}/${params.repo} · Lyra` }] }),
  component: Activity,
});
