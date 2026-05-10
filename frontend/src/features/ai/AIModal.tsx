import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { ArrowRight, Loader2, MoveRight, Pencil, Sparkles, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { cancelCommand, executeCommand, planCommand } from "@/api/ai";
import { ApiError } from "@/api/types";
import type { AICommandResponse, AIOperation, TreeNodeResponse } from "@/api/types";
import { useAIModalStore } from "@/store/ai.store";
import { useTree } from "../files/useFilesData";

const SAMPLE_PROMPTS = [
  "Удали отчёт за май",
  "Переименуй фото-2024 в путешествие-крым",
  "Перемести все файлы из «inbox» в «архив»",
];

export function AIModal() {
  const isOpen = useAIModalStore((s) => s.isOpen);
  const close = useAIModalStore((s) => s.close);
  const reset = useAIModalStore((s) => s.reset);
  const state = useAIModalStore((s) => s.state);
  const setState = useAIModalStore((s) => s.setState);
  const queryClient = useQueryClient();

  const tree = useTree();
  const [input, setInput] = useState("");

  // Reset on close so the user always lands on a fresh prompt.
  useEffect(() => {
    if (!isOpen) {
      const t = setTimeout(() => {
        setInput("");
        reset();
      }, 200);
      return () => clearTimeout(t);
    }
    return undefined;
  }, [isOpen, reset]);

  const plan = useMutation({
    mutationFn: (text: string) => planCommand({ input: text }),
    onMutate: (text) => setState({ phase: "planning", input: text }),
    onSuccess: (cmd) => {
      if (cmd.status === "awaiting_confirmation") {
        setState({ phase: "plan-ready", cmd });
      } else if (cmd.status === "executed") {
        setState({ phase: "done", cmd });
      } else {
        setState({
          phase: "error",
          message: `Неожиданный статус плана: ${cmd.status}`,
        });
      }
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : "ИИ не смог построить план";
      setState({ phase: "error", message: msg });
    },
  });

  const exec = useMutation({
    mutationFn: (id: string) => executeCommand(id),
    onMutate: (id) => setState({ phase: "executing", id }),
    onSuccess: (cmd) => {
      setState({ phase: "done", cmd });
      // Coarse cache invalidation — many ops touched the tree.
      void queryClient.invalidateQueries({ queryKey: ["tree"], exact: false });
      void queryClient.invalidateQueries({ queryKey: ["children"], exact: false });
      toast.success("Операции выполнены");
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : "Не удалось выполнить операции";
      setState({ phase: "error", message: msg });
    },
  });

  const cancel = useMutation({
    mutationFn: (id: string) => cancelCommand(id),
    onSuccess: () => {
      toast("План отменён");
      reset();
    },
  });

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const text = input.trim();
    if (text.length === 0) return;
    plan.mutate(text);
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => (open ? null : close())}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-accent-1" />
            ИИ-помощник
          </DialogTitle>
          <DialogDescription>
            Опишите, что нужно сделать. ИИ построит план — каждое действие
            требует вашего подтверждения.
          </DialogDescription>
        </DialogHeader>

        {state.phase === "idle" && (
          <form onSubmit={onSubmit} className="flex flex-col gap-3">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              autoFocus
              rows={3}
              maxLength={2000}
              placeholder="Например: «удали все файлы старше года из папки бэкапы»"
            />
            <div className="flex flex-wrap gap-1.5">
              {SAMPLE_PROMPTS.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setInput(p)}
                  className="rounded-full border border-border-1 bg-bg-2 px-3 h-7 text-[12px] text-fg-2 hover:bg-bg-3 hover:text-fg-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-1 focus-visible:ring-offset-2 focus-visible:ring-offset-bg-3"
                >
                  {p}
                </button>
              ))}
            </div>
            <DialogFooter>
              <Button intent="secondary" type="button" onClick={close}>
                Отмена
              </Button>
              <Button type="submit" disabled={input.trim().length === 0}>
                <Sparkles className="h-4 w-4" />
                Построить план
              </Button>
            </DialogFooter>
          </form>
        )}

        {state.phase === "planning" && (
          <div className="flex flex-col items-center gap-3 py-8">
            <Loader2 className="h-8 w-8 animate-spin text-accent-1" />
            <div className="text-fg-2 text-sm">
              ИИ обдумывает: «{state.input}»…
            </div>
          </div>
        )}

        {state.phase === "plan-ready" && (
          <PlanView
            cmd={state.cmd}
            tree={tree.data}
            onApply={() => exec.mutate(state.cmd.id)}
            onCancel={() => cancel.mutate(state.cmd.id)}
            onClose={close}
            isApplying={exec.isPending}
            isCancelling={cancel.isPending}
          />
        )}

        {state.phase === "executing" && (
          <div className="flex flex-col items-center gap-3 py-8">
            <Loader2 className="h-8 w-8 animate-spin text-accent-1" />
            <div className="text-fg-2 text-sm">Выполняется…</div>
          </div>
        )}

        {state.phase === "done" && (
          <DoneView
            cmd={state.cmd}
            tree={tree.data}
            onClose={close}
            onAgain={() => {
              reset();
              setInput("");
            }}
          />
        )}

        {state.phase === "error" && (
          <div className="flex flex-col gap-3 py-2">
            <div className="rounded-md border border-danger/30 bg-danger-soft p-3 text-sm text-danger">
              {state.message}
            </div>
            <DialogFooter>
              <Button intent="secondary" onClick={close}>
                Закрыть
              </Button>
              <Button onClick={() => reset()}>Попробовать снова</Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

interface PlanViewProps {
  cmd: AICommandResponse;
  tree: TreeNodeResponse | undefined;
  onApply: () => void;
  onCancel: () => void;
  onClose: () => void;
  isApplying: boolean;
  isCancelling: boolean;
}

function PlanView({ cmd, tree, onApply, onCancel, onClose, isApplying, isCancelling }: PlanViewProps) {
  const hasOps = cmd.plan.length > 0;
  return (
    <div className="flex flex-col gap-3">
      <div className="text-fg-2 text-sm">
        <span className="text-fg-3">Запрос:</span> «{cmd.input}»
      </div>
      {cmd.explanation && (
        <div className="rounded-md border border-border-1 bg-bg-2 p-3 text-sm text-fg-1">
          {cmd.explanation}
        </div>
      )}
      {!hasOps ? (
        <div className="text-fg-2 text-sm border border-dashed border-border-1 rounded-md p-4 text-center">
          ИИ не нашёл подходящих действий. Попробуйте уточнить запрос.
        </div>
      ) : (
        <ul className="rounded-md border border-border-1 bg-bg-2 divide-y divide-border-1 max-h-72 overflow-auto">
          {cmd.plan.map((op, idx) => (
            <li key={idx} className="px-3 py-2.5 text-sm">
              <OpRow op={op} tree={tree} />
            </li>
          ))}
        </ul>
      )}
      <DialogFooter>
        <Button
          intent="secondary"
          type="button"
          onClick={onClose}
          disabled={isApplying || isCancelling}
        >
          Закрыть
        </Button>
        <Button
          intent="ghost"
          type="button"
          onClick={onCancel}
          disabled={isApplying || isCancelling}
        >
          {isCancelling ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
          Отменить план
        </Button>
        <Button
          type="button"
          onClick={onApply}
          disabled={!hasOps || isApplying || isCancelling}
        >
          {isApplying ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Выполнение…
            </>
          ) : (
            <>
              <ArrowRight className="h-4 w-4" />
              Применить
            </>
          )}
        </Button>
      </DialogFooter>
    </div>
  );
}

function OpRow({ op, tree }: { op: AIOperation; tree: TreeNodeResponse | undefined }) {
  const target = findInTree(tree, op.node_id);
  const targetName = target?.name ?? `id:${op.node_id.slice(0, 8)}…`;
  const newParent = op.new_parent_id ? findInTree(tree, op.new_parent_id) : null;

  if (op.kind === "delete") {
    return (
      <div className="flex items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-danger-soft text-danger">
          <Trash2 className="h-4 w-4" />
        </div>
        <div className="min-w-0 flex-1">
          <div>Удалить «{targetName}»</div>
          {target && (
            <div className="text-fg-3 text-[11px]">
              {target.kind === "folder" ? "Папка" : "Файл"} · {target.path}
            </div>
          )}
        </div>
      </div>
    );
  }
  if (op.kind === "rename") {
    return (
      <div className="flex items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-bg-3 text-fg-2">
          <Pencil className="h-4 w-4" />
        </div>
        <div className="min-w-0 flex-1">
          <div>
            Переименовать «{targetName}» → «{op.new_name ?? "?"}»
          </div>
        </div>
      </div>
    );
  }
  if (op.kind === "move") {
    return (
      <div className="flex items-center gap-3">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-bg-3 text-fg-2">
          <MoveRight className="h-4 w-4" />
        </div>
        <div className="min-w-0 flex-1">
          <div>
            Переместить «{targetName}» → «{newParent?.name ?? "?"}»
          </div>
        </div>
      </div>
    );
  }
  return <div className="text-fg-2">Неизвестная операция</div>;
}

function DoneView({
  cmd,
  tree,
  onClose,
  onAgain,
}: {
  cmd: AICommandResponse;
  tree: TreeNodeResponse | undefined;
  onClose: () => void;
  onAgain: () => void;
}) {
  const ok = cmd.results?.filter((r) => r.success).length ?? 0;
  const failed = cmd.results?.filter((r) => !r.success) ?? [];

  return (
    <div className="flex flex-col gap-3">
      <div className="text-sm">
        Выполнено: <span className="text-success font-medium">{ok}</span>
        {failed.length > 0 && (
          <>
            {" "}
            · Ошибок: <span className="text-danger font-medium">{failed.length}</span>
          </>
        )}
      </div>
      {failed.length > 0 && (
        <ul className="rounded-md border border-border-1 bg-bg-2 divide-y divide-border-1">
          {failed.map((r) => {
            const target = findInTree(tree, r.node_id);
            return (
              <li key={r.index} className="px-3 py-2 text-sm">
                <div className="text-fg-1">
                  {target?.name ?? `id:${r.node_id.slice(0, 8)}…`}
                </div>
                <div className="text-danger text-[12px]">
                  {r.error_message ?? r.error_code ?? "Ошибка"}
                </div>
              </li>
            );
          })}
        </ul>
      )}
      <DialogFooter>
        <Button intent="secondary" onClick={onClose}>
          Закрыть
        </Button>
        <Button onClick={onAgain}>Ещё команда</Button>
      </DialogFooter>
    </div>
  );
}

function findInTree(
  root: TreeNodeResponse | undefined,
  id: string,
): TreeNodeResponse | null {
  if (!root) return null;
  if (root.id === id) return root;
  for (const c of root.children ?? []) {
    const found = findInTree(c, id);
    if (found) return found;
  }
  return null;
}
