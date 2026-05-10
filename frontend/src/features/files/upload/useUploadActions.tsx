import { type PropsWithChildren } from "react";

/**
 * Stub. The real implementation (presigned PUT, drag-n-drop, queue, progress
 * panel) lands in the upload-feature commit.
 */
export function useUploadActions(_parentId: string) {
  function openFilePicker() {
    /* TODO */
  }

  function DropZone({ children }: PropsWithChildren) {
    return <div className="flex-1 overflow-auto">{children}</div>;
  }

  return { openFilePicker, DropZone };
}
