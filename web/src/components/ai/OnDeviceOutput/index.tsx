import { type ReactNode } from "react";
import { FormError } from "@/components/forms/FormError";
import { RichText } from "@/components/markdown/RichText";
import "@/styles/controls.css";
import "@/styles/workspace.css";
import "./styles.css";

type OnDeviceOutputProps = {
  title: string;
  output: string;
  progress: number | null;
  error: Error | null;
  pending: boolean;
  onStop: () => void;
  onDiscard: () => void;
  content?: ReactNode;
  children?: ReactNode;
};

export function OnDeviceOutput({
  title,
  output,
  progress,
  error,
  pending,
  onStop,
  onDiscard,
  content: result,
  children,
}: OnDeviceOutputProps) {
  if (!pending && !output && !error) {
    return null;
  }
  let content;
  if (progress !== null) {
    content = (
      <label className="lyra-onDevice-download">
        Downloading the model to this device…
        <progress value={progress} />
      </label>
    );
  } else if (result && output) {
    content = result;
  } else if (pending && !output) {
    content = (
      <output className="lyra-workspace-plainText lyra-workspace-muted">Generating…</output>
    );
  } else if (pending) {
    content = <div className="lyra-workspace-plainText">{output}</div>;
  } else {
    content = <RichText source={output} />;
  }
  return (
    <section className="lyra-workspace-panel lyra-onDevice" aria-label={title} aria-busy={pending}>
      <div className="lyra-workspace-panelHeading">
        <div>
          <strong>{title}</strong>
          <p className="lyra-workspace-meta">Generated on this device. Check it before using it.</p>
        </div>
        <div className="lyra-onDevice-actions">
          {!pending && output && children}
          {pending ? (
            <button className="lyra-controls-button" type="button" onClick={onStop}>
              Stop
            </button>
          ) : (
            <button className="lyra-controls-button" type="button" onClick={onDiscard}>
              Discard
            </button>
          )}
        </div>
      </div>
      {content}
      <FormError error={error} />
    </section>
  );
}
