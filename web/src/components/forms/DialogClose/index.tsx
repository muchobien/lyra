import { XIcon } from "@primer/octicons-react";
import "./styles.css";

type DialogCloseProps = { target: string };

export function DialogClose({ target }: DialogCloseProps) {
  return (
    <button
      type="button"
      className="lyra-workspace-close"
      commandfor={target}
      command="request-close"
      aria-label="Close dialog"
    >
      <XIcon size={16} />
    </button>
  );
}
