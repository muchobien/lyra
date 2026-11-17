import { HeadContent, Link } from "@tanstack/react-router";
import { type ReactElement, type ReactNode } from "react";
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

const root = document.getElementById("root");
export const initialPageError = root?.dataset.errorMessage
  ? {
      pathname: location.pathname,
      error: new Error(root.dataset.errorMessage),
      title: root.dataset.errorTitle,
    }
  : undefined;

const themes = [
  { id: "system", name: "System", icon: DeviceDesktopIcon },
  { id: "light", name: "Light", icon: SunIcon },
  { id: "dark", name: "Dark", icon: MoonIcon },
];

const repositoryTabs = [
  { to: "/$owner/$repo/pulls", icon: GitPullRequestIcon, label: "Pull requests" },
  { to: "/$owner/$repo/issues", icon: IssueOpenedIcon, label: "Issues" },
  { to: "/$owner/$repo/runs", icon: PlayIcon, label: "Actions" },
  { to: "/$owner/$repo/activity", icon: PulseIcon, label: "Activity" },
  { to: "/$owner/$repo/settings", icon: GearIcon, label: "Settings" },
] as const;

export function App({ children }: { children: ReactNode }): ReactElement {
  const {
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
  } = useConnect();

  let content = children;
  const pageError = initialPageError?.pathname === pathname ? initialPageError : undefined;
  if (identity.error && identity.data) {
    content = <ErrorNotice error={identity.error} retry={retryIdentity} />;
  }
  if (pageError) {
    content = <ErrorNotice error={pageError.error} retry={retryPage} title={pageError.title} />;
  }

  return (
    <>
      {pageError ? (
        <title>{pageError.title ? `${pageError.title} · Lyra` : "Lyra"}</title>
      ) : (
        <HeadContent />
      )}

      <a href="#main" className="lyra-app-skip">
        Skip to content
      </a>

      <header className="lyra-app-header" data-loading={loading || undefined}>
        <div className="lyra-app-bar">
          <Link
            to="/"
            activeOptions={{ exact: true }}
            className="lyra-app-brand"
            aria-label="Lyra repositories"
          >
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
              <Link to="/$owner" params={{ owner }} activeOptions={{ exact: true }}>
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
              activeOptions={{ exact: true, includeSearch: false }}
              data-pending={
                loading &&
                matchRoute({
                  to: "/$owner/$repo",
                  params: { owner, repo },
                  pending: true,
                  includeSearch: false,
                })
                  ? true
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
            {repositoryTabs.map(({ to, icon: Icon, label }) => (
              <Link
                key={to}
                to={to}
                params={{ owner, repo }}
                search={{}}
                activeOptions={{ includeSearch: false }}
                data-pending={
                  loading &&
                  matchRoute({
                    to,
                    params: { owner, repo },
                    pending: true,
                    fuzzy: true,
                    includeSearch: false,
                  })
                    ? true
                    : undefined
                }
              >
                <Icon size={16} />
                {label}
              </Link>
            ))}
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
              <span>{entryRepo}</span>
              {entryRepo === repo && <CheckIcon size={16} />}
            </Link>
          );
        })}
        <Link to="/" onClick={closePicker}>
          All repositories
        </Link>
      </div>

      <main id="main" className="lyra-app-main" aria-busy={loading}>
        <IdentityContext value={identity.data || null}>{content}</IdentityContext>
      </main>
    </>
  );
}
