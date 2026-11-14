import {
  useGetMe,
  useListRepos,
  getGetMeQueryKey,
  getRenderMarkdownQueryKey,
} from "./api/generated/lyra";
import { useEffect, useState } from "react";
import { Link, Outlet, useLocation, useParams } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import {
  Code,
  GitPullRequest,
  CircleDot,
  CirclePlay,
  LockKeyhole,
  Moon,
  Sun,
  Check,
  Activity,
  Settings,
} from "lucide-react";
import { ConnectionContext, IdentityContext } from "./api";
import { ErrorNotice, ScrollNavigation } from "./Controls";
import styles from "./App.module.css";

const initialPath = location.pathname;
const serverError = document.getElementById("root")?.dataset.errorMessage;
const serverErrorTitle = document.getElementById("root")?.dataset.errorTitle;

export function App() {
  const pathname = useLocation({ select: (location) => location.pathname });
  const { owner, repo, section } = useParams({ strict: false });
  const queryClient = useQueryClient();
  const identity = useGetMe();
  const repositories = useListRepos();
  const [theme, setTheme] = useState(localStorage.getItem("lyra-theme") || "system");
  const [connection, setConnection] = useState({ repository: "", status: "connecting" });

  useEffect(() => {
    if (!owner || !repo) {
      return;
    }
    const stream = new EventSource(
      `/api/events/stream?repo=${encodeURIComponent(`${owner}/${repo}`)}`,
    );
    const markdownKey = getRenderMarkdownQueryKey({ body: "" });
    let pending: ReturnType<typeof setTimeout> | undefined;
    function refresh() {
      clearTimeout(pending);
      pending = setTimeout(() => {
        void queryClient.invalidateQueries({
          predicate: (query) =>
            !(query.queryKey[0] === markdownKey[0] && query.queryKey[1] === markdownKey[1]) &&
            query.queryKey[0] !== getGetMeQueryKey()[0],
        });
      }, 200);
    }
    stream.onopen = () => {
      setConnection({ repository: `${owner}/${repo}`, status: "live" });
      refresh();
    };
    stream.onerror = () =>
      setConnection({ repository: `${owner}/${repo}`, status: "reconnecting" });
    for (const action of [
      "push",
      "issue.opened",
      "issue.edited",
      "issue.closed",
      "issue.reopened",
      "issue.comment",
      "issue.labeled",
      "issue.unlabeled",
      "issue.assigned",
      "issue.unassigned",
      "pr.opened",
      "pr.edited",
      "pr.closed",
      "pr.reopened",
      "pr.comment",
      "pr.labeled",
      "pr.unlabeled",
      "pr.assigned",
      "pr.unassigned",
      "pr.review",
      "pr.synchronized",
      "pr.merged",
      "pr.checks",
      "run.queued",
      "run.completed",
      "run.cancelled",
      "job.completed",
      "release.created",
      "release.updated",
      "release.published",
      "release.deleted",
      "release.asset",
      "image.pushed",
      "image.deleted",
      "rule.set",
      "rule.delete",
      "label.create",
      "label.delete",
      "secret.set",
      "secret.delete",
      "variable.set",
      "variable.delete",
    ]) {
      stream.addEventListener(action, refresh);
    }
    return () => {
      clearTimeout(pending);
      stream.close();
    };
  }, [owner, repo, queryClient]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("lyra-theme", theme);
  }, [theme]);

  function changeTheme() {
    const dark =
      theme === "dark" ||
      (theme === "system" && matchMedia("(prefers-color-scheme: dark)").matches);
    setTheme(dark ? "light" : "dark");
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
  let content = <Outlet />;
  if (identity.error) {
    content = <ErrorNotice error={identity.error} retry={retryIdentity} />;
  } else if (identity.isPending) {
    content = <output>Loading workspace…</output>;
  }
  if (serverError && pathname === initialPath) {
    content = (
      <ErrorNotice error={new Error(serverError)} retry={retryPage} title={serverErrorTitle} />
    );
  }

  return (
    <>
      <a href="#main" className={styles.skip}>
        Skip to content
      </a>
      <header className={styles.header}>
        <div className={styles.bar}>
          <Link to="/" className={styles.brand} aria-label="Lyra repositories">
            <svg viewBox="0 0 32 32" aria-hidden="true">
              <path d="m7 7 4 19h10l4-19M7 7l9 8 9-8M11 26l5-11 5 11" />
              <circle cx="7" cy="7" r="2" />
              <circle cx="25" cy="7" r="2" />
              <circle cx="16" cy="15" r="2" />
            </svg>
          </Link>
          {owner && (
            <div className={styles.context}>
              <Link to="/$owner" params={{ owner }}>
                {owner}
              </Link>
              {repo && (
                <>
                  <span>/</span>
                  <button
                    className={styles.repositoryTrigger}
                    popoverTarget="repository-picker"
                    type="button"
                    aria-label={`Switch repository: ${repo}`}
                  >
                    <LockKeyhole size={13} />
                    <strong>{repo}</strong>
                  </button>
                </>
              )}
            </div>
          )}
          <div className={styles.appearance}>
            {owner && repo && (
              <Link
                to="/$owner/$repo/$section"
                params={{ owner, repo, section: "settings" }}
                search={{}}
                className={styles.theme}
                aria-label="Repository settings"
              >
                <Settings size={18} />
              </Link>
            )}
            <button
              className={styles.theme}
              type="button"
              onClick={changeTheme}
              aria-label="Switch appearance"
            >
              {theme === "light" ? <Moon size={18} /> : <Sun size={18} />}
            </button>
            <span className={styles.avatar} title={identity.data?.actor.user}>
              {identity.data?.actor.user.slice(0, 1).toUpperCase() || "?"}
            </span>
          </div>
        </div>
        {owner && repo && (
          <ScrollNavigation
            className={styles.navigation}
            label="Repository navigation"
            active={section || "code"}
          >
            <Link
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{}}
              className={
                !section ||
                section === "commits" ||
                section === "commit" ||
                section === "tags" ||
                section === "releases" ||
                section === "images"
                  ? styles.selected
                  : undefined
              }
              aria-current={
                !section || ["commits", "commit", "tags", "releases", "images"].includes(section)
                  ? "page"
                  : undefined
              }
            >
              <Code size={16} />
              Code
            </Link>
            <Link
              to="/$owner/$repo/$section"
              params={{ owner, repo, section: "pulls" }}
              search={{}}
              className={section === "pulls" ? styles.selected : undefined}
              aria-current={section === "pulls" ? "page" : undefined}
            >
              <GitPullRequest size={16} />
              Pull requests
            </Link>
            <Link
              to="/$owner/$repo/$section"
              params={{ owner, repo, section: "issues" }}
              search={{}}
              className={section === "issues" ? styles.selected : undefined}
              aria-current={section === "issues" ? "page" : undefined}
            >
              <CircleDot size={16} />
              Issues
            </Link>
            <Link
              to="/$owner/$repo/$section"
              params={{ owner, repo, section: "runs" }}
              search={{}}
              className={section === "runs" ? styles.selected : undefined}
              aria-current={section === "runs" ? "page" : undefined}
            >
              <CirclePlay size={16} />
              Actions
            </Link>
            <Link
              to="/$owner/$repo/$section"
              params={{ owner, repo, section: "activity" }}
              search={{}}
              className={section === "activity" ? styles.selected : undefined}
              aria-current={section === "activity" ? "page" : undefined}
            >
              <Activity size={16} />
              Activity
            </Link>
          </ScrollNavigation>
        )}
      </header>
      <div id="repository-picker" popover="auto" className={styles.repositoryPicker}>
        <span className={styles.pickerLabel}>Repositories</span>
        {repositories.data?.map((entry) => {
          const [entryOwner, entryRepo] = entry.name.split("/");
          return (
            <Link
              key={entry.name}
              to="/$owner/$repo"
              params={{ owner: entryOwner, repo: entryRepo }}
              search={{}}
              onClick={closePicker}
            >
              <span>{entry.name}</span>
              {entryOwner === owner && entryRepo === repo && <Check size={15} />}
            </Link>
          );
        })}
        <Link to="/" onClick={closePicker}>
          All repositories
        </Link>
      </div>
      <main id="main" className={styles.main}>
        <IdentityContext value={identity.data || null}>
          <ConnectionContext
            value={connection.repository === `${owner}/${repo}` ? connection.status : "connecting"}
          >
            {content}
          </ConnectionContext>
        </IdentityContext>
      </main>
      <link rel="stylesheet" href="/api/code.css" />
    </>
  );
}
