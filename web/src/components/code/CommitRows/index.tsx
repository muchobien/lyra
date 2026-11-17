import { Link } from "@tanstack/react-router";
import { type Commit } from "@/api";
import { FileDirectoryIcon, VerifiedIcon } from "@primer/octicons-react";
import { Status } from "@/components/status/Status";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "./styles.css";

type CommitRowsProps = { commits: Commit[]; owner: string; repo: string };

export function CommitRows({ commits, owner, repo }: CommitRowsProps) {
  const dayFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "full" });
  const timeFormat = new Intl.DateTimeFormat(undefined, { timeStyle: "short" });
  return (
    <div className="lyra-workspace-panel lyra-workspace-commits">
      {commits.map((commit, index) => {
        const day = dayFormat.format(new Date(commit.committer.date));
        const previous = commits[index - 1];
        return (
          <div key={commit.sha}>
            {(!previous || dayFormat.format(new Date(previous.committer.date)) !== day) && (
              <h2 className="lyra-workspace-dayHeading">{day}</h2>
            )}
            <article className="lyra-workspace-row">
              <div className="lyra-workspace-grow">
                <Link
                  className="lyra-workspace-rowTitle"
                  to="/$owner/$repo/commit/$sha"
                  params={{ owner, repo, sha: commit.sha }}
                  search={{}}
                >
                  {commit.subject}
                </Link>
                <p className="lyra-workspace-meta">
                  <strong>{commit.author.name}</strong> committed at{" "}
                  <time
                    dateTime={commit.committer.date}
                    title={new Date(commit.committer.date).toLocaleString()}
                  >
                    {timeFormat.format(new Date(commit.committer.date))}
                  </time>
                </p>
              </div>
              {commit.signature.status === "verified" && (
                <span className="lyra-workspace-verified" title="Verified">
                  <VerifiedIcon size={16} />
                  <span className="lyra-controls-srOnly">Verified</span>
                </span>
              )}
              {!["verified", "unverified"].includes(commit.signature.status) && (
                <Status value={commit.signature.status} />
              )}
              <Link
                className="lyra-workspace-sha"
                to="/$owner/$repo/commit/$sha"
                params={{ owner, repo, sha: commit.sha }}
                search={{}}
              >
                {commit.sha.slice(0, 7)}
              </Link>
              <Link
                className="lyra-workspace-browse"
                to="/$owner/$repo"
                params={{ owner, repo }}
                search={{ ref: commit.sha }}
                aria-label={`Browse files at ${commit.sha.slice(0, 7)}`}
                title="Browse files"
              >
                <FileDirectoryIcon size={16} />
              </Link>
            </article>
          </div>
        );
      })}
      {commits.length === 0 && <p className="lyra-workspace-empty">No commits.</p>}
    </div>
  );
}
