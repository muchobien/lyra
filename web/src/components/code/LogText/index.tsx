import { type CSSProperties } from "react";
import "./styles.css";
import { linkPattern, useLogic, type LogLine, type LogSpan, type LogTextProps } from "./logic";

type SpansProps = { spans: LogSpan[] };

type LineProps = { line: LogLine; targeted: boolean; at?: string };

type TimeProps = { at?: string };

const timeFormat = new Intl.DateTimeFormat(undefined, { timeStyle: "medium", hourCycle: "h23" });

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
        {span.text.split(linkPattern).map((part, partIndex) =>
          partIndex % 2 ? (
            <a key={partIndex} href={part} target="_blank" rel="noopener noreferrer">
              {part}
            </a>
          ) : (
            part
          ),
        )}
      </span>
    );
  });
}

function Time({ at }: TimeProps) {
  if (!at) {
    return null;
  }
  return (
    <time className="lyra-log-text-time" dateTime={at} title={at}>
      {timeFormat.format(new Date(at))}
    </time>
  );
}

function Line({ line, targeted, at }: LineProps) {
  return (
    <div
      className="lyra-log-text-line"
      data-target={targeted || undefined}
      data-kind={line.kind}
      data-level={line.level}
      data-line={line.number}
    >
      <a className="lyra-log-text-number" href={`#L${line.number}`}>
        {line.number}
      </a>
      <Time at={at} />
      <span className="lyra-log-text-content">
        {line.level && <b>{line.level[0].toUpperCase() + line.level.slice(1)}: </b>}
        {line.title && `${line.title} · `}
        <Spans spans={line.spans} />
        {line.location && <span className="lyra-log-text-location">{line.location}</span>}
      </span>
    </div>
  );
}

export function LogText({ lines, times, start, end, query, target, empty }: LogTextProps) {
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
            open={block.group.failed || undefined}
          >
            <summary
              className="lyra-log-text-line"
              data-target={target === block.group.number || undefined}
            >
              <a className="lyra-log-text-number" href={`#L${block.group.number}`}>
                {block.group.number}
              </a>
              <Time at={times?.[block.group.number - 1]} />
              <span className="lyra-log-text-content">
                <Spans spans={block.group.title} />
              </span>
            </summary>
            {block.group.lines.map((line) => (
              <Line
                key={line.number}
                line={line}
                targeted={target === line.number}
                at={times?.[line.number - 1]}
              />
            ))}
          </details>
        ) : (
          <Line
            key={block.line.number}
            line={block.line}
            targeted={target === block.line.number}
            at={times?.[block.line.number - 1]}
          />
        ),
      )}
    </div>
  );
}
