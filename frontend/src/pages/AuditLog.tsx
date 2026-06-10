/**
 * Audit Log page. Lists audit entries with filters, search, and pagination.
 * The "Verify chain" button hits /api/v1/admin/audit-log/verify and shows
 * a tamper banner if the chain breaks.
 *
 * The columns follow the models.AuditEntry shape: timestamp, user,
 * action, resource, result, ip. Each row is plain text; we don't
 * try to render an "action icon" because the action vocabulary is
 * not stable across versions.
 * v0.3.1 Phase E: Enhanced with advanced filtering, export, and timeline view.
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
import { ErrorState, EmptyState } from "@/lib/ui/Feedback";
import { auditKeys } from "@/lib/query/keys";
import { listAudit, verifyAudit, type AuditEntry } from "@/lib/api/system";

const PAGE_SIZE = 25;

// View modes for audit log display
type ViewMode = "table" | "timeline";

export function AuditLogPage() {
  // Filters and pagination
  const [limit, setLimit] = useState(100);
  const [searchQuery, setSearchQuery] = useState("");
  const [resultFilter, setResultFilter] = useState<string>("all");
  const [actionFilter, setActionFilter] = useState<string>("all");
  const [resourceFilter, setResourceFilter] = useState<string>("all");
  const [dateFrom, setDateFrom] = useState<string>("");
  const [dateTo, setDateTo] = useState<string>("");
  const [currentPage, setCurrentPage] = useState(1);
  const [viewMode, setViewMode] = useState<ViewMode>("table");
  const [expandedEntry, setExpandedEntry] = useState<string | null>(null);

  const q = useQuery({ queryKey: auditKeys.list(limit), queryFn: () => listAudit(limit) });
  const verify = useMutation({ mutationFn: verifyAudit });

  // Extract unique action types and resource types from entries
  const uniqueActions = useMemo(() => {
    const actions = new Set<string>();
    q.data?.entries?.forEach((e) => actions.add(e.action));
    return Array.from(actions).sort();
  }, [q.data?.entries]);

  const uniqueResources = useMemo(() => {
    const resources = new Set<string>();
    q.data?.entries?.forEach((e) => resources.add(e.resource_type));
    return Array.from(resources).sort();
  }, [q.data?.entries]);

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

    // Action filter
    if (actionFilter !== "all") {
      entries = entries.filter((e) => e.action === actionFilter);
    }

    // Resource filter
    if (resourceFilter !== "all") {
      entries = entries.filter((e) => e.resource_type === resourceFilter);
    }

    // Date from filter
    if (dateFrom) {
      const fromDate = new Date(dateFrom);
      entries = entries.filter((e) => new Date(e.timestamp) >= fromDate);
    }

    // Date to filter
    if (dateTo) {
      const toDate = new Date(dateTo);
      toDate.setHours(23, 59, 59, 999);
      entries = entries.filter((e) => new Date(e.timestamp) <= toDate);
    }

    return entries;
  }, [q.data?.entries, searchQuery, resultFilter, actionFilter, resourceFilter, dateFrom, dateTo]);

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

  // Clear all filters
  const clearFilters = () => {
    setSearchQuery("");
    setResultFilter("all");
    setActionFilter("all");
    setResourceFilter("all");
    setDateFrom("");
    setDateTo("");
    setCurrentPage(1);
  };

  // Export to CSV
  const exportToCSV = () => {
    const headers = ["Timestamp", "User", "Action", "Resource Type", "Resource ID", "Resource Name", "Result", "IP", "Hash"];
    const rows = filteredEntries.map((e) => [
      new Date(e.timestamp).toISOString(),
      e.user_email || e.user_id,
      e.action,
      e.resource_type,
      e.resource_id,
      e.resource_name || "",
      e.result,
      e.actor_ip,
      e.hash || "",
    ]);
    const csv = [headers, ...rows].map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `audit-log-${new Date().toISOString().split("T")[0]}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  // Group entries by date for timeline view
  const timelineGroups = useMemo(() => {
    const groups: { [date: string]: AuditEntry[] } = {};
    filteredEntries.forEach((entry) => {
      const date = new Date(entry.timestamp).toLocaleDateString();
      if (!groups[date]) groups[date] = [];
      groups[date].push(entry);
    });
    return Object.entries(groups).map(([date, entries]) => ({ date, entries }));
  }, [filteredEntries]);

  const columns: Column<AuditEntry>[] = [
    {
      key: "timestamp",
      header: "When",
      cell: (e) => (
        <div className="font-mono text-xs">
          <div className="text-ink-1">{new Date(e.timestamp).toLocaleDateString()}</div>
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
    {
      key: "expand",
      header: "",
      cell: (e) => (
        <button
          className="rounded p-1 hover:bg-surface-2"
          onClick={(ev) => {
            ev.stopPropagation();
            setExpandedEntry(expandedEntry === e.id ? null : e.id);
          }}
        >
          <svg
            className={`h-4 w-4 text-ink-3 transition-transform ${expandedEntry === e.id ? "rotate-180" : ""}`}
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={2}
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Log"
        description={`${filteredEntries.length} entries — append-only, hash-chained record of every action.`}
        actions={
          <>
            <div className="flex rounded-md border border-surface-border">
              <button
                className={`px-3 py-1.5 text-sm ${viewMode === "table" ? "bg-surface-2 text-ink-1" : "text-ink-3 hover:text-ink-2"}`}
                onClick={() => setViewMode("table")}
              >
                Table
              </button>
              <button
                className={`px-3 py-1.5 text-sm border-l border-surface-border ${viewMode === "timeline" ? "bg-surface-2 text-ink-1" : "text-ink-3 hover:text-ink-2"}`}
                onClick={() => setViewMode("timeline")}
              >
                Timeline
              </button>
            </div>
            <Button variant="secondary" size="sm" onClick={exportToCSV}>
              <svg className="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              Export CSV
            </Button>
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
          <div className="w-full sm:w-36">
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
          <div className="w-full sm:w-36">
            <Select
              label="Action"
              value={actionFilter}
              onChange={(e) => { setActionFilter(e.target.value); setCurrentPage(1); }}
            >
              <option value="all">All actions</option>
              {uniqueActions.map((action) => (
                <option key={action} value={action}>{action}</option>
              ))}
            </Select>
          </div>
          <div className="w-full sm:w-36">
            <Select
              label="Resource"
              value={resourceFilter}
              onChange={(e) => { setResourceFilter(e.target.value); setCurrentPage(1); }}
            >
              <option value="all">All resources</option>
              {uniqueResources.map((res) => (
                <option key={res} value={res}>{res}</option>
              ))}
            </Select>
          </div>
          <div className="flex items-end gap-2">
            <Input
              label="From"
              type="date"
              value={dateFrom}
              onChange={(e) => { setDateFrom(e.target.value); setCurrentPage(1); }}
              className="w-32"
            />
            <Input
              label="To"
              type="date"
              value={dateTo}
              onChange={(e) => { setDateTo(e.target.value); setCurrentPage(1); }}
              className="w-32"
            />
          </div>
          {(searchQuery || resultFilter !== "all" || actionFilter !== "all" || resourceFilter !== "all" || dateFrom || dateTo) && (
            <Button variant="ghost" size="sm" onClick={clearFilters}>
              Clear filters
            </Button>
          )}
        </div>

        {q.isError ? (
          <ErrorState description="Failed to load audit log." onRetry={() => q.refetch()} />
        ) : (
          <>
            {viewMode === "table" ? (
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
                {/* Expanded entry details */}
                {paginatedEntries.map((entry) => (
                  expandedEntry === entry.id && (
                    <EntryDetails key={`details-${entry.id}`} entry={entry} />
                  )
                ))}
              </>
            ) : (
              <TimelineView groups={timelineGroups} expandedEntry={expandedEntry} onToggle={setExpandedEntry} />
            )}

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

// Entry details component for expanded row view
function EntryDetails({ entry }: { entry: AuditEntry }) {
  return (
    <div className="mt-4 rounded-md border border-surface-border bg-surface-2 p-4">
      <div className="mb-3 text-sm font-semibold text-ink-1">Entry Details</div>
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-ink-3">User ID</dt>
          <dd className="mt-0.5 font-mono text-xs text-ink-1">{entry.user_id}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-ink-3">User Email</dt>
          <dd className="mt-0.5 font-mono text-xs text-ink-1">{entry.user_email || "—"}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-ink-3">Resource ID</dt>
          <dd className="mt-0.5 font-mono text-xs text-ink-1">{entry.resource_id}</dd>
        </div>
        <div>
          <dt className="text-[11px] uppercase tracking-wider text-ink-3">Resource Name</dt>
          <dd className="mt-0.5 font-mono text-xs text-ink-1">{entry.resource_name || "—"}</dd>
        </div>
        <div className="col-span-2">
          <dt className="text-[11px] uppercase tracking-wider text-ink-3">Previous Hash</dt>
          <dd className="mt-0.5 font-mono text-xs text-ink-1 break-all">{entry.prev_hash || "—"}</dd>
        </div>
        <div className="col-span-2">
          <dt className="text-[11px] uppercase tracking-wider text-ink-3">Entry Hash</dt>
          <dd className="mt-0.5 font-mono text-xs text-ink-1 break-all">{entry.hash || "—"}</dd>
        </div>
        {entry.details && (
          <div className="col-span-2">
            <dt className="text-[11px] uppercase tracking-wider text-ink-3">Details</dt>
            <dd className="mt-0.5 font-mono text-xs text-ink-1 whitespace-pre-wrap">{entry.details}</dd>
          </div>
        )}
      </div>
    </div>
  );
}

// Timeline view component for grouped entries by date
function TimelineView({
  groups,
  expandedEntry,
  onToggle,
}: {
  groups: { date: string; entries: AuditEntry[] }[];
  expandedEntry: string | null;
  onToggle: (id: string | null) => void;
}) {
  if (groups.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-ink-3">
        No entries to display
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {groups.map((group) => (
        <div key={group.date}>
          <div className="mb-3 flex items-center gap-3">
            <div className="h-px flex-1 bg-surface-border" />
            <span className="text-sm font-medium text-ink-2">{group.date}</span>
            <div className="h-px flex-1 bg-surface-border" />
          </div>
          <div className="space-y-2">
            {group.entries.map((entry) => (
              <div key={entry.id}>
                <div
                  className="flex cursor-pointer items-center gap-3 rounded-md border border-surface-border bg-surface-1 p-3 hover:bg-surface-2"
                  onClick={() => onToggle(expandedEntry === entry.id ? null : entry.id)}
                >
                  <StatusPill
                    tone={entry.result === "success" ? "success" : entry.result === "denied" ? "warning" : "danger"}
                    dot
                  >
                    {entry.result}
                  </StatusPill>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs text-ink-1">{entry.action}</span>
                      <span className="text-xs text-ink-3">·</span>
                      <span className="font-mono text-xs text-ink-3">{entry.resource_type}</span>
                    </div>
                    <div className="mt-0.5 flex items-center gap-2 text-[11px] text-ink-3">
                      <span>{entry.user_email || entry.user_id.slice(0, 12) + "..."}</span>
                      <span>·</span>
                      <span>{new Date(entry.timestamp).toLocaleTimeString()}</span>
                      <span>·</span>
                      <span>{entry.actor_ip}</span>
                    </div>
                  </div>
                  <svg
                    className={`h-4 w-4 text-ink-3 transition-transform ${expandedEntry === entry.id ? "rotate-180" : ""}`}
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                  </svg>
                </div>
                {expandedEntry === entry.id && (
                  <EntryDetails entry={entry} />
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}