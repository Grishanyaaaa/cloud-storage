/** Generates a short opaque id. Uses crypto.randomUUID when available. */
export function genUploadId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  // Fallback for environments without crypto.randomUUID (very old browsers).
  return `up_${Math.random().toString(36).slice(2, 10)}_${Date.now().toString(36)}`;
}
