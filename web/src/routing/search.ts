import { type SearchMiddleware } from "@tanstack/react-router";

type SearchValues<Key extends string> = Partial<Record<Key, string>>;

export const stripEmptySearch: SearchMiddleware<Record<string, unknown>> = ({ search, next }) =>
  Object.fromEntries(Object.entries(next(search)).filter(([, value]) => value !== ""));

export function searchValues<const Key extends string>(
  search: Record<string, unknown>,
  keys: Key[],
) {
  const parsed: SearchValues<Key> = {};
  for (const key of keys) {
    if (typeof search[key] === "string") {
      parsed[key] = search[key];
    }
  }
  return parsed;
}

export function searchLimit(value: string | undefined, minimum: number, maximum = 1000) {
  return Math.max(minimum, Math.min(maximum, Math.trunc(Number(value) || minimum)));
}

export function itemListParams(
  search: SearchValues<"state" | "q" | "label" | "assignee" | "limit">,
) {
  return {
    state: search.state || "open",
    limit: searchLimit(search.limit, 50),
    q: search.q || undefined,
    label: search.label ? [search.label] : undefined,
    assignee: search.assignee ? [search.assignee] : undefined,
  };
}
