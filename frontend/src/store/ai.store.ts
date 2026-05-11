import { create } from "zustand";
import type { AICommandResponse } from "@/api/types";

export type AIModalState =
  | { phase: "idle" }
  | { phase: "planning"; input: string }
  | { phase: "plan-ready"; cmd: AICommandResponse }
  | { phase: "executing"; id: string }
  | { phase: "done"; cmd: AICommandResponse }
  | { phase: "error"; message: string };

interface AIModalStore {
  isOpen: boolean;
  state: AIModalState;
  open(): void;
  close(): void;
  setState(state: AIModalState): void;
  reset(): void;
}

export const useAIModalStore = create<AIModalStore>((set) => ({
  isOpen: false,
  state: { phase: "idle" },
  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  setState: (state) => set({ state }),
  reset: () => set({ state: { phase: "idle" } }),
}));
