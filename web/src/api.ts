export type Actor = { user: string; agent: string; via: string; session: string };
export type Identity = { actor: Actor; scope: string; repos: string[] };
export type Repository = {
  name: string;
  description: string;
  default_branch: string;
  branches: { name: string; sha: string }[];
  clone_url: string;
};
export type Owner = { name: string; description: string; repos: Repository[] };
export type Commit = {
  sha: string;
  subject: string;
  committer: { name: string; date: string };
  signature: { status: string; key_user?: string; key_agent?: string };
};
export type Tree = {
  ref: string;
  sha: string;
  path: string;
  empty: boolean;
  entries: { name: string; type: string; size: number }[];
  last: Commit | null;
  readme: string;
};
export type Blob = { path: string; size: number; code: string; binary: boolean };
export type Image = {
  name: string;
  tag: string;
  reference: string;
  digest: string;
  pushed_at: string;
};
export type Release = { tag: string; title: string; published_at: string; images: Image[] };

export class APIError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function api<Value>(path: string, signal?: AbortSignal): Promise<Value> {
  const response = await fetch(path, { signal, credentials: "same-origin" });
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    if (response.headers.get("Content-Type")?.includes("application/json")) {
      const failure: { error?: string } = await response.json();
      message = failure.error || message;
    }
    throw new APIError(response.status, message);
  }
  return response.json();
}

export function repositoryAPI(name: string, resource = "") {
  return `/api/repos/${name.split("/").map(encodeURIComponent).join("/")}${resource}`;
}
