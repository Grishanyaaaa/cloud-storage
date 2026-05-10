import { LogOut, User } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLogout } from "@/features/auth/useLogout";
import { useAuthStore } from "@/store/auth.store";

export function UserMenu() {
  const user = useAuthStore((s) => s.user);
  const logout = useLogout();

  const initials = user?.email ? user.email[0]?.toUpperCase() ?? "?" : "?";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="flex h-8 w-8 items-center justify-center rounded-full bg-bg-3 border border-border-1 text-fg-1 hover:bg-bg-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-1 focus-visible:ring-offset-2 focus-visible:ring-offset-bg-1"
        aria-label="Меню пользователя"
      >
        <span className="text-[12px] font-semibold">{initials}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuLabel className="flex flex-col">
          <span className="text-fg-1 truncate">{user?.email ?? "—"}</span>
          {user?.id && (
            <span className="text-[11px] text-fg-3 font-normal truncate">
              {user.id}
            </span>
          )}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled>
          <User className="h-4 w-4" />
          Профиль
        </DropdownMenuItem>
        <DropdownMenuItem
          intent="danger"
          onSelect={(e) => {
            e.preventDefault();
            logout.mutate();
          }}
        >
          <LogOut className="h-4 w-4" />
          Выйти
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
