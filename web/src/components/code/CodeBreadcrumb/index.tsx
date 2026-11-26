import { type ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import "./styles.css";

type CodeBreadcrumbProps = {
  owner: string;
  repo: string;
  reference: string;
  section: "commits" | "tags" | "branches" | "compare" | "releases" | "images";
  label: string;
  current?: string;
  children?: ReactNode;
};

export function CodeBreadcrumb({
  owner,
  repo,
  reference,
  section,
  label,
  current,
  children,
}: CodeBreadcrumbProps) {
  return (
    <header className="lyra-workspace-crumbs">
      <nav aria-label="Breadcrumb">
        {current && section === "commits" && (
          <Link
            to="/$owner/$repo/commits/$"
            params={{ owner, repo, _splat: reference }}
            search={{}}
          >
            {label}
          </Link>
        )}
        {current && section !== "commits" && (
          <Link to={`/$owner/$repo/${section}`} params={{ owner, repo }} search={{}}>
            {label}
          </Link>
        )}
        {current ? (
          <>
            <span>/</span>
            <strong>{current}</strong>
          </>
        ) : (
          <strong>{label}</strong>
        )}
      </nav>
      {children}
    </header>
  );
}
