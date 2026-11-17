import { QueryClient } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { APIError } from "./api";
import { RouteError, RoutePending } from "./routing/boundaries";
import { NotFound } from "./pages/NotFound";
import { routeTree } from "./routeTree.gen";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: (failures, error) =>
        failures < 1 &&
        !(error instanceof APIError && [400, 401, 403, 404, 410, 422].includes(error.status)),
    },
  },
});

export const router = createRouter({
  routeTree,
  context: { queryClient },
  parseSearch: (search) => Object.fromEntries(new URLSearchParams(search)),
  stringifySearch: (search) => {
    const params = new URLSearchParams();
    for (const [key, value] of Object.entries(search)) {
      if (typeof value === "string") {
        params.set(key, value);
      }
    }
    const query = params.toString();
    return query ? `?${query}` : "";
  },
  scrollRestoration: true,
  defaultPreload: "intent",
  defaultPreloadStaleTime: 0,
  defaultStructuralSharing: true,
  defaultPendingMs: 250,
  defaultPendingMinMs: 150,
  defaultPendingComponent: RoutePending,
  defaultErrorComponent: RouteError,
  defaultNotFoundComponent: NotFound,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
  interface StaticDataRouteOption {
    section?: string;
  }
}
