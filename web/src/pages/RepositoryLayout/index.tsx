import { Outlet } from "@tanstack/react-router";
import { ConnectionContext } from "@/api";
import { useConnect } from "./connect";

export function RepositoryLayout() {
  const { connection } = useConnect();
  return (
    <ConnectionContext value={connection}>
      <Outlet />
    </ConnectionContext>
  );
}
