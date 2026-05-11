import { apiPublicFetch } from "./client";
import type {
  DownloadURLResponse,
  ListChildrenResponse,
  NodeResponse,
  PublicShareResponse,
  TreeNodeResponse,
} from "./types";

/**
 * Public share endpoints — token authentication via URL path; no Bearer.
 * The frontend uses `apiPublicFetch` (skipAuth=true) to make sure the
 * caller's JWT is NOT leaked to the server when viewing a share.
 */

export async function getShareInfo(token: string): Promise<PublicShareResponse> {
  return apiPublicFetch<PublicShareResponse>(
    `/storage/v1/public/${encodeURIComponent(token)}`,
    { method: "GET" },
  );
}

export async function getShareTree(
  token: string,
  args?: { maxDepth?: number },
): Promise<TreeNodeResponse> {
  return apiPublicFetch<TreeNodeResponse>(
    `/storage/v1/public/${encodeURIComponent(token)}/tree`,
    {
      method: "GET",
      query: { max_depth: args?.maxDepth },
    },
  );
}

export async function listShareChildren(
  token: string,
  args: { parentId: string; cursor?: string; limit?: number },
): Promise<ListChildrenResponse> {
  return apiPublicFetch<ListChildrenResponse>(
    `/storage/v1/public/${encodeURIComponent(token)}/folders/${encodeURIComponent(args.parentId)}/children`,
    {
      method: "GET",
      query: { cursor: args.cursor, limit: args.limit },
    },
  );
}

export async function getShareNode(token: string, nodeId: string): Promise<NodeResponse> {
  return apiPublicFetch<NodeResponse>(
    `/storage/v1/public/${encodeURIComponent(token)}/nodes/${encodeURIComponent(nodeId)}`,
    { method: "GET" },
  );
}

export async function getShareDownloadURL(
  token: string,
  nodeId: string,
  disposition: "attachment" | "inline" = "attachment",
): Promise<DownloadURLResponse> {
  return apiPublicFetch<DownloadURLResponse>(
    `/storage/v1/public/${encodeURIComponent(token)}/files/${encodeURIComponent(nodeId)}/download-url`,
    {
      method: "GET",
      query: { disposition },
    },
  );
}

export async function renameShareNode(
  token: string,
  nodeId: string,
  input: { name: string },
): Promise<NodeResponse> {
  return apiPublicFetch<NodeResponse>(
    `/storage/v1/public/${encodeURIComponent(token)}/nodes/${encodeURIComponent(nodeId)}/rename`,
    {
      method: "PATCH",
      body: input,
    },
  );
}

export async function deleteShareNode(token: string, nodeId: string): Promise<void> {
  return apiPublicFetch<void>(
    `/storage/v1/public/${encodeURIComponent(token)}/nodes/${encodeURIComponent(nodeId)}`,
    { method: "DELETE" },
  );
}
