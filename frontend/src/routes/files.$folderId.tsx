import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { FilesPage } from "@/features/files/files.page";

const filesSearchSchema = z.object({
  preview: z.string().optional(),
  include_deleted: z.boolean().optional(),
});

export const Route = createFileRoute("/files/$folderId")({
  validateSearch: filesSearchSchema,
  component: FilesPage,
});
