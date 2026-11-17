import {
  type ChangeEvent,
  type SubmitEventHandler,
  type ReactNode,
  type ToggleEvent,
  type ToggleEventHandler,
} from "react";
import "@/styles/controls.css";
import "./styles.css";
import "@/styles/workspace.css";
import { FormError } from "@/components/forms/FormError";
import { DialogClose } from "@/components/forms/DialogClose";

type FormDialogProps = {
  id: string;
  title: ReactNode;
  children: ReactNode;
  onSubmit: SubmitEventHandler<HTMLFormElement>;
  onToggle?: ToggleEventHandler<HTMLDialogElement>;
  formKey?: string;
  fieldsKey?: string;
  description?: ReactNode;
  pending: boolean;
  error: Error | null;
  submitLabel: ReactNode;
  submitDisabled?: boolean;
};

export function FormDialog({
  id,
  title,
  children,
  onSubmit,
  onToggle,
  formKey,
  fieldsKey,
  description,
  pending,
  error,
  submitLabel,
  submitDisabled = false,
}: FormDialogProps) {
  function markDirty(event: ChangeEvent<HTMLFormElement>) {
    event.currentTarget.dataset.dirty = "";
  }
  function toggled(event: ToggleEvent<HTMLDialogElement>) {
    if (event.newState === "closed") {
      event.currentTarget.querySelector("form")?.removeAttribute("data-dirty");
    }
    onToggle?.(event);
  }
  return (
    <dialog
      id={id}
      className="lyra-workspace-dialog"
      closedby="any"
      aria-labelledby={`${id}-title`}
      onToggle={toggled}
    >
      <form key={formKey} onChange={markDirty} onSubmit={onSubmit}>
        <h2 id={`${id}-title`}>{title}</h2>
        {description}
        <fieldset key={fieldsKey} disabled={pending}>
          {children}
          <FormError error={error} />
          <div className="lyra-workspace-dialogActions">
            <button className="lyra-controls-button" type="button" commandfor={id} command="close">
              Cancel
            </button>
            <button className="lyra-workspace-primary" type="submit" disabled={submitDisabled}>
              {submitLabel}
            </button>
          </div>
        </fieldset>
        <DialogClose target={id} />
      </form>
    </dialog>
  );
}
