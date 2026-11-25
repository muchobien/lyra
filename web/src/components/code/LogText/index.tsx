import { type CSSProperties } from "react";
import "./styles.css";
import { useLogic, type LogLine, type LogSpan, type LogTextProps } from "./logic";

type SpansProps = { spans: LogSpan[] };

type LineProps = { line: LogLine };

function Spans({ spans }: SpansProps) {
  return spans.map((span, index) => {
    const { bold, dim, italic, underline, color, background } = span.style;
    const style: CSSProperties = {};
    if (typeof color === "string") {
      style.color = color;
    }
    if (typeof background === "string") {
      style.background = background;
    }
    return (
      <span
        key={index}
        style={style}
        data-bold={bold || undefined}
        data-dim={dim || undefined}
        data-italic={italic || undefined}
        data-underline={underline || undefined}
        data-color={typeof color === "number" ? color : undefined}
        data-background={typeof background === "number" ? background : undefined}
      >
        {span.text}
      </span>
    );
  });
}

function Line({ line }: LineProps) {
  return (
    <div className="lyra-log-text-line" data-kind={line.kind} data-level={line.level}>
      <span className="lyra-log-text-number" aria-hidden="true">
        {line.number}
      </span>
      <span className="lyra-log-text-content">
        {line.level && <b>{line.level[0].toUpperCase() + line.level.slice(1)}: </b>}
        {line.title && `${line.title} · `}
        <Spans spans={line.spans} />
        {line.location && <span className="lyra-log-text-location">{line.location}</span>}
      </span>
    </div>
  );
}

export function LogText({ lines, start, end, query, empty, live }: LogTextProps) {
  const { blocks } = useLogic({ lines, start, end, query });

  if (!blocks.length) {
    return <pre className="lyra-actions-logText">{empty}</pre>;
  }
  return (
    <div className="lyra-log-text">
      {blocks.map((block) =>
        block.group ? (
          <details
            key={block.group.number}
            className="lyra-log-text-group"
            open={block.group.failed || (live && !block.group.ended) || undefined}
          >
            <summary className="lyra-log-text-line">
              <span className="lyra-log-text-number" aria-hidden="true">
                {block.group.number}
              </span>
              <span className="lyra-log-text-content">
                <Spans spans={block.group.title} />
              </span>
            </summary>
            {block.group.lines.map((line) => (
              <Line key={line.number} line={line} />
            ))}
          </details>
        ) : (
          <Line key={block.line.number} line={block.line} />
        ),
      )}
    </div>
  );
}
