import { useState } from "react";

export type CopyButtonProps = { value: string; label: string; compact?: boolean };

export function useLogic({ value }: Pick<CopyButtonProps, "value">) {
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setError("");
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError("Couldn’t copy. Select the command to copy it manually.");
    }
  }

  return { copied, error, copy };
}
