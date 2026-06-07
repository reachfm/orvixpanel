/**
 * Accounts API. Maps 1:1 to the Go handlers in internal/api/v1/accounts.go.
 *
 * The backend returns the full `models.Account` JSON for create / get /
 * suspend / unsuspend, and `{accounts: [...]}` for list.
 */

import { request } from "./client";

export interface Account {
  id: string;
  username: string;
  domain: string;
  tenant_id: string;
  plan: string;            // basic | pro | unlimited
  disk_quota_mb: number;
  bandwidth_gb: number;
  disk_used_mb?: number;   // populated by GET /:id (live)
  status: "active" | "suspended" | "deleted";
  created_at: string;
  updated_at: string;
}

export interface CreateAccountRequest {
  username: string;
  domain: string;
  plan?: string;
  disk_quota_mb?: number;
  bandwidth_gb?: number;
}

export function listAccounts(): Promise<{ accounts: Account[] }> {
  return request("/api/v1/accounts");
}

export function getAccount(id: string): Promise<Account> {
  return request(`/api/v1/accounts/${encodeURIComponent(id)}`);
}

export function createAccount(body: CreateAccountRequest): Promise<Account> {
  return request("/api/v1/accounts", { method: "POST", body });
}

export function suspendAccount(id: string): Promise<Account> {
  return request(`/api/v1/accounts/${encodeURIComponent(id)}/suspend`, { method: "POST" });
}

export function unsuspendAccount(id: string): Promise<Account> {
  return request(`/api/v1/accounts/${encodeURIComponent(id)}/unsuspend`, { method: "POST" });
}

export function deleteAccount(id: string): Promise<void> {
  return request(`/api/v1/accounts/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export interface AccountUsage {
  // Shape of /api/v1/accounts/:id/usage — see accounts.go.
  // The exact fields are kept loose; the UI only renders them.
  [key: string]: unknown;
}
export function accountUsage(id: string): Promise<AccountUsage> {
  return request(`/api/v1/accounts/${encodeURIComponent(id)}/usage`);
}
