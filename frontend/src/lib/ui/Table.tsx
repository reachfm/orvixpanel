import { type ReactNode } from "react";
import { cn } from "./cn";

export interface Column<T> {
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  width?: string;       // tailwind width class, e.g. "w-32"
  align?: "left" | "right" | "center";
  className?: string;
}

export function Table<T>({
  columns,
  rows,
  keyOf,
  emptyState,
  isLoading,
  onRowClick,
}: {
  columns: Column<T>[];
  rows: T[];
  keyOf: (row: T) => string;
  emptyState?: ReactNode;
  isLoading?: boolean;
  onRowClick?: (row: T) => void;
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-sm text-ink-3">
        <span className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-brand-500 border-r-transparent" />
        <span className="ml-2">Loading…</span>
      </div>
    );
  }

  if (rows.length === 0 && emptyState) {
    return <div className="py-8">{emptyState}</div>;
  }

  return (
    <div className="overflow-hidden rounded-lg border border-surface-border">
      <table className="w-full table-fixed text-sm">
        <thead className="bg-surface-2 text-left text-xs uppercase tracking-wide text-ink-3">
          <tr>
            {columns.map((c) => (
              <th
                key={c.key}
                className={cn(
                  "px-4 py-2.5 font-medium",
                  c.align === "right" && "text-right",
                  c.align === "center" && "text-center",
                  c.width,
                  c.className,
                )}
              >
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-surface-border">
          {rows.map((r) => (
            <tr
              key={keyOf(r)}
              className={cn(
                "bg-surface-1 hover:bg-surface-2",
                onRowClick && "cursor-pointer",
              )}
              onClick={onRowClick ? () => onRowClick(r) : undefined}
            >
              {columns.map((c) => (
                <td
                  key={c.key}
                  className={cn(
                    "px-4 py-3 text-ink-1",
                    c.align === "right" && "text-right",
                    c.align === "center" && "text-center",
                  )}
                >
                  {c.cell(r)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
