import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { repositoryQueries } from "@/api/repositoryQueries";

export function useConnect() {
  const { owner, repo } = useParams({ from: "/$owner/$repo" });
  const queryClient = useQueryClient();

  const [connection, setConnection] = useState("connecting");

  useEffect(() => {
    if (!owner || !repo) {
      return;
    }
    const stream = new EventSource(
      `/api/events/stream?repo=${encodeURIComponent(`${owner}/${repo}`)}`,
    );
    let pending: ReturnType<typeof setTimeout> | undefined;
    function refresh() {
      clearTimeout(pending);
      pending = setTimeout(() => {
        void queryClient.invalidateQueries(repositoryQueries(owner, repo));
      }, 200);
    }
    stream.onopen = () => {
      setConnection("live");
      refresh();
    };
    stream.onerror = () => setConnection("reconnecting");
    for (const action of [
      "push",
      "issue.opened",
      "issue.edited",
      "issue.closed",
      "issue.reopened",
      "issue.comment",
      "issue.labeled",
      "issue.unlabeled",
      "issue.assigned",
      "issue.unassigned",
      "pr.opened",
      "pr.edited",
      "pr.closed",
      "pr.reopened",
      "pr.comment",
      "pr.labeled",
      "pr.unlabeled",
      "pr.assigned",
      "pr.unassigned",
      "pr.review",
      "pr.synchronized",
      "pr.merged",
      "pr.checks",
      "run.queued",
      "run.completed",
      "run.cancelled",
      "job.completed",
      "release.created",
      "release.updated",
      "release.published",
      "release.deleted",
      "release.asset",
      "image.pushed",
      "image.deleted",
      "rule.set",
      "rule.delete",
      "label.create",
      "label.delete",
      "secret.set",
      "secret.delete",
      "variable.set",
      "variable.delete",
    ]) {
      stream.addEventListener(action, refresh);
    }
    return () => {
      clearTimeout(pending);
      stream.close();
    };
  }, [owner, repo, queryClient]);

  return { connection };
}
