import { type ReactNode } from "react";
import { cn } from "./cn";

export type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";

const toneClasses: Record<StatusTone, string> = {
  neutral: "bg-surface-2 text-ink-2 border-surface-border",
  success: "bg-success/10 text-success border-success/30",
  warning: "bg-warning/10 text-warning border-warning/30",
  danger:  "bg-danger/10 text-danger border-danger/30",
  info:    "bg-info/10 text-info border-info/30",
};

export function StatusPill({
  tone = "neutral",
  children,
  dot = true,
  className,
  title,
}: {
  tone?: StatusTone;
  children: ReactNode;
  dot?: boolean;
  className?: string;
  title?: string;
}) {
  return (
    <span
      title={title}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium",
        toneClasses[tone],
        className,
      )}
    >
      {dot && (
        <span
          className={cn(
            "h-1.5 w-1.5 rounded-full",
            tone === "success" && "bg-success",
            tone === "warning" && "bg-warning",
            tone === "danger"  && "bg-danger",
            tone === "info"    && "bg-info",
            tone === "neutral" && "bg-ink-3",
          )}
        />
      )}
      {children}
    </span>
  );
}
