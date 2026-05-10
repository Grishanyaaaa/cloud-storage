/**
 * RouterProvider stub. Wired up in the next commit (file-based routes).
 * Keeping bootstrap minimal so `npm run build` succeeds without a real
 * route tree.
 */
export function Router() {
  return (
    <div className="flex h-full items-center justify-center bg-bg-0 text-fg-2">
      <div className="text-center">
        <div className="text-fg-1 text-xl font-semibold mb-2">cloud-storage</div>
        <div className="text-fg-2 text-sm">Frontend bootstrapped. Routing wires up next.</div>
      </div>
    </div>
  );
}
