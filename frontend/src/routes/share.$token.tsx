import { createFileRoute } from "@tanstack/react-router";
import { SharePage } from "@/features/share/share.page";

export const Route = createFileRoute("/share/$token")({
  component: SharePage,
});
