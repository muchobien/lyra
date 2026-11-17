import { createFileRoute, notFound } from "@tanstack/react-router";
import { RepositoryList } from "@/pages/RepositoryList";
import { ownerListOptions } from "@/api/repositoryQueries";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/")({
  loader: async ({ context, params }) => {
    const owners = await loadQuery(context.queryClient, ownerListOptions);
    if (!owners.some((owner) => owner.name === params.owner)) {
      throw notFound({ data: `Owner ${params.owner} not found.` });
    }
  },
  head: ({ params }) => ({ meta: [{ title: `${params.owner} · Lyra` }] }),
  component: () => <RepositoryList owner={Route.useParams().owner} />,
});
