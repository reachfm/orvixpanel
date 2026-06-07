/**
 * System Health page. Professional status page showing all system metrics.
 * Polls the public health probes + admin endpoints. No fake data.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { systemKeys } from "@/lib/query/keys";
import { healthz, readyz, systemInfo, license, licenseRenewal } from "@/lib/api/system";

interface HealthMetric {
  name: string;
  status: "healthy" | "degraded" | "down" | "unknown";
  endpoint: string;
  description: string;
  value?: string;
  details?: Record<string, unknown>;
}

export function SystemHealthPage() {
  const hz = useQuery({ queryKey: systemKeys.healthz(), queryFn: healthz, refetchInterval: 5_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(), queryFn: readyz, refetchInterval: 5_000 });
  const sys = useQuery({ queryKey: systemKeys.info(), queryFn: systemInfo, refetchInterval: 30_000 });
  const lic = useQuery({ queryKey: systemKeys.license(), queryFn: license });
  const ren = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal, refetchInterval: 30_000 });

  // Build health metrics
  const metrics = useMemo<HealthMetric[]>(() => {
    const result: HealthMetric[] = [];

    // API Health
    result.push({
      name: "API Health",
      status: hz.isLoading ? "unknown" : hz.data?.status === "ok" ? "healthy" : "down",
      endpoint: "GET /healthz",
      description: "HTTP liveness probe",
      value: hz.isLoading ? "checking..." : hz.data?.status ?? "down",
    });

    // Database
    result.push({
      name: "Database",
      status: rz.isLoading ? "unknown" : rz.data?.status === "ready" ? "healthy" : "degraded",
      endpoint: "GET /readyz",
      description: "Database connectivity and migrations",
      value: rz.isLoading ? "checking..." : rz.data?.status ?? "down",
      details: rz.data,
    });

    // License
    if (ren.data) {
      result.push({
        name: "License",
        status: ren.data.status === "active" ? "healthy" : ren.data.status === "grace" ? "degraded" : "down",
        endpoint: "GET /api/v1/admin/license/renewal-info",
        description: `Tier: ${ren.data.tier}`,
        value: `${ren.data.status} (${ren.data.days_remaining}d)`,
      });
    } else if (!ren.isLoading) {
      result.push({
        name: "License",
        status: "unknown",
        endpoint: "GET /api/v1/admin/license/renewal-info",
        description: "Unable to fetch license information",
        value: "unknown",
      });
    }

    // System Build
    if (sys.data) {
      result.push({
        name: "System Build",
        status: "healthy",
        endpoint: "GET /api/v1/admin/system",
        description: `Version ${sys.data.version}`,
        value: sys.data.name,
      });
    } else if (!sys.isLoading) {
      result.push({
        name: "System Build",
        status: "unknown",
        endpoint: "GET /api/v1/admin/system",
        description: "Unable to fetch system information",
        value: "unknown",
      });
    }

    return result;
  }, [hz, rz, sys, lic, ren]);

  const overallStatus = useMemo(() => {
    if (metrics.some((m) => m.status === "down")) return "down";
    if (metrics.some((m) => m.status === "degraded")) return "degraded";
    if (metrics.every((m) => m.status === "healthy")) return "healthy";
    return "unknown";
  }, [metrics]);

  const statusTone = (status: HealthMetric["status"]): "success" | "warning" | "danger" | "neutral" => {
    switch (status) {
      case "healthy": return "success";
      case "degraded": return "warning";
      case "down": return "danger";
      default: return "neutral";
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Health"
        description="Live status of every layer the panel relies on. All values are real responses from the matching endpoint."
      />

      {/* Overall status banner */}
      <div className={`rounded-lg border p-4 ${
        overallStatus === "healthy" ? "border-success/30 bg-success/5" :
        overallStatus === "degraded" ? "border-warning/30 bg-warning/5" :
        overallStatus === "down" ? "border-danger/30 bg-danger/5" :
        "border-surface-border bg-surface-1"
      }`}>
        <div className="flex items-center gap-3">
          <StatusPill tone={statusTone(overallStatus)} className="text-sm">
            {overallStatus === "healthy" ? "All Systems Operational" :
             overallStatus === "degraded" ? "Degraded Performance" :
             overallStatus === "down" ? "System Down" : "Checking..."}
          </StatusPill>
          <span className="text-sm text-ink-2">
            {metrics.filter((m) => m.status === "healthy").length} of {metrics.length} services healthy
          </span>
        </div>
      </div>

      {/* Health metrics grid */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {metrics.map((metric) => (
          <Card key={metric.name}>
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-medium text-ink-1">{metric.name}</h3>
                  <StatusPill tone={statusTone(metric.status)} dot>
                    {metric.status}
                  </StatusPill>
                </div>
                <p className="mt-1 text-xs text-ink-3">{metric.description}</p>
                <p className="mt-1 font-mono text-xs text-ink-2">{metric.endpoint}</p>
              </div>
            </div>
            {metric.details && (
              <pre className="mt-3 overflow-x-auto rounded bg-surface-2 p-2 text-xs font-mono text-ink-1">
                {JSON.stringify(metric.details, null, 2)}
              </pre>
            )}
          </Card>
        ))}
      </div>

      {/* Detailed info section */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader
            title="API Health Check"
            description="GET /healthz — HTTP liveness probe"
            actions={
              <StatusPill tone={hz.data?.status === "ok" ? "success" : "danger"}>
                {hz.data?.status ?? "down"}
              </StatusPill>
            }
          />
          <pre className="overflow-x-auto rounded-md bg-surface-2 p-3 text-xs font-mono text-ink-1">
            {hz.data ? JSON.stringify(hz.data, null, 2) : hz.isLoading ? "checking..." : hz.error?.message ?? "—"}
          </pre>
        </Card>

        <Card>
          <CardHeader
            title="Database Readiness"
            description="GET /readyz — Database ping and migration status"
            actions={
              <StatusPill tone={rz.data?.status === "ready" ? "success" : "danger"}>
                {rz.data?.status ?? "down"}
              </StatusPill>
            }
          />
          <pre className="overflow-x-auto rounded-md bg-surface-2 p-3 text-xs font-mono text-ink-1">
            {rz.data ? JSON.stringify(rz.data, null, 2) : rz.isLoading ? "checking..." : rz.error?.message ?? "—"}
          </pre>
        </Card>

        <Card>
          <CardHeader
            title="System Information"
            description="GET /api/v1/admin/system"
            actions={
              <StatusPill tone={sys.data ? "success" : "neutral"}>
                {sys.data ? "ok" : sys.isError ? "error" : "loading"}
              </StatusPill>
            }
          />
          {sys.data ? (
            <dl className="grid grid-cols-2 gap-3 text-sm">
              <Field label="Name" value={sys.data.name} />
              <Field label="Version" value={sys.data.version} />
              <Field label="Uptime at" value={new Date(sys.data.uptime_at).toLocaleString()} />
              <Field label="Started" value={formatUptime(sys.data.uptime_at)} />
            </dl>
          ) : sys.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Spinner size={18} />
            </div>
          ) : (
            <ErrorState title="Failed to load" onRetry={() => sys.refetch()} />
          )}
        </Card>

        <Card>
          <CardHeader
            title="License Status"
            description="GET /api/v1/admin/license + /renewal-info"
            actions={ren.data && (
              <StatusPill tone={ren.data.status === "active" ? "success" : ren.data.status === "grace" ? "warning" : "danger"}>
                {ren.data.status}
              </StatusPill>
            )}
          />
          {lic.isLoading || ren.isLoading ? (
            <div className="flex items-center justify-center py-6">
              <Spinner size={18} />
            </div>
          ) : (
            <dl className="grid grid-cols-2 gap-3 text-sm">
              <Field label="Tier" value={lic.data?.tier ?? "—"} />
              <Field label="Features" value={String(lic.data?.features?.length ?? 0)} />
              <Field label="Max servers" value={String(lic.data?.max_servers ?? "—")} />
              <Field label="Days remaining" value={ren.data ? String(ren.data.days_remaining) : "—"} />
              <Field label="Grace days" value={ren.data ? String(ren.data.grace_days) : "—"} />
              <Field label="Expires" value={lic.data?.expires_at ? new Date(lic.data.expires_at * 1000).toLocaleDateString() : "—"} />
            </dl>
          )}
        </Card>
      </div>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className="mt-0.5 font-mono text-sm text-ink-1 break-all">{value}</dd>
    </div>
  );
}

function formatUptime(uptimeAt: string): string {
  const start = new Date(uptimeAt).getTime();
  const diffMs = Date.now() - start;
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffDays > 0) return `${diffDays}d ${diffHours % 24}h`;
  if (diffHours > 0) return `${diffHours}h ${diffMinutes % 60}m`;
  if (diffMinutes > 0) return `${diffMinutes}m`;
  return `${diffSeconds}s`;
}