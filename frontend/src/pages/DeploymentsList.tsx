/**
 * Deployments — cross-account view. Like Domains, fetches each
 * account's deployment list in parallel and merges.
 *
 * Empty state is honest: "0 deployments across N accounts" is the
 * real number; we don't fabricate charts or counters.
 */

import { useQuery, useQueries } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState, EmptyState } from "@/lib/ui/Feedback";
import { accountKeys, deploymentKeys } from "@/lib/query/keys";
import { listAccounts } from "@/lib/api/accounts";
import { listDeployments, type Deployment } from "@/lib/api/deployments";

export function DeploymentsListPage() {
  const accts = useQuery({ queryKey: accountKeys.list(), queryFn: listAccounts });
  const accounts = accts.data?.accounts ?? [];

  const depQueries = useQueries({
    queries: accounts.map((a) => ({
      queryKey: [...deploymentKeys.byAccount(a.id), "for-deployments-page"] as const,
      queryFn: () => listDeployments(a.id),
    })),
  });

  const isLoading = accts.isLoading || depQueries.some((q) => q.isLoading);
  const isError   = accts.isError || depQueries.some((q) => q.isError);

  const all: Array<Deployment & { account_username: string }> = [];
  for (let i = 0; i < accounts.length; i++) {
    const acct = accounts[i];
    const r = depQueries[i];
    if (!r?.data) continue;
    for (const d of r.data.deployments) all.push({ ...d, account_username: acct.username });
  }
  // Most-recent first.
  all.sort((a, b) => b.modified_at.localeCompare(a.modified_at));

  const columns: Column<Deployment & { account_username: string }>[] = [
    {
      key: "account", header: "Account",
      cell: (d) => (
        <Link to="/accounts/$id" params={{ id: d.account_id }} className="text-brand-600 hover:underline">
          {d.account_username}
        </Link>
      ),
    },
    { key: "domain",   header: "Domain",  cell: (d) => <span className="font-mono text-xs">{d.domain}</span> },
    { key: "release",  header: "Release", cell: (d) => <span className="font-mono text-xs">{d.release}</span> },
    {
      key: "current", header: "State",
      cell: (d) => d.is_current
        ? <StatusPill tone="success">current</StatusPill>
        : <span className="text-ink-3">archived</span>,
    },
    { key: "size",    header: "Size",    cell: (d) => <span className="font-mono text-xs">{formatBytes(d.size_bytes)}</span> },
    { key: "modified", header: "Modified", cell: (d) => <span className="font-mono text-xs">{new Date(d.modified_at).toLocaleString()}</span> },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Deployments"
        description="Release directories across every account."
      />

      <Card>
        {isError ? (
          <ErrorState description="Failed to load deployments." onRetry={() => accts.refetch()} />
        ) : isLoading ? (
          <div className="flex items-center justify-center py-12 text-sm text-ink-3">
            <Spinner size={20} /> <span className="ml-2">Loading…</span>
          </div>
        ) : all.length === 0 ? (
          <EmptyState
            title="No releases yet"
            description={
              accounts.length === 0
                ? "You don't have any accounts. Create one to start deploying."
                : "0 deployments across " + accounts.length + " account" + (accounts.length === 1 ? "" : "s") + ". Releases are created on first deploy; see Releases / Rollback in the roadmap (v0.4.0)."
            }
            action={
              <Link to="/accounts/new" className="inline-flex h-9 items-center rounded-md bg-brand-600 px-3.5 text-sm font-medium text-white hover:bg-brand-700">
                New account
              </Link>
            }
          />
        ) : (
          <Table columns={columns} rows={all} keyOf={(d) => `${d.account_id}:${d.domain}:${d.release}`} />
        )}
      </Card>
    </div>
  );
}

function formatBytes(n: number): string {
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1);
  return `${(n / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}
