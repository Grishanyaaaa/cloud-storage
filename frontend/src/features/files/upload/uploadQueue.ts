import { useUploadsStore } from "@/store/uploads.store";
import { env } from "@/lib/env";
import { runUpload } from "./uploadFile";

/**
 * Tiny worker pool. Polls the uploads store for queued entries and starts up
 * to UPLOAD_PARALLELISM in parallel. We don't over-engineer with promises:
 * each finished slot triggers a re-poll.
 */

let active = 0;

function tick() {
  const max = env.UPLOAD_PARALLELISM;
  if (active >= max) return;
  const queued = useUploadsStore
    .getState()
    .entries.find((e) => e.status === "queued");
  if (!queued) return;

  active++;
  // Mark started so the next tick won't pick it again.
  useUploadsStore.getState().update(queued.id, { status: "hashing" });

  runUpload({ entry: queued })
    .catch(() => {})
    .finally(() => {
      active--;
      // Drain anything else that's queued.
      tick();
    });
}

let unsub: (() => void) | null = null;

export function ensureUploadQueueStarted(): void {
  if (unsub) return;
  unsub = useUploadsStore.subscribe(() => tick());
  // Initial drain in case entries exist already.
  tick();
}

export function stopUploadQueue(): void {
  unsub?.();
  unsub = null;
}
