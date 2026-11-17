import { type QueryExecuteOptions, type QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import { APIError } from "@/api";
import { getGetRepoQueryOptions, getListCommitsQueryOptions } from "@/api/generated/lyra";

export async function loadQuery<Data, ErrorType, Key extends readonly unknown[]>(
  client: QueryClient,
  options: QueryExecuteOptions<Data, ErrorType, Data, Data, Key>,
) {
  try {
    return await client.query(options);
  } catch (error) {
    if (error instanceof APIError && error.status === 404) {
      throw notFound({ data: error.message });
    }
    throw error;
  }
}

export async function loadCommits(
  client: QueryClient,
  owner: string,
  repo: string,
  target: string,
  limit: number,
) {
  const metadata = await loadQuery(client, getGetRepoQueryOptions(owner, repo));
  if (metadata.branches.length) {
    await loadQuery(
      client,
      getListCommitsQueryOptions(owner, repo, {
        ref: target || metadata.default_branch || "main",
        limit,
      }),
    );
  }
}
