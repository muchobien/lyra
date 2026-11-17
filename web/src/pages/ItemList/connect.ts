import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useContext, type ChangeEvent, type FormEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { IdentityContext } from "@/api";
import {
  type CreateIssueRequest,
  type ItemWrite,
  type PullWrite,
  type Item,
  type Pull,
} from "@/api/generated/models";
import {
  useListIssues,
  useListPulls,
  useListLabels,
  useGetRepo,
  useCreateIssue,
  useCreatePull,
} from "@/api/generated/lyra";
import { formValue } from "@/utils/forms";
import { useDraft } from "@/hooks/useDraft";
import { itemListPaths, itemDetailPaths } from "@/routing/paths";

export type ItemListProps = { kind: "pulls" | "issues" };

export function useConnect({ kind }: Pick<ItemListProps, "kind">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: `${itemListPaths[kind]}/` });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);
  const name = `${owner}/${repo}`;
  const canWrite = identity?.scope !== "read";
  const state = search.state || "open";
  const limit = Math.max(50, Math.min(1000, Number(search.limit) || 50));
  const params = {
    state,
    limit,
    q: search.q,
    label: search.label ? [search.label] : undefined,
    assignee: search.assignee ? [search.assignee] : undefined,
  };
  const issues = useListIssues(owner, repo, params, { query: { enabled: kind === "issues" } });
  const pulls = useListPulls(owner, repo, params, { query: { enabled: kind === "pulls" } });
  const items = kind === "pulls" ? pulls : issues;
  const listedItems: (Item | Pull)[] = items.data || [];
  const labels = useListLabels(owner, repo);
  const metadata = useGetRepo(owner, repo);
  const title = useDraft(`${identity?.actor.user}/${name}/${kind}/new/title`);
  const description = useDraft(`${identity?.actor.user}/${name}/${kind}/new/body`);
  function created(item: ItemWrite | PullWrite) {
    title.clear();
    description.clear();
    document.querySelector<HTMLDialogElement>("#new-item")?.close();
    void queryClient.invalidateQueries();
    void navigate({
      to: itemDetailPaths[kind],
      params: { owner, repo, number: String(item.number) },
      search: {},
    });
  }
  const issueCreation = useCreateIssue({ mutation: { onSuccess: created } });
  const pullCreation = useCreatePull({ mutation: { onSuccess: created } });
  const creation = kind === "pulls" ? pullCreation : issueCreation;
  useDocumentTitle(`${kind === "pulls" ? "Pull requests" : "Issues"} · ${name}`);
  function retry() {
    void items.refetch();
  }
  function filtersChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    if (!event.currentTarget.form) {
      return;
    }
    const form = new FormData(event.currentTarget.form);
    void navigate({
      to: itemListPaths[kind],
      params: { owner, repo },
      search: {
        state,
        q: formValue(form, "q"),
        label: formValue(form, "label"),
        assignee: formValue(form, "assignee"),
      },
      replace: true,
      resetScroll: false,
    });
  }
  function preventSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
  }
  function titleChanged(event: ChangeEvent<HTMLInputElement>) {
    title.setValue(event.target.value);
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const body: CreateIssueRequest = {
      title: title.value,
      body: description.value,
      labels: formValue(form, "labels")
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
      assignees: formValue(form, "assignees")
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean),
    };
    if (kind === "pulls") {
      pullCreation.mutate({
        owner,
        repo,
        data: { ...body, base: formValue(form, "base"), head: formValue(form, "head") },
      });
    } else {
      issueCreation.mutate({ owner, repo, data: body });
    }
  }

  return {
    owner,
    repo,
    search,
    canWrite,
    state,
    limit,
    items,
    listedItems,
    labels,
    metadata,
    title,
    description,
    creation,
    retry,
    filtersChanged,
    preventSearchSubmit,
    titleChanged,
    submit,
  };
}
