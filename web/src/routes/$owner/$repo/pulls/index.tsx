import { createFileRoute } from "@tanstack/react-router";
import { ItemList } from "@/pages/ItemList";
import { itemListParams, searchValues } from "@/routing/search";
import { getListPullsQueryOptions } from "@/api/generated/lyra";

export const Route = createFileRoute("/$owner/$repo/pulls/")({
  staticData: { section: "pulls" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["state", "q", "label", "assignee", "limit"]),
  loaderDeps: ({ search }) => itemListParams(search),
  loader: ({ context, params: { owner, repo }, deps }) => {
    void context.queryClient.query(getListPullsQueryOptions(owner, repo, deps)).catch(() => {});
  },
  head: ({ params }) => ({
    meta: [{ title: `Pull requests · ${params.owner}/${params.repo} · Lyra` }],
  }),
  component: () => <ItemList kind="pulls" />,
});
