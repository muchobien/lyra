import { Link } from "@tanstack/react-router";
import { RepoIcon } from "@primer/octicons-react";
import { ErrorNotice } from "@/components/controls/ErrorNotice";
import "./styles.css";
import { useConnect } from "./connect";

export function RepositoryList({ owner }: { owner?: string }) {
  const { groups, error, retry } = useConnect(owner);
  return (
    <div className="lyra-repository-list-list">
      <h1>{owner || "Repositories"}</h1>
      {error && <ErrorNotice error={error} retry={retry} />}
      {groups.map((group) => (
        <section key={group.name} className="lyra-repository-list-group">
          {!owner && (
            <h2>
              <Link to="/$owner" params={{ owner: group.name }}>
                {group.name}
              </Link>
            </h2>
          )}
          {group.description && <p>{group.description}</p>}
          <div className="lyra-repository-list-repositories">
            {group.repos.map((repository) => (
              <Link
                className="lyra-repository-list-repository"
                key={repository.name}
                to="/$owner/$repo"
                params={{ owner: group.name, repo: repository.name.split("/")[1] }}
                search={{}}
              >
                <RepoIcon size={16} />
                <div>
                  <strong>{repository.name.split("/")[1]}</strong>
                  {repository.description && <p>{repository.description}</p>}
                </div>
              </Link>
            ))}
          </div>
          {group.repos.length === 0 && (
            <p className="lyra-repository-list-loading">No repositories in this owner yet.</p>
          )}
        </section>
      ))}
      {groups.length === 0 && (
        <p className="lyra-repository-list-loading">No repositories available.</p>
      )}
    </div>
  );
}
