import "./styles.css";
import "@/styles/workspace.css";
import { RichText } from "@/components/markdown/RichText";
import { OnDeviceOutput } from "@/components/ai/OnDeviceOutput";
import { useLogic, type EditorProps } from "./logic";

export function MarkdownEditor({
  id,
  label,
  value,
  onChange,
  required = false,
  placeholder = "Write with Markdown…",
}: EditorProps) {
  const { preview, proofreading, change, edit, showPreview, proofread, acceptProofread } = useLogic(
    {
      value,
      onChange,
    },
  );
  return (
    <div className="lyra-workspace-editor">
      <div className="lyra-workspace-editorTabs">
        <label htmlFor={id}>{label}</label>
        <button type="button" onClick={edit} aria-pressed={!preview}>
          Write
        </button>
        <button type="button" onClick={showPreview} aria-pressed={preview}>
          Preview
        </button>
        {proofreading.available && (
          <button
            type="button"
            onClick={proofread}
            disabled={!value.trim() || proofreading.pending}
          >
            Proofread
          </button>
        )}
      </div>
      <textarea
        id={id}
        name="body"
        value={value}
        onChange={change}
        placeholder={placeholder}
        required={required}
        maxLength={65536}
        hidden={preview}
      />
      {preview && (
        <div className="lyra-workspace-preview">
          {value ? (
            <RichText source={value} />
          ) : (
            <p className="lyra-workspace-muted">Nothing to preview.</p>
          )}
        </div>
      )}
      <OnDeviceOutput
        title="Proofread"
        output={proofreading.output}
        progress={proofreading.progress}
        error={proofreading.error}
        pending={proofreading.pending}
        onStop={proofreading.stop}
        onDiscard={proofreading.reset}
      >
        <button className="lyra-controls-button" type="button" onClick={acceptProofread}>
          Replace
        </button>
      </OnDeviceOutput>
    </div>
  );
}
