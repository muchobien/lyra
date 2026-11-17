import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useListOwners } from "@/api/generated/lyra";
import { useParams } from "@tanstack/react-router";

export function useConnect() {
  const { owner } = useParams({ strict: false });
  const owners = useListOwners();
  useDocumentTitle(`${owner || "Repositories"}`);
  function retry() {
    void owners.refetch();
  }

  return { owner, owners, retry };
}
