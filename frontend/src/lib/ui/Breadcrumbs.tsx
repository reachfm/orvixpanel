import { type ReactNode } from "react";
import { Link, useMatchRoute } from "@tanstack/react-router";
import { cn } from "@/lib/ui/cn";

/**
 * Global breadcrumbs. Renders an array of {label, to?} as a slash-
 * separated nav. If `to` is omitted the crumb is rendered as plain
 * text (terminal crumb). The router's useMatchRoute is used to
 * highlight the active route in the sidebar; the breadcrumbs are
 * route-derived (each page passes a list).
 */
export interface Crumb {
  label: ReactNode;
  to?: string;
}

export function Breadcrumbs({ items }: { items: Crumb[] }) {
  if (items.length === 0) return null;
  return (
    <nav aria-label="Breadcrumb" className="flex items-center text-xs text-ink-3">
      {items.map((c, i) => (
        <span key={i} className="flex items-center">
          {i > 0 && <span className="mx-1.5 text-ink-4">/</span>}
          {c.to ? (
            <Link
              to={c.to}
              className="rounded px-1 py-0.5 hover:bg-surface-2 hover:text-ink-1"
            >
              {c.label}
            </Link>
          ) : (
            <span className="px-1 py-0.5 text-ink-2">{c.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}

/** Convenience: build breadcrumbs from a pathname like "/accounts/123". */
export function useBreadcrumbsFromPath(
  path: string,
  labelMap: Record<string, string>,
): Crumb[] {
  const parts = path.split("/").filter(Boolean);
  const crumbs: Crumb[] = [{ label: "Home", to: "/" }];
  let acc = "";
  for (const p of parts) {
    acc += `/${p}`;
    const label = labelMap[p] ?? p;
    const isLast = acc === path;
    crumbs.push({ label, to: isLast ? undefined : acc });
  }
  return crumbs;
}

export function SidebarItem({
  to,
  label,
  icon,
  end,
}: {
  to: string;
  label: string;
  icon?: ReactNode;
  end?: boolean;
}) {
  const matchRoute = useMatchRoute();
  const active = !!matchRoute({ to, fuzzy: !end });
  return (
    <Link
      to={to}
      activeOptions={end ? { exact: true } : undefined}
      className={cn(
        "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium",
        "transition-colors",
        active
          ? "bg-brand-600/15 text-brand-600 dark:text-brand-300"
          : "text-ink-2 hover:bg-surface-2 hover:text-ink-1",
      )}
    >
      {icon && <span className="h-4 w-4 shrink-0">{icon}</span>}
      <span className="truncate">{label}</span>
    </Link>
  );
}
