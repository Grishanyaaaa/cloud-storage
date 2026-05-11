import { env } from "@/lib/env";
import { mapServerError } from "@/lib/errors";
import { tokens } from "@/lib/tokens";
import { ApiError, NetworkError, type AuthTokenPair, type ServerEnvelope } from "./types";

/* ------------------------------------------------------------------ */
/* Refresh mutex                                                       */
/* ------------------------------------------------------------------ */

let inflightRefresh: Promise<void> | null = null;

async function refreshAccessToken(): Promise<void> {
  if (inflightRefresh) return inflightRefresh;
  inflightRefresh = (async () => {
    const refresh = tokens.getRefresh();
    if (!refresh) {
      throw new ApiError({
        status: 401,
        code: "UNAUTHORIZED",
        message: "Необходимо войти в систему.",
      });
    }
    const resp = await fetch(`${env.API_BASE_URL}/auth/v1/refresh`, {
      method: "POST",
      headers: { "content-type": "application/json", accept: "application/json" },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    if (!resp.ok) {
      tokens.clear();
      authBus.emit("logout");
      throw new ApiError({
        status: resp.status,
        code: "UNAUTHORIZED",
        message: "Сессия истекла. Войдите снова.",
      });
    }
    const body = (await resp.json()) as ServerEnvelope<unknown>;
    if (body.status !== "success") {
      tokens.clear();
      authBus.emit("logout");
      throw new ApiError({
        status: 401,
        code: "UNAUTHORIZED",
        message: "Сессия истекла. Войдите снова.",
      });
    }
    const pair = readTokenPair(body.data);
    if (!pair) {
      tokens.clear();
      authBus.emit("logout");
      throw new ApiError({
        status: 502,
        code: "SERVER_ERROR",
        message: "Сервер вернул некорректный ответ при обновлении токена.",
      });
    }
    tokens.setPair(pair.access_token, pair.refresh_token);
    authBus.emit("refreshed");
  })().finally(() => {
    inflightRefresh = null;
  });
  return inflightRefresh;
}

/* ------------------------------------------------------------------ */
/* TokenPair tolerant reader                                          */
/* ------------------------------------------------------------------ */

/**
 * auth-service's TokenPairResponse currently lacks json tags so it serialises
 * as PascalCase (`AccessToken`, `RefreshToken`). Tolerate both shapes.
 */
export function readTokenPair(data: unknown): AuthTokenPair | null {
  if (!data || typeof data !== "object") return null;
  const obj = data as Record<string, unknown>;
  const access =
    (typeof obj["access_token"] === "string" && obj["access_token"]) ||
    (typeof obj["AccessToken"] === "string" && obj["AccessToken"]) ||
    null;
  const refresh =
    (typeof obj["refresh_token"] === "string" && obj["refresh_token"]) ||
    (typeof obj["RefreshToken"] === "string" && obj["RefreshToken"]) ||
    null;
  if (!access || !refresh) return null;
  return { access_token: access, refresh_token: refresh };
}

export function readUserId(data: unknown): string | null {
  if (!data || typeof data !== "object") return null;
  const obj = data as Record<string, unknown>;
  return (
    (typeof obj["user_id"] === "string" && obj["user_id"]) ||
    (typeof obj["UserID"] === "string" && obj["UserID"]) ||
    (typeof obj["id"] === "string" && obj["id"]) ||
    null
  );
}

/* ------------------------------------------------------------------ */
/* Tiny event bus for auth state changes                              */
/* ------------------------------------------------------------------ */

type AuthEvent = "logout" | "refreshed";
type Listener = (e: AuthEvent) => void;
const listeners = new Set<Listener>();
export const authBus = {
  on(fn: Listener): () => void {
    listeners.add(fn);
    return () => listeners.delete(fn);
  },
  emit(e: AuthEvent): void {
    for (const fn of listeners) fn(e);
  },
};

/* ------------------------------------------------------------------ */
/* Public API                                                          */
/* ------------------------------------------------------------------ */

export interface ApiFetchOptions<TBody = unknown> {
  method?: "GET" | "POST" | "PATCH" | "PUT" | "DELETE";
  body?: TBody;
  query?: Record<string, string | number | boolean | undefined | null>;
  signal?: AbortSignal;
  /** Skip Authorization header (used by /auth endpoints and /share/{token}). */
  skipAuth?: boolean;
  /** Skip JSON body parsing — caller wants raw Response. */
  raw?: boolean;
}

function buildUrl(path: string, query?: ApiFetchOptions["query"]): string {
  const url = new URL(`${env.API_BASE_URL}${path.startsWith("/") ? path : `/${path}`}`);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v === undefined || v === null) continue;
      url.searchParams.set(k, String(v));
    }
  }
  return url.toString();
}

async function execute<T>(path: string, opts: ApiFetchOptions, retried: boolean): Promise<T> {
  const url = buildUrl(path, opts.query);
  const headers = new Headers();
  if (!opts.skipAuth) {
    const access = tokens.getAccess();
    if (access) headers.set("authorization", `Bearer ${access}`);
  }
  let body: BodyInit | undefined;
  if (opts.body !== undefined && opts.body !== null) {
    if (
      typeof opts.body === "string" ||
      opts.body instanceof FormData ||
      opts.body instanceof Blob ||
      opts.body instanceof ArrayBuffer
    ) {
      body = opts.body as BodyInit;
    } else {
      body = JSON.stringify(opts.body);
      headers.set("content-type", "application/json");
    }
  }
  headers.set("accept", "application/json");

  let response: Response;
  try {
    response = await fetch(url, {
      method: opts.method ?? (opts.body ? "POST" : "GET"),
      headers,
      body,
      signal: opts.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw err;
    }
    throw new NetworkError("Сеть недоступна. Проверьте подключение.", err);
  }

  // 401 with a Bearer token in flight → refresh once and retry once.
  if (response.status === 401 && !opts.skipAuth && !retried && tokens.getAccess()) {
    try {
      await refreshAccessToken();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        throw err;
      }
      throw err;
    }
    return execute<T>(path, opts, true);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  if (opts.raw) {
    return response as unknown as T;
  }

  let parsed: unknown;
  try {
    parsed = await response.json();
  } catch {
    parsed = null;
  }

  if (!response.ok) {
    throw mapServerError(parsed, response.status);
  }
  if (parsed && typeof parsed === "object" && "status" in parsed) {
    const env = parsed as ServerEnvelope<T>;
    if (env.status === "error") {
      throw mapServerError(env, response.status);
    }
    return env.data as T;
  }
  return parsed as T;
}

export async function apiFetch<T>(path: string, opts: ApiFetchOptions = {}): Promise<T> {
  return execute<T>(path, opts, false);
}

export async function apiPublicFetch<T>(
  path: string,
  opts: Omit<ApiFetchOptions, "skipAuth"> = {},
): Promise<T> {
  return execute<T>(path, { ...opts, skipAuth: true }, false);
}
