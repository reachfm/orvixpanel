/**
 * Accounts list. Real data, with search filter and pagination.
 * Each row links to the account detail. Actions use confirmation modals.
 */

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState, Spinner } from "@/lib/ui/Feedback";
import { accountKeys, domainKeys } from "@/lib/query/keys";
import {
  listAccounts, suspendAccount, unsuspendAccount, deleteAccount,
  type Account,
} from "@/lib/api/accounts";

const PAGE_SIZE = 20;

export function AccountsListPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Filter and pagination state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Modal state
  const [confirmModal, setConfirmModal] = useState<{
    type: "suspend" | "unsuspend" | "delete";
    account: Account;
  } | null>(null);

  const q = useQuery({
    queryKey: accountKeys.list(),
    queryFn: listAccounts,
  });

  const accounts = q.data?.accounts ?? [];

  // Filter accounts
  const filteredAccounts = useMemo(() => {
    let result = accounts;

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (a) =>
          a.username.toLowerCase().includes(query) ||
          (a.domain || "").toLowerCase().includes(query) ||
          (a.email || "").toLowerCase().includes(query),
      );
    }

    // Status filter
    if (statusFilter !== "all") {
      result = result.filter((a) => a.status === statusFilter);
    }

    return result;
  }, [accounts, searchQuery, statusFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredAccounts.length / PAGE_SIZE);
  const paginatedAccounts = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredAccounts.slice(start, start + PAGE_SIZE);
  }, [filteredAccounts, currentPage]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  const handleStatusChange = (value: string) => {
    setStatusFilter(value);
    setCurrentPage(1);
  };

  // Mutations
  const suspend = useMutation({
    mutationFn: (id: string) => suspendAccount(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: accountKeys.all() });
      setConfirmModal(null);
    },
  });
  const unsuspend = useMutation({
    mutationFn: (id: string) => unsuspendAccount(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: accountKeys.all() });
      setConfirmModal(null);
    },
  });
  const del = useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: accountKeys.all() });
      qc.invalidateQueries({ queryKey: domainKeys.all() });
      setConfirmModal(null);
    },
  });

  const handleConfirmAction = () => {
    if (!confirmModal) return;
    const { type, account } = confirmModal;
    if (type === "suspend") suspend.mutate(account.id);
    else if (type === "unsuspend") unsuspend.mutate(account.id);
    else if (type === "delete") del.mutate(account.id);
  };

  const isActionPending = suspend.isPending || unsuspend.isPending || del.isPending;

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
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          {a.status === "active" ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setConfirmModal({ type: "suspend", account: a })}
            >
              Suspend
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setConfirmModal({ type: "unsuspend", account: a })}
            >
              Unsuspend
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="text-danger"
            onClick={() => setConfirmModal({ type: "delete", account: a })}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  const confirmModalContent = confirmModal ? {
    suspend: {
      title: "Suspend Account",
      description: `Are you sure you want to suspend "${confirmModal.account.username}"? This will temporarily disable all services for this account.`,
      confirmText: "Suspend",
      confirmVariant: "warning" as const,
    },
    unsuspend: {
      title: "Unsuspend Account",
      description: `Are you sure you want to unsuspend "${confirmModal.account.username}"? This will restore all services for this account.`,
      confirmText: "Unsuspend",
      confirmVariant: "primary" as const,
    },
    delete: {
      title: "Delete Account",
      description: `Are you sure you want to delete "${confirmModal.account.username}"? This will permanently remove the system user and all associated domains. This action cannot be undone.`,
      confirmText: "Delete",
      confirmVariant: "danger" as const,
    },
  }[confirmModal.type] : null;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Accounts"
        description={`${filteredAccounts.length} account${filteredAccounts.length === 1 ? "" : "s"} on this panel`}
        actions={
          <Button variant="primary" onClick={() => navigate({ to: "/accounts/new" })}>
            New account
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
              placeholder="Search by username, domain, or email…"
            />
          </div>
          <div className="w-full sm:w-48">
            <Select
              label="Status"
              value={statusFilter}
              onChange={(e) => handleStatusChange(e.target.value)}
            >
              <option value="all">All statuses</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
            </Select>
          </div>
        </div>

        {q.isError ? (
          <ErrorState
            description="Failed to load accounts."
            onRetry={() => q.refetch()}
          />
        ) : (
          <>
            <Table
              columns={columns}
              rows={paginatedAccounts}
              keyOf={(a) => a.id}
              isLoading={q.isLoading}
              emptyState={
                <EmptyState
                  title={searchQuery || statusFilter !== "all" ? "No accounts match your filters" : "No accounts yet"}
                  description={
                    searchQuery || statusFilter !== "all"
                      ? "Try adjusting your search or filters."
                      : "Create your first account to start serving sites."
                  }
                  action={
                    !searchQuery && statusFilter === "all" && (
                      <Button variant="primary" onClick={() => navigate({ to: "/accounts/new" })}>
                        Create account
                      </Button>
                    )
                  }
                />
              }
              onRowClick={(a) => navigate({ to: "/accounts/$id", params: { id: a.id } })}
            />

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredAccounts.length)} of{" "}
                  {filteredAccounts.length} accounts
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

      {/* Confirmation Modal */}
      <Modal
        open={!!confirmModal}
        onClose={() => !isActionPending && setConfirmModal(null)}
        title={confirmModalContent?.title ?? ""}
        description={confirmModalContent?.description}
        width="md"
        footer={
          <>
            <Button
              variant="secondary"
              onClick={() => setConfirmModal(null)}
              disabled={isActionPending}
            >
              Cancel
            </Button>
            <Button
              variant={confirmModalContent?.confirmVariant}
              loading={isActionPending}
              onClick={handleConfirmAction}
            >
              {confirmModalContent?.confirmText}
            </Button>
          </>
        }
      >
        <div className="text-sm text-ink-2">
          {confirmModal?.type === "delete" && (
            <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-danger">
              This action is irreversible and will permanently delete all data.
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}
