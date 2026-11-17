import { createFileRoute } from "@tanstack/react-router";
import { ActionsList } from "@/pages/ActionsList";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/runs/")({
  staticData: { section: "runs" },
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["state", "q", "workflow", "ref", "limit"]),
  component: ActionsList,
});
