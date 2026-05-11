import { apiFetch } from "./client";
import type { AICommandResponse } from "./types";

export async function planCommand(input: { input: string }): Promise<AICommandResponse> {
  return apiFetch<AICommandResponse>("/ai/v1/commands", {
    method: "POST",
    body: input,
  });
}

export async function getCommand(id: string): Promise<AICommandResponse> {
  return apiFetch<AICommandResponse>(`/ai/v1/commands/${encodeURIComponent(id)}`, {
    method: "GET",
  });
}

export async function executeCommand(id: string): Promise<AICommandResponse> {
  return apiFetch<AICommandResponse>(`/ai/v1/commands/${encodeURIComponent(id)}/execute`, {
    method: "POST",
  });
}

export async function cancelCommand(id: string): Promise<AICommandResponse> {
  return apiFetch<AICommandResponse>(`/ai/v1/commands/${encodeURIComponent(id)}/cancel`, {
    method: "POST",
  });
}
