import { createRootRouteWithContext } from "@tanstack/react-router";
import { type QueryClient } from "@tanstack/react-query";
import { getGetMeQueryOptions, getMe } from "@/api/generated/lyra";
import { App, initialPageError } from "@/pages/App";
import { NotFound } from "@/pages/NotFound";
import { stripEmptySearch } from "@/routing/search";

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  shellComponent: App,
  pendingMs: 0,
  pendingMinMs: 0,
  pendingComponent: () => null,
  beforeLoad: async ({ context, location }) => {
    if (initialPageError && initialPageError.pathname === location.pathname) {
      throw initialPageError.error;
    }
    await context.queryClient.query({
      ...getGetMeQueryOptions(),
      queryFn: () => getMe(),
      staleTime: "static",
    });
  },
  head: ({ matches }) => ({
    meta: matches.length === 1 ? [{ title: "Lyra" }] : [],
    links: [{ rel: "stylesheet", href: "/api/code.css" }],
  }),
  search: { middlewares: [stripEmptySearch] },
  notFoundComponent: NotFound,
});
