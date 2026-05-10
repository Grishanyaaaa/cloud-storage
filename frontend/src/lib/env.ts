/**
 * Centralised env access. Falls back to `window.__APP_CONFIG__` if present (set
 * by k8s init-container at runtime); otherwise uses Vite-injected values.
 */
function read(key: keyof ImportMetaEnv, fallback: string): string {
  const runtime = window.__APP_CONFIG__ ?? {};
  if (key === "VITE_API_BASE_URL" && runtime.API_BASE_URL) return runtime.API_BASE_URL;
  if (key === "VITE_SHARE_BASE_URL" && runtime.SHARE_BASE_URL) return runtime.SHARE_BASE_URL;
  const v = import.meta.env[key];
  return typeof v === "string" && v.length > 0 ? v : fallback;
}

function parseInt10(v: string, fallback: number): number {
  const n = Number.parseInt(v, 10);
  return Number.isFinite(n) ? n : fallback;
}

export const env = {
  API_BASE_URL: read("VITE_API_BASE_URL", "http://localhost:8080").replace(/\/+$/, ""),
  APP_NAME: read("VITE_APP_NAME", "cloud-storage"),
  DEFAULT_TREE_DEPTH: parseInt10(read("VITE_DEFAULT_TREE_DEPTH", "10"), 10),
  TREE_MAX_NODES: parseInt10(read("VITE_TREE_MAX_NODES", "500"), 500),
  UPLOAD_PARALLELISM: parseInt10(read("VITE_UPLOAD_PARALLELISM", "3"), 3),
  UPLOAD_PROGRESS_THROTTLE_MS: parseInt10(
    read("VITE_UPLOAD_PROGRESS_THROTTLE_MS", "200"),
    200,
  ),
  SHARE_BASE_URL: read("VITE_SHARE_BASE_URL", window.location.origin).replace(/\/+$/, ""),
} as const;
