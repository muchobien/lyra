import { useContext, useEffect, useState } from "react";
import type { ChangeEvent, FormEvent, MouseEvent } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Download, Package, Plus, Tags, Pencil, Trash2 } from "lucide-react";
import {
  api,
  IdentityContext,
  repositoryAPI,
  write,
  type Image,
  type Release,
  type Repository,
} from "./api";
import { CopyButton, ErrorNotice, formValue } from "./Controls";
import {
  ActorName,
  ConfirmDialog,
  DateTime,
  DialogClose,
  FormError,
  MarkdownEditor,
  RichText,
} from "./Workspace";
import { CodeNavigation } from "./CodeScreens";
import { useDraft } from "./drafts";
import controls from "./Controls.module.css";
import styles from "./Workspace.module.css";
import delivery from "./Releases.module.css";

type ReleaseEditor = {
  title: string;
  notes: string;
  target?: string;
  tag?: string;
  draft: boolean;
  prerelease: boolean;
};
type AssetDeletion = { release: number; name: string };
type ReleasesProps = { tag: string };
export function Releases({ tag }: ReleasesProps) {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const base = repositoryAPI(name, "/releases");
  const list = useQuery({
    queryKey: [base],
    queryFn: ({ signal }) => api<Release[]>(base, signal),
  });
  const metadata = useQuery({
    queryKey: ["repository", name],
    queryFn: ({ signal }) => api<Repository>(repositoryAPI(name), signal),
  });
  const singlePath = `${base}/tags/${tag.split("/").map(encodeURIComponent).join("/")}`;
  const single = useQuery({
    queryKey: [singlePath],
    queryFn: ({ signal }) => api<Release>(singlePath, signal),
    enabled: !!tag,
  });
  const [editing, setEditing] = useState<Release | null>(null);
  const [draft, setDraft] = useState(true);
  const [prerelease, setPrerelease] = useState(false);
  const [deleting, setDeleting] = useState<Release | null>(null);
  const [cleanupTag, setCleanupTag] = useState(false);
  const [assetDeletion, setAssetDeletion] = useState<AssetDeletion | null>(null);
  const draftKey = `${identity?.actor.user}/${name}/release/${editing?.id || "new"}`;
  const title = useDraft(`${draftKey}/title`, editing?.title);
  const notes = useDraft(`${draftKey}/notes`, editing?.notes);
  const releaseTag = useDraft(`${draftKey}/tag`, editing?.tag);
  const target = useDraft(`${draftKey}/target`, editing?.target || metadata.data?.default_branch);
  const canWrite = identity?.scope !== "read";
  async function refresh() {
    await queryClient.invalidateQueries();
  }
  const saving = useMutation({
    mutationFn: (body: ReleaseEditor) =>
      write<Release>(editing ? `${base}/${editing.id}` : base, editing ? "PATCH" : "POST", body),
    onSuccess: async (release) => {
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
    },
  });
  const generating = useMutation({
    mutationFn: () =>
      api<{ notes: string }>(
        `${base}/notes?${new URLSearchParams({ tag: releaseTag.value, target: target.value, prerelease: String(prerelease) })}`,
      ),
    onSuccess: (result) =>
      notes.setValue([notes.value.trim(), result.notes].filter(Boolean).join("\n\n")),
  });
  const removing = useMutation({
    mutationFn: () =>
      write(`${base}/${deleting?.id}${cleanupTag ? "?cleanup_tag=true" : ""}`, "DELETE"),
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
  });
  const removeAsset = useMutation({
    mutationFn: () =>
      write(
        `${base}/${assetDeletion?.release}/assets/${encodeURIComponent(assetDeletion?.name || "")}`,
        "DELETE",
      ),
    onSuccess: async () => {
      document.querySelector<HTMLDialogElement>("#delete-asset")?.close();
      await refresh();
    },
  });
  const upload = useMutation({
    mutationFn: async (form: FormData) => {
      const file = form.get("asset");
      if (!(file instanceof File) || !file.name) {
        throw new Error("Choose a file to upload.");
      }
      return api(
        `${base}/${formValue(form, "release")}/assets?${new URLSearchParams({ name: file.name, clobber: String(form.has("clobber")) })}`,
        undefined,
        {
          method: "POST",
          body: file,
          headers: { "Content-Type": file.type || "application/octet-stream" },
        },
      );
    },
    onSuccess: async () => {
      document.querySelector<HTMLFormElement>("#asset-upload")?.reset();
      await refresh();
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
    saving.reset();
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
  function saveRelease(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const body: ReleaseEditor = { title: title.value, notes: notes.value, draft, prerelease };
    if (!editing) {
      body.tag = releaseTag.value;
    }
    if (!editing || editing.draft) {
      body.target = target.value;
    }
    saving.mutate(body);
  }
  function generateNotes() {
    generating.mutate();
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
    removing.mutate();
  }
  function prepareAssetDelete(event: MouseEvent<HTMLButtonElement>) {
    setAssetDeletion({
      release: Number(event.currentTarget.dataset.release),
      name: event.currentTarget.dataset.name || "",
    });
    removeAsset.reset();
  }
  function deleteAsset() {
    removeAsset.mutate();
  }
  function uploadAsset(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    upload.mutate(new FormData(event.currentTarget));
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
      <CodeNavigation
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        active=""
      />
      <div className={styles.toolbar}>
        {tag ? (
          <Link
            className={styles.link}
            to="/$owner/$repo/$section"
            params={{ owner, repo, section: "releases" }}
            search={{}}
          >
            Releases
          </Link>
        ) : (
          <>
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
          </>
        )}
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
            <Plus size={15} />
            New release
          </button>
        )}
      </div>
      {error && <ErrorNotice error={error} retry={retry} />}
      {(tag ? single.isPending : list.isPending) && (
        <output className={styles.empty}>Loading releases…</output>
      )}
      {shown.map((release) => (
        <article className={styles.panel} key={release.id}>
          <div className={styles.panelHeading}>
            <div>
              <Link
                className={styles.rowTitle}
                to="/$owner/$repo/$section/$"
                params={{ owner, repo, section: "releases", _splat: `tag/${release.tag}` }}
                search={{}}
              >
                <Tags size={16} /> {release.tag} · {release.title}
              </Link>
              <p className={styles.meta}>
                <ActorName actor={release.author} /> ·{" "}
                <DateTime value={release.published_at || release.created_at} /> ·{" "}
                <code>{release.target}</code>
              </p>
            </div>
            <div className={styles.headingActions}>
              {release.draft && <span className={styles.label}>Draft</span>}
              {release.prerelease && <span className={styles.count}>Prerelease</span>}
              {canWrite && (
                <>
                  <button
                    className={controls.button}
                    type="button"
                    data-id={release.id}
                    onClick={prepareEditor}
                    commandfor="release-editor"
                    command="show-modal"
                    aria-label={`Edit ${release.tag}`}
                  >
                    <Pencil size={14} />
                  </button>
                  <button
                    className={controls.button}
                    type="button"
                    data-id={release.id}
                    onClick={prepareDelete}
                    commandfor="delete-release"
                    command="show-modal"
                    aria-label={`Delete ${release.tag}`}
                  >
                    <Trash2 size={14} />
                  </button>
                </>
              )}
            </div>
          </div>
          {release.notes ? (
            <RichText source={release.notes} />
          ) : (
            <p className={styles.empty}>No release notes.</p>
          )}
          {(release.assets.length > 0 || tag) && (
            <section className={delivery.assets}>
              <h2>
                Assets <span className={styles.count}>{release.assets.length}</span>
              </h2>
              {release.assets.map((asset) => (
                <div className={styles.row} key={asset.id}>
                  <Download size={16} />
                  <div className={styles.grow}>
                    <a href={asset.download_url} className={styles.link}>
                      {asset.name}
                    </a>
                    <p className={styles.meta}>
                      <code>sha256:{asset.sha256}</code>
                    </p>
                  </div>
                  <span className={styles.meta}>
                    {new Intl.NumberFormat(undefined, {
                      style: "unit",
                      unit: "byte",
                      unitDisplay: "narrow",
                      notation: "compact",
                    }).format(asset.size)}
                  </span>
                  <CopyButton
                    value={asset.sha256}
                    label={`Copy checksum for ${asset.name}`}
                    compact
                  />
                  {canWrite && (
                    <button
                      className={controls.button}
                      type="button"
                      data-name={asset.name}
                      data-release={release.id}
                      onClick={prepareAssetDelete}
                      commandfor="delete-asset"
                      command="show-modal"
                      aria-label={`Delete asset ${asset.name}`}
                    >
                      <Trash2 size={14} />
                    </button>
                  )}
                </div>
              ))}
              {release.assets.length === 0 && <p className={styles.meta}>No uploaded assets.</p>}
              {tag && canWrite && (
                <form id="asset-upload" onSubmit={uploadAsset}>
                  <input type="hidden" name="release" value={release.id} />
                  <label className={styles.field}>
                    <span>Upload asset</span>
                    <input type="file" name="asset" required disabled={upload.isPending} />
                  </label>
                  <label className={styles.checkbox}>
                    <input type="checkbox" name="clobber" />
                    Replace an asset with the same name
                  </label>
                  <FormError error={upload.error} />
                  <button type="submit" className={controls.button} disabled={upload.isPending}>
                    {upload.isPending ? "Uploading…" : "Upload asset"}
                  </button>
                </form>
              )}
            </section>
          )}
          {!!release.images?.length && (
            <section className={delivery.assets}>
              <h2>Containers linked to {release.tag}</h2>
              {release.images.map((image) => (
                <div className={styles.row} key={`${image.name}:${image.tag}`}>
                  <Package size={16} />
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
      <ConfirmDialog
        id="delete-asset"
        title="Delete release asset?"
        confirm="Delete asset"
        onConfirm={deleteAsset}
        pending={removeAsset.isPending}
        error={removeAsset.error}
        danger
      >
        <p>{assetDeletion?.name} will be removed from the release.</p>
      </ConfirmDialog>
    </>
  );
}

export function Containers() {
  const { owner = "", repo = "" } = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const navigate = useNavigate({ from: "/$owner/$repo/$section" });
  const name = `${owner}/${repo}`;
  const images = useQuery({
    queryKey: [repositoryAPI(name, "/images")],
    queryFn: ({ signal }) => api<Image[]>(repositoryAPI(name, "/images"), signal),
  });
  const releases = useQuery({
    queryKey: [repositoryAPI(name, "/releases")],
    queryFn: ({ signal }) => api<Release[]>(repositoryAPI(name, "/releases"), signal),
  });
  const metadata = useQuery({
    queryKey: ["repository", name],
    queryFn: ({ signal }) => api<Repository>(repositoryAPI(name), signal),
  });
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
      <CodeNavigation
        owner={owner}
        repo={repo}
        reference={metadata.data?.default_branch || "main"}
        active=""
      />
      <div className={styles.toolbar}>
        <Package size={18} />
        <strong>Containers</strong>
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
              <Package size={20} />
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
