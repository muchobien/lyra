import { useGetMe, useListRepos } from "@/api/generated/lyra";
import { useEffect, useState, type ChangeEvent } from "react";
import { useLocation, useMatches, useParams } from "@tanstack/react-router";

export function useConnect() {
  const pathname = useLocation({ select: (location) => location.pathname });
  const { owner, repo } = useParams({ strict: false });
  const section = useMatches({ select: (matches) => matches.at(-1)?.staticData.section });
  const identity = useGetMe();
  const repositories = useListRepos();
  const [theme, setTheme] = useState(localStorage.getItem("lyra-theme") || "system");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("lyra-theme", theme);
  }, [theme]);

  function changeTheme(event: ChangeEvent<HTMLInputElement>) {
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
