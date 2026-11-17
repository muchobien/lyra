import { createFileRoute } from "@tanstack/react-router";
import { RunDetail } from "@/pages/RunDetail";
import { searchValues } from "@/routing/search";
import { numberParams } from "@/routing/paths";
import { getGetRunQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/runs/$number")({
  staticData: { section: "runs" },
  params: numberParams,
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["q", "job"]),
  remountDeps: ({ params }) => params,
  loader: async ({ context, params: { owner, repo, number } }) => {
    await loadQuery(context.queryClient, getGetRunQueryOptions(owner, repo, number));
  },
  component: () => <RunDetail number={Route.useParams().number} />,
});
