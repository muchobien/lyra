import { type ReactNode } from "react";
import {
  CheckCircleIcon,
  CheckIcon,
  ClockIcon,
  CommentIcon,
  GitMergeIcon,
  GitPullRequestIcon,
  IssueOpenedIcon,
  SkipIcon,
  XCircleIcon,
} from "@primer/octicons-react";
import "./styles.css";

type StatusProps = { value: string; issue?: boolean; children?: ReactNode };

export function Status({ value, issue, children }: StatusProps) {
  const icons = {
    success: CheckCircleIcon,
    failure: XCircleIcon,
    cancelled: XCircleIcon,
    merged: GitMergeIcon,
    approved: CheckIcon,
    changes_requested: XCircleIcon,
    commented: CommentIcon,
    open: GitPullRequestIcon,
    closed: XCircleIcon,
    waiting: ClockIcon,
    blocked: ClockIcon,
    running: ClockIcon,
    pending: ClockIcon,
    skipped: SkipIcon,
  };
  const Icon =
    (issue && value === "open" ? IssueOpenedIcon : icons[value as keyof typeof icons]) ||
    IssueOpenedIcon;
  return (
    <span className="lyra-workspace-status" data-state={value}>
      <Icon size={16} />
      {children || value.replaceAll("_", " ")}
    </span>
  );
}
