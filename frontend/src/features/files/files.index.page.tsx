import { useNavigate } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { useEffect } from "react";
import { Loader2 } from "lucide-react";
import { ensureRoot } from "@/api/storage";
import { ApiError } from "@/api/types";
import { AuthGate } from "@/components/common/AuthGate";
import { AppShell } from "@/components/layout/AppShell";

/**
 * Index page that resolves the current user's root folder via
 * POST /storage/v1/me/root (idempotent), then redirects to /files/$rootId
 * so the URL is stable and copy-pasteable.
 */
export function FilesIndexPage() {
  return (
    <AuthGate>
      <FilesIndexInner />
    </AuthGate>
  );
}

function FilesIndexInner() {
  const navigate = useNavigate();
  const mutation = useMutation({
    mutationFn: () => ensureRoot(),
    onSuccess: async (root) => {
      await navigate({ to: "/files/$folderId", params: { folderId: root.id }, replace: true });
    },
  });

  useEffect(() => {
    mutation.mutate();
    // intentional: only run once on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <AppShell>
      <div className="flex h-full items-center justify-center">
        {mutation.isError ? (
          <div className="text-center">
            <div className="text-fg-1 font-semibold mb-1">Не удалось открыть хранилище</div>
            <div className="text-fg-2 text-sm">
              {mutation.error instanceof ApiError
                ? mutation.error.message
                : "Попробуйте обновить страницу."}
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-2 text-fg-2 text-sm">
            <Loader2 className="h-4 w-4 animate-spin" />
            Открываем хранилище…
          </div>
        )}
      </div>
    </AppShell>
  );
}
