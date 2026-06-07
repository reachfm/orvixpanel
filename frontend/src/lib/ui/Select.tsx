import { forwardRef, type SelectHTMLAttributes, type ReactNode } from "react";
import { cn } from "./cn";

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  hint?: string;
  error?: string | null;
  children: ReactNode;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { label, hint, error, className, id, children, ...rest },
  ref,
) {
  const inputId = id ?? rest.name;
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <label htmlFor={inputId} className="text-xs font-medium text-ink-2">
          {label}
        </label>
      )}
      <div
        className={cn(
          "flex items-center rounded-md border bg-surface-1",
          error
            ? "border-danger/70 focus-within:border-danger"
            : "border-surface-border focus-within:border-brand-500",
          "transition-colors",
        )}
      >
        <select
          ref={ref}
          id={inputId}
          className={cn(
            "flex-1 appearance-none bg-transparent px-3 py-2 text-sm text-ink-1",
            "focus:outline-none disabled:opacity-60",
            className,
          )}
          {...rest}
        >
          {children}
        </select>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          className="mr-2 h-3.5 w-3.5 text-ink-3"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </div>
      {error ? (
        <p className="text-xs text-danger">{error}</p>
      ) : hint ? (
        <p className="text-xs text-ink-3">{hint}</p>
      ) : null}
    </div>
  );
});
