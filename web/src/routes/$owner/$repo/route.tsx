import { createFileRoute } from "@tanstack/react-router";
import { RepositoryLayout } from "@/pages/RepositoryLayout";

export const Route = createFileRoute("/$owner/$repo")({
  remountDeps: ({ params }) => ({ owner: params.owner, repo: params.repo }),
  component: RepositoryLayout,
});
