import { useState } from "react";
import { ChevronRight, Folder, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import type { NodeResponse, TreeNodeResponse } from "@/api/types";
import { cn } from "@/lib/cn";
import { useTree } from "../useFilesData";
import { useMoveNode } from "../useFileMutations";

interface Props {
  node: NodeResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MoveDialog({ node, open, onOpenChange }: Props) {
  const tree = useTree();
  const move = useMoveNode();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  // The target must not be the node itself, nor any of its descendants.
  const forbidden = collectIds(findInTree(tree.data, node.id));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>Переместить</DialogTitle>
          <DialogDescription>
            Выберите папку назначения для «{node.name}»
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-md border border-border-1 bg-bg-2">
          <ScrollArea className="max-h-80">
            {tree.isLoading ? (
              <div className="p-4 space-y-2">
                <Skeleton className="h-5 w-2/3" />
                <Skeleton className="h-5 w-1/2" />
                <Skeleton className="h-5 w-3/4" />
              </div>
            ) : tree.data ? (
              <FolderRow
                node={tree.data}
                level={0}
                selectedId={selectedId}
                onSelect={(id) => setSelectedId(id)}
                expanded={expanded}
                toggleExpand={(id) =>
                  setExpanded((prev) => {
                    const next = new Set(prev);
                    if (next.has(id)) next.delete(id);
                    else next.add(id);
                    return next;
                  })
                }
                forbidden={forbidden}
              />
            ) : (
              <div className="p-4 text-fg-2 text-sm">Не удалось загрузить дерево.</div>
            )}
          </ScrollArea>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" intent="secondary">
              Отмена
            </Button>
          </DialogClose>
          <Button
            type="button"
            disabled={!selectedId || selectedId === node.parent_id || move.isPending}
            onClick={() => {
              if (!selectedId) return;
              move.mutate(
                { id: node.id, new_parent_id: selectedId, oldParentId: node.parent_id ?? undefined },
                { onSuccess: () => onOpenChange(false) },
              );
            }}
          >
            {move.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Перемещение…
              </>
            ) : (
              "Переместить"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface FolderRowProps {
  node: TreeNodeResponse;
  level: number;
  selectedId: string | null;
  onSelect: (id: string) => void;
  expanded: Set<string>;
  toggleExpand: (id: string) => void;
  forbidden: Set<string>;
}

function FolderRow({
  node,
  level,
  selectedId,
  onSelect,
  expanded,
  toggleExpand,
  forbidden,
}: FolderRowProps) {
  if (node.kind !== "folder" || node.deleted_at) return null;
  const isOpen = level === 0 || expanded.has(node.id);
  const isSelected = selectedId === node.id;
  const isForbidden = forbidden.has(node.id);
  const childFolders = (node.children ?? []).filter(
    (c) => c.kind === "folder" && !c.deleted_at,
  );

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          if (!isForbidden) onSelect(node.id);
          if (childFolders.length > 0 && level > 0) toggleExpand(node.id);
        }}
        disabled={isForbidden}
        className={cn(
          "flex w-full items-center gap-2 px-3 py-1.5 text-sm transition-colors",
          isSelected && !isForbidden && "bg-accent-soft text-accent-1",
          !isSelected && !isForbidden && "hover:bg-bg-3 text-fg-1",
          isForbidden && "text-fg-3 cursor-not-allowed",
        )}
        style={{ paddingLeft: `${12 + level * 16}px` }}
      >
        {childFolders.length > 0 ? (
          <ChevronRight
            className={cn(
              "h-3.5 w-3.5 shrink-0 text-fg-3 transition-transform",
              isOpen && "rotate-90",
            )}
          />
        ) : (
          <span className="w-3.5 shrink-0" />
        )}
        <Folder className="h-4 w-4 shrink-0 text-accent-1" />
        <span className="truncate">{level === 0 ? "Главная" : node.name}</span>
      </button>
      {isOpen &&
        childFolders.map((child) => (
          <FolderRow
            key={child.id}
            node={child}
            level={level + 1}
            selectedId={selectedId}
            onSelect={onSelect}
            expanded={expanded}
            toggleExpand={toggleExpand}
            forbidden={forbidden}
          />
        ))}
    </div>
  );
}

function findInTree(
  root: TreeNodeResponse | undefined,
  id: string,
): TreeNodeResponse | null {
  if (!root) return null;
  if (root.id === id) return root;
  for (const child of root.children ?? []) {
    const found = findInTree(child, id);
    if (found) return found;
  }
  return null;
}

function collectIds(root: TreeNodeResponse | null): Set<string> {
  const set = new Set<string>();
  if (!root) return set;
  const walk = (n: TreeNodeResponse) => {
    set.add(n.id);
    for (const c of n.children ?? []) walk(c);
  };
  walk(root);
  return set;
}
