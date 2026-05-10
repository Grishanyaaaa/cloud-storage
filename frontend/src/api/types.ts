/**
 * Shared API types and the canonical ServerEnvelope shape.
 *
 * All upstream services (auth/storage/ai) wrap their bodies in:
 *   { "status": "success", "data": ... }     // success
 *   { "status": "error",   "error": "...",   // failure
 *     "code": "DOMAIN_CODE" }                // (storage-service only sets `code`)
 */

export type ServerEnvelope<T> =
  | { status: "success"; data: T }
  | { status: "error"; error: string; code?: string };

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(args: { status: number; code: string; message: string }) {
    super(args.message);
    this.name = "ApiError";
    this.status = args.status;
    this.code = args.code;
  }
}

export class NetworkError extends Error {
  readonly cause: unknown;

  constructor(message: string, cause?: unknown) {
    super(message);
    this.name = "NetworkError";
    this.cause = cause;
  }
}

/* ---------------- canonical DTOs (JSON wire format) ---------------- */

/**
 * Auth DTOs. NOTE: auth-service currently serialises the TokenPair without
 * json tags, which yields PascalCase keys (`AccessToken`, `RefreshToken`).
 * The frontend tolerates both shapes via `pickField`.
 */
export interface AuthTokenPair {
  access_token: string;
  refresh_token: string;
}

export interface RegisterResult {
  user_id: string;
}

/* Storage DTOs (all use snake_case via explicit json tags). */

export type NodeKind = "file" | "folder";
export type FileStatus = "pending" | "active" | "failed";
export type SharePermission = "view" | "edit";

export interface NodeResponse {
  id: string;
  owner_id: string;
  parent_id?: string | null;
  kind: NodeKind;
  name: string;
  path: string;
  depth: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;

  size_bytes?: number | null;
  mime_type?: string | null;
  status?: FileStatus | null;
}

export interface ListChildrenResponse {
  items: NodeResponse[];
  next_cursor?: string;
}

export interface TreeNodeResponse extends NodeResponse {
  children?: TreeNodeResponse[];
}

export interface UploadURLResponse {
  node_id: string;
  url: string;
  method: "PUT";
  headers?: Record<string, string>;
  expires_at: string;
}

export interface DownloadURLResponse {
  url: string;
  method: "GET";
  expires_at: string;
}

export interface ShareResponse {
  id: string;
  node_id: string;
  permission: SharePermission;
  url?: string;
  token?: string;
  expires_at?: string | null;
  revoked_at?: string | null;
  created_at: string;
}

export interface ListSharesResponse {
  items: ShareResponse[];
}

export interface PublicShareResponse {
  node_id: string;
  kind: NodeKind;
  name: string;
  permission: SharePermission;
  expires_at?: string | null;
}

/* AI DTOs. */

export type AICommandStatus =
  | "awaiting_confirmation"
  | "executed"
  | "cancelled"
  | "expired"
  | "failed";

export type AIOpKind = "delete" | "rename" | "move";

export interface AIOperation {
  kind: AIOpKind;
  node_id: string;
  new_name?: string;
  new_parent_id?: string | null;
}

export interface AIOperationResult {
  index: number;
  kind: AIOpKind;
  node_id: string;
  success: boolean;
  error_code?: string;
  error_message?: string;
}

export interface AICommandResponse {
  id: string;
  user_id: string;
  input: string;
  plan: AIOperation[];
  explanation: string;
  status: AICommandStatus;
  llm_model?: string;
  llm_tokens_in?: number;
  llm_tokens_out?: number;
  results?: AIOperationResult[];
  created_at: string;
  expires_at: string;
  executed_at?: string | null;
  cancelled_at?: string | null;
}
