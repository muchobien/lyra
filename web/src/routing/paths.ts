export const itemListPaths = {
  issues: "/$owner/$repo/issues",
  pulls: "/$owner/$repo/pulls",
} as const;

export const itemDetailPaths = {
  issues: "/$owner/$repo/issues/$number",
  pulls: "/$owner/$repo/pulls/$number",
} as const;

export const diffPaths = {
  ...itemDetailPaths,
  commit: "/$owner/$repo/commit/$sha",
} as const;
