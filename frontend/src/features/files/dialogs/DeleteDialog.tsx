import { Loader2, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { NodeResponse } from "@/api/types";
import { useDeleteNode } from "../useFileMutations";

interface Props {
  node: NodeResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DeleteDialog({ node, open, onOpenChange }: Props) {
  const mutation = useDeleteNode();
  const isFolder = node.kind === "folder";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-danger-soft text-danger">
              <AlertTriangle className="h-5 w-5" />
            </div>
            <div>
              <DialogTitle>Удалить {isFolder ? "папку" : "файл"}?</DialogTitle>
              <DialogDescription>
                «{node.name}»{isFolder && " и всё её содержимое"} будут перемещены
                в корзину.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" intent="secondary">
              Отмена
            </Button>
          </DialogClose>
          <Button
            type="button"
            intent="danger"
            onClick={() =>
              mutation.mutate(node.id, {
                onSuccess: () => onOpenChange(false),
              })
            }
            disabled={mutation.isPending}
          >
            {mutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Удаление…
              </>
            ) : (
              "Удалить"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
