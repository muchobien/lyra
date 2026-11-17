import { useQuery } from "@tanstack/react-query";
import { APIError } from "./transport";
import { getLatestRelease, getGetLatestReleaseQueryKey } from "./generated/lyra";

export function useLatestRelease(owner: string, repo: string) {
  return useQuery({
    queryKey: getGetLatestReleaseQueryKey(owner, repo),
    queryFn: async ({ signal }) => {
      try {
        return await getLatestRelease(owner, repo, { signal });
      } catch (error) {
        if (error instanceof APIError && error.status === 404) {
          return null;
        }
        throw error;
      }
    },
  });
}
