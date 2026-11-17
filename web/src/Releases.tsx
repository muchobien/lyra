import { useContext, useEffect, useState } from "react";
import type { ChangeEvent, MouseEvent, SubmitEvent } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  DownloadIcon,
  GitCommitIcon,
  PackageIcon,
  PencilIcon,
  PlusIcon,
  TagIcon,
  TrashIcon,
} from "@primer/octicons-react";
import { APIError, IdentityContext, type Release } from "./api";
import type { ReleaseWrite } from "./api/generated/models";
import {
  useListReleases,
  useGetRepo,
  useGetReleaseByTag,
  getLatestRelease,
  getGetLatestReleaseQueryKey,
  useListImages,
  useCreateRelease,
  useEditRelease,
  usePreviewReleaseNotes,
  useDeleteRelease,
} from "./api/generated/lyra";
import { CopyButton, ErrorNotice } from "./Controls";
import {
  ActorName,
  ConfirmDialog,
  DateTime,
  DialogClose,
  FormError,
  MarkdownEditor,
  RichText,
} from "./Workspace";
import { CodeBreadcrumb } from "./CodeScreens";
import { FileIcon } from "./FileIcon";
import { useDraft } from "./drafts";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";
import delivery from "./Releases.module.css";

type ReleasesProps = { tag: string };
export function Releases({ tag }: ReleasesProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const list = useListReleases(owner, repo);
  const metadata = useGetRepo(owner, repo);
  const single = useGetReleaseByTag(owner, repo, tag, { query: { enabled: !!tag } });
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
  const [editing, setEditing] = useState<Release | null>(null);
  const [draft, setDraft] = useState(true);
  const [prerelease, setPrerelease] = useState(false);
  const [deleting, setDeleting] = useState<Release | null>(null);
  const [cleanupTag, setCleanupTag] = useState(false);
  const draftKey = `${identity?.actor.user}/${name}/release/${editing?.id || "new"}`;
  const title = useDraft(`${draftKey}/title`, editing?.title);
  const notes = useDraft(`${draftKey}/notes`, editing?.notes);
  const releaseTag = useDraft(`${draftKey}/tag`, editing?.tag);
  const target = useDraft(`${draftKey}/target`, editing?.target || metadata.data?.default_branch);
  const canWrite = identity?.scope !== "read";
  async function refresh() {
    await queryClient.invalidateQueries();
  }
  async function saved(release: ReleaseWrite) {
    document.querySelector<HTMLDialogElement>("#release-editor")?.close();
    title.clear();
    notes.clear();
    releaseTag.clear();
    target.clear();
    await refresh();
    void navigate({
      to: "/$owner/$repo/$section/$",
      params: { owner, repo, section: "releases", _splat: `tag/${release.tag}` },
      search: {},
    });
  }
  const creating = useCreateRelease({ mutation: { onSuccess: saved } });
  const updating = useEditRelease({ mutation: { onSuccess: saved } });
  const saving = editing ? updating : creating;
  const generating = usePreviewReleaseNotes({
    mutation: {
      onSuccess: (result) =>
        notes.setValue([notes.value.trim(), result.notes].filter(Boolean).join("\n\n")),
    },
  });
  const removing = useDeleteRelease({
    mutation: {
      onSuccess: async () => {
        document.querySelector<HTMLDialogElement>("#delete-release")?.close();
        await refresh();
        if (tag === deleting?.tag) {
          void navigate({
            to: "/$owner/$repo/$section",
            params: { owner, repo, section: "releases" },
            search: {},
          });
        }
      },
    },
  });
  useEffect(() => {
    document.title = `${single.data?.title || "Releases"} · ${name} · Lyra`;
  }, [single.data?.title, name]);
  function retry() {
    void list.refetch();
    if (tag) {
      void single.refetch();
    }
  }
  function prepareEditor(event: MouseEvent<HTMLButtonElement>) {
    const release =
      list.data?.find((release) => release.id === Number(event.currentTarget.dataset.id)) || null;
    setEditing(release);
    setDraft(release?.draft ?? true);
    setPrerelease(release?.prerelease ?? false);
    creating.reset();
    updating.reset();
    generating.reset();
  }
  function editorChanged(event: ChangeEvent<HTMLInputElement>) {
    const { name, value, checked } = event.target;
    if (name === "title") {
      title.setValue(value);
    }
    if (name === "tag") {
      releaseTag.setValue(value);
    }
    if (name === "target") {
      target.setValue(value);
    }
    if (name === "draft") {
      setDraft(checked);
    }
    if (name === "prerelease") {
      setPrerelease(checked);
    }
  }
  function saveRelease(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const body = { title: title.value, notes: notes.value, draft, prerelease };
    if (editing) {
      updating.mutate({
        owner,
        repo,
        id: editing.id,
        data: {
          ...body,
          target: editing.draft ? target.value : undefined,
        },
      });
    } else {
      creating.mutate({
        owner,
        repo,
        data: { ...body, tag: releaseTag.value, target: target.value },
      });
    }
  }
  function generateNotes() {
    generating.mutate({
      owner,
      repo,
      params: { tag: releaseTag.value, target: target.value, prerelease },
    });
  }
  function prepareDelete(event: MouseEvent<HTMLButtonElement>) {
    setDeleting(
      list.data?.find((release) => release.id === Number(event.currentTarget.dataset.id)) || null,
    );
    setCleanupTag(false);
    removing.reset();
  }
  function cleanupChanged(event: ChangeEvent<HTMLInputElement>) {
    setCleanupTag(event.target.checked);
  }
  function deleteRelease() {
    if (deleting) {
      removing.mutate({ owner, repo, id: deleting.id, params: { cleanup_tag: cleanupTag } });
    }
  }
  function filtersChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    void navigate({
      to: "/$owner/$repo/$section",
      params: { owner, repo, section: "releases" },
      search: { ...search, [event.target.name]: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }
  const shown = tag
    ? [single.data].filter((release): release is Release => !!release)
    : (list.data || []).filter(
        (release) =>
          (!search.q ||
            `${release.tag} ${release.title}`.toLowerCase().includes(search.q.toLowerCase())) &&
          (!search.state ||
            (search.state === "draft" && release.draft) ||
            (search.state === "prerelease" && release.prerelease && !release.draft) ||
            (search.state === "stable" && !release.draft && !release.prerelease)),
      );
  const error = list.error || single.error;
  let saveLabel = "Save changes";
  if (!editing || editing.draft) {
    saveLabel = draft ? "Save draft" : "Publish release";
  }
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
          <div className={styles.headingActions}>
            <button
              className={controls.button}
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
              className={styles.danger}
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
        <div className={styles.toolbar}>
          <input
            type="search"
            className={styles.search}
            name="q"
            aria-label="Search releases"
            placeholder="Search releases…"
            value={search.q || ""}
            onChange={filtersChanged}
          />
          <select
            className={controls.select}
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
          <span className={styles.grow} />
          {canWrite && (
            <button
              className={styles.primary}
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
        <output className={styles.empty}>Loading releases…</output>
      )}
      {shown.map((release) => (
        <article className={styles.panel} key={release.id}>
          <div className={`${styles.panelHeading} ${delivery.releaseHeading}`}>
            <div>
              <div className={delivery.titleLine}>
                {tag ? (
                  <h1 className={delivery.title}>{release.title || release.tag}</h1>
                ) : (
                  <Link
                    className={styles.rowTitle}
                    to="/$owner/$repo/$section/$"
                    params={{ owner, repo, section: "releases", _splat: `tag/${release.tag}` }}
                    search={{}}
                  >
                    {release.title || release.tag}
                  </Link>
                )}
                {latest.data?.id === release.id && <span className={delivery.latest}>Latest</span>}
                {release.draft && <span className={styles.label}>Draft</span>}
                {release.prerelease && <span className={styles.count}>Prerelease</span>}
              </div>
              <p className={styles.meta}>
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
                      className={styles.link}
                      to="/$owner/$repo/$section/$"
                      params={{ owner, repo, section: "commit", _splat: release.commit }}
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
            <p className={styles.empty}>No release notes.</p>
          )}
          {!tag && release.assets.length > 0 && (
            <p className={delivery.assetLine}>
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
            <section className={delivery.assets}>
              <h2>
                Assets <span className={styles.count}>{release.assets.length}</span>
              </h2>
              {release.assets.map((asset) => (
                <div className={delivery.assetRow} key={asset.id}>
                  <FileIcon name={asset.name} />
                  <a href={asset.download_url} className={delivery.assetName}>
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
                  <a className={controls.button} href={asset.download_url}>
                    <DownloadIcon size={16} />
                    Download
                  </a>
                </div>
              ))}
            </section>
          )}
          {!!release.images?.length && (
            <section className={delivery.assets}>
              <h2>Containers linked to {release.tag}</h2>
              {release.images.map((image) => (
                <div className={styles.row} key={`${image.name}:${image.tag}`}>
                  <PackageIcon size={16} />
                  <div className={styles.grow}>
                    <Link
                      className={styles.link}
                      to="/$owner/$repo/$section"
                      params={{ owner, repo, section: "images" }}
                      search={{ q: image.tag }}
                    >
                      {image.name}:{image.tag}
                    </Link>
                    <p className={styles.meta}>
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
        <div className={styles.panel}>
          <p className={styles.empty}>No matching releases.</p>
        </div>
      )}
      {canWrite && (
        <dialog
          id="release-editor"
          className={styles.dialog}
          closedby="any"
          aria-labelledby="release-editor-title"
        >
          <form onSubmit={saveRelease}>
            <h2 id="release-editor-title">{editing ? `Edit ${editing.tag}` : "New release"}</h2>
            <fieldset disabled={saving.isPending}>
              <label className={styles.field}>
                <span>Tag</span>
                <input
                  name="tag"
                  value={releaseTag.value}
                  onChange={editorChanged}
                  readOnly={!!editing}
                  required
                />
              </label>
              {(!editing || editing.draft) && (
                <label className={styles.field}>
                  <span>Target branch, tag, or commit</span>
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
                </label>
              )}
              <label className={styles.field}>
                <span>Title</span>
                <input
                  name="title"
                  value={title.value}
                  onChange={editorChanged}
                  maxLength={256}
                  placeholder="Release title"
                />
              </label>
              <MarkdownEditor
                id="release-notes"
                label="Release notes"
                value={notes.value}
                onChange={notes.setValue}
              />
              <button
                className={controls.button}
                type="button"
                onClick={generateNotes}
                disabled={generating.isPending}
              >
                {generating.isPending ? "Generating…" : "Generate notes from merged pull requests"}
              </button>
              <FormError error={generating.error} />
              <label className={styles.checkbox}>
                <input
                  type="checkbox"
                  name="prerelease"
                  checked={prerelease}
                  onChange={editorChanged}
                />
                Mark as a prerelease
              </label>
              {(!editing || editing.draft) && (
                <label className={styles.checkbox}>
                  <input type="checkbox" name="draft" checked={draft} onChange={editorChanged} />
                  Save as draft
                </label>
              )}
              <FormError error={saving.error} />
              <div className={styles.dialogActions}>
                <button
                  className={controls.button}
                  type="button"
                  commandfor="release-editor"
                  command="close"
                >
                  Cancel
                </button>
                <button className={styles.primary} type="submit">
                  {saving.isPending ? "Saving…" : saveLabel}
                </button>
              </div>
            </fieldset>
            <DialogClose target="release-editor" />
          </form>
        </dialog>
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
        <label className={styles.checkbox}>
          <input type="checkbox" checked={cleanupTag} onChange={cleanupChanged} />
          Also delete Git tag {deleting?.tag}
        </label>
      </ConfirmDialog>
    </>
  );
}

export function Containers() {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate({ from: "/$owner/$repo/$section" });
  const name = `${owner}/${repo}`;
  const images = useListImages(owner, repo);
  const releases = useListReleases(owner, repo);
  const metadata = useGetRepo(owner, repo);
  useEffect(() => {
    document.title = `Containers · ${name} · Lyra`;
  }, [name]);
  function retry() {
    void images.refetch();
  }
  function searchChanged(event: ChangeEvent<HTMLInputElement>) {
    void navigate({ search: { q: event.target.value }, replace: true, resetScroll: false });
  }
  const visible =
    images.data
      ?.filter(
        (image) =>
          !search.q ||
          `${image.name}:${image.tag} ${image.digest}`
            .toLowerCase()
            .includes(search.q.toLowerCase()),
      )
      .toSorted((first, second) => Date.parse(second.pushed_at) - Date.parse(first.pushed_at)) ||
    [];
  return (
    <>
      <CodeBreadcrumb
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        section="images"
        label="Containers"
      />
      <div className={styles.toolbar}>
        <input
          className={styles.search}
          type="search"
          aria-label="Search containers"
          placeholder="Search image, tag, or digest…"
          value={search.q || ""}
          onChange={searchChanged}
        />
      </div>
      {images.error && <ErrorNotice error={images.error} retry={retry} />}
      {images.isPending && <output className={styles.empty}>Loading containers…</output>}
      {images.data && (
        <div className={styles.panel}>
          {visible.map((image) => (
            <article className={styles.row} key={`${image.name}:${image.tag}`}>
              <PackageIcon size={16} />
              <div className={styles.grow}>
                <strong className={styles.rowTitle}>
                  {image.name}:<span className={styles.link}>{image.tag}</span>
                </strong>
                <p className={styles.meta}>
                  <code>{image.digest}</code> · <DateTime value={image.pushed_at} />
                </p>
                <div className={styles.labels}>
                  {releases.data
                    ?.filter(
                      (release) =>
                        !release.draft &&
                        (release.tag === image.tag || release.tag === `v${image.tag}`),
                    )
                    .map((release) => (
                      <Link
                        className={styles.label}
                        key={release.id}
                        to="/$owner/$repo/$section/$"
                        params={{ owner, repo, section: "releases", _splat: `tag/${release.tag}` }}
                        search={{}}
                      >
                        Release {release.tag}
                      </Link>
                    ))}
                </div>
                <code className={delivery.pullCommand}>docker pull {image.reference}</code>
              </div>
              <CopyButton
                value={`docker pull ${image.reference}`}
                label={`Copy pull command for ${image.name}:${image.tag}`}
                compact
              />
            </article>
          ))}
          {visible.length === 0 && <p className={styles.empty}>No matching containers.</p>}
        </div>
      )}
    </>
  );
}
