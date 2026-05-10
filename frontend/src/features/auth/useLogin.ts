import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { login } from "@/api/auth";
import { ApiError } from "@/api/types";
import { useAuthStore } from "@/store/auth.store";
import type { LoginInput } from "./auth.schema";

export function useLogin() {
  const setSession = useAuthStore((s) => s.setSession);
  const navigate = useNavigate();

  return useMutation({
    mutationFn: (input: LoginInput) => login(input),
    onSuccess: async (pair) => {
      setSession(pair.access_token, pair.refresh_token);
      toast.success("Добро пожаловать");
      const next = new URLSearchParams(window.location.search).get("next") ?? "/files";
      await navigate({ to: next, replace: true });
    },
    onError: (err: unknown) => {
      const msg = err instanceof ApiError ? err.message : "Не удалось войти";
      toast.error(msg);
    },
  });
}
