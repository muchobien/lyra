import { createFileRoute } from "@tanstack/react-router";
import { Releases } from "@/pages/Releases";

export const Route = createFileRoute("/$owner/$repo/releases/tag/$")({
  staticData: { section: "releases" },
  remountDeps: ({ params }) => params,
  component: () => <Releases tag={Route.useParams()._splat || ""} />,
});
