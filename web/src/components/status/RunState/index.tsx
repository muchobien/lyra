import {
  BlockedIcon,
  CheckCircleFillIcon,
  ClockIcon,
  SkipIcon,
  StopIcon,
  XCircleFillIcon,
} from "@primer/octicons-react";
import "@/styles/controls.css";
import "@/components/status/Status/styles.css";
import "./styles.css";

const runIcons = {
  success: CheckCircleFillIcon,
  failure: XCircleFillIcon,
  waiting: ClockIcon,
  blocked: BlockedIcon,
  cancelled: StopIcon,
  skipped: SkipIcon,
};

type RunStateProps = { status: string };

export function RunState({ status }: RunStateProps) {
  const RunIcon = runIcons[status as keyof typeof runIcons] || ClockIcon;
  return (
    <span
      className="lyra-workspace-status lyra-workspace-runState"
      data-state={status}
      title={status}
    >
      {status === "running" ? (
        <svg className="lyra-workspace-spinner" viewBox="0 0 16 16" width="16" height="16">
          <circle cx="8" cy="8" r="6.25" />
          <path d="M8 1.75a6.25 6.25 0 0 1 6.25 6.25" />
        </svg>
      ) : (
        <RunIcon size={16} />
      )}
      <span className="lyra-controls-srOnly">{status}</span>
    </span>
  );
}
