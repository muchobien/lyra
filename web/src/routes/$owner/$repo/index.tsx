import { createFileRoute } from "@tanstack/react-router";
import { RepositoryPage } from "@/pages/RepositoryPage";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/")({
  validateSearch: (search: Record<string, unknown>) =>
    searchValues(search, ["ref", "path", "file"]),
  component: RepositoryPage,
});
