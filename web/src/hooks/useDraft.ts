import { useSyncExternalStore } from "react";

const drafts = new Map<string, string>();
const listeners = new Set<() => void>();

export function useDraft(key: string, initial = "") {
  const value = useSyncExternalStore(
    function subscribe(listener) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    () => drafts.get(key) ?? initial,
  );
  function setValue(value: string) {
    drafts.set(key, value);
    listeners.forEach((listener) => listener());
  }
  function clear() {
    drafts.delete(key);
    listeners.forEach((listener) => listener());
  }
  return { value, setValue, clear };
}
