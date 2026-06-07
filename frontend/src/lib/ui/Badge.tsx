import { type ReactNode } from "react";
import { cn } from "./cn";

export type BadgeTone = "neutral" | "info" | "success" | "warning" | "danger" | "brand";
export type BadgeVariant = "soft" | "solid" | "outline";

const toneClasses: Record<BadgeVariant, Record<BadgeTone, string>> = {
  // Soft (default) — translucent background + colored text.
  soft: {
    neutral: "bg-surface-2 text-ink-2",
    info:    "bg-info/10 text-info",
    success: "bg-success/10 text-success",
    warning: "bg-warning/10 text-warning",
    danger:  "bg-danger/10 text-danger",
    brand:   "bg-brand-600/10 text-brand-600 dark:text-brand-300",
  },
  // Solid — fully filled.
  solid: {
    neutral: "bg-ink-3 text-surface-0",
    info:    "bg-info text-white",
    success: "bg-success text-white",
    warning: "bg-warning text-ink-1",
    danger:  "bg-danger text-white",
    brand:   "bg-brand-600 text-white",
  },
  // Outline — border only.
  outline: {
    neutral: "border border-surface-border text-ink-2",
    info:    "border border-info/40 text-info",
    success: "border border-success/40 text-success",
    warning: "border border-warning/40 text-warning",
    danger:  "border border-danger/40 text-danger",
    brand:   "border border-brand-600/40 text-brand-600 dark:text-brand-300",
  },
};

export function Badge({
  tone = "neutral",
  variant = "soft",
  children,
  className,
  leftIcon,
}: {
  tone?: BadgeTone;
  variant?: BadgeVariant;
  children: ReactNode;
  className?: string;
  leftIcon?: ReactNode;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
        toneClasses[variant][tone],
        className,
      )}
    >
      {leftIcon}
      {children}
    </span>
  );
}
