import { useState, type ChangeEvent } from "react";

export type EditorProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
  placeholder?: string;
};

export function useLogic({ onChange }: Pick<EditorProps, "onChange">) {
  const [preview, setPreview] = useState(false);
  function change(event: ChangeEvent<HTMLTextAreaElement>) {
    onChange(event.target.value);
  }
  function edit() {
    setPreview(false);
  }
  function showPreview() {
    setPreview(true);
  }

  return { preview, change, edit, showPreview };
}
