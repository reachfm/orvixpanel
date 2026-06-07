/**
 * Loading / Empty / Error primitives.
 *
 * The UI library exposes three distinct states so pages have a
 * consistent shape regardless of where the data is coming from.
 *
 *   - LoadingState — full-page or inline loading shell.
 *   - EmptyState   — "0 records" with an optional CTA. NEVER shows
 *                    placeholder metrics or fake counts.
 *   - ErrorState   — "request failed" with an optional Retry.
 */

import { type ReactNode } from "react";
import { cn } from "./cn";

export function Spinner({ size = 20 }: { size?: number }) {
  return (
    <span
      className="inline-block animate-spin rounded-full border-2 border-brand-500 border-r-transparent"
      style={{ width: size, height: size }}
      aria-label="Loading"
    />
  );
}

/** Full-page or section loading shell. */
export function LoadingState({
  message = "Loading…",
  full = true,
}: {
  message?: ReactNode;
  full?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-center text-sm text-ink-3",
        full ? "min-h-[40vh]" : "py-12",
      )}
      role="status"
    >
      <Spinner size={20} />
      <span className="ml-2">{message}</span>
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
  icon,
}: {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-12 text-center">
      {icon && <div className="mb-3 text-ink-3">{icon}</div>}
      <h3 className="text-sm font-semibold text-ink-1">{title}</h3>
      {description && <p className="mt-1 max-w-md text-sm text-ink-3">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function ErrorState({
  title = "Something went wrong",
  description,
  onRetry,
}: {
  title?: ReactNode;
  description?: ReactNode;
  onRetry?: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-danger/30 bg-danger/5 px-6 py-10 text-center">
      <div className="mb-2 grid h-8 w-8 place-items-center rounded-full bg-danger/10 text-danger font-bold">!</div>
      <h3 className="text-sm font-semibold text-ink-1">{title}</h3>
      {description && <p className="mt-1 max-w-md text-sm text-ink-3">{description}</p>}
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-4 inline-flex h-8 items-center rounded-md border border-surface-border bg-surface-1 px-3 text-xs font-medium text-ink-1 hover:bg-surface-2"
        >
          Retry
        </button>
      )}
    </div>
  );
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn("animate-pulse rounded bg-surface-3", className)} />;
}
