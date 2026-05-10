import { abortUpload, finalizeUpload, getUploadURL } from "@/api/storage";
import { ApiError } from "@/api/types";
import { useUploadsStore, type UploadEntry } from "@/store/uploads.store";
import { sha256Hex } from "./sha256";

interface RunUploadArgs {
  entry: UploadEntry;
}

/**
 * Drives a single upload through the 3-phase storage-service flow:
 *   1. POST /storage/v1/files/upload-url  →  { node_id, url, headers }
 *   2. PUT to S3 (XHR for progress events; uses returned headers verbatim)
 *   3. POST /storage/v1/files/{id}/finalize  with size + sha256-hex
 *
 * On any failure after step 1 we best-effort POST /abort so the pending
 * blob doesn't stay around as a ghost — the storage-service janitor will
 * eventually expire it, but explicit cleanup is faster.
 *
 * The XHR is exposed via entry.abort so the user can cancel mid-flight.
 */
export async function runUpload({ entry }: RunUploadArgs): Promise<void> {
  const update = useUploadsStore.getState().update;

  // Phase 0: hash. Uses Web Crypto; whole file in memory (see sha256.ts).
  update(entry.id, { status: "hashing", uploadedBytes: 0 });
  let checksum: string;
  try {
    checksum = await sha256Hex(entry.file);
  } catch (err) {
    update(entry.id, {
      status: "error",
      error: err instanceof Error ? err.message : "Не удалось вычислить хэш",
    });
    return;
  }

  // Phase 1: ask storage-service for a presigned PUT.
  update(entry.id, { status: "issuing" });
  let issued: Awaited<ReturnType<typeof getUploadURL>>;
  try {
    issued = await getUploadURL({
      parent_id: entry.parentId,
      name: entry.name,
      size_bytes: entry.size,
      ...(entry.file.type && { mime_type: entry.file.type }),
    });
  } catch (err) {
    update(entry.id, {
      status: "error",
      error: err instanceof ApiError ? err.message : "Не удалось получить ссылку для загрузки",
    });
    return;
  }

  // Phase 2: PUT to S3 via XHR (fetch has no progress events).
  update(entry.id, { status: "uploading", nodeId: issued.node_id });
  try {
    await new Promise<void>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open(issued.method || "PUT", issued.url, true);
      for (const [k, v] of Object.entries(issued.headers ?? {})) {
        xhr.setRequestHeader(k, v);
      }
      xhr.upload.onprogress = (ev) => {
        if (ev.lengthComputable) {
          update(entry.id, { uploadedBytes: ev.loaded });
        }
      };
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          update(entry.id, { uploadedBytes: entry.size });
          resolve();
        } else {
          reject(new Error(`S3 ответил со статусом ${xhr.status}`));
        }
      };
      xhr.onerror = () => reject(new Error("Сетевой сбой при загрузке"));
      xhr.onabort = () => reject(new DOMException("aborted", "AbortError"));
      // Expose abort handle.
      useUploadsStore.getState().update(entry.id, { abort: () => xhr.abort() });
      xhr.send(entry.file);
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      // Best-effort cleanup of the pending blob.
      void abortUpload(issued.node_id).catch(() => {});
      update(entry.id, { status: "cancelled" });
      return;
    }
    void abortUpload(issued.node_id).catch(() => {});
    update(entry.id, {
      status: "error",
      error: err instanceof Error ? err.message : "Сбой загрузки",
    });
    return;
  }

  // Phase 3: finalize with the precomputed checksum.
  update(entry.id, { status: "finalizing" });
  try {
    await finalizeUpload(issued.node_id, {
      size_bytes: entry.size,
      checksum,
    });
    update(entry.id, { status: "done" });
  } catch (err) {
    void abortUpload(issued.node_id).catch(() => {});
    update(entry.id, {
      status: "error",
      error: err instanceof ApiError ? err.message : "Не удалось завершить загрузку",
    });
  }
}
