import { createFileRoute } from "@tanstack/react-router";
import { Releases } from "@/pages/Releases";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/releases/")({
  staticData: { section: "releases" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["q", "state"]),
  component: () => <Releases tag="" />,
});
