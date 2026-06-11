/**
 * Plans list. Real data with search filter and status toggle.
 * Each row links to the plan detail. Actions use confirmation modals.
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
import { EmptyState, ErrorState } from "@/lib/ui/Feedback";
import { planKeys } from "@/lib/query/keys";
import { formatDate } from "@/lib/utils";
import {
  listPlans,
  activatePlan,
  deactivatePlan,
  deletePlan,
  type Plan,
} from "@/lib/api/plans";

const PAGE_SIZE = 20;

export function PlansListPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Filter state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Single action modal state
  const [confirmModal, setConfirmModal] = useState<{
    type: "activate" | "deactivate" | "delete";
    plan: Plan;
  } | null>(null);

  const q = useQuery({
    queryKey: planKeys.list(),
    queryFn: () => listPlans(),
  });

  const plans = q.data?.plans ?? [];

  // Filter plans
  const filteredPlans = useMemo(() => {
    let result = plans;

    // Search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (p) =>
          p.name.toLowerCase().includes(query) ||
          p.display_name.toLowerCase().includes(query) ||
          (p.description || "").toLowerCase().includes(query),
      );
    }

    // Status filter
    if (statusFilter !== "all") {
      result = result.filter((p) =>
        statusFilter === "active" ? p.is_active : !p.is_active
      );
    }

    return result;
  }, [plans, searchQuery, statusFilter]);

  // Pagination
  const totalPages = Math.ceil(filteredPlans.length / PAGE_SIZE);
  const paginatedPlans = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredPlans.slice(start, start + PAGE_SIZE);
  }, [filteredPlans, currentPage]);

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
  const activate = useMutation({
    mutationFn: (id: string) => activatePlan(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: planKeys.all() });
      setConfirmModal(null);
    },
  });
  const deactivate = useMutation({
    mutationFn: (id: string) => deactivatePlan(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: planKeys.all() });
      setConfirmModal(null);
    },
  });
  const del = useMutation({
    mutationFn: (id: string) => deletePlan(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: planKeys.all() });
      setConfirmModal(null);
    },
  });

  const handleConfirmAction = () => {
    if (!confirmModal) return;
    const { type, plan } = confirmModal;
    if (type === "activate") activate.mutate(plan.id);
    else if (type === "deactivate") deactivate.mutate(plan.id);
    else if (type === "delete") del.mutate(plan.id);
  };

  const isActionPending = activate.isPending || deactivate.isPending || del.isPending;

  const columns: Column<Plan>[] = [
    {
      key: "name",
      header: "Name",
      cell: (p) => (
        <Link
          to="/hosting/plans/$id"
          params={{ id: p.id }}
          className="flex flex-col"
        >
          <span className="font-medium text-brand-600 hover:underline">{p.display_name || p.name}</span>
          <span className="font-mono text-xs text-ink-3">{p.name}</span>
        </Link>
      ),
    },
    {
      key: "price",
      header: "Price",
      cell: (p) => (
        <span className="font-mono">
          ${p.monthly_price.toFixed(2)}<span className="text-ink-3">/mo</span>
        </span>
      ),
    },
    {
      key: "resources",
      header: "Resources",
      cell: (p) => (
        <div className="flex flex-col gap-0.5 text-xs">
          <span className="font-mono">{p.disk_quota_mb} MB disk</span>
          <span className="font-mono">{p.bandwidth_gb} GB bandwidth</span>
        </div>
      ),
    },
    {
      key: "limits",
      header: "Limits",
      cell: (p) => (
        <div className="flex flex-wrap gap-1">
          <span className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs">
            {p.max_domains} domains
          </span>
          <span className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs">
            {p.max_users} users
          </span>
          <span className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs">
            {p.max_ssl} SSL
          </span>
        </div>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (p) => (
        <div className="flex items-center gap-2">
          <StatusPill tone={p.is_active ? "success" : "neutral"}>
            {p.is_active ? "Active" : "Inactive"}
          </StatusPill>
          {p.is_default && (
            <span className="rounded bg-brand-500/10 px-1.5 py-0.5 text-xs font-medium text-brand-600">
              Default
            </span>
          )}
        </div>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (p) => <span className="font-mono text-xs text-ink-2">{formatDate(p.created_at)}</span>,
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (p) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          {p.is_active ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setConfirmModal({ type: "deactivate", plan: p })}
              disabled={p.is_default}
            >
              Deactivate
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setConfirmModal({ type: "activate", plan: p })}
            >
              Activate
            </Button>
          )}
          {!p.is_default && (
            <Button
              variant="ghost"
              size="sm"
              className="text-danger"
              onClick={() => setConfirmModal({ type: "delete", plan: p })}
            >
              Delete
            </Button>
          )}
        </div>
      ),
    },
  ];

  const confirmModalContent = confirmModal ? {
    activate: {
      title: "Activate Plan",
      description: `Are you sure you want to activate "${confirmModal.plan.display_name || confirmModal.plan.name}"? This will make it available for new accounts.`,
      confirmText: "Activate",
      confirmVariant: "primary" as const,
    },
    deactivate: {
      title: "Deactivate Plan",
      description: `Are you sure you want to deactivate "${confirmModal.plan.display_name || confirmModal.plan.name}"? Existing accounts will not be affected, but new accounts cannot use this plan.`,
      confirmText: "Deactivate",
      confirmVariant: "secondary" as const,
    },
    delete: {
      title: "Delete Plan",
      description: `Are you sure you want to delete "${confirmModal.plan.display_name || confirmModal.plan.name}"? This action cannot be undone.`,
      confirmText: "Delete",
      confirmVariant: "danger" as const,
    },
  }[confirmModal.type] : null;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Hosting Plans"
        description={`${filteredPlans.length} plan${filteredPlans.length === 1 ? "" : "s"} configured`}
        actions={
          <Button variant="primary" onClick={() => navigate({ to: "/hosting/plans/new" })}>
            New plan
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
              placeholder="Search by name or description…"
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
              <option value="inactive">Inactive</option>
            </Select>
          </div>
        </div>

        {q.isError ? (
          <ErrorState
            description="Failed to load plans."
            onRetry={() => q.refetch()}
          />
        ) : (
          <>
            <Table
              columns={columns}
              rows={paginatedPlans}
              keyOf={(p) => p.id}
              isLoading={q.isLoading}
              emptyState={
                <EmptyState
                  title={searchQuery || statusFilter !== "all" ? "No plans match your filters" : "No plans yet"}
                  description={
                    searchQuery || statusFilter !== "all"
                      ? "Try adjusting your search or filters."
                      : "Create your first hosting plan to get started."
                  }
                  action={
                    !searchQuery && statusFilter === "all" && (
                      <Button variant="primary" onClick={() => navigate({ to: "/hosting/plans/new" })}>
                        Create plan
                      </Button>
                    )
                  }
                />
              }
              onRowClick={(p) => navigate({ to: "/hosting/plans/$id", params: { id: p.id } })}
            />

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredPlans.length)} of{" "}
                  {filteredPlans.length} plans
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
              This action is irreversible and will permanently delete the plan.
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
}