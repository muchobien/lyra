import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { getListBranchesQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/branches")({
  staticData: { section: "branches" },
  loader: async ({ context, params: { owner, repo } }) => {
    await loadQuery(context.queryClient, getListBranchesQueryOptions(owner, repo));
  },
  head: ({ params }) => ({ meta: [{ title: `Branches · ${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <CodeScreens section="branches" target="" />,
});
