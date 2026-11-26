import { useEffect, useState, type ChangeEvent, type MouseEvent } from "react";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { type DiffFile, type DiffLine, type DiscussionComment } from "@/api";
import { type Comment, type CommentRequest } from "@/api/generated/models";
import { diffPaths } from "@/routing/paths";

const reviewedFiles = new Map<string, Set<string>>();

type DiffDisplayRow = {
  line: DiffLine;
  old?: DiffLine;
  next?: DiffLine;
  comments: Comment[];
};

export type DiffProps = {
  files: DiffFile[];
  owner: string;
  repo: string;
  section: keyof typeof diffPaths;
  target: string;
  head: string;
  canComment?: boolean;
  onComment?: (anchor: CommentAnchor) => void;
  comments?: DiscussionComment[];
};

export type CommentAnchor = {
  path: string;
  line: number;
  side: NonNullable<CommentRequest["side"]>;
  head_sha: string;
};

export function useLogic({
  files,
  owner,
  repo,
  section,
  target,
  head,
  onComment,
  comments = [],
}: Pick<
  DiffProps,
  "files" | "owner" | "repo" | "section" | "target" | "head" | "onComment" | "comments"
>) {
  const search = useSearch({
    strict: false,
    select: ({ file, layout, tab }) => ({ file, layout, tab }),
  });
  const navigate = useNavigate();
  const storageKey = `${owner}/${repo}/${section}/${target}/${head}`;

  const [viewed, setViewed] = useState(() => new Set(reviewedFiles.get(storageKey)));
  const [largeFiles, setLargeFiles] = useState(new Set<string>());
  const [wide, setWide] = useState(() => matchMedia("(width > 800px)").matches);

  useEffect(() => {
    const media = matchMedia("(width > 800px)");
    function changed() {
      setWide(media.matches);
    }
    media.addEventListener("change", changed);
    return () => media.removeEventListener("change", changed);
  }, []);

  const split = search.layout === "split" && wide;
  const selected = files.find((file) => file.path === search.file);
  const visible = selected ? [selected] : files;
  const displayedFiles = visible.map((file) => {
    const rows: DiffDisplayRow[] = [];
    const lines = file.lines || [];
    for (let index = 0; index < lines.length; index++) {
      const line = lines[index];
      if (split && line.kind === "delete") {
        const deleted: DiffLine[] = [];
        const added: DiffLine[] = [];
        while (lines[index]?.kind === "delete") {
          deleted.push(lines[index++]);
        }
        while (lines[index]?.kind === "add") {
          added.push(lines[index++]);
        }
        index--;
        for (let position = 0; position < Math.max(deleted.length, added.length); position++) {
          const old = deleted[position];
          const next = added[position];
          rows.push({
            line: old || next,
            old,
            next,
            comments: [...(old?.comments || []), ...(next?.comments || [])],
          });
        }
      } else {
        rows.push({
          line,
          old: split && line.kind === "add" ? undefined : line,
          next: split && line.kind === "delete" ? undefined : line,
          comments: line.comments || [],
        });
      }
    }
    return { file, rows };
  });
  const commentHTML = new Map(comments.map((comment) => [comment.id, comment.html]));

  function layoutChanged(event: ChangeEvent<HTMLSelectElement>) {
    void navigate({
      to: diffPaths[section],
      params: { owner, repo, number: Number(target), sha: target, _splat: target },
      search: { ...search, layout: event.target.value },
      replace: true,
      resetScroll: false,
    });
  }

  function viewedChanged(event: ChangeEvent<HTMLInputElement>) {
    const next = new Set(viewed);
    if (event.target.checked) {
      next.add(event.target.value);
    } else {
      next.delete(event.target.value);
    }
    reviewedFiles.set(storageKey, next);
    setViewed(next);
  }

  function showLarge(event: MouseEvent<HTMLButtonElement>) {
    setLargeFiles(new Set([...largeFiles, event.currentTarget.dataset.path || ""]));
  }

  function commentOnLine(event: MouseEvent<HTMLButtonElement>) {
    const { path = "", line = "0", side = "new" } = event.currentTarget.dataset;
    if (side !== "new" && side !== "old") {
      return;
    }
    onComment?.({ path, line: Number(line), side, head_sha: head });
  }

  const selectedIndex = selected ? files.indexOf(selected) : -1;
  const nextFile = files[selectedIndex + 1];

  return {
    search,
    viewed,
    largeFiles,
    wide,
    split,
    selected,
    displayedFiles,
    commentHTML,
    layoutChanged,
    viewedChanged,
    showLarge,
    commentOnLine,
    nextFile,
  };
}
