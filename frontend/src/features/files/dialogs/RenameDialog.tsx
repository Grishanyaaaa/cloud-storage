import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { z } from "zod";
import { Loader2 } from "lucide-react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { NodeResponse } from "@/api/types";
import { useRenameNode } from "../useFileMutations";

const schema = z.object({
  name: z
    .string()
    .min(1, "Имя не может быть пустым")
    .max(255, "Слишком длинное имя")
    .refine((v) => !/[\\/\u0000]/.test(v), "Имя содержит запрещённые символы"),
});
type Form = z.infer<typeof schema>;

interface Props {
  node: NodeResponse;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RenameDialog({ node, open, onOpenChange }: Props) {
  const mutation = useRenameNode();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { name: node.name },
  });

  useEffect(() => {
    if (open) reset({ name: node.name });
  }, [open, node.name, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Переименовать</DialogTitle>
          <DialogDescription>
            {node.kind === "folder" ? "Папка" : "Файл"} «{node.name}»
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={handleSubmit((v) =>
            mutation.mutate(
              { id: node.id, name: v.name.trim() },
              {
                onSuccess: () => onOpenChange(false),
              },
            ),
          )}
          className="flex flex-col gap-4"
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="rename-name">Новое имя</Label>
            <Input
              id="rename-name"
              autoFocus
              aria-invalid={Boolean(errors.name)}
              {...register("name")}
            />
            {errors.name && (
              <span className="text-[12px] text-danger">{errors.name.message}</span>
            )}
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" intent="secondary">
                Отмена
              </Button>
            </DialogClose>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Переименование…
                </>
              ) : (
                "Сохранить"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
