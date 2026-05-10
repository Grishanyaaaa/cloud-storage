import { create } from "zustand";
import { authBus } from "@/api/client";
import { decodeJwtEmail, decodeJwtSubject, tokens } from "@/lib/tokens";

interface AuthUser {
  id: string;
  email: string;
}

interface AuthState {
  user: AuthUser | null;
  hasTokens: boolean;
  setSession(access: string, refresh: string): void;
  refreshFromStorage(): void;
  logout(): void;
}

function deriveUser(access: string | null): AuthUser | null {
  if (!access) return null;
  const id = decodeJwtSubject(access);
  const email = decodeJwtEmail(access);
  if (!id) return null;
  return { id, email: email ?? "" };
}

export const useAuthStore = create<AuthState>((set) => ({
  user: deriveUser(tokens.getAccess()),
  hasTokens: Boolean(tokens.getAccess() && tokens.getRefresh()),
  setSession(access, refresh) {
    tokens.setPair(access, refresh);
    set({ user: deriveUser(access), hasTokens: true });
  },
  refreshFromStorage() {
    const access = tokens.getAccess();
    set({ user: deriveUser(access), hasTokens: Boolean(access && tokens.getRefresh()) });
  },
  logout() {
    tokens.clear();
    set({ user: null, hasTokens: false });
  },
}));

/* ------------------------------------------------------------------ */
/* Cross-tab + cross-module sync                                      */
/* ------------------------------------------------------------------ */

if (typeof window !== "undefined") {
  // Token refreshed elsewhere (other tab) → recompute user.
  window.addEventListener("storage", (e) => {
    if (e.key === "cs:access_token" || e.key === "cs:refresh_token") {
      useAuthStore.getState().refreshFromStorage();
    }
  });

  // The HTTP client's refresh path emits 'logout' on hard 401 / 'refreshed'
  // after rotating the pair. Both should re-sync the store.
  authBus.on((event) => {
    if (event === "logout") {
      useAuthStore.getState().logout();
    } else if (event === "refreshed") {
      useAuthStore.getState().refreshFromStorage();
    }
  });
}
