import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { searchValues } from "@/routing/search";
import { getGetCommitWorkspaceQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/commit/$sha")({
  staticData: { section: "commit" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["file", "layout"]),
  remountDeps: ({ params }) => params,
  loader: async ({ context, params: { owner, repo, sha } }) => {
    await loadQuery(context.queryClient, getGetCommitWorkspaceQueryOptions(owner, repo, sha));
  },
  component: () => <CodeScreens section="commit" target={Route.useParams().sha} />,
});
