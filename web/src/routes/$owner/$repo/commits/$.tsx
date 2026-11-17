import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/commits/$")({
  staticData: { section: "commits" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["limit"]),
  remountDeps: ({ params }) => params,
  component: () => <CodeScreens section="commits" target={Route.useParams()._splat || ""} />,
});
