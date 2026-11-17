import { type ReactNode, type SubmitEvent } from "react";

export type DialogProps = {
  id: string;
  title: string;
  children: ReactNode;
  confirm: string;
  onConfirm: (form: FormData) => void;
  pending?: boolean;
  error?: Error | null;
  danger?: boolean;
};

export function useLogic({ onConfirm }: Pick<DialogProps, "onConfirm">) {
  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    onConfirm(new FormData(event.currentTarget));
  }

  return { submit };
}
