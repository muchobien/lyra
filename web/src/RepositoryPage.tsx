import { useEffect, useState } from "react";
import type { ChangeEvent, ToggleEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import {
  Search,
  Terminal,
  Folder,
  File,
  Tags,
  Package,
  X,
  ShieldCheck,
  GitBranch,
} from "lucide-react";
import { APIError } from "./api";
import {
  useGetRepo,
  useGetRepositoryTree,
  useGetRepositoryBlob,
  getLatestRelease,
  getGetLatestReleaseQueryKey,
  useListImages,
  useListRepositoryFiles,
  getGetRepositoryRawFileUrl,
} from "./api/generated/lyra";
import { CopyButton, ErrorNotice } from "./Controls";
import { CodeNavigation } from "./CodeScreens";
import controls from "./Controls.module.css";
import styles from "./RepositoryPage.module.css";

export function RepositoryPage() {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: "/$owner/$repo" });
  const navigate = useNavigate({ from: "/$owner/$repo" });
  const name = `${owner}/${repo}`;
  const metadata = useGetRepo(owner, repo);
  const hasBranches = Boolean(metadata.data?.branches.length);
  const ref = search.ref || metadata.data?.default_branch || "main";
  const params = { ref, path: search.path || "" };
  const tree = useGetRepositoryTree(owner, repo, params, {
    query: { enabled: !!metadata.data && !search.file },
  });
  const blobParams = { ref, path: search.file || "" };
  const blob = useGetRepositoryBlob(owner, repo, blobParams, {
    query: { enabled: !!metadata.data && !!search.file },
  });
  const latest = useQuery({
    queryKey: getGetLatestReleaseQueryKey(owner, repo),
    queryFn: async ({ signal }) => {
      try {
        return await getLatestRelease(owner, repo, { signal });
      } catch (error) {
        if (error instanceof APIError && error.status === 404) {
          return null;
        }
        throw error;
      }
    },
  });
  const images = useListImages(owner, repo);
  const [finderOpen, setFinderOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const files = useListRepositoryFiles(owner, repo, { ref }, { query: { enabled: finderOpen } });
  const visibleFiles =
    files.data?.filter((path) => path.toLowerCase().includes(filter.toLowerCase())) || [];

  useEffect(() => {
    document.title = `${search.file || search.path || name} · Lyra`;
  }, [name, search.file, search.path]);

  useEffect(() => {
    function shortcut(event: KeyboardEvent) {
      if (
        !hasBranches ||
        event.key.toLowerCase() !== "t" ||
        event.metaKey ||
        event.ctrlKey ||
        event.altKey
      ) {
        return;
      }
      if (
        event.target instanceof HTMLElement &&
        event.target.closest("input, textarea, select, [contenteditable=true], dialog")
      ) {
        return;
      }
      const dialog = document.getElementById("file-finder");
      if (dialog instanceof HTMLDialogElement) {
        event.preventDefault();
        dialog.showModal();
      }
    }
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, [hasBranches]);

  function branchChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({ search: { ref: event.target.value } });
  }
  function searchChanged(event: ChangeEvent<HTMLInputElement>) {
    setFilter(event.target.value);
  }
  function finderToggled(event: ToggleEvent<HTMLDialogElement>) {
    setFinderOpen(event.newState === "open");
  }
  function closeFinder() {
    const dialog = document.getElementById("file-finder");
    if (dialog instanceof HTMLDialogElement) {
      dialog.close();
    }
    setFilter("");
  }
  function retry() {
    void metadata.refetch();
    void tree.refetch();
    if (search.file) {
      void blob.refetch();
    }
  }
  function retryFiles() {
    void files.refetch();
  }

  const failure = metadata.error || (search.file ? blob.error : tree.error);
  if (failure) {
    return <ErrorNotice error={failure} retry={retry} />;
  }
  if (!metadata.data) {
    return <output className={styles.loading}>Loading repository…</output>;
  }

  const activePath = search.file || search.path || "";
  const parts = activePath.split("/").filter(Boolean);
  const latestImages = latest.data?.images || [];
  const containers = latestImages.length
    ? latestImages
    : (images.data || [])
        .toSorted((first, second) => Date.parse(second.pushed_at) - Date.parse(first.pushed_at))
        .slice(0, 1);
  const rawURL = getGetRepositoryRawFileUrl(owner, repo, blobParams);

  return (
    <>
      <CodeNavigation owner={owner} repo={repo} reference={ref} active="files" />
      <div className={styles.workspace}>
        <section className={styles.files}>
          <div className={styles.toolbar}>
            <label className={controls.srOnly} htmlFor="branch-select">
              Branch
            </label>
            <select
              id="branch-select"
              className={controls.select}
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
            <div className={styles.toolbarActions}>
              <button
                className={controls.button}
                type="button"
                commandfor="file-finder"
                command="show-modal"
                disabled={!hasBranches}
              >
                <Search size={15} />
                Go to file <kbd>T</kbd>
              </button>
              <button
                className={`${controls.button} ${styles.cloneTrigger}`}
                type="button"
                popoverTarget="clone-options"
              >
                <Terminal size={15} />
                Clone
              </button>
            </div>
          </div>
          {activePath && (
            <nav className={styles.breadcrumb} aria-label="File path">
              <Link to="/$owner/$repo" params={{ owner, repo }} search={{ ref: search.ref }}>
                {repo}
              </Link>
              {parts.map((part, index) => (
                <span key={parts.slice(0, index + 1).join("/")}>
                  <span>/</span>
                  {search.file && index === parts.length - 1 ? (
                    <strong>{part}</strong>
                  ) : (
                    <Link
                      to="/$owner/$repo"
                      params={{ owner, repo }}
                      search={{ ref: search.ref, path: parts.slice(0, index + 1).join("/") }}
                    >
                      {part}
                    </Link>
                  )}
                </span>
              ))}
            </nav>
          )}
          {!search.file && tree.isPending && (
            <div className={styles.skeleton} aria-label="Loading files" aria-busy="true" />
          )}
          {!search.file && tree.data?.empty && (
            <div className={styles.empty}>
              <GitBranch size={28} />
              <h1>Your repository is ready</h1>
              <p>Push your first commit to start browsing files.</p>
              <div className={styles.command}>
                <code>git clone {metadata.data.clone_url}</code>
                <CopyButton
                  value={`git clone ${metadata.data.clone_url}`}
                  label="Copy clone command"
                  compact
                />
              </div>
            </div>
          )}
          {!search.file && tree.data && !tree.data.empty && (
            <>
              <div className={styles.tree}>
                {tree.data.last && (
                  <div className={styles.commit}>
                    <span>
                      <strong>{tree.data.last.committer.name}</strong>
                      <Link
                        to="/$owner/$repo/$section/$"
                        params={{ owner, repo, section: "commit", _splat: tree.data.last.sha }}
                        search={{}}
                      >
                        {tree.data.last.subject}
                      </Link>
                    </span>
                    {tree.data.last.signature.status === "verified" && (
                      <span className={styles.verified}>
                        <ShieldCheck size={14} />
                        Verified
                      </span>
                    )}
                    <Link
                      to="/$owner/$repo/$section/$"
                      params={{ owner, repo, section: "commit", _splat: tree.data.last.sha }}
                      search={{}}
                      className={styles.sha}
                    >
                      {tree.data.last.sha.slice(0, 7)}
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
                    <div className={styles.entry} key={entry.name}>
                      {entry.type === "tree" ? <Folder size={16} /> : <File size={16} />}
                      <Link
                        to="/$owner/$repo"
                        params={{ owner, repo }}
                        search={{
                          ref: search.ref,
                          path: search.path,
                          file: undefined,
                          ...(entry.type === "tree"
                            ? { path: [search.path, entry.name].filter(Boolean).join("/") }
                            : { file: [search.path, entry.name].filter(Boolean).join("/") }),
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
                  <p className={styles.loading}>This directory is empty.</p>
                )}
              </div>
              {tree.data.readme && (
                <section className={styles.readme}>
                  <div className={styles.fileHeading}>
                    <File size={15} />
                    README
                  </div>
                  <div
                    className={styles.rendered}
                    dangerouslySetInnerHTML={{ __html: tree.data.readme }}
                  />
                </section>
              )}
            </>
          )}
          {search.file && blob.isPending && (
            <output className={styles.loading}>Loading file…</output>
          )}
          {search.file && blob.data && (
            <section className={styles.blob}>
              <div className={styles.fileHeading}>
                <File size={15} />
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
                <p className={styles.loading}>
                  Binary or large file. <a href={rawURL}>Download raw file</a>
                </p>
              ) : (
                <div
                  className={styles.rendered}
                  dangerouslySetInnerHTML={{ __html: blob.data.code }}
                />
              )}
            </section>
          )}
        </section>
        <aside className={styles.sidebar}>
          <section className={styles.about}>
            <h2>About</h2>
            {metadata.data.description && <p>{metadata.data.description}</p>}
          </section>
          <section className={styles.delivery}>
            <div className={styles.sidebarHeading}>
              <h2>Releases</h2>
              <Link
                to="/$owner/$repo/$section"
                params={{ owner, repo, section: "releases" }}
                search={{}}
              >
                View all
              </Link>
            </div>
            {latest.isPending && (
              <output className={styles.secondary}>Loading latest release…</output>
            )}
            {latest.error && <p className={styles.secondary}>Releases unavailable.</p>}
            {latest.data === null && <p className={styles.secondary}>No stable release yet.</p>}
            {latest.data && (
              <>
                <Link
                  className={styles.releaseVersion}
                  to="/$owner/$repo/$section/$"
                  params={{ owner, repo, section: "releases", _splat: `tag/${latest.data.tag}` }}
                  search={{}}
                >
                  <Tags size={17} />
                  <strong>{latest.data.tag}</strong>
                  <span>Latest</span>
                </Link>
                <Link
                  className={styles.releaseTitle}
                  to="/$owner/$repo/$section/$"
                  params={{ owner, repo, section: "releases", _splat: `tag/${latest.data.tag}` }}
                  search={{}}
                >
                  {latest.data.title}
                </Link>
                <time
                  className={styles.secondary}
                  dateTime={latest.data.published_at || latest.data.created_at}
                >
                  {new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
                    new Date(latest.data.published_at || latest.data.created_at),
                  )}
                </time>
              </>
            )}
            {containers.length > 0 && (
              <div className={styles.containers}>
                <div className={styles.sidebarHeading}>
                  <h2>Containers</h2>
                  <Link
                    to="/$owner/$repo/$section"
                    params={{ owner, repo, section: "images" }}
                    search={{}}
                  >
                    View all
                  </Link>
                </div>
                <p className={styles.secondary}>
                  {latestImages.length ? `Linked to ${latest.data?.tag}` : "Latest container"}
                </p>
                {containers.map((image) => (
                  <div className={styles.image} key={`${image.name}:${image.tag}`}>
                    <Link
                      to="/$owner/$repo/$section"
                      params={{ owner, repo, section: "images" }}
                      search={{}}
                    >
                      <Package size={16} />
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
      <div id="clone-options" popover="auto" className={styles.clonePopover}>
        <h2>Clone repository</h2>
        <div className={styles.command}>
          <code>{metadata.data.clone_url}</code>
          <CopyButton value={metadata.data.clone_url} label="Copy clone URL" compact />
        </div>
      </div>
      <dialog
        id="file-finder"
        closedby="any"
        className={styles.finder}
        onToggle={finderToggled}
        aria-labelledby="finder-title"
      >
        <div className={styles.dialogHeading}>
          <h2 id="finder-title">Go to file</h2>
        </div>
        <label htmlFor="file-search" className={controls.srOnly}>
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
          className={`${controls.button} ${styles.dialogClose}`}
          type="button"
          commandfor="file-finder"
          command="close"
          aria-label="Close file finder"
        >
          <X size={17} />
        </button>
        <div className={styles.finderResults}>
          {files.isPending && <output>Loading files…</output>}
          {files.error && <ErrorNotice error={files.error} retry={retryFiles} />}{" "}
          {visibleFiles.slice(0, 100).map((path) => (
            <Link
              key={path}
              to="/$owner/$repo"
              params={{ owner, repo }}
              search={{ ref: search.ref, file: path }}
              onClick={closeFinder}
            >
              <File size={15} />
              {path}
            </Link>
          ))}
          {files.data && visibleFiles.length === 0 && <p>No matching files.</p>}
        </div>
        <p className={styles.finderHint}>
          {name}
          <span>Esc to close</span>
        </p>
      </dialog>
      <link rel="stylesheet" href="/api/code.css" />
    </>
  );
}
