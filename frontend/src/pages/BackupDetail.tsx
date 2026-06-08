/**
 * Backup detail page with restore functionality.
 */

import { useState } from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Modal } from "@/lib/ui/Modal";
import { Table, type Column } from "@/lib/ui/Table";
import { StatusPill } from "@/lib/ui/StatusPill";
import { EmptyState, ErrorState, LoadingState } from "@/lib/ui/Feedback";
import {
  backupKeys,
  getBackup,
  getBackupFiles,
  getRestorePoints,
  restoreBackup,
  deleteBackup,
  type BackupFile,
  type RestorePoint,
  type RestoreBackupRequest,
} from "@/lib/api/backup";

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

function getStatusTone(status: string): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "completed":
      return "success";
    case "running":
      return "neutral";
    case "pending":
      return "neutral";
    case "failed":
      return "danger";
    default:
      return "neutral";
  }
}

export function BackupDetailPage() {
  const { id } = useParams({ strict: false }) as { id: string };
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Modal state
  const [restoreModalOpen, setRestoreModalOpen] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  // Restore form state
  const [restoreTargetDir, setRestoreTargetDir] = useState("");
  const [rollbackEnabled, setRollbackEnabled] = useState(true);

  // Fetch backup
  const backupQ = useQuery({
    queryKey: backupKeys.detail(id),
    queryFn: () => getBackup(id),
  });

  // Fetch files
  const filesQ = useQuery({
    queryKey: backupKeys.files(id),
    queryFn: () => getBackupFiles(id),
  });

  // Fetch restore points
  const restoresQ = useQuery({
    queryKey: backupKeys.restores(id),
    queryFn: () => getRestorePoints(id),
  });

  // Restore mutation
  const restoreMutation = useMutation({
    mutationFn: (data: RestoreBackupRequest) => restoreBackup(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: backupKeys.restores(id) });
      setRestoreModalOpen(false);
      resetRestoreForm();
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => deleteBackup(id),
    onSuccess: () => {
      navigate({ to: "/backup" });
    },
  });

  const backup = backupQ.data;
  const files = filesQ.data?.files ?? [];
  const restorePoints = restoresQ.data?.restore_points ?? [];

  // Form handlers
  const resetRestoreForm = () => {
    setRestoreTargetDir("");
    setRollbackEnabled(true);
  };

  const handleRestore = () => {
    if (!restoreTargetDir.trim()) return;
    restoreMutation.mutate({
      target_dir: restoreTargetDir.trim(),
      rollback_enabled: rollbackEnabled,
    });
  };

  // File table columns
  const fileColumns: Column<BackupFile>[] = [
    {
      key: "original_path",
      header: "Original Path",
      cell: (file) => (
        <span className="font-mono text-sm truncate max-w-[300px] block" title={file.original_path}>
          {file.original_path}
        </span>
      ),
    },
    {
      key: "size",
      header: "Size",
      cell: (file) => (
        <span className="text-gray-600 dark:text-gray-400">{formatBytes(file.size)}</span>
      ),
    },
    {
      key: "checksum",
      header: "Checksum",
      cell: (file) => (
        <span className="font-mono text-xs text-gray-500 truncate max-w-[150px] block" title={file.checksum}>
          {file.checksum.substring(0, 16)}...
        </span>
      ),
    },
  ];

  // Restore point columns
  const restoreColumns: Column<RestorePoint>[] = [
    {
      key: "id",
      header: "ID",
      cell: (rp) => (
        <span className="font-mono text-sm">{rp.id.substring(0, 16)}...</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (rp) => (
        <StatusPill tone={getStatusTone(rp.status)}>
          {rp.status.replace("_", " ")}
        </StatusPill>
      ),
    },
    {
      key: "files_restored",
      header: "Files",
      cell: (rp) => <span>{rp.files_restored}</span>,
    },
    {
      key: "bytes_restored",
      header: "Size",
      cell: (rp) => <span>{formatBytes(rp.bytes_restored)}</span>,
    },
    {
      key: "rollback",
      header: "Rollback",
      cell: (rp) => (
        <span className={rp.rollback_used ? "text-red-600" : "text-gray-400"}>
          {rp.rollback_used ? "Used" : "Not used"}
        </span>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (rp) => <span className="text-gray-600 dark:text-gray-400">{formatDate(rp.created_at)}</span>,
    },
  ];

  if (backupQ.isLoading) return <LoadingState />;
  if (backupQ.isError) return <ErrorState description="Failed to load backup" onRetry={() => backupQ.refetch()} />;
  if (!backup) return <ErrorState description="Backup not found" />;

  return (
    <div className="space-y-6">
      <PageHeader
        title={backup.name || "Backup Details"}
        description={`Backup ID: ${backup.id}`}
        actions={
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setRestoreModalOpen(true)}>
              Restore
            </Button>
            <Button
              variant="danger"
              onClick={() => setDeleteModalOpen(true)}
            >
              Delete
            </Button>
          </div>
        }
      />

      {/* Backup Info */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card className="p-4">
          <h3 className="text-lg font-semibold mb-4">Backup Information</h3>
          <dl className="space-y-3">
            <div className="flex justify-between">
              <dt className="text-gray-500">Status</dt>
              <dd>
                <StatusPill tone={getStatusTone(backup.status)}>
                  {backup.status.replace("_", " ")}
                </StatusPill>
              </dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Type</dt>
              <dd className="capitalize">{backup.type}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Storage Backend</dt>
              <dd>{backup.storage_backend}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Retention</dt>
              <dd>{backup.retention_days} days</dd>
            </div>
            {backup.checksum && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Checksum</dt>
                <dd className="font-mono text-xs truncate max-w-[200px]" title={backup.checksum}>
                  {backup.checksum.substring(0, 32)}...
                </dd>
              </div>
            )}
          </dl>
        </Card>

        <Card className="p-4">
          <h3 className="text-lg font-semibold mb-4">File Statistics</h3>
          <dl className="space-y-3">
            <div className="flex justify-between">
              <dt className="text-gray-500">Total Files</dt>
              <dd className="font-semibold">{backup.file_count.toLocaleString()}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Total Size</dt>
              <dd className="font-semibold">{formatBytes(backup.file_size)}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Created</dt>
              <dd>{formatDate(backup.created_at)}</dd>
            </div>
            {backup.completed_at && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Completed</dt>
                <dd>{formatDate(backup.completed_at)}</dd>
              </div>
            )}
            {backup.expires_at && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Expires</dt>
                <dd>{formatDate(backup.expires_at)}</dd>
              </div>
            )}
          </dl>
        </Card>
      </div>

      {/* Error Message */}
      {backup.error_message && (
        <Card className="p-4 border-red-300 dark:border-red-800">
          <h3 className="text-lg font-semibold mb-2 text-red-600 dark:text-red-400">Error</h3>
          <p className="text-sm text-red-600 dark:text-red-400">{backup.error_message}</p>
        </Card>
      )}

      {/* Files */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Backed Up Files</h3>
        {filesQ.isLoading ? (
          <div className="py-8 text-center text-gray-500">Loading files...</div>
        ) : files.length === 0 ? (
          <EmptyState
            title="No files"
            description="File list is not available"
          />
        ) : (
          <Table<BackupFile> rows={files} columns={fileColumns} keyOf={(f) => f.id} />
        )}
      </Card>

      {/* Restore History */}
      <Card className="p-4">
        <h3 className="text-lg font-semibold mb-4">Restore History</h3>
        {restoresQ.isLoading ? (
          <div className="py-8 text-center text-gray-500">Loading restore history...</div>
        ) : restorePoints.length === 0 ? (
          <EmptyState
            title="No restores"
            description="This backup has never been restored"
          />
        ) : (
          <Table<RestorePoint> rows={restorePoints} columns={restoreColumns} keyOf={(r) => r.id} />
        )}
      </Card>

      {/* Restore Modal */}
      <Modal
        open={restoreModalOpen}
        onClose={() => setRestoreModalOpen(false)}
        title="Restore Backup"
        footer={
          <>
            <Button variant="outline" onClick={() => setRestoreModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleRestore}
              disabled={!restoreTargetDir.trim() || restoreMutation.isPending}
            >
              {restoreMutation.isPending ? "Restoring..." : "Restore"}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Target Directory *</label>
            <Input
              placeholder="/var/www/example.com"
              value={restoreTargetDir}
              onChange={(e) => setRestoreTargetDir(e.target.value)}
            />
            <p className="mt-1 text-xs text-gray-500">
              Files will be restored to this directory. Ensure it exists and is writable.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="rollback-enabled"
              checked={rollbackEnabled}
              onChange={(e) => setRollbackEnabled(e.target.checked)}
              className="rounded border-gray-300"
            />
            <label htmlFor="rollback-enabled" className="text-sm">
              Enable automatic rollback (recommended)
            </label>
          </div>
          <p className="text-xs text-gray-500">
            If enabled, a temporary backup of the current files will be created before restore.
            If the restore fails, the original files will be automatically restored.
          </p>
          {restoreMutation.isError && (
            <p className="text-sm text-red-600 dark:text-red-400">
              Restore failed. Please check the target directory and try again.
            </p>
          )}
        </div>
      </Modal>

      {/* Delete Modal */}
      <Modal
        open={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="Delete Backup"
        footer={
          <>
            <Button variant="outline" onClick={() => setDeleteModalOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={() => deleteMutation.mutate()}
              loading={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </Button>
          </>
        }
      >
        <p>
          Are you sure you want to delete this backup? This action cannot be undone.
        </p>
      </Modal>
    </div>
  );
}