/**
 * Backups list page. Professional cPanel-style backup management.
 */

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Select } from "@/lib/ui/Select";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState, Spinner } from "@/lib/ui/Feedback";
import { formatDate, formatBytes } from "@/lib/utils";
import {
  backupKeys,
  listBackups,
  createBackup,
  deleteBackup,
  getBackupStats,
  type BackupJob,
  type BackupStatus,
  type BackupType,
  type CreateBackupRequest,
} from "@/lib/api/backup";

const PAGE_SIZE = 20;

function getStatusTone(status: BackupStatus): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "completed":
      return "success";
    case "running":
      return "neutral";
    case "pending":
      return "neutral";
    case "failed":
      return "danger";
    case "canceled":
      return "warning";
    default:
      return "neutral";
  }
}

export function BackupsListPage() {
  const qc = useQueryClient();

  // Filter and pagination state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);

  // Modal state
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [deleteModal, setDeleteModal] = useState<BackupJob | null>(null);

  // Create form state
  const [newBackupType, setNewBackupType] = useState<BackupType>("files");
  const [newBackupName, setNewBackupName] = useState("");
  const [newBackupDomainID, setNewBackupDomainID] = useState("");
  const [newBackupRetention, setNewBackupRetention] = useState(30);

  // Fetch backups
  const q = useQuery({
    queryKey: backupKeys.list({ page: String(currentPage), status: statusFilter, type: typeFilter }),
    queryFn: () =>
      listBackups({
        page: currentPage,
        page_size: PAGE_SIZE,
        status: statusFilter === "all" ? undefined : statusFilter,
        type: typeFilter === "all" ? undefined : typeFilter,
      }),
  });

  // Fetch stats
  const statsQ = useQuery({
    queryKey: backupKeys.stats(),
    queryFn: getBackupStats,
  });

  const backups = q.data?.backups ?? [];
  const totalPages = q.data?.total_pages ?? 1;

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateBackupRequest) => createBackup(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: backupKeys.all });
      qc.invalidateQueries({ queryKey: backupKeys.stats() });
      setCreateModalOpen(false);
      resetCreateForm();
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteBackup(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: backupKeys.all });
      qc.invalidateQueries({ queryKey: backupKeys.stats() });
      setDeleteModal(null);
    },
  });

  // Filter backups
  const filteredBackups = useMemo(() => {
    if (!searchQuery) return backups;
    const query = searchQuery.toLowerCase();
    return backups.filter(
      (b) =>
        b.name?.toLowerCase().includes(query) ||
        b.id.toLowerCase().includes(query) ||
        b.storage_path?.toLowerCase().includes(query)
    );
  }, [backups, searchQuery]);

  // Reset page when filters change
  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1);
  };

  // Form handlers
  const resetCreateForm = () => {
    setNewBackupType("files");
    setNewBackupName("");
    setNewBackupDomainID("");
    setNewBackupRetention(30);
  };

  const handleCreateBackup = () => {
    createMutation.mutate({
      type: newBackupType,
      name: newBackupName || undefined,
      domain_id: newBackupDomainID || undefined,
      retention_days: newBackupRetention,
    });
  };

  const isPending = createMutation.isPending || deleteMutation.isPending;

  // Table columns
  const columns: Column<BackupJob>[] = [
    {
      key: "name",
      header: "Name",
      cell: (backup) => (
        <div>
          <div className="font-medium text-ink-1">{backup.name || "Unnamed Backup"}</div>
          <div className="font-mono text-xs text-ink-3">{backup.id}</div>
        </div>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (backup) => (
        <span className="capitalize text-ink-2">{backup.type}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (backup) => (
        <StatusPill tone={getStatusTone(backup.status)}>
          {backup.status.replace("_", " ")}
        </StatusPill>
      ),
    },
    {
      key: "size",
      header: "Size",
      cell: (backup) => (
        <span className="text-ink-2">{formatBytes(backup.file_size)}</span>
      ),
    },
    {
      key: "files",
      header: "Files",
      cell: (backup) => (
        <span className="text-ink-2">{backup.file_count.toLocaleString()}</span>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (backup) => (
        <span className="font-mono text-xs text-ink-2">{formatDate(backup.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (backup) => (
        <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
          <Link
            to="/backup/$id"
            params={{ id: backup.id }}
            className="text-xs font-medium text-brand-600 hover:underline"
          >
            Details
          </Link>
          <Button
            variant="ghost"
            size="sm"
            className="text-danger hover:text-danger"
            onClick={() => setDeleteModal(backup)}
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
        title="Backups"
        description={`${backups.length} backup${backups.length === 1 ? "" : "s"} configured`}
        actions={
          <Button variant="primary" onClick={() => setCreateModalOpen(true)}>
            Create Backup
          </Button>
        }
      />

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">Total Backups</div>
          <div className="mt-1.5 text-2xl font-bold text-ink-1">
            {statsQ.isLoading ? <Spinner size={16} /> : statsQ.data?.total_backups ?? 0}
          </div>
        </Card>
        <Card>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">Active</div>
          <div className="mt-1.5 text-2xl font-bold text-success">
            {statsQ.isLoading ? <Spinner size={16} /> : statsQ.data?.active_backups ?? 0}
          </div>
        </Card>
        <Card>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">Failed</div>
          <div className="mt-1.5 text-2xl font-bold text-danger">
            {statsQ.isLoading ? <Spinner size={16} /> : statsQ.data?.failed_backups ?? 0}
          </div>
        </Card>
        <Card>
          <div className="text-[11px] font-semibold uppercase tracking-wider text-ink-3">Storage Used</div>
          <div className="mt-1.5 text-2xl font-bold text-ink-1">
            {statsQ.isLoading ? <Spinner size={16} /> : `${statsQ.data?.total_storage_mb ?? 0} MB`}
          </div>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1">
            <Input
              label="Search"
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Search backups..."
            />
          </div>
          <div className="w-full sm:w-36">
            <Select
              label="Status"
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setCurrentPage(1);
              }}
            >
              <option value="all">All statuses</option>
              <option value="pending">Pending</option>
              <option value="running">Running</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
            </Select>
          </div>
          <div className="w-full sm:w-36">
            <Select
              label="Type"
              value={typeFilter}
              onChange={(e) => {
                setTypeFilter(e.target.value);
                setCurrentPage(1);
              }}
            >
              <option value="all">All types</option>
              <option value="full">Full</option>
              <option value="files">Files</option>
              <option value="database">Database</option>
            </Select>
          </div>
        </div>

        {q.isError ? (
          <ErrorState description="Failed to load backups" onRetry={() => q.refetch()} />
        ) : q.isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner size={24} />
          </div>
        ) : filteredBackups.length === 0 ? (
          <EmptyState
            title={searchQuery || statusFilter !== "all" || typeFilter !== "all" ? "No backups match your filters" : "No backups yet"}
            description={
              searchQuery || statusFilter !== "all" || typeFilter !== "all"
                ? "Try adjusting your search or filters."
                : "Create your first backup to protect your data."
            }
            action={
              !searchQuery && statusFilter === "all" && typeFilter === "all" && (
                <Button variant="primary" onClick={() => setCreateModalOpen(true)}>
                  Create Backup
                </Button>
              )
            }
          />
        ) : (
          <>
            <Table
              rows={filteredBackups}
              columns={columns}
              keyOf={(b) => b.id}
            />
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-4">
                <div className="text-sm text-ink-3">
                  Showing {(currentPage - 1) * PAGE_SIZE + 1} to{" "}
                  {Math.min(currentPage * PAGE_SIZE, filteredBackups.length)} of{" "}
                  {filteredBackups.length} backups
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

      {/* Create Backup Modal */}
      <Modal
        open={createModalOpen}
        onClose={() => !isPending && setCreateModalOpen(false)}
        title="Create Backup"
        description="Start a new backup job for files and/or database."
        footer={
          <>
            <Button variant="secondary" onClick={() => setCreateModalOpen(false)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleCreateBackup}
              disabled={createMutation.isPending}
              loading={createMutation.isPending}
            >
              Create Backup
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <Select
            label="Backup Type *"
            value={newBackupType}
            onChange={(e) => setNewBackupType(e.target.value as BackupType)}
          >
            <option value="files">Files Only</option>
            <option value="database">Database Only</option>
            <option value="full">Full Backup</option>
          </Select>
          <Input
            label="Name (Optional)"
            placeholder="My Backup"
            value={newBackupName}
            onChange={(e) => setNewBackupName(e.target.value)}
          />
          <Input
            label="Domain ID (Optional)"
            placeholder="Domain to backup"
            value={newBackupDomainID}
            onChange={(e) => setNewBackupDomainID(e.target.value)}
            description="Leave empty to backup all accessible data"
          />
          <Input
            label="Retention Days"
            type="number"
            min={1}
            max={365}
            value={newBackupRetention}
            onChange={(e) => setNewBackupRetention(parseInt(e.target.value) || 30)}
            description="Backup will be automatically deleted after this period"
          />
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => !isPending && setDeleteModal(null)}
        title="Delete Backup"
        description={`Are you sure you want to delete "${deleteModal?.name || deleteModal?.id}"?`}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleteModal(null)} disabled={isPending}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={() => deleteModal && deleteMutation.mutate(deleteModal.id)}
              loading={deleteMutation.isPending}
            >
              Delete Backup
            </Button>
          </>
        }
      >
        <div className="rounded-md border border-danger/30 bg-danger/5 p-3 text-sm text-danger">
          This will permanently delete the backup. This action cannot be undone.
        </div>
      </Modal>
    </div>
  );
}

// Re-export types
export type { BackupJob, BackupStatus, BackupType } from "@/lib/api/backup";