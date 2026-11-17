import { Link } from "@tanstack/react-router";
import { FileDirectoryIcon, TagIcon, VerifiedIcon } from "@primer/octicons-react";
import { CopyButton } from "@/components/controls/CopyButton";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { DateTime } from "@/components/metadata/DateTime";
import { Status } from "@/components/status/Status";
import { ChecksPanel } from "@/components/code/ChecksPanel";
import { CommitRows } from "@/components/code/CommitRows";
import { DiffViewer } from "@/components/code/DiffViewer";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "@/components/code/CommitRows/styles.css";
import "@/components/markdown/RichText/styles.css";
import { CodeBreadcrumb } from "@/components/code/CodeBreadcrumb";
import { useConnect, type CodeScreenProps } from "./connect";

export function CodeScreens({ section, target }: CodeScreenProps) {
  const {
    owner,
    repo,
    metadata,
    reference,
    limit,
    commits,
    tags,
    detail,
    branchChanged,
    retry,
    error,
  } = useConnect({ section, target });
  if (error) {
    return <ErrorNotice error={error} retry={retry} />;
  }
  let content;
  if (section === "tags") {
    content = tags.data ? (
      <div className="lyra-workspace-panel lyra-workspace-commits">
        {tags.data.map((tag) => (
          <article className="lyra-workspace-row" key={tag.name}>
            <TagIcon size={16} />
            <div className="lyra-workspace-grow">
              <Link
                className="lyra-workspace-rowTitle"
                to="/$owner/$repo"
                params={{ owner, repo }}
                search={{ ref: `refs/tags/${tag.name}` }}
              >
                {tag.name}
              </Link>
              <p className="lyra-workspace-meta">
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
              <span className="lyra-workspace-verified" title="Verified">
                <VerifiedIcon size={16} />
                <span className="lyra-controls-srOnly">Verified</span>
              </span>
            )}
            {!["verified", "unverified"].includes(tag.signature.status) && (
              <Status value={tag.signature.status} />
            )}
            <Link
              className="lyra-workspace-sha"
              to="/$owner/$repo/commit/$sha"
              params={{ owner, repo, sha: tag.target }}
              search={{}}
            >
              {tag.target.slice(0, 7)}
            </Link>
            <Link
              className="lyra-workspace-browse"
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
        {tags.data.length === 0 && <p className="lyra-workspace-empty">No tags yet.</p>}
      </div>
    ) : (
      <output className="lyra-workspace-empty">Loading tags…</output>
    );
  } else if (section === "commits") {
    if (metadata.data?.branches.length === 0) {
      content = (
        <div className="lyra-workspace-panel">
          <p className="lyra-workspace-empty">No commits yet.</p>
        </div>
      );
    } else if (commits.data) {
      content = (
        <>
          <div className="lyra-workspace-toolbar">
            <label className="lyra-controls-srOnly" htmlFor="commits-branch">
              Branch
            </label>
            <select
              id="commits-branch"
              className="lyra-controls-select"
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
            <div className="lyra-workspace-actions">
              <Link
                className="lyra-controls-button"
                to="/$owner/$repo/commits/$"
                params={{ owner, repo, _splat: reference }}
                search={{ limit: String(limit + 30) }}
              >
                Load more
              </Link>
            </div>
          )}
          {commits.data.length >= 500 && commits.data.at(-1)?.parents[0] && (
            <Link
              className="lyra-controls-button"
              to="/$owner/$repo/commits/$"
              params={{
                owner,
                repo,
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
      content = <output className="lyra-workspace-empty">Loading commits…</output>;
    }
  } else if (detail.data) {
    const { commit, checks, files } = detail.data;
    content = (
      <>
        <div className="lyra-workspace-heading">
          <div className="lyra-workspace-grow">
            <h1 className="lyra-workspace-title">{commit.subject}</h1>
            <p className="lyra-workspace-meta">
              <strong>{commit.author.name}</strong> committed{" "}
              <DateTime value={commit.committer.date} /> · <code>{commit.sha.slice(0, 7)}</code>
              {commit.parents.length > 0 && (
                <>
                  {" "}
                  · {commit.parents.length === 1 ? "parent" : "parents"}{" "}
                  {commit.parents.map((parent) => (
                    <Link
                      key={parent}
                      className="lyra-workspace-link"
                      to="/$owner/$repo/commit/$sha"
                      params={{ owner, repo, sha: parent }}
                      search={{}}
                    >
                      <code>{parent.slice(0, 7)}</code>{" "}
                    </Link>
                  ))}
                </>
              )}
            </p>
          </div>
          <div className="lyra-workspace-headingActions">
            <Status value={commit.signature.status} />
            <Link
              className="lyra-controls-button"
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
          <div className="lyra-workspace-panel">
            <pre className="lyra-workspace-plainText">{commit.body.trim()}</pre>
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
    content = <output className="lyra-workspace-empty">Loading commit…</output>;
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
