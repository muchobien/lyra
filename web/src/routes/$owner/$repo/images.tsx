import { createFileRoute } from "@tanstack/react-router";
import { Containers } from "@/pages/Containers";
import { searchValues } from "@/routing/search";

export const Route = createFileRoute("/$owner/$repo/images")({
  staticData: { section: "images" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["q"]),
  component: Containers,
});
