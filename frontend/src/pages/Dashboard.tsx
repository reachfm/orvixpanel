/**
 * Enterprise Dashboard — cPanel-style with real data only.
 * Sections:
 *   - Welcome + Quick Actions
 *   - System Status Cards (API, DB, License, Uptime)
 *   - Resource Summary (Accounts, Domains, Mail, DNS, SSL, Backups)
 *   - Recent Activity
 *   - Get Started CTA for fresh installs
 */

import { useMemo } from "react";
import { useQuery, useQueries } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner } from "@/lib/ui/Feedback";
import { Button } from "@/lib/ui/Button";
import { systemKeys, accountKeys, domainKeys, auditKeys } from "@/lib/query/keys";
import {
  healthz, readyz, systemInfo, licenseRenewal, listAudit,
} from "@/lib/api/system";
import { listAccounts } from "@/lib/api/accounts";
import { listDomains } from "@/lib/api/domains";
import { listDeployments } from "@/lib/api/deployments";
import { useAuthStore } from "@/lib/auth/store";
import { sslKeys, getSSLDashboardStats } from "@/lib/api/ssl";
import { timeAgo, safeUpper } from "@/lib/utils";

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  const hz = useQuery({ queryKey: systemKeys.healthz(), queryFn: healthz, refetchInterval: 15_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(), queryFn: readyz, refetchInterval: 15_000 });
  const sys = useQuery({ queryKey: systemKeys.info(), queryFn: systemInfo });
  const lic = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal });
  const accts = useQuery({ queryKey: accountKeys.list(), queryFn: listAccounts });
  const audit = useQuery({ queryKey: auditKeys.list(5), queryFn: () => listAudit(5) });
  const sslStats = useQuery({ queryKey: sslKeys.dashboard(), queryFn: getSSLDashboardStats });

  // Fetch domains and deployments for each account
  const accounts = accts.data?.accounts ?? [];
  const domainQueries = useQueries({
    queries: accounts.map((a) => ({
      queryKey: [...domainKeys.byAccount(a.id), "dashboard-count"],
      queryFn: () => listDomains(a.id),
      enabled: accts.isSuccess,
    })),
  });
  const deploymentQueries = useQueries({
    queries: accounts.map((a) => ({
      queryKey: ["deployments", a.id, "dashboard-count"],
      queryFn: () => listDeployments(a.id),
      enabled: accts.isSuccess,
    })),
  });

  // Calculate aggregated counts safely
  const stats = useMemo(() => {
    const domainCount = domainQueries.reduce((sum, q) => sum + (q.data?.domains?.length ?? 0), 0);
    const deploymentCount = deploymentQueries.reduce((sum, q) => sum + (q.data?.deployments?.length ?? 0), 0);
    const accountCount = accounts.length;
    const activeCount = accounts.filter((a) => a.status === "active").length;
    const suspendedCount = accounts.filter((a) => a.status === "suspended").length;
    return { accountCount, activeCount, suspendedCount, domainCount, deploymentCount };
  }, [accounts, domainQueries, deploymentQueries]);

  const recentAuditEntries = audit.data?.entries ?? [];

  // Calculate uptime string
  const uptimeString = useMemo(() => {
    if (!sys.data?.uptime_at) return null;
    try {
      const start = new Date(sys.data.uptime_at).getTime();
      const now = Date.now();
      const diffMs = now - start;
      const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
      const diffDays = Math.floor(diffHours / 24);
      if (diffDays > 0) return `${diffDays}d ${diffHours % 24}h`;
      if (diffHours > 0) return `${diffHours}h`;
      const diffMins = Math.floor(diffMs / (1000 * 60));
      return `${diffMins}m`;
    } catch {
      return null;
    }
  }, [sys.data?.uptime_at]);

  const isFreshInstall = stats.accountCount === 0;

  return (
    <div className="space-y-6">
      {/* Welcome + Quick Actions */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <PageHeader
            title={`Welcome${user?.email ? `, ${user.email.split("@")[0]}` : ""}`}
            description="Operational overview of your OrvixPanel instance."
          />
        </div>
        {!isFreshInstall && (
          <div className="flex flex-wrap gap-2">
            <Link to="/accounts/new">
              <Button variant="primary" size="sm">
                New Account
              </Button>
            </Link>
            <Link to="/mail/mailboxes">
              <Button variant="secondary" size="sm">
                New Mailbox
              </Button>
            </Link>
            <Link to="/backup">
              <Button variant="secondary" size="sm">
                Create Backup
              </Button>
            </Link>
          </div>
        )}
      </div>

      {/* System Status Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatusCard
          label="API Health"
          value={hz.isLoading ? <Spinner size={18} /> : hz.data?.status === "ok" ? "OK" : "Down"}
          tone={hz.data?.status === "ok" ? "success" : "danger"}
          badge="HTTP"
          subtitle={hz.data?.status === "ok" ? "Responding normally" : "Check server"}
        />
        <StatusCard
          label="Database"
          value={rz.isLoading ? <Spinner size={18} /> : rz.data?.status ?? "—"}
          tone={rz.data?.status === "ready" ? "success" : "danger"}
          badge="DB"
          subtitle={rz.data?.status === "ready" ? "Connected" : "Connection issue"}
        />
        <StatusCard
          label="License"
          value={lic.isLoading ? <Spinner size={18} /> : lic.data?.loaded === false ? "NO LICENSE" : safeUpper(lic.data?.tier) || "—"}
          tone={lic.data?.loaded === false ? "danger" : lic.data?.status === "active" ? "success" : lic.data?.status === "grace" ? "warning" : "danger"}
          badge={lic.data?.loaded === false ? "LOCKED" : lic.data?.mode ?? lic.data?.status ?? "—"}
          subtitle={lic.data?.loaded === false ? "Configure license" : lic.data ? `${lic.data.days_remaining ?? 0}d remaining` : undefined}
        />
        <StatusCard
          label="Uptime"
          value={uptimeString ?? (sys.isLoading ? <Spinner size={18} /> : "—")}
          tone="neutral"
          badge="Runtime"
          subtitle={uptimeString ? "Since last restart" : undefined}
        />
      </div>

      {/* Resource Summary */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <MetricCard
          label="Accounts"
          value={stats.accountCount}
          isLoading={accts.isLoading}
          href="/accounts"
          subtitle={`${stats.activeCount} active`}
          icon={<IconUsers />}
        />
        <MetricCard
          label="Domains"
          value={stats.domainCount}
          isLoading={accts.isLoading || domainQueries.some((q) => q.isLoading)}
          href="/domains"
          icon={<IconGlobe />}
        />
        <MetricCard
          label="Deployments"
          value={stats.deploymentCount}
          isLoading={accts.isLoading || deploymentQueries.some((q) => q.isLoading)}
          href="/deployments"
          icon={<IconRocket />}
        />
        <MetricCard
          label="SSL Certs"
          value={sslStats.data?.total_active ?? 0}
          isLoading={sslStats.isLoading}
          href="/ssl/certificates"
          subtitle={sslStats.data?.expiring_soon ? `${sslStats.data.expiring_soon} expiring` : undefined}
          icon={<IconSSL />}
        />
        <MetricCard
          label="Version"
          value={sys.data?.version ?? "—"}
          isLoading={sys.isLoading}
          isText
          icon={<IconInfo />}
        />
        <MetricCard
          label="Audit Events"
          value={recentAuditEntries.length}
          isLoading={audit.isLoading}
          href="/audit-log"
          subtitle="Last 5"
          icon={<IconScroll />}
        />
      </div>

      {/* System Info + Recent Activity */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader
            title="System Information"
            description="Build metadata returned by the system API."
            actions={
              <StatusPill tone={sys.isError ? "danger" : sys.data ? "success" : "neutral"}>
                {sys.isError ? "error" : sys.isLoading ? "loading" : "ok"}
              </StatusPill>
            }
          />
          {sys.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Spinner size={18} />
            </div>
          ) : sys.data ? (
            <dl className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
              <Field label="Name" value={sys.data.name ?? "—"} />
              <Field label="Version" value={sys.data.version ?? "—"} />
              <Field label="Started" value={sys.data.uptime_at ? new Date(sys.data.uptime_at).toLocaleString() : "—"} />
              <Field label="Uptime" value={uptimeString ?? "—"} />
            </dl>
          ) : (
            <p className="text-sm text-ink-3">System information unavailable</p>
          )}
        </Card>

        <Card>
          <CardHeader
            title="Recent Activity"
            actions={
              <Link to="/audit-log" className="text-xs text-brand-600 hover:underline">
                View all
              </Link>
            }
          />
          {audit.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Spinner size={18} />
            </div>
          ) : recentAuditEntries.length === 0 ? (
            <div className="py-6 text-center">
              <p className="text-sm text-ink-3">No recent activity</p>
              <p className="mt-1 text-xs text-ink-4">Activity will appear here as you use the panel</p>
            </div>
          ) : (
            <ul className="space-y-3">
              {recentAuditEntries.slice(0, 5).map((entry) => (
                <li key={entry.id} className="flex items-start gap-3">
                  <StatusPill
                    tone={entry.result === "success" ? "success" : entry.result === "denied" ? "warning" : "danger"}
                    className="mt-0.5 shrink-0 text-[10px]"
                  >
                    {entry.result?.slice(0, 4) ?? "—"}
                  </StatusPill>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm text-ink-1">{entry.action ?? "—"}</div>
                    <div className="text-[11px] text-ink-3">{timeAgo(entry.timestamp)}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      {/* Get Started for fresh installs */}
      {isFreshInstall && (
        <Card className="border-brand-500/30 bg-brand-500/5">
          <CardHeader
            title="Get Started"
            description="Create your first account to start serving sites."
          />
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div className="flex-1 text-sm text-ink-2">
              <p>You have no accounts yet. Create one to begin.</p>
              <p className="mt-1 text-xs text-ink-3">Each account can host multiple domains and websites.</p>
            </div>
            <Link to="/accounts/new">
              <Button variant="primary">
                Create your first account
              </Button>
            </Link>
          </div>
        </Card>
      )}
    </div>
  );
}

function StatusCard({
  label,
  value,
  tone,
  badge,
  subtitle,
}: {
  label: string;
  value: React.ReactNode;
  tone: "success" | "danger" | "warning" | "neutral" | "info";
  badge: string;
  subtitle?: string;
}) {
  return (
    <Card className="transition-shadow hover:shadow-md">
      <div className="flex items-start justify-between">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">{label}</div>
          <div className="mt-1.5 text-2xl font-bold text-ink-1">{value}</div>
          {subtitle && <div className="mt-0.5 text-[11px] text-ink-3">{subtitle}</div>}
        </div>
        <StatusPill tone={tone} className="shrink-0">
          {badge}
        </StatusPill>
      </div>
    </Card>
  );
}

function MetricCard({
  label,
  value,
  isLoading,
  href,
  subtitle,
  isText,
  icon,
}: {
  label: string;
  value: string | number;
  isLoading?: boolean;
  href?: string;
  subtitle?: string;
  isText?: boolean;
  icon?: React.ReactNode;
}) {
  const content = (
    <div className="flex items-start justify-between">
      <div>
        <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-ink-3">
          {icon && <span className="opacity-60">{icon}</span>}
          {label}
        </div>
        <div className={`mt-1.5 font-bold text-ink-1 ${isText ? "text-base" : "text-2xl"}`}>
          {isLoading ? <Spinner size={16} /> : value}
        </div>
        {subtitle && <div className="mt-0.5 text-[11px] text-ink-3">{subtitle}</div>}
      </div>
      {href && !isLoading && (
        <span className="text-[11px] font-medium text-brand-600 hover:underline">View</span>
      )}
    </div>
  );

  if (href) {
    return (
      <Link to={href} className="block">
        <Card className="transition-all hover:border-brand-500/30 hover:shadow-md">
          {content}
        </Card>
      </Link>
    );
  }

  return <Card>{content}</Card>;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className="mt-0.5 font-mono text-sm text-ink-1 break-all">{value}</dd>
    </div>
  );
}

// Inline icons
function Icon({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className ?? "h-3.5 w-3.5"}
    >
      {children}
    </svg>
  );
}
function IconUsers() {
  return <Icon><circle cx="9" cy="8" r="3.5" /><path d="M2 21c0-3.5 3-6 7-6s7 2.5 7 6" /></Icon>;
}
function IconGlobe() {
  return <Icon><circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" /></Icon>;
}
function IconRocket() {
  return <Icon><path d="M14 4c4 0 6 2 6 6-1 3-4 6-7 8l-4-4c2-3 5-6 8-7" /><path d="M9 15l-4 4-1-1 4-4" /></Icon>;
}
function IconSSL() {
  return <Icon><path d="M12 2a5 5 0 0 1 5 5v4a5 5 0 0 1-10 0V7a5 5 0 0 1 5-5z" /><path d="M12 14v6" /><path d="M8 22h8" /></Icon>;
}
function IconInfo() {
  return <Icon><circle cx="12" cy="12" r="10" /><path d="M12 16v-4M12 8h.01" /></Icon>;
}
function IconScroll() {
  return <Icon><path d="M5 3h12a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5z" /><path d="M8 8h8M8 12h8M8 16h5" /></Icon>;
}