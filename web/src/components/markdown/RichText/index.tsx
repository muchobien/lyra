import "./styles.css";
import { Markdown } from "@/components/markdown/Markdown";
import { useLogic, type RichTextProps } from "./logic";

export function RichText({ source }: RichTextProps) {
  const { rendered } = useLogic({ source });
  if (!source) {
    return null;
  }
  if (rendered.data) {
    return <Markdown html={rendered.data.html} />;
  }
  return <div className="lyra-workspace-plainText">{source}</div>;
}
