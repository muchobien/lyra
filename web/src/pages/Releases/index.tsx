import { Field } from "@/components/forms/Field";
import { FormDialog } from "@/components/forms/FormDialog";
import { Link } from "@tanstack/react-router";
import {
  DownloadIcon,
  GitCommitIcon,
  PackageIcon,
  PencilIcon,
  PlusIcon,
  TagIcon,
  TrashIcon,
} from "@primer/octicons-react";
import { CopyButton } from "@/components/controls/CopyButton";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import { ActorName } from "@/components/metadata/ActorName";
import { DateTime } from "@/components/metadata/DateTime";
import { MarkdownEditor } from "@/components/markdown/MarkdownEditor";
import { RichText } from "@/components/markdown/RichText";
import { ConfirmDialog } from "@/components/forms/ConfirmDialog";
import { FormError } from "@/components/forms/FormError";
import { CodeBreadcrumb } from "@/components/code/CodeBreadcrumb";
import { FileIcon } from "@/components/code/FileIcon";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "./styles.css";
import { useConnect, type ReleasesProps } from "./connect";

export function Releases({ tag }: ReleasesProps) {
  const {
    owner,
    repo,
    search,
    list,
    metadata,
    single,
    latest,
    editing,
    draft,
    editorVersion,
    deleting,
    title,
    notes,
    releaseTag,
    target,
    canWrite,
    saving,
    generating,
    removing,
    retry,
    prepareEditor,
    editorChanged,
    saveRelease,
    generateNotes,
    prepareDelete,
    deleteRelease,
    filtersChanged,
    shown,
    error,
    saveLabel,
  } = useConnect({ tag });
  return (
    <>
      <CodeBreadcrumb
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        section="releases"
        label="Releases"
        current={tag || undefined}
      >
        {canWrite && tag && single.data && (
          <div className="lyra-workspace-headingActions">
            <button
              className="lyra-controls-button"
              type="button"
              data-id={single.data.id}
              onClick={prepareEditor}
              commandfor="release-editor"
              command="show-modal"
            >
              <PencilIcon size={16} />
              Edit
            </button>
            <button
              className="lyra-workspace-danger"
              type="button"
              data-id={single.data.id}
              onClick={prepareDelete}
              commandfor="delete-release"
              command="show-modal"
            >
              <TrashIcon size={16} />
              Delete
            </button>
          </div>
        )}
      </CodeBreadcrumb>
      {!tag && (
        <div className="lyra-workspace-toolbar">
          <input
            type="search"
            className="lyra-workspace-search"
            name="q"
            aria-label="Search releases"
            placeholder="Search releases…"
            value={search.q || ""}
            onChange={filtersChanged}
          />
          <select
            className="lyra-controls-select"
            name="state"
            aria-label="Release status"
            value={search.state || ""}
            onChange={filtersChanged}
          >
            <option value="">All releases</option>
            <option value="stable">Stable</option>
            <option value="prerelease">Prereleases</option>
            {canWrite && <option value="draft">Drafts</option>}
          </select>
          <span className="lyra-workspace-grow" />
          {canWrite && (
            <button
              className="lyra-workspace-primary"
              type="button"
              commandfor="release-editor"
              command="show-modal"
              onClick={prepareEditor}
              disabled={!metadata.data?.branches.length}
            >
              <PlusIcon size={16} />
              New release
            </button>
          )}
        </div>
      )}
      {error && <ErrorNotice error={error} retry={retry} />}
      {(tag ? single.isPending : list.isPending) && (
        <output className="lyra-workspace-empty">Loading releases…</output>
      )}
      {shown.map((release) => (
        <article className="lyra-workspace-panel" key={release.id}>
          <div className="lyra-workspace-panelHeading lyra-releases-releaseHeading">
            <div>
              <div className="lyra-releases-titleLine">
                {tag ? (
                  <h1 className="lyra-releases-title">{release.title || release.tag}</h1>
                ) : (
                  <Link
                    className="lyra-workspace-rowTitle"
                    to="/$owner/$repo/releases/tag/$"
                    params={{ owner, repo, _splat: `${release.tag}` }}
                    search={{}}
                  >
                    {release.title || release.tag}
                  </Link>
                )}
                {latest.data?.id === release.id && (
                  <span className="lyra-releases-latest">Latest</span>
                )}
                {release.draft && <span className="lyra-workspace-label">Draft</span>}
                {release.prerelease && <span className="lyra-workspace-count">Prerelease</span>}
              </div>
              <p className="lyra-workspace-meta">
                {release.title && release.title !== release.tag && (
                  <>
                    <TagIcon size={16} /> {release.tag} ·{" "}
                  </>
                )}
                <ActorName actor={release.author} /> ·{" "}
                <DateTime value={release.published_at || release.created_at} />
                {release.commit && (
                  <>
                    {" "}
                    ·{" "}
                    <Link
                      className="lyra-workspace-link"
                      to="/$owner/$repo/commit/$sha"
                      params={{ owner, repo, sha: release.commit }}
                      search={{}}
                    >
                      <GitCommitIcon size={16} /> <code>{release.commit.slice(0, 7)}</code>
                    </Link>
                  </>
                )}
              </p>
            </div>
          </div>
          {release.notes ? (
            <RichText source={release.notes} />
          ) : (
            <p className="lyra-workspace-empty">No release notes.</p>
          )}
          {!tag && release.assets.length > 0 && (
            <p className="lyra-releases-assetLine">
              <PackageIcon size={16} />
              {release.assets.length} {release.assets.length === 1 ? "asset" : "assets"} ·
              {release.assets.map((asset) => (
                <a key={asset.id} href={asset.download_url}>
                  {asset.name}
                </a>
              ))}
            </p>
          )}
          {tag && release.assets.length > 0 && (
            <section className="lyra-releases-assets">
              <h2>
                Assets <span className="lyra-workspace-count">{release.assets.length}</span>
              </h2>
              {release.assets.map((asset) => (
                <div className="lyra-releases-assetRow" key={asset.id}>
                  <FileIcon name={asset.name} />
                  <a href={asset.download_url} className="lyra-releases-assetName">
                    {asset.name}
                  </a>
                  <span>
                    {new Intl.NumberFormat(undefined, {
                      style: "unit",
                      unit: "byte",
                      unitDisplay: "narrow",
                      notation: "compact",
                    }).format(asset.size)}
                  </span>
                  <code title={`sha256:${asset.sha256}`}>sha256:{asset.sha256.slice(0, 12)}…</code>
                  <CopyButton
                    value={asset.sha256}
                    label={`Copy checksum for ${asset.name}`}
                    compact
                  />
                  <a className="lyra-controls-button" href={asset.download_url}>
                    <DownloadIcon size={16} />
                    Download
                  </a>
                </div>
              ))}
            </section>
          )}
          {!!release.images?.length && (
            <section className="lyra-releases-assets">
              <h2>Containers linked to {release.tag}</h2>
              {release.images.map((image) => (
                <div className="lyra-workspace-row" key={`${image.name}:${image.tag}`}>
                  <PackageIcon size={16} />
                  <div className="lyra-workspace-grow">
                    <Link
                      className="lyra-workspace-link"
                      to="/$owner/$repo/images"
                      params={{ owner, repo }}
                      search={{ q: image.tag }}
                    >
                      {image.name}:{image.tag}
                    </Link>
                    <p className="lyra-workspace-meta">
                      <code>{image.digest}</code>
                    </p>
                  </div>
                  <CopyButton
                    value={`docker pull ${image.reference}`}
                    label={`Copy pull command for ${image.name}:${image.tag}`}
                    compact
                  />
                </div>
              ))}
            </section>
          )}
        </article>
      ))}
      {!error && !(tag ? single.isPending : list.isPending) && shown.length === 0 && (
        <div className="lyra-workspace-panel">
          <p className="lyra-workspace-empty">No matching releases.</p>
        </div>
      )}
      {canWrite && (
        <FormDialog
          id="release-editor"
          formKey={String(editorVersion)}
          title={editing ? `Edit ${editing.tag}` : "New release"}
          onSubmit={saveRelease}
          pending={saving.isPending}
          error={saving.error}
          submitLabel={saving.isPending ? "Saving…" : saveLabel}
        >
          <Field label="Tag">
            <input
              name="tag"
              value={releaseTag.value}
              onChange={editorChanged}
              readOnly={!!editing}
              required
            />
          </Field>
          {(!editing || editing.draft) && (
            <Field label="Target branch, tag, or commit">
              <input
                name="target"
                value={target.value}
                onChange={editorChanged}
                list="release-targets"
                required
              />
              <datalist id="release-targets">
                {metadata.data?.branches.map((branch) => (
                  <option key={branch.name}>{branch.name}</option>
                ))}
              </datalist>
            </Field>
          )}
          <Field label="Title">
            <input
              name="title"
              value={title.value}
              onChange={editorChanged}
              maxLength={256}
              placeholder="Release title"
            />
          </Field>
          <MarkdownEditor
            id="release-notes"
            label="Release notes"
            value={notes.value}
            onChange={notes.setValue}
          />
          <button
            className="lyra-controls-button"
            type="button"
            onClick={generateNotes}
            disabled={generating.isPending}
          >
            {generating.isPending ? "Generating…" : "Generate notes from merged pull requests"}
          </button>
          <FormError error={generating.error} />
          <label className="lyra-workspace-checkbox">
            <input
              type="checkbox"
              name="prerelease"
              defaultChecked={editing?.prerelease ?? false}
            />
            Mark as a prerelease
          </label>
          {(!editing || editing.draft) && (
            <label className="lyra-workspace-checkbox">
              <input type="checkbox" name="draft" checked={draft} onChange={editorChanged} />
              Save as draft
            </label>
          )}
        </FormDialog>
      )}
      <ConfirmDialog
        id="delete-release"
        title={`Delete ${deleting?.tag || "release"}?`}
        confirm="Delete release"
        onConfirm={deleteRelease}
        pending={removing.isPending}
        error={removing.error}
        danger
      >
        <p>The release and its uploaded assets will be removed. Container images are kept.</p>
        <label className="lyra-workspace-checkbox">
          <input type="checkbox" name="cleanup_tag" />
          Also delete Git tag {deleting?.tag}
        </label>
      </ConfirmDialog>
    </>
  );
}
