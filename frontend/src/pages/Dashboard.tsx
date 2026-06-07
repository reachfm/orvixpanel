/**
 * Dashboard. Real data only:
 *   - healthz + readyz
 *   - /api/v1/admin/system
 *   - /api/v1/admin/license
 *   - account count from /api/v1/accounts
 *
 * No placeholder metrics, no fake charts. Counts are derived from
 * the real lists; if the lists are empty, the page shows 0 and a
 * "Get started" CTA. The license card surfaces tier / days
 * remaining / status with the same shape as the renewal-info API.
 */

import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner } from "@/lib/ui/Feedback";
import { systemKeys, accountKeys } from "@/lib/query/keys";
import {
  healthz, readyz, systemInfo, licenseRenewal,
} from "@/lib/api/system";
import { listAccounts } from "@/lib/api/accounts";
import { useAuthStore } from "@/lib/auth/store";

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  const hz = useQuery({ queryKey: systemKeys.healthz(),  queryFn: healthz,  refetchInterval: 15_000 });
  const rz = useQuery({ queryKey: systemKeys.readyz(),   queryFn: readyz,   refetchInterval: 15_000 });
  const sys = useQuery({ queryKey: systemKeys.info(),     queryFn: systemInfo });
  const lic = useQuery({ queryKey: systemKeys.licenseRenewal(), queryFn: licenseRenewal });
  const accts = useQuery({ queryKey: accountKeys.list(), queryFn: listAccounts });

  const accountCount = accts.data?.accounts.length ?? 0;
  const activeCount  = accts.data?.accounts.filter((a) => a.status === "active").length ?? 0;

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Welcome${user?.email ? `, ${user.email}` : ""}`}
        description="Operational overview of your OrvixPanel instance."
      />

      {/* Top row — 4 status tiles */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card>
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs uppercase tracking-wider text-ink-3">Healthz</div>
              <div className="mt-1 text-2xl font-semibold text-ink-1">
                {hz.isLoading ? <Spinner size={18} /> : hz.data?.status === "ok" ? "ok" : "down"}
              </div>
            </div>
            <StatusPill tone={hz.data?.status === "ok" ? "success" : "danger"}>HTTP</StatusPill>
          </div>
        </Card>

        <Card>
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs uppercase tracking-wider text-ink-3">Readyz (DB)</div>
              <div className="mt-1 text-2xl font-semibold text-ink-1">
                {rz.isLoading ? <Spinner size={18} /> : rz.data?.status ?? "—"}
              </div>
            </div>
            <StatusPill tone={rz.data?.status === "ready" ? "success" : "danger"}>DB</StatusPill>
          </div>
        </Card>

        <Card>
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs uppercase tracking-wider text-ink-3">Accounts</div>
              <div className="mt-1 text-2xl font-semibold text-ink-1">
                {accts.isLoading ? <Spinner size={18} /> : accountCount}
              </div>
              <div className="text-[11px] text-ink-3">{activeCount} active</div>
            </div>
            <Link to="/accounts" className="text-xs text-brand-600 hover:underline">View →</Link>
          </div>
        </Card>

        <Card>
          <div className="flex items-start justify-between">
            <div>
              <div className="text-xs uppercase tracking-wider text-ink-3">License</div>
              <div className="mt-1 text-2xl font-semibold text-ink-1">
                {lic.isLoading ? <Spinner size={18} /> : lic.data ? lic.data.tier.toUpperCase() : "—"}
              </div>
              {lic.data && (
                <div className="text-[11px] text-ink-3">
                  {lic.data.days_remaining} day{lic.data.days_remaining === 1 ? "" : "s"} remaining
                </div>
              )}
            </div>
            {lic.data && (
              <StatusPill tone={lic.data.status === "active" ? "success" : lic.data.status === "grace" ? "warning" : "danger"}>
                {lic.data.status}
              </StatusPill>
            )}
          </div>
        </Card>
      </div>

      {/* Second row — system + get started */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader
            title="System"
            description="Build metadata returned by /api/v1/admin/system."
            actions={
              <StatusPill tone={sys.isError ? "danger" : sys.data ? "success" : "neutral"}>
                {sys.isError ? "error" : sys.isLoading ? "loading" : "ok"}
              </StatusPill>
            }
          />
          {sys.data ? (
            <dl className="grid grid-cols-3 gap-3 text-sm">
              <Field label="Name" value={sys.data.name} />
              <Field label="Version" value={sys.data.version} />
              <Field label="Uptime at" value={new Date(sys.data.uptime_at).toLocaleString()} />
            </dl>
          ) : (
            <div className="text-sm text-ink-3">—</div>
          )}
        </Card>

        <Card>
          <CardHeader title="Get started" />
          {accountCount === 0 ? (
            <div className="space-y-3 text-sm text-ink-2">
              <p>You have no accounts yet. Create one to start serving sites.</p>
              <Link
                to="/accounts/new"
                className="inline-flex h-9 items-center rounded-md bg-brand-600 px-3.5 text-sm font-medium text-white hover:bg-brand-700"
              >
                Create your first account
              </Link>
            </div>
          ) : (
            <div className="space-y-3 text-sm text-ink-2">
              <p>You have {accountCount} account{accountCount === 1 ? "" : "s"}. Manage them from the Accounts page.</p>
              <Link
                to="/accounts"
                className="inline-flex h-9 items-center rounded-md border border-surface-border bg-surface-1 px-3.5 text-sm font-medium text-ink-1 hover:bg-surface-2"
              >
                Open Accounts
              </Link>
            </div>
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
