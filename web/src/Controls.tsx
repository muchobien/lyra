import { useState } from "react";
import { Copy, Check } from "lucide-react";
import styles from "./Controls.module.css";

type CopyButtonProps = { value: string; label: string; compact?: boolean };

export function CopyButton({ value, label, compact = false }: CopyButtonProps) {
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
  return (
    <>
      <button
        className={styles.button}
        type="button"
        onClick={copy}
        aria-label={copied ? "Copied" : label}
        title={label}
      >
        {copied ? <Check size={15} /> : <Copy size={15} />}{" "}
        {!compact && (copied ? "Copied" : label)}
      </button>
      <output className={styles.srOnly}>{copied ? "Copied to clipboard." : error}</output>
    </>
  );
}

type NoticeProps = { error: Error; retry: () => void };

export function ErrorNotice({ error, retry }: NoticeProps) {
  return (
    <section className={styles.notice} role="alert">
      <h2>Couldn’t load this view</h2>
      <p>{error.message}</p>
      <button className={styles.button} type="button" onClick={retry}>
        Retry
      </button>
    </section>
  );
}
