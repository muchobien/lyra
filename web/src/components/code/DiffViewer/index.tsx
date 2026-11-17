import { Link } from "@tanstack/react-router";
import { CheckIcon, FileCodeIcon } from "@primer/octicons-react";
import { ActorName } from "@/components/metadata/ActorName";
import { Markdown } from "@/components/markdown/Markdown";
import { RichText } from "@/components/markdown/RichText";
import { FileIcon } from "@/components/code/FileIcon";
import { ResponsiveDetails } from "@/components/controls/ResponsiveDetails";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "./styles.css";
import { diffPaths } from "@/routing/paths";
import { useLogic, type DiffProps } from "./logic";

export type { CommentAnchor } from "./logic";

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
  const {
    search,
    viewed,
    largeFiles,
    wide,
    split,
    selected,
    displayedFiles,
    commentHTML,
    layoutChanged,
    viewedChanged,
    showLarge,
    commentOnLine,
    nextFile,
  } = useLogic({ files, owner, repo, section, target, head, onComment, comments });
  return (
    <section aria-label="Files changed">
      <div className="lyra-workspace-toolbar">
        <span className="lyra-workspace-meta">
          {files.length} files changed · {viewed.size} viewed
        </span>
        <span className="lyra-workspace-grow" />
        {selected && (
          <Link
            className="lyra-controls-button"
            to={diffPaths[section]}
            params={{ owner, repo, number: target, sha: target }}
            search={{ ...search, file: undefined }}
          >
            All files
          </Link>
        )}
        {selected && nextFile && (
          <Link
            className="lyra-controls-button"
            to={diffPaths[section]}
            params={{ owner, repo, number: target, sha: target }}
            search={{ ...search, file: nextFile.path }}
          >
            Next file
          </Link>
        )}
        {wide && (
          <select
            className="lyra-controls-select"
            aria-label="Diff layout"
            value={split ? "split" : "unified"}
            onChange={layoutChanged}
          >
            <option value="unified">Unified</option>
            <option value="split">Split</option>
          </select>
        )}
      </div>
      <div className="lyra-diff-viewer-workspace">
        <ResponsiveDetails className="lyra-diff-viewer-navigation" query="(width > 800px)">
          <summary>
            <FileCodeIcon size={16} />
            Files <span>{files.length}</span>
          </summary>
          <nav aria-label="Changed files">
            {files.map((file) => (
              <Link
                key={file.path}
                to={diffPaths[section]}
                params={{ owner, repo, number: target, sha: target }}
                search={{ ...search, file: file.path }}
                aria-current={selected?.path === file.path ? "page" : undefined}
              >
                <FileIcon name={file.path.split("/").at(-1) ?? file.path} />
                <span className="lyra-diff-viewer-fileName" title={file.path}>
                  <strong>{file.path.split("/").at(-1)}</strong>
                  {file.path.includes("/") && (
                    <small>{file.path.split("/").slice(0, -1).join("/")}</small>
                  )}
                </span>
                {viewed.has(file.path) && <CheckIcon size={16} />}
              </Link>
            ))}
          </nav>
        </ResponsiveDetails>
        <div className="lyra-diff-viewer-files" data-layout={split ? "split" : "unified"}>
          {displayedFiles.map(({ file, rows }) => (
            <details className="lyra-diff-viewer-file" key={file.path} open>
              <summary className="lyra-diff-viewer-fileHeading">
                <FileIcon name={file.path.split("/").at(-1) ?? file.path} />
                <code>
                  {file.old_path && <span>{file.old_path} renamed to </span>}
                  {file.path}
                </code>
                <span className="lyra-diff-viewer-fileStats">
                  <span>+{file.lines?.filter((line) => line.kind === "add").length || 0}</span>
                  <span>−{file.lines?.filter((line) => line.kind === "delete").length || 0}</span>
                </span>
                <label className="lyra-workspace-checkbox lyra-diff-viewer-viewed">
                  <input
                    type="checkbox"
                    value={file.path}
                    checked={viewed.has(file.path)}
                    onChange={viewedChanged}
                  />
                  Viewed
                </label>
              </summary>
              {file.binary && <p className="lyra-workspace-empty">Binary file changed.</p>}
              {!file.binary &&
                file.lines &&
                file.lines.length > 2000 &&
                !largeFiles.has(file.path) && (
                  <div className="lyra-workspace-empty">
                    <p>This file contains {file.lines.length} diff lines.</p>
                    <button
                      type="button"
                      className="lyra-controls-button"
                      data-path={file.path}
                      onClick={showLarge}
                    >
                      Show diff
                    </button>
                  </div>
                )}
              {!file.binary &&
                (!file.lines || file.lines.length <= 2000 || largeFiles.has(file.path)) && (
                  <div className="lyra-diff-viewer-codeScroll">
                    {rows.map(({ line, old, next, comments: rowComments }, index) => (
                      <div key={index}>
                        {line.kind === "hunk" ? (
                          <div className="lyra-diff-viewer-hunk">{line.text}</div>
                        ) : (
                          <div
                            className="lyra-diff-viewer-line"
                            data-kind={line.kind}
                            data-old-kind={old?.kind}
                            data-new-kind={next?.kind}
                          >
                            <div className="lyra-diff-viewer-oldNumber">
                              {old &&
                                old.old > 0 &&
                                (canComment ? (
                                  <button
                                    type="button"
                                    onClick={commentOnLine}
                                    data-path={file.old_path || file.path}
                                    data-line={old.old}
                                    data-side="old"
                                    aria-label={`Comment on old line ${old.old} of ${file.old_path || file.path}`}
                                  >
                                    {old.old}
                                  </button>
                                ) : (
                                  old.old
                                ))}
                            </div>
                            {split && (
                              <div className="lyra-diff-viewer-oldCode">
                                {old && old.kind !== "add" && (
                                  <>
                                    <span aria-hidden="true">
                                      {old.kind === "delete" ? "−" : " "}
                                    </span>
                                    {old.html ? (
                                      <div dangerouslySetInnerHTML={{ __html: old.html }} />
                                    ) : (
                                      <code>{old.text}</code>
                                    )}
                                  </>
                                )}
                              </div>
                            )}
                            <div className="lyra-diff-viewer-newNumber">
                              {next &&
                                next.new > 0 &&
                                (canComment ? (
                                  <button
                                    type="button"
                                    onClick={commentOnLine}
                                    data-path={file.path}
                                    data-line={next.new}
                                    data-side="new"
                                    aria-label={`Comment on new line ${next.new} of ${file.path}`}
                                  >
                                    {next.new}
                                  </button>
                                ) : (
                                  next.new
                                ))}
                            </div>
                            <div className="lyra-diff-viewer-code">
                              {next && (!split || next.kind !== "delete") && (
                                <>
                                  <span aria-hidden="true">
                                    {({ add: "+", delete: "−" } as Record<string, string>)[
                                      next.kind
                                    ] || " "}
                                  </span>
                                  {next.html ? (
                                    <div dangerouslySetInnerHTML={{ __html: next.html }} />
                                  ) : (
                                    <code>{next.text}</code>
                                  )}
                                </>
                              )}
                            </div>
                          </div>
                        )}
                        {rowComments.map((comment) => (
                          <article
                            className="lyra-diff-viewer-lineComment"
                            key={comment.id}
                            id={`comment-${comment.id}`}
                          >
                            <div className="lyra-workspace-panelHeading">
                              <ActorName actor={comment.author} />
                              <span className="lyra-workspace-meta">
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
                    {!file.lines?.length && (
                      <p className="lyra-workspace-empty">No text changes.</p>
                    )}
                  </div>
                )}
            </details>
          ))}
          {files.length === 0 && (
            <div className="lyra-workspace-panel">
              <p className="lyra-workspace-empty">No file changes.</p>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
