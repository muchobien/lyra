import { createFileRoute } from "@tanstack/react-router";
import { RepositoryPage } from "@/pages/RepositoryPage";
import { loadSource } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/blame/$")({
  loader: ({ context: { queryClient }, params: { owner, repo, _splat } }) =>
    loadSource(queryClient, owner, repo, "blame", _splat || ""),
  head: ({ params }) => ({
    meta: [{ title: `Blame · ${params._splat} · ${params.owner}/${params.repo} · Lyra` }],
  }),
  component: () => <RepositoryPage view="blame" target={Route.useParams()._splat || ""} />,
});
