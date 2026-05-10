import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { login, register } from "@/api/auth";
import { ApiError } from "@/api/types";
import { useAuthStore } from "@/store/auth.store";
import type { RegisterInput } from "./auth.schema";

export function useRegister() {
  const setSession = useAuthStore((s) => s.setSession);
  const navigate = useNavigate();

  return useMutation({
    mutationFn: async (input: RegisterInput) => {
      await register(input);
      // Auto-login right after register so the user lands on /files.
      const pair = await login(input);
      return pair;
    },
    onSuccess: async (pair) => {
      setSession(pair.access_token, pair.refresh_token);
      toast.success("Аккаунт создан");
      await navigate({ to: "/files", replace: true });
    },
    onError: (err: unknown) => {
      const msg = err instanceof ApiError ? err.message : "Не удалось зарегистрироваться";
      toast.error(msg);
    },
  });
}
