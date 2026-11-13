import { useEffect, useState } from "react";
import type { ChangeEvent, MouseEvent } from "react";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { FileCode2, Check } from "lucide-react";
import type { DiffFile, DiscussionComment } from "./api";
import { ActorName, Markdown, RichText } from "./Workspace";
import controls from "./Controls.module.css";
import common from "./Workspace.module.css";
import styles from "./DiffViewer.module.css";

const reviewedFiles = new Map<string, Set<string>>();
export type CommentAnchor = { path: string; line: number; side: string; head_sha: string };
type DiffProps = {
  files: DiffFile[];
  owner: string;
  repo: string;
  section: string;
  target: string;
  head: string;
  canComment?: boolean;
  onComment?: (anchor: CommentAnchor) => void;
  comments?: DiscussionComment[];
};
export function DiffViewer({
  files,
  owner,
  repo,
  section,
  target,
  head,
  canComment = false,
  onComment,
  comments = [],
}: DiffProps) {
  const search = useSearch({ strict: false });
  const navigate = useNavigate({ from: "/$owner/$repo/$section/$" });
  const storageKey = `${owner}/${repo}/${section}/${target}/${head}`;
  const [viewed, setViewed] = useState(() => new Set(reviewedFiles.get(storageKey)));
  const [largeFiles, setLargeFiles] = useState(new Set<string>());
  const [wide, setWide] = useState(() => matchMedia("(width > 800px)").matches);
  const [navigationOpen] = useState(wide);
  useEffect(() => {
    const media = matchMedia("(width > 800px)");
    function changed() {
      setWide(media.matches);
    }
    media.addEventListener("change", changed);
    return () => media.removeEventListener("change", changed);
  }, []);
  const split = search.layout === "split" && wide;
  const selected = files.find((file) => file.path === search.file);
  const visible = selected ? [selected] : files;
  const commentHTML = new Map(comments.map((comment) => [comment.id, comment.html]));
  function layoutChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({
      search: { ...search, layout: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }
  function viewedChanged(event: ChangeEvent<HTMLInputElement>) {
    const next = new Set(viewed);
    if (event.target.checked) {
      next.add(event.target.value);
    } else {
      next.delete(event.target.value);
    }
    reviewedFiles.set(storageKey, next);
    setViewed(next);
  }
  function showLarge(event: MouseEvent<HTMLButtonElement>) {
    setLargeFiles(new Set([...largeFiles, event.currentTarget.dataset.path || ""]));
  }
  function commentOnLine(event: MouseEvent<HTMLButtonElement>) {
    const { path = "", line = "0", side = "new" } = event.currentTarget.dataset;
    onComment?.({ path, line: Number(line), side, head_sha: head });
  }
  const selectedIndex = selected ? files.indexOf(selected) : -1;
  const nextFile = files[selectedIndex + 1];
  return (
    <section aria-label="Files changed">
      <div className={common.toolbar}>
        <span className={common.meta}>
          {files.length} files changed · {viewed.size} viewed
        </span>
        <span className={common.grow} />
        {selected && (
          <Link
            className={controls.button}
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section, _splat: target }}
            search={{ ...search, file: undefined }}
          >
            All files
          </Link>
        )}
        {selected && nextFile && (
          <Link
            className={controls.button}
            to="/$owner/$repo/$section/$"
            params={{ owner, repo, section, _splat: target }}
            search={{ ...search, file: nextFile.path }}
          >
            Next file
          </Link>
        )}
        {wide && (
          <select
            className={controls.select}
            aria-label="Diff layout"
            value={split ? "split" : "unified"}
            onChange={layoutChanged}
          >
            <option value="unified">Unified</option>
            <option value="split">Split</option>
          </select>
        )}
      </div>
      <div className={styles.workspace}>
        <details className={styles.navigation} open={navigationOpen}>
          <summary>
            <FileCode2 size={15} />
            Files <span>{files.length}</span>
          </summary>
          <nav aria-label="Changed files">
            {files.map((file) => (
              <Link
                key={file.path}
                to="/$owner/$repo/$section/$"
                params={{ owner, repo, section, _splat: target }}
                search={{ ...search, file: file.path }}
                aria-current={selected?.path === file.path ? "page" : undefined}
              >
                <span>{file.path}</span>
                {viewed.has(file.path) && <Check size={13} />}
              </Link>
            ))}
          </nav>
        </details>
        <div className={styles.files} data-layout={split ? "split" : "unified"}>
          {visible.map((file) => (
            <details className={styles.file} key={file.path} open>
              <summary className={styles.fileHeading}>
                <code>
                  {file.old_path && <span>{file.old_path} / </span>}
                  {file.path}
                </code>
                <span className={styles.fileStats}>
                  <span>+{file.lines?.filter((line) => line.kind === "add").length || 0}</span>
                  <span>−{file.lines?.filter((line) => line.kind === "delete").length || 0}</span>
                </span>
              </summary>
              <div className={styles.viewed}>
                <label className={common.checkbox}>
                  <input
                    type="checkbox"
                    value={file.path}
                    checked={viewed.has(file.path)}
                    onChange={viewedChanged}
                  />
                  Viewed
                </label>
              </div>
              {file.binary && <p className={common.empty}>Binary file changed.</p>}
              {!file.binary &&
                file.lines &&
                file.lines.length > 2000 &&
                !largeFiles.has(file.path) && (
                  <div className={common.empty}>
                    <p>This file contains {file.lines.length} diff lines.</p>
                    <button
                      type="button"
                      className={controls.button}
                      data-path={file.path}
                      onClick={showLarge}
                    >
                      Show diff
                    </button>
                  </div>
                )}
              {!file.binary &&
                (!file.lines || file.lines.length <= 2000 || largeFiles.has(file.path)) && (
                  <div className={styles.codeScroll}>
                    {file.lines?.map((line, index) => (
                      <div key={index}>
                        {line.kind === "hunk" ? (
                          <div className={styles.hunk}>{line.text}</div>
                        ) : (
                          <div className={styles.line} data-kind={line.kind}>
                            <div className={styles.oldNumber}>
                              {line.old > 0 &&
                                (canComment ? (
                                  <button
                                    type="button"
                                    onClick={commentOnLine}
                                    data-path={file.old_path || file.path}
                                    data-line={line.old}
                                    data-side="old"
                                    aria-label={`Comment on old line ${line.old} of ${file.old_path || file.path}`}
                                  >
                                    {line.old}
                                  </button>
                                ) : (
                                  line.old
                                ))}
                            </div>
                            {split && (
                              <div className={styles.oldCode}>
                                {line.kind !== "add" && (
                                  <>
                                    <span aria-hidden="true">
                                      {line.kind === "delete" ? "−" : " "}
                                    </span>
                                    {line.html ? (
                                      <div dangerouslySetInnerHTML={{ __html: line.html }} />
                                    ) : (
                                      <code>{line.text}</code>
                                    )}
                                  </>
                                )}
                              </div>
                            )}
                            <div className={styles.newNumber}>
                              {line.new > 0 &&
                                (canComment ? (
                                  <button
                                    type="button"
                                    onClick={commentOnLine}
                                    data-path={file.path}
                                    data-line={line.new}
                                    data-side="new"
                                    aria-label={`Comment on new line ${line.new} of ${file.path}`}
                                  >
                                    {line.new}
                                  </button>
                                ) : (
                                  line.new
                                ))}
                            </div>
                            <div className={styles.code}>
                              {(!split || line.kind !== "delete") && (
                                <>
                                  <span aria-hidden="true">
                                    {({ add: "+", delete: "−" } as Record<string, string>)[
                                      line.kind
                                    ] || " "}
                                  </span>
                                  {line.html ? (
                                    <div dangerouslySetInnerHTML={{ __html: line.html }} />
                                  ) : (
                                    <code>{line.text}</code>
                                  )}
                                </>
                              )}
                            </div>
                          </div>
                        )}
                        {line.comments?.map((comment) => (
                          <article
                            className={styles.lineComment}
                            key={comment.id}
                            id={`comment-${comment.id}`}
                          >
                            <div className={common.panelHeading}>
                              <ActorName actor={comment.author} />
                              <span className={common.meta}>
                                {comment.side} line {comment.line}
                              </span>
                            </div>
                            {commentHTML.get(comment.id) ? (
                              <Markdown html={commentHTML.get(comment.id) || ""} />
                            ) : (
                              <RichText source={comment.body} />
                            )}
                          </article>
                        ))}
                      </div>
                    ))}
                    {!file.lines?.length && <p className={common.empty}>No text changes.</p>}
                  </div>
                )}
            </details>
          ))}
          {files.length === 0 && (
            <div className={common.panel}>
              <p className={common.empty}>No file changes.</p>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
