import { createFileRoute } from "@tanstack/react-router";
import { RepositoryPage } from "@/pages/RepositoryPage";
import { loadSource } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/tree/$")({
  loader: ({ context: { queryClient }, params: { owner, repo, _splat } }) =>
    loadSource(queryClient, owner, repo, "tree", _splat || ""),
  head: ({ params }) => ({
    meta: [{ title: `${params._splat} · ${params.owner}/${params.repo} · Lyra` }],
  }),
  component: () => <RepositoryPage view="tree" target={Route.useParams()._splat || ""} />,
});
