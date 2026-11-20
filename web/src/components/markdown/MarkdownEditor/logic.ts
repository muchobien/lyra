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
  function acceptProofread() {
    onChange(proofreading.output);
    proofreading.reset();
  }

  return { preview, proofreading, change, edit, showPreview, proofread, acceptProofread };
}
