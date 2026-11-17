import { createFileRoute } from "@tanstack/react-router";
import { Settings } from "@/pages/Settings";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/settings")({
  staticData: { section: "settings" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["section"]),
  component: () => <Settings scope="repository" />,
});
