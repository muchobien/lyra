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
  compare: "/$owner/$repo/compare/$",
} as const;

export function splitSource(target: string, branches: string[], tags: string[]) {
  const names = [
    ...branches,
    ...branches.map((name) => `refs/heads/${name}`),
    ...tags,
    ...tags.map((name) => `refs/tags/${name}`),
  ];
  let ref = target.split("/")[0];
  for (const name of names) {
    if (name.length > ref.length && (target === name || target.startsWith(`${name}/`))) {
      ref = name;
    }
  }
  return { ref, path: target.slice(ref.length + 1) };
}

export const numberParams = {
  parse: ({ number }: { number: string }) => {
    const parsed = Number(number);
    if (!/^\+?\d+$/.test(number) || !Number.isSafeInteger(parsed) || parsed < 1) {
      return false as const;
    }
    return { number: parsed };
  },
  stringify: ({ number }: { number: number }) => ({ number: String(number) }),
};
