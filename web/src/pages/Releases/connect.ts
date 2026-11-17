import { useLatestRelease } from "@/api/releases";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useContext, useState, type ChangeEvent, type FormEvent, type MouseEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { IdentityContext, type Release } from "@/api";
import { type ReleaseWrite } from "@/api/generated/models";
import {
  useListReleases,
  useGetRepo,
  useGetReleaseByTag,
  useCreateRelease,
  useEditRelease,
  usePreviewReleaseNotes,
  useDeleteRelease,
} from "@/api/generated/lyra";
import { useDraft } from "@/hooks/useDraft";

export type ReleasesProps = { tag: string };

export function useConnect({ tag }: Pick<ReleasesProps, "tag">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ strict: false });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const list = useListReleases(owner, repo);
  const metadata = useGetRepo(owner, repo);
  const single = useGetReleaseByTag(owner, repo, tag, { query: { enabled: !!tag } });
  const latest = useLatestRelease(owner, repo);
  const [editing, setEditing] = useState<Release | null>(null);
  const [draft, setDraft] = useState(true);
  const [editorVersion, setEditorVersion] = useState(0);
  const [deleting, setDeleting] = useState<Release | null>(null);
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
      to: "/$owner/$repo/releases/tag/$",
      params: { owner, repo, _splat: release.tag },
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
            to: "/$owner/$repo/releases",
            params: { owner, repo },
            search: {},
          });
        }
      },
    },
  });
  useDocumentTitle(`${single.data?.title || "Releases"} · ${name}`);
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
    setEditorVersion(editorVersion + 1);
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
  }
  function saveRelease(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const body = {
      title: title.value,
      notes: notes.value,
      draft,
      prerelease: form.has("prerelease"),
    };
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
  function generateNotes(event: MouseEvent<HTMLButtonElement>) {
    if (!event.currentTarget.form) {
      return;
    }
    const form = new FormData(event.currentTarget.form);
    generating.mutate({
      owner,
      repo,
      params: { tag: releaseTag.value, target: target.value, prerelease: form.has("prerelease") },
    });
  }
  function prepareDelete(event: MouseEvent<HTMLButtonElement>) {
    setDeleting(
      list.data?.find((release) => release.id === Number(event.currentTarget.dataset.id)) || null,
    );
    document.querySelector<HTMLDialogElement>("#delete-release")?.querySelector("form")?.reset();
    removing.reset();
  }
  function deleteRelease(form: FormData) {
    if (deleting) {
      removing.mutate({
        owner,
        repo,
        id: deleting.id,
        params: { cleanup_tag: form.has("cleanup_tag") },
      });
    }
  }
  function filtersChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    void navigate({
      to: "/$owner/$repo/releases",
      params: { owner, repo },
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

  return {
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
  };
}
