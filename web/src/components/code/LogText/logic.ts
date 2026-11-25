import { useMemo } from "react";

export type LogTextProps = {
  lines: string[];
  start: number;
  end: number;
  query?: string;
  empty: string;
};

type LogStyle = {
  bold?: boolean;
  dim?: boolean;
  italic?: boolean;
  underline?: boolean;
  color?: number | string;
  background?: number | string;
};

export type LogSpan = { text: string; style: LogStyle };

export type LogLine = {
  number: number;
  kind: "text" | "command" | "annotation";
  spans: LogSpan[];
  level?: "warning" | "error" | "notice";
  title?: string;
  location?: string;
};

export type LogGroup = { number: number; title: LogSpan[]; lines: LogLine[]; failed: boolean };

type LogBlock = { line: LogLine; group?: undefined } | { group: LogGroup; line?: undefined };

type LogEntry =
  | { kind: "group"; text: string }
  | { kind: "endgroup" }
  | { kind: "hidden" }
  | { kind: "text" | "command"; text: string }
  | {
      kind: "annotation";
      level: "warning" | "error" | "notice";
      text: string;
      title?: string;
      location?: string;
    };

// oxlint-disable-next-line no-control-regex
const escapes = /\x1b(?:\[[0-9;?]*[A-Za-z]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[()][A-Za-z0-9])/g;
export const linkPattern = /(\bhttps?:\/\/[^\s"'<>`]*[^\s"'<>`.,;:!?)\]}])/;
const workflowCommand = /^(?:::([a-z-]+)(?: ([^:]*))?::(.*)|##\[([a-z]+)\](.*))$/;
const hiddenCommands = new Set([
  "add-matcher",
  "remove-matcher",
  "debug",
  "add-mask",
  "save-state",
  "set-output",
]);
const annotationLevels = new Set(["warning", "error", "notice"]);

export function stripAnsi(text: string) {
  return text.replace(escapes, "");
}

function unescapeCommand(text: string) {
  return text
    .replaceAll("%0D", "\r")
    .replaceAll("%0A", "\n")
    .replaceAll("%3A", ":")
    .replaceAll("%2C", ",")
    .replaceAll("%25", "%");
}

function xtermColor(value: number) {
  if (value < 16) {
    return value;
  }
  if (value >= 232) {
    const gray = 8 + (value - 232) * 10;
    return `rgb(${gray} ${gray} ${gray})`;
  }
  const cube = value - 16;
  const level = (part: number) => (part ? 55 + part * 40 : 0);
  return `rgb(${level(Math.floor(cube / 36))} ${level(Math.floor(cube / 6) % 6)} ${level(cube % 6)})`;
}

function applySgr(previous: LogStyle, sequence: string) {
  const codes = sequence
    .slice(2, -1)
    .split(";")
    .map((code) => Number(code || 0));
  let style = { ...previous };
  for (let index = 0; index < codes.length; index++) {
    const code = codes[index];
    const key = code === 38 ? "color" : "background";
    if (code === 0) {
      style = {};
    } else if (code === 1) {
      style.bold = true;
    } else if (code === 2) {
      style.dim = true;
    } else if (code === 3) {
      style.italic = true;
    } else if (code === 4) {
      style.underline = true;
    } else if (code === 22) {
      style.bold = false;
      style.dim = false;
    } else if (code === 23) {
      style.italic = false;
    } else if (code === 24) {
      style.underline = false;
    } else if (code >= 30 && code <= 37) {
      style.color = code - 30;
    } else if (code >= 90 && code <= 97) {
      style.color = code - 82;
    } else if (code === 39) {
      delete style.color;
    } else if (code >= 40 && code <= 47) {
      style.background = code - 40;
    } else if (code >= 100 && code <= 107) {
      style.background = code - 92;
    } else if (code === 49) {
      delete style.background;
    } else if ((code === 38 || code === 48) && codes[index + 1] === 5) {
      style[key] = xtermColor(codes[index + 2]);
      index += 2;
    } else if ((code === 38 || code === 48) && codes[index + 1] === 2) {
      style[key] = `rgb(${codes[index + 2]} ${codes[index + 3]} ${codes[index + 4]})`;
      index += 4;
    }
  }
  return style;
}

function ansiSpans(text: string) {
  const spans: LogSpan[] = [];
  let style: LogStyle = {};
  let last = 0;
  for (const match of text.matchAll(escapes)) {
    if (match.index > last) {
      spans.push({ text: text.slice(last, match.index), style });
    }
    last = match.index + match[0].length;
    if (match[0].startsWith("\x1b[") && match[0].endsWith("m")) {
      style = applySgr(style, match[0]);
    }
  }
  if (last < text.length) {
    spans.push({ text: text.slice(last), style });
  }
  return spans;
}

function classify(raw: string): LogEntry {
  const plain = stripAnsi(raw);
  const match = plain.match(workflowCommand);
  if (match) {
    const name = match[1] || match[4];
    const message = match[1] ? match[3] : match[5];
    if (name === "group") {
      return { kind: "group", text: message };
    }
    if (name === "endgroup") {
      return { kind: "endgroup" };
    }
    if (hiddenCommands.has(name)) {
      return { kind: "hidden" };
    }
    if (annotationLevels.has(name)) {
      const fields = new Map(
        (match[2] || "")
          .split(",")
          .filter(Boolean)
          .map((pair) => {
            const [key, ...value] = pair.split("=");
            return [key.trim(), unescapeCommand(value.join("="))];
          }),
      );
      const text = unescapeCommand(message);
      const file = fields.get("file");
      const location =
        file && [file, fields.get("line"), fields.get("col")].filter(Boolean).join(":");
      return {
        kind: "annotation",
        level: name as "warning" | "error" | "notice",
        text,
        title: fields.get("title"),
        location: location && !text.includes(location) ? location : undefined,
      };
    }
  }
  if (plain.startsWith("[command]")) {
    return { kind: "command", text: raw.slice(raw.indexOf("[command]") + "[command]".length) };
  }
  return { kind: "text", text: raw };
}

function logLine(number: number, entry: LogEntry): LogLine {
  if (entry.kind === "annotation") {
    return {
      number,
      kind: "annotation",
      spans: [{ text: entry.text, style: {} }],
      level: entry.level,
      title: entry.title,
      location: entry.location,
    };
  }
  if (entry.kind === "command" || entry.kind === "text") {
    return { number, kind: entry.kind, spans: ansiSpans(entry.text) };
  }
  return { number, kind: "text", spans: [] };
}

function visibleText(line: LogLine) {
  return [line.title, ...line.spans.map((span) => span.text), line.location].join(" ");
}

export function useLogic({ lines, start, end, query }: Omit<LogTextProps, "empty">) {
  const blocks = useMemo(() => {
    const result: LogBlock[] = [];
    const needle = query?.toLowerCase();
    let group: LogGroup | undefined;
    for (let index = start; index < Math.min(end, lines.length); index++) {
      const entry = classify(lines[index]);
      if (entry.kind === "hidden") {
        continue;
      }
      if (needle) {
        if (entry.kind === "endgroup") {
          continue;
        }
        const line =
          entry.kind === "group"
            ? { number: index + 1, kind: "text" as const, spans: ansiSpans(entry.text) }
            : logLine(index + 1, entry);
        if (visibleText(line).toLowerCase().includes(needle)) {
          result.push({ line });
        }
        continue;
      }
      if (entry.kind === "endgroup") {
        group = undefined;
        continue;
      }
      if (entry.kind === "group") {
        group = { number: index + 1, title: ansiSpans(entry.text), lines: [], failed: false };
        result.push({ group });
        continue;
      }
      const line = logLine(index + 1, entry);
      if (group) {
        group.lines.push(line);
        group.failed ||= line.level === "error";
      } else {
        result.push({ line });
      }
    }
    return result;
  }, [lines, start, end, query]);

  return { blocks };
}
