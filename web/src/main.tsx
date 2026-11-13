import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRootRoute, createRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { App } from "./App";
import { RepositoryList } from "./RepositoryList";
import { RepositoryPage } from "./RepositoryPage";
import "./tokens.css";

const rootRoute = createRootRoute({ component: App });
type RepositorySearch = { ref?: string; path?: string; file?: string };
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: RepositoryList,
});
const ownerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/$owner",
  component: RepositoryList,
});
const repositoryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/$owner/$repo",
  component: RepositoryPage,
  validateSearch: (search: Record<string, unknown>) => {
    const value: RepositorySearch = {};
    if (typeof search.ref === "string") {
      value.ref = search.ref;
    }
    if (typeof search.path === "string") {
      value.path = search.path;
    }
    if (typeof search.file === "string") {
      value.file = search.file;
    }
    return value;
  },
});
const router = createRouter({
  routeTree: rootRoute.addChildren([indexRoute, ownerRoute, repositoryRoute]),
  scrollRestoration: true,
  defaultPreload: "intent",
});
const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

const container = document.getElementById("root");
if (!container) {
  throw new Error("Missing application root");
}
createRoot(container).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
);
