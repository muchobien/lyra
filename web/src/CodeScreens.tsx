import { useEffect } from "react";
import { Link, useParams, useSearch } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Code, GitCommitHorizontal, Tags } from "lucide-react";
import {
  api,
  repositoryAPI,
  type Commit,
  type CommitWorkspace,
  type Repository,
  type Tag,
} from "./api";
import { CopyButton, ErrorNotice, ScrollNavigation } from "./Controls";
import { ChecksPanel, CommitRows, DateTime, Status } from "./Workspace";
import { DiffViewer } from "./DiffViewer";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";

type CodeNavigationProps = { owner: string; repo: string; reference: string; active: string };
export function CodeNavigation({ owner, repo, reference, active }: CodeNavigationProps) {
  return (
    <ScrollNavigation className={styles.tabs} label="Code sections" active={active}>
      <Link
        to="/$owner/$repo"
        params={{ owner, repo }}
        search={{ ref: reference }}
        aria-current={active === "files" ? "page" : undefined}
      >
        <Code size={15} />
        Files
      </Link>
      <Link
        to="/$owner/$repo/$section/$"
        params={{ owner, repo, section: "commits", _splat: reference }}
        search={{}}
        aria-current={active === "commits" ? "page" : undefined}
      >
        <GitCommitHorizontal size={15} />
        Commits
      </Link>
      <Link
        to="/$owner/$repo/$section"
        params={{ owner, repo, section: "tags" }}
        search={{}}
        aria-current={active === "tags" ? "page" : undefined}
      >
        <Tags size={15} />
        Tags
      </Link>
    </ScrollNavigation>
  );
}

type CodeScreenProps = { section: string; target: string };
export function CodeScreens({ section, target }: CodeScreenProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const name = `${owner}/${repo}`;
  const metadata = useQuery({
    queryKey: ["repository", name],
    queryFn: ({ signal }) => api<Repository>(repositoryAPI(name), signal),
  });
  const reference = target || metadata.data?.default_branch || "main";
  const limit = Math.max(30, Math.min(500, Number(search.limit) || 30));
  const commitsPath = repositoryAPI(
    name,
    `/commits?${new URLSearchParams({ ref: reference, limit: String(limit) })}`,
  );
  const commits = useQuery({
    queryKey: [commitsPath],
    queryFn: ({ signal }) => api<Commit[]>(commitsPath, signal),
    enabled: section === "commits" && !!metadata.data?.branches.length,
  });
  const tags = useQuery({
    queryKey: [repositoryAPI(name, "/tags")],
    queryFn: ({ signal }) => api<Tag[]>(repositoryAPI(name, "/tags"), signal),
    enabled: section === "tags",
  });
  const detailPath = repositoryAPI(name, `/commits/${encodeURIComponent(target)}/view`);
  const detail = useQuery({
    queryKey: [detailPath],
    queryFn: ({ signal }) => api<CommitWorkspace>(detailPath, signal),
    enabled: section === "commit" && !!target,
  });
  useEffect(() => {
    document.title = `${detail.data?.commit.subject || section} · ${name} · Lyra`;
  }, [detail.data?.commit.subject, section, name]);
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
      <div className={styles.panel}>
        {tags.data.map((tag) => (
          <article className={styles.row} key={tag.name}>
            <Tags size={18} />
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
                {tag.annotated ? "Annotated tag" : "Lightweight tag"}
                {tag.tagger && (
                  <>
                    {" "}
                    · {tag.tagger.name} · <DateTime value={tag.tagger.date} />
                  </>
                )}
              </p>
              {tag.message && <p className={styles.meta}>{tag.message.split("\n")[0]}</p>}
            </div>
            <Status value={tag.signature.status} />
            <Link
              className={styles.link}
              to="/$owner/$repo/$section/$"
              params={{ owner, repo, section: "commit", _splat: tag.target }}
              search={{}}
            >
              <code>{tag.target.slice(0, 7)}</code>
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
            <span className={styles.meta}>
              History of <code>{reference}</code>
            </span>
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
              {commit.author.name} committed <DateTime value={commit.committer.date} />
            </p>
          </div>
          <Status value={commit.signature.status} />
          <CopyButton value={commit.sha} label="Copy commit SHA" compact />
        </div>
        <div className={styles.panel}>
          <div className={styles.panelHeading}>
            <code>{commit.sha}</code>
            <Link
              className={styles.link}
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{ ref: commit.sha }}
            >
              Browse files
            </Link>
          </div>
          {commit.body && <pre className={styles.plainText}>{commit.body}</pre>}
          {commit.parents.length > 0 && (
            <p className={styles.meta} style={{ padding: "0 1.25rem 1rem" }}>
              Parents:{" "}
              {commit.parents.map((parent) => (
                <Link
                  key={parent}
                  className={styles.link}
                  to="/$owner/$repo/$section/$"
                  params={{ owner, repo, section: "commit", _splat: parent }}
                  search={{}}
                >
                  {parent.slice(0, 7)}{" "}
                </Link>
              ))}
            </p>
          )}
        </div>
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
      <CodeNavigation
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        active={section === "tags" ? "tags" : "commits"}
      />
      {content}
    </>
  );
}
