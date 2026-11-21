import {
  commentIssue,
  commentPull,
  createIssue,
  getIssueWorkspace,
  getJobLogs,
  getPullDiff,
  getPullWorkspace,
  getRun,
  listIssues,
  listPulls,
  listRepos,
  listRuns,
  useGetMe,
  useListRepos,
} from "@/api/generated/lyra";
import { useEffect, useState, type ChangeEvent } from "react";
import {
  useBlocker,
  useLocation,
  useMatches,
  useMatchRoute,
  useParams,
  useRouterState,
} from "@tanstack/react-router";
import { hasDrafts } from "@/hooks/useDraft";

export function useConnect() {
  const pathname = useLocation({ select: (location) => location.pathname });
  const loading = useRouterState({ select: (state) => state.isLoading });
  const matchRoute = useMatchRoute();
  const { owner, repo } = useParams({ strict: false });
  const section = useMatches({ select: (matches) => matches.at(-1)?.staticData.section });

  const identity = useGetMe();
  const repositories = useListRepos({
    query: {
      enabled: !!identity.data && !!repo,
      select: (repositories) => repositories.filter((entry) => entry.name.startsWith(`${owner}/`)),
    },
  });

  useEffect(() => {
    const context = document.modelContext;
    if (!identity.data || !context) {
      return;
    }
    const controller = new AbortController();
    const options = { signal: controller.signal };
    const repository = {
      owner: { type: "string", description: "The repository's owner" },
      repo: { type: "string", description: "The repository's name, without the owner" },
    } as const;
    const item = {
      ...repository,
      number: { type: "integer", description: "The issue or pull request number" },
    } as const;
    const reading = { readOnlyHint: true, untrustedContentHint: true };
    const registrations = [
      context.registerTool(
        {
          name: "list_repositories",
          description: "List the Lyra repositories you can read, with their descriptions.",
          inputSchema: { type: "object", properties: {} },
          annotations: { readOnlyHint: true },
          execute: async (_, { signal }) =>
            (await listRepos({ signal })).map(({ name, description }) => ({ name, description })),
        },
        options,
      ),
      context.registerTool(
        {
          name: "list_issues",
          description: "List a repository's issues, newest first.",
          inputSchema: {
            type: "object",
            properties: {
              ...repository,
              state: { type: "string", enum: ["open", "closed", "all"] },
              query: { type: "string", description: "Words in the title, or an issue number" },
            },
            required: ["owner", "repo"],
          },
          annotations: reading,
          execute: ({ owner, repo, state, query }, { signal }) =>
            listIssues(owner, repo, { state, q: query }, { signal }),
        },
        options,
      ),
      context.registerTool(
        {
          name: "list_pull_requests",
          description: "List a repository's pull requests, newest first.",
          inputSchema: {
            type: "object",
            properties: {
              ...repository,
              state: { type: "string", enum: ["open", "closed", "merged", "all"] },
              query: {
                type: "string",
                description: "Words in the title, or a pull request number",
              },
            },
            required: ["owner", "repo"],
          },
          annotations: reading,
          execute: ({ owner, repo, state, query }, { signal }) =>
            listPulls(owner, repo, { state, q: query }, { signal }),
        },
        options,
      ),
      context.registerTool(
        {
          name: "get_issue",
          description: "Read an issue with its description and comments.",
          inputSchema: { type: "object", properties: item, required: ["owner", "repo", "number"] },
          annotations: reading,
          execute: async ({ owner, repo, number }, { signal }) => {
            const workspace = await getIssueWorkspace(owner, repo, number, { signal });
            return {
              issue: workspace.item,
              comments: workspace.comments.map(({ author, body, created_at }) => ({
                author,
                body,
                created_at,
              })),
            };
          },
        },
        options,
      ),
      context.registerTool(
        {
          name: "get_pull_request",
          description:
            "Read a pull request with its description, comments, reviews, checks and merge status.",
          inputSchema: { type: "object", properties: item, required: ["owner", "repo", "number"] },
          annotations: reading,
          execute: async ({ owner, repo, number }, { signal }) => {
            const workspace = await getPullWorkspace(owner, repo, number, { signal });
            return {
              pull_request: workspace.item,
              branches: workspace.pull,
              checks: workspace.checks,
              merge: workspace.merge,
              comments: workspace.comments.map(({ author, body, created_at, path, line }) => ({
                author,
                body,
                created_at,
                path,
                line,
              })),
              reviews: workspace.reviews.map(({ author, state, body, comments }) => ({
                author,
                state,
                body,
                comments: comments.map(({ path, line, body: comment }) => ({
                  path,
                  line,
                  body: comment,
                })),
              })),
            };
          },
        },
        options,
      ),
      context.registerTool(
        {
          name: "get_pull_request_diff",
          description: "Read a pull request's changes as a unified diff.",
          inputSchema: { type: "object", properties: item, required: ["owner", "repo", "number"] },
          annotations: reading,
          execute: ({ owner, repo, number }, { signal }) =>
            getPullDiff(owner, repo, number, { signal }),
        },
        options,
      ),
      context.registerTool(
        {
          name: "list_runs",
          description: "List a repository's latest Actions runs with their status.",
          inputSchema: { type: "object", properties: repository, required: ["owner", "repo"] },
          annotations: reading,
          execute: ({ owner, repo }, { signal }) => listRuns(owner, repo, undefined, { signal }),
        },
        options,
      ),
      context.registerTool(
        {
          name: "get_run",
          description: "Read an Actions run with its jobs and their steps.",
          inputSchema: {
            type: "object",
            properties: { ...repository, run: { type: "integer", description: "The run number" } },
            required: ["owner", "repo", "run"],
          },
          annotations: reading,
          execute: ({ owner, repo, run }, { signal }) => getRun(owner, repo, run, { signal }),
        },
        options,
      ),
      context.registerTool(
        {
          name: "get_job_log",
          description: "Read the last 300 lines of an Actions job's log.",
          inputSchema: {
            type: "object",
            properties: {
              ...repository,
              run: { type: "integer", description: "The run number" },
              job: { type: "integer", description: "The job id from get_run" },
            },
            required: ["owner", "repo", "run", "job"],
          },
          annotations: reading,
          execute: async ({ owner, repo, run, job }, { signal }) =>
            (await getJobLogs(owner, repo, run, job, undefined, { signal }))
              .split("\n")
              .slice(-300)
              .join("\n"),
        },
        options,
      ),
    ];
    if (identity.data.scope !== "read") {
      registrations.push(
        context.registerTool(
          {
            name: "create_issue",
            description: "Open an issue in a repository.",
            inputSchema: {
              type: "object",
              properties: {
                ...repository,
                title: { type: "string" },
                body: { type: "string", description: "Markdown" },
              },
              required: ["owner", "repo", "title"],
            },
            annotations: { consequentialHint: true },
            execute: ({ owner, repo, title, body }, { signal }) =>
              createIssue(owner, repo, { title, body }, { signal }),
          },
          options,
        ),
        context.registerTool(
          {
            name: "add_comment",
            description: "Comment on an issue or a pull request.",
            inputSchema: {
              type: "object",
              properties: {
                ...item,
                kind: { type: "string", enum: ["issue", "pull_request"] },
                body: { type: "string", description: "Markdown" },
              },
              required: ["owner", "repo", "number", "kind", "body"],
            },
            annotations: { consequentialHint: true },
            execute: ({ owner, repo, number, kind, body }, { signal }) =>
              kind === "pull_request"
                ? commentPull(owner, repo, number, { body }, { signal })
                : commentIssue(owner, repo, number, { body }, { signal }),
          },
          options,
        ),
      );
    }
    Promise.all(registrations).catch(console.warn);
    return () => controller.abort();
  }, [identity.data]);

  const [theme, setTheme] = useState(localStorage.getItem("lyra-theme") || "system");

  useBlocker({
    shouldBlockFn: ({ current, next }) =>
      hasUnsavedForm() &&
      (current.pathname !== next.pathname ||
        JSON.stringify(current.search) !== JSON.stringify(next.search)) &&
      !window.confirm("Leave this page with unsaved form changes?"),
    enableBeforeUnload: () => hasUnsavedForm() || hasDrafts(),
  });

  useEffect(() => {
    const meta = document.querySelector<HTMLMetaElement>('meta[name="color-scheme"]');
    if (theme === "light" || theme === "dark") {
      meta?.setAttribute("content", theme);
    } else {
      meta?.setAttribute("content", "light dark");
    }
  }, [theme]);

  useEffect(() => {
    function synced(event: StorageEvent) {
      if (event.key === "lyra-theme") {
        setTheme(event.newValue || "system");
      }
    }
    window.addEventListener("storage", synced);
    return () => window.removeEventListener("storage", synced);
  }, []);

  function changeTheme(event: ChangeEvent<HTMLInputElement>) {
    if (event.target.value === "system") {
      localStorage.removeItem("lyra-theme");
    } else {
      localStorage.setItem("lyra-theme", event.target.value);
    }
    setTheme(event.target.value);
  }

  function closeAccount() {
    document.getElementById("account-menu")?.hidePopover();
  }

  function closePicker() {
    document.getElementById("repository-picker")?.hidePopover();
  }

  function retryIdentity() {
    void identity.refetch();
  }

  function retryPage() {
    location.reload();
  }

  const instanceAdmin = identity.data?.scope === "admin" && !identity.data.repos.length;

  return {
    pathname,
    loading,
    matchRoute,
    owner,
    repo,
    section,
    identity,
    repositories,
    theme,
    changeTheme,
    closeAccount,
    closePicker,
    retryIdentity,
    retryPage,
    instanceAdmin,
  };
}

function hasUnsavedForm() {
  return !!document.querySelector("dialog[open] form[data-dirty]");
}
