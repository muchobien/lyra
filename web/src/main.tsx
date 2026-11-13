import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRootRoute, createRoute, createRouter, RouterProvider } from "@tanstack/react-router";
import { App } from "./App";
import { RepositoryList } from "./RepositoryList";
import { RepositoryPage } from "./RepositoryPage";
import { RepositoryScreen } from "./RepositoryScreen";
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
type ScreenSearch = {
  state?: string;
  q?: string;
  label?: string;
  assignee?: string;
  tab?: string;
  file?: string;
  layout?: string;
  job?: string;
  workflow?: string;
  ref?: string;
  section?: string;
  limit?: string;
  action?: string;
};
function validateScreenSearch(search: Record<string, unknown>) {
  const parsed: ScreenSearch = {};
  for (const key of [
    "state",
    "q",
    "label",
    "assignee",
    "tab",
    "file",
    "layout",
    "job",
    "workflow",
    "ref",
    "section",
    "limit",
    "action",
  ] as const) {
    if (typeof search[key] === "string") {
      parsed[key] = search[key];
    }
  }
  return parsed;
}
const sectionRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/$owner/$repo/$section",
  component: RepositoryScreen,
  validateSearch: validateScreenSearch,
});
const detailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/$owner/$repo/$section/$",
  component: RepositoryScreen,
  validateSearch: validateScreenSearch,
});
const router = createRouter({
  routeTree: rootRoute.addChildren([
    indexRoute,
    ownerRoute,
    repositoryRoute,
    sectionRoute,
    detailRoute,
  ]),
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
