import { createFileRoute } from "@tanstack/react-router";
import { ItemList } from "@/pages/ItemList";
import { itemListParams, searchValues } from "@/routing/search";
import { getListIssuesQueryOptions } from "@/api/generated/lyra";

export const Route = createFileRoute("/$owner/$repo/issues/")({
  staticData: { section: "issues" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["state", "q", "label", "assignee", "limit"]),
  loaderDeps: ({ search }) => itemListParams(search),
  loader: ({ context, params: { owner, repo }, deps }) => {
    void context.queryClient.query(getListIssuesQueryOptions(owner, repo, deps)).catch(() => {});
  },
  head: ({ params }) => ({ meta: [{ title: `Issues · ${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <ItemList kind="issues" />,
});
