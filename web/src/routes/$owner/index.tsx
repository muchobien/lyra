import { createFileRoute } from "@tanstack/react-router";
import { RepositoryList } from "@/pages/RepositoryList";

export const Route = createFileRoute("/$owner/")({
  component: RepositoryList,
});
