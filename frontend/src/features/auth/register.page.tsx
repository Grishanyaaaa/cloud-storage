import { Link, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/store/auth.store";
import { type RegisterInput, registerSchema } from "./auth.schema";
import { useRegister } from "./useRegister";
import { AuthShell } from "./AuthShell";

export function RegisterPage() {
  const hasTokens = useAuthStore((s) => s.hasTokens);
  const navigate = useNavigate();

  useEffect(() => {
    if (hasTokens) void navigate({ to: "/files", replace: true });
  }, [hasTokens, navigate]);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterInput>({
    resolver: zodResolver(registerSchema),
    defaultValues: { email: "", password: "" },
  });

  const mutation = useRegister();

  return (
    <AuthShell title="Регистрация" subtitle="Создайте новый аккаунт">
      <form
        onSubmit={handleSubmit((v) => mutation.mutate(v))}
        className="flex flex-col gap-4"
        noValidate
      >
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="reg-email">Email</Label>
          <Input
            id="reg-email"
            type="email"
            autoComplete="email"
            aria-invalid={Boolean(errors.email)}
            aria-describedby={errors.email ? "reg-email-err" : undefined}
            {...register("email")}
          />
          {errors.email && (
            <span id="reg-email-err" className="text-[12px] text-danger">
              {errors.email.message}
            </span>
          )}
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="reg-password">Пароль</Label>
          <Input
            id="reg-password"
            type="password"
            autoComplete="new-password"
            aria-invalid={Boolean(errors.password)}
            aria-describedby={errors.password ? "reg-password-err" : undefined}
            {...register("password")}
          />
          <span className="text-[12px] text-fg-3">
            Минимум 8 символов. Бэкенд проверяет дополнительные требования к
            сложности.
          </span>
          {errors.password && (
            <span id="reg-password-err" className="text-[12px] text-danger">
              {errors.password.message}
            </span>
          )}
        </div>
        <Button type="submit" disabled={mutation.isPending} className="w-full mt-2">
          {mutation.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Создание аккаунта…
            </>
          ) : (
            "Зарегистрироваться"
          )}
        </Button>
        <div className="text-center text-[13px] text-fg-2">
          Уже есть аккаунт?{" "}
          <Link to="/login" className="text-accent-1 hover:underline">
            Войти
          </Link>
        </div>
      </form>
    </AuthShell>
  );
}
