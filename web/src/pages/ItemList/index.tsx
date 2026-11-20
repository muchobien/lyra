import { Field } from "@/components/forms/Field";
import { FormDialog } from "@/components/forms/FormDialog";
import { type CSSProperties } from "react";
import { Link } from "@tanstack/react-router";
import {
  CommentIcon,
  GitMergeIcon,
  GitPullRequestClosedIcon,
  GitPullRequestIcon,
  IssueClosedIcon,
  IssueOpenedIcon,
  PersonIcon,
  PlusIcon,
  SparkleIcon,
} from "@primer/octicons-react";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ActorName } from "@/components/metadata/ActorName";
import { DateTime } from "@/components/metadata/DateTime";
import { MarkdownEditor } from "@/components/markdown/MarkdownEditor";
import { OnDeviceOutput } from "@/components/ai/OnDeviceOutput";
import { itemListPaths, itemDetailPaths } from "@/routing/paths";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "./styles.css";
import { useConnect, type ItemListProps } from "./connect";

const stateIcons = {
  issues: { open: IssueOpenedIcon, closed: IssueClosedIcon, merged: IssueClosedIcon },
  pulls: { open: GitPullRequestIcon, closed: GitPullRequestClosedIcon, merged: GitMergeIcon },
};

export function ItemList({ kind }: ItemListProps) {
  const {
    owner,
    repo,
    search,
    canWrite,
    state,
    limit,
    items,
    listedItems,
    labels,
    metadata,
    title,
    description,
    creation,
    draft,
    retry,
    filtersChanged,
    preventSearchSubmit,
    titleChanged,
    draftDescription,
    insertDraft,
    submit,
  } = useConnect({ kind });

  return (
    <>
      <search>
        <form className="lyra-workspace-toolbar" onSubmit={preventSearchSubmit}>
          <input
            className="lyra-workspace-search"
            aria-label={`Search ${kind === "pulls" ? "pull requests" : "issues"}`}
            name="q"
            placeholder="Search title or #number…"
            value={search.q || ""}
            onChange={filtersChanged}
          />
          <select
            className="lyra-controls-select"
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
            className="lyra-controls-select"
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
              className="lyra-workspace-primary"
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
              <PlusIcon size={16} />
              {kind === "pulls" ? "New pull request" : "New issue"}
            </button>
          )}
        </form>
      </search>

      {items.error && <ErrorNotice error={items.error} retry={retry} />}
      {items.isPending && (
        <output className="lyra-workspace-empty">
          Loading {kind === "pulls" ? "pull requests" : "issues"}…
        </output>
      )}

      {items.data && (
        <div className="lyra-workspace-panel">
          <nav
            className="lyra-workspace-panelHeading lyra-item-list-states"
            aria-label="State filter"
          >
            <Link
              to={itemListPaths[kind]}
              params={{ owner, repo }}
              search={{ ...search, state: "open" }}
              aria-current={state === "open" ? "page" : undefined}
            >
              Open
            </Link>
            <Link
              to={itemListPaths[kind]}
              params={{ owner, repo }}
              search={{ ...search, state: "closed" }}
              aria-current={state === "closed" ? "page" : undefined}
            >
              Closed
            </Link>
            {kind === "pulls" && (
              <Link
                to={itemListPaths[kind]}
                params={{ owner, repo }}
                search={{ ...search, state: "merged" }}
                aria-current={state === "merged" ? "page" : undefined}
              >
                Merged
              </Link>
            )}
            <Link
              to={itemListPaths[kind]}
              params={{ owner, repo }}
              search={{ ...search, state: "all" }}
              aria-current={state === "all" ? "page" : undefined}
            >
              All
            </Link>
          </nav>
          {listedItems.map((item) => {
            const StateIcon =
              stateIcons[kind][item.state as keyof (typeof stateIcons)["issues"]] ||
              stateIcons[kind].open;
            return (
              <article className="lyra-workspace-row lyra-item-list-item" key={item.number}>
                <span
                  className="lyra-item-list-state"
                  data-kind={kind}
                  data-state={item.state}
                  title={item.state}
                >
                  <StateIcon size={16} />
                  <span className="lyra-controls-srOnly">{item.state}</span>
                </span>
                <div className="lyra-workspace-grow">
                  <div className="lyra-item-list-titleLine">
                    <Link
                      className="lyra-workspace-rowTitle"
                      to={itemDetailPaths[kind]}
                      params={{ owner, repo, number: item.number }}
                      search={{}}
                    >
                      {item.title}
                    </Link>
                    {item.labels.map((label) => {
                      const color = labels.data?.find((entry) => entry.name === label)?.color;
                      return (
                        <span
                          className="lyra-workspace-label"
                          key={label}
                          style={color ? ({ "--label": `#${color}` } as CSSProperties) : undefined}
                        >
                          {label}
                        </span>
                      );
                    })}
                  </div>
                  <p className="lyra-workspace-meta">
                    #{item.number} opened <DateTime value={item.created_at} /> by{" "}
                    <ActorName actor={item.author} />
                    {"head" in item && (
                      <>
                        {" "}
                        · <code>{item.head}</code> → <code>{item.base}</code>
                      </>
                    )}
                  </p>
                </div>
                {item.assignees.length > 0 && (
                  <span className="lyra-item-list-side" title="Assignees">
                    <PersonIcon size={16} />
                    {item.assignees.join(", ")}
                  </span>
                )}
                {item.comment_count > 0 && (
                  <span
                    className="lyra-item-list-side"
                    aria-label={`${item.comment_count} comments`}
                  >
                    <CommentIcon size={16} />
                    {item.comment_count}
                  </span>
                )}
              </article>
            );
          })}
          {items.data.length === 0 && (
            <p className="lyra-workspace-empty">
              No matching {kind === "pulls" ? "pull requests" : "issues"}.
            </p>
          )}
        </div>
      )}

      {items.data?.length === limit && limit < 1000 && (
        <div className="lyra-workspace-actions">
          <Link
            className="lyra-controls-button"
            to={itemListPaths[kind]}
            params={{ owner, repo }}
            search={{ ...search, limit: String(limit + 50) }}
          >
            Load more
          </Link>
        </div>
      )}

      {(kind !== "pulls" || metadata.data) && (
        <FormDialog
          id="new-item"
          title={kind === "pulls" ? "New pull request" : "New issue"}
          onSubmit={submit}
          pending={creation.isPending}
          error={creation.error}
          submitLabel={creation.isPending ? "Creating…" : "Create"}
        >
          {kind === "pulls" && (
            <div className="lyra-workspace-toolbar">
              <Field label="Base">
                <select
                  className="lyra-controls-select"
                  name="base"
                  defaultValue={metadata.data?.default_branch}
                >
                  {metadata.data?.branches.map((branch) => (
                    <option key={branch.name}>{branch.name}</option>
                  ))}
                </select>
              </Field>
              <Field label="Compare">
                <select
                  className="lyra-controls-select"
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
              </Field>
            </div>
          )}
          <Field label="Title">
            <input
              name="title"
              required
              maxLength={256}
              value={title.value}
              onChange={titleChanged}
              placeholder="What needs to change?"
            />
          </Field>
          <MarkdownEditor
            id="new-item-body"
            label="Description"
            value={description.value}
            onChange={description.setValue}
          />
          {kind === "pulls" && draft.available && (
            <button
              className="lyra-controls-button"
              type="button"
              onClick={draftDescription}
              disabled={draft.pending}
            >
              <SparkleIcon size={16} />
              Draft description from commits
            </button>
          )}
          <OnDeviceOutput
            title="Draft description"
            output={draft.output}
            progress={draft.progress}
            error={draft.error}
            pending={draft.pending}
            onStop={draft.stop}
            onDiscard={draft.reset}
          >
            <button className="lyra-controls-button" type="button" onClick={insertDraft}>
              Insert
            </button>
          </OnDeviceOutput>
          <Field label="Labels">
            <input name="labels" placeholder="bug, enhancement" list="available-labels" />
          </Field>
          <datalist id="available-labels">
            {labels.data?.map((label) => (
              <option key={label.name} value={label.name}>
                {label.name}
              </option>
            ))}
          </datalist>
          <Field label="Assignees">
            <input name="assignees" placeholder="Names, separated by commas" />
          </Field>
        </FormDialog>
      )}
    </>
  );
}
