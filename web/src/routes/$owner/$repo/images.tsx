import { createFileRoute } from "@tanstack/react-router";
import { Containers } from "@/pages/Containers";
import { searchValues } from "@/routing/search";
import { getListImagesQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/images")({
  staticData: { section: "images" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["q"]),
  loader: async ({ context, params: { owner, repo } }) => {
    await loadQuery(context.queryClient, getListImagesQueryOptions(owner, repo));
  },
  head: ({ params }) => ({
    meta: [{ title: `Containers · ${params.owner}/${params.repo} · Lyra` }],
  }),
  component: Containers,
});
