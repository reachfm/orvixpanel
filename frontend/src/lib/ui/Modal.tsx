/**
 * Modal — a basic controlled dialog. Native <dialog> would be nicer
 * but its fullscreen / focus-restoration behavior across browsers
 * is still uneven in 2026; a div with role="dialog" is predictable
 * for the enterprise use case.
 */

import { useEffect, type ReactNode } from "react";
import { cn } from "./cn";

export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  width = "md",
}: {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  width?: "sm" | "md" | "lg" | "xl";
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [open, onClose]);

  if (!open) return null;
  const w =
    width === "sm" ? "max-w-sm" :
    width === "md" ? "max-w-md" :
    width === "lg" ? "max-w-lg" :
                     "max-w-2xl";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink-1/40 backdrop-blur-sm"
      onClick={onClose}
      role="presentation"
    >
      <div
        role="dialog"
        aria-modal="true"
        onClick={(e) => e.stopPropagation()}
        className={cn(
          "w-full rounded-lg border border-surface-border bg-surface-1 shadow-pop",
          w,
        )}
      >
        <div className="px-5 pt-4">
          <h2 className="text-base font-semibold text-ink-1">{title}</h2>
          {description && <p className="mt-1 text-xs text-ink-3">{description}</p>}
        </div>
        <div className="px-5 py-4">{children}</div>
        {footer && (
          <div className="flex items-center justify-end gap-2 border-t border-surface-border bg-surface-2 px-5 py-3">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
