import { useContext, useEffect } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { GitPullRequest, CircleDot, MessageSquare, Plus } from "lucide-react";
import { IdentityContext } from "./api";
import type { CreateIssueRequest, ItemWrite, PullWrite, Item, Pull } from "./api/generated/models";
import {
  useListIssues,
  useListPulls,
  useListLabels,
  useGetRepo,
  useCreateIssue,
  useCreatePull,
} from "./api/generated/lyra";
import { ErrorNotice, formValue } from "./Controls";
import { ActorName, DateTime, DialogClose, FormError, MarkdownEditor, Status } from "./Workspace";
import { useDraft } from "./drafts";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

type ItemListProps = { kind: "pulls" | "issues" };
export function ItemList({ kind }: ItemListProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const canWrite = identity?.scope !== "read";
  const state = search.state || "open";
  const limit = Math.max(50, Math.min(1000, Number(search.limit) || 50));
  const params = {
    state,
    limit,
    q: search.q,
    label: search.label ? [search.label] : undefined,
    assignee: search.assignee ? [search.assignee] : undefined,
  };
  const issues = useListIssues(owner, repo, params, { query: { enabled: kind === "issues" } });
  const pulls = useListPulls(owner, repo, params, { query: { enabled: kind === "pulls" } });
  const items = kind === "pulls" ? pulls : issues;
  const listedItems: (Item | Pull)[] = items.data || [];
  const labels = useListLabels(owner, repo);
  const metadata = useGetRepo(owner, repo);
  const title = useDraft(`${identity?.actor.user}/${name}/${kind}/new/title`);
  const description = useDraft(`${identity?.actor.user}/${name}/${kind}/new/body`);
  function created(item: ItemWrite | PullWrite) {
    title.clear();
    description.clear();
    document.querySelector<HTMLDialogElement>("#new-item")?.close();
    void queryClient.invalidateQueries();
    void navigate({
      to: "/$owner/$repo/$section/$",
      params: { owner, repo, section: kind, _splat: String(item.number) },
      search: {},
    });
  }
  const issueCreation = useCreateIssue({ mutation: { onSuccess: created } });
  const pullCreation = useCreatePull({ mutation: { onSuccess: created } });
  const creation = kind === "pulls" ? pullCreation : issueCreation;
  useEffect(() => {
    document.title = `${kind === "pulls" ? "Pull requests" : "Issues"} · ${name} · Lyra`;
  }, [kind, name]);
  function retry() {
    void items.refetch();
  }
  function filtersChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    if (!event.currentTarget.form) {
      return;
    }
    const form = new FormData(event.currentTarget.form);
    void navigate({
      to: "/$owner/$repo/$section",
      params: { owner, repo, section: kind },
      search: {
        state,
        q: formValue(form, "q"),
        label: formValue(form, "label"),
        assignee: formValue(form, "assignee"),
      },
      replace: true,
      resetScroll: false,
    });
  }
  function preventSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
  }
  function titleChanged(event: ChangeEvent<HTMLInputElement>) {
    title.setValue(event.target.value);
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const body: CreateIssueRequest = {
      title: title.value,
      body: description.value,
      labels: formValue(form, "labels")
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
      assignees: formValue(form, "assignees")
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
    };
    if (kind === "pulls") {
      pullCreation.mutate({
        owner,
        repo,
        data: { ...body, base: formValue(form, "base"), head: formValue(form, "head") },
      });
    } else {
      issueCreation.mutate({ owner, repo, data: body });
    }
  }
  return (
    <>
      <search>
        <form className={styles.toolbar} onSubmit={preventSearchSubmit}>
          <input
            className={styles.search}
            aria-label={`Search ${kind === "pulls" ? "pull requests" : "issues"}`}
            name="q"
            placeholder="Search title or #number…"
            value={search.q || ""}
            onChange={filtersChanged}
          />
          <select
            className={controls.select}
            name="label"
            aria-label="Filter by label"
            value={search.label || ""}
            onChange={filtersChanged}
          >
            <option value="">All labels</option>
            {labels.data?.map((label) => (
              <option key={label.name}>{label.name}</option>
            ))}
          </select>
          <select
            className={controls.select}
            name="assignee"
            aria-label="Filter by assignee"
            value={search.assignee || ""}
            onChange={filtersChanged}
          >
            <option value="">Anyone</option>
            <option value="@me">Assigned to me</option>
            {search.assignee && search.assignee !== "@me" && (
              <option value={search.assignee}>{search.assignee}</option>
            )}
          </select>
          {canWrite && (
            <button
              className={styles.primary}
              type="button"
              commandfor="new-item"
              command="show-modal"
              disabled={kind === "pulls" && (metadata.data?.branches.length || 0) < 2}
              title={
                kind === "pulls" && (metadata.data?.branches.length || 0) < 2
                  ? "Push another branch to open a pull request"
                  : undefined
              }
            >
              <Plus size={15} />
              {kind === "pulls" ? "New pull request" : "New issue"}
            </button>
          )}
        </form>
      </search>
      <nav className={styles.tabs} aria-label="State filter">
        <Link
          to="/$owner/$repo/$section"
          params={{ owner, repo, section: kind }}
          search={{ ...search, state: "open" }}
          aria-current={state === "open" ? "page" : undefined}
        >
          Open
        </Link>
        <Link
          to="/$owner/$repo/$section"
          params={{ owner, repo, section: kind }}
          search={{ ...search, state: "closed" }}
          aria-current={state === "closed" ? "page" : undefined}
        >
          Closed
        </Link>
        {kind === "pulls" && (
          <Link
            to="/$owner/$repo/$section"
            params={{ owner, repo, section: kind }}
            search={{ ...search, state: "merged" }}
            aria-current={state === "merged" ? "page" : undefined}
          >
            Merged
          </Link>
        )}
        <Link
          to="/$owner/$repo/$section"
          params={{ owner, repo, section: kind }}
          search={{ ...search, state: "all" }}
          aria-current={state === "all" ? "page" : undefined}
        >
          All
        </Link>
      </nav>
      {items.error && <ErrorNotice error={items.error} retry={retry} />}
      {items.isPending && (
        <output className={styles.empty}>
          Loading {kind === "pulls" ? "pull requests" : "issues"}…
        </output>
      )}
      {items.data && (
        <div className={styles.panel}>
          {listedItems.map((item) => (
            <article className={styles.row} key={item.number}>
              {kind === "pulls" ? <GitPullRequest size={18} /> : <CircleDot size={18} />}
              <div className={styles.grow}>
                <Link
                  className={styles.rowTitle}
                  to="/$owner/$repo/$section/$"
                  params={{ owner, repo, section: kind, _splat: String(item.number) }}
                  search={{}}
                >
                  {item.title}
                </Link>
                <p className={styles.meta}>
                  #{item.number} · <ActorName actor={item.author} /> ·{" "}
                  <DateTime value={item.created_at} />
                  {"head" in item && (
                    <>
                      {" "}
                      · <code>{item.head}</code> into <code>{item.base}</code>
                    </>
                  )}
                </p>
                <div className={styles.labels}>
                  {item.labels.map((label) => (
                    <span className={styles.label} key={label}>
                      {label}
                    </span>
                  ))}
                  {item.assignees.length > 0 && (
                    <span className={styles.meta}>{item.assignees.join(", ")}</span>
                  )}
                </div>
              </div>
              <Status value={item.state} issue={kind === "issues"} />
              {item.comment_count > 0 && (
                <span className={styles.meta} aria-label={`${item.comment_count} comments`}>
                  <MessageSquare size={13} /> {item.comment_count}
                </span>
              )}
            </article>
          ))}
          {items.data.length === 0 && (
            <p className={styles.empty}>
              No matching {kind === "pulls" ? "pull requests" : "issues"}.
            </p>
          )}
        </div>
      )}
      {items.data?.length === limit && limit < 1000 && (
        <div className={styles.actions}>
          <Link
            className={controls.button}
            to="/$owner/$repo/$section"
            params={{ owner, repo, section: kind }}
            search={{ ...search, limit: String(limit + 50) }}
          >
            Load more
          </Link>
        </div>
      )}
      {(kind !== "pulls" || metadata.data) && (
        <dialog
          id="new-item"
          className={styles.dialog}
          closedby="any"
          aria-labelledby="new-item-title"
        >
          <form onSubmit={submit}>
            <h2 id="new-item-title">{kind === "pulls" ? "New pull request" : "New issue"}</h2>
            <fieldset disabled={creation.isPending}>
              {kind === "pulls" && (
                <div className={styles.toolbar}>
                  <label className={styles.field}>
                    <span>Base</span>
                    <select
                      className={controls.select}
                      name="base"
                      defaultValue={metadata.data?.default_branch}
                    >
                      {metadata.data?.branches.map((branch) => (
                        <option key={branch.name}>{branch.name}</option>
                      ))}
                    </select>
                  </label>
                  <label className={styles.field}>
                    <span>Compare</span>
                    <select
                      className={controls.select}
                      name="head"
                      defaultValue={
                        metadata.data?.branches.find(
                          (branch) => branch.name !== metadata.data?.default_branch,
                        )?.name
                      }
                    >
                      {metadata.data?.branches.map((branch) => (
                        <option key={branch.name}>{branch.name}</option>
                      ))}
                    </select>
                  </label>
                </div>
              )}
              <label className={styles.field}>
                <span>Title</span>
                <input
                  name="title"
                  required
                  maxLength={256}
                  value={title.value}
                  onChange={titleChanged}
                  placeholder="What needs to change?"
                />
              </label>
              <MarkdownEditor
                id="new-item-body"
                label="Description"
                value={description.value}
                onChange={description.setValue}
              />
              <label className={styles.field}>
                <span>Labels</span>
                <input name="labels" placeholder="bug, enhancement" list="available-labels" />
              </label>
              <datalist id="available-labels">
                {labels.data?.map((label) => (
                  <option key={label.name} value={label.name}>
                    {label.name}
                  </option>
                ))}
              </datalist>
              <label className={styles.field}>
                <span>Assignees</span>
                <input name="assignees" placeholder="Names, separated by commas" />
              </label>
              <FormError error={creation.error} />
              <div className={styles.dialogActions}>
                <button
                  className={controls.button}
                  type="button"
                  commandfor="new-item"
                  command="close"
                >
                  Cancel
                </button>
                <button className={styles.primary} type="submit">
                  {creation.isPending ? "Creating…" : "Create"}
                </button>
              </div>
            </fieldset>
            <DialogClose target="new-item" />
          </form>
        </dialog>
      )}
    </>
  );
}
