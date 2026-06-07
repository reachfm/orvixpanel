import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";
import { cn } from "./cn";

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  hint?: string;
  error?: string | null;
  leftAddon?: ReactNode;
  rightAddon?: ReactNode;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, hint, error, leftAddon, rightAddon, className, id, ...rest },
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
        {leftAddon && <div className="pl-3 text-ink-3">{leftAddon}</div>}
        <input
          ref={ref}
          id={inputId}
          className={cn(
            "flex-1 bg-transparent px-3 py-2 text-sm text-ink-1 placeholder:text-ink-4",
            "focus:outline-none disabled:opacity-60",
            className,
          )}
          {...rest}
        />
        {rightAddon && <div className="pr-3 text-ink-3">{rightAddon}</div>}
      </div>
      {error ? (
        <p className="text-xs text-danger">{error}</p>
      ) : hint ? (
        <p className="text-xs text-ink-3">{hint}</p>
      ) : null}
    </div>
  );
});
