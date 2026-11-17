import "./styles.css";
import "@/styles/workspace.css";
import { RichText } from "@/components/markdown/RichText";
import { useLogic, type EditorProps } from "./logic";

export function MarkdownEditor({
  id,
  label,
  value,
  onChange,
  required = false,
  placeholder = "Write with Markdown…",
}: EditorProps) {
  const { preview, change, edit, showPreview } = useLogic({ onChange });
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
    </div>
  );
}
