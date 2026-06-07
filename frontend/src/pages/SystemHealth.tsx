/**
 * System Health page. Polls the public health probes + the admin
 * endpoints so an operator can see whether the panel is responding
 * at every layer. No fake gauges — every tile is the literal value
 * returned by the matching endpoint.
 */

import { useQuery } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { systemKeys } from "@/lib/query/keys";
import { healthz, readyz, systemInfo, license, licenseRenewal } from "@/lib/api/system";

export function SystemHealthPage() {
  const hz = useQuery({ queryKey: systemKeys.healthz(), queryFn: healthz, refetchInterval: 5_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(),  queryFn: readyz,  refetchInterval: 5_000 });
  const sys = useQuery({ queryKey: systemKeys.info(),    queryFn: systemInfo, refetchInterval: 30_000 });
  const lic = useQuery({ queryKey: systemKeys.license(), queryFn: license });
  const ren = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal, refetchInterval: 30_000 });

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Health"
        description="Live status of every layer the panel relies on. All values are real responses from the matching endpoint."
      />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card>
          <CardHeader
            title="Liveness"
            description="GET /healthz"
            actions={
              <StatusPill tone={hz.data?.status === "ok" ? "success" : "danger"}>
                {hz.isLoading ? "checking…" : hz.data?.status ?? "down"}
              </StatusPill>
            }
          />
          <pre className="overflow-x-auto rounded-md bg-surface-2 p-3 text-xs font-mono text-ink-1">
            {hz.data ? JSON.stringify(hz.data, null, 2) : (hz.isLoading ? "loading…" : (hz.error?.message ?? "—"))}
          </pre>
        </Card>

        <Card>
          <CardHeader
            title="Readiness"
            description="GET /readyz — includes a DB ping"
            actions={
              <StatusPill tone={rz.data?.status === "ready" ? "success" : "danger"}>
                {rz.isLoading ? "checking…" : rz.data?.status ?? "down"}
              </StatusPill>
            }
          />
          <pre className="overflow-x-auto rounded-md bg-surface-2 p-3 text-xs font-mono text-ink-1">
            {rz.data ? JSON.stringify(rz.data, null, 2) : (rz.isLoading ? "loading…" : (rz.error?.message ?? "—"))}
          </pre>
        </Card>

        <Card>
          <CardHeader title="Build" description="GET /api/v1/admin/system" />
          {sys.isLoading ? <Spinner size={18} /> : sys.data ? (
            <dl className="grid grid-cols-2 gap-3 text-sm">
              <Field label="Name"    value={sys.data.name} />
              <Field label="Version" value={sys.data.version} />
              <Field label="Uptime at" value={new Date(sys.data.uptime_at).toLocaleString()} />
            </dl>
          ) : <ErrorState title="Failed to load" onRetry={() => sys.refetch()} />}
        </Card>

        <Card>
          <CardHeader
            title="License"
            description="GET /api/v1/admin/license + /renewal-info"
            actions={ren.data && (
              <StatusPill tone={ren.data.status === "active" ? "success" : ren.data.status === "grace" ? "warning" : "danger"}>
                {ren.data.status}
              </StatusPill>
            )}
          />
          {lic.isLoading || ren.isLoading ? <Spinner size={18} /> : (
            <dl className="grid grid-cols-2 gap-3 text-sm">
              <Field label="Tier" value={lic.data?.tier ?? "—"} />
              <Field label="Features" value={String(lic.data?.features?.length ?? 0)} />
              <Field label="Max servers" value={String(lic.data?.max_servers ?? "—")} />
              <Field label="Days remaining" value={ren.data ? String(ren.data.days_remaining) : "—"} />
              <Field label="Grace days" value={ren.data ? String(ren.data.grace_days) : "—"} />
              <Field label="Expires" value={lic.data?.expires_at ? new Date(lic.data.expires_at * 1000).toLocaleString() : "—"} />
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
