import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { getListTagsQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/tags")({
  staticData: { section: "tags" },
  loader: async ({ context, params: { owner, repo } }) => {
    await loadQuery(context.queryClient, getListTagsQueryOptions(owner, repo));
  },
  head: ({ params }) => ({ meta: [{ title: `Tags · ${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <CodeScreens section="tags" target="" />,
});
