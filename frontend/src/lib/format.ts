/**
 * Human-friendly formatters in ru-RU locale.
 */

const KB = 1024;
const MB = KB * 1024;
const GB = MB * 1024;
const TB = GB * 1024;

export function formatBytes(bytes: number | null | undefined): string {
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < KB) return `${bytes} Б`;
  if (bytes < MB) return `${(bytes / KB).toFixed(1)} КБ`;
  if (bytes < GB) return `${(bytes / MB).toFixed(1)} МБ`;
  if (bytes < TB) return `${(bytes / GB).toFixed(2)} ГБ`;
  return `${(bytes / TB).toFixed(2)} ТБ`;
}

const dateFormatter = new Intl.DateTimeFormat("ru-RU", {
  day: "2-digit",
  month: "short",
  year: "numeric",
});

const dateTimeFormatter = new Intl.DateTimeFormat("ru-RU", {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

export function formatDate(input: string | Date | null | undefined): string {
  if (!input) return "—";
  const d = typeof input === "string" ? new Date(input) : input;
  if (Number.isNaN(d.getTime())) return "—";
  return dateFormatter.format(d);
}

export function formatDateTime(input: string | Date | null | undefined): string {
  if (!input) return "—";
  const d = typeof input === "string" ? new Date(input) : input;
  if (Number.isNaN(d.getTime())) return "—";
  return dateTimeFormatter.format(d);
}

const RTF = new Intl.RelativeTimeFormat("ru-RU", { numeric: "auto" });

export function formatRelativeTime(input: string | Date | null | undefined): string {
  if (!input) return "—";
  const d = typeof input === "string" ? new Date(input) : input;
  if (Number.isNaN(d.getTime())) return "—";

  const diffMs = d.getTime() - Date.now();
  const abs = Math.abs(diffMs);

  const minute = 60_000;
  const hour = 60 * minute;
  const day = 24 * hour;
  const week = 7 * day;
  const month = 30 * day;
  const year = 365 * day;

  const sign = diffMs < 0 ? -1 : 1;
  if (abs < minute) return RTF.format(sign * Math.round(abs / 1000), "second");
  if (abs < hour) return RTF.format(sign * Math.round(abs / minute), "minute");
  if (abs < day) return RTF.format(sign * Math.round(abs / hour), "hour");
  if (abs < week) return RTF.format(sign * Math.round(abs / day), "day");
  if (abs < month) return RTF.format(sign * Math.round(abs / week), "week");
  if (abs < year) return RTF.format(sign * Math.round(abs / month), "month");
  return RTF.format(sign * Math.round(abs / year), "year");
}

/**
 * The storage-service returns share URLs pointing at the API gateway (e.g.
 * `https://api.cloud-storage.local/storage/v1/public/<token>`). The frontend
 * substitutes that with its own `/share/<token>` URL when the user copies a
 * share link, so it lands on the SPA rather than on the JSON API.
 */
export function rewriteShareUrl(serverUrl: string, token: string, frontendBase: string): string {
  if (!serverUrl || !token) return serverUrl;
  return `${frontendBase.replace(/\/+$/, "")}/share/${token}`;
}
