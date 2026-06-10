/**
 * Domains list. Cross-account view with search, filter, and pagination.
 * Each row links to the domain detail page.
 */

import { useState, useMemo } from "react";
import { useQuery, useQueries } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { Spinner, ErrorState, EmptyState } from "@/lib/ui/Feedback";
import { accountKeys, domainKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";
import { listAccounts } from "@/lib/api/accounts";
import { listDomains, type Domain } from "@/lib/api/domains";

const PAGE_SIZE = 25;

interface DomainWithAccount extends Domain {
  account_username: string;
}

export function DomainsListPage() {
  const navigate = useNavigate();

  // Filter and pagination state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

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
  const isError = accts.isError || domQueries.some((q) => q.isError);

  // Build domains with account info
  const allDomains = useMemo<DomainWithAccount[]>(() => {
    const result: DomainWithAccount[] = [];
    for (let i = 0; i < accounts.length; i++) {
      const acct = accounts[i];
      const r = domQueries[i];
      if (!r?.data) continue;
      for (const d of r.data.domains) {
        result.push({ ...d, account_username: acct.username });
      }
    }
    return result;
  }, [accounts, domQueries]);

  // Filter domains
  const filteredDomains = useMemo(() => {
    let result = allDomains;

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (d) =>
          d.name.toLowerCase().includes(query) ||
          d.account_username.toLowerCase().includes(query) ||
          (d.document_root || "").toLowerCase().includes(query),
      );
    }

    // Status filter
    if (statusFilter !== "all") {
      result = result.filter((d) => d.status === statusFilter);
    }

    return result;
  }, [allDomains, searchQuery, statusFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredDomains.length / PAGE_SIZE);
  const paginatedDomains = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredDomains.slice(start, start + PAGE_SIZE);
  }, [filteredDomains, currentPage]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  const handleStatusChange = (value: string) => {
    setStatusFilter(value);
    setCurrentPage(1);
  };

  const columns: Column<DomainWithAccount>[] = [
    {
      key: "name",
      header: "Domain",
      cell: (d) => (
        <span className="font-mono text-xs font-medium text-ink-1">{d.name}</span>
      ),
    },
    {
      key: "account",
      header: "Account",
      cell: (d) => (
        <Link
          to="/accounts/$id"
          params={{ id: d.account_id }}
          className="text-xs text-brand-600 hover:underline"
        >
          {d.account_username}
        </Link>
      ),
    },
    {
      key: "document_root",
      header: "Document root",
      cell: (d) => <span className="font-mono text-xs text-ink-2">{d.document_root || "—"}</span>,
    },
    {
      key: "status",
      header: "Status",
      cell: (d) => (
        <StatusPill tone={d.status === "active" ? "success" : d.status === "suspended" ? "warning" : "neutral"}>
          {d.status}
        </StatusPill>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (d) => <span className="font-mono text-xs text-ink-2">{formatDate(d.created_at)}</span>,
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Domains"
        description={`${filteredDomains.length} domain${filteredDomains.length === 1 ? "" : "s"} across all accounts`}
        actions={
          <Button variant="secondary" onClick={() => navigate({ to: "/accounts" })}>
            Manage accounts
          </Button>
        }
      />

      <Card>
        {/* Filters */}
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1">
            <Input
              label="Search"
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Search by domain, account, or document root…"
            />
          </div>
          <div className="w-full sm:w-40">
            <Select
              label="Status"
              value={statusFilter}
              onChange={(e) => handleStatusChange(e.target.value)}
            >
              <option value="all">All statuses</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="pending">Pending</option>
            </Select>
          </div>
        </div>

        {isError ? (
          <ErrorState
            description="Failed to load domains."
            onRetry={() => accts.refetch()}
          />
        ) : isLoading ? (
          <div className="flex items-center justify-center py-12 text-sm text-ink-3">
            <Spinner size={20} /> <span className="ml-2">Loading domains…</span>
          </div>
        ) : paginatedDomains.length === 0 ? (
          <EmptyState
            title={searchQuery || statusFilter !== "all" ? "No domains match your filters" : "No domains yet"}
            description={
              searchQuery || statusFilter !== "all"
                ? "Try adjusting your search or filters."
                : "Create an account with a primary domain, or add a domain to an existing account."
            }
            action={
              !searchQuery && statusFilter === "all" && (
                <Button variant="primary" onClick={() => navigate({ to: "/accounts/new" })}>
                  Create account
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table
              columns={columns}
              rows={paginatedDomains}
              keyOf={(d) => `${d.account_id}:${d.id}`}
            />

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredDomains.length)} of{" "}
                  {filteredDomains.length} domains
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={currentPage === 1}
                    onClick={() => setCurrentPage((p) => p - 1)}
                  >
                    Previous
                  </Button>
                  <span className="flex items-center px-2 text-sm text-ink-3">
                    Page {currentPage} of {totalPages}
                  </span>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={currentPage === totalPages}
                    onClick={() => setCurrentPage((p) => p + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  );
}