import { useRouter, type ErrorComponentProps } from "@tanstack/react-router";
import { ErrorNotice } from "@/components/controls/ErrorNotice";

export function RouteError({ error, reset }: ErrorComponentProps) {
  const router = useRouter();

  async function retry() {
    await router.invalidate();
    reset();
  }

  return (
    <ErrorNotice error={error instanceof Error ? error : new Error(String(error))} retry={retry} />
  );
}
