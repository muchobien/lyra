export class APIError<Body = unknown> extends Error {
  status: number;
  body?: Body;

  constructor(status: number, message: string, body?: Body) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

export type ErrorType<Body> = APIError<Body>;

export async function request(path: string, signal?: AbortSignal, options?: RequestInit) {
  const response = await fetch(path, { ...options, signal, credentials: "same-origin" });
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    let failure: unknown;
    if (response.headers.get("Content-Type")?.includes("application/json")) {
      failure = await response.json();
      if (
        failure &&
        typeof failure === "object" &&
        "error" in failure &&
        typeof failure.error === "string"
      ) {
        message = failure.error;
      }
    }
    throw new APIError(response.status, message, failure);
  }
  return response;
}

type FetchOptions = RequestInit & { responseType?: "blob" };

export async function orvalFetch<Value>(path: string, options: FetchOptions): Promise<Value> {
  const { responseType, ...init } = options;
  const response = await request(path, init.signal || undefined, init);
  if (response.status === 204 || response.status === 205 || options.method === "HEAD") {
    return undefined as Value;
  }
  if (responseType === "blob") {
    return response.blob() as Promise<Value>;
  }
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    const text = await response.text();
    const data: unknown = text ? JSON.parse(text) : undefined;
    return data as Value;
  }
  if (contentType.startsWith("text/")) {
    return response.text() as Promise<Value>;
  }
  return response.blob() as Promise<Value>;
}
