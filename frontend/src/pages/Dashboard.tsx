/**
 * Enterprise Dashboard — cPanel-style with real data only.
 * v0.3.1 Phase A: Enterprise Dashboard enhancements:
 *   - Real-time system health integration
 *   - Activity timeline with richer context
 *   - Alerts section for warnings
 *   - Service status grid
 *   - SSL health overview
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
  healthz, readyz, systemInfo, licenseRenewal, listAudit, systemHealth,
} from "@/lib/api/system";
import { listAccounts } from "@/lib/api/accounts";
import { listDomains } from "@/lib/api/domains";
import { listDeployments } from "@/lib/api/deployments";
import { useAuthStore } from "@/lib/auth/store";
import { sslKeys, getSSLDashboardStats, getSSLHealth } from "@/lib/api/ssl";
import { timeAgo, safeUpper } from "@/lib/utils";

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  // Real-time health checks (15s refresh)
  const hz = useQuery({ queryKey: systemKeys.healthz(), queryFn: healthz, refetchInterval: 15_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(), queryFn: readyz, refetchInterval: 15_000 });
  const sys = useQuery({ queryKey: systemKeys.info(), queryFn: systemInfo });
  const lic = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal });
  const health = useQuery({ queryKey: systemKeys.health(), queryFn: systemHealth });

  // Data queries
  const accts = useQuery({ queryKey: accountKeys.list(), queryFn: listAccounts });
  const audit = useQuery({ queryKey: auditKeys.list(10), queryFn: () => listAudit(10) });
  const sslStats = useQuery({ queryKey: sslKeys.dashboard(), queryFn: getSSLDashboardStats });
  const sslHealth = useQuery({ queryKey: sslKeys.health(), queryFn: getSSLHealth });

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

  // Build alerts from various sources
  const alerts = useMemo(() => {
    const list: { type: "warning" | "danger" | "info"; title: string; message: string; href?: string }[] = [];

    // SSL alerts
    if (sslHealth.data?.expiring_soon) {
      list.push({
        type: "warning",
        title: `${sslHealth.data.expiring_soon} SSL certs expiring soon`,
        message: "Review and renew before expiration",
        href: "/ssl/certificates",
      });
    }
    if (sslHealth.data?.failed) {
      list.push({
        type: "danger",
        title: `${sslHealth.data.failed} SSL certs failed`,
        message: "Check certificate status for details",
        href: "/ssl/certificates",
      });
    }
    if (sslHealth.data?.expired) {
      list.push({
        type: "danger",
        title: `${sslHealth.data.expired} SSL certs expired`,
        message: "Immediate attention required",
        href: "/ssl/certificates",
      });
    }

    // License alerts
    if (lic.data?.loaded === false) {
      list.push({ type: "danger", title: "No license configured", message: "Add a license to unlock all features" });
    } else if (lic.data?.status === "grace") {
      list.push({
        type: "warning",
        title: `License in grace period (${lic.data.days_remaining ?? 0}d remaining)`,
        message: "Renew before grace period ends",
      });
    }

    // System health warnings
    health.data?.checks.forEach((check) => {
      if (check.status === "fail") {
        list.push({ type: "danger", title: check.name, message: check.message });
      } else if (check.status === "warn") {
        list.push({ type: "warning", title: check.name, message: check.message });
      }
    });

    // Account alerts
    if (stats.suspendedCount > 0) {
      list.push({
        type: "info",
        title: `${stats.suspendedCount} accounts suspended`,
        message: "Review suspended accounts",
        href: "/accounts",
      });
    }

    return list;
  }, [sslHealth.data, lic.data, health.data, stats.suspendedCount]);

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

      {/* Alerts Section */}
      {alerts.length > 0 && (
        <div className="space-y-2">
          {alerts.map((alert, i) => (
            <AlertCard key={i} {...alert} />
          ))}
        </div>
      )}

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
          tone={sslStats.data?.expiring_soon ? "warning" : undefined}
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
          value={audit.data?.entries?.length ?? 0}
          isLoading={audit.isLoading}
          href="/audit-log"
          subtitle="Recent"
          icon={<IconScroll />}
        />
      </div>

      {/* System Info + Recent Activity + SSL Health */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* System Information */}
        <Card className="lg:col-span-1">
          <CardHeader
            title="System Info"
            description="Build metadata"
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
            <dl className="space-y-2 text-sm">
              <Field label="Name" value={sys.data.name ?? "—"} />
              <Field label="Version" value={sys.data.version ?? "—"} />
              <Field label="Started" value={sys.data.uptime_at ? new Date(sys.data.uptime_at).toLocaleString() : "—"} />
              <Field label="Uptime" value={uptimeString ?? "—"} />
            </dl>
          ) : (
            <p className="text-sm text-ink-3">System information unavailable</p>
          )}
        </Card>

        {/* SSL Health Overview */}
        <Card className="lg:col-span-1">
          <CardHeader
            title="SSL Health"
            description="Certificate status"
            actions={
              <Link to="/ssl/certificates" className="text-xs text-brand-600 hover:underline">
                View all
              </Link>
            }
          />
          {sslHealth.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Spinner size={18} />
            </div>
          ) : sslHealth.data ? (
            <div className="space-y-3">
              <SSLHealthRow
                label="Healthy"
                value={sslHealth.data.healthy ?? 0}
                total={sslHealth.data.total ?? 0}
                tone="success"
              />
              <SSLHealthRow
                label="Expiring Soon"
                value={sslHealth.data.expiring_soon ?? 0}
                total={sslHealth.data.total ?? 0}
                tone="warning"
              />
              <SSLHealthRow
                label="Failed"
                value={sslHealth.data.failed ?? 0}
                total={sslHealth.data.total ?? 0}
                tone="danger"
              />
              <SSLHealthRow
                label="Expired"
                value={sslHealth.data.expired ?? 0}
                total={sslHealth.data.total ?? 0}
                tone="danger"
              />
            </div>
          ) : (
            <p className="text-sm text-ink-3">SSL health data unavailable</p>
          )}
        </Card>

        {/* Recent Activity */}
        <Card className="lg:col-span-1">
          <CardHeader
            title="Recent Activity"
            description="Latest audit events"
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
          ) : !audit.data?.entries?.length ? (
            <div className="py-6 text-center">
              <p className="text-sm text-ink-3">No recent activity</p>
              <p className="mt-1 text-xs text-ink-4">Activity will appear here as you use the panel</p>
            </div>
          ) : (
            <ul className="space-y-2">
              {audit.data.entries.slice(0, 5).map((entry) => (
                <ActivityItem key={entry.id} entry={entry} />
              ))}
            </ul>
          )}
        </Card>
      </div>

      {/* System Health Checks (if any failures) */}
      {health.data?.checks && health.data.checks.some(c => c.status === "fail" || c.status === "warn") && (
        <Card className="border-amber-500/30 bg-amber-500/5">
          <CardHeader
            title="System Health Warnings"
            description="Checks requiring attention"
          />
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {health.data.checks
              .filter(c => c.status === "fail" || c.status === "warn")
              .map((check) => (
                <div key={check.name} className="flex items-start gap-3 rounded-md border border-ink-6 bg-canvas-1 p-3">
                  <StatusPill tone={check.status === "fail" ? "danger" : "warning"} className="shrink-0">
                    {check.status}
                  </StatusPill>
                  <div className="min-w-0">
                    <div className="font-medium text-sm">{check.name}</div>
                    <div className="text-xs text-ink-3">{check.message}</div>
                    {check.suggestions && check.suggestions.length > 0 && (
                      <div className="mt-1 text-xs text-ink-2">{check.suggestions[0]}</div>
                    )}
                  </div>
                </div>
              ))}
          </div>
        </Card>
      )}

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

// --- Alert Card ---
function AlertCard({ type, title, message, href }: {
  type: "warning" | "danger" | "info";
  title: string;
  message: string;
  href?: string;
}) {
  const toneClasses = {
    warning: "border-amber-500/30 bg-amber-500/5",
    danger: "border-red-500/30 bg-red-500/5",
    info: "border-blue-500/30 bg-blue-500/5",
  };
  const iconColor = {
    warning: "text-amber-500",
    danger: "text-red-500",
    info: "text-blue-500",
  };

  return (
    <div className={`flex items-center gap-3 rounded-md border px-4 py-3 ${toneClasses[type]}`}>
      <svg className={`h-5 w-5 shrink-0 ${iconColor[type]}`} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        {type === "danger" ? (
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        ) : type === "warning" ? (
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
        ) : (
          <path strokeLinecap="round" strokeLinejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
        )}
      </svg>
      <div className="flex-1 min-w-0">
        <div className="font-medium text-sm text-ink-1">{title}</div>
        <div className="text-xs text-ink-3">{message}</div>
      </div>
      {href && (
        <Link to={href} className="shrink-0 text-xs font-medium text-brand-600 hover:underline">
          Review
        </Link>
      )}
    </div>
  );
}

// --- Activity Item ---
function ActivityItem({ entry }: { entry: {
  id: string;
  action: string;
  resource_type: string;
  resource_name: string;
  result: string;
  user_email: string;
  timestamp: string;
}}) {
  return (
    <li className="flex items-start gap-3">
      <StatusPill
        tone={entry.result === "success" ? "success" : entry.result === "denied" ? "warning" : "danger"}
        className="mt-0.5 shrink-0 text-[10px]"
      >
        {entry.result?.slice(0, 4) ?? "—"}
      </StatusPill>
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm text-ink-1">
          <span className="font-medium">{entry.action ?? "—"}</span>
          {entry.resource_name && (
            <span className="text-ink-3"> · {entry.resource_name}</span>
          )}
        </div>
        <div className="flex items-center gap-2 text-[11px] text-ink-4">
          <span>{entry.user_email}</span>
          <span>·</span>
          <span>{timeAgo(entry.timestamp)}</span>
        </div>
      </div>
    </li>
  );
}

// --- SSL Health Row ---
function SSLHealthRow({ label, value, total, tone }: {
  label: string;
  value: number;
  total: number;
  tone: "success" | "warning" | "danger";
}) {
  const percent = total > 0 ? (value / total) * 100 : 0;
  const colorClass = {
    success: "bg-green-500",
    warning: "bg-amber-500",
    danger: "bg-red-500",
  }[tone];

  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-sm text-ink-2">{label}</span>
      <div className="flex items-center gap-2">
        <div className="h-1.5 w-16 overflow-hidden rounded-full bg-ink-6">
          <div className={`h-full ${colorClass}`} style={{ width: `${percent}%` }} />
        </div>
        <span className="w-8 text-right font-mono text-sm text-ink-1">{value}</span>
      </div>
    </div>
  );
}

// --- Status Card ---
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

// --- Metric Card ---
function MetricCard({
  label,
  value,
  isLoading,
  href,
  subtitle,
  isText,
  icon,
  tone,
}: {
  label: string;
  value: string | number;
  isLoading?: boolean;
  href?: string;
  subtitle?: string;
  isText?: boolean;
  icon?: React.ReactNode;
  tone?: "warning" | "danger";
}) {
  const borderClass = tone === "warning" ? "border-amber-500/30" : tone === "danger" ? "border-red-500/30" : "";

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
        <Card className={`transition-all hover:border-brand-500/30 hover:shadow-md ${borderClass}`}>
          {content}
        </Card>
      </Link>
    );
  }

  return <Card className={borderClass}>{content}</Card>;
}

// --- Field ---
function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className="mt-0.5 font-mono text-sm text-ink-1 break-all">{value}</dd>
    </div>
  );
}

// --- Inline Icons ---
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