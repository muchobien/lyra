import { createFileRoute } from "@tanstack/react-router";
import { ItemDetail } from "@/pages/ItemDetail";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/pulls/$number")({
  staticData: { section: "pulls" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["tab", "file", "layout"]),
  remountDeps: ({ params }) => params,
  component: () => <ItemDetail kind="pulls" number={Route.useParams().number} />,
});
