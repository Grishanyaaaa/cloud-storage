import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { NodeResponse } from "@/api/types";

interface Props {
  node: NodeResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Stub. Wired up in the share-feature commit.
 */
export function ShareDialog({ node, open, onOpenChange }: Props) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Поделиться</DialogTitle>
          <DialogDescription>«{node.name}»</DialogDescription>
        </DialogHeader>
        <div className="text-fg-2 text-sm">Скоро.</div>
      </DialogContent>
    </Dialog>
  );
}
