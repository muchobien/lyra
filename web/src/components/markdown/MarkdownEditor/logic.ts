import { useState, type ChangeEvent } from "react";
import { englishText, useOnDeviceModel } from "@/hooks/useOnDeviceModel";

export type EditorProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
  placeholder?: string;
};

export function useLogic({ value, onChange }: Pick<EditorProps, "value" | "onChange">) {
  const [preview, setPreview] = useState(false);
  const imageInput: LanguageModelCreateCoreOptions = {
    expectedInputs: [{ type: "text", languages: ["en"] }, { type: "image" }],
    expectedOutputs: [{ type: "text", languages: ["en"] }],
  };
  const describing = useOnDeviceModel("language-model-image", async () =>
    "LanguageModel" in self ? LanguageModel.availability(imageInput) : "unavailable",
  );
  const images = [...value.matchAll(/!\[([^\]]*)\]\(([^)\s]+)\)/g)].filter(([, alt, url]) => {
    const text = alt.trim();
    return (
      URL.canParse(url, location.href) &&
      new URL(url, location.href).origin === location.origin &&
      (!text || /^[\w.-]+\.\w{2,5}$/.test(text))
    );
  });
  const descriptions: { source: string; url: string; alt: string }[] = describing.output
    .split("\n")
    .filter(Boolean)
    .map((line) => JSON.parse(line));
  const proofreading = useOnDeviceModel("language-model", async () =>
    "LanguageModel" in self ? LanguageModel.availability(englishText) : "unavailable",
  );
  function change(event: ChangeEvent<HTMLTextAreaElement>) {
    onChange(event.target.value);
  }
  function edit() {
    setPreview(false);
  }
  function showPreview() {
    setPreview(true);
  }
  function proofread() {
    void proofreading.run(async (signal, monitor) => {
      const session = await LanguageModel.create({
        ...englishText,
        initialPrompts: [
          {
            role: "system",
            content:
              "You proofread Markdown written by a software developer. Fix spelling, grammar and punctuation, and reword sentences only where they are unclear. Keep the meaning, tone and Markdown structure, and keep links, code spans, code blocks, file paths and @mentions exactly as written. Output only the corrected Markdown.",
          },
        ],
        signal,
        monitor,
      });
      return session.promptStreaming(value, { signal });
    });
  }
  function describeImages() {
    void describing.run(async (signal, monitor) => {
      const session = await LanguageModel.create({
        ...imageInput,
        initialPrompts: [
          {
            role: "system",
            content:
              "You write alt text for images in software issues and pull requests, such as screenshots and diagrams. In one sentence of at most 125 characters, describe what the image shows that matters to the reader, and include short text that matters, such as an error message. Don't start with 'Image of'. Output only the alt text.",
          },
        ],
        signal,
        monitor,
      });
      return (async function* () {
        for (const [source, , url] of images) {
          const response = await fetch(url, { signal });
          const image = await response.blob();
          if (!response.ok || !image.type.startsWith("image/")) {
            continue;
          }
          const imageSession = await session.clone({ signal });
          const alt = await imageSession.prompt(
            [{ role: "user", content: [{ type: "image", value: image }] }],
            { signal },
          );
          imageSession.destroy();
          yield JSON.stringify({ source, url, alt: alt.replace(/[\]\n]+/g, " ").trim() }) + "\n";
        }
        yield "\n";
      })();
    });
  }
  function acceptDescriptions() {
    let next = value;
    for (const description of descriptions) {
      next = next.replace(description.source, `![${description.alt}](${description.url})`);
    }
    onChange(next);
    describing.reset();
  }
  function acceptProofread() {
    onChange(proofreading.output);
    proofreading.reset();
  }

  return {
    preview,
    proofreading,
    describing,
    images,
    descriptions,
    change,
    edit,
    showPreview,
    proofread,
    acceptProofread,
    describeImages,
    acceptDescriptions,
  };
}
