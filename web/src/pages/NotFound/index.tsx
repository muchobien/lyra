import { type NotFoundRouteProps } from "@tanstack/react-router";
import "@/styles/workspace.css";

export function NotFound({ data }: NotFoundRouteProps) {
  return (
    <div className="lyra-workspace-panel">
      <p className="lyra-workspace-empty">{typeof data === "string" ? data : "Page not found."}</p>
    </div>
  );
}
