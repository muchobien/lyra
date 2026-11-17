import { useState, type ReactNode } from "react";

export type ResponsiveDetailsProps = { query: string; className: string; children: ReactNode };

export function useLogic({ query }: Pick<ResponsiveDetailsProps, "query">) {
  const [initiallyOpen] = useState(() => matchMedia(query).matches);

  return { initiallyOpen };
}
