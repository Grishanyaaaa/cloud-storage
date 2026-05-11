import { Link } from "@tanstack/react-router";
import { ChevronRight, Home } from "lucide-react";
import type { TreeNodeResponse } from "@/api/types";

interface Crumb {
  id: string;
  name: string;
  isRoot: boolean;
}

/**
 * Walks the cached tree to derive ancestors of `currentId`. Falls back to
 * a single 'Главная' crumb if the tree isn't loaded yet.
 */
export function Breadcrumbs({
  tree,
  currentId,
}: {
  tree: TreeNodeResponse | undefined;
  currentId: string;
}) {
  const crumbs = tree ? findAncestors(tree, currentId) : [];

  return (
    <nav aria-label="Хлебные крошки" className="flex items-center gap-1 text-sm">
      {crumbs.length === 0 ? (
        <span className="flex items-center gap-1 text-fg-2">
          <Home className="h-4 w-4" />
          Главная
        </span>
      ) : (
        crumbs.map((c, i) => {
          const isLast = i === crumbs.length - 1;
          return (
            <span key={c.id} className="flex items-center gap-1">
              {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-fg-3" />}
              {isLast ? (
                <span className="font-medium text-fg-1 truncate max-w-[280px]">
                  {c.isRoot ? "Главная" : c.name}
                </span>
              ) : (
                <Link
                  to="/files/$folderId"
                  params={{ folderId: c.id }}
                  className="text-fg-2 hover:text-fg-1 hover:underline truncate max-w-[200px]"
                >
                  {c.isRoot ? (
                    <span className="flex items-center gap-1">
                      <Home className="h-4 w-4" />
                      Главная
                    </span>
                  ) : (
                    c.name
                  )}
                </Link>
              )}
            </span>
          );
        })
      )}
    </nav>
  );
}

function findAncestors(root: TreeNodeResponse, targetId: string): Crumb[] {
  const path: Crumb[] = [];

  function walk(node: TreeNodeResponse, isRoot: boolean): boolean {
    path.push({ id: node.id, name: node.name, isRoot });
    if (node.id === targetId) return true;
    for (const child of node.children ?? []) {
      if (walk(child, false)) return true;
    }
    path.pop();
    return false;
  }

  walk(root, true);
  return path;
}
