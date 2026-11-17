import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/commit/$sha")({
  staticData: { section: "commit" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["file", "layout"]),
  remountDeps: ({ params }) => params,
  component: () => <CodeScreens section="commit" target={Route.useParams().sha} />,
});
