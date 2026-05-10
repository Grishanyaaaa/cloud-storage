import type { PropsWithChildren } from "react";

interface Props {
  title: string;
  subtitle?: string;
}

export function AuthShell({ title, subtitle, children }: PropsWithChildren<Props>) {
  return (
    <div className="min-h-screen bg-bg-0 flex items-center justify-center p-6">
      <div className="w-full max-w-sm">
        <div className="flex flex-col items-center mb-8 select-none">
          <Logomark />
          <div className="mt-3 text-fg-1 text-lg font-semibold">cloud-storage</div>
        </div>
        <div className="rounded-xl border border-border-1 bg-bg-1 p-6 shadow-sm">
          <div className="mb-6">
            <h1 className="text-xl font-semibold">{title}</h1>
            {subtitle && <p className="text-fg-2 text-sm mt-1">{subtitle}</p>}
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}

function Logomark() {
  return (
    <div
      aria-hidden
      className="h-12 w-12 rounded-xl bg-bg-2 border border-border-1 flex items-center justify-center"
    >
      <svg width="22" height="22" viewBox="0 0 32 32" fill="none">
        <path
          d="M9 11.5l4.2-2.8 4.2 2.8M22.6 11.5l-4.2-2.8M9 11.5l4.2 2.8M22.6 11.5l-4.2 2.8M13.2 18.5l4.2-2.8M13.2 18.5L9 21.3M13.2 18.5l4.2 2.8M22.6 21.3l-4.2-2.8"
          stroke="var(--accent-1)"
          strokeWidth="1.7"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </div>
  );
}
