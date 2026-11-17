import { createFileRoute } from "@tanstack/react-router";
import { ItemDetail } from "@/pages/ItemDetail";
import { searchValues } from "@/routing/search";
import { numberParams } from "@/routing/paths";
import { getGetIssueWorkspaceQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/issues/$number")({
  staticData: { section: "issues" },
  params: numberParams,
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["tab", "file", "layout"]),
  remountDeps: ({ params }) => params,
  loader: async ({ context, params: { owner, repo, number } }) => {
    await loadQuery(context.queryClient, getGetIssueWorkspaceQueryOptions(owner, repo, number));
  },
  component: () => <ItemDetail kind="issues" number={Route.useParams().number} />,
});
