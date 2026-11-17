import { Link, Outlet } from "@tanstack/react-router";
import {
  CheckIcon,
  CodeIcon,
  GearIcon,
  GitPullRequestIcon,
  IssueOpenedIcon,
  LockIcon,
  DeviceDesktopIcon,
  MoonIcon,
  PlayIcon,
  PulseIcon,
  SunIcon,
} from "@primer/octicons-react";
import { IdentityContext } from "@/api";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ScrollNavigation } from "@/components/controls/ScrollNavigation";
import "./styles.css";
import { useConnect } from "./connect";

const initialPath = location.pathname;

const serverError = document.getElementById("root")?.dataset.errorMessage;

const serverErrorTitle = document.getElementById("root")?.dataset.errorTitle;

const themes = [
  { id: "system", name: "System", icon: DeviceDesktopIcon },
  { id: "light", name: "Light", icon: SunIcon },
  { id: "dark", name: "Dark", icon: MoonIcon },
];

export function App() {
  const {
    pathname,
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
  } = useConnect();
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
      <a href="#main" className="lyra-app-skip">
        Skip to content
      </a>
      <header className="lyra-app-header">
        <div className="lyra-app-bar">
          <Link to="/" className="lyra-app-brand" aria-label="Lyra repositories">
            <svg viewBox="0 0 32 32" aria-hidden="true">
              <path d="M21 6l6 4-6.5 3.5L21 6m-.5 7.5L10 15 7.5 27 18 25.5l2.5-12" />
              <g className="lyra-app-stars">
                <circle cx="21" cy="6" r="3.2" />
                <circle cx="27" cy="10" r="1.6" />
                <circle cx="20.5" cy="13.5" r="1.9" />
                <circle cx="10" cy="15" r="1.9" />
                <circle cx="7.5" cy="27" r="2.2" />
                <circle cx="18" cy="25.5" r="2.1" />
              </g>
            </svg>
          </Link>
          {owner && (
            <div className="lyra-app-context">
              <Link to="/$owner" params={{ owner }}>
                {owner}
              </Link>
              {repo && (
                <>
                  <span>/</span>
                  <button
                    className="lyra-app-repositoryTrigger"
                    popoverTarget="repository-picker"
                    type="button"
                    aria-label={`Switch repository: ${repo}`}
                  >
                    <LockIcon size={16} />
                    <strong>{repo}</strong>
                  </button>
                </>
              )}
            </div>
          )}
          <button
            className="lyra-app-avatar"
            popoverTarget="account-menu"
            type="button"
            aria-label="Account and appearance"
          >
            {identity.data?.actor.user.slice(0, 1).toUpperCase() || "?"}
          </button>
        </div>
        {owner && repo && (
          <ScrollNavigation
            className="lyra-app-navigation"
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
                  ? "lyra-app-selected"
                  : undefined
              }
              aria-current={
                !section || ["commits", "commit", "tags", "releases", "images"].includes(section)
                  ? "page"
                  : undefined
              }
            >
              <CodeIcon size={16} />
              Code
            </Link>
            <Link
              to="/$owner/$repo/pulls"
              params={{ owner, repo }}
              search={{}}
              className={section === "pulls" ? "lyra-app-selected" : undefined}
              aria-current={section === "pulls" ? "page" : undefined}
            >
              <GitPullRequestIcon size={16} />
              Pull requests
            </Link>
            <Link
              to="/$owner/$repo/issues"
              params={{ owner, repo }}
              search={{}}
              className={section === "issues" ? "lyra-app-selected" : undefined}
              aria-current={section === "issues" ? "page" : undefined}
            >
              <IssueOpenedIcon size={16} />
              Issues
            </Link>
            <Link
              to="/$owner/$repo/runs"
              params={{ owner, repo }}
              search={{}}
              className={section === "runs" ? "lyra-app-selected" : undefined}
              aria-current={section === "runs" ? "page" : undefined}
            >
              <PlayIcon size={16} />
              Actions
            </Link>
            <Link
              to="/$owner/$repo/activity"
              params={{ owner, repo }}
              search={{}}
              className={section === "activity" ? "lyra-app-selected" : undefined}
              aria-current={section === "activity" ? "page" : undefined}
            >
              <PulseIcon size={16} />
              Activity
            </Link>
            <Link
              to="/$owner/$repo/settings"
              params={{ owner, repo }}
              search={{}}
              className={section === "settings" ? "lyra-app-selected" : undefined}
              aria-current={section === "settings" ? "page" : undefined}
            >
              <GearIcon size={16} />
              Settings
            </Link>
          </ScrollNavigation>
        )}
      </header>
      <div id="account-menu" popover="auto" className="lyra-app-accountMenu">
        <span className="lyra-app-pickerLabel">Signed in as</span>
        <strong className="lyra-app-account">{identity.data?.actor.user}</strong>
        <fieldset className="lyra-app-themes">
          <legend className="lyra-app-pickerLabel">Theme</legend>
          {themes.map((entry) => {
            const Icon = entry.icon;
            return (
              <label key={entry.id}>
                <input
                  type="radio"
                  name="theme"
                  value={entry.id}
                  checked={theme === entry.id}
                  onChange={changeTheme}
                />
                <Icon size={16} />
                {entry.name}
              </label>
            );
          })}
        </fieldset>
        {instanceAdmin && (
          <Link to="/settings" search={{}} onClick={closeAccount}>
            <GearIcon size={16} />
            Instance settings
          </Link>
        )}
      </div>
      <div id="repository-picker" popover="auto" className="lyra-app-repositoryPicker">
        <span className="lyra-app-pickerLabel">Repositories</span>
        {repositories.data
          ?.filter((entry) => entry.name.startsWith(`${owner}/`))
          .map((entry) => {
            const [entryOwner, entryRepo] = entry.name.split("/");
            return (
              <Link
                key={entry.name}
                to="/$owner/$repo"
                params={{ owner: entryOwner, repo: entryRepo }}
                search={{}}
                onClick={closePicker}
              >
                <span>{entryRepo}</span>
                {entryRepo === repo && <CheckIcon size={16} />}
              </Link>
            );
          })}
        <Link to="/" onClick={closePicker}>
          All repositories
        </Link>
      </div>
      <main id="main" className="lyra-app-main">
        <IdentityContext value={identity.data || null}>{content}</IdentityContext>
      </main>
      <link rel="stylesheet" href="/api/code.css" />
    </>
  );
}
