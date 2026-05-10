import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { getDownloadURL } from "@/api/storage";
import { ApiError } from "@/api/types";

/**
 * Two-step download: fetch a presigned GET URL from the storage-service,
 * then point the browser at it. Browser handles the actual S3 GET — the
 * URL has Content-Disposition: attachment baked in.
 *
 * Stub for the dedicated commit; real implementation follows.
 */
export function useDownload() {
  return useMutation({
    mutationFn: (nodeId: string) => getDownloadURL(nodeId, "attachment"),
    onSuccess: (resp) => {
      // Force a download via temporary anchor (works around popup-blockers
      // when triggered from a click handler).
      const link = document.createElement("a");
      link.href = resp.url;
      link.rel = "noopener";
      document.body.appendChild(link);
      link.click();
      link.remove();
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : "Не удалось скачать файл";
      toast.error(msg);
    },
  });
}
