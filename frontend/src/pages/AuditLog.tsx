/**
 * Audit Log page. Lists audit entries with filters, search, and pagination.
 * The "Verify chain" button hits /api/v1/admin/audit-log/verify and shows
 * a tamper banner if the chain breaks.
 *
 * The columns follow the models.AuditEntry shape: timestamp, user,
 * action, resource, result, ip. Each row is plain text; we don't
 * try to render an "action icon" because the action vocabulary is
 * not stable across versions.
 */

import { useState, useMemo } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { ErrorState, EmptyState, Spinner } from "@/lib/ui/Feedback";
import { auditKeys } from "@/lib/query/keys";
import { listAudit, verifyAudit, type AuditEntry } from "@/lib/api/system";

const PAGE_SIZE = 25;

export function AuditLogPage() {
  // Filters and pagination
  const [limit, setLimit] = useState(100);
  const [searchQuery, setSearchQuery] = useState("");
  const [resultFilter, setResultFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  const q = useQuery({ queryKey: auditKeys.list(limit), queryFn: () => listAudit(limit) });
  const verify = useMutation({ mutationFn: verifyAudit });

  // Filter entries
  const filteredEntries = useMemo(() => {
    let entries = q.data?.entries ?? [];

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      entries = entries.filter(
        (e) =>
          e.action.toLowerCase().includes(query) ||
          e.user_email?.toLowerCase().includes(query) ||
          e.resource_type.toLowerCase().includes(query) ||
          e.resource_name?.toLowerCase().includes(query) ||
          e.actor_ip.toLowerCase().includes(query),
      );
    }

    // Result filter
    if (resultFilter !== "all") {
      entries = entries.filter((e) => e.result === resultFilter);
    }

    return entries;
  }, [q.data?.entries, searchQuery, resultFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredEntries.length / PAGE_SIZE);
  const paginatedEntries = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredEntries.slice(start, start + PAGE_SIZE);
  }, [filteredEntries, currentPage]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  const handleResultChange = (value: string) => {
    setResultFilter(value);
    setCurrentPage(1);
  };

  const handleLimitChange = (value: number) => {
    setLimit(value);
    setCurrentPage(1);
  };

  const columns: Column<AuditEntry>[] = [
    {
      key: "timestamp",
      header: "When",
      cell: (e) => (
        <div className="font-mono text-xs">
          <div>{new Date(e.timestamp).toLocaleDateString()}</div>
          <div className="text-ink-3">{new Date(e.timestamp).toLocaleTimeString()}</div>
        </div>
      ),
    },
    {
      key: "user",
      header: "User",
      cell: (e) => (
        <span className="font-mono text-xs" title={e.user_id}>
          {e.user_email || e.user_id.slice(0, 12) + "..."}
        </span>
      ),
    },
    {
      key: "action",
      header: "Action",
      cell: (e) => <span className="font-mono text-xs">{e.action}</span>,
    },
    {
      key: "resource",
      header: "Resource",
      cell: (e) => (
        <span className="font-mono text-xs" title={e.resource_id}>
          {e.resource_type}
          {e.resource_name ? ` · ${e.resource_name}` : ""}
        </span>
      ),
    },
    {
      key: "result",
      header: "Result",
      cell: (e) => (
        <StatusPill tone={e.result === "success" ? "success" : e.result === "denied" ? "warning" : "danger"}>
          {e.result}
        </StatusPill>
      ),
    },
    {
      key: "ip",
      header: "IP",
      cell: (e) => <span className="font-mono text-xs">{e.actor_ip}</span>,
    },
    {
      key: "hash",
      header: "Hash",
      cell: (e) => <span className="font-mono text-[10px] text-ink-3">{e.hash?.slice(0, 12) ?? "—"}</span>,
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Log"
        description={`${filteredEntries.length} entries — append-only, hash-chained record of every action.`}
        actions={
          <>
            <Select
              value={String(limit)}
              onChange={(e) => handleLimitChange(parseInt(e.target.value, 10))}
              className="w-24"
            >
              <option value={50}>50</option>
              <option value={100}>100</option>
              <option value={200}>200</option>
              <option value={500}>500</option>
            </Select>
            <Button
              variant="secondary"
              onClick={() => verify.mutate()}
              loading={verify.isPending}
            >
              Verify chain
            </Button>
          </>
        }
      />

      {/* Verification status banner */}
      {verify.data && (
        <div
          className={
            "rounded-md border px-4 py-3 text-sm " +
            (verify.data.tampered
              ? "border-danger/30 bg-danger/5 text-danger"
              : "border-success/30 bg-success/5 text-success")
          }
        >
          <div className="flex items-center gap-2">
            <StatusPill tone={verify.data.tampered ? "danger" : "success"} dot>
              {verify.data.tampered ? "Chain Broken" : "Chain Verified"}
            </StatusPill>
            <span>
              {verify.data.tampered
                ? `Chain broken at row ${verify.data.first_bad_row}. ${verify.data.error ?? ""}`
                : "No tampering detected."}
            </span>
          </div>
        </div>
      )}

      <Card>
        {/* Filters */}
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1">
            <Input
              label="Search"
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Search by action, user, resource, or IP…"
            />
          </div>
          <div className="w-full sm:w-40">
            <Select
              label="Result"
              value={resultFilter}
              onChange={(e) => handleResultChange(e.target.value)}
            >
              <option value="all">All results</option>
              <option value="success">Success</option>
              <option value="denied">Denied</option>
              <option value="error">Error</option>
            </Select>
          </div>
        </div>

        {q.isError ? (
          <ErrorState description="Failed to load audit log." onRetry={() => q.refetch()} />
        ) : (
          <>
            <Table
              columns={columns}
              rows={paginatedEntries}
              keyOf={(e) => e.id}
              isLoading={q.isLoading}
              emptyState={
                <EmptyState
                  title={searchQuery || resultFilter !== "all" ? "No entries match your filters" : "No audit entries"}
                  description={
                    searchQuery || resultFilter !== "all"
                      ? "Try adjusting your search or filters."
                      : "Actions you take in the panel will appear here."
                  }
                />
              }
            />

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredEntries.length)} of{" "}
                  {filteredEntries.length} entries
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