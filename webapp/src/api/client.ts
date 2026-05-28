import { getStoredToken, clearStoredToken } from "@/stores/auth";
import type { ApiErrorBody } from "./types";

export class ApiError extends Error {
  status: number;
  body?: ApiErrorBody;

  constructor(status: number, message: string, body?: ApiErrorBody) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

export interface ApiRequestOptions {
  method?: string;
  body?: unknown;
  signal?: AbortSignal;
  // If true, the response is returned as raw text instead of parsed JSON.
  raw?: boolean;
  // If false, skip Authorization header injection (e.g. login probe).
  auth?: boolean;
}

const API_BASE = "/api";

let onUnauthorized: (() => void) | null = null;

export function setUnauthorizedHandler(handler: (() => void) | null) {
  onUnauthorized = handler;
}

async function readErrorBody(response: Response): Promise<ApiErrorBody | undefined> {
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    return undefined;
  }
  try {
    return (await response.json()) as ApiErrorBody;
  } catch {
    return undefined;
  }
}

export async function apiRequest<T = unknown>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  const { method = "GET", body, signal, raw = false, auth = true } = options;
  const isFormData =
    typeof FormData !== "undefined" && body instanceof FormData;
  const headers: Record<string, string> = {};
  if (body !== undefined && !isFormData) {
    headers["Content-Type"] = "application/json";
  }
  if (auth) {
    const token = getStoredToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body:
      body === undefined
        ? undefined
        : isFormData
          ? body
          : JSON.stringify(body),
    signal,
  });

  if (response.status === 401) {
    clearStoredToken();
    if (onUnauthorized) {
      onUnauthorized();
    }
    const errorBody = await readErrorBody(response);
    throw new ApiError(401, errorBody?.error ?? "unauthorized", errorBody);
  }

  if (!response.ok) {
    const errorBody = await readErrorBody(response);
    throw new ApiError(
      response.status,
      errorBody?.error ?? response.statusText,
      errorBody,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  if (raw) {
    return (await response.text()) as unknown as T;
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await response.json()) as T;
  }
  return (await response.text()) as unknown as T;
}

// Build a subscription URL with the token embedded as a query string.
// Pass token explicitly because the subscription endpoint authenticates via query param.
export function buildSubscriptionURL(id: string, token: string): string {
  const url = new URL(window.location.href);
  url.pathname = `${API_BASE}/subscriptions/${encodeURIComponent(id)}.yaml`;
  url.search = `token=${encodeURIComponent(token)}`;
  url.hash = "";
  return url.toString();
}
