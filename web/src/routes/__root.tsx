import { createRootRoute } from "@tanstack/react-router";
import { App } from "@/pages/App";
import { NotFound } from "@/pages/NotFound";

export const Route = createRootRoute({
  component: App,
  notFoundComponent: NotFound,
});
