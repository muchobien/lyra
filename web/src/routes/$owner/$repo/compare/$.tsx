import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/compare/$")({
  staticData: { section: "compare" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["file", "layout"]),
  head: ({ params }) => ({ meta: [{ title: `Compare · ${params.owner}/${params.repo} · Lyra` }] }),
  remountDeps: ({ params }) => params,
  component: () => <CodeScreens section="compare" target={Route.useParams()._splat || ""} />,
});
