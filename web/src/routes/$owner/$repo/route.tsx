import { createFileRoute } from "@tanstack/react-router";
import { RepositoryLayout } from "@/pages/RepositoryLayout";
import { getGetRepoQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo")({
  loader: async ({ context, params: { owner, repo } }) => {
    await loadQuery(context.queryClient, getGetRepoQueryOptions(owner, repo));
  },
  remountDeps: ({ params }) => ({ owner: params.owner, repo: params.repo }),
  component: RepositoryLayout,
});
