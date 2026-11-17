import { useListEvents } from "@/api/generated/lyra";
import { useContext, type ChangeEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { ConnectionContext } from "@/api";
import { searchLimit } from "@/routing/search";

export function useConnect() {
  const connection = useContext(ConnectionContext);
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: "/$owner/$repo/activity" });
  const navigate = useNavigate({ from: "/$owner/$repo/activity" });

  const name = `${owner}/${repo}`;
  const limit = searchLimit(search.limit, 100);
  const events = useListEvents({ repo: name, limit });

  function retry() {
    void events.refetch();
  }

  function filterChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    void navigate({
      search: { ...search, [event.target.name]: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }

  const visible =
    events.data?.filter(
      (event) =>
        (!search.action || event.action.startsWith(search.action)) &&
        (!search.q ||
          `${event.actor.user} ${event.actor.agent || ""} ${event.action} ${JSON.stringify(event.data)}`
            .toLowerCase()
            .includes(search.q.toLowerCase())),
    ) || [];
  const dayFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "full" });
  const timeFormat = new Intl.DateTimeFormat(undefined, { timeStyle: "short" });

  return {
    connection,
    owner,
    repo,
    search,
    limit,
    events,
    retry,
    filterChanged,
    visible,
    dayFormat,
    timeFormat,
  };
}
