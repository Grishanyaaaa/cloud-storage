import { apiFetch } from "./client";
import type {
  DownloadURLResponse,
  ListChildrenResponse,
  ListSharesResponse,
  NodeResponse,
  ShareResponse,
  SharePermission,
  TreeNodeResponse,
  UploadURLResponse,
} from "./types";

/* ----------------------------- root ------------------------------ */

export async function ensureRoot(): Promise<NodeResponse> {
  return apiFetch<NodeResponse>("/storage/v1/me/root", { method: "POST" });
}

/* ------------------------------ tree ----------------------------- */

export async function getTree(args?: {
  rootId?: string;
  maxDepth?: number;
  includeDeleted?: boolean;
}): Promise<TreeNodeResponse> {
  return apiFetch<TreeNodeResponse>("/storage/v1/tree", {
    method: "GET",
    query: {
      root_id: args?.rootId,
      max_depth: args?.maxDepth,
      include_deleted: args?.includeDeleted,
    },
  });
}

/* ----------------------------- nodes ----------------------------- */

export async function getNode(id: string): Promise<NodeResponse> {
  return apiFetch<NodeResponse>(`/storage/v1/nodes/${encodeURIComponent(id)}`, {
    method: "GET",
  });
}

export async function listChildren(args: {
  parentId: string;
  cursor?: string;
  limit?: number;
  includeDeleted?: boolean;
}): Promise<ListChildrenResponse> {
  return apiFetch<ListChildrenResponse>(
    `/storage/v1/folders/${encodeURIComponent(args.parentId)}/children`,
    {
      method: "GET",
      query: {
        cursor: args.cursor,
        limit: args.limit,
        include_deleted: args.includeDeleted,
      },
    },
  );
}

export async function createFolder(input: {
  parent_id: string;
  name: string;
}): Promise<NodeResponse> {
  return apiFetch<NodeResponse>("/storage/v1/folders/", {
    method: "POST",
    body: input,
  });
}

export async function renameNode(id: string, input: { name: string }): Promise<NodeResponse> {
  return apiFetch<NodeResponse>(`/storage/v1/nodes/${encodeURIComponent(id)}/rename`, {
    method: "PATCH",
    body: input,
  });
}

export async function moveNode(
  id: string,
  input: { new_parent_id: string },
): Promise<NodeResponse> {
  return apiFetch<NodeResponse>(`/storage/v1/nodes/${encodeURIComponent(id)}/move`, {
    method: "PATCH",
    body: input,
  });
}

export async function deleteNode(id: string): Promise<void> {
  return apiFetch<void>(`/storage/v1/nodes/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export async function restoreNode(id: string): Promise<NodeResponse> {
  return apiFetch<NodeResponse>(`/storage/v1/nodes/${encodeURIComponent(id)}/restore`, {
    method: "POST",
  });
}

/* ------------------------------ files ---------------------------- */

export async function generateUploadURL(input: {
  parent_id: string;
  name: string;
  size_bytes: number;
  mime_type: string;
}): Promise<UploadURLResponse> {
  return apiFetch<UploadURLResponse>("/storage/v1/files/upload-url", {
    method: "POST",
    body: input,
  });
}

export async function finalizeUpload(
  nodeId: string,
  input: { size_bytes: number; checksum: string },
): Promise<NodeResponse> {
  return apiFetch<NodeResponse>(`/storage/v1/files/${encodeURIComponent(nodeId)}/finalize`, {
    method: "POST",
    body: input,
  });
}

export async function abortUpload(nodeId: string): Promise<void> {
  return apiFetch<void>(`/storage/v1/files/${encodeURIComponent(nodeId)}/abort`, {
    method: "POST",
  });
}

export async function getDownloadURL(
  nodeId: string,
  disposition: "attachment" | "inline" = "attachment",
): Promise<DownloadURLResponse> {
  return apiFetch<DownloadURLResponse>(
    `/storage/v1/files/${encodeURIComponent(nodeId)}/download-url`,
    {
      method: "GET",
      query: { disposition },
    },
  );
}

/* ----------------------------- shares ---------------------------- */

export async function createShare(
  nodeId: string,
  input: { permission: SharePermission; expires_in?: string },
): Promise<ShareResponse> {
  return apiFetch<ShareResponse>(`/storage/v1/nodes/${encodeURIComponent(nodeId)}/shares`, {
    method: "POST",
    body: input,
  });
}

export async function listShares(
  nodeId: string,
  includeRevoked = false,
): Promise<ListSharesResponse> {
  return apiFetch<ListSharesResponse>(
    `/storage/v1/nodes/${encodeURIComponent(nodeId)}/shares`,
    {
      method: "GET",
      query: { include_revoked: includeRevoked },
    },
  );
}

export async function revokeShare(shareId: string): Promise<void> {
  return apiFetch<void>(`/storage/v1/shares/${encodeURIComponent(shareId)}`, {
    method: "DELETE",
  });
}
