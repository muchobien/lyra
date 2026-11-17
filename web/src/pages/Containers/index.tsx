import { Link } from "@tanstack/react-router";
import { PackageIcon } from "@primer/octicons-react";
import { CopyButton } from "@/components/controls/CopyButton";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { DateTime } from "@/components/metadata/DateTime";
import { CodeBreadcrumb } from "@/components/code/CodeBreadcrumb";
import "@/styles/workspace.css";
import "./styles.css";
import { useConnect } from "./connect";

export function Containers() {
  const { owner, repo, search, images, releases, metadata, retry, searchChanged, visible } =
    useConnect();
  return (
    <>
      <CodeBreadcrumb
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        section="images"
        label="Containers"
      />
      <div className="lyra-workspace-toolbar">
        <input
          className="lyra-workspace-search"
          type="search"
          aria-label="Search containers"
          placeholder="Search image, tag, or digest…"
          value={search.q || ""}
          onChange={searchChanged}
        />
      </div>
      {images.error && <ErrorNotice error={images.error} retry={retry} />}
      {images.isPending && <output className="lyra-workspace-empty">Loading containers…</output>}
      {images.data && (
        <div className="lyra-workspace-panel">
          {visible.map((image) => (
            <article className="lyra-workspace-row" key={`${image.name}:${image.tag}`}>
              <PackageIcon size={16} />
              <div className="lyra-workspace-grow">
                <strong className="lyra-workspace-rowTitle">
                  {image.name}:<span className="lyra-workspace-link">{image.tag}</span>
                </strong>
                <p className="lyra-workspace-meta">
                  <code>{image.digest}</code> · <DateTime value={image.pushed_at} />
                </p>
                <div className="lyra-workspace-labels">
                  {releases.data
                    ?.filter(
                      (release) =>
                        !release.draft &&
                        (release.tag === image.tag || release.tag === `v${image.tag}`),
                    )
                    .map((release) => (
                      <Link
                        className="lyra-workspace-label"
                        key={release.id}
                        to="/$owner/$repo/releases/tag/$"
                        params={{ owner, repo, _splat: `${release.tag}` }}
                        search={{}}
                      >
                        Release {release.tag}
                      </Link>
                    ))}
                </div>
                <code className="lyra-releases-pullCommand">docker pull {image.reference}</code>
              </div>
              <CopyButton
                value={`docker pull ${image.reference}`}
                label={`Copy pull command for ${image.name}:${image.tag}`}
                compact
              />
            </article>
          ))}
          {visible.length === 0 && <p className="lyra-workspace-empty">No matching containers.</p>}
        </div>
      )}
    </>
  );
}
