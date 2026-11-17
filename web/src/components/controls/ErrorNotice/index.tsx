import { APIError } from "@/api";
import "./styles.css";
import "@/styles/controls.css";

type NoticeProps = { error: Error; retry: () => void; title?: string };

export function ErrorNotice({ error, retry, title = "Couldn’t load this view" }: NoticeProps) {
  return (
    <section className="lyra-controls-notice" role="alert">
      <h2>{title}</h2>
      <p>{error.message}</p>
      {!(error instanceof APIError && [401, 403, 404, 410].includes(error.status)) && (
        <button className="lyra-controls-button" type="button" onClick={retry}>
          Retry
        </button>
      )}
    </section>
  );
}
