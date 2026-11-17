import { CheckIcon, CopyIcon } from "@primer/octicons-react";
import "@/styles/controls.css";
import { useLogic, type CopyButtonProps } from "./logic";

export function CopyButton({ value, label, compact = false }: CopyButtonProps) {
  const { copied, error, copy } = useLogic({ value });
  return (
    <>
      <button
        className="lyra-controls-button"
        type="button"
        onClick={copy}
        aria-label={copied ? "Copied" : label}
        title={label}
      >
        {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}{" "}
        {!compact && (copied ? "Copied" : label)}
      </button>
      <output className="lyra-controls-srOnly">{copied ? "Copied to clipboard." : error}</output>
    </>
  );
}
