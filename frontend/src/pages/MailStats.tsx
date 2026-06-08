/**
 * Mail Statistics Page
 */

import { useQuery } from "@tanstack/react-query";
import { getMailStats, getQuotaStats } from "@/lib/api/mail";
import { Card } from "@/lib/ui/Card";
import { Spinner } from "@/lib/ui/Feedback";

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: string;
}

function StatCard({ title, value, subtitle, color = "text-primary-600" }: StatCardProps) {
  return (
    <Card>
      <div className="text-sm text-gray-500">{title}</div>
      <div className={`text-3xl font-bold ${color}`}>{value}</div>
      {subtitle && <div className="text-xs text-gray-400 mt-1">{subtitle}</div>}
    </Card>
  );
}

export function MailStatsPage() {
  // Query mail stats
  const { data: stats, isLoading: statsLoading, error: statsError } = useQuery({
    queryKey: ["mail", "stats"],
    queryFn: getMailStats,
  });

  // Query quota stats
  const { data: quotaStats } = useQuery({
    queryKey: ["mail", "quota", "stats"],
    queryFn: getQuotaStats,
  });

  if (statsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner />
      </div>
    );
  }

  if (statsError) {
    return (
      <div className="text-center text-red-500">
        Failed to load mail statistics. Please try again.
      </div>
    );
  }

  const storageUsedGB = stats ? (stats.storage_used_bytes / (1024 * 1024 * 1024)).toFixed(2) : "0";
  const storageAvailableGB = stats ? (stats.storage_available_bytes / (1024 * 1024 * 1024)).toFixed(2) : "0";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Mail Statistics</h1>
        <p className="text-gray-500">
          Overview of mail system usage and capacity
        </p>
      </div>

      {/* Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Domains"
          value={stats?.total_domains || 0}
          subtitle="configured domains"
        />
        <StatCard
          title="Total Mailboxes"
          value={stats?.total_mailboxes || 0}
          subtitle="active mailboxes"
        />
        <StatCard
          title="Total Aliases"
          value={stats?.total_aliases || 0}
          subtitle="email aliases"
        />
        <StatCard
          title="Total Forwarders"
          value={stats?.total_forwarders || 0}
          subtitle="email forwarders"
        />
      </div>

      {/* Storage Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard
          title="Storage Used"
          value={`${storageUsedGB} GB`}
          color="text-blue-600"
        />
        <StatCard
          title="Storage Available"
          value={`${storageAvailableGB} GB`}
          color="text-green-600"
        />
        <StatCard
          title="Active Today"
          value={stats?.active_today || 0}
          subtitle="mailboxes with activity"
        />
      </div>

      {/* Quota Distribution */}
      <Card>
        <h2 className="text-lg font-semibold mb-4">Quota Usage Distribution</h2>
        {quotaStats && quotaStats.summary ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Healthy (0-50%)</span>
              <span className="text-2xl font-bold text-green-600">
                {quotaStats.summary.healthy_count}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Warning (50-80%)</span>
              <span className="text-2xl font-bold text-yellow-600">
                {quotaStats.summary.warning_count}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Critical (80-100%)</span>
              <span className="text-2xl font-bold text-red-600">
                {quotaStats.summary.critical_count}
              </span>
            </div>
            <div className="pt-4 border-t">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">Total Quota Allocated</span>
                <span className="text-xl font-semibold">
                  {(quotaStats.summary.total_quota_bytes / (1024 * 1024 * 1024)).toFixed(1)} GB
                </span>
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center text-gray-500 py-8">
            No quota data available
          </div>
        )}
      </Card>

      {/* Activity Chart Placeholder */}
      <Card>
        <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
        <div className="h-48 flex items-center justify-center text-gray-400">
          <div className="text-center">
            <svg className="w-16 h-16 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            <p>Activity chart requires VPS integration</p>
            <p className="text-sm">Run smoke tests to enable real data</p>
          </div>
        </div>
      </Card>

      {/* System Status */}
      <Card>
        <h2 className="text-lg font-semibold mb-4">Service Status</h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-green-500" />
              <span className="text-sm">Database</span>
            </div>
            <span className="text-sm text-green-600">Connected</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-yellow-500" />
              <span className="text-sm">Postfix</span>
            </div>
            <span className="text-sm text-yellow-600">Requires VPS Setup</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-yellow-500" />
              <span className="text-sm">Dovecot</span>
            </div>
            <span className="text-sm text-yellow-600">Requires VPS Setup</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-yellow-500" />
              <span className="text-sm">OpenDKIM</span>
            </div>
            <span className="text-sm text-yellow-600">Requires VPS Setup</span>
          </div>
        </div>
      </Card>
    </div>
  );
}