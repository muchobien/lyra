import { Link } from "@tanstack/react-router";
import { type Checks } from "@/api";
import { Status } from "@/components/status/Status";
import { RunState } from "@/components/status/RunState";
import "@/styles/workspace.css";
import "./styles.css";

type ChecksProps = { checks: Checks; owner: string; repo: string };

export function ChecksPanel({ checks, owner, repo }: ChecksProps) {
  return (
    <section className="lyra-workspace-panel lyra-workspace-checks" aria-label="Checks">
      <div className="lyra-workspace-panelHeading">
        <h2>Checks</h2>
        <Status value={checks.state} />
      </div>

      {checks.jobs.map((check) => (
        <Link
          className="lyra-workspace-row"
          key={`${check.run}-${check.job_id}`}
          to="/$owner/$repo/runs/$number"
          params={{ owner, repo, number: check.run }}
          search={{ job: String(check.job_id) }}
        >
          <RunState status={check.status} />
          <span className="lyra-workspace-grow">
            <strong>{check.name}</strong>{" "}
            <span className="lyra-workspace-muted">· {check.workflow.split("/").at(-1)}</span>
          </span>
          <span className="lyra-workspace-link">View logs</span>
        </Link>
      ))}
      {checks.jobs.length === 0 && (
        <p className="lyra-workspace-empty">No checks have run for this commit.</p>
      )}
    </section>
  );
}
