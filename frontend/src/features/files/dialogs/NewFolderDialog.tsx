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
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCreateFolder } from "../useFileMutations";

const schema = z.object({
  name: z
    .string()
    .min(1, "Имя не может быть пустым")
    .max(255, "Слишком длинное имя")
    .refine((v) => !v.includes("/") && !v.includes("\\") && !v.includes("\u0000"), "Имя содержит запрещённые символы"),
});
type Form = z.infer<typeof schema>;

interface Props {
  parentId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function NewFolderDialog({ parentId, open, onOpenChange }: Props) {
  const mutation = useCreateFolder();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  });

  useEffect(() => {
    if (open) reset({ name: "" });
  }, [open, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Новая папка</DialogTitle>
        </DialogHeader>
        <form
          onSubmit={handleSubmit((v) =>
            mutation.mutate(
              { parent_id: parentId, name: v.name.trim() },
              {
                onSuccess: () => onOpenChange(false),
              },
            ),
          )}
          className="flex flex-col gap-4"
        >
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="new-folder-name">Имя папки</Label>
            <Input
              id="new-folder-name"
              autoFocus
              placeholder="Например, «Документы»"
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
                  Создание…
                </>
              ) : (
                "Создать"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
