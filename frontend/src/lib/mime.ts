import {
  File,
  FileArchive,
  FileAudio,
  FileCode,
  FileImage,
  FileSpreadsheet,
  FileText,
  FileVideo,
  Folder,
  type LucideIcon,
} from "lucide-react";

/**
 * Map (kind, mime, name) to a lucide icon. Used in the file table and AI plan
 * preview rows.
 */
export function iconForNode(args: {
  kind: "file" | "folder";
  mime?: string | null | undefined;
  name?: string | null | undefined;
}): LucideIcon {
  if (args.kind === "folder") return Folder;
  const mime = (args.mime ?? "").toLowerCase();
  const name = (args.name ?? "").toLowerCase();

  if (mime.startsWith("image/")) return FileImage;
  if (mime.startsWith("audio/")) return FileAudio;
  if (mime.startsWith("video/")) return FileVideo;
  if (mime === "application/pdf" || name.endsWith(".pdf")) return FileText;
  if (
    mime === "application/zip" ||
    mime === "application/x-7z-compressed" ||
    mime === "application/x-rar-compressed" ||
    mime === "application/x-tar" ||
    mime === "application/gzip" ||
    /\.(zip|rar|7z|tar|gz|tgz|bz2)$/.test(name)
  ) {
    return FileArchive;
  }
  if (
    mime.includes("spreadsheet") ||
    mime === "text/csv" ||
    /\.(csv|xlsx?|ods)$/.test(name)
  ) {
    return FileSpreadsheet;
  }
  if (
    mime.startsWith("text/") ||
    mime === "application/json" ||
    mime === "application/xml" ||
    /\.(json|xml|html|css|js|ts|tsx|jsx|md|yml|yaml|toml|sh|py|go|java|c|cpp|rs)$/.test(name)
  ) {
    return FileCode;
  }
  if (
    mime === "application/msword" ||
    mime === "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
    /\.(docx?|odt|rtf|txt)$/.test(name)
  ) {
    return FileText;
  }
  return File;
}
