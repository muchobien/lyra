import { createFileRoute } from "@tanstack/react-router";
import { Activity } from "@/pages/Activity";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/activity")({
  staticData: { section: "activity" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["q", "action", "limit"]),
  component: Activity,
});
