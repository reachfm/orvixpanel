/**
 * Tabs — local in-page tab strip, controlled. Used by pages that
 * have a fixed sub-navigation (Account detail, Settings).
 */

import { type ReactNode } from "react";
import { cn } from "./cn";

export interface TabSpec {
  key: string;
  label: ReactNode;
  panel: ReactNode;
}

export function Tabs({
  tabs,
  active,
  onChange,
}: {
  tabs: TabSpec[];
  active: string;
  onChange: (key: string) => void;
}) {
  return (
    <div>
      <div className="flex border-b border-surface-border">
        {tabs.map((t) => {
          const isActive = t.key === active;
          return (
            <button
              key={t.key}
              type="button"
              onClick={() => onChange(t.key)}
              className={cn(
                "relative -mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors",
                isActive
                  ? "border-brand-600 text-brand-600 dark:border-brand-400 dark:text-brand-300"
                  : "border-transparent text-ink-3 hover:text-ink-1",
              )}
            >
              {t.label}
            </button>
          );
        })}
      </div>
      <div className="pt-4">
        {tabs.find((t) => t.key === active)?.panel}
      </div>
    </div>
  );
}
