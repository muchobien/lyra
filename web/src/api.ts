import { createContext } from "react";

export type Actor = { user: string; agent?: string; via: string; session?: string };
export type Identity = { actor: Actor; scope: string; repos: string[] };
export const IdentityContext = createContext<Identity | null>(null);
export const ConnectionContext = createContext("connecting");
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
  body: string;
  parents: string[];
  author: { name: string; date: string; email: string };
  committer: { name: string; date: string };
  signature: Signature;
};
export type Signature = {
  status: string;
  fingerprint?: string;
  key_user?: string;
  key_agent?: string;
};
export type Tag = {
  name: string;
  target: string;
  annotated: boolean;
  message?: string;
  tagger?: { name: string; date: string };
  signature: Signature;
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
export type Release = {
  id: number;
  tag: string;
  title: string;
  notes: string;
  draft: boolean;
  prerelease: boolean;
  target: string;
  commit: string;
  author: Actor;
  created_at: string;
  published_at: string | null;
  images?: Image[];
  assets: ReleaseAsset[];
};
export type ReleaseAsset = {
  id: number;
  name: string;
  size: number;
  sha256: string;
  content_type: string;
  download_url: string;
};
export type Issue = {
  number: number;
  repo: string;
  kind: "issue" | "pr";
  title: string;
  body: string;
  state: string;
  author: Actor;
  labels: string[];
  assignees: string[];
  comment_count: number;
  created_at: string;
  updated_at: string;
};
export type PullRequest = Issue & {
  base: string;
  head: string;
  head_sha: string;
  base_sha: string;
  merge_sha?: string;
};
export type DiscussionComment = {
  id: number;
  body: string;
  html: string;
  author: Actor;
  created_at: string;
  review_id?: number;
  path?: string;
  line?: number;
  side?: string;
  commit_sha?: string;
  outdated?: boolean;
};
export type Review = {
  id: number;
  state: string;
  body: string;
  html: string;
  author: Actor;
  created_at: string;
  commit_sha: string;
  comments: DiscussionComment[];
};
export type MergePlan = {
  head_sha: string;
  base_sha: string;
  fast_forward: boolean;
  conflicts: string[];
  methods: string[];
  blocked: string[];
};
export type Check = {
  run: number;
  workflow: string;
  event: string;
  job_id: number;
  name: string;
  status: string;
};
export type Checks = { sha: string; state: string; jobs: Check[] };
export type ItemWorkspace = {
  item: Issue;
  pull: PullRequest | null;
  html: string;
  comments: DiscussionComment[];
  reviews: Review[];
  merge: MergePlan | null;
  merge_error: string;
  checks: Checks | null;
  can_write: boolean;
  is_author: boolean;
};
export type DiffLine = {
  kind: string;
  old: number;
  new: number;
  text: string;
  html: string;
  comments: DiscussionComment[] | null;
};
export type DiffFile = {
  path: string;
  old_path: string;
  binary: boolean;
  lines: DiffLine[] | null;
};
export type CommitWorkspace = { commit: Commit; checks: Checks; files: DiffFile[] };
export type Label = { name: string; color: string; description: string };
export type Run = {
  id: number;
  number: number;
  workflow: string;
  name: string;
  title: string;
  event: string;
  ref: string;
  sha: string;
  status: string;
  pull?: number;
  head_ref?: string;
  base_ref?: string;
  author: Actor;
  created_at: string;
  started_at: string | null;
  stopped_at: string | null;
  jobs?: Job[];
};
export type Job = {
  id: number;
  key: string;
  name: string;
  status: string;
  attempt: number;
  needs: string[];
  runs_on: string[];
  steps: TaskStep[];
  started_at: string | null;
  stopped_at: string | null;
  task_id?: number;
};
export type TaskStep = {
  id: number;
  name: string;
  result: string;
  started_at?: string;
  stopped_at?: string;
  log_index: number;
  log_length: number;
};
export type WorkflowInput = {
  description: string;
  required: boolean;
  default: string;
  type: string;
  options?: string[];
};
export type Workflow = {
  path: string;
  name: string;
  dispatch: boolean;
  inputs: Record<string, WorkflowInput>;
  error?: string;
};
export type ActivityEvent = {
  id: number;
  at: string;
  action: string;
  repo?: string;
  item?: number;
  actor: Actor;
  data: Record<string, unknown>;
};
export type Rule = {
  id: number;
  pattern: string;
  require_signed: boolean;
  require_approval: boolean;
  require_checks: boolean;
  allow_force_push: boolean;
  allow_deletion: boolean;
  merge_methods: string[];
};
export type Variable = { name: string; value: string; user: string; updated_at: string };
export type Secret = Omit<Variable, "value">;
export type AccessToken = {
  id: number;
  name: string;
  prefix: string;
  scope: string;
  repos: string[];
  created_at: string;
  expires_at: string | null;
  last_used_at: string | null;
};
export type SigningKey = {
  id: number;
  title: string;
  agent: string;
  fingerprint: string;
  public_key: string;
  signing: boolean;
};
export type Runner = {
  id: number;
  name: string;
  uuid: string;
  labels: string[];
  version: string;
  online: boolean;
  ephemeral: boolean;
  last_seen_at: string | null;
};
export type RunnerToken = { id: number; name: string; prefix: string; created_at: string };

export class APIError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function request(path: string, signal?: AbortSignal, options?: RequestInit) {
  const response = await fetch(path, { ...options, signal, credentials: "same-origin" });
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    if (response.headers.get("Content-Type")?.includes("application/json")) {
      const failure: { error?: string } = await response.json();
      message = failure.error || message;
    }
    throw new APIError(response.status, message);
  }
  return response;
}

export async function api<Value>(
  path: string,
  signal?: AbortSignal,
  options?: RequestInit,
): Promise<Value> {
  return (await request(path, signal, options)).json();
}

export function write<Value>(path: string, method: string, body?: unknown) {
  return api<Value>(path, undefined, {
    method,
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

export function repositoryAPI(name: string, resource = "") {
  return `/api/repos/${name.split("/").map(encodeURIComponent).join("/")}${resource}`;
}
