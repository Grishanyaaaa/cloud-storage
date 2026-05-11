import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { logout as logoutApi } from "@/api/auth";
import { tokens } from "@/lib/tokens";
import { useAuthStore } from "@/store/auth.store";

export function useLogout() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const logoutLocal = useAuthStore((s) => s.logout);

  return useMutation({
    mutationFn: async () => {
      const refresh = tokens.getRefresh();
      if (refresh) {
        // best-effort — even if the server fails we still clear local state
        await logoutApi({ refresh_token: refresh }).catch(() => undefined);
      }
    },
    onSettled: async () => {
      logoutLocal();
      // Important: drop all cached server data, otherwise next user sees
      // the previous user's tree.
      queryClient.clear();
      toast("Вы вышли");
      await navigate({ to: "/login", replace: true });
    },
  });
}
