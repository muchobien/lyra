import { createFileRoute } from "@tanstack/react-router";
import { RepositoryPage } from "@/pages/RepositoryPage";
import { loadSource } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/blob/$")({
  loader: ({ context: { queryClient }, params: { owner, repo, _splat } }) =>
    loadSource(queryClient, owner, repo, "blob", _splat || ""),
  head: ({ params }) => ({
    meta: [{ title: `${params._splat} · ${params.owner}/${params.repo} · Lyra` }],
  }),
  component: () => <RepositoryPage view="blob" target={Route.useParams()._splat || ""} />,
});
