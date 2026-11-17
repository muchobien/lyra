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
