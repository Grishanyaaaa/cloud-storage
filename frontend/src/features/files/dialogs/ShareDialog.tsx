import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Copy, Loader2, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogClose,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { createShare, listShares, revokeShare } from "@/api/storage";
import { ApiError, type NodeResponse, type ShareResponse, type SharePermission } from "@/api/types";
import { qk } from "../queryKeys";
import { rewriteShareUrl } from "@/lib/format";
import { env } from "@/lib/env";

interface Props {
  node: NodeResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ShareDialog({ node, open, onOpenChange }: Props) {
  const queryClient = useQueryClient();
  const [permission, setPermission] = useState<SharePermission>("view");
  const [expiresIn, setExpiresIn] = useState<string>("never");

  const sharesQuery = useQuery({
    queryKey: qk.shares(node.id, false),
    queryFn: () => listShares(node.id, false),
    enabled: open,
  });

  const create = useMutation({
    mutationFn: () =>
      createShare(node.id, {
        permission,
        ...(expiresIn !== "never" && { expires_in: expiresIn }),
      }),
    onSuccess: async (created) => {
      // Returned `url` is the API-gateway public-share URL. Rewrite to the
      // frontend-served /share/{token} so the link points at the SPA.
      const link = created.token
        ? rewriteShareUrl(created.url ?? "", created.token, env.SHARE_BASE_URL)
        : created.url ?? "";
      if (link) {
        await copyToClipboard(link);
        toast.success("Ссылка создана и скопирована");
      } else {
        toast.success("Ссылка создана");
      }
      void queryClient.invalidateQueries({ queryKey: qk.shares(node.id, false) });
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : "Не удалось создать ссылку";
      toast.error(msg);
    },
  });

  const revoke = useMutation({
    mutationFn: (shareId: string) => revokeShare(shareId),
    onSuccess: () => {
      toast.success("Ссылка отозвана");
      void queryClient.invalidateQueries({ queryKey: qk.shares(node.id, false) });
    },
    onError: (err) => {
      toast.error(err instanceof ApiError ? err.message : "Не удалось отозвать");
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Поделиться</DialogTitle>
          <DialogDescription>
            «{node.name}» — управление публичными ссылками
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-[1fr_140px_auto] gap-2 items-end">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="share-perm">Права</Label>
            <Select
              value={permission}
              onValueChange={(v) => setPermission(v as SharePermission)}
            >
              <SelectTrigger id="share-perm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="view">Только просмотр</SelectItem>
                <SelectItem value="edit">Редактирование</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="share-exp">Истекает</Label>
            <Select value={expiresIn} onValueChange={setExpiresIn}>
              <SelectTrigger id="share-exp">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="never">Никогда</SelectItem>
                <SelectItem value="1h">1 час</SelectItem>
                <SelectItem value="24h">1 день</SelectItem>
                <SelectItem value="168h">1 неделя</SelectItem>
                <SelectItem value="720h">30 дней</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button
            type="button"
            disabled={create.isPending}
            onClick={() => create.mutate()}
          >
            {create.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Создать
          </Button>
        </div>

        <div className="mt-2">
          <div className="text-[12px] uppercase tracking-wider text-fg-3 mb-2">
            Активные ссылки
          </div>
          {sharesQuery.isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </div>
          ) : sharesQuery.data?.items.length ? (
            <ul className="divide-y divide-border-1 rounded-md border border-border-1 bg-bg-2">
              {sharesQuery.data.items.map((share) => (
                <ShareRow
                  key={share.id}
                  share={share}
                  onCopy={async () => {
                    const link = share.token
                      ? rewriteShareUrl(share.url ?? "", share.token, env.SHARE_BASE_URL)
                      : share.url ?? "";
                    if (link) {
                      await copyToClipboard(link);
                      toast.success("Скопировано");
                    } else {
                      toast.error("Ссылка недоступна");
                    }
                  }}
                  onRevoke={() => revoke.mutate(share.id)}
                  isRevoking={revoke.isPending && revoke.variables === share.id}
                />
              ))}
            </ul>
          ) : (
            <div className="text-fg-3 text-sm text-center py-4 border border-dashed border-border-1 rounded-md">
              Пока нет ссылок
            </div>
          )}
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button intent="secondary" type="button">
              Закрыть
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface ShareRowProps {
  share: ShareResponse;
  onCopy: () => void;
  onRevoke: () => void;
  isRevoking: boolean;
}

function ShareRow({ share, onCopy, onRevoke, isRevoking }: ShareRowProps) {
  // The token is only included on Create — for previously-existing shares
  // we fall back to displaying the rewritten URL if the server provides one.
  const display = share.token
    ? rewriteShareUrl(share.url ?? "", share.token, env.SHARE_BASE_URL)
    : share.url ?? "—";

  return (
    <li className="flex items-center gap-2 px-3 py-2">
      <div className="flex-1 min-w-0">
        <Input readOnly value={display} className="text-[12px] h-8" />
        <div className="text-[11px] text-fg-3 mt-1">
          {share.permission === "edit" ? "Редактирование" : "Только просмотр"}
          {share.expires_at && ` · до ${new Date(share.expires_at).toLocaleString("ru-RU")}`}
        </div>
      </div>
      <Button
        intent="ghost"
        size="icon"
        aria-label="Скопировать"
        onClick={onCopy}
        disabled={display === "—"}
      >
        <Copy className="h-4 w-4" />
      </Button>
      <Button
        intent="ghost"
        size="icon"
        aria-label="Отозвать"
        onClick={onRevoke}
        disabled={isRevoking}
      >
        {isRevoking ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4 text-danger" />}
      </Button>
    </li>
  );
}

async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    // Fallback for older browsers / non-secure contexts.
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
  }
}
