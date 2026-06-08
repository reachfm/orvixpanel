/**
 * Sidebar — the persistent left enterprise navigation. The icons
 * are inline SVGs (no icon lib dependency for the v0.2.3 foundation;
 * later we can swap to lucide-react at the call site without
 * changing the layout).
 */

import { type ReactNode } from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { cn } from "@/lib/ui/cn";
import { useAuthStore } from "@/lib/auth/store";

const navItems = [
  { to: "/",                label: "Dashboard",      end: true,  icon: <IconHome /> },
  { to: "/accounts",        label: "Accounts",       icon: <IconUsers /> },
  { to: "/domains",         label: "Domains",        icon: <IconGlobe /> },
  { to: "/dns/zones",       label: "DNS Zones",      icon: <IconDNS /> },
  { to: "/deployments",     label: "Deployments",    icon: <IconRocket /> },
  { to: "/system-health",   label: "System Health",  icon: <IconPulse /> },
  { to: "/audit-log",       label: "Audit Log",      icon: <IconScroll /> },
  { to: "/settings",        label: "Settings",       icon: <IconGear /> },
];

export function Sidebar() {
  const user = useAuthStore((s) => s.user);
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-surface-border bg-surface-1">
      {/* Brand */}
      <div className="flex h-14 items-center gap-2 border-b border-surface-border px-4">
        <div className="grid h-8 w-8 place-items-center rounded-md bg-brand-600 text-sm font-bold text-white">O</div>
        <div className="leading-tight">
          <div className="text-sm font-semibold text-ink-1">OrvixPanel</div>
          <div className="text-[10px] uppercase tracking-wider text-ink-3">Enterprise</div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 space-y-0.5 overflow-y-auto p-3 scroll-surface">
        {navItems.map((it) => {
          const active = it.end ? pathname === it.to : pathname.startsWith(it.to);
          return (
            <Link
              key={it.to}
              to={it.to}
              className={cn(
                "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium",
                "transition-colors",
                active
                  ? "bg-brand-600/15 text-brand-600 dark:text-brand-300"
                  : "text-ink-2 hover:bg-surface-2 hover:text-ink-1",
              )}
            >
              <span className="h-4 w-4 shrink-0">{it.icon}</span>
              <span className="truncate">{it.label}</span>
            </Link>
          );
        })}
      </nav>

      {/* User footer */}
      <div className="border-t border-surface-border p-3">
        <div className="text-xs text-ink-3">Signed in as</div>
        <div className="truncate text-sm font-medium text-ink-1">
          {user?.email ?? "—"}
        </div>
        <div className="mt-0.5 text-[10px] uppercase tracking-wider text-ink-3">
          {user?.role ?? "—"}
        </div>
      </div>
    </aside>
  );
}

function Icon({ children }: { children: ReactNode }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-4 w-4"
    >
      {children}
    </svg>
  );
}
function IconHome()     { return <Icon><path d="M3 11 12 4l9 7"/><path d="M5 10v10h14V10"/></Icon>; }
function IconUsers()    { return <Icon><circle cx="9" cy="8" r="3.5"/><path d="M2 21c0-3.5 3-6 7-6s7 2.5 7 6"/><circle cx="17" cy="9" r="2.5"/><path d="M22 19c0-2-1.5-3.5-4-4"/></Icon>; }
function IconGlobe()    { return <Icon><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/></Icon>; }
function IconDNS()      { return <Icon><path d="M12 2a5 5 0 1 0 0 10 5 5 0 0 0 0-10z"/><path d="M12 8v8"/><path d="M8 12h8"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="M12 2v2"/></Icon>; }
function IconRocket()   { return <Icon><path d="M14 4c4 0 6 2 6 6-1 3-4 6-7 8l-4-4c2-3 5-6 8-7"/><path d="M9 15l-4 4-1-1 4-4"/><path d="M14 4l-2 2 4 4 2-2"/></Icon>; }
function IconPulse()    { return <Icon><path d="M3 12h4l2-6 4 12 2-6h6"/></Icon>; }
function IconScroll()   { return <Icon><path d="M5 3h12a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></Icon>; }
function IconGear()     { return <Icon><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3h0a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></Icon>; }
