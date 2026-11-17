import "./styles.css";

type FormErrorProps = { error: Error | null };

export function FormError({ error }: FormErrorProps) {
  return error ? (
    <p className="lyra-workspace-formError" role="alert">
      {error.message}
    </p>
  ) : null;
}
