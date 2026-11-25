import { type QueryExecuteOptions, type QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import { APIError } from "@/api";
import {
  getGetRepoQueryOptions,
  getGetRepositoryBlobQueryOptions,
  getGetRepositoryBlameQueryOptions,
  getGetRepositoryTreeQueryOptions,
  getListCommitsQueryOptions,
  getListTagsQueryOptions,
} from "@/api/generated/lyra";
import { splitSource } from "@/routing/paths";

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

export async function loadSource(
  client: QueryClient,
  owner: string,
  repo: string,
  view: "tree" | "blob" | "blame",
  target: string,
) {
  const [metadata, tags] = await Promise.all([
    loadQuery(client, getGetRepoQueryOptions(owner, repo)),
    loadQuery(client, getListTagsQueryOptions(owner, repo)),
  ]);
  const source = splitSource(
    target,
    metadata.branches.map((branch) => branch.name),
    tags.map((tag) => tag.name),
  );
  const params = { ref: source.ref || metadata.default_branch || "main", path: source.path };
  if (view === "blob") {
    await loadQuery(client, getGetRepositoryBlobQueryOptions(owner, repo, params));
  } else if (view === "blame") {
    await loadQuery(client, getGetRepositoryBlameQueryOptions(owner, repo, params));
  } else {
    await loadQuery(client, getGetRepositoryTreeQueryOptions(owner, repo, params));
  }
}
