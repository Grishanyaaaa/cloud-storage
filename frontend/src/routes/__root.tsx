import { createRootRoute, Outlet } from "@tanstack/react-router";
import { ErrorBoundary } from "@/components/common/ErrorBoundary";

export const Route = createRootRoute({
  component: RootLayout,
  notFoundComponent: NotFound,
});

function RootLayout() {
  return (
    <ErrorBoundary>
      <div className="min-h-screen bg-bg-0 text-fg-1">
        <Outlet />
      </div>
    </ErrorBoundary>
  );
}

function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-2">
      <div className="text-2xl font-semibold">404</div>
      <div className="text-fg-2 text-sm">Страница не найдена</div>
    </div>
  );
}
