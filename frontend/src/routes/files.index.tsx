import { createFileRoute } from "@tanstack/react-router";
import { FilesIndexPage } from "@/features/files/files.index.page";

export const Route = createFileRoute("/files/")({
  component: FilesIndexPage,
});
