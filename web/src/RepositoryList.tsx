import { useListOwners } from "./api/generated/lyra";
import { useEffect } from "react";
import { Link, useParams } from "@tanstack/react-router";
import { BookMarked } from "lucide-react";
import { ErrorNotice } from "./Controls";
import styles from "./RepositoryList.module.css";

export function RepositoryList() {
  const { owner } = useParams({ strict: false });
  const owners = useListOwners();
  useEffect(() => {
    document.title = `${owner || "Repositories"} · Lyra`;
  }, [owner]);
  function retry() {
    void owners.refetch();
  }
  if (owners.error) {
    return <ErrorNotice error={owners.error} retry={retry} />;
  }
  if (owners.isPending) {
    return <output className={styles.loading}>Loading repositories…</output>;
  }
  const groups = owners.data.filter((group) => !owner || group.name === owner);
  return (
    <div className={styles.list}>
      <h1>{owner || "Repositories"}</h1>
      {groups.map((group) => (
        <section key={group.name} className={styles.group}>
          {!owner && (
            <h2>
              <Link to="/$owner" params={{ owner: group.name }}>
                {group.name}
              </Link>
            </h2>
          )}
          {group.description && <p>{group.description}</p>}
          <div className={styles.repositories}>
            {group.repos.map((repository) => (
              <Link
                className={styles.repository}
                key={repository.name}
                to="/$owner/$repo"
                params={{ owner: group.name, repo: repository.name.split("/")[1] }}
                search={{}}
              >
                <BookMarked size={20} />
                <div>
                  <strong>{repository.name.split("/")[1]}</strong>
                  {repository.description && <p>{repository.description}</p>}
                </div>
              </Link>
            ))}
          </div>
          {group.repos.length === 0 && (
            <p className={styles.loading}>No repositories in this owner yet.</p>
          )}
        </section>
      ))}
      {groups.length === 0 && <p className={styles.loading}>No repositories available.</p>}
    </div>
  );
}
