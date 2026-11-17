import { createFileRoute } from "@tanstack/react-router";
import { ItemDetail } from "@/pages/ItemDetail";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/issues/$number")({
  staticData: { section: "issues" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["tab", "file", "layout"]),
  remountDeps: ({ params }) => params,
  component: () => <ItemDetail kind="issues" number={Route.useParams().number} />,
});
