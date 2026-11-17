import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";

export const Route = createFileRoute("/$owner/$repo/tags")({
  staticData: { section: "tags" },
  component: () => <CodeScreens section="tags" target="" />,
});
