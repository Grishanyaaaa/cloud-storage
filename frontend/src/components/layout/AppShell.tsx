import type { PropsWithChildren } from "react";
import { UploadPanel } from "@/features/files/upload/UploadPanel";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

export function AppShell({ children }: PropsWithChildren) {
  return (
    <div className="grid h-screen grid-cols-[var(--sidebar-w)_1fr] grid-rows-[var(--topbar-h)_1fr] bg-bg-0">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-2 focus:top-2 focus:z-50 focus:rounded-md focus:bg-bg-3 focus:px-3 focus:py-2 focus:text-sm focus:text-fg-1"
      >
        Перейти к содержимому
      </a>
      <div className="row-span-2 border-r border-border-1 bg-bg-1 overflow-hidden">
        <Sidebar />
      </div>
      <div className="border-b border-border-1 bg-bg-1">
        <Topbar />
      </div>
      <main id="main" className="overflow-auto bg-bg-0">
        {children}
      </main>
      <UploadPanel />
    </div>
  );
}
