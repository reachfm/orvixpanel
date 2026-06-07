/**
 * Accounts list. Real data, with a search filter that runs client-side
 * over the real list (the backend doesn't have a /accounts?q= route
 * yet — v0.3.x can add it). Each row links to the account detail.
 */

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState } from "@/lib/ui/Feedback";
import { accountKeys, domainKeys } from "@/lib/query/keys";
import {
  listAccounts, suspendAccount, unsuspendAccount, deleteAccount,
  type Account,
} from "@/lib/api/accounts";

export function AccountsListPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [filter, setFilter] = useState("");

  const q = useQuery({
    queryKey: accountKeys.list(),
    queryFn: listAccounts,
  });

  // We also want a domain count per account, but the backend doesn't
  // expose a count endpoint. We do a parallel per-account listDomains
  // via a derived query. For 50 accounts this is fine; for thousands
  // the backend would need a /accounts/:id/stats endpoint (v0.4.0).
  const accounts = q.data?.accounts ?? [];
  const filtered = accounts.filter((a) =>
    !filter || a.username.includes(filter.toLowerCase()) || (a.domain || "").includes(filter.toLowerCase()),
  );

  const suspend = useMutation({
    mutationFn: (id: string) => suspendAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: accountKeys.all() }),
  });
  const unsuspend = useMutation({
    mutationFn: (id: string) => unsuspendAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: accountKeys.all() }),
  });
  const del = useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: accountKeys.all() });
      qc.invalidateQueries({ queryKey: domainKeys.all() });
    },
  });

  const columns: Column<Account>[] = [
    {
      key: "username",
      header: "Username",
      cell: (a) => (
        <Link
          to="/accounts/$id"
          params={{ id: a.id }}
          className="font-medium text-brand-600 hover:underline"
        >
          {a.username}
        </Link>
      ),
    },
    {
      key: "domain",
      header: "Primary domain",
      cell: (a) => <span className="font-mono text-xs">{a.domain || "—"}</span>,
    },
    {
      key: "plan",
      header: "Plan",
      cell: (a) => <span className="capitalize">{a.plan}</span>,
    },
    {
      key: "quota",
      header: "Disk",
      cell: (a) => (
        <span className="font-mono text-xs">
          {a.disk_used_mb != null ? `${a.disk_used_mb} / ${a.disk_quota_mb} MB` : `${a.disk_quota_mb} MB`}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (a) => (
        <StatusPill tone={a.status === "active" ? "success" : a.status === "suspended" ? "warning" : "neutral"}>
          {a.status}
        </StatusPill>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (a) => <span className="font-mono text-xs">{new Date(a.created_at).toLocaleString()}</span>,
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (a) => (
        <div className="flex justify-end gap-1">
          {a.status === "active" ? (
            <Button variant="ghost" size="sm" loading={suspend.isPending} onClick={(e) => { e.stopPropagation(); suspend.mutate(a.id); }}>
              Suspend
            </Button>
          ) : (
            <Button variant="ghost" size="sm" loading={unsuspend.isPending} onClick={(e) => { e.stopPropagation(); unsuspend.mutate(a.id); }}>
              Unsuspend
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="text-danger"
            loading={del.isPending}
            onClick={(e) => {
              e.stopPropagation();
              if (window.confirm(`Delete account "${a.username}"? This also deletes the system user and all domains.`)) {
                del.mutate(a.id);
              }
            }}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title="Accounts"
        description="Hosting accounts on this panel."
        actions={
          <Button variant="primary" onClick={() => navigate({ to: "/accounts/new" })}>
            New account
          </Button>
        }
      />

      <Card>
        <div className="mb-3 max-w-sm">
          <Input
            label="Filter"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="username or domain…"
          />
        </div>

        {q.isError ? (
          <ErrorState
            description="Failed to load accounts."
            onRetry={() => q.refetch()}
          />
        ) : (
          <Table
            columns={columns}
            rows={filtered}
            keyOf={(a) => a.id}
            isLoading={q.isLoading}
            emptyState={
              <EmptyState
                title={filter ? "No accounts match your filter" : "No accounts yet"}
                description={filter ? "Try a different filter." : "Create your first account to start serving sites."}
                action={
                  !filter && (
                    <Button variant="primary" onClick={() => navigate({ to: "/accounts/new" })}>
                      Create account
                    </Button>
                  )
                }
              />
            }
            onRowClick={(a) => navigate({ to: "/accounts/$id", params: { id: a.id } })}
          />
        )}
      </Card>
    </div>
  );
}
