/**
 * System Health page. Professional status page showing all system metrics.
 * Polls the public health probes + admin endpoints. No fake data.
 * v0.3.1 Phase D: Enhanced with detailed health checks, remediation, and alert management.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { systemKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";
import { healthz, readyz, systemInfo, license, licenseRenewal, systemHealth } from "@/lib/api/system";

interface HealthMetric {
  name: string;
  status: "healthy" | "degraded" | "down" | "unknown";
  endpoint: string;
  description: string;
  value?: string;
  details?: unknown;
}

interface HealthCheck {
  name: string;
  status: "pass" | "warn" | "fail" | "skip" | "unknown";
  message: string;
  details?: string;
  suggestions?: string[];
}

export function SystemHealthPage() {
  const hz = useQuery({ queryKey: systemKeys.healthz(), queryFn: healthz, refetchInterval: 5_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(), queryFn: readyz, refetchInterval: 5_000 });
  const sys = useQuery({ queryKey: systemKeys.info(), queryFn: systemInfo, refetchInterval: 30_000 });
  const lic = useQuery({ queryKey: systemKeys.license(), queryFn: license });
  const ren = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal, refetchInterval: 30_000 });
  const health = useQuery({ queryKey: systemKeys.health(), queryFn: systemHealth, refetchInterval: 30_000 });

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
      details: rz.data as unknown,
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

  // Group health checks by status
  const healthChecks = health.data?.checks ?? [];
  const failedChecks = healthChecks.filter(c => c.status === "fail");
  const warnChecks = healthChecks.filter(c => c.status === "warn");
  const passedChecks = healthChecks.filter(c => c.status === "pass");

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Health"
        description="Live status of every layer the panel relies on. All values are real responses from the matching endpoint."
        actions={
          <Button variant="secondary" size="sm" onClick={() => health.refetch()}>
            Refresh
          </Button>
        }
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
            {typeof metric.details === 'object' && metric.details !== null && (
              <pre className="mt-3 overflow-x-auto rounded bg-surface-2 p-2 text-xs font-mono text-ink-1">
                {JSON.stringify(metric.details, null, 2)}
              </pre>
            )}
          </Card>
        ))}
      </div>

      {/* System Health Checks Section */}
      {health.isLoading ? (
        <Card>
          <CardHeader title="System Health Checks" description="Running diagnostic checks..." />
          <div className="flex items-center justify-center py-8">
            <Spinner size={24} />
            <span className="ml-3 text-sm text-ink-2">Running checks...</span>
          </div>
        </Card>
      ) : health.data ? (
        <>
          {/* Failed Checks - High Priority */}
          {failedChecks.length > 0 && (
            <Card className="border-danger/30 bg-danger/5">
              <CardHeader
                title="Failed Checks"
                description={`${failedChecks.length} check${failedChecks.length === 1 ? "" : "s"} require immediate attention`}
                actions={
                  <StatusPill tone="danger">{failedChecks.length} Failed</StatusPill>
                }
              />
              <div className="space-y-3">
                {failedChecks.map((check, i) => (
                  <HealthCheckCard key={i} check={check} />
                ))}
              </div>
            </Card>
          )}

          {/* Warning Checks */}
          {warnChecks.length > 0 && (
            <Card className="border-amber-500/30 bg-amber-500/5">
              <CardHeader
                title="Warning Checks"
                description={`${warnChecks.length} check${warnChecks.length === 1 ? "" : "s"} need attention`}
                actions={
                  <StatusPill tone="warning">{warnChecks.length} Warnings</StatusPill>
                }
              />
              <div className="space-y-3">
                {warnChecks.map((check, i) => (
                  <HealthCheckCard key={i} check={check} />
                ))}
              </div>
            </Card>
          )}

          {/* Passed Checks Summary */}
          {passedChecks.length > 0 && (
            <Card>
              <CardHeader
                title="Passed Checks"
                description={`${passedChecks.length} check${passedChecks.length === 1 ? "" : "s"} passed successfully`}
                actions={
                  <StatusPill tone="success">{passedChecks.length} Passed</StatusPill>
                }
              />
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                {passedChecks.map((check, i) => (
                  <div key={i} className="flex items-center gap-2 rounded-md bg-success/5 px-3 py-2">
                    <svg className="h-4 w-4 text-success" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                    <span className="text-sm font-medium text-ink-1">{check.name}</span>
                  </div>
                ))}
              </div>
            </Card>
          )}

          {/* All Checks Summary */}
          {healthChecks.length > 0 && (
            <Card>
              <CardHeader title="All Checks Summary" />
              <div className="flex items-center gap-6">
                <div className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-success" />
                  <span className="text-sm text-ink-2">Passed: {passedChecks.length}</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-amber-500" />
                  <span className="text-sm text-ink-2">Warnings: {warnChecks.length}</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-danger" />
                  <span className="text-sm text-ink-2">Failed: {failedChecks.length}</span>
                </div>
              </div>
            </Card>
          )}
        </>
      ) : null}

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
              <Field label="Uptime at" value={formatDate(sys.data.uptime_at)} />
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
              <Field label="Expires" value={lic.data?.expires_at ? formatDate(new Date(lic.data.expires_at * 1000).toISOString()) : "—"} />
            </dl>
          )}
        </Card>
      </div>
    </div>
  );
}

// Health check card component with remediation suggestions
function HealthCheckCard({ check }: { check: HealthCheck }) {
  const tone = check.status === "fail" ? "danger" : "warning";
  const toneClasses = check.status === "fail"
    ? "border-danger/30 bg-danger/5"
    : "border-amber-500/30 bg-amber-500/5";

  return (
    <div className={`rounded-md border p-4 ${toneClasses}`}>
      <div className="flex items-start gap-3">
        <StatusPill tone={tone} className="shrink-0">
          {check.status}
        </StatusPill>
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm text-ink-1">{check.name}</div>
          <div className="mt-1 text-xs text-ink-3">{check.message}</div>
          {check.details && (
            <div className="mt-2 rounded bg-surface-2 p-2">
              <pre className="text-xs font-mono text-ink-2 overflow-x-auto">{check.details}</pre>
            </div>
          )}
          {check.suggestions && check.suggestions.length > 0 && (
            <div className="mt-3">
              <div className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">Suggested Actions</div>
              <ul className="mt-1 space-y-1">
                {check.suggestions.map((suggestion, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-ink-2">
                    <svg className="mt-0.5 h-3.5 w-3.5 shrink-0 text-brand-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span>{suggestion}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
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