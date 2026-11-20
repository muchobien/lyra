import { createFileRoute } from "@tanstack/react-router";
import { RepositoryPage } from "@/pages/RepositoryPage";
import { loadSource } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/")({
  loader: ({ context: { queryClient }, params: { owner, repo } }) =>
    loadSource(queryClient, owner, repo, "tree", ""),
  head: ({ params }) => ({ meta: [{ title: `${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <RepositoryPage view="tree" target="" />,
});
