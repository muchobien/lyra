import { useGetMe, useListRepos } from "@/api/generated/lyra";
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
