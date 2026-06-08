/**
 * Topbar — Professional top status bar.
 * Sections: breadcrumbs, search, health, license, theme, notifications, user
 * All values handled safely with fallbacks.
 */

import { useState, type FormEvent, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { systemKeys } from "@/lib/query/keys";
import { healthz, licenseRenewal } from "@/lib/api/system";
import { useThemeStore } from "@/lib/theme/store";
import { useAuthStore } from "@/lib/auth/store";
import { Dropdown } from "./Dropdown";
import { StatusPill } from "./StatusPill";
import { logout as apiLogout } from "@/lib/api/auth";
import { cn } from "./cn";

export function Topbar({ breadcrumbs }: { breadcrumbs?: ReactNode }) {
  const [query, setQuery] = useState("");
  const navigate = useNavigate();
  const { theme, toggle } = useThemeStore();
  const user = useAuthStore((s) => s.user);
  const clearAuth = useAuthStore((s) => s.clear);

  // Health and license polling
  const hz = useQuery({
    queryKey: systemKeys.healthz(),
    queryFn: healthz,
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });
  const lic = useQuery({
    queryKey: systemKeys.licenseRenewal(),
    queryFn: licenseRenewal,
    refetchInterval: 60_000,
  });

  const onSearch = (e: FormEvent) => {
    e.preventDefault();
    const q = query.trim();
    if (!q) return;
    navigate({ to: "/accounts" });
  };

  const onLogout = async () => {
    try {
      await apiLogout();
    } catch {
      // ignore
    }
    clearAuth();
    navigate({ to: "/login" });
  };

  // Safe user initial
  const userInitial = user?.email?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <header
      className={cn(
        "flex h-14 shrink-0 items-center gap-4 border-b border-surface-border bg-surface-1",
        "px-4",
      )}
    >
      {/* breadcrumbs */}
      <div className="min-w-0 flex-1 truncate">{breadcrumbs}</div>

      {/* search */}
      <form onSubmit={onSearch} className="hidden md:block w-80">
        <div className="relative">
          <input
            type="search"
            placeholder="Search accounts, domains…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className={cn(
              "w-full rounded-md border border-surface-border bg-surface-0",
              "py-2 pl-9 pr-4 text-sm placeholder:text-ink-4",
              "focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500/20",
            )}
          />
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.8"
            className="absolute left-3 top-2.5 h-4 w-4 text-ink-3"
          >
            <circle cx="11" cy="11" r="7" />
            <path d="m20 20-3.5-3.5" />
          </svg>
        </div>
      </form>

      {/* Status indicators */}
      <div className="hidden lg:flex items-center gap-2">
        {/* Health pill */}
        <StatusPill
          tone={hz.data?.status === "ok" ? "success" : "danger"}
          title={`API Health: ${hz.data?.status ?? "unknown"}`}
        >
          {hz.isLoading ? (
            <span className="flex items-center gap-1">
              <span className="h-1.5 w-1.5 rounded-full bg-current animate-pulse" />
              checking
            </span>
          ) : hz.data?.status === "ok" ? (
            "Healthy"
          ) : (
            "Down"
          )}
        </StatusPill>

        {/* License badge */}
        {lic.data && (
          <StatusPill
            tone={
              lic.data.status === "active"
                ? "success"
                : lic.data.status === "grace"
                  ? "warning"
                  : "danger"
            }
            title={`License: ${lic.data.tier ?? "—"} — ${lic.data.days_remaining ?? 0}d remaining`}
          >
            {lic.data.tier?.toUpperCase() ?? "—"} · {lic.data.days_remaining ?? 0}d
          </StatusPill>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1">
        {/* Theme toggle */}
        <button
          type="button"
          onClick={toggle}
          className={cn(
            "grid h-9 w-9 place-items-center rounded-md text-ink-2",
            "hover:bg-surface-2 hover:text-ink-1 transition-colors",
          )}
          aria-label="Toggle theme"
          title={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
        >
          {theme === "dark" ? <IconSun /> : <IconMoon />}
        </button>

        {/* Notifications */}
        <button
          type="button"
          onClick={() => navigate({ to: "/audit-log" })}
          className={cn(
            "relative grid h-9 w-9 place-items-center rounded-md text-ink-2",
            "hover:bg-surface-2 hover:text-ink-1 transition-colors",
          )}
          aria-label="Notifications"
          title="Recent activity"
        >
          <IconBell />
          <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-brand-500" />
        </button>

        {/* User menu */}
        <Dropdown
          trigger={(open) => (
            <div
              className={cn(
                "flex items-center gap-2.5 rounded-md px-2.5 py-1.5",
                open ? "bg-surface-2" : "hover:bg-surface-2 transition-colors",
              )}
            >
              <div className="grid h-8 w-8 place-items-center rounded-full bg-brand-600 text-sm font-semibold text-white">
                {userInitial}
              </div>
              <div className="hidden sm:block text-left leading-tight">
                <div className="max-w-[140px] truncate text-xs font-medium text-ink-1">
                  {user?.email ?? "—"}
                </div>
                <div className="text-[10px] uppercase tracking-wider text-ink-3">
                  {user?.role ?? "Admin"}
                </div>
              </div>
            </div>
          )}
          items={[
            {
              key: "theme",
              label: theme === "dark" ? "Light mode" : "Dark mode",
              onClick: toggle,
              icon: theme === "dark" ? <IconSun className="h-3.5 w-3.5" /> : <IconMoon className="h-3.5 w-3.5" />,
            },
            { key: "settings", label: "Settings", onClick: () => navigate({ to: "/settings" }) },
            { key: "divider", type: "divider" as const },
            { key: "logout", label: "Sign out", onClick: onLogout, destructive: true },
          ]}
        />
      </div>
    </header>
  );
}

function Icon({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className ?? "h-4 w-4"}
    >
      {children}
    </svg>
  );
}
function IconSun({ className }: { className?: string }) {
  return (
    <Icon className={className}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </Icon>
  );
}
function IconMoon({ className }: { className?: string }) {
  return (
    <Icon className={className}>
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </Icon>
  );
}
function IconBell() {
  return (
    <Icon>
      <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
      <path d="M10 21a2 2 0 0 0 4 0" />
    </Icon>
  );
}