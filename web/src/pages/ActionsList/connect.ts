import { useContext, type ChangeEvent, type SubmitEvent } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { IdentityContext } from "@/api";
import {
  useGetRepo,
  useListWorkflows,
  useListTags,
  useListRuns,
  useDispatchWorkflow,
} from "@/api/generated/lyra";
import { formValue } from "@/utils/forms";
import { repositoryQueries } from "@/api/repositoryQueries";
import { searchLimit } from "@/routing/search";

export function useConnect() {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: "/$owner/$repo/runs/" });
  const navigate = useNavigate({ from: "/$owner/$repo/runs/" });
  const identity = useContext(IdentityContext);
  const queryClient = useQueryClient();
  const limit = searchLimit(search.limit, 50);
  const metadata = useGetRepo(owner, repo);
  const reference = search.ref || metadata.data?.default_branch || "main";
  const workflows = useListWorkflows(
    owner,
    repo,
    { ref: reference },
    { query: { enabled: !!metadata.data } },
  );
  const tags = useListTags(owner, repo);
  const runs = useListRuns(
    owner,
    repo,
    { limit },
    {
      query: {
        refetchInterval: (query) =>
          query.state.data?.some((run) => ["waiting", "running", "blocked"].includes(run.status))
            ? 3000
            : false,
      },
    },
  );
  const selected = workflows.data?.find((workflow) => workflow.path === search.workflow);
  const visible =
    runs.data?.filter(
      (run) =>
        (!search.workflow || run.workflow === search.workflow) &&
        (!search.state || run.status === search.state) &&
        (!search.q ||
          `${run.title} ${run.name} ${run.number}`.toLowerCase().includes(search.q.toLowerCase())),
    ) || [];
  const dispatch = useDispatchWorkflow({
    mutation: {
      onSuccess: (run) => {
        document.querySelector<HTMLDialogElement>("#dispatch-workflow")?.close();
        void queryClient.invalidateQueries(repositoryQueries(owner, repo));
        void navigate({
          to: "/$owner/$repo/runs/$number",
          params: { owner, repo, number: run.number },
          search: {},
        });
      },
    },
  });
  function retry() {
    void runs.refetch();
    void workflows.refetch();
  }
  function filtersChanged(event: ChangeEvent<HTMLInputElement | HTMLSelectElement>) {
    void navigate({
      search: { ...search, [event.target.name]: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }
  function dispatchRun(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected) {
      return;
    }
    const form = new FormData(event.currentTarget);
    const inputs: Record<string, string> = {};
    for (const [name, input] of Object.entries(selected.inputs)) {
      inputs[name] =
        input.type === "boolean"
          ? String(form.has(`input:${name}`))
          : formValue(form, `input:${name}`);
    }
    dispatch.mutate({ owner, repo, data: { workflow: selected.path, ref: reference, inputs } });
  }
  function resetDispatch() {
    dispatch.reset();
  }
  const triggerLabels: Record<string, string> = {
    push: "Push",
    pull_request: "Pull request",
    workflow_dispatch: "Manual",
    schedule: "Scheduled",
    release: "Release",
  };

  return {
    owner,
    repo,
    search,
    identity,
    limit,
    metadata,
    reference,
    workflows,
    tags,
    runs,
    selected,
    visible,
    dispatch,
    retry,
    filtersChanged,
    dispatchRun,
    resetDispatch,
    triggerLabels,
  };
}
