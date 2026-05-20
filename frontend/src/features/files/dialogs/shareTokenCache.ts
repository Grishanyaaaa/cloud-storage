import { authBus } from "@/api/client";

/**
 * Per-tab cache of raw share tokens (`shareId -> token`).
 *
 * The storage-service stores only a SHA-256 hash of the share token and
 * returns the raw token exactly once — in the response of `POST /shares`.
 * We persist it in `sessionStorage` so the owner can still see/copy the
 * URL after a TanStack Query background refetch, a dialog re-open, or a
 * page reload within the same tab. The cache is cleared on logout so a
 * different user signing in on the same tab cannot read stale tokens.
 */

const STORAGE_KEY = "cs:share-tokens";

type TokenMap = Record<string, string>;

function loadFromStorage(): TokenMap {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      const out: TokenMap = {};
      for (const [k, v] of Object.entries(parsed as Record<string, unknown>)) {
        if (typeof v === "string") out[k] = v;
      }
      return out;
    }
  } catch {
    // ignore — fall through to empty map
  }
  return {};
}

function saveToStorage(map: TokenMap): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(map));
  } catch {
    // sessionStorage may be unavailable (SSR, privacy mode) or full — ignore.
  }
}

const cache: TokenMap = loadFromStorage();

export const shareTokenCache = {
  get(shareId: string): string | undefined {
    return cache[shareId];
  },
  set(shareId: string, token: string): void {
    cache[shareId] = token;
    saveToStorage(cache);
  },
  remove(shareId: string): void {
    if (shareId in cache) {
      delete cache[shareId];
      saveToStorage(cache);
    }
  },
  clear(): void {
    for (const k of Object.keys(cache)) delete cache[k];
    try {
      sessionStorage.removeItem(STORAGE_KEY);
    } catch {
      // ignore
    }
  },
};

authBus.on((event) => {
  if (event === "logout") shareTokenCache.clear();
});
