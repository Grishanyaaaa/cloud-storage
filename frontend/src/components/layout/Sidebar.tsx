import { Link, useRouterState } from "@tanstack/react-router";
import { Files, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/cn";
import { useAIModalStore } from "@/store/ai.store";

export function Sidebar() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const openAi = useAIModalStore((s) => s.open);

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-[var(--topbar-h)] items-center gap-2 border-b border-border-1 px-4 select-none">
        <div className="h-7 w-7 rounded-md bg-bg-3 flex items-center justify-center">
          <svg width="16" height="16" viewBox="0 0 32 32" fill="none">
            <path
              d="M9 11.5l4.2-2.8 4.2 2.8M22.6 11.5l-4.2-2.8M9 11.5l4.2 2.8M22.6 11.5l-4.2 2.8M13.2 18.5l4.2-2.8M13.2 18.5L9 21.3M13.2 18.5l4.2 2.8M22.6 21.3l-4.2-2.8"
              stroke="var(--accent-1)"
              strokeWidth="1.7"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>
        <span className="font-semibold tracking-tight">cloud-storage</span>
      </div>

      <nav className="flex-1 px-2 py-3 space-y-1 text-[13px]">
        <SidebarLink
          to="/files"
          label="Мои файлы"
          icon={<Files className="h-4 w-4" />}
          active={pathname.startsWith("/files")}
        />
      </nav>

      <div className="px-3 pb-3">
        <Button
          intent="secondary"
          size="md"
          className="w-full justify-start"
          onClick={openAi}
        >
          <Sparkles className="h-4 w-4" />
          ИИ-помощник
        </Button>
      </div>
    </div>
  );
}

interface SidebarLinkProps {
  to: string;
  label: string;
  icon: React.ReactNode;
  active: boolean;
}

function SidebarLink({ to, label, icon, active }: SidebarLinkProps) {
  return (
    <Link
      to={to}
      className={cn(
        "flex h-9 items-center gap-3 rounded-md px-3 transition-colors",
        active ? "bg-bg-3 text-fg-1" : "text-fg-2 hover:bg-bg-2 hover:text-fg-1",
      )}
    >
      <span className="text-fg-2">{icon}</span>
      {label}
    </Link>
  );
}
