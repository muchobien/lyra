import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";

export const Route = createFileRoute("/$owner/$repo/compare/")({
  staticData: { section: "compare" },
  head: ({ params }) => ({ meta: [{ title: `Compare · ${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <CodeScreens section="compare" target="" />,
});
