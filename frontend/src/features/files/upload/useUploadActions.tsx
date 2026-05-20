import { useCallback, useEffect, useRef, useState, type DragEvent, type PropsWithChildren } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Upload as UploadIcon } from "lucide-react";
import { useUploadsStore } from "@/store/uploads.store";
import { env } from "@/lib/env";
import { cn } from "@/lib/cn";
import { genUploadId } from "@/lib/id";
import { ensureUploadQueueStarted } from "./uploadQueue";

/**
 * Hook used by FilesPage. Exposes:
 *   - openFilePicker: button-driven multi-file select
 *   - DropZone: wrapper component that handles dragenter/over/drop on the
 *     scrollable file area
 *
 * Both push entries into the upload store, which the worker pool drains.
 * On every transition into a fresh "done" state we invalidate the children
 * queue once to refresh the table.
 */
export function useUploadActions(parentId: string) {
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const enqueueFilesRef = useRef<(files: FileList | File[]) => void>();
  const [dragOver, setDragOver] = useState(false);
  const dragCounterRef = useRef(0);

  // Boot the worker pool once (idempotent).
  useEffect(() => {
    ensureUploadQueueStarted();
  }, []);

  // When an entry transitions to "done", refresh the affected children list.
  useEffect(() => {
    let lastDone = 0;
    const unsub = useUploadsStore.subscribe((state) => {
      const doneNow = state.entries.filter((e) => e.status === "done").length;
      if (doneNow > lastDone) {
        lastDone = doneNow;
        void queryClient.invalidateQueries({ queryKey: ["children"], exact: false });
        void queryClient.invalidateQueries({ queryKey: ["tree"], exact: false });
      } else {
        lastDone = doneNow;
      }
    });
    return () => unsub();
  }, [queryClient]);

  const enqueueFiles = useCallback(
    (files: FileList | File[]) => {
      const arr = Array.from(files);
      const add = useUploadsStore.getState().add;
      for (const file of arr) {
        if (file.size > env.UPLOAD_MAX_BYTES) {
          add({
            id: genUploadId(),
            parentId,
            file,
            name: file.name,
            size: file.size,
            uploadedBytes: 0,
            status: "error",
            error: "Файл слишком большой",
            updatedAt: Date.now(),
          });
          continue;
        }
        add({
          id: genUploadId(),
          parentId,
          file,
          name: file.name,
          size: file.size,
          uploadedBytes: 0,
          status: "queued",
          updatedAt: Date.now(),
        });
      }
    },
    [parentId],
  );

  // Keep ref updated so event listener always uses current version
  enqueueFilesRef.current = enqueueFiles;

  const openFilePicker = useCallback(() => {
    let el = inputRef.current;
    if (!el) {
      el = document.createElement("input");
      el.type = "file";
      el.multiple = true;
      el.style.display = "none";
      el.addEventListener("change", () => {
        if (el && el.files && enqueueFilesRef.current) {
          enqueueFilesRef.current(el.files);
        }
      });
      document.body.appendChild(el);
      inputRef.current = el;
    }
    el.value = "";
    el.click();
  }, []);

  const onDragEnter = useCallback((e: DragEvent) => {
    e.preventDefault();
    if (!e.dataTransfer.types.includes("Files")) return;
    dragCounterRef.current++;
    setDragOver(true);
  }, []);
  const onDragOver = useCallback((e: DragEvent) => {
    e.preventDefault();
    if (e.dataTransfer.types.includes("Files")) {
      e.dataTransfer.dropEffect = "copy";
    }
  }, []);
  const onDragLeave = useCallback((e: DragEvent) => {
    e.preventDefault();
    dragCounterRef.current--;
    if (dragCounterRef.current <= 0) {
      dragCounterRef.current = 0;
      setDragOver(false);
    }
  }, []);
  const onDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault();
      dragCounterRef.current = 0;
      setDragOver(false);
      if (e.dataTransfer.files.length > 0) {
        enqueueFiles(e.dataTransfer.files);
      }
    },
    [enqueueFiles],
  );

  const DropZone = useCallback(
    ({ children }: PropsWithChildren) => (
      <div
        onDragEnter={onDragEnter}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
        className="relative flex-1 overflow-auto"
      >
        {children}
        <div
          aria-hidden={!dragOver}
          className={cn(
            "pointer-events-none absolute inset-2 rounded-lg border-2 border-dashed border-accent-1 bg-accent-soft/40 transition-opacity flex items-center justify-center",
            dragOver ? "opacity-100" : "opacity-0",
          )}
        >
          <div className="flex flex-col items-center gap-2 text-fg-1">
            <UploadIcon className="h-8 w-8 text-accent-1" />
            <div className="text-sm font-medium">Отпустите, чтобы загрузить</div>
          </div>
        </div>
      </div>
    ),
    [dragOver, onDragEnter, onDragOver, onDragLeave, onDrop],
  );

  return { openFilePicker, DropZone };
}
