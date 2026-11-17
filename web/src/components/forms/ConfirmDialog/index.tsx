import "@/styles/controls.css";
import "@/components/forms/FormDialog/styles.css";
import "@/styles/workspace.css";
import { FormError } from "@/components/forms/FormError";
import { useLogic, type DialogProps } from "./logic";

export function ConfirmDialog({
  id,
  title,
  children,
  confirm,
  onConfirm,
  pending = false,
  error = null,
  danger = false,
}: DialogProps) {
  const { submit } = useLogic({ onConfirm });
  return (
    <dialog
      id={id}
      className="lyra-workspace-dialog"
      closedby="any"
      aria-labelledby={`${id}-title`}
    >
      <form onSubmit={submit}>
        <h2 id={`${id}-title`}>{title}</h2>
        {children}
        <FormError error={error} />
        <div className="lyra-workspace-dialogActions">
          <button
            type="button"
            className="lyra-controls-button"
            commandfor={id}
            command="close"
            disabled={pending}
          >
            Cancel
          </button>
          <button
            type="submit"
            className={danger ? "lyra-workspace-danger" : "lyra-workspace-primary"}
            disabled={pending}
          >
            {pending ? "Working…" : confirm}
          </button>
        </div>
      </form>
    </dialog>
  );
}
