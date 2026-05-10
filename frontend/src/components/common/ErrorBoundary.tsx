import { Component, type PropsWithChildren, type ReactNode } from "react";

type State = { error: Error | null };

export class ErrorBoundary extends Component<PropsWithChildren, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  override componentDidCatch(error: Error, info: { componentStack: string | null }): void {
    // eslint-disable-next-line no-console
    console.error("ErrorBoundary caught:", error, info.componentStack);
  }

  override render(): ReactNode {
    if (this.state.error) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-3 p-6">
          <div className="text-2xl font-semibold">Что-то пошло не так</div>
          <div className="text-fg-2 text-sm">{this.state.error.message}</div>
          <button
            type="button"
            className="mt-2 rounded-md bg-accent-1 px-3 py-2 text-sm font-medium text-white hover:bg-accent-1h"
            onClick={() => window.location.reload()}
          >
            Перезагрузить
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
