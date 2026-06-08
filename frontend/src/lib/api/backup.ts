/**
 * Backup API client.
 */

import { request } from "./client";

// Types
export type BackupStatus = "pending" | "running" | "completed" | "failed" | "canceled";
export type BackupType = "full" | "files" | "database";

export interface BackupJob {
  id: string;
  tenant_id: string;
  account_id?: string;
  domain_id?: string;
  type: BackupType;
  status: BackupStatus;
  name?: string;
  description?: string;
  storage_backend: string;
  storage_path?: string;
  file_size: number;
  file_count: number;
  checksum?: string;
  checksum_algo: string;
  retention_days: number;
  expires_at?: string;
  error_message?: string;
  scheduled_at?: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface BackupFile {
  id: string;
  backup_job_id: string;
  original_path: string;
  archive_path: string;
  size: number;
  checksum: string;
  is_directory: boolean;
  created_at: string;
}

export interface RestorePoint {
  id: string;
  tenant_id: string;
  account_id?: string;
  domain_id?: string;
  backup_job_id: string;
  status: BackupStatus;
  staging_dir?: string;
  target_dir?: string;
  files_restored: number;
  bytes_restored: number;
  rollback_enabled: boolean;
  rollback_used: boolean;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
}

export interface BackupSchedule {
  id: string;
  tenant_id: string;
  account_id?: string;
  domain_id?: string;
  name: string;
  description?: string;
  backup_type: BackupType;
  cron_expr: string;
  retention_days: number;
  storage_backend: string;
  is_enabled: boolean;
  last_run_at?: string;
  next_run_at?: string;
  created_by: string;
  created_at: string;
}

export interface BackupStats {
  total_backups: number;
  active_backups: number;
  failed_backups: number;
  total_storage_mb: number;
}

export interface CreateBackupRequest {
  account_id?: string;
  domain_id?: string;
  type: BackupType;
  name?: string;
  description?: string;
  storage_backend?: string;
  retention_days?: number;
  source_path?: string;
}

export interface RestoreBackupRequest {
  target_dir: string;
  files_to_restore?: string[];
  rollback_enabled?: boolean;
}

export interface CreateScheduleRequest {
  account_id?: string;
  domain_id?: string;
  name: string;
  description?: string;
  backup_type: BackupType;
  cron_expr: string;
  retention_days?: number;
  storage_backend?: string;
}

export interface ListBackupsResponse {
  backups: BackupJob[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// API Functions
export const backupKeys = {
  all: ["backups"] as const,
  lists: () => [...backupKeys.all, "list"] as const,
  list: (filters?: Record<string, string>) => [...backupKeys.lists(), filters] as const,
  stats: () => [...backupKeys.all, "stats"] as const,
  details: () => [...backupKeys.all, "detail"] as const,
  detail: (id: string) => [...backupKeys.details(), id] as const,
  files: (id: string) => [...backupKeys.detail(id), "files"] as const,
  restores: (id: string) => [...backupKeys.detail(id), "restores"] as const,
  schedules: () => [...backupKeys.all, "schedules"] as const,
};

export async function listBackups(params?: {
  page?: number;
  page_size?: number;
  type?: string;
  status?: string;
}): Promise<ListBackupsResponse> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.page_size) searchParams.set("page_size", String(params.page_size));
  if (params?.type) searchParams.set("type", params.type);
  if (params?.status) searchParams.set("status", params.status);

  return request<ListBackupsResponse>(`/api/v1/backups?${searchParams.toString()}`);
}

export async function getBackup(id: string): Promise<BackupJob> {
  return request<BackupJob>(`/api/v1/backups/${id}`);
}

export async function createBackup(data: CreateBackupRequest): Promise<BackupJob> {
  return request<BackupJob>("/api/v1/backups", { method: "POST", body: data });
}

export async function deleteBackup(id: string): Promise<void> {
  await request(`/api/v1/backups/${id}`, { method: "DELETE" });
}

export async function getBackupFiles(id: string): Promise<{ backup_id: string; files: BackupFile[] }> {
  return request<{ backup_id: string; files: BackupFile[] }>(`/api/v1/backups/${id}/files`);
}

export async function restoreBackup(id: string, data: RestoreBackupRequest): Promise<RestorePoint> {
  return request<RestorePoint>(`/api/v1/backups/${id}/restore`, { method: "POST", body: data });
}

export async function getRestorePoints(id: string): Promise<{ backup_id: string; restore_points: RestorePoint[] }> {
  return request<{ backup_id: string; restore_points: RestorePoint[] }>(`/api/v1/backups/${id}/restores`);
}

export async function getBackupStats(): Promise<BackupStats> {
  return request<BackupStats>("/api/v1/backups/stats");
}

// Schedule API
export async function createSchedule(data: CreateScheduleRequest): Promise<BackupSchedule> {
  return request<BackupSchedule>("/api/v1/backups/schedules", { method: "POST", body: data });
}

export async function listSchedules(): Promise<{ schedules: BackupSchedule[] }> {
  return request<{ schedules: BackupSchedule[] }>("/api/v1/backups/schedules");
}

export async function deleteSchedule(id: string): Promise<void> {
  await request(`/api/v1/backups/schedules/${id}`, { method: "DELETE" });
}