import { Link, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/store/auth.store";
import { type LoginInput, loginSchema } from "./auth.schema";
import { useLogin } from "./useLogin";
import { AuthShell } from "./AuthShell";

export function LoginPage() {
  const hasTokens = useAuthStore((s) => s.hasTokens);
  const navigate = useNavigate();

  // If already signed-in, redirect away.
  useEffect(() => {
    if (hasTokens) {
      const next = new URLSearchParams(window.location.search).get("next") ?? "/files";
      void navigate({ to: next, replace: true });
    }
  }, [hasTokens, navigate]);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginInput>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useLogin();

  return (
    <AuthShell title="Вход" subtitle="Войдите в свой аккаунт cloud-storage">
      <form
        onSubmit={handleSubmit((v) => mutation.mutate(v))}
        className="flex flex-col gap-4"
        noValidate
      >
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="login-email">Email</Label>
          <Input
            id="login-email"
            type="email"
            autoComplete="email"
            aria-invalid={Boolean(errors.email)}
            aria-describedby={errors.email ? "login-email-err" : undefined}
            {...register("email")}
          />
          {errors.email && (
            <span id="login-email-err" className="text-[12px] text-danger">
              {errors.email.message}
            </span>
          )}
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="login-password">Пароль</Label>
          <Input
            id="login-password"
            type="password"
            autoComplete="current-password"
            aria-invalid={Boolean(errors.password)}
            aria-describedby={errors.password ? "login-password-err" : undefined}
            {...register("password")}
          />
          {errors.password && (
            <span id="login-password-err" className="text-[12px] text-danger">
              {errors.password.message}
            </span>
          )}
        </div>
        <Button type="submit" disabled={mutation.isPending} className="w-full mt-2">
          {mutation.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Вход…
            </>
          ) : (
            "Войти"
          )}
        </Button>
        <div className="text-center text-[13px] text-fg-2">
          Нет аккаунта?{" "}
          <Link to="/register" className="text-accent-1 hover:underline">
            Зарегистрироваться
          </Link>
        </div>
      </form>
    </AuthShell>
  );
}
