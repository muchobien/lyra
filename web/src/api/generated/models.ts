export interface ReadOnlyRequest {
  enabled: boolean;
}

export interface LastBackup {
  at: string;
  bytes: number;
  objects: number;
  pause_ms: number;
  repos: number;
}

export interface StatusView {
  /** null until the first backup. */
  last_backup: LastBackup;
  read_only: boolean;
}

export interface ErrorBody {
  error: string;
}

export interface RegistryCollected {
  blobs_after: number;
  blobs_before: number;
  bytes_freed: number;
  ms: number;
}

export interface StoredObject {
  kind: string;
  name: string;
  path: string;
  repo: string;
  sha256?: string;
  size?: number;
}

export interface StorageCheck {
  checked: number;
  missing: StoredObject[];
}

export interface Actor {
  agent?: string;
  session?: string;
  user: string;
  via: string;
}

export interface Event {
  action: string;
  actor: Actor;
  at: string;
  data: unknown;
  id: number;
  item?: number;
  repo?: string;
}

export interface Key {
  agent: string;
  created_at: string;
  fingerprint: string;
  id: number;
  public_key: string;
  signing: boolean;
  title: string;
  user: string;
}

export interface CreateKeyRequest {
  agent?: string;
  /** One OpenSSH public key line */
  public_key: string;
  /** Must be true: only signing keys are supported */
  signing: boolean;
  title?: string;
}

export interface DeletedID {
  deleted: number;
}

export interface MarkdownRequest {
  body: string;
}

export interface RenderedMarkdown {
  html: string;
}

export interface MeView {
  actor: Actor;
  repos: string[];
  scope: string;
}

export interface Repo {
  created_at: string;
  created_by: string;
  description: string;
  name: string;
}

export interface OwnerGroup {
  created_at: string;
  description: string;
  name: string;
  repos: Repo[];
  user: string;
}

export interface CreateOwnerRequest {
  description?: string;
  name: string;
}

export interface OwnerWrite {
  created_at: string;
  description: string;
  event_id: number;
  name: string;
  user: string;
}

export interface DeletedWithEvent {
  deleted: string;
  event_id: number;
}

export interface Ref {
  name: string;
  sha: string;
}

export interface RepoView {
  branches: Ref[];
  clone_url: string;
  created_at: string;
  created_by: string;
  default_branch: string;
  description: string;
  name: string;
}

export interface CreateRepoRequest {
  description?: string;
  /** owner/name */
  name: string;
}

export interface DeletedName {
  deleted: string;
}

export interface BlobResponse {
  binary: boolean;
  code: string;
  path: string;
  size: number;
}

export interface Person {
  date: string;
  email: string;
  name: string;
}

export interface Signature {
  fingerprint?: string;
  key_agent?: string;
  key_id?: number;
  key_user?: string;
  status: string;
}

export interface Commit {
  author: Person;
  body: string;
  committer: Person;
  parents: string[];
  sha: string;
  signature: Signature;
  subject: string;
}

export interface Check {
  event: string;
  job_id: number;
  name: string;
  run: number;
  status: string;
  workflow: string;
}

export interface Checks {
  jobs: Check[];
  sha: string;
  state: string;
}

export interface Comment {
  author: Actor;
  body: string;
  commit_sha?: string;
  created_at: string;
  id: number;
  line?: number;
  outdated?: boolean;
  path?: string;
  review_id?: number;
  side?: string;
}

export interface DiffLine {
  comments: Comment[];
  html: string;
  kind: string;
  new: number;
  old: number;
  text: string;
}

export interface DiffFile {
  binary: boolean;
  lines: DiffLine[];
  old_path: string;
  path: string;
}

export interface CommitWorkspace {
  checks: Checks;
  commit: Commit;
  files: DiffFile[];
}

export interface Image {
  digest: string;
  media_type: string;
  name: string;
  pushed_at: string;
  pusher: Actor;
  reference?: string;
  repo: string;
  tag: string;
}

export interface Item {
  assignees: string[];
  author: Actor;
  body: string;
  /** @nullable */
  closed_at: string | null;
  comment_count: number;
  created_at: string;
  kind: string;
  labels: string[];
  number: number;
  repo: string;
  state: string;
  title: string;
  updated_at: string;
}

export interface CreateIssueRequest {
  /** Logins or agent names; @me is the caller. */
  assignees?: string[];
  body?: string;
  /** Labels must exist. */
  labels?: string[];
  title: string;
}

export interface ItemWrite {
  assignees: string[];
  author: Actor;
  body: string;
  /** @nullable */
  closed_at: string | null;
  comment_count: number;
  created_at: string;
  event_id?: number;
  kind: string;
  labels: string[];
  number: number;
  repo: string;
  state: string;
  title: string;
  updated_at: string;
}

export interface ItemView {
  assignees: string[];
  author: Actor;
  body: string;
  /** @nullable */
  closed_at: string | null;
  comment_count: number;
  comments: Comment[];
  created_at: string;
  kind: string;
  labels: string[];
  number: number;
  repo: string;
  state: string;
  title: string;
  updated_at: string;
}

/**
 * @nullable
 */
export type EditItemRequestState = typeof EditItemRequestState[keyof typeof EditItemRequestState] | null;


export const EditItemRequestState = {
  open: 'open',
  closed: 'closed',
} as const;

export interface EditItemRequest {
  add_assignees?: string[];
  add_labels?: string[];
  /** @nullable */
  body?: string | null;
  remove_assignees?: string[];
  remove_labels?: string[];
  /** @nullable */
  state?: EditItemRequestState;
  /** @nullable */
  title?: string | null;
}

/**
 * new is the head version, old the base version.
 */
export type CommentRequestSide = typeof CommentRequestSide[keyof typeof CommentRequestSide];


export const CommentRequestSide = {
  new: 'new',
  old: 'old',
} as const;

export interface CommentRequest {
  body: string;
  /** Reject a line comment if the pull request changed from this head. */
  head_sha?: string;
  /** Pull requests only: a line of path in the diff. */
  line?: number;
  /** Pull requests only: a file in the diff. */
  path?: string;
  /** new is the head version, old the base version. */
  side?: CommentRequestSide;
}

export interface CommentWrite {
  author: Actor;
  body: string;
  commit_sha?: string;
  created_at: string;
  event_id: number;
  id: number;
  line?: number;
  outdated?: boolean;
  path?: string;
  review_id?: number;
  side?: string;
}

export interface RenderedComment {
  author: Actor;
  body: string;
  commit_sha?: string;
  created_at: string;
  html: string;
  id: number;
  line?: number;
  outdated?: boolean;
  path?: string;
  review_id?: number;
  side?: string;
}

export interface MergePlan {
  base_sha: string;
  blocked: string[];
  conflicts: string[];
  fast_forward: boolean;
  head_sha: string;
  methods: string[];
}

export interface Pull {
  assignees: string[];
  author: Actor;
  base: string;
  base_sha: string;
  body: string;
  /** @nullable */
  closed_at: string | null;
  comment_count: number;
  created_at: string;
  head: string;
  head_sha: string;
  kind: string;
  labels: string[];
  merge_sha?: string;
  /** @nullable */
  merged_at: string | null;
  number: number;
  repo: string;
  state: string;
  title: string;
  updated_at: string;
}

export interface RenderedReview {
  author: Actor;
  body: string;
  comments: Comment[];
  commit_sha: string;
  created_at: string;
  html: string;
  id: number;
  state: string;
}

export interface ItemWorkspace {
  can_write: boolean;
  checks: Checks | null;
  comments: RenderedComment[];
  html: string;
  is_author: boolean;
  item: Item;
  merge: MergePlan | null;
  merge_error: string;
  pull: Pull | null;
  reviews: RenderedReview[];
}

export interface Label {
  color: string;
  description: string;
  id: number;
  name: string;
  repo: string;
}

export interface CreateLabelRequest {
  /** Hex without #, such as d73a4a. */
  color?: string;
  description?: string;
  name: string;
}

export interface LabelWrite {
  color: string;
  description: string;
  event_id: number;
  id: number;
  name: string;
  repo: string;
}

export interface CreatePullRequest {
  assignees?: string[];
  /** Defaults to the default branch. */
  base?: string;
  body?: string;
  /** The branch to merge. */
  head: string;
  labels?: string[];
  title: string;
}

export interface PullWrite {
  actions?: string[];
  assignees: string[];
  author: Actor;
  base: string;
  base_sha: string;
  body: string;
  /** @nullable */
  closed_at: string | null;
  comment_count: number;
  created_at: string;
  event_id?: number;
  head: string;
  head_sha: string;
  kind: string;
  labels: string[];
  merge_sha?: string;
  /** @nullable */
  merged_at: string | null;
  number: number;
  repo: string;
  state: string;
  title: string;
  updated_at: string;
}

export interface Review {
  author: Actor;
  body: string;
  comments: Comment[];
  commit_sha: string;
  created_at: string;
  id: number;
  state: string;
}

export interface PullView {
  assignees: string[];
  author: Actor;
  base: string;
  base_sha: string;
  body: string;
  checks: Checks;
  /** @nullable */
  closed_at: string | null;
  comment_count: number;
  comments: Comment[];
  created_at: string;
  head: string;
  head_sha: string;
  kind: string;
  labels: string[];
  merge?: MergePlan;
  merge_error?: string;
  merge_sha?: string;
  /** @nullable */
  merged_at: string | null;
  number: number;
  repo: string;
  reviews: Review[];
  state: string;
  title: string;
  updated_at: string;
}

export interface FileChange {
  additions: number;
  binary?: boolean;
  deletions: number;
  old_path?: string;
  path: string;
  status: string;
}

export type MergeRequestMethod = typeof MergeRequestMethod[keyof typeof MergeRequestMethod];


export const MergeRequestMethod = {
  squash: 'squash',
  ff: 'ff',
} as const;

export interface MergeRequest {
  delete_branch?: boolean;
  /** Refuse the merge unless the head branch is at this commit. */
  head_sha?: string;
  /** Squash commit body. */
  message?: string;
  method: MergeRequestMethod;
  /** Squash commit title; defaults to the title with (#number). */
  title?: string;
}

export type ReviewRequestState = typeof ReviewRequestState[keyof typeof ReviewRequestState];


export const ReviewRequestState = {
  approve: 'approve',
  request_changes: 'request_changes',
  comment: 'comment',
} as const;

export interface LineComment {
  body: string;
  commit_sha?: string;
  line?: number;
  path?: string;
  side?: string;
}

export interface ReviewRequest {
  body?: string;
  /** Line comments on lines in the diff. */
  comments?: LineComment[];
  /** Reject a review of a changed head. */
  head_sha?: string;
  state: ReviewRequestState;
}

export interface ReviewWrite {
  author: Actor;
  body: string;
  comments: Comment[];
  commit_sha: string;
  created_at: string;
  event_id: number;
  id: number;
  state: string;
}

export interface Asset {
  content_type: string;
  created_at: string;
  download_url?: string;
  id: number;
  name: string;
  sha256: string;
  size: number;
  uploader: Actor;
}

export interface Release {
  assets: Asset[];
  author: Actor;
  commit: string;
  created_at: string;
  draft: boolean;
  id: number;
  images?: Image[];
  notes: string;
  prerelease: boolean;
  /** @nullable */
  published_at: string | null;
  repo: string;
  tag: string;
  target: string;
  title: string;
}

export interface CreateReleaseRequest {
  draft?: boolean;
  /** List the pull requests merged since the last release. */
  generate_notes?: boolean;
  notes?: string;
  prerelease?: boolean;
  tag: string;
  /** Branch or commit to tag; defaults to the default branch. */
  target?: string;
  /** Defaults to the tag. */
  title?: string;
}

export interface ReleaseWrite {
  actions?: string[];
  assets: Asset[];
  author: Actor;
  commit: string;
  created_at: string;
  draft: boolean;
  event_id: number;
  id: number;
  images?: Image[];
  notes: string;
  prerelease: boolean;
  /** @nullable */
  published_at: string | null;
  repo: string;
  tag: string;
  target: string;
  title: string;
}

export interface NotePull {
  author: string;
  number: number;
  title: string;
}

export interface ReleaseNotes {
  notes: string;
  previous_tag: string;
  pulls: NotePull[];
}

export interface ReleaseDeleted {
  deleted: string;
  event_id: number;
  tag_deleted: boolean;
}

export interface EditReleaseRequest {
  /**
     * false publishes a draft.
     * @nullable
     */
  draft?: boolean | null;
  /** @nullable */
  notes?: string | null;
  /** @nullable */
  prerelease?: boolean | null;
  /**
     * Drafts only.
     * @nullable
     */
  target?: string | null;
  /** @nullable */
  title?: string | null;
}

export interface AssetWrite {
  content_type: string;
  created_at: string;
  download_url?: string;
  event_id: number;
  id: number;
  name: string;
  sha256: string;
  size: number;
  uploader: Actor;
}

export interface Rule {
  allow_deletion: boolean;
  allow_force_push: boolean;
  created_at: string;
  id: number;
  merge_methods: string[];
  pattern: string;
  repo: string;
  require_approval: boolean;
  require_checks: boolean;
  require_signed: boolean;
}

export type SetRuleRequestMergeMethodsItem = typeof SetRuleRequestMergeMethodsItem[keyof typeof SetRuleRequestMergeMethodsItem];


export const SetRuleRequestMergeMethodsItem = {
  ff: 'ff',
  squash: 'squash',
} as const;

export interface SetRuleRequest {
  allow_deletion?: boolean;
  allow_force_push?: boolean;
  /** Defaults to both. */
  merge_methods?: SetRuleRequestMergeMethodsItem[];
  /** A branch name or glob, such as main or release/*. */
  pattern: string;
  require_approval?: boolean;
  require_checks?: boolean;
  require_signed?: boolean;
}

export interface RuleWrite {
  allow_deletion: boolean;
  allow_force_push: boolean;
  created_at: string;
  event_id: number;
  id: number;
  merge_methods: string[];
  pattern: string;
  repo: string;
  require_approval: boolean;
  require_checks: boolean;
  require_signed: boolean;
}

export interface TaskStep {
  id: number;
  log_index: number;
  log_length: number;
  name: string;
  result: string;
  started_at?: string;
  stopped_at?: string;
}

export type JobOutputs = {[key: string]: string};

export interface Job {
  attempt: number;
  id: number;
  key: string;
  name: string;
  needs: string[];
  outputs: JobOutputs;
  runs_on: string[];
  /** @nullable */
  started_at: string | null;
  status: string;
  steps: TaskStep[];
  /** @nullable */
  stopped_at: string | null;
  task_id?: number;
}

export interface Run {
  author: Actor;
  base_ref?: string;
  created_at: string;
  event: string;
  head_ref?: string;
  id: number;
  jobs?: Job[];
  name: string;
  number: number;
  pull?: number;
  ref: string;
  repo: string;
  sha: string;
  /** @nullable */
  started_at: string | null;
  status: string;
  /** @nullable */
  stopped_at: string | null;
  title: string;
  workflow: string;
}

export interface RunCancelled {
  cancelled: number;
  event_id: number;
}

export interface RunWrite {
  author: Actor;
  base_ref?: string;
  created_at: string;
  event: string;
  event_id: number;
  head_ref?: string;
  id: number;
  jobs?: Job[];
  name: string;
  number: number;
  pull?: number;
  ref: string;
  repo: string;
  sha: string;
  /** @nullable */
  started_at: string | null;
  status: string;
  /** @nullable */
  stopped_at: string | null;
  title: string;
  workflow: string;
}

export interface Secret {
  name: string;
  updated_at: string;
  user: string;
}

export interface ValueRequest {
  value: string;
}

export interface SecretSet {
  event_id: number;
  name: string;
}

export interface Tag {
  annotated: boolean;
  message?: string;
  name: string;
  signature: Signature;
  tagger?: Person;
  target: string;
}

export interface Entry {
  mode: string;
  name: string;
  sha: string;
  size: number;
  type: string;
}

export interface TreeResponse {
  empty: boolean;
  entries: Entry[];
  last: Commit | null;
  path: string;
  readme: string;
  ref: string;
  sha: string;
}

export interface Variable {
  name: string;
  updated_at: string;
  user: string;
  value: string;
}

export interface VariableSet {
  event_id: number;
  name: string;
  value: string;
}

export interface WorkflowInput {
  default: string;
  description: string;
  options: string[];
  required: boolean;
  type: string;
}

export type WorkflowWorkspaceInputs = {[key: string]: WorkflowInput};

export interface WorkflowWorkspace {
  dispatch: boolean;
  error?: string;
  inputs: WorkflowWorkspaceInputs;
  name: string;
  path: string;
}

export type DispatchRequestInputs = {[key: string]: string};

export interface DispatchRequest {
  inputs?: DispatchRequestInputs;
  /** Defaults to the default branch. */
  ref?: string;
  /** The workflow file name. */
  workflow: string;
}

export interface RunnerView {
  created_at: string;
  ephemeral: boolean;
  id: number;
  labels: string[];
  /** @nullable */
  last_seen_at: string | null;
  name: string;
  online: boolean;
  user: string;
  uuid: string;
  version: string;
}

export interface NameRequest {
  name: string;
}

export interface RunnerCreated {
  created_at: string;
  ephemeral: boolean;
  id: number;
  labels: string[];
  /** @nullable */
  last_seen_at: string | null;
  name: string;
  token: string;
  url: string;
  user: string;
  uuid: string;
  version: string;
}

export interface RunnerToken {
  created_at: string;
  id: number;
  name: string;
  prefix: string;
  user: string;
}

export interface RunnerTokenCreated {
  created_at: string;
  event_id: number;
  id: number;
  name: string;
  prefix: string;
  secret: string;
  url: string;
  user: string;
}

export interface DeletedIDWithEvent {
  deleted: number;
  event_id: number;
}

export interface SigningKeyView {
  allowed_signers: string;
  fingerprint: string;
  public_key: string;
}

export interface Token {
  created_at: string;
  /** @nullable */
  expires_at: string | null;
  id: number;
  /** @nullable */
  last_used_at: string | null;
  name: string;
  prefix: string;
  repos: string[];
  scope: string;
  user: string;
}

export type CreateTokenRequestScope = typeof CreateTokenRequestScope[keyof typeof CreateTokenRequestScope];


export const CreateTokenRequestScope = {
  read: 'read',
  write: 'write',
  admin: 'admin',
} as const;

export interface CreateTokenRequest {
  /** A Go duration, such as 720h */
  expires_in?: string;
  name: string;
  /** owner/repo or owner/*; empty means unrestricted */
  repos?: string[];
  scope: CreateTokenRequestScope;
}

export interface TokenCreated {
  secret: string;
  token: Token;
}

export interface RevokedID {
  revoked: number;
}

export type ListEventsParams = {
  repo?: string;
  /** 1 to 1000; defaults to 50. */
  limit?: number;
};

export type GetRepositoryBlobParams = {
  /** Branch, tag, or commit; defaults to the repository's default branch. */
  ref?: string;
  /** Path relative to the repository root. */
  path?: string;
};

export type ListCommitsParams = {
  /** Defaults to the default branch. */
  ref?: string;
  /** 1 to 500; defaults to 30. */
  limit?: number;
};

export type ListRepositoryFilesParams = {
  /** Branch, tag, or commit; defaults to the repository's default branch. */
  ref?: string;
  /** Path relative to the repository root. */
  path?: string;
};

export type ListIssuesParams = {
  /** open (default), closed or all; merged too for pull requests. */
  state?: string;
  /** Items carrying every one of these labels. */
  label?: string[];
  /** Items assigned to every one of these names; @me is the caller. */
  assignee?: string[];
  /** Defaults to 100. */
  limit?: number;
  /** Search the title or an exact issue/pull request number. */
  q?: string;
};

export type ListPullsParams = {
  /** open (default), closed or all; merged too for pull requests. */
  state?: string;
  /** Items carrying every one of these labels. */
  label?: string[];
  /** Items assigned to every one of these names; @me is the caller. */
  assignee?: string[];
  /** Defaults to 100. */
  limit?: number;
  /** Search the title or an exact issue/pull request number. */
  q?: string;
};

export type GetRepositoryRawFileParams = {
  /** Branch, tag, or commit; defaults to the repository's default branch. */
  ref?: string;
  /** Path relative to the repository root. */
  path?: string;
};

export type PreviewReleaseNotesParams = {
  tag?: string;
  /** Used when the tag doesn't exist; defaults to the default branch. */
  target?: string;
  prerelease?: boolean;
};

export type DeleteReleaseParams = {
  /** Delete the tag too. */
  cleanup_tag?: boolean;
};

export type UploadAssetParams = {
  name: string;
  /** Replace an asset with the same name. */
  clobber?: boolean;
};

export type DeleteRuleParams = {
  pattern: string;
};

export type ListRunsParams = {
  /** 1 to 1000; defaults to 30. */
  limit?: number;
};

export type GetJobLogsParams = {
  /** The first line to return. */
  offset?: number;
};

export type RerunRunParams = {
  /** Rerun only the jobs that didn't succeed, and the jobs that need them. */
  failed?: boolean;
};

export type GetRepositoryTreeParams = {
  /** Branch, tag, or commit; defaults to the repository's default branch. */
  ref?: string;
  /** Path relative to the repository root. */
  path?: string;
};

export type ListWorkflowsParams = {
  /** Branch, tag, or commit; defaults to the repository's default branch. */
  ref?: string;
  /** Path relative to the repository root. */
  path?: string;
};

