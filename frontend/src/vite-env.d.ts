/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string;
  readonly VITE_APP_NAME: string;
  readonly VITE_DEFAULT_TREE_DEPTH: string;
  readonly VITE_TREE_MAX_NODES: string;
  readonly VITE_UPLOAD_PARALLELISM: string;
  readonly VITE_UPLOAD_PROGRESS_THROTTLE_MS: string;
  readonly VITE_UPLOAD_MAX_BYTES: string;
  readonly VITE_SHARE_BASE_URL: string;
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare global {
  interface Window {
    __APP_CONFIG__?: {
      API_BASE_URL?: string;
      SHARE_BASE_URL?: string;
    };
  }
}

export {};
