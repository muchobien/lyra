import { useRenderMarkdown } from "@/api/generated/lyra";

export type RichTextProps = { source: string };

export function useLogic({ source }: Pick<RichTextProps, "source">) {
  const rendered = useRenderMarkdown(
    { body: source },
    { query: { staleTime: Infinity, enabled: !!source } },
  );

  return { rendered };
}
