import "./styles.css";

type MarkdownProps = { html: string };

export function Markdown({ html }: MarkdownProps) {
  return <div className="lyra-workspace-markdown" dangerouslySetInnerHTML={{ __html: html }} />;
}
