import { createFileRoute } from "@tanstack/react-router";
import { RepositoryList } from "@/pages/RepositoryList";
import { ownerListOptions } from "@/api/repositoryQueries";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/")({
  loader: async ({ context }) => {
    await loadQuery(context.queryClient, ownerListOptions);
  },
  head: () => ({ meta: [{ title: "Repositories · Lyra" }] }),
  component: () => <RepositoryList />,
});
