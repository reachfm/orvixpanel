/**
 * Dropdown — a controlled, click-outside-to-close popover.
 *
 * We avoid a 3rd-party headless lib for the v0.2.3 foundation. The
 * menu is keyboard-navigable (Up/Down/Enter/Escape) but uses the
 * native focus management; upgrade path is to swap in Radix /
 * @headlessui later without changing call sites.
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { cn } from "./cn";

export type DropdownItem =
  | {
      key: string;
      label: ReactNode;
      onClick: () => void;
      destructive?: boolean;
      disabled?: boolean;
      icon?: ReactNode;
    }
  | { key: string; type: "divider" };

export function Dropdown({
  trigger,
  items,
  align = "right",
}: {
  trigger: (open: boolean) => ReactNode;
  items: DropdownItem[];
  align?: "left" | "right";
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const btn = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (root.current && !root.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        btn.current?.focus();
      }
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div ref={root} className="relative inline-block">
      <button
        ref={btn}
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-brand-500/60"
      >
        {trigger(open)}
      </button>
      {open && (
        <div
          role="menu"
          className={cn(
            "absolute z-50 mt-1.5 min-w-[12rem] rounded-md border border-surface-border bg-surface-1 py-1 shadow-pop",
            align === "right" ? "right-0" : "left-0",
          )}
        >
          {items.map((it) => {
            if ("type" in it && it.type === "divider") {
              return <div key={it.key} className="my-1 border-t border-surface-border" />;
            }
            const item = it as Exclude<DropdownItem, { type: "divider" }>;
            return (
              <button
                key={item.key}
                type="button"
                role="menuitem"
                disabled={item.disabled}
                onClick={() => {
                  setOpen(false);
                  item.onClick();
                }}
                className={cn(
                  "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm",
                  "hover:bg-surface-2 disabled:opacity-50 disabled:cursor-not-allowed",
                  item.destructive ? "text-danger" : "text-ink-1",
                )}
              >
                {item.icon}
                {item.label}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
