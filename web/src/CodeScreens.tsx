import {
  useGetRepo,
  useListCommits,
  useListTags,
  useGetCommitWorkspace,
} from "./api/generated/lyra";
import { useEffect } from "react";
import type { ChangeEvent, ReactNode } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { FileDirectoryIcon, TagIcon, VerifiedIcon } from "@primer/octicons-react";
import { CopyButton, ErrorNotice } from "./Controls";
import { ChecksPanel, CommitRows, DateTime, Status } from "./Workspace";
import { DiffViewer } from "./DiffViewer";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

type CodeBreadcrumbProps = {
  owner: string;
  repo: string;
  reference: string;
  section: string;
  label: string;
  current?: string;
  children?: ReactNode;
};
export function CodeBreadcrumb({
  owner,
  repo,
  reference,
  section,
  label,
  current,
  children,
}: CodeBreadcrumbProps) {
  return (
    <header className={styles.crumbs}>
      <nav aria-label="Breadcrumb">
        <Link to="/$owner/$repo" params={{ owner, repo }} search={{}}>
          {repo}
        </Link>
        <span>/</span>
        {current && section === "commits" && (
          <Link
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section, _splat: reference }}
            search={{}}
          >
            {label}
          </Link>
        )}
        {current && section !== "commits" && (
          <Link to="/$owner/$repo/$section" params={{ owner, repo, section }} search={{}}>
            {label}
          </Link>
        )}
        {current ? (
          <>
            <span>/</span>
            <strong>{current}</strong>
          </>
        ) : (
          <strong>{label}</strong>
        )}
      </nav>
      {children}
    </header>
  );
}

type CodeScreenProps = { section: string; target: string };
export function CodeScreens({ section, target }: CodeScreenProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const name = `${owner}/${repo}`;
  const metadata = useGetRepo(owner, repo);
  const navigate = useNavigate();
  const reference = target || metadata.data?.default_branch || "main";
  const limit = Math.max(30, Math.min(500, Number(search.limit) || 30));
  const commits = useListCommits(
    owner,
    repo,
    { ref: reference, limit },
    { query: { enabled: section === "commits" && !!metadata.data?.branches.length } },
  );
  const tags = useListTags(owner, repo, { query: { enabled: section === "tags" } });
  const detail = useGetCommitWorkspace(owner, repo, target, {
    query: { enabled: section === "commit" && !!target },
  });
  useEffect(() => {
    document.title = `${detail.data?.commit.subject || section} · ${name} · Lyra`;
  }, [detail.data?.commit.subject, section, name]);
  function branchChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({
      to: "/$owner/$repo/$section/$",
      params: { owner, repo, section: "commits", _splat: event.target.value },
      search: {},
    });
  }
  function retry() {
    if (section === "commits") {
      void commits.refetch();
    }
    if (section === "commit") {
      void detail.refetch();
    }
    if (section === "tags") {
      void tags.refetch();
    }
    void metadata.refetch();
  }
  const error = metadata.error || commits.error || tags.error || detail.error;
  if (error) {
    return <ErrorNotice error={error} retry={retry} />;
  }
  let content;
  if (section === "tags") {
    content = tags.data ? (
      <div className={`${styles.panel} ${styles.commits}`}>
        {tags.data.map((tag) => (
          <article className={styles.row} key={tag.name}>
            <TagIcon size={16} />
            <div className={styles.grow}>
              <Link
                className={styles.rowTitle}
                to="/$owner/$repo"
                params={{ owner, repo }}
                search={{ ref: `refs/tags/${tag.name}` }}
              >
                {tag.name}
              </Link>
              <p className={styles.meta}>
                {tag.tagger ? (
                  <>
                    <strong>{tag.tagger.name}</strong> tagged <DateTime value={tag.tagger.date} />
                  </>
                ) : (
                  "Lightweight tag"
                )}
                {tag.message && tag.message.split("\n")[0] !== tag.name && (
                  <> · {tag.message.split("\n")[0]}</>
                )}
              </p>
            </div>
            {tag.signature.status === "verified" && (
              <span className={styles.verified} title="Verified">
                <VerifiedIcon size={16} />
                <span className={controls.srOnly}>Verified</span>
              </span>
            )}
            {!["verified", "unverified"].includes(tag.signature.status) && (
              <Status value={tag.signature.status} />
            )}
            <Link
              className={styles.sha}
              to="/$owner/$repo/$section/$"
              params={{ owner, repo, section: "commit", _splat: tag.target }}
              search={{}}
            >
              {tag.target.slice(0, 7)}
            </Link>
            <Link
              className={styles.browse}
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{ ref: `refs/tags/${tag.name}` }}
              aria-label={`Browse files at ${tag.name}`}
              title="Browse files"
            >
              <FileDirectoryIcon size={16} />
            </Link>
          </article>
        ))}
        {tags.data.length === 0 && <p className={styles.empty}>No tags yet.</p>}
      </div>
    ) : (
      <output className={styles.empty}>Loading tags…</output>
    );
  } else if (section === "commits") {
    if (metadata.data?.branches.length === 0) {
      content = (
        <div className={styles.panel}>
          <p className={styles.empty}>No commits yet.</p>
        </div>
      );
    } else if (commits.data) {
      content = (
        <>
          <div className={styles.toolbar}>
            <label className={controls.srOnly} htmlFor="commits-branch">
              Branch
            </label>
            <select
              id="commits-branch"
              className={controls.select}
              value={reference}
              onChange={branchChanged}
            >
              {!metadata.data?.branches.some((branch) => branch.name === reference) && (
                <option value={reference}>{reference.slice(0, 12)}</option>
              )}
              {metadata.data?.branches.map((branch) => (
                <option key={branch.name} value={branch.name}>
                  {branch.name}
                </option>
              ))}
            </select>
          </div>
          <CommitRows commits={commits.data} owner={owner} repo={repo} />
          {commits.data.length === limit && limit < 500 && (
            <div className={styles.actions}>
              <Link
                className={controls.button}
                to="/$owner/$repo/$section/$"
                params={{ owner, repo, section, _splat: reference }}
                search={{ limit: String(limit + 30) }}
              >
                Load more
              </Link>
            </div>
          )}
          {commits.data.length >= 500 && commits.data.at(-1)?.parents[0] && (
            <Link
              className={controls.button}
              to="/$owner/$repo/$section/$"
              params={{
                owner,
                repo,
                section,
                _splat: commits.data.at(-1)?.parents[0] || reference,
              }}
              search={{}}
            >
              Earlier history
            </Link>
          )}
        </>
      );
    } else {
      content = <output className={styles.empty}>Loading commits…</output>;
    }
  } else if (detail.data) {
    const { commit, checks, files } = detail.data;
    content = (
      <>
        <div className={styles.heading}>
          <div className={styles.grow}>
            <h1 className={styles.title}>{commit.subject}</h1>
            <p className={styles.meta}>
              <strong>{commit.author.name}</strong> committed{" "}
              <DateTime value={commit.committer.date} /> · <code>{commit.sha.slice(0, 7)}</code>
              {commit.parents.length > 0 && (
                <>
                  {" "}
                  · {commit.parents.length === 1 ? "parent" : "parents"}{" "}
                  {commit.parents.map((parent) => (
                    <Link
                      key={parent}
                      className={styles.link}
                      to="/$owner/$repo/$section/$"
                      params={{ owner, repo, section: "commit", _splat: parent }}
                      search={{}}
                    >
                      <code>{parent.slice(0, 7)}</code>{" "}
                    </Link>
                  ))}
                </>
              )}
            </p>
          </div>
          <div className={styles.headingActions}>
            <Status value={commit.signature.status} />
            <Link
              className={controls.button}
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{ ref: commit.sha }}
            >
              <FileDirectoryIcon size={16} />
              Browse files
            </Link>
            <CopyButton value={commit.sha} label="Copy commit SHA" compact />
          </div>
        </div>
        {commit.body.trim() && (
          <div className={styles.panel}>
            <pre className={styles.plainText}>{commit.body.trim()}</pre>
          </div>
        )}
        <ChecksPanel checks={checks} owner={owner} repo={repo} />
        <DiffViewer
          key={commit.sha}
          files={files}
          owner={owner}
          repo={repo}
          section="commit"
          target={target}
          head={commit.sha}
        />
      </>
    );
  } else {
    content = <output className={styles.empty}>Loading commit…</output>;
  }
  return (
    <>
      <CodeBreadcrumb
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        section={section === "tags" ? "tags" : "commits"}
        label={section === "tags" ? "Tags" : "Commits"}
        current={section === "commit" ? target.slice(0, 7) : undefined}
      />
      {content}
    </>
  );
}
