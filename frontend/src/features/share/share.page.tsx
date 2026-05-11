import { useParams } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Download, ExternalLink, FileText, Folder } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  getShareDownloadURL,
  getShareInfo,
  listShareChildren,
} from "@/api/share";
import { ApiError } from "@/api/types";
import type { NodeResponse } from "@/api/types";
import { cn } from "@/lib/cn";
import { formatBytes, formatRelativeTime } from "@/lib/format";
import { iconForNode } from "@/lib/mime";
import { qk } from "@/features/files/queryKeys";

export function SharePage() {
  const { token } = useParams({ from: "/share/$token" });

  const info = useQuery({
    queryKey: qk.publicShare(token),
    queryFn: () => getShareInfo(token),
    retry: 1,
  });

  if (info.isLoading) {
    return <SharePageShell><SharePageSkeleton /></SharePageShell>;
  }
  if (info.isError) {
    const msg = info.error instanceof ApiError ? info.error.message : "Не удалось открыть ссылку";
    return (
      <SharePageShell>
        <div className="text-center py-16">
          <div className="text-2xl font-semibold mb-2">Ссылка недоступна</div>
          <div className="text-fg-2 text-sm">{msg}</div>
        </div>
      </SharePageShell>
    );
  }
  if (!info.data) return null;

  return (
    <SharePageShell>
      <SharedView token={token} share={info.data} />
    </SharePageShell>
  );
}

function SharePageShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-bg-0 text-fg-1">
      <header className="h-[var(--topbar-h)] border-b border-border-1 bg-bg-1 flex items-center justify-between px-6">
        <div className="flex items-center gap-2 select-none">
          <div className="h-7 w-7 rounded-md bg-bg-3 flex items-center justify-center">
            <svg width="16" height="16" viewBox="0 0 32 32" fill="none">
              <path
                d="M9 11.5l4.2-2.8 4.2 2.8M22.6 11.5l-4.2-2.8M9 11.5l4.2 2.8M22.6 11.5l-4.2 2.8M13.2 18.5l4.2-2.8M13.2 18.5L9 21.3M13.2 18.5l4.2 2.8M22.6 21.3l-4.2-2.8"
                stroke="var(--accent-1)"
                strokeWidth="1.7"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
          <span className="font-semibold tracking-tight">cloud-storage</span>
        </div>
      </header>
      <main className="max-w-5xl mx-auto px-6 py-8">{children}</main>
    </div>
  );
}

function SharePageSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-8 w-1/3" />
      <Skeleton className="h-5 w-1/4" />
      <div className="mt-6 space-y-2">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    </div>
  );
}

interface SharedViewProps {
  token: string;
  share: import("@/api/types").PublicShareResponse;
}

function SharedView({ token, share }: SharedViewProps) {
  const [currentId, setCurrentId] = useState<string>(share.node_id);
  const [stack, setStack] = useState<{ id: string; name: string }[]>([]);

  const isFile = share.kind === "file";

  if (isFile) {
    return <FileView token={token} share={share} />;
  }

  return (
    <div className="space-y-4">
      <div>
        <div className="text-fg-2 text-sm mb-1">Поделились с вами</div>
        <h1 className="text-2xl font-semibold flex items-center gap-2 truncate">
          <Folder className="h-6 w-6 text-accent-1" />
          {share.name}
        </h1>
        <div className="text-fg-3 text-[13px] mt-1">
          Доступ: {share.permission === "edit" ? "редактирование" : "только просмотр"}
          {share.expires_at && ` · до ${new Date(share.expires_at).toLocaleString("ru-RU")}`}
        </div>
      </div>
      <FolderListing
        token={token}
        currentId={currentId}
        canGoBack={stack.length > 0}
        onBack={() => {
          setStack((s) => {
            const copy = [...s];
            copy.pop();
            return copy;
          });
          const prev = stack[stack.length - 1];
          if (prev) setCurrentId(prev.id);
          else setCurrentId(share.node_id);
        }}
        onNavigate={(node) => {
          setStack((s) => [...s, { id: currentId, name: stack.length === 0 ? share.name : node.name }]);
          setCurrentId(node.id);
        }}
      />
    </div>
  );
}

function FileView({
  token,
  share,
}: {
  token: string;
  share: import("@/api/types").PublicShareResponse;
}) {
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [isLoading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function fetchUrl(disposition: "attachment" | "inline") {
    setLoading(true);
    setErr(null);
    try {
      const resp = await getShareDownloadURL(token, share.node_id, disposition);
      setDownloadUrl(resp.url);
      const link = document.createElement("a");
      link.href = resp.url;
      link.rel = "noopener";
      if (disposition === "inline") link.target = "_blank";
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : "Не удалось получить ссылку");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="rounded-xl border border-border-1 bg-bg-1 p-8 text-center">
      <FileText className="h-12 w-12 mx-auto text-fg-2 mb-3" />
      <div className="text-xl font-semibold mb-1 break-all">{share.name}</div>
      <div className="text-fg-3 text-sm mb-6">
        Доступ: {share.permission === "edit" ? "редактирование" : "только просмотр"}
      </div>
      <div className="flex items-center justify-center gap-2">
        <Button onClick={() => fetchUrl("attachment")} disabled={isLoading}>
          <Download className="h-4 w-4" />
          Скачать
        </Button>
        <Button intent="secondary" onClick={() => fetchUrl("inline")} disabled={isLoading}>
          <ExternalLink className="h-4 w-4" />
          Открыть
        </Button>
      </div>
      {err && <div className="text-danger text-sm mt-3">{err}</div>}
      {downloadUrl && !err && (
        <div className="text-fg-3 text-[12px] mt-3">
          Если ничего не открылось, проверьте блокировщик всплывающих окон.
        </div>
      )}
    </div>
  );
}

interface FolderListingProps {
  token: string;
  currentId: string;
  canGoBack: boolean;
  onBack: () => void;
  onNavigate: (node: NodeResponse) => void;
}

function FolderListing({ token, currentId, canGoBack, onBack, onNavigate }: FolderListingProps) {
  const children = useQuery({
    queryKey: qk.publicShareChildren(token, currentId),
    queryFn: () => listShareChildren(token, { parentId: currentId, limit: 200 }),
  });

  return (
    <div className="rounded-xl border border-border-1 bg-bg-1 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2 border-b border-border-1">
        <div className="flex items-center gap-2">
          {canGoBack && (
            <Button intent="ghost" size="sm" onClick={onBack}>
              <ArrowLeft className="h-4 w-4" />
              Назад
            </Button>
          )}
          <span className="text-fg-2 text-[13px]">
            {children.data?.items.length ?? 0} {plural(children.data?.items.length ?? 0, ["элемент", "элемента", "элементов"])}
          </span>
        </div>
      </div>
      {children.isLoading ? (
        <div className="p-4 space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : children.isError ? (
        <div className="p-8 text-fg-2 text-center">Не удалось загрузить содержимое.</div>
      ) : children.data?.items.length === 0 ? (
        <div className="p-8 text-fg-2 text-center">Папка пуста.</div>
      ) : (
        <ul className="divide-y divide-border-1">
          {children.data?.items.map((node) => (
            <SharedRow
              key={node.id}
              token={token}
              node={node}
              onClick={() => {
                if (node.kind === "folder") onNavigate(node);
              }}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function SharedRow({
  token,
  node,
  onClick,
}: {
  token: string;
  node: NodeResponse;
  onClick: () => void;
}) {
  const Icon = iconForNode({ kind: node.kind, mime: node.mime_type ?? null, name: node.name });
  const isFolder = node.kind === "folder";

  async function downloadFile() {
    try {
      const resp = await getShareDownloadURL(token, node.id, "attachment");
      const link = document.createElement("a");
      link.href = resp.url;
      link.rel = "noopener";
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch {
      /* swallow — toast not configured on public share page */
    }
  }

  return (
    <li>
      <div className="flex items-center gap-3 px-4 h-12 hover:bg-bg-2 transition-colors">
        <button
          type="button"
          onClick={onClick}
          disabled={!isFolder}
          className={cn(
            "flex flex-1 items-center gap-3 min-w-0 text-left",
            isFolder ? "cursor-pointer" : "cursor-default",
          )}
        >
          <Icon className={cn("h-5 w-5 shrink-0", isFolder ? "text-accent-1" : "text-fg-2")} />
          <span className="truncate">{node.name}</span>
        </button>
        <span className="text-fg-3 text-[12px] hidden md:inline w-24 text-right">
          {isFolder ? "—" : formatBytes(node.size_bytes)}
        </span>
        <span className="text-fg-3 text-[12px] hidden md:inline w-32 text-right">
          {formatRelativeTime(node.updated_at)}
        </span>
        {!isFolder && (
          <Button intent="ghost" size="icon" aria-label="Скачать" onClick={downloadFile}>
            <Download className="h-4 w-4" />
          </Button>
        )}
      </div>
    </li>
  );
}

function plural(n: number, forms: [string, string, string]): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod10 === 1 && mod100 !== 11) return forms[0];
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return forms[1];
  return forms[2];
}

