import { ChevronDown, ChevronUp, X } from "lucide-react";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { useUploadsStore, type UploadEntry, type UploadStatus } from "@/store/uploads.store";
import { cn } from "@/lib/cn";
import { formatBytes } from "@/lib/format";

const STATUS_LABELS: Record<UploadStatus, string> = {
  queued: "В очереди",
  hashing: "Хэширование",
  issuing: "Запрос ссылки",
  uploading: "Загрузка",
  finalizing: "Финализация",
  done: "Готово",
  error: "Ошибка",
  cancelled: "Отменено",
};

export function UploadPanel() {
  const entries = useUploadsStore((s) => s.entries);
  const open = useUploadsStore((s) => s.panelOpen);
  const setOpen = useUploadsStore((s) => s.setPanelOpen);
  const remove = useUploadsStore((s) => s.remove);
  const clearCompleted = useUploadsStore((s) => s.clearCompleted);

  if (entries.length === 0) return null;
  const inFlight = entries.filter((e) =>
    ["queued", "hashing", "issuing", "uploading", "finalizing"].includes(e.status),
  ).length;

  return (
    <div className="fixed bottom-4 right-4 w-[360px] max-w-[calc(100vw-2rem)] z-40 rounded-lg border border-border-1 bg-bg-1 shadow-lg">
      <header className="flex items-center justify-between gap-2 px-3 h-10 border-b border-border-1">
        <div className="text-sm font-medium">
          Загрузки {inFlight > 0 && <span className="text-fg-3">· {inFlight} в процессе</span>}
        </div>
        <div className="flex items-center gap-1">
          <Button
            intent="ghost"
            size="icon"
            aria-label={open ? "Свернуть" : "Развернуть"}
            onClick={() => setOpen(!open)}
          >
            {open ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />}
          </Button>
          <Button
            intent="ghost"
            size="icon"
            aria-label="Закрыть"
            onClick={clearCompleted}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </header>
      {open && (
        <ul className="max-h-[40vh] overflow-auto divide-y divide-border-1">
          {entries.map((entry) => (
            <li key={entry.id} className="px-3 py-2.5">
              <UploadRow entry={entry} onRemove={() => remove(entry.id)} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function UploadRow({ entry, onRemove }: { entry: UploadEntry; onRemove: () => void }) {
  const isTerminal = entry.status === "done" || entry.status === "error" || entry.status === "cancelled";
  const inProgress =
    entry.status === "uploading" || entry.status === "hashing" || entry.status === "issuing" || entry.status === "finalizing";
  const pct = entry.size > 0 ? Math.min(100, Math.round((entry.uploadedBytes / entry.size) * 100)) : 0;

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="text-sm text-fg-1 truncate" title={entry.name}>
            {entry.name}
          </div>
          <div className="text-[12px] text-fg-3">
            {formatBytes(entry.size)} · {STATUS_LABELS[entry.status]}
            {entry.status === "error" && entry.error ? ` · ${entry.error}` : ""}
          </div>
        </div>
        {!isTerminal && entry.abort && (
          <Button
            intent="ghost"
            size="icon"
            aria-label="Отменить"
            onClick={() => entry.abort?.()}
          >
            <X className="h-4 w-4" />
          </Button>
        )}
        {isTerminal && (
          <Button intent="ghost" size="icon" aria-label="Убрать из списка" onClick={onRemove}>
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>
      {inProgress ? (
        entry.status === "uploading" ? (
          <Progress value={pct} />
        ) : (
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-bg-4">
            <div className="h-full w-1/3 animate-pulse bg-accent-1" />
          </div>
        )
      ) : null}
      <div
        className={cn(
          "text-[11px] tabular-nums",
          entry.status === "error" ? "text-danger" : "text-fg-3",
        )}
      >
        {entry.status === "uploading" && `${pct}% · ${formatBytes(entry.uploadedBytes)} / ${formatBytes(entry.size)}`}
      </div>
    </div>
  );
}
