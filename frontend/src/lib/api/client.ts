/**
 * OrvixPanel API client.
 *
 * Wraps the native `fetch` API with:
 *   - automatic Authorization header injection (Bearer JWT)
 *   - automatic token refresh on 401, with a single in-flight refresh
 *     promise so concurrent requests don't fan out
 *   - a typed `ApiError` with the structured `{ error, request_id }`
 *     payload the Go backend returns
 *   - JSON request/response bodies
 *
 * The client is split into *modules* (auth.ts, accounts.ts, ...) that
 * re-use this base `request` function. Each module owns a slice of the
 * API surface and is the only thing the rest of the app imports.
 */

import { useAuthStore } from "@/lib/auth/store";

export class ApiError extends Error {
  status: number;
  code: string;
  requestId: string | null;

  constructor(status: number, code: string, requestId: string | null, message?: string) {
    super(message ?? code);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

let inflightRefresh: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (inflightRefresh) return inflightRefresh;
  inflightRefresh = (async () => {
    const refreshToken = useAuthStore.getState().refreshToken;
    if (!refreshToken) return false;
    try {
      const res = await fetch("/auth/refresh", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      if (!res.ok) return false;
      const json = (await res.json()) as {
        access_token: string;
        refresh_token: string;
        expires_at: string;
      };
      useAuthStore.getState().setTokens({
        accessToken: json.access_token,
        refreshToken: json.refresh_token,
        expiresAt: json.expires_at,
      });
      return true;
    } catch {
      return false;
    } finally {
      inflightRefresh = null;
    }
  })();
  return inflightRefresh;
}

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  // Skip the access token (e.g. for the login call itself).
  noAuth?: boolean;
  // If true, return the raw Response (for streaming / non-JSON).
  raw?: boolean;
  // Override the default 15s timeout.
  timeoutMs?: number;
  // Per-request signal so callers can cancel via TanStack Query.
  signal?: AbortSignal;
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, noAuth, raw, timeoutMs = 15000, signal } = opts;

  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (!noAuth) {
    const token = useAuthStore.getState().accessToken;
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  // Compose external signal with a timeout signal.
  const ctrl = new AbortController();
  const onExternalAbort = () => ctrl.abort();
  if (signal) signal.addEventListener("abort", onExternalAbort);
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);

  let res: Response;
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: ctrl.signal,
      credentials: "same-origin",
    });
  } finally {
    clearTimeout(timer);
    if (signal) signal.removeEventListener("abort", onExternalAbort);
  }

  // 401 — try a single refresh then retry the original request.
  if (res.status === 401 && !noAuth) {
    const ok = await tryRefresh();
    if (ok) return request<T>(path, opts);
    // Refresh failed — drop tokens, the route guard will redirect to /login.
    useAuthStore.getState().clear();
    throw new ApiError(401, "unauthorized", res.headers.get("X-Request-Id"));
  }

  if (raw) return res as unknown as T;

  // 204 / 205 — no body.
  if (res.status === 204 || res.status === 205) return undefined as T;

  let json: unknown = null;
  const text = await res.text();
  if (text) {
    try {
      json = JSON.parse(text);
    } catch {
      // Not JSON — surface as a generic error.
      if (!res.ok) {
        throw new ApiError(res.status, "non_json_response", res.headers.get("X-Request-Id"), text.slice(0, 200));
      }
      return text as unknown as T;
    }
  }

  if (!res.ok) {
    const body = (json ?? {}) as { error?: string; request_id?: string };
    throw new ApiError(
      res.status,
      body.error ?? `http_${res.status}`,
      body.request_id ?? res.headers.get("X-Request-Id"),
    );
  }

  return json as T;
}
