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
 * v0.3.1 Phase G: Enhanced with comprehensive error handling.
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

// --- Phase G: Enhanced Error Handling Components ---

/** Network error state with specific messaging */
export function NetworkError({
  onRetry,
  message = "Unable to connect to the server. Please check your connection.",
}: {
  onRetry?: () => void;
  message?: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-warning/30 bg-warning/5 px-6 py-8 text-center">
      <div className="mb-3 text-warning">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="h-10 w-10">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <h3 className="text-sm font-semibold text-ink-1">Connection Error</h3>
      <p className="mt-1 max-w-md text-sm text-ink-3">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-4 inline-flex h-8 items-center gap-2 rounded-md border border-surface-border bg-surface-1 px-4 text-xs font-medium text-ink-1 hover:bg-surface-2"
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-3.5 w-3.5">
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Try Again
        </button>
      )}
    </div>
  );
}

/** Authentication error state */
export function AuthError({
  message = "Your session has expired. Please log in again.",
}: {
  message?: string;
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-lg border border-danger/30 bg-danger/5 px-6 py-8 text-center">
      <div className="mb-3 text-danger">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="h-10 w-10">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
      </div>
      <h3 className="text-sm font-semibold text-ink-1">Authentication Required</h3>
      <p className="mt-1 max-w-md text-sm text-ink-3">{message}</p>
      <button
        type="button"
        onClick={() => { window.location.href = "/login"; }}
        className="mt-4 inline-flex h-8 items-center gap-2 rounded-md bg-brand-600 px-4 text-xs font-medium text-white hover:bg-brand-700"
      >
        Log In
      </button>
    </div>
  );
}

/** Inline error message for forms */
export function InlineError({
  message,
  className,
}: {
  message: string;
  className?: string;
}) {
  return (
    <div className={cn("flex items-center gap-1.5 text-xs text-danger", className)}>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-3.5 w-3.5 shrink-0">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <span>{message}</span>
    </div>
  );
}

/** Form submission error with summary */
export function FormError({
  title = "Please correct the errors below",
  errors,
  className,
}: {
  title?: string;
  errors?: string[];
  className?: string;
}) {
  if (!errors || errors.length === 0) return null;

  return (
    <div className={cn("rounded-md border border-danger/30 bg-danger/5 p-4", className)}>
      <div className="flex items-start gap-3">
        <div className="mt-0.5 text-danger">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <div className="flex-1">
          <h4 className="text-sm font-medium text-danger">{title}</h4>
          <ul className="mt-2 space-y-1">
            {errors.map((error, i) => (
              <li key={i} className="text-xs text-ink-2">
                • {error}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}

/** Warning banner for non-critical issues */
export function WarningBanner({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex items-start gap-3 rounded-md border border-warning/30 bg-warning/5 p-4">
      <div className="mt-0.5 text-warning">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <div className="flex-1">
        <h4 className="text-sm font-medium text-ink-1">{title}</h4>
        {description && <p className="mt-1 text-xs text-ink-3">{description}</p>}
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}

/** Toast-style error for transient errors */
export function ToastError({
  message,
  onDismiss,
}: {
  message: string;
  onDismiss?: () => void;
}) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-danger/30 bg-danger/5 p-4 shadow-pop">
      <div className="text-danger">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
          <circle cx="12" cy="12" r="10" />
          <line x1="15" y1="9" x2="9" y2="15" />
          <line x1="9" y1="9" x2="15" y2="15" />
        </svg>
      </div>
      <div className="flex-1 text-sm text-ink-1">{message}</div>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="text-ink-3 hover:text-ink-2"
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-4 w-4">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      )}
    </div>
  );
}
