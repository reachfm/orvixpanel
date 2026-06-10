/**
 * Account detail. Tabs:
 *   - Overview   : real fields from /api/v1/accounts/:id (live disk
 *                  usage is recomputed on the server on each read)
 *   - Domains    : child route component, lists the account's domains
 *                  via /api/v1/accounts/:id/domains
 *   - Deployments: real list via the v0.2.3 /deployments endpoint
 *   - Usage      : raw /api/v1/accounts/:id/usage (the fields are
 *                  open-ended; we just dump them as a JSON panel)
 * v0.3.1 Phase B: Enhanced with disk usage visualization and quick actions.
 */

import { useState } from "react";
import { useParams, Link, useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Modal } from "@/lib/ui/Modal";
import { Tabs } from "@/lib/ui/Tabs";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { Table, type Column } from "@/lib/ui/Table";
import { accountKeys, domainKeys, deploymentKeys } from "@/lib/query/keys";
import { formatDate, formatBytes } from "@/lib/utils";
import { getAccount, accountUsage, suspendAccount, unsuspendAccount, deleteAccount } from "@/lib/api/accounts";
import { listDomains, deleteDomain, type Domain } from "@/lib/api/domains";
import { listDeployments, type Deployment } from "@/lib/api/deployments";

export function AccountDetailPage() {
  const { id } = useParams({ strict: false }) as { id: string };
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [tab, setTab] = useState("overview");
  const [deleteModal, setDeleteModal] = useState(false);

  const q = useQuery({ queryKey: accountKeys.detail(id), queryFn: () => getAccount(id) });
  const doms = useQuery({ queryKey: domainKeys.byAccount(id), queryFn: () => listDomains(id) });
  const deps = useQuery({ queryKey: deploymentKeys.byAccount(id), queryFn: () => listDeployments(id) });
  const usage = useQuery({ queryKey: accountKeys.usage(id), queryFn: () => accountUsage(id) });

  const suspend = useMutation({
    mutationFn: () => suspendAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: accountKeys.all() }),
  });
  const unsuspend = useMutation({
    mutationFn: () => unsuspendAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: accountKeys.all() }),
  });
  const del = useMutation({
    mutationFn: () => deleteAccount(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: accountKeys.all() });
      qc.invalidateQueries({ queryKey: domainKeys.all() });
      navigate({ to: "/accounts" });
    },
  });

  const isPending = suspend.isPending || unsuspend.isPending || del.isPending;

  if (q.isLoading) {
    return <div className="flex min-h-[40vh] items-center justify-center"><Spinner size={28} /></div>;
  }
  if (q.isError || !q.data) {
    return <ErrorState description="Failed to load account." onRetry={() => q.refetch()} />;
  }

  const a = q.data;
  const actions = (
    <>
      {a.status === "active" ? (
        <Button variant="secondary" loading={suspend.isPending} onClick={() => suspend.mutate()}>
          Suspend
        </Button>
      ) : (
        <Button variant="secondary" loading={unsuspend.isPending} onClick={() => unsuspend.mutate()}>
          Unsuspend
        </Button>
      )}
      <Button
        variant="danger"
        loading={del.isPending}
        onClick={() => setDeleteModal(true)}
      >
        Delete
      </Button>
    </>
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title={
          <span className="flex items-center gap-3">
            <span>{a.username}</span>
            <StatusPill tone={a.status === "active" ? "success" : a.status === "suspended" ? "warning" : "neutral"}>
              {a.status}
            </StatusPill>
          </span>
        }
        description={
          <span>
            <Link to="/accounts" className="text-brand-600 hover:underline">Accounts</Link>
            <span className="mx-1.5 text-ink-4">/</span>
            <span className="font-mono text-xs">{a.id}</span>
          </span>
        }
        actions={actions}
      />

      <Tabs
        active={tab}
        onChange={setTab}
        tabs={[
          {
            key: "overview",
            label: "Overview",
            panel: (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                <Card>
                  <CardHeader title="Account" />
                  <dl className="grid grid-cols-2 gap-3 text-sm">
                    <Field label="Username"  value={a.username} />
                    <Field label="Plan"      value={a.plan} />
                    <Field label="Domain"    value={a.domain || "—"} />
                    <Field label="Tenant"    value={a.tenant_id} mono />
                    <Field label="Disk quota" value={`${a.disk_quota_mb} MB`} />
                    <Field label="Disk used"  value={a.disk_used_mb != null ? `${a.disk_used_mb} MB` : "—"} />
                    <Field label="Bandwidth" value={`${a.bandwidth_gb} GB`} />
                    <Field label="Created"   value={formatDate(a.created_at)} />
                  </dl>

                  {/* Disk usage visualization */}
                  {a.disk_used_mb != null && a.disk_quota_mb > 0 && (
                    <div className="mt-4">
                      <div className="mb-1 flex items-center justify-between text-xs">
                        <span className="text-ink-3">Disk Usage</span>
                        <span className="font-mono text-ink-2">
                          {a.disk_used_mb} / {a.disk_quota_mb} MB
                        </span>
                      </div>
                      <DiskUsageBar used={a.disk_used_mb} total={a.disk_quota_mb} />
                    </div>
                  )}
                </Card>
                <Card className="md:col-span-2">
                  <CardHeader
                    title="Usage"
                    description="From /api/v1/accounts/:id/usage. Schema is intentionally open; v0.4.0 will lock it down."
                  />
                  <pre className="overflow-x-auto rounded-md bg-surface-2 p-3 text-xs font-mono text-ink-1">
                    {usage.isLoading ? "loading…" : usage.isError ? "error" : JSON.stringify(usage.data ?? {}, null, 2)}
                  </pre>
                </Card>
              </div>
            ),
          },
          {
            key: "domains",
            label: `Domains (${doms.data?.domains.length ?? 0})`,
            panel: <DomainsTab accountId={id} rows={doms.data?.domains ?? []} isLoading={doms.isLoading} onReload={() => doms.refetch()} />,
          },
          {
            key: "deployments",
            label: `Deployments (${deps.data?.deployments.length ?? 0})`,
            panel: <DeploymentsTab rows={deps.data?.deployments ?? []} isLoading={deps.isLoading} onReload={() => deps.refetch()} />,
          },
        ]}
      />

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal}
        onClose={() => !isPending && setDeleteModal(false)}
        title="Delete Account"
        description={`Are you sure you want to delete "${a.username}"? This will permanently remove the system user and all associated domains.`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleteModal(false)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="danger"
              loading={del.isPending}
              onClick={() => del.mutate()}
            >
              Delete Account
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This action is irreversible and will permanently delete all data associated with this account.
        </div>
      </Modal>
    </div>
  );
}

// Disk usage bar component
function DiskUsageBar({ used, total }: { used: number; total: number }) {
  const pct = Math.min((used / total) * 100, 100);
  const tone = pct > 90 ? "danger" : pct > 70 ? "warning" : "success";

  return (
    <>
      <div className="h-2 w-full overflow-hidden rounded-full bg-ink-6">
        <div
          className={`h-full rounded-full bg-${tone}-500 transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="mt-1 flex items-center justify-between text-[11px] text-ink-3">
        <span>{Math.round(pct)}% used</span>
        <span>{total - used} MB free</span>
      </div>
    </>
  );
}

function Field({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-wider text-ink-3">{label}</dt>
      <dd className={"mt-0.5 text-sm text-ink-1 break-all " + (mono ? "font-mono" : "")}>{value}</dd>
    </div>
  );
}

function DomainsTab({
  accountId, rows, isLoading, onReload,
}: { accountId: string; rows: Domain[]; isLoading: boolean; onReload: () => void }) {
  const qc = useQueryClient();
  const del = useMutation({
    mutationFn: (d: string) => deleteDomain(accountId, d),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: domainKeys.byAccount(accountId) });
      qc.invalidateQueries({ queryKey: deploymentKeys.byAccount(accountId) });
    },
  });

  const columns: Column<Domain>[] = [
    { key: "name",   header: "Domain",       cell: (d) => <span className="font-mono text-xs">{d.name}</span> },
    { key: "doc",    header: "Document root", cell: (d) => <span className="font-mono text-xs">{d.document_root}</span> },
    { key: "status", header: "Status",       cell: (d) => <StatusPill tone={d.status === "active" ? "success" : "neutral"}>{d.status}</StatusPill> },
    { key: "created", header: "Created",     cell: (d) => <span className="font-mono text-xs text-ink-2">{formatDate(d.created_at)}</span> },
    {
      key: "actions", header: "", align: "right",
      cell: (d) => (
        <Button
          variant="ghost" size="sm" className="text-danger"
          loading={del.isPending}
          onClick={() => del.mutate(d.name)}
        >
          Delete
        </Button>
      ),
    },
  ];

  return (
    <Card>
      <div className="mb-3 flex items-center justify-between">
        <div className="text-sm text-ink-2">Domains owned by this account.</div>
        <Button variant="ghost" size="sm" onClick={onReload}>Refresh</Button>
      </div>
      <Table columns={columns} rows={rows} keyOf={(d) => d.id} isLoading={isLoading} />
    </Card>
  );
}

function DeploymentsTab({
  rows, isLoading, onReload,
}: { rows: Deployment[]; isLoading: boolean; onReload: () => void }) {
  const columns: Column<Deployment>[] = [
    { key: "release",    header: "Release",     cell: (d) => <span className="font-mono text-xs">{d.release}</span> },
    { key: "domain",     header: "Domain",      cell: (d) => <span className="font-mono text-xs">{d.domain}</span> },
    { key: "is_current", header: "Current",     cell: (d) => d.is_current ? <StatusPill tone="success">current</StatusPill> : <span className="text-ink-3">—</span> },
    { key: "size",       header: "Size",        cell: (d) => <span className="font-mono text-xs">{formatBytes(d.size_bytes)}</span> },
    { key: "modified",   header: "Modified at", cell: (d) => <span className="font-mono text-xs text-ink-2">{formatDate(d.modified_at)}</span> },
  ];
  return (
    <Card>
      <div className="mb-3 flex items-center justify-between">
        <div className="text-sm text-ink-2">Release directories on disk. "Current" matches the document-root symlink.</div>
        <Button variant="ghost" size="sm" onClick={onReload}>Refresh</Button>
      </div>
      <Table columns={columns} rows={rows} keyOf={(d) => `${d.domain}:${d.release}`} isLoading={isLoading} />
    </Card>
  );
}