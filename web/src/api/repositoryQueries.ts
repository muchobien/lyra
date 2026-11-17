import { queryOptions, type QueryFilters } from "@tanstack/react-query";
import {
  getGetRepoQueryKey,
  getListEventsQueryKey,
  getListOwnersQueryKey,
  listOwners,
} from "./generated/lyra";

export const ownerListOptions = queryOptions({
  queryKey: getListOwnersQueryKey(),
  queryFn: ({ signal }) => listOwners({ signal }),
});

export function repositoryQueries(owner: string, repo: string): QueryFilters {
  const [repository] = getGetRepoQueryKey(owner, repo);
  const [events] = getListEventsQueryKey();
  const name = `${owner}/${repo}`;
  return {
    predicate: ({ queryKey: [path, params] }) =>
      typeof path === "string" &&
      (path === repository ||
        path.startsWith(`${repository}/`) ||
        (path === events &&
          !!params &&
          typeof params === "object" &&
          "repo" in params &&
          params.repo === name)),
  };
}
