import { Link } from "@tanstack/react-router";
import {
  FileZipIcon,
  GitBranchIcon,
  HistoryIcon,
  PackageIcon,
  SearchIcon,
  TagIcon,
  TerminalIcon,
  VerifiedIcon,
  VersionsIcon,
  XIcon,
} from "@primer/octicons-react";
import { getGetRepositoryRawFileUrl } from "@/api/generated/lyra";
import { CopyButton } from "@/components/controls/CopyButton";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { FileIcon } from "@/components/code/FileIcon";
import { DateTime } from "@/components/metadata/DateTime";
import "@/styles/controls.css";
import "./styles.css";
import { useConnect, type RepositoryPageProps } from "./connect";

export function RepositoryPage({ view, target, plain = false }: RepositoryPageProps) {
  const {
    owner,
    repo,
    path,
    name,
    metadata,
    hasBranches,
    ref,
    tree,
    blobParams,
    blob,
    latest,
    images,
    tags,
    releases,
    filter,
    files,
    visibleFiles,
    branchChanged,
    searchChanged,
    finderToggled,
    closeFinder,
    retry,
    retryFiles,
    archiveURL,
    failure,
  } = useConnect({ view, target });
  if (failure) {
    return <ErrorNotice error={failure} retry={retry} />;
  }
  if (!metadata.data) {
    return <output className="lyra-repository-page-loading">Loading repository…</output>;
  }

  const parts = path.split("/").filter(Boolean);
  const latestImages = latest.data?.images || [];
  const containers = latestImages.length
    ? latestImages
    : (images.data || [])
        .toSorted((first, second) => Date.parse(second.pushed_at) - Date.parse(first.pushed_at))
        .slice(0, 1);
  const rawURL = getGetRepositoryRawFileUrl(owner, repo, blobParams);

  return (
    <>
      <div className="lyra-repository-page-workspace">
        <section className="lyra-repository-page-files">
          <div className="lyra-repository-page-toolbar">
            <div className="lyra-repository-page-refs">
              <label className="lyra-controls-srOnly" htmlFor="branch-select">
                Branch
              </label>
              <select
                id="branch-select"
                className="lyra-controls-select"
                value={ref}
                onChange={branchChanged}
              >
                {!metadata.data.branches.some((branch) => branch.name === ref) && (
                  <option value={ref}>{ref}</option>
                )}
                {metadata.data.branches.map((branch) => (
                  <option key={branch.name} value={branch.name}>
                    {branch.name}
                  </option>
                ))}
              </select>
              <Link to="/$owner/$repo/branches" params={{ owner, repo }} search={{}}>
                <GitBranchIcon size={16} />
                <strong>{metadata.data.branches.length}</strong>
                {metadata.data.branches.length === 1 ? "branch" : "branches"}
              </Link>
              {tags.data && (
                <Link to="/$owner/$repo/tags" params={{ owner, repo }} search={{}}>
                  <TagIcon size={16} />
                  <strong>{tags.data.length}</strong>
                  {tags.data.length === 1 ? "tag" : "tags"}
                </Link>
              )}
            </div>
            <div className="lyra-repository-page-toolbarActions">
              <button
                className="lyra-controls-button"
                type="button"
                commandfor="file-finder"
                command="show-modal"
                disabled={!hasBranches}
              >
                <SearchIcon size={16} />
                Go to file <kbd>T</kbd>
              </button>
              <button
                className="lyra-controls-button lyra-repository-page-cloneTrigger"
                type="button"
                popoverTarget="clone-options"
              >
                <TerminalIcon size={16} />
                Clone
              </button>
            </div>
          </div>
          {path && (
            <nav className="lyra-repository-page-breadcrumb" aria-label="File path">
              <Link to="/$owner/$repo/tree/$" params={{ owner, repo, _splat: ref }}>
                {repo}
              </Link>
              {parts.map((part, index) => (
                <span key={parts.slice(0, index + 1).join("/")}>
                  <span>/</span>
                  {view === "blob" && index === parts.length - 1 ? (
                    <strong>{part}</strong>
                  ) : (
                    <Link
                      to="/$owner/$repo/tree/$"
                      params={{
                        owner,
                        repo,
                        _splat: [ref, ...parts.slice(0, index + 1)].join("/"),
                      }}
                    >
                      {part}
                    </Link>
                  )}
                </span>
              ))}
            </nav>
          )}
          {view === "tree" && tree.isPending && (
            <div
              className="lyra-repository-page-skeleton"
              aria-label="Loading files"
              aria-busy="true"
            />
          )}
          {view === "tree" && tree.data?.empty && (
            <div className="lyra-repository-page-empty">
              <GitBranchIcon size={24} />
              <h1>Your repository is ready</h1>
              <p>Push your first commit to start browsing files.</p>
              <div className="lyra-repository-page-command">
                <code>git clone {metadata.data.clone_url}</code>
                <CopyButton
                  value={`git clone ${metadata.data.clone_url}`}
                  label="Copy clone command"
                  compact
                />
              </div>
            </div>
          )}
          {view === "tree" && tree.data && !tree.data.empty && (
            <>
              <div className="lyra-repository-page-tree">
                {tree.data.last && (
                  <div className="lyra-repository-page-commit">
                    <span>
                      <strong>{tree.data.last.committer.name}</strong>
                      <Link
                        to="/$owner/$repo/commit/$sha"
                        params={{ owner, repo, sha: tree.data.last.sha }}
                        search={{}}
                      >
                        {tree.data.last.subject}
                      </Link>
                    </span>
                    {tree.data.last.signature.status === "verified" && (
                      <span className="lyra-repository-page-verified" title="Verified">
                        <VerifiedIcon size={16} />
                        <span className="lyra-controls-srOnly">Verified</span>
                      </span>
                    )}
                    <Link
                      to="/$owner/$repo/commit/$sha"
                      params={{ owner, repo, sha: tree.data.last.sha }}
                      search={{}}
                      className="lyra-repository-page-sha"
                    >
                      {tree.data.last.sha.slice(0, 7)}
                    </Link>
                    <DateTime value={tree.data.last.committer.date} />
                    <Link
                      className="lyra-repository-page-history"
                      to="/$owner/$repo/commits/$"
                      params={{ owner, repo, _splat: ref }}
                      search={{}}
                    >
                      <HistoryIcon size={16} />
                      History
                    </Link>
                  </div>
                )}
                {tree.data.entries
                  .toSorted((first, second) => {
                    if (first.type === second.type) {
                      return first.name.localeCompare(second.name);
                    }
                    return first.type === "tree" ? -1 : 1;
                  })
                  .map((entry) => (
                    <div className="lyra-repository-page-entry" key={entry.name}>
                      <FileIcon name={entry.name} folder={entry.type === "tree"} />
                      <Link
                        to={entry.type === "tree" ? "/$owner/$repo/tree/$" : "/$owner/$repo/blob/$"}
                        params={{
                          owner,
                          repo,
                          _splat: [ref, path, entry.name].filter(Boolean).join("/"),
                        }}
                      >
                        {entry.name}
                      </Link>
                      <span>
                        {entry.type === "blob"
                          ? new Intl.NumberFormat(undefined, {
                              style: "unit",
                              unit: "byte",
                              unitDisplay: "narrow",
                              notation: "compact",
                            }).format(entry.size)
                          : ""}
                      </span>
                    </div>
                  ))}
                {tree.data.entries.length === 0 && (
                  <p className="lyra-repository-page-loading">This directory is empty.</p>
                )}
              </div>
              {tree.data.readme && (
                <section className="lyra-repository-page-readme">
                  <div className="lyra-repository-page-fileHeading">
                    <FileIcon name="README.md" />
                    README
                  </div>
                  <div
                    className="lyra-repository-page-rendered"
                    dangerouslySetInnerHTML={{ __html: tree.data.readme }}
                  />
                </section>
              )}
            </>
          )}
          {view === "blob" && blob.isPending && (
            <output className="lyra-repository-page-loading">Loading file…</output>
          )}
          {view === "blob" && blob.data && (
            <section className="lyra-repository-page-blob">
              <div className="lyra-repository-page-fileHeading">
                {blob.data.markdown && (
                  <nav className="lyra-repository-page-views" aria-label="File view">
                    <Link
                      to="/$owner/$repo/blob/$"
                      params={{ owner, repo, _splat: target }}
                      search={{}}
                      activeOptions={{ exact: true }}
                    >
                      Preview
                    </Link>
                    <Link
                      to="/$owner/$repo/blob/$"
                      params={{ owner, repo, _splat: target }}
                      search={{ plain: "1" }}
                      activeOptions={{ exact: true }}
                    >
                      Code
                    </Link>
                  </nav>
                )}
                <FileIcon name={parts.at(-1) ?? ""} />
                <strong>{parts.at(-1)}</strong>
                <span>
                  {new Intl.NumberFormat(undefined, {
                    style: "unit",
                    unit: "byte",
                    unitDisplay: "narrow",
                    notation: "compact",
                  }).format(blob.data.size)}
                </span>
                <a href={rawURL}>Raw</a>
              </div>
              {blob.data.binary ? (
                <p className="lyra-repository-page-loading">
                  Binary or large file. <a href={rawURL}>Download raw file</a>
                </p>
              ) : (
                <div
                  className="lyra-repository-page-rendered"
                  dangerouslySetInnerHTML={{
                    __html: blob.data.markdown && !plain ? blob.data.markdown : blob.data.code,
                  }}
                />
              )}
            </section>
          )}
        </section>
        <aside className="lyra-repository-page-sidebar">
          <section className="lyra-repository-page-about">
            <h2>About</h2>
            {metadata.data.description && <p>{metadata.data.description}</p>}
            <ul className="lyra-repository-page-stats">
              <li>
                <Link to="/$owner/$repo/releases" params={{ owner, repo }} search={{}}>
                  <VersionsIcon size={16} />
                  <strong>{releases.data?.length ?? 0}</strong>
                  {releases.data?.length === 1 ? "release" : "releases"}
                </Link>
              </li>
              <li>
                <Link to="/$owner/$repo/images" params={{ owner, repo }} search={{}}>
                  <PackageIcon size={16} />
                  <strong>{images.data?.length ?? 0}</strong>
                  {images.data?.length === 1 ? "container" : "containers"}
                </Link>
              </li>
            </ul>
          </section>
          <section className="lyra-repository-page-delivery">
            <div className="lyra-repository-page-sidebarHeading">
              <h2>Releases</h2>
              <Link to="/$owner/$repo/releases" params={{ owner, repo }} search={{}}>
                View all
              </Link>
            </div>
            {latest.isPending && (
              <output className="lyra-repository-page-secondary">Loading latest release…</output>
            )}
            {latest.error && (
              <p className="lyra-repository-page-secondary">Releases unavailable.</p>
            )}
            {latest.data === null && (
              <p className="lyra-repository-page-secondary">No stable release yet.</p>
            )}
            {latest.data && (
              <>
                <Link
                  className="lyra-repository-page-releaseVersion"
                  to="/$owner/$repo/releases/tag/$"
                  params={{ owner, repo, _splat: `${latest.data.tag}` }}
                  search={{}}
                >
                  <TagIcon size={16} />
                  <strong>{latest.data.title || latest.data.tag}</strong>
                  <span>Latest</span>
                </Link>
                <time
                  className="lyra-repository-page-secondary"
                  dateTime={latest.data.published_at || latest.data.created_at}
                >
                  {new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
                    new Date(latest.data.published_at || latest.data.created_at),
                  )}
                </time>
              </>
            )}
            {containers.length > 0 && (
              <div className="lyra-repository-page-containers">
                <div className="lyra-repository-page-sidebarHeading">
                  <h2>Containers</h2>
                  <Link to="/$owner/$repo/images" params={{ owner, repo }} search={{}}>
                    View all
                  </Link>
                </div>
                <p className="lyra-repository-page-secondary">
                  {latestImages.length ? `Linked to ${latest.data?.tag}` : "Latest container"}
                </p>
                {containers.map((image) => (
                  <div className="lyra-repository-page-image" key={`${image.name}:${image.tag}`}>
                    <Link to="/$owner/$repo/images" params={{ owner, repo }} search={{}}>
                      <PackageIcon size={16} />
                      <span>
                        {image.name}
                        <strong>:{image.tag}</strong>
                      </span>
                    </Link>
                    <CopyButton
                      value={`docker pull ${image.reference}`}
                      label={`Copy pull command for ${image.name}:${image.tag}`}
                      compact
                    />
                  </div>
                ))}
              </div>
            )}
          </section>
        </aside>
      </div>
      <div id="clone-options" popover="auto" className="lyra-repository-page-clonePopover">
        <h2>Clone repository</h2>
        <div className="lyra-repository-page-command">
          <code>{metadata.data.clone_url}</code>
          <CopyButton value={metadata.data.clone_url} label="Copy clone URL" compact />
        </div>
        {hasBranches && (
          <div className="lyra-repository-page-downloads">
            <a className="lyra-controls-button" href={`${archiveURL}.zip`} download>
              <FileZipIcon size={16} />
              Download ZIP
            </a>
            <a className="lyra-controls-button" href={`${archiveURL}.tar.gz`} download>
              <FileZipIcon size={16} />
              Download tar.gz
            </a>
          </div>
        )}
      </div>
      <dialog
        id="file-finder"
        closedby="any"
        className="lyra-repository-page-finder"
        onToggle={finderToggled}
        aria-labelledby="finder-title"
      >
        <div className="lyra-repository-page-dialogHeading">
          <h2 id="finder-title">Go to file</h2>
        </div>
        <label htmlFor="file-search" className="lyra-controls-srOnly">
          Search file paths
        </label>
        <input
          id="file-search"
          type="search"
          placeholder="Type a file name or path…"
          value={filter}
          onChange={searchChanged}
          autoComplete="off"
        />
        <button
          className="lyra-controls-button lyra-repository-page-dialogClose"
          type="button"
          commandfor="file-finder"
          command="close"
          aria-label="Close file finder"
        >
          <XIcon size={16} />
        </button>
        <div className="lyra-repository-page-finderResults">
          {files.isPending && <output>Loading files…</output>}
          {files.error && <ErrorNotice error={files.error} retry={retryFiles} />}{" "}
          {visibleFiles.slice(0, 100).map((filePath) => (
            <Link
              key={filePath}
              to="/$owner/$repo/blob/$"
              params={{ owner, repo, _splat: `${ref}/${filePath}` }}
              onClick={closeFinder}
            >
              <FileIcon name={filePath.split("/").at(-1) ?? filePath} />
              {filePath}
            </Link>
          ))}
          {files.data && visibleFiles.length === 0 && <p>No matching files.</p>}
        </div>
        <p className="lyra-repository-page-finderHint">
          {name}
          <span>Esc to close</span>
        </p>
      </dialog>
      <link rel="stylesheet" href="/api/code.css" />
    </>
  );
}
