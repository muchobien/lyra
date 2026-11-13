import { useEffect, useState } from "react";
import { Link, Outlet, useParams } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import {
  Code,
  GitPullRequest,
  CircleDot,
  CirclePlay,
  LockKeyhole,
  Moon,
  Sun,
  Check,
} from "lucide-react";
import { api, type Identity, type Repository } from "./api";
import styles from "./App.module.css";

export function App() {
  const { owner, repo } = useParams({ strict: false });
  const identity = useQuery({
    queryKey: ["identity"],
    queryFn: ({ signal }) => api<Identity>("/api/me", signal),
  });
  const repositories = useQuery({
    queryKey: ["repositories"],
    queryFn: ({ signal }) => api<Repository[]>("/api/repos", signal),
  });
  const [theme, setTheme] = useState(localStorage.getItem("lyra-theme") || "system");

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
          <nav className={styles.navigation} aria-label="Repository navigation">
            <Link
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{}}
              className={styles.selected}
              aria-current="page"
            >
              <Code size={16} />
              Code
            </Link>
            <a href={`/${owner}/${repo}/pulls`}>
              <GitPullRequest size={16} />
              Pull requests
            </a>
            <a href={`/${owner}/${repo}/issues`}>
              <CircleDot size={16} />
              Issues
            </a>
            <a href={`/${owner}/${repo}/runs`}>
              <CirclePlay size={16} />
              Actions
            </a>
          </nav>
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
        <Outlet />
      </main>
    </>
  );
}
