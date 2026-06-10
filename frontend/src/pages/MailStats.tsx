/**
 * Mail Statistics Page — Enterprise mail system overview.
 * Shows domain counts, mailbox counts, quota usage, and storage metrics.
 */

import { useQuery } from "@tanstack/react-query";
import { getMailStats } from "@/lib/api/mail";
import { Card } from "@/lib/ui/Card";
import { Badge } from "@/lib/ui/Badge";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";

interface MetricCardProps {
  title: string;
  value: number | string;
  subtitle?: string;
  tone?: "neutral" | "success" | "warning" | "danger";
  icon?: React.ReactNode;
}

function MetricCard({ title, value, subtitle, tone = "neutral", icon }: MetricCardProps) {
  const toneClasses = {
    neutral: "text-ink-1",
    success: "text-success",
    warning: "text-warning",
    danger: "text-danger",
  };

  return (
    <Card>
      <div className="flex items-start justify-between">
        <div>
          <div className="text-sm text-ink-3 font-medium">{title}</div>
          <div className={`text-3xl font-bold mt-1 ${toneClasses[tone]}`}>{value}</div>
          {subtitle && <div className="text-xs text-ink-2 mt-1">{subtitle}</div>}
        </div>
        {icon && <div className="text-ink-3">{icon}</div>}
      </div>
    </Card>
  );
}

interface QuotaBarProps {
  used: number;
  total: number;
  label: string;
}

function QuotaBar({ used, total, label }: QuotaBarProps) {
  const percent = total > 0 ? Math.min((used / total) * 100, 100) : 0;
  const tone = percent >= 90 ? "danger" : percent >= 70 ? "warning" : "success";

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <span className="text-ink-2">{label}</span>
        <span className="text-ink-1 font-mono">
          {used.toLocaleString()} / {total.toLocaleString()} MB
        </span>
      </div>
      <div className="h-2 rounded-full bg-surface-2 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${
            tone === "danger" ? "bg-danger" : tone === "warning" ? "bg-warning" : "bg-success"
          }`}
          style={{ width: `${percent}%` }}
        />
      </div>
      <div className="text-xs text-ink-3 text-right">{percent.toFixed(1)}% used</div>
    </div>
  );
}

export function MailStatsPage() {
  // Query mail stats
  const { data: stats, isLoading, error, refetch } = useQuery({
    queryKey: ["mail", "stats"],
    queryFn: getMailStats,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Spinner size={32} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Mail Statistics"
          description="Overview of mail system usage and capacity"
        />
        <Card>
          <ErrorState
            description="Failed to load mail statistics."
            onRetry={() => refetch()}
          />
        </Card>
      </div>
    );
  }

  const totalQuotaGB = (stats?.total_quota_mb ?? 0) / 1024;
  const usedQuotaGB = (stats?.used_quota_mb ?? 0) / 1024;
  const availableQuotaGB = totalQuotaGB - usedQuotaGB;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Mail Statistics"
        description="Overview of mail system usage and capacity"
      />

      {/* Domain Metrics */}
      <div>
        <h2 className="text-sm font-semibold text-ink-3 uppercase tracking-wide mb-3">Domains</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            title="Total Domains"
            value={stats?.total_domains ?? 0}
            subtitle="configured"
            tone="neutral"
          />
          <MetricCard
            title="Active Domains"
            value={stats?.active_domains ?? 0}
            subtitle="ready for mail"
            tone="success"
          />
          <MetricCard
            title="Inactive"
            value={(stats?.total_domains ?? 0) - (stats?.active_domains ?? 0)}
            subtitle="pending or suspended"
            tone="warning"
          />
          <MetricCard
            title="Total Aliases"
            value={stats?.total_aliases ?? 0}
            subtitle="email aliases"
            tone="neutral"
          />
        </div>
      </div>

      {/* Mailbox Metrics */}
      <div>
        <h2 className="text-sm font-semibold text-ink-3 uppercase tracking-wide mb-3">Mailboxes</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            title="Total Mailboxes"
            value={stats?.total_mailboxes ?? 0}
            subtitle="configured"
            tone="neutral"
          />
          <MetricCard
            title="Active"
            value={(stats?.total_mailboxes ?? 0) - (stats?.suspended_mailboxes ?? 0)}
            subtitle="operational"
            tone="success"
          />
          <MetricCard
            title="Suspended"
            value={stats?.suspended_mailboxes ?? 0}
            subtitle="disabled"
            tone="warning"
          />
          <MetricCard
            title="Forwarders"
            value={stats?.total_forwarders ?? 0}
            subtitle="email forwarders"
            tone="neutral"
          />
        </div>
      </div>

      {/* Quota Usage */}
      <Card>
        <h2 className="text-lg font-semibold text-ink-1 mb-4">Quota Usage</h2>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <QuotaBar
              used={stats?.used_quota_mb ?? 0}
              total={stats?.total_quota_mb ?? 0}
              label="Total Quota Allocation"
            />
            <div className="mt-6 grid grid-cols-3 gap-4">
              <div className="text-center p-3 rounded-lg bg-surface-2">
                <div className="text-2xl font-bold text-ink-1">{usedQuotaGB.toFixed(1)}</div>
                <div className="text-xs text-ink-3">GB Used</div>
              </div>
              <div className="text-center p-3 rounded-lg bg-surface-2">
                <div className="text-2xl font-bold text-ink-1">{availableQuotaGB.toFixed(1)}</div>
                <div className="text-xs text-ink-3">GB Available</div>
              </div>
              <div className="text-center p-3 rounded-lg bg-surface-2">
                <div className="text-2xl font-bold text-ink-1">{totalQuotaGB.toFixed(1)}</div>
                <div className="text-xs text-ink-3">GB Total</div>
              </div>
            </div>
          </div>
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-ink-2">Quick Stats</h3>
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-ink-2">Avg. Quota per Mailbox</span>
                <Badge tone="neutral">
                  {stats?.total_mailboxes && stats.total_mailboxes > 0
                    ? Math.round((stats.total_quota_mb || 0) / stats.total_mailboxes)
                    : 0} MB
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-ink-2">Avg. Usage per Mailbox</span>
                <Badge tone="neutral">
                  {stats?.total_mailboxes && stats.total_mailboxes > 0
                    ? Math.round((stats.used_quota_mb || 0) / stats.total_mailboxes)
                    : 0} MB
                </Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-ink-2">Overall Utilization</span>
                <Badge
                  tone={
                    stats?.total_quota_mb && stats.total_quota_mb > 0
                      ? (stats.used_quota_mb / stats.total_quota_mb) >= 0.9
                        ? "danger"
                        : (stats.used_quota_mb / stats.total_quota_mb) >= 0.7
                        ? "warning"
                        : "success"
                      : "neutral"
                  }
                >
                  {stats?.total_quota_mb && stats.total_quota_mb > 0
                    ? ((stats.used_quota_mb / stats.total_quota_mb) * 100).toFixed(1)
                    : 0}%
                </Badge>
              </div>
            </div>
          </div>
        </div>
      </Card>

      {/* System Status */}
      <Card>
        <h2 className="text-lg font-semibold text-ink-1 mb-4">Service Status</h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between py-2 border-b border-surface-2">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 rounded-full bg-success" />
              <span className="text-sm text-ink-1">Database</span>
            </div>
            <Badge tone="success">Connected</Badge>
          </div>
          <div className="flex items-center justify-between py-2 border-b border-surface-2">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 rounded-full bg-warning" />
              <span className="text-sm text-ink-1">Postfix</span>
            </div>
            <Badge tone="warning">Requires VPS Setup</Badge>
          </div>
          <div className="flex items-center justify-between py-2 border-b border-surface-2">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 rounded-full bg-warning" />
              <span className="text-sm text-ink-1">Dovecot</span>
            </div>
            <Badge tone="warning">Requires VPS Setup</Badge>
          </div>
          <div className="flex items-center justify-between py-2">
            <div className="flex items-center gap-3">
              <div className="w-3 h-3 rounded-full bg-warning" />
              <span className="text-sm text-ink-1">OpenDKIM</span>
            </div>
            <Badge tone="warning">Requires VPS Setup</Badge>
          </div>
        </div>
      </Card>

      {/* Last Updated */}
      <div className="text-center text-xs text-ink-3">
        Last updated: {stats?.generated_at ? new Date(stats.generated_at).toLocaleString() : "—"}
      </div>
    </div>
  );
}