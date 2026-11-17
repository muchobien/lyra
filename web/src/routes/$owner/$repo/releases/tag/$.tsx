import { createFileRoute } from "@tanstack/react-router";
import { Releases } from "@/pages/Releases";
import { getGetReleaseByTagQueryOptions } from "@/api/generated/lyra";
import { loadQuery } from "@/routing/load";

export const Route = createFileRoute("/$owner/$repo/releases/tag/$")({
  staticData: { section: "releases" },
  remountDeps: ({ params }) => params,
  loader: async ({ context, params: { owner, repo, _splat } }) => {
    await loadQuery(context.queryClient, getGetReleaseByTagQueryOptions(owner, repo, _splat || ""));
  },
  component: () => <Releases tag={Route.useParams()._splat || ""} />,
});
