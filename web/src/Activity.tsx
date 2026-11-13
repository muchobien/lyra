import { useContext, useEffect } from "react";
import type { ChangeEvent } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Activity as ActivityIcon, GitCommitHorizontal } from "lucide-react";
import { api, ConnectionContext, type ActivityEvent } from "./api";
import { ErrorNotice } from "./Controls";
import { ActorName, DateTime } from "./Workspace";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

export function Activity() {
  const connection = useContext(ConnectionContext);
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate({ from: "/$owner/$repo/$section" });
  const name = `${owner}/${repo}`;
  const limit = Math.max(100, Math.min(1000, Number(search.limit) || 100));
  const endpoint = `/api/events?${new URLSearchParams({ repo: name, limit: String(limit) })}`;
  const events = useQuery({
    queryKey: [endpoint],
    queryFn: ({ signal }) => api<ActivityEvent[]>(endpoint, signal),
  });
  useEffect(() => {
    document.title = `Activity · ${name} · Lyra`;
  }, [name]);
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
  return (
    <>
      <div className={styles.toolbar}>
        <ActivityIcon size={18} />
        <span className={styles.live} data-connected={connection === "live"}>
          {connection}
        </span>
        <input
          className={styles.search}
          type="search"
          name="q"
          aria-label="Search activity"
          placeholder="Search activity…"
          value={search.q || ""}
          onChange={filterChanged}
        />
        <select
          className={controls.select}
          name="action"
          aria-label="Activity type"
          value={search.action || ""}
          onChange={filterChanged}
        >
          <option value="">All activity</option>
          <option value="push">Pushes</option>
          <option value="pr.">Pull requests</option>
          <option value="issue.">Issues</option>
          <option value="run.">Actions</option>
          <option value="release.">Releases</option>
          <option value="image.">Containers</option>
          <option value="rule.">Branch rules</option>
          <option value="variable.">Variables</option>
          <option value="secret.">Secrets</option>
        </select>
      </div>
      {events.error && <ErrorNotice error={events.error} retry={retry} />}
      {events.isPending && <output className={styles.empty}>Loading activity…</output>}
      {events.data && (
        <div className={styles.panel}>
          {visible.map((event) => (
            <article className={styles.row} key={event.id}>
              <GitCommitHorizontal size={17} />
              <div className={styles.grow}>
                <p>
                  <ActorName actor={event.actor} />{" "}
                  <span className={styles.muted}>{event.action.replaceAll(".", " ")}</span>
                  {event.item && (
                    <Link
                      className={styles.link}
                      to="/$owner/$repo/$section/$"
                      params={{
                        owner,
                        repo,
                        section: event.action.startsWith("pr.") ? "pulls" : "issues",
                        _splat: String(event.item),
                      }}
                      search={{}}
                    >
                      {" "}
                      #{event.item}
                    </Link>
                  )}
                </p>
                {typeof event.data.title === "string" && (
                  <p className={styles.meta}>{event.data.title}</p>
                )}
                {typeof event.data.name === "string" && (
                  <p className={styles.meta}>{event.data.name}</p>
                )}
                {event.action.startsWith("run.") && typeof event.data.number === "number" && (
                  <Link
                    className={styles.link}
                    to="/$owner/$repo/$section/$"
                    params={{ owner, repo, section: "runs", _splat: String(event.data.number) }}
                    search={{}}
                  >
                    Run #{event.data.number}
                  </Link>
                )}
                <details className={styles.disclosure}>
                  <summary>Event data</summary>
                  <pre className={styles.plainText}>{JSON.stringify(event.data, null, 2)}</pre>
                </details>
              </div>
              <span className={styles.meta}>
                <DateTime value={event.at} />
              </span>
            </article>
          ))}
          {visible.length === 0 && <p className={styles.empty}>No matching activity.</p>}
        </div>
      )}
      {events.data?.length === limit && limit < 1000 && (
        <div className={styles.actions}>
          <Link
            className={controls.button}
            to="/$owner/$repo/$section"
            params={{ owner, repo, section: "activity" }}
            search={{ ...search, limit: String(limit + 100) }}
          >
            Load older activity
          </Link>
        </div>
      )}
    </>
  );
}
