/**
 * System / health / license / audit endpoints. All back the corresponding
 * v0.3.0 / Phase 2 admin pages.
 */

import { request } from "./client";

// --- health ------------------------------------------------------------

export interface HealthzResponse { status: "ok" }
export interface ReadyzResponse { status: "ready" | "db_unavailable" }

export function healthz(): Promise<HealthzResponse> {
  return request("/healthz", { noAuth: true, timeoutMs: 3000 });
}
export function readyz(): Promise<ReadyzResponse> {
  return request("/readyz", { noAuth: true, timeoutMs: 3000 });
}

// --- system / license / audit -----------------------------------------

export interface SystemInfo {
  name: string;
  version: string;
  uptime_at: string;
}
export function systemInfo(): Promise<SystemInfo> {
  return request("/api/v1/admin/system");
}

export interface License {
  tier: string;
  features: string[];
  max_servers?: number;
  expires_at?: number;
  issued_at?: number;
  grace_days?: number;
}
export function license(): Promise<License> {
  return request("/api/v1/admin/license");
}

export interface LicenseRenewal {
  // Present when license is loaded
  loaded: boolean;
  tier?: string;
  expires_at?: string;
  days_remaining?: number;
  grace_days?: number;
  status?: "active" | "grace" | "expired";
  mode?: "active" | "grace" | "locked";
  licensed_to?: string;
  issued_at?: string;
  grace_ends_at?: string;
  days_until_locked?: number;
  max_servers?: number;
  feature_count?: number;
  // Additional fields accepted loosely
  [key: string]: unknown;
}
export function licenseRenewal(): Promise<LicenseRenewal> {
  return request("/api/v1/admin/license/renewal-info");
}

export function uploadLicense(key: string): Promise<License> {
  return request("/api/v1/admin/license", { method: "PUT", body: { key } });
}

export interface AuditEntry {
  id: string;
  timestamp: string;
  user_id: string;
  user_email: string;
  action: string;
  resource_type: string;
  resource_id: string;
  resource_name: string;
  result: string;
  actor_ip: string;
  prev_hash: string;
  hash: string;
  details: string;
}
export function listAudit(limit = 100): Promise<{ entries: AuditEntry[] }> {
  return request(`/api/v1/admin/audit-log?limit=${limit}`);
}

export function searchAudit(body: {
  user_email?: string;
  action?: string;
  resource_type?: string;
  resource_id?: string;
  since?: string;
  until?: string;
  limit?: number;
}): Promise<{ entries: AuditEntry[]; next_cursor?: string }> {
  return request(`/api/v1/admin/audit-log/search`, { method: "POST", body });
}

export interface AuditVerifyResult {
  tampered: boolean;
  first_bad_row: number;
  error?: string;
}
export function verifyAudit(): Promise<AuditVerifyResult> {
  return request(`/api/v1/admin/audit-log/verify`, { method: "POST" });
}

// --- update manager ----------------------------------------------------

export interface UpdateStatus {
  current_version: string;
  current_commit: string;
  build_date: string;
  channel: string;
  health_endpoint: string;
  ready_endpoint: string;
  update_check_enabled: boolean;
  auto_update_enabled: boolean;
  update_history: UpdateHistoryEntry[];
}

export interface UpdateHistoryEntry {
  id: string;
  from_version: { tag: string; commit: string };
  to_version: { tag: string; commit: string };
  timestamp: string;
  channel: string;
  result: "success" | "failed" | "rolled_back" | "in_progress";
  backup_id?: string;
  rollback_backup_id?: string;
  error_message?: string;
  duration_seconds: number;
}

export interface UpdateCheckResult {
  update_available: boolean;
  current_version: string;
  latest_version: string;
  channel: string;
}

export interface PreflightCheck {
  name: string;
  status: "pass" | "warn" | "fail" | "skip" | "unknown";
  message: string;
  details?: string;
  suggestions?: string[];
}

export interface SystemHealthResult {
  checks: PreflightCheck[];
}

export function updateStatus(): Promise<UpdateStatus> {
  return request("/api/v1/admin/update/status");
}

export function checkForUpdates(channel = "stable"): Promise<UpdateCheckResult> {
  return request(`/api/v1/admin/update/check?channel=${channel}`, { method: "POST" });
}

export function installUpdate(channel = "stable"): Promise<{ status: string; history_id: string; message: string }> {
  return request("/api/v1/admin/update/install", { method: "POST", body: { channel } });
}

export function rollbackUpdate(backupId?: string): Promise<{ status: string; from_version: string; to_version: string; backup_id: string }> {
  const path = backupId ? `/api/v1/admin/update/rollback/${backupId}` : "/api/v1/admin/update/rollback";
  return request(path, { method: "POST" });
}

export function getUpdateHistory(): Promise<{ history: UpdateHistoryEntry[] }> {
  return request("/api/v1/admin/update/history");
}

export function enableScheduler(): Promise<{ status: string; message: string }> {
  return request("/api/v1/admin/update/scheduler/enable", { method: "POST" });
}

export function disableScheduler(): Promise<{ status: string; message: string }> {
  return request("/api/v1/admin/update/scheduler/disable", { method: "POST" });
}

export function systemHealth(): Promise<SystemHealthResult> {
  return request("/api/v1/admin/system/health");
}
