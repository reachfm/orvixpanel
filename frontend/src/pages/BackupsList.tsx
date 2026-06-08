/**
 * Backups list page.
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
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
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

function formatBytes(bytes: number): string {
  if (bytes === 0) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  let size = bytes;
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024;
    i++;
  }
  return `${size.toFixed(1)} ${units[i]}`;
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return "—";
  return new Date(dateStr).toLocaleString();
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

  // Table columns
  const columns: Column<BackupJob>[] = [
    {
      key: "name",
      header: "Name",
      cell: (backup) => (
        <div>
          <div className="font-medium">{backup.name || "Unnamed Backup"}</div>
          <div className="text-xs text-gray-500">{backup.id}</div>
        </div>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (backup) => (
        <span className="capitalize text-gray-600 dark:text-gray-400">
          {backup.type}
        </span>
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
        <span className="text-gray-600 dark:text-gray-400">
          {formatBytes(backup.file_size)}
        </span>
      ),
    },
    {
      key: "files",
      header: "Files",
      cell: (backup) => (
        <span className="text-gray-600 dark:text-gray-400">
          {backup.file_count.toLocaleString()}
        </span>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (backup) => (
        <span className="text-gray-600 dark:text-gray-400">
          {formatDate(backup.created_at)}
        </span>
      ),
    },
    {
      key: "actions",
      header: "Actions",
      cell: (backup) => (
        <div className="flex items-center gap-2">
          <Link
            to="/backup/$id"
            params={{ id: backup.id }}
            className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 text-sm"
          >
            Details
          </Link>
          <Button
            variant="ghost"
            size="sm"
            className="text-red-600 hover:text-red-800 dark:text-red-400"
            onClick={() => setDeleteModal(backup)}
          >
            Delete
          </Button>
        </div>
      ),
    },
  ];

  if (q.isLoading) return <LoadingState />;
  if (q.isError) return <ErrorState description="Failed to load backups" onRetry={() => q.refetch()} />;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Backups"
        description="Manage automated and manual backups with restore capability"
        actions={
          <Button onClick={() => setCreateModalOpen(true)}>Create Backup</Button>
        }
      />

      {/* Stats Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card className="p-4">
          <div className="text-sm text-gray-500">Total Backups</div>
          <div className="text-2xl font-bold">
            {statsQ.isLoading ? "—" : statsQ.data?.total_backups ?? 0}
          </div>
        </Card>
        <Card className="p-4">
          <div className="text-sm text-gray-500">Active</div>
          <div className="text-2xl font-bold text-green-600">
            {statsQ.isLoading ? "—" : statsQ.data?.active_backups ?? 0}
          </div>
        </Card>
        <Card className="p-4">
          <div className="text-sm text-gray-500">Failed</div>
          <div className="text-2xl font-bold text-red-600">
            {statsQ.isLoading ? "—" : statsQ.data?.failed_backups ?? 0}
          </div>
        </Card>
        <Card className="p-4">
          <div className="text-sm text-gray-500">Storage Used</div>
          <div className="text-2xl font-bold">
            {statsQ.isLoading ? "—" : `${statsQ.data?.total_storage_mb ?? 0} MB`}
          </div>
        </Card>
      </div>

      {/* Filters */}
      <Card className="p-4">
        <div className="flex flex-wrap gap-4">
          <div className="flex-1 min-w-[200px]">
            <Input
              placeholder="Search backups..."
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
            />
          </div>
          <Select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setCurrentPage(1);
            }}
            className="w-40"
          >
            <option value="all">All Status</option>
            <option value="pending">Pending</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </Select>
          <Select
            value={typeFilter}
            onChange={(e) => {
              setTypeFilter(e.target.value);
              setCurrentPage(1);
            }}
            className="w-40"
          >
            <option value="all">All Types</option>
            <option value="full">Full</option>
            <option value="files">Files</option>
            <option value="database">Database</option>
          </Select>
        </div>
      </Card>

      {/* Table */}
      {filteredBackups.length === 0 ? (
        <EmptyState
          title="No backups found"
          description={
            searchQuery || statusFilter !== "all" || typeFilter !== "all"
              ? "Try adjusting your filters"
              : "Create your first backup to get started"
          }
          action={
            !searchQuery && statusFilter === "all" && typeFilter === "all" ? (
              <Button onClick={() => setCreateModalOpen(true)}>Create Backup</Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <Table<BackupJob> rows={filteredBackups} columns={columns} keyOf={(b) => b.id} />
          {totalPages > 1 && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === 1}
                onClick={() => setCurrentPage((p) => p - 1)}
              >
                Previous
              </Button>
              <span className="px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
                Page {currentPage} of {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === totalPages}
                onClick={() => setCurrentPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          )}
        </>
      )}

      {/* Create Backup Modal */}
      <Modal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title="Create Backup"
        footer={
          <>
            <Button variant="outline" onClick={() => setCreateModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleCreateBackup}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Backup Type *</label>
            <Select
              value={newBackupType}
              onChange={(e) => setNewBackupType(e.target.value as BackupType)}
            >
              <option value="files">Files Only</option>
              <option value="database">Database Only</option>
              <option value="full">Full Backup</option>
            </Select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Name (Optional)</label>
            <Input
              placeholder="My Backup"
              value={newBackupName}
              onChange={(e) => setNewBackupName(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Domain ID (Optional)</label>
            <Input
              placeholder="Domain to backup"
              value={newBackupDomainID}
              onChange={(e) => setNewBackupDomainID(e.target.value)}
            />
            <p className="mt-1 text-xs text-gray-500">Leave empty to backup all accessible data</p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Retention Days</label>
            <Input
              type="number"
              min={1}
              max={365}
              value={newBackupRetention}
              onChange={(e) => setNewBackupRetention(parseInt(e.target.value) || 30)}
            />
            <p className="mt-1 text-xs text-gray-500">Backup will be automatically deleted after this period</p>
          </div>
          {createMutation.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              Failed to create backup. Please try again.
            </p>
          )}
        </div>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        open={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        title="Delete Backup"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeleteModal(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={() => deleteModal && deleteMutation.mutate(deleteModal.id)}
              loading={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to delete backup <strong>{deleteModal?.name || deleteModal?.id}</strong>?
          This action cannot be undone.
        </p>
      </Modal>
    </div>
  );
}

// Re-export types
export type { BackupJob, BackupStatus, BackupType } from "@/lib/api/backup";