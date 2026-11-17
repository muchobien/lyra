import { createFileRoute } from "@tanstack/react-router";
import { CodeScreens } from "@/pages/CodeScreens";
import { searchLimit, searchValues } from "@/routing/search";
import { loadCommits } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/commits/")({
  staticData: { section: "commits" },
  validateSearch: (search: Record<string, unknown>) => searchValues(search, ["limit"]),
  loaderDeps: ({ search }) => ({ limit: searchLimit(search.limit, 30, 500) }),
  loader: ({ context, params: { owner, repo }, deps }) =>
    loadCommits(context.queryClient, owner, repo, "", deps.limit),
  head: ({ params }) => ({ meta: [{ title: `Commits · ${params.owner}/${params.repo} · Lyra` }] }),
  component: () => <CodeScreens section="commits" target="" />,
});
