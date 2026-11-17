import { useSuspenseQuery } from "@tanstack/react-query";
import { ownerListOptions } from "@/api/repositoryQueries";

export function useConnect(owner?: string) {
  const { data, error, refetch } = useSuspenseQuery({
    ...ownerListOptions,
    select: (groups) => (owner ? groups.filter((group) => group.name === owner) : groups),
  });

  function retry() {
    void refetch();
  }

  return { groups: data, error, retry };
}
