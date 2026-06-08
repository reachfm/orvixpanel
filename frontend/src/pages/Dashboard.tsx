/**
 * Enterprise Dashboard. Real data only:
 *   - healthz + readyz
 *   - /api/v1/admin/system
 *   - /api/v1/admin/license
 *   - account count from /api/v1/accounts
 *   - domain count (aggregated per account)
 *   - deployment count (aggregated per account)
 *   - recent audit entries
 *
 * No placeholder metrics, no fake charts. Counts are derived from
 * the real lists; if the lists are empty, the page shows 0 and a
 * "Get started" CTA. The license card surfaces tier / days
 * remaining / status with the same shape as the renewal-info API.
 */

import { useMemo } from "react";
import { useQuery, useQueries } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner } from "@/lib/ui/Feedback";
import { systemKeys, accountKeys, domainKeys, auditKeys } from "@/lib/query/keys";
import {
  healthz, readyz, systemInfo, licenseRenewal, listAudit,
} from "@/lib/api/system";
import { listAccounts } from "@/lib/api/accounts";
import { listDomains } from "@/lib/api/domains";
import { listDeployments } from "@/lib/api/deployments";
import { useAuthStore } from "@/lib/auth/store";
import { sslKeys, getSSLDashboardStats } from "@/lib/api/ssl";

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  const hz = useQuery({ queryKey: systemKeys.healthz(), queryFn: healthz, refetchInterval: 15_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(), queryFn: readyz, refetchInterval: 15_000 });
  const sys = useQuery({ queryKey: systemKeys.info(), queryFn: systemInfo });
  const lic = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal });
  const accts = useQuery({ queryKey: accountKeys.list(), queryFn: listAccounts });
  const audit = useQuery({ queryKey: auditKeys.list(5), queryFn: () => listAudit(5) });
  const sslStats = useQuery({ queryKey: sslKeys.dashboard(), queryFn: getSSLDashboardStats });

  // Fetch domains and deployments for each account in parallel
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

  // Calculate aggregated counts
  const stats = useMemo(() => {
    const domainCount = domainQueries.reduce((sum, q) => sum + (q.data?.domains.length ?? 0), 0);
    const deploymentCount = deploymentQueries.reduce((sum, q) => sum + (q.data?.deployments.length ?? 0), 0);
    const accountCount = accounts.length;
    const activeCount = accounts.filter((a) => a.status === "active").length;
    const suspendedCount = accounts.filter((a) => a.status === "suspended").length;
    return { accountCount, activeCount, suspendedCount, domainCount, deploymentCount };
  }, [accounts, domainQueries, deploymentQueries]);

  const recentAuditEntries = audit.data?.entries ?? [];

  // Calculate uptime string
  const uptimeString = useMemo(() => {
    if (!sys.data?.uptime_at) return null;
    const start = new Date(sys.data.uptime_at).getTime();
    const now = Date.now();
    const diffMs = now - start;
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays > 0) return `${diffDays}d ${diffHours % 24}h`;
    if (diffHours > 0) return `${diffHours}h`;
    const diffMins = Math.floor(diffMs / (1000 * 60));
    return `${diffMins}m`;
  }, [sys.data?.uptime_at]);

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Welcome${user?.email ? `, ${user.email}` : ""}`}
        description="Operational overview of your OrvixPanel instance."
      />

      {/* Top row — status tiles */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatusCard
          label="API Health"
          value={hz.isLoading ? <Spinner size={18} /> : hz.data?.status === "ok" ? "ok" : "down"}
          tone={hz.data?.status === "ok" ? "success" : "danger"}
          badge="HTTP"
        />
        <StatusCard
          label="Database"
          value={rz.isLoading ? <Spinner size={18} /> : rz.data?.status ?? "—"}
          tone={rz.data?.status === "ready" ? "success" : "danger"}
          badge="DB"
        />
        <StatusCard
          label="License"
          value={lic.isLoading ? <Spinner size={18} /> : lic.data ? lic.data.tier.toUpperCase() : "—"}
          tone={lic.data?.status === "active" ? "success" : lic.data?.status === "grace" ? "warning" : "danger"}
          badge={lic.data?.status ?? "—"}
          subtitle={lic.data ? `${lic.data.days_remaining}d remaining` : undefined}
        />
        <StatusCard
          label="Uptime"
          value={uptimeString ?? (sys.isLoading ? <Spinner size={18} /> : "—")}
          tone="neutral"
          badge="Runtime"
        />
      </div>

      {/* Second row — resource counts */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <MetricCard
          label="Accounts"
          value={stats.accountCount}
          isLoading={accts.isLoading}
          href="/accounts"
          subtitle={`${stats.activeCount} active, ${stats.suspendedCount} suspended`}
        />
        <MetricCard
          label="Domains"
          value={stats.domainCount}
          isLoading={accts.isLoading || domainQueries.some((q) => q.isLoading)}
          href="/domains"
        />
        <MetricCard
          label="Deployments"
          value={stats.deploymentCount}
          isLoading={accts.isLoading || deploymentQueries.some((q) => q.isLoading)}
          href="/deployments"
        />
        <MetricCard
          label="SSL Certs"
          value={sslStats.data?.total_active ?? 0}
          isLoading={sslStats.isLoading}
          href="/ssl/certificates"
          subtitle={sslStats.data?.expiring_soon ? `${sslStats.data.expiring_soon} expiring soon` : undefined}
        />
        <MetricCard
          label="Version"
          value={sys.data?.version ?? "—"}
          isLoading={sys.isLoading}
          isText
        />
        <MetricCard
          label="Audit Events"
          value={recentAuditEntries.length}
          isLoading={audit.isLoading}
          href="/audit-log"
          subtitle="Last 5"
        />
      </div>

      {/* Third row — system info + recent activity */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader
            title="System Information"
            description="Build metadata returned by /api/v1/admin/system."
            actions={
              <StatusPill tone={sys.isError ? "danger" : sys.data ? "success" : "neutral"}>
                {sys.isError ? "error" : sys.isLoading ? "loading" : "ok"}
              </StatusPill>
            }
          />
          {sys.data ? (
            <dl className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
              <Field label="Name" value={sys.data.name} />
              <Field label="Version" value={sys.data.version} />
              <Field label="Started" value={new Date(sys.data.uptime_at).toLocaleString()} />
              <Field label="Uptime" value={uptimeString ?? "—"} />
            </dl>
          ) : (
            <div className="text-sm text-ink-3">—</div>
          )}
        </Card>

        <Card>
          <CardHeader title="Recent Activity" />
          {audit.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Spinner size={18} />
            </div>
          ) : recentAuditEntries.length === 0 ? (
            <p className="text-sm text-ink-3">No recent activity</p>
          ) : (
            <ul className="space-y-2 text-sm">
              {recentAuditEntries.slice(0, 5).map((entry) => (
                <li key={entry.id} className="flex items-start gap-2">
                  <StatusPill
                    tone={entry.result === "success" ? "success" : entry.result === "denied" ? "warning" : "danger"}
                    className="mt-0.5 shrink-0"
                  >
                    {entry.result.slice(0, 4)}
                  </StatusPill>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-ink-1">{entry.action}</div>
                    <div className="text-[11px] text-ink-3">{formatTimeAgo(entry.timestamp)}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      {/* Fourth row — get started */}
      {stats.accountCount === 0 && (
        <Card>
          <CardHeader title="Get started" />
          <div className="space-y-3 text-sm text-ink-2">
            <p>You have no accounts yet. Create one to start serving sites.</p>
            <Link
              to="/accounts/new"
              className="inline-flex h-9 items-center rounded-md bg-brand-600 px-3.5 text-sm font-medium text-white hover:bg-brand-700"
            >
              Create your first account
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
    <Card>
      <div className="flex items-start justify-between">
        <div>
          <div className="text-xs uppercase tracking-wider text-ink-3">{label}</div>
          <div className="mt-1 text-2xl font-semibold text-ink-1">{value}</div>
          {subtitle && <div className="text-[11px] text-ink-3">{subtitle}</div>}
        </div>
        <StatusPill tone={tone}>{badge}</StatusPill>
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
}: {
  label: string;
  value: string | number;
  isLoading?: boolean;
  href?: string;
  subtitle?: string;
  isText?: boolean;
}) {
  const content = (
    <div className="flex items-start justify-between">
      <div>
        <div className="text-[11px] uppercase tracking-wider text-ink-3">{label}</div>
        <div className={`mt-1 font-semibold text-ink-1 ${isText ? "text-lg" : "text-2xl"}`}>
          {isLoading ? <Spinner size={16} /> : value}
        </div>
        {subtitle && <div className="text-[11px] text-ink-3">{subtitle}</div>}
      </div>
      {href && !isLoading && (
        <span className="text-xs text-brand-600 hover:underline">View</span>
      )}
    </div>
  );

  if (href) {
    return (
      <Link to={href} className="block">
        <Card className="hover:bg-surface-2 transition-colors">{content}</Card>
      </Link>
    );
  }

  return <Card>{content}</Card>;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className="mt-0.5 font-mono text-sm text-ink-1 break-all">{value}</dd>
    </div>
  );
}

function formatTimeAgo(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffMs = now - then;
  const diffMins = Math.floor(diffMs / (1000 * 60));
  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHours = Math.floor(diffMins / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d ago`;
  return new Date(timestamp).toLocaleDateString();
}
