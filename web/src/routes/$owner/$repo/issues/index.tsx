import { createFileRoute } from "@tanstack/react-router";
import { ItemList } from "@/pages/ItemList";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/issues/")({
  staticData: { section: "issues" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["state", "q", "label", "assignee", "limit"]),
  component: () => <ItemList kind="issues" />,
});
