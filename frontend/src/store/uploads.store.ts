import { create } from "zustand";

export type UploadStatus =
  | "queued"
  | "hashing"
  | "issuing"
  | "uploading"
  | "finalizing"
  | "done"
  | "error"
  | "cancelled";

export interface UploadEntry {
  id: string;
  parentId: string;
  file: File;
  name: string;
  size: number;
  uploadedBytes: number;
  status: UploadStatus;
  error?: string;
  nodeId?: string;
  /** Last update time — used by progress throttling. */
  updatedAt: number;
  abort?: () => void;
}

interface UploadsState {
  entries: UploadEntry[];
  panelOpen: boolean;
  setPanelOpen(open: boolean): void;
  add(entry: UploadEntry): void;
  update(id: string, patch: Partial<UploadEntry>): void;
  remove(id: string): void;
  clearCompleted(): void;
}

export const useUploadsStore = create<UploadsState>((set) => ({
  entries: [],
  panelOpen: false,
  setPanelOpen: (open) => set({ panelOpen: open }),
  add: (entry) =>
    set((s) => ({ entries: [entry, ...s.entries], panelOpen: true })),
  update: (id, patch) =>
    set((s) => ({
      entries: s.entries.map((e) =>
        e.id === id ? { ...e, ...patch, updatedAt: Date.now() } : e,
      ),
    })),
  remove: (id) =>
    set((s) => ({ entries: s.entries.filter((e) => e.id !== id) })),
  clearCompleted: () =>
    set((s) => ({
      entries: s.entries.filter((e) => e.status !== "done" && e.status !== "cancelled"),
    })),
}));
