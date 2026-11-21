import { useContext, type ChangeEvent, type MouseEvent, type SubmitEvent } from "react";
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
  listCommits,
} from "@/api/generated/lyra";
import { formValue } from "@/utils/forms";
import { useDraft } from "@/hooks/useDraft";
import { itemListPaths, itemDetailPaths } from "@/routing/paths";
import { itemListParams } from "@/routing/search";
import { repositoryQueries } from "@/api/repositoryQueries";
import { englishText, useOnDeviceModel } from "@/hooks/useOnDeviceModel";

export type ItemListProps = { kind: "pulls" | "issues" };

export function useConnect({ kind }: Pick<ItemListProps, "kind">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: `${itemListPaths[kind]}/` });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const identity = useContext(IdentityContext);

  const name = `${owner}/${repo}`;
  const canWrite = identity?.scope !== "read";
  const params = itemListParams(search);
  const { state, limit } = params;

  const issues = useListIssues(owner, repo, params, { query: { enabled: kind === "issues" } });
  const pulls = useListPulls(owner, repo, params, { query: { enabled: kind === "pulls" } });
  const items = kind === "pulls" ? pulls : issues;
  const listedItems: (Item | Pull)[] = items.data || [];
  const labels = useListLabels(owner, repo);
  const metadata = useGetRepo(owner, repo);

  const title = useDraft(`${identity?.actor.user}/${name}/${kind}/new/title`);
  const description = useDraft(`${identity?.actor.user}/${name}/${kind}/new/body`);

  const draft = useOnDeviceModel("language-model", async () =>
    "LanguageModel" in self ? LanguageModel.availability(englishText) : "unavailable",
  );

  const labelSuggestion = useOnDeviceModel("language-model", async () =>
    "LanguageModel" in self ? LanguageModel.availability(englishText) : "unavailable",
  );
  const suggestedLabels: string[] = labelSuggestion.output
    ? JSON.parse(labelSuggestion.output).labels
    : [];

  function created(item: ItemWrite | PullWrite) {
    title.clear();
    description.clear();
    document.querySelector<HTMLDialogElement>("#new-item")?.close();
    void queryClient.invalidateQueries(repositoryQueries(owner, repo));
    void navigate({
      to: itemDetailPaths[kind],
      params: { owner, repo, number: item.number },
      search: {},
    });
  }

  const issueCreation = useCreateIssue({ mutation: { onSuccess: created } });
  const pullCreation = useCreatePull({ mutation: { onSuccess: created } });
  const creation = kind === "pulls" ? pullCreation : issueCreation;

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

  function preventSearchSubmit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
  }

  function titleChanged(event: ChangeEvent<HTMLInputElement>) {
    title.setValue(event.target.value);
  }

  function draftDescription(event: MouseEvent<HTMLButtonElement>) {
    if (!event.currentTarget.form) {
      return;
    }
    const form = new FormData(event.currentTarget.form);
    const base = formValue(form, "base");
    const head = formValue(form, "head");
    void draft.run(async (signal, monitor) => {
      const [headCommits, baseCommits] = await Promise.all([
        listCommits(owner, repo, { ref: head, limit: 100 }, { signal }),
        listCommits(owner, repo, { ref: base, limit: 500 }, { signal }),
      ]);
      const merged = new Set(baseCommits.map((commit) => commit.sha));
      const commits = headCommits.filter((commit) => !merged.has(commit.sha));
      if (!commits.length) {
        throw new Error(`${head} has no commits that aren't in ${base}.`);
      }
      const session = await LanguageModel.create({
        ...englishText,
        initialPrompts: [
          {
            role: "system",
            content:
              "You write pull request descriptions in Markdown from a list of commits. Use a '## Summary' section with one to three sentences on what changes and why, then a '## Changes' section with one bullet per meaningful change. Only state what the commits support. Output only the description.",
          },
        ],
        signal,
        monitor,
      });
      const log = commits
        .map((commit) => [`- ${commit.subject}`, commit.body.trim()].filter(Boolean).join("\n\n"))
        .join("\n");
      return session.promptStreaming(
        `Pull request${title.value ? ` "${title.value}"` : ""} merging ${head} into ${base}. Commits, newest first:\n\n${log}`,
        { signal },
      );
    });
  }

  function suggestLabels() {
    const available = labels.data || [];
    void labelSuggestion.run(async (signal, monitor) => {
      const session = await LanguageModel.create({
        ...englishText,
        initialPrompts: [
          {
            role: "system",
            content: `You pick labels for a new ${kind === "pulls" ? "pull request" : "issue"} from the repository's labels. Pick only labels that clearly apply, at most three, and none when no label fits. The labels are:\n${available
              .map((label) => `- ${label.name}${label.description ? `: ${label.description}` : ""}`)
              .join("\n")}`,
          },
        ],
        signal,
        monitor,
      });
      const result = await session.prompt(
        `Title: ${title.value}\n\nDescription:\n${description.value || "None."}`,
        {
          signal,
          responseConstraint: {
            type: "object",
            properties: {
              labels: {
                type: "array",
                maxItems: 3,
                items: { type: "string", enum: available.map((label) => label.name) },
              },
            },
            required: ["labels"],
          },
        },
      );
      return (async function* () {
        yield result;
      })();
    });
  }

  function applyLabels() {
    const input = document.querySelector<HTMLInputElement>('#new-item input[name="labels"]');
    if (input) {
      const current = input.value
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean);
      input.value = [...new Set([...current, ...suggestedLabels])].join(", ");
    }
    labelSuggestion.reset();
  }

  function insertDraft() {
    description.setValue([description.value.trim(), draft.output].filter(Boolean).join("\n\n"));
    draft.reset();
  }

  function submit(event: SubmitEvent<HTMLFormElement>) {
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
    draft,
    labelSuggestion,
    suggestedLabels,
    retry,
    filtersChanged,
    preventSearchSubmit,
    titleChanged,
    draftDescription,
    insertDraft,
    suggestLabels,
    applyLabels,
    submit,
  };
}
