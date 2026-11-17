type SearchValues<Key extends string> = Partial<Record<Key, string>>;

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
