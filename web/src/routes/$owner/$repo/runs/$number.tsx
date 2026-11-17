import { createFileRoute } from "@tanstack/react-router";
import { RunDetail } from "@/pages/RunDetail";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/runs/$number")({
  staticData: { section: "runs" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["q", "job"]),
  remountDeps: ({ params }) => params,
  component: () => <RunDetail number={Route.useParams().number} />,
});
