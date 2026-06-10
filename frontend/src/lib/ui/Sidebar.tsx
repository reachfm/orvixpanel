/**
 * Sidebar — Professional cPanel-style navigation with section labels.
 * Grouped by: Core, Mail, Services, System
 * Icons are inline SVGs for zero icon lib dependency.
 */

import { type ReactNode } from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { cn } from "@/lib/ui/cn";
import { useAuthStore } from "@/lib/auth/store";

interface NavItem {
  to: string;
  label: string;
  end?: boolean;
  icon: ReactNode;
}

interface NavSection {
  label: string;
  items: NavItem[];
}

const navSections: NavSection[] = [
  {
    label: "Core",
    items: [
      { to: "/", label: "Dashboard", end: true, icon: <IconHome /> },
      { to: "/accounts", label: "Accounts", icon: <IconUsers /> },
      { to: "/domains", label: "Domains", icon: <IconGlobe /> },
    ],
  },
  {
    label: "Mail",
    items: [
      { to: "/mail/domains", label: "Mail Domains", icon: <IconEmail /> },
      { to: "/mail/mailboxes", label: "Mailboxes", icon: <IconMailbox /> },
      { to: "/mail/aliases", label: "Aliases", icon: <IconAlias /> },
      { to: "/mail/forwarders", label: "Forwarders", icon: <IconForward /> },
      { to: "/mail/audit", label: "Mail Audit", icon: <IconScroll /> },
    ],
  },
  {
    label: "Services",
    items: [
      { to: "/dns/zones", label: "DNS Zones", icon: <IconDNS /> },
      { to: "/ssl/certificates", label: "SSL Certificates", icon: <IconSSL /> },
      { to: "/backup", label: "Backups", icon: <IconBackup /> },
      { to: "/deployments", label: "Deployments", icon: <IconRocket /> },
    ],
  },
  {
    label: "System",
    items: [
      { to: "/system-health", label: "System Health", icon: <IconPulse /> },
      { to: "/updates", label: "Update Center", icon: <IconRefresh /> },
      { to: "/audit-log", label: "Audit Log", icon: <IconShield /> },
      { to: "/settings", label: "Settings", icon: <IconGear /> },
      { to: "/settings/notifications", label: "Notifications", icon: <IconBell /> },
    ],
  },
];

export function Sidebar() {
  const user = useAuthStore((s) => s.user);
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-surface-border bg-surface-1">
      {/* Brand */}
      <div className="flex h-14 items-center gap-3 border-b border-surface-border px-4">
        <div className="grid h-9 w-9 place-items-center rounded-lg bg-brand-600 text-sm font-bold text-white shadow-sm">
          O
        </div>
        <div className="leading-tight">
          <div className="text-base font-semibold text-ink-1">OrvixPanel</div>
          <div className="text-[10px] font-medium uppercase tracking-widest text-ink-3">
            Enterprise
          </div>
        </div>
      </div>

      {/* Nav with sections */}
      <nav className="flex-1 overflow-y-auto p-3 scroll-surface">
        {navSections.map((section) => (
          <div key={section.label} className="mb-4">
            <div className="mb-1.5 px-2.5 text-[10px] font-semibold uppercase tracking-widest text-ink-3">
              {section.label}
            </div>
            <div className="space-y-0.5">
              {section.items.map((item) => {
                const active = item.end ? pathname === item.to : pathname.startsWith(item.to);
                return (
                  <Link
                    key={item.to}
                    to={item.to}
                    className={cn(
                      "flex items-center gap-3 rounded-md px-2.5 py-2 text-sm font-medium",
                      "transition-colors duration-150",
                      active
                        ? "bg-brand-600/15 text-brand-600 dark:text-brand-300"
                        : "text-ink-2 hover:bg-surface-2 hover:text-ink-1",
                    )}
                  >
                    <span className="h-4 w-4 shrink-0 opacity-80">{item.icon}</span>
                    <span className="truncate">{item.label}</span>
                    {active && (
                      <span className="ml-auto h-1.5 w-1.5 rounded-full bg-brand-500" />
                    )}
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* User footer */}
      <div className="border-t border-surface-border p-4">
        <div className="flex items-center gap-3">
          <div className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-brand-600 text-sm font-semibold text-white">
            {(user?.email ?? "?").charAt(0).toUpperCase()}
          </div>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium text-ink-1">
              {user?.email ?? "—"}
            </div>
            <div className="text-[10px] uppercase tracking-wider text-ink-3">
              {user?.role ?? "Admin"}
            </div>
          </div>
        </div>
      </div>
    </aside>
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

function IconHome() {
  return (
    <Icon>
      <path d="M3 11 12 4l9 7" />
      <path d="M5 10v10h14V10" />
    </Icon>
  );
}
function IconUsers() {
  return (
    <Icon>
      <circle cx="9" cy="8" r="3.5" />
      <path d="M2 21c0-3.5 3-6 7-6s7 2.5 7 6" />
      <circle cx="17" cy="9" r="2.5" />
      <path d="M22 19c0-2-1.5-3.5-4-4" />
    </Icon>
  );
}
function IconGlobe() {
  return (
    <Icon>
      <circle cx="12" cy="12" r="9" />
      <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
    </Icon>
  );
}
function IconDNS() {
  return (
    <Icon>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
    </Icon>
  );
}
function IconRocket() {
  return (
    <Icon>
      <path d="M14 4c4 0 6 2 6 6-1 3-4 6-7 8l-4-4c2-3 5-6 8-7" />
      <path d="M9 15l-4 4-1-1 4-4" />
      <path d="M14 4l-2 2 4 4 2-2" />
    </Icon>
  );
}
function IconPulse() {
  return <Icon><path d="M3 12h4l2-6 4 12 2-6h6" /></Icon>;
}
function IconScroll() {
  return (
    <Icon>
      <path d="M5 3h12a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5z" />
      <path d="M8 8h8M8 12h8M8 16h5" />
    </Icon>
  );
}
function IconGear() {
  return (
    <Icon>
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3h0a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z" />
    </Icon>
  );
}
function IconSSL() {
  return (
    <Icon>
      <path d="M12 2a5 5 0 0 1 5 5v4a5 5 0 0 1-10 0V7a5 5 0 0 1 5-5z" />
      <path d="M12 14v6" />
      <path d="M8 22h8" />
      <circle cx="12" cy="7" r="1.5" />
    </Icon>
  );
}
function IconEmail() {
  return (
    <Icon>
      <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
      <polyline points="22,6 12,13 2,6" />
    </Icon>
  );
}
function IconMailbox() {
  return (
    <Icon>
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <path d="M2 8l10 5 10-5" />
    </Icon>
  );
}
function IconAlias() {
  return (
    <Icon>
      <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
      <rect x="8" y="2" width="8" height="4" rx="1" />
      <path d="M9 14l2 2 4-4" />
    </Icon>
  );
}
function IconForward() {
  return (
    <Icon>
      <path d="M5 12h14" />
      <path d="M12 5l7 7-7 7" />
    </Icon>
  );
}
function IconBackup() {
  return (
    <Icon>
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
      <polyline points="7 10 12 15 17 10" />
      <line x1="12" y1="15" x2="12" y2="3" />
    </Icon>
  );
}
function IconShield() {
  return (
    <Icon>
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <path d="M9 12l2 2 4-4" />
    </Icon>
  );
}
function IconRefresh() {
  return (
    <Icon>
      <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
      <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
      <path d="M3 21v-5h5" />
    </Icon>
  );
}
function IconBell() {
  return (
    <Icon>
      <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
      <path d="M13.73 21a2 2 0 0 1-3.46 0" />
    </Icon>
  );
}