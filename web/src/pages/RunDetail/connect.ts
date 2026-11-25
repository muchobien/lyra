import {
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type MouseEvent,
  type ToggleEvent,
  type UIEvent,
} from "react";
import { useLocation, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { IdentityContext, request } from "@/api";
import { useGetRun, useCancelRun, useRerunRun, getGetJobLogsUrl } from "@/api/generated/lyra";
import { repositoryQueries } from "@/api/repositoryQueries";
import { englishText, useOnDeviceModel } from "@/hooks/useOnDeviceModel";
import { classify, readableText } from "@/components/code/LogText/logic";

type Logs = { lines: string[]; times: string[]; offset: number; done: boolean };

export type RunDetailProps = { number: number };

export function useConnect({ number }: Pick<RunDetailProps, "number">) {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const search = useSearch({ from: "/$owner/$repo/runs/$number" });
  const navigate = useNavigate({ from: "/$owner/$repo/runs/$number" });
  const hash = useLocation({ select: (location) => location.hash });
  const identity = useContext(IdentityContext);
  const queryClient = useQueryClient();

  const name = `${owner}/${repo}`;
  const run = useGetRun(owner, repo, number, {
    query: {
      refetchInterval: (query) =>
        query.state.data && ["waiting", "blocked", "running"].includes(query.state.data.status)
          ? 2000
          : false,
    },
  });

  const jobs = run.data?.jobs || [];
  const selected =
    jobs.find((job) => String(job.id) === search.job) ||
    jobs.find((job) => job.status === "failure") ||
    jobs.find((job) => job.status === "running") ||
    jobs.toSorted(
      (first, second) =>
        (second.started_at ? Date.parse(second.started_at) : 0) -
        (first.started_at ? Date.parse(first.started_at) : 0),
    )[0];
  const active = !!run.data && ["waiting", "blocked", "running"].includes(run.data.status);
  const logsURL = selected ? getGetJobLogsUrl(owner, repo, number, selected.id) : "";
  const logKey = [logsURL, selected?.attempt || 0, selected?.started_at || "queued"];

  const logs = useQuery({
    queryKey: logKey,
    enabled: !!selected?.started_at,
    queryFn: async ({ signal }) => {
      const previous = queryClient.getQueryData<Logs>(logKey);
      const response = await request(
        getGetJobLogsUrl(owner, repo, number, selected?.id || 0, {
          offset: previous?.offset || 0,
          timestamps: true,
        }),
        signal,
      );
      const text = await response.text();
      const entries = text ? text.replace(/\n$/, "").split("\n") : [];
      return {
        lines: [
          ...(previous?.lines || []),
          ...entries.map((entry) => entry.slice(entry.indexOf(" ") + 1)),
        ],
        times: [
          ...(previous?.times || []),
          ...entries.map((entry) => entry.slice(0, entry.indexOf(" "))),
        ],
        offset: Number(response.headers.get("Lyra-Log-Next") || 0),
        done: response.headers.get("Lyra-Log-Done") === "true",
      };
    },
    refetchInterval: (query) => {
      if (query.state.error || query.state.data?.done) {
        return false;
      }
      return 1000;
    },
  });

  const annotations = useMemo(
    () =>
      (logs.data?.lines || []).flatMap((raw, index) => {
        const entry = classify(raw);
        if (entry.kind !== "annotation") {
          return [];
        }
        const step = selected?.steps.find(
          (candidate) =>
            index >= candidate.log_index && index < candidate.log_index + candidate.log_length,
        );
        return [{ ...entry, line: index + 1, step }];
      }),
    [logs.data?.lines, selected],
  );

  const explanation = useOnDeviceModel("language-model", async () =>
    "LanguageModel" in self ? LanguageModel.availability(englishText) : "unavailable",
  );
  const [explainedJob, setExplainedJob] = useState(0);

  const [following, setFollowing] = useState(true);
  const [timestamps, setTimestamps] = useState(false);
  const [fullLogOpen, setFullLogOpen] = useState(false);
  const logPanel = useRef<HTMLDivElement>(null);
  const targetLine = Number(/^L(\d+)$/.exec(hash)?.[1] || 0);
  const logsLoaded = !!logs.data;
  const locatedLog = useRef("");

  const cancel = useCancelRun({
    mutation: {
      onSuccess: () => {
        document.querySelector<HTMLDialogElement>("#cancel-run")?.close();
        void queryClient.invalidateQueries(repositoryQueries(owner, repo));
      },
    },
  });

  const rerun = useRerunRun({
    mutation: {
      onSuccess: () => {
        document.getElementById("rerun-options")?.hidePopover();
        void queryClient.invalidateQueries(repositoryQueries(owner, repo));
      },
    },
  });

  useEffect(() => {
    if (following && active && logPanel.current) {
      logPanel.current.scrollTop = logPanel.current.scrollHeight;
    }
  }, [logs.data?.offset, following, active]);

  useEffect(() => {
    const element = document.querySelector("[data-target]");
    if (!element) {
      return;
    }
    for (
      let details = element.closest("details");
      details;
      details = details.parentElement?.closest("details") || null
    ) {
      details.open = true;
    }
    element.scrollIntoView({ block: "nearest" });
  }, [targetLine, logsLoaded]);

  useEffect(() => {
    const logID = logKey.join(" ");
    const failed = selected?.steps.some((step) => step.result === "failure");
    if (
      active ||
      !failed ||
      !logs.data?.done ||
      !logPanel.current ||
      locatedLog.current === logID
    ) {
      return;
    }
    locatedLog.current = logID;
    const stepLog = logPanel.current.querySelector(".lyra-actions-step");
    const line =
      stepLog?.querySelector<HTMLElement>('.lyra-log-text-line[data-level="error"]') ||
      [...(stepLog?.querySelectorAll<HTMLElement>(".lyra-log-text-line") || [])].at(-1);
    if (!line) {
      return;
    }
    line.closest("details")?.setAttribute("open", "");
    line.scrollIntoView({ block: "center", container: "nearest" } as ScrollIntoViewOptions);
    line.animate([{ background: "var(--danger-border)" }, { background: "transparent" }], {
      duration: 2000,
      easing: "ease-in",
    });
  });

  function retry() {
    void run.refetch();
  }

  function retryLogs() {
    void logs.refetch();
  }

  function rerunClicked(event: MouseEvent<HTMLButtonElement>) {
    rerun.mutate({
      owner,
      repo,
      number,
      params: { failed: event.currentTarget.dataset.failed === "true" },
    });
  }

  function cancelRun() {
    cancel.mutate({ owner, repo, number });
  }

  function explainFailure() {
    if (!selected || !logs.data) {
      return;
    }
    const step = selected.steps.find((entry) => entry.result === "failure");
    let lines = (
      step
        ? logs.data.lines.slice(step.log_index, step.log_index + step.log_length)
        : logs.data.lines
    ).flatMap((line) => readableText(line) ?? []);
    setExplainedJob(selected.id);
    void explanation.run(async (signal, monitor) => {
      const session = await LanguageModel.create({
        ...englishText,
        initialPrompts: [
          {
            role: "system",
            content:
              "You help a developer understand why a CI job failed. Answer in Markdown: one sentence with the most likely cause, the log lines that show it in a code block, and a suggested fix. Be brief. If the log doesn't show the cause, say so.",
          },
        ],
        signal,
        monitor,
      });
      const prompt = (tail: string[]) =>
        `Job "${selected.name}"${step ? `, step "${step.name}"` : ""}. The end of its log:\n\n${tail.join("\n")}`;
      lines = lines.slice(-400);
      while (
        lines.length > 20 &&
        (await session.measureContextUsage(prompt(lines), { signal })) >
          session.contextWindow - session.contextUsage - 1024
      ) {
        lines = lines.slice(Math.floor(lines.length / 4));
      }
      return session.promptStreaming(prompt(lines), { signal });
    });
  }

  function annotationClicked(event: MouseEvent<HTMLButtonElement>) {
    const { step, line } = event.currentTarget.dataset;
    const panel = logPanel.current;
    const target =
      panel?.querySelector<HTMLElement>(`[data-step="${step}"] [data-line="${line}"]`) ||
      panel?.querySelector<HTMLElement>(`[data-line="${line}"]`);
    if (!panel || !target) {
      return;
    }
    for (
      let details = target.closest("details");
      details;
      details = details.parentElement?.closest("details") || null
    ) {
      details.open = true;
    }
    panel.querySelector("[data-selected]")?.removeAttribute("data-selected");
    target.setAttribute("data-selected", "");
    setFollowing(false);
    target.scrollIntoView({ block: "center" });
  }

  function toggleFollow() {
    setFollowing(!following);
  }

  function logScrolled(event: UIEvent<HTMLDivElement>) {
    const element = event.currentTarget;
    if (element.scrollHeight - element.scrollTop - element.clientHeight > 40) {
      setFollowing(false);
    }
  }

  function timestampsChanged(event: ChangeEvent<HTMLInputElement>) {
    setTimestamps(event.target.checked);
  }

  function fullLogToggled(event: ToggleEvent<HTMLDetailsElement>) {
    setFullLogOpen(event.newState === "open");
  }

  function jobChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({ search: { ...search, job: event.target.value }, resetScroll: false });
    setFollowing(true);
  }

  function searchChanged(event: ChangeEvent<HTMLInputElement>) {
    void navigate({
      search: { ...search, q: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }

  return {
    owner,
    repo,
    search,
    identity,
    name,
    run,
    jobs,
    selected,
    active,
    logsURL,
    logs,
    targetLine,
    annotations,
    following,
    timestamps,
    fullLogOpen,
    logPanel,
    cancel,
    rerun,
    explanation,
    explainedJob,
    retry,
    retryLogs,
    rerunClicked,
    cancelRun,
    explainFailure,
    annotationClicked,
    toggleFollow,
    logScrolled,
    timestampsChanged,
    fullLogToggled,
    jobChanged,
    searchChanged,
  };
}
