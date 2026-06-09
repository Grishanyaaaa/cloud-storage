import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  createFolder,
  deleteNode,
  moveNode,
  renameNode,
  restoreNode,
} from "@/api/storage";
import { ApiError } from "@/api/types";
import { qk } from "./queryKeys";

function errMessage(err: unknown, fallback: string): string {
  return err instanceof ApiError ? err.message : fallback;
}

/**
 * After any mutation that affects the tree, we invalidate broadly:
 * the children of the affected parent, the entire tree (sidebar / breadcrumbs),
 * and the touched node. This is intentionally coarse — per blueprint §10 we
 * trade pinpoint cache surgery for correctness.
 */
function invalidateTreeAndChildren(
  queryClient: ReturnType<typeof useQueryClient>,
  affectedParents: string[],
) {
  const promises: Promise<unknown>[] = [
    queryClient.invalidateQueries({ queryKey: ["tree"], exact: false }),
  ];
  for (const parentId of affectedParents) {
    if (parentId) {
      promises.push(
        queryClient.invalidateQueries({
          queryKey: ["children", parentId],
          exact: false,
        }),
      );
    }
  }
  void Promise.all(promises);
}

export function useCreateFolder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { parent_id: string; name: string }) => createFolder(input),
    onSuccess: (created) => {
      toast.success(`Папка «${created.name}» создана`);
      invalidateTreeAndChildren(queryClient, [created.parent_id ?? ""]);
    },
    onError: (err) => toast.error(errMessage(err, "Не удалось создать папку")),
  });
}

export function useRenameNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (args: { id: string; name: string }) =>
      renameNode(args.id, { name: args.name }),
    onSuccess: (updated) => {
      toast.success(`Переименовано в «${updated.name}»`);
      void queryClient.invalidateQueries({ queryKey: qk.node(updated.id) });
      invalidateTreeAndChildren(queryClient, [updated.parent_id ?? ""]);
    },
    onError: (err) => toast.error(errMessage(err, "Не удалось переименовать")),
  });
}

export function useMoveNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (args: { id: string; new_parent_id: string; oldParentId?: string }) =>
      moveNode(args.id, { new_parent_id: args.new_parent_id }),
    onSuccess: (updated, vars) => {
      toast.success(`«${updated.name}» перемещён`);

      const oldParentId = vars.oldParentId;
      const affectedParents = [oldParentId, vars.new_parent_id].filter(Boolean) as string[];
      invalidateTreeAndChildren(queryClient, affectedParents);
      void queryClient.invalidateQueries({ queryKey: qk.node(updated.id) });
    },
    onError: (err) => toast.error(errMessage(err, "Не удалось переместить")),
  });
}

export function useDeleteNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteNode(id),
    onSuccess: () => {
      toast.success("Удалено");
      // We don't know the parent here without an extra round-trip; invalidate
      // the whole tree + all children caches.
      void queryClient.invalidateQueries({ queryKey: ["tree"], exact: false });
      void queryClient.invalidateQueries({ queryKey: ["children"], exact: false });
    },
    onError: (err) => toast.error(errMessage(err, "Не удалось удалить")),
  });
}

export function useRestoreNode() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => restoreNode(id),
    onSuccess: (restored) => {
      toast.success(`«${restored.name}» восстановлен`);
      void queryClient.invalidateQueries({ queryKey: ["tree"], exact: false });
      void queryClient.invalidateQueries({ queryKey: ["children"], exact: false });
    },
    onError: (err) => toast.error(errMessage(err, "Не удалось восстановить")),
  });
}
