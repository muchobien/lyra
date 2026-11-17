import { createFileRoute } from "@tanstack/react-router";
import { Releases } from "@/pages/Releases";
import { searchValues } from "@/routing/search";
import { getListReleasesQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/releases/")({
  staticData: { section: "releases" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["q", "state"]),
  loader: async ({ context, params: { owner, repo } }) => {
    await loadQuery(context.queryClient, getListReleasesQueryOptions(owner, repo));
  },
  head: ({ params }) => ({ meta: [{ title: `Releases · ${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <Releases tag="" />,
});
