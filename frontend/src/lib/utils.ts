/**
 * Utility functions — safe helpers for undefined/null handling
 * No raw .toUpperCase() on potentially undefined values.
 */

export function safeUpper(value: string | undefined | null, fallback: string = ""): string {
  if (!value) return fallback;
  return value.toUpperCase();
}

export function formatNumber(value: number | undefined | null, fallback: string = "0"): string {
  if (value == null || isNaN(value)) return fallback;
  return value.toLocaleString();
}

export function formatBytes(bytes: number | undefined | null): string {
  if (bytes == null || isNaN(bytes)) return "0 B";
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

export function formatMB(mb: number | undefined | null): string {
  if (mb == null || isNaN(mb)) return "0 MB";
  if (mb >= 1024) {
    return (mb / 1024).toFixed(1) + " GB";
  }
  return mb + " MB";
}

export function formatDate(dateStr: string | undefined | null): string {
  if (!dateStr) return "—";
  try {
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return "—";
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "—";
  }
}

export function formatDateShort(dateStr: string | undefined | null): string {
  if (!dateStr) return "—";
  try {
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) return "—";
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  } catch {
    return "—";
  }
}

export function formatPercent(value: number | undefined | null, total: number | undefined | null): string {
  if (value == null || total == null || total === 0) return "0%";
  return Math.round((value / total) * 100) + "%";
}

export function truncate(str: string | undefined | null, maxLength: number = 50): string {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength) + "…";
}

export function safeString(value: unknown, fallback: string = ""): string {
  if (value == null) return fallback;
  if (typeof value === "string") return value;
  return String(value);
}

export function pluralize(count: number, singular: string, plural?: string): string {
  return count === 1 ? singular : (plural ?? singular + "s");
}

export function timeAgo(timestamp: string | undefined | null): string {
  if (!timestamp) return "—";
  try {
    const now = Date.now();
    const then = new Date(timestamp).getTime();
    const diffMs = now - then;
    const diffMins = Math.floor(diffMs / (1000 * 60));
    if (diffMins < 1) return "just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays}d ago`;
    return formatDateShort(timestamp);
  } catch {
    return "—";
  }
}

export function daysUntil(timestamp: string | undefined | null): number | null {
  if (!timestamp) return null;
  try {
    const target = new Date(timestamp).getTime();
    const now = Date.now();
    return Math.ceil((target - now) / (1000 * 60 * 60 * 24));
  } catch {
    return null;
  }
}

export function classNames(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(" ");
}