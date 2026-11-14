import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { Copy, Check } from "lucide-react";
import { APIError } from "./api";
import styles from "./Controls.module.css";

type ScrollNavigationProps = {
  label: string;
  active: string;
  className: string;
  children: ReactNode;
};
export function ScrollNavigation({ label, active, className, children }: ScrollNavigationProps) {
  const navigation = useRef<HTMLElement>(null);
  useEffect(() => {
    const element = navigation.current;
    const selected = element?.querySelector<HTMLElement>('[aria-current="page"]');
    if (!element || !selected || element.scrollWidth <= element.clientWidth) {
      return;
    }
    const offset = selected.getBoundingClientRect().left - element.getBoundingClientRect().left;
    element.scrollLeft += offset - (element.clientWidth - selected.clientWidth) / 2;
  }, [active]);
  return (
    <nav ref={navigation} className={className} aria-label={label}>
      {children}
    </nav>
  );
}

export function formValue(form: FormData, name: string) {
  const value = form.get(name);
  return typeof value === "string" ? value : "";
}

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

type NoticeProps = { error: Error; retry: () => void; title?: string };

export function ErrorNotice({ error, retry, title = "Couldn’t load this view" }: NoticeProps) {
  return (
    <section className={styles.notice} role="alert">
      <h2>{title}</h2>
      <p>{error.message}</p>
      {!(error instanceof APIError && [401, 403, 404, 410].includes(error.status)) && (
        <button className={styles.button} type="button" onClick={retry}>
          Retry
        </button>
      )}
    </section>
  );
}
