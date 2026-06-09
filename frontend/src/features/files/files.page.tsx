import { useParams } from "@tanstack/react-router";
import { useState } from "react";
import { FolderPlus, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AuthGate } from "@/components/common/AuthGate";
import { AppShell } from "@/components/layout/AppShell";
import { Breadcrumbs } from "./Breadcrumbs";
import { FileTable } from "./FileTable";
import { FileToolbar } from "./FileToolbar";
import { NewFolderDialog } from "./dialogs/NewFolderDialog";
import { useChildren, useTree } from "./useFilesData";
import { useUploadActions } from "./upload/useUploadActions";
import { useFileFilters, useFilteredItems } from "./useFileFilters";

export function FilesPage() {
  return (
    <AuthGate>
      <FilesPageInner />
    </AuthGate>
  );
}

function FilesPageInner() {
  const { folderId } = useParams({ from: "/files/$folderId" });
  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const tree = useTree();
  const children = useChildren(folderId);
  const upload = useUploadActions(folderId);
  const filters = useFileFilters();

  const allItems = children.data?.items ?? [];
  const filteredItems = useFilteredItems(allItems, filters);

  return (
    <AppShell>
      <div className="flex flex-col h-full">
        <header className="flex items-center justify-between gap-3 px-6 py-3 border-b border-border-1 bg-bg-0">
          <Breadcrumbs tree={tree.data} currentId={folderId} />
          <div className="flex items-center gap-2">
            <Button intent="secondary" size="md" onClick={() => setNewFolderOpen(true)}>
              <FolderPlus className="h-4 w-4" />
              Новая папка
            </Button>
            <Button intent="primary" size="md" onClick={upload.openFilePicker}>
              <Upload className="h-4 w-4" />
              Загрузить
            </Button>
          </div>
        </header>
        <FileToolbar
          filters={filters}
          totalCount={allItems.length}
          filteredCount={filteredItems.length}
        />
        <upload.DropZone>
          <FileTable
            items={filteredItems}
            isLoading={children.isLoading}
            isError={children.isError}
            emptyMessage={
              filters.search || filters.kindFilter !== "all"
                ? "Ничего не найдено. Попробуйте изменить фильтры."
                : undefined
            }
            sortField={filters.sortField}
            sortDir={filters.sortDir}
            onSort={filters.setSort}
          />
        </upload.DropZone>
      </div>
      <NewFolderDialog
        parentId={folderId}
        open={newFolderOpen}
        onOpenChange={setNewFolderOpen}
      />
    </AppShell>
  );
}
