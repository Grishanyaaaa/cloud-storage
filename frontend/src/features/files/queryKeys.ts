/** Query key tree for storage-service caches. */
export const qk = {
  me: ["me"] as const,
  root: ["root"] as const,
  tree: (rootId?: string) => ["tree", rootId ?? "root"] as const,
  children: (parentId: string, opts: { includeDeleted?: boolean } = {}) =>
    ["children", parentId, opts.includeDeleted ?? false] as const,
  node: (id: string) => ["node", id] as const,
  shares: (nodeId: string, includeRevoked = false) =>
    ["shares", nodeId, includeRevoked] as const,
  command: (id: string) => ["command", id] as const,

  // Public share queries — namespaced to avoid colliding with owner queries.
  publicShare: (token: string) => ["public-share", token] as const,
  publicShareTree: (token: string) => ["public-share-tree", token] as const,
  publicShareChildren: (token: string, parentId: string) =>
    ["public-share-children", token, parentId] as const,
};
