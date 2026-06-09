import { Link } from "@tanstack/react-router";
import { MoreHorizontal, Trash2, Pencil, Move, Share2, Download, RotateCcw, ArrowUp, ArrowDown } from "lucide-react";
import { useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import type { NodeResponse } from "@/api/types";
import { cn } from "@/lib/cn";
import { formatBytes, formatRelativeTime } from "@/lib/format";
import { iconForNode } from "@/lib/mime";
import type { SortField, SortDir } from "./useFileFilters";
import { RenameDialog } from "./dialogs/RenameDialog";
import { DeleteDialog } from "./dialogs/DeleteDialog";
import { MoveDialog } from "./dialogs/MoveDialog";
import { ShareDialog } from "./dialogs/ShareDialog";
import { useDownload } from "./useDownload";
import { useRestoreNode } from "./useFileMutations";

interface Props {
  items: NodeResponse[];
  isLoading: boolean;
  isError: boolean;
  emptyMessage?: string;
  sortField?: SortField;
  sortDir?: SortDir;
  onSort?: (field: SortField) => void;
}

export function FileTable({ items, isLoading, isError, emptyMessage, sortField, sortDir, onSort }: Props) {
  if (isLoading) return <FileTableSkeleton />;
  if (isError) {
    return (
      <div className="px-4 py-12 text-center text-fg-2">
        Не удалось загрузить содержимое.
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <div className="px-4 py-16 text-center text-fg-2">
        {emptyMessage ?? "Папка пуста. Перетащите файлы или нажмите «Загрузить»."}
      </div>
    );
  }

  return (
    <div className="border-t border-border-1">
      <div
        role="table"
        aria-label="Содержимое папки"
        className="text-sm"
      >
        <div
          role="row"
          className="grid grid-cols-[1fr_140px_180px_44px] items-center gap-3 px-4 h-9 text-[12px] uppercase tracking-wider text-fg-3 border-b border-border-1"
        >
          <SortableHeader field="name" label="Имя" sortField={sortField} sortDir={sortDir} onSort={onSort} />
          <SortableHeader field="size" label="Размер" sortField={sortField} sortDir={sortDir} onSort={onSort} />
          <SortableHeader field="updated_at" label="Изменён" sortField={sortField} sortDir={sortDir} onSort={onSort} />
          <div role="columnheader" />
        </div>
        {items.map((node) => (
          <FileRow key={node.id} node={node} />
        ))}
      </div>
    </div>
  );
}

interface SortableHeaderProps {
  field: SortField;
  label: string;
  sortField?: SortField;
  sortDir?: SortDir;
  onSort?: (field: SortField) => void;
}

function SortableHeader({ field, label, sortField, sortDir, onSort }: SortableHeaderProps) {
  const isActive = sortField === field;
  const Icon = isActive && sortDir === "desc" ? ArrowDown : ArrowUp;

  if (!onSort) {
    return <div role="columnheader">{label}</div>;
  }

  return (
    <button
      type="button"
      role="columnheader"
      aria-sort={isActive ? (sortDir === "asc" ? "ascending" : "descending") : "none"}
      className="flex items-center gap-1 cursor-pointer select-none hover:text-fg-1 transition-colors"
      onClick={() => onSort(field)}
    >
      {label}
      <Icon
        className={cn(
          "h-3 w-3 transition-opacity",
          isActive ? "opacity-100 text-fg-1" : "opacity-0",
        )}
      />
    </button>
  );
}

function FileRow({ node }: { node: NodeResponse }) {
  const [renameOpen, setRenameOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [moveOpen, setMoveOpen] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);

  const Icon = iconForNode({
    kind: node.kind,
    mime: node.mime_type ?? null,
    name: node.name,
  });
  const isFolder = node.kind === "folder";
  const isDeleted = Boolean(node.deleted_at);

  return (
    <div
      role="row"
      className={cn(
        "grid grid-cols-[1fr_140px_180px_44px] items-center gap-3 px-4 transition-colors",
        "h-[var(--row-h)] hover:bg-bg-2",
        isDeleted && "opacity-60",
      )}
    >
      <div role="cell" className="flex items-center gap-3 min-w-0">
        <Icon
          className={cn(
            "h-5 w-5 shrink-0",
            isFolder ? "text-accent-1" : "text-fg-2",
          )}
        />
        {isFolder && !isDeleted ? (
          <Link
            to="/files/$folderId"
            params={{ folderId: node.id }}
            className="truncate hover:underline"
          >
            {node.name}
          </Link>
        ) : (
          <span className="truncate">{node.name}</span>
        )}
      </div>
      <div role="cell" className="text-fg-2 truncate">
        {isFolder ? "—" : formatBytes(node.size_bytes)}
      </div>
      <div role="cell" className="text-fg-2 truncate">
        {formatRelativeTime(node.updated_at)}
      </div>
      <div role="cell" className="text-right">
        <RowActionsMenu
          node={node}
          onRename={() => setRenameOpen(true)}
          onDelete={() => setDeleteOpen(true)}
          onMove={() => setMoveOpen(true)}
          onShare={() => setShareOpen(true)}
        />
      </div>

      <RenameDialog open={renameOpen} onOpenChange={setRenameOpen} node={node} />
      <DeleteDialog open={deleteOpen} onOpenChange={setDeleteOpen} node={node} />
      <MoveDialog open={moveOpen} onOpenChange={setMoveOpen} node={node} />
      <ShareDialog open={shareOpen} onOpenChange={setShareOpen} node={node} />
    </div>
  );
}

interface RowActionsProps {
  node: NodeResponse;
  onRename: () => void;
  onDelete: () => void;
  onMove: () => void;
  onShare: () => void;
}

function RowActionsMenu({ node, onRename, onDelete, onMove, onShare }: RowActionsProps) {
  const download = useDownload();
  const restore = useRestoreNode();
  const isDeleted = Boolean(node.deleted_at);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={`Действия для ${node.name}`}
        className="inline-flex h-7 w-7 items-center justify-center rounded-md text-fg-2 hover:bg-bg-3 hover:text-fg-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-1 focus-visible:ring-offset-2 focus-visible:ring-offset-bg-0"
      >
        <MoreHorizontal className="h-4 w-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        {isDeleted ? (
          <DropdownMenuItem onSelect={() => restore.mutate(node.id)}>
            <RotateCcw className="h-4 w-4" />
            Восстановить
          </DropdownMenuItem>
        ) : (
          <>
            {node.kind === "file" && (
              <DropdownMenuItem onSelect={() => download.mutate(node.id)}>
                <Download className="h-4 w-4" />
                Скачать
              </DropdownMenuItem>
            )}
            <DropdownMenuItem onSelect={onRename}>
              <Pencil className="h-4 w-4" />
              Переименовать
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={onMove}>
              <Move className="h-4 w-4" />
              Переместить
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={onShare}>
              <Share2 className="h-4 w-4" />
              Поделиться
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem intent="danger" onSelect={onDelete}>
              <Trash2 className="h-4 w-4" />
              Удалить
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function FileTableSkeleton() {
  return (
    <div className="border-t border-border-1">
      {Array.from({ length: 8 }).map((_, i) => (
        <div
          key={i}
          className="grid grid-cols-[1fr_140px_180px_44px] items-center gap-3 px-4 h-[var(--row-h)] border-b border-border-1"
        >
          <div className="flex items-center gap-3">
            <Skeleton className="h-5 w-5 rounded" />
            <Skeleton className="h-4 w-1/2" />
          </div>
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-24" />
          <div />
        </div>
      ))}
    </div>
  );
}
