import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useEffect, type PropsWithChildren } from "react";
import { useAuthStore } from "@/store/auth.store";

/**
 * Renders children only when there's a valid token in storage. Otherwise
 * redirects to /login?next=<current-path>.
 *
 * The 401-refresh dance lives inside api/client.ts (mutex-protected). This
 * component is the cosmetic gate — it stops unauthorized rendering, keeps
 * the URL in sync, and adds a `next` param so the user lands back where
 * they were.
 */
export function AuthGate({ children }: PropsWithChildren) {
  const hasTokens = useAuthStore((s) => s.hasTokens);
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const navigate = useNavigate();

  useEffect(() => {
    if (!hasTokens) {
      const next = encodeURIComponent(pathname);
      // Use full URL with raw search string — bypasses search-param type
      // validation (login route has no schema).
      void navigate({ to: `/login?next=${next}`, replace: true });
    }
  }, [hasTokens, pathname, navigate]);

  if (!hasTokens) {
    return null;
  }
  return <>{children}</>;
}
