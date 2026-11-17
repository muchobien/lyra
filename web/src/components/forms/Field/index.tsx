import { type ReactNode } from "react";
import "./styles.css";

type FieldProps = { label: ReactNode; children: ReactNode };

export function Field({ label, children }: FieldProps) {
  return (
    <label className="lyra-workspace-field">
      <span>{label}</span>
      {children}
    </label>
  );
}
