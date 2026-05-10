/**
 * Token storage. Per blueprint §6.1 we keep access+refresh in localStorage
 * (option `b`) for MVP. Switching to httpOnly cookies later means rewriting
 * this module — nothing else.
 */

const ACCESS_KEY = "cs:access_token";
const REFRESH_KEY = "cs:refresh_token";
const USER_KEY = "cs:user";

export const tokens = {
  getAccess(): string | null {
    return localStorage.getItem(ACCESS_KEY);
  },
  getRefresh(): string | null {
    return localStorage.getItem(REFRESH_KEY);
  },
  setPair(access: string, refresh: string): void {
    localStorage.setItem(ACCESS_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  },
  setUser(payload: string): void {
    localStorage.setItem(USER_KEY, payload);
  },
  getUser(): string | null {
    return localStorage.getItem(USER_KEY);
  },
  clear(): void {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(USER_KEY);
  },
};

/**
 * Extract `email` claim from a JWT without verifying the signature. Verification
 * is the gateway's job; we only use this for displaying the username in the
 * topbar — never for authorisation.
 */
export function decodeJwtEmail(token: string): string | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = parts[1];
    if (!payload) return null;
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    const decoded = JSON.parse(json) as { email?: string; sub?: string };
    return decoded.email ?? decoded.sub ?? null;
  } catch {
    return null;
  }
}

export function decodeJwtSubject(token: string): string | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = parts[1];
    if (!payload) return null;
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    const decoded = JSON.parse(json) as { sub?: string };
    return decoded.sub ?? null;
  } catch {
    return null;
  }
}
