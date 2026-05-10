import { useQuery } from "@tanstack/react-query";
import { getNode, getTree, listChildren } from "@/api/storage";
import { env } from "@/lib/env";
import { qk } from "./queryKeys";

export function useTree(rootId?: string) {
  return useQuery({
    queryKey: qk.tree(rootId),
    queryFn: () =>
      getTree({
        ...(rootId !== undefined && { rootId }),
        maxDepth: env.DEFAULT_TREE_DEPTH,
      }),
    staleTime: 60_000,
  });
}

export function useChildren(parentId: string, opts: { includeDeleted?: boolean } = {}) {
  return useQuery({
    queryKey: qk.children(parentId, opts),
    queryFn: () =>
      listChildren({
        parentId,
        limit: 200,
        ...(opts.includeDeleted !== undefined && { includeDeleted: opts.includeDeleted }),
      }),
    enabled: Boolean(parentId),
    staleTime: 30_000,
  });
}

export function useNode(id: string | null | undefined) {
  return useQuery({
    queryKey: qk.node(id ?? ""),
    queryFn: () => getNode(id as string),
    enabled: Boolean(id),
    staleTime: 60_000,
  });
}
