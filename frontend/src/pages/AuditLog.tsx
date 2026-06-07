/**
 * Audit Log page. Lists the most recent N entries from
 * /api/v1/admin/audit-log. The "Verify chain" button hits
 * /api/v1/admin/audit-log/verify and shows a tamper banner if the
 * chain breaks.
 *
 * The columns follow the models.AuditEntry shape: timestamp, user,
 * action, resource, result, ip. Each row is plain text; we don't
 * try to render an "action icon" because the action vocabulary is
 * not stable across versions.
 */

import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { ErrorState, EmptyState } from "@/lib/ui/Feedback";
import { auditKeys } from "@/lib/query/keys";
import { listAudit, verifyAudit, type AuditEntry } from "@/lib/api/system";

export function AuditLogPage() {
  const [limit, setLimit] = useState(100);
  const q = useQuery({ queryKey: auditKeys.list(limit), queryFn: () => listAudit(limit) });
  const verify = useMutation({ mutationFn: verifyAudit });

  const columns: Column<AuditEntry>[] = [
    {
      key: "timestamp", header: "When",
      cell: (e) => <span className="font-mono text-xs">{new Date(e.timestamp).toLocaleString()}</span>,
    },
    { key: "user", header: "User", cell: (e) => <span className="font-mono text-xs">{e.user_email || e.user_id}</span> },
    { key: "action", header: "Action", cell: (e) => <span className="font-mono text-xs">{e.action}</span> },
    { key: "resource", header: "Resource", cell: (e) => <span className="font-mono text-xs">{e.resource_type}{e.resource_name ? ` · ${e.resource_name}` : ""}</span> },
    {
      key: "result", header: "Result",
      cell: (e) => (
        <StatusPill tone={e.result === "success" ? "success" : e.result === "denied" ? "warning" : "danger"}>
          {e.result}
        </StatusPill>
      ),
    },
    { key: "ip", header: "IP", cell: (e) => <span className="font-mono text-xs">{e.actor_ip}</span> },
    { key: "hash", header: "Hash", cell: (e) => <span className="font-mono text-[10px] text-ink-3">{e.hash?.slice(0, 12) ?? "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit Log"
        description="Append-only, hash-chained record of every action."
        actions={
          <>
            <select
              className="h-9 rounded-md border border-surface-border bg-surface-1 px-2 text-sm text-ink-1"
              value={limit}
              onChange={(e) => setLimit(parseInt(e.target.value, 10))}
            >
              <option value={50}>50</option>
              <option value={100}>100</option>
              <option value={200}>200</option>
              <option value={500}>500</option>
            </select>
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

      {verify.data && (
        <div className={
          "rounded-md border px-3 py-2 text-sm " +
          (verify.data.tampered
            ? "border-danger/30 bg-danger/5 text-danger"
            : "border-success/30 bg-success/5 text-success")
        }>
          {verify.data.tampered
            ? `Chain broken at row ${verify.data.first_bad_row}. ${verify.data.error ?? ""}`
            : "Chain verified — no tampering detected."}
        </div>
      )}

      <Card>
        {q.isError ? (
          <ErrorState description="Failed to load audit log." onRetry={() => q.refetch()} />
        ) : (
          <Table
            columns={columns}
            rows={q.data?.entries ?? []}
            keyOf={(e) => e.id}
            isLoading={q.isLoading}
            emptyState={
              <EmptyState
                title="No audit entries"
                description="Actions you take in the panel will appear here."
              />
            }
          />
        )}
      </Card>
    </div>
  );
}
