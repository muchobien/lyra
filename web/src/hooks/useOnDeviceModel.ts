import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";

type Start = (
  signal: AbortSignal,
  monitor: CreateMonitorCallback,
) => Promise<ReadableStream<string>>;

export const englishText: LanguageModelCreateCoreOptions = {
  expectedInputs: [{ type: "text", languages: ["en"] }],
  expectedOutputs: [{ type: "text", languages: ["en"] }],
};

export function useOnDeviceModel(name: string, availability: () => Promise<Availability>) {
  const support = useQuery({
    queryKey: ["on-device-model", name],
    queryFn: availability,
    staleTime: Infinity,
    retry: false,
  });
  const [output, setOutput] = useState("");
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [pending, setPending] = useState(false);
  const controller = useRef<AbortController | null>(null);

  useEffect(() => () => controller.current?.abort(), []);

  function monitor(target: CreateMonitor) {
    target.addEventListener("downloadprogress", (event) => setProgress(event.loaded));
  }

  async function run(start: Start) {
    controller.current?.abort();
    const current = new AbortController();
    controller.current = current;
    setOutput("");
    setError(null);
    setProgress(null);
    setPending(true);
    try {
      const stream = await start(current.signal, monitor);
      setProgress(null);
      let text = "";
      for await (const chunk of stream) {
        if (current.signal.aborted) {
          break;
        }
        text += chunk;
        setOutput(text);
      }
    } catch (caught) {
      if (!current.signal.aborted) {
        setError(caught instanceof Error ? caught : new Error(String(caught)));
      }
    } finally {
      current.abort();
      if (controller.current === current) {
        setPending(false);
        setProgress(null);
      }
    }
  }

  function stop() {
    controller.current?.abort();
  }

  function reset() {
    controller.current?.abort();
    setOutput("");
    setError(null);
  }

  return {
    available: !!support.data && support.data !== "unavailable",
    output,
    progress,
    error,
    pending,
    run,
    stop,
    reset,
  };
}
