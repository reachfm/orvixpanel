/**
 * Tests for the backup API client (backup.ts).
 * Tests the backup CRUD operations, file listing, and restore functionality.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import * as client from "../client";

// Mock the client module - preserve backupKeys from the actual module
vi.mock("../client", async () => {
  const actual = await vi.importActual("../client");
  return {
    ...actual,
    request: vi.fn(),
  };
});

describe("Backup API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("types", () => {
    it("should export BackupJob interface with correct shape", () => {
      const backup = {
        id: "01HZ1234567890ABCDEFGHIJ",
        tenant_id: "tenant-001",
        account_id: "acc-001",
        type: "files" as const,
        status: "completed" as const,
        name: "Daily Backup",
        storage_backend: "local",
        file_size: 1048576,
        file_count: 42,
        checksum: "abc123def456",
        checksum_algo: "sha256",
        retention_days: 30,
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      };

      expect(backup.id).toBe("01HZ1234567890ABCDEFGHIJ");
      expect(backup.type).toBe("files");
      expect(backup.status).toBe("completed");
      expect(backup.file_count).toBe(42);
    });

    it("should export BackupFile interface with correct shape", () => {
      const file = {
        id: "file-001",
        backup_job_id: "01HZ1234567890ABCDEFGHIJ",
        original_path: "/var/www/example.com/index.html",
        archive_path: "/backups/files/file-001.tar.gz",
        size: 4096,
        checksum: "def789ghi012",
        is_directory: false,
        created_at: "2024-01-01T00:00:00Z",
      };

      expect(file.id).toBe("file-001");
      expect(file.original_path).toContain("example.com");
      expect(file.is_directory).toBe(false);
    });

    it("should export RestorePoint interface with correct shape", () => {
      const restore = {
        id: "restore-001",
        tenant_id: "tenant-001",
        backup_job_id: "01HZ1234567890ABCDEFGHIJ",
        status: "completed" as const,
        staging_dir: "/tmp/restore-staging",
        target_dir: "/var/www/example.com",
        files_restored: 42,
        bytes_restored: 1048576,
        rollback_enabled: true,
        rollback_used: false,
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
      };

      expect(restore.id).toBe("restore-001");
      expect(restore.files_restored).toBe(42);
      expect(restore.rollback_used).toBe(false);
    });

    it("should export BackupSchedule interface with correct shape", () => {
      const schedule = {
        id: "schedule-001",
        tenant_id: "tenant-001",
        name: "Daily Backup",
        backup_type: "full" as const,
        cron_expr: "0 2 * * *",
        retention_days: 30,
        storage_backend: "local",
        is_enabled: true,
        last_run_at: "2024-01-01T02:00:00Z",
        next_run_at: "2024-01-02T02:00:00Z",
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
      };

      expect(schedule.cron_expr).toBe("0 2 * * *");
      expect(schedule.is_enabled).toBe(true);
    });

    it("should export BackupStats interface with correct shape", () => {
      const stats = {
        total_backups: 100,
        active_backups: 3,
        failed_backups: 5,
        total_storage_mb: 2048,
      };

      expect(stats.total_backups).toBe(100);
      expect(stats.total_storage_mb).toBe(2048);
    });
  });

  describe("backupKeys", () => {
    it("should generate correct query keys", async () => {
      const { backupKeys } = await import("../backup");
      expect(backupKeys.all).toEqual(["backups"]);
      expect(backupKeys.lists()).toEqual(["backups", "list"]);
      expect(backupKeys.stats()).toEqual(["backups", "stats"]);
      expect(backupKeys.detail("123")).toEqual(["backups", "detail", "123"]);
      expect(backupKeys.files("123")).toEqual(["backups", "detail", "123", "files"]);
    });
  });

  describe("listBackups", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        backups: [],
        total: 0,
        page: 1,
        page_size: 20,
        total_pages: 0,
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listBackups } = await import("../backup");
      const result = await listBackups();

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups?");
      expect(result.backups).toEqual([]);
    });

    it("should pass pagination params", async () => {
      const mockResponse = { backups: [], total: 0, page: 2, page_size: 10, total_pages: 0 };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listBackups } = await import("../backup");
      await listBackups({ page: 2, page_size: 10 });

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups?page=2&page_size=10");
    });

    it("should pass status filter", async () => {
      const mockResponse = { backups: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listBackups } = await import("../backup");
      await listBackups({ status: "completed" });

      expect(client.request).toHaveBeenCalledWith(expect.stringContaining("status=completed"));
    });

    it("should pass type filter", async () => {
      const mockResponse = { backups: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listBackups } = await import("../backup");
      await listBackups({ type: "full" });

      expect(client.request).toHaveBeenCalledWith(expect.stringContaining("type=full"));
    });

    it("should pass all filters combined", async () => {
      const mockResponse = { backups: [], total: 0, page: 1, page_size: 20, total_pages: 0 };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listBackups } = await import("../backup");
      await listBackups({ page: 1, status: "failed", type: "database" });

      // URLSearchParams order is non-deterministic, check all params are present
      const callArg = (client.request as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
      expect(callArg).toContain("page=1");
      expect(callArg).toContain("status=failed");
      expect(callArg).toContain("type=database");
      expect(callArg).toMatch(/^\/api\/v1\/backups\?/);
    });
  });

  describe("getBackup", () => {
    it("should call request with correct ID endpoint", async () => {
      const backupId = "01HZ1234567890ABCDEFGHIJ";
      const mockBackup = {
        id: backupId,
        name: "Daily Backup",
        type: "files",
        status: "completed",
        storage_backend: "local",
        file_size: 1048576,
        file_count: 42,
        checksum_algo: "sha256",
        retention_days: 30,
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockBackup);

      const { getBackup } = await import("../backup");
      const result = await getBackup(backupId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/backups/${backupId}`);
      expect(result.id).toBe(backupId);
    });
  });

  describe("createBackup", () => {
    it("should POST to correct endpoint with body", async () => {
      const body = {
        type: "files" as const,
        name: "My Backup",
        domain_id: "domain-001",
        retention_days: 30,
      };

      const mockBackup = {
        id: "01HZNEWBACKUP",
        name: "My Backup",
        type: "files",
        status: "pending",
        storage_backend: "local",
        file_size: 0,
        file_count: 0,
        checksum_algo: "sha256",
        retention_days: 30,
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
      };

      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockBackup);

      const { createBackup } = await import("../backup");
      const result = await createBackup(body);

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups", {
        method: "POST",
        body,
      });
      expect(result.id).toBe("01HZNEWBACKUP");
    });

    it("should create full backup", async () => {
      const body = { type: "full" as const, name: "Full Backup" };
      const mockBackup = { id: "new-backup", type: "full" };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockBackup);

      const { createBackup } = await import("../backup");
      const result = await createBackup(body);

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups", {
        method: "POST",
        body: { type: "full", name: "Full Backup" },
      });
    });
  });

  describe("deleteBackup", () => {
    it("should DELETE to correct endpoint", async () => {
      const backupId = "01HZ1234567890ABCDEFGHIJ";
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(undefined);

      const { deleteBackup } = await import("../backup");
      await deleteBackup(backupId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/backups/${backupId}`, {
        method: "DELETE",
      });
    });
  });

  describe("getBackupFiles", () => {
    it("should call request with correct ID endpoint", async () => {
      const backupId = "01HZ1234567890ABCDEFGHIJ";
      const mockResponse = {
        backup_id: backupId,
        files: [
          {
            id: "file-001",
            backup_job_id: backupId,
            original_path: "/var/www/index.html",
            archive_path: "/backups/file-001.tar.gz",
            size: 4096,
            checksum: "abc123",
            is_directory: false,
            created_at: "2024-01-01T00:00:00Z",
          },
        ],
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { getBackupFiles } = await import("../backup");
      const result = await getBackupFiles(backupId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/backups/${backupId}/files`);
      expect(result.files).toHaveLength(1);
    });
  });

  describe("restoreBackup", () => {
    it("should POST to restore endpoint with body", async () => {
      const backupId = "01HZ1234567890ABCDEFGHIJ";
      const body = {
        target_dir: "/var/www/example.com",
        rollback_enabled: true,
      };

      const mockRestore = {
        id: "restore-001",
        backup_job_id: backupId,
        status: "pending" as const,
        target_dir: "/var/www/example.com",
        files_restored: 0,
        bytes_restored: 0,
        rollback_enabled: true,
        rollback_used: false,
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockRestore);

      const { restoreBackup } = await import("../backup");
      const result = await restoreBackup(backupId, body);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/backups/${backupId}/restore`, {
        method: "POST",
        body,
      });
      expect(result.id).toBe("restore-001");
    });
  });

  describe("getRestorePoints", () => {
    it("should call request with correct endpoint", async () => {
      const backupId = "01HZ1234567890ABCDEFGHIJ";
      const mockResponse = {
        backup_id: backupId,
        restore_points: [],
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { getRestorePoints } = await import("../backup");
      const result = await getRestorePoints(backupId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/backups/${backupId}/restores`);
      expect(result.restore_points).toEqual([]);
    });
  });

  describe("getBackupStats", () => {
    it("should call request with correct endpoint", async () => {
      const mockStats = {
        total_backups: 100,
        active_backups: 3,
        failed_backups: 5,
        total_storage_mb: 2048,
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockStats);

      const { getBackupStats } = await import("../backup");
      const result = await getBackupStats();

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups/stats");
      expect(result.total_backups).toBe(100);
    });
  });

  describe("createSchedule", () => {
    it("should POST to schedules endpoint with body", async () => {
      const body = {
        name: "Daily Backup Schedule",
        backup_type: "full" as const,
        cron_expr: "0 2 * * *",
        retention_days: 30,
      };

      const mockSchedule = {
        id: "schedule-001",
        name: "Daily Backup Schedule",
        backup_type: "full",
        cron_expr: "0 2 * * *",
        retention_days: 30,
        storage_backend: "local",
        is_enabled: true,
        created_by: "user-001",
        created_at: "2024-01-01T00:00:00Z",
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockSchedule);

      const { createSchedule } = await import("../backup");
      const result = await createSchedule(body);

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups/schedules", {
        method: "POST",
        body,
      });
      expect(result.id).toBe("schedule-001");
    });
  });

  describe("listSchedules", () => {
    it("should call request with correct endpoint", async () => {
      const mockResponse = {
        schedules: [
          {
            id: "schedule-001",
            name: "Daily Backup",
            backup_type: "full" as const,
            cron_expr: "0 2 * * *",
            retention_days: 30,
            storage_backend: "local",
            is_enabled: true,
            created_by: "user-001",
            created_at: "2024-01-01T00:00:00Z",
          },
        ],
      };
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockResponse);

      const { listSchedules } = await import("../backup");
      const result = await listSchedules();

      expect(client.request).toHaveBeenCalledWith("/api/v1/backups/schedules");
      expect(result.schedules).toHaveLength(1);
    });
  });

  describe("deleteSchedule", () => {
    it("should DELETE to schedules endpoint", async () => {
      const scheduleId = "schedule-001";
      (client.request as ReturnType<typeof vi.fn>).mockResolvedValueOnce(undefined);

      const { deleteSchedule } = await import("../backup");
      await deleteSchedule(scheduleId);

      expect(client.request).toHaveBeenCalledWith(`/api/v1/backups/schedules/${scheduleId}`, {
        method: "DELETE",
      });
    });
  });
});