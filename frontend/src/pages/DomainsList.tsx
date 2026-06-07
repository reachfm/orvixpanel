/**
 * Domains list. Cross-account view: for each account we fetch its
 * domains in parallel and merge the rows. For N accounts this is
 * N requests; v0.4.0 should add /api/v1/domains?account_id=… or a
 * /api/v1/domains aggregate. The UI is honest about the count.
 */

import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState, EmptyState } from "@/lib/ui/Feedback";
import { accountKeys, domainKeys } from "@/lib/query/keys";
import { listAccounts } from "@/lib/api/accounts";
import { listDomains, type Domain } from "@/lib/api/domains";
import { useQueries } from "@tanstack/react-query";

export function DomainsListPage() {
  const accts = useQuery({ queryKey: accountKeys.list(), queryFn: listAccounts });
  const accounts = accts.data?.accounts ?? [];

  // Fire a listDomains per account in parallel.
  const domQueries = useQueries({
    queries: accounts.map((a) => ({
      queryKey: [...domainKeys.byAccount(a.id), "for-domains-page"] as const,
      queryFn: () => listDomains(a.id),
    })),
  });

  const isLoading = accts.isLoading || domQueries.some((q) => q.isLoading);
  const isError   = accts.isError || domQueries.some((q) => q.isError);
  const allDomains: Array<Domain & { account_username: string }> = [];
  for (let i = 0; i < accounts.length; i++) {
    const acct = accounts[i];
    const r = domQueries[i];
    if (!r?.data) continue;
    for (const d of r.data.domains) allDomains.push({ ...d, account_username: acct.username });
  }

  const columns: Column<Domain & { account_username: string }>[] = [
    {
      key: "name",
      header: "Domain",
      cell: (d) => <span className="font-mono text-xs">{d.name}</span>,
    },
    {
      key: "account",
      header: "Account",
      cell: (d) => (
        <Link to="/accounts/$id" params={{ id: d.account_id }} className="text-brand-600 hover:underline">
          {d.account_username}
        </Link>
      ),
    },
    { key: "doc", header: "Document root", cell: (d) => <span className="font-mono text-xs">{d.document_root}</span> },
    {
      key: "status", header: "Status",
      cell: (d) => <StatusPill tone={d.status === "active" ? "success" : "neutral"}>{d.status}</StatusPill>,
    },
    {
      key: "created", header: "Created",
      cell: (d) => <span className="font-mono text-xs">{new Date(d.created_at).toLocaleString()}</span>,
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Domains"
        description="Every domain across every account on this panel."
      />

      <Card>
        {isError ? (
          <ErrorState description="Failed to load domains." onRetry={() => accts.refetch()} />
        ) : isLoading ? (
          <div className="flex items-center justify-center py-12 text-sm text-ink-3">
            <Spinner size={20} /> <span className="ml-2">Loading…</span>
          </div>
        ) : allDomains.length === 0 ? (
          <EmptyState
            title="No domains yet"
            description="Create an account with a primary domain, or add a domain to an existing account."
            action={
              <Link to="/accounts/new" className="inline-flex h-9 items-center rounded-md bg-brand-600 px-3.5 text-sm font-medium text-white hover:bg-brand-700">
                New account
              </Link>
            }
          />
        ) : (
          <Table columns={columns} rows={allDomains} keyOf={(d) => `${d.account_id}:${d.id}`} />
        )}
      </Card>
    </div>
  );
}
