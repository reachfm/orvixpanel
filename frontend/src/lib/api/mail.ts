/**
 * Mail API client for OrvixPanel
 */

import { request } from "./client";

// Types
export type MailStatus = "active" | "suspended" | "deleted";
export type MailDirection = "inbound" | "outbound";

// Mail Domain
export interface MailDomain {
  id: string;
  tenant_id: string;
  account_id?: string;
  domain: string;
  dkim_selector: string;
  dkim_public?: string;
  spf_record: string;
  dmarc_policy: string;
  is_catch_all: boolean;
  max_mailboxes: number;
  status: MailStatus;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// Mailbox
export interface Mailbox {
  id: string;
  tenant_id: string;
  account_id?: string;
  mail_domain_id: string;
  local_part: string;
  email: string;
  quota_mb: number;
  quota_used_mb: number;
  enable_imap: boolean;
  enable_pop3: boolean;
  enable_smtp: boolean;
  status: MailStatus;
  last_login_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// Mail Alias
export interface MailAlias {
  id: string;
  tenant_id: string;
  account_id?: string;
  mail_domain_id: string;
  source_email: string;
  destinations: string; // JSON array
  is_catch_all: boolean;
  status: MailStatus;
  created_by: string;
  created_at: string;
}

// Mail Forwarder
export interface MailForwarder {
  id: string;
  tenant_id: string;
  account_id?: string;
  mail_domain_id: string;
  source_email: string;
  destinations: string; // JSON array
  keep_copy: boolean;
  status: MailStatus;
  created_by: string;
  created_at: string;
}

// Rate Limit
export interface MailRateLimit {
  id: string;
  tenant_id: string;
  account_id?: string;
  mailbox_id?: string;
  rate_type: "outbound" | "inbound" | "relay";
  max_messages: number;
  window_minutes: number;
  max_size_mb: number;
  status: MailStatus;
  created_at: string;
}

// Audit Log
export interface MailAuditLog {
  id: string;
  tenant_id: string;
  mailbox_id?: string;
  action: string;
  direction: MailDirection;
  from_email: string;
  to_email: string;
  subject?: string;
  message_id?: string;
  size_bytes: number;
  status: string;
  error_code?: string;
  remote_ip?: string;
  user_agent?: string;
  created_at: string;
}

// Quota Status
export interface QuotaStatus {
  mailbox_id: string;
  used_mb: number;
  limit_mb: number;
  used_percent: number;
  status: "ok" | "warning" | "exceeded";
  allow_send: boolean;
  allow_receive: boolean;
}

// Stats
export interface MailStats {
  total_domains: number;
  total_mailboxes: number;
  total_aliases: number;
  total_forwarders: number;
  storage_used_bytes: number;
  storage_available_bytes: number;
  active_today: number;
  generated_at: string;
}

// Quota Stats
export interface QuotaStats {
  summary: {
    healthy_count: number;
    warning_count: number;
    critical_count: number;
    total_quota_bytes: number;
  };
}

// DNS Records
export interface DNSRecords {
  spf: string;
  dmarc: string;
  dkim: string;
}

// Request types
export interface CreateDomainRequest {
  domain: string;
  is_catch_all?: boolean;
  max_mailboxes?: number;
}

export interface CreateMailboxRequest {
  email: string;
  password: string;
  domain_id: string;
  quota_mb?: number;
  enable_imap?: boolean;
  enable_pop3?: boolean;
  enable_smtp?: boolean;
}

export interface CreateAliasRequest {
  source_email: string;
  destinations: string[];
  domain_id: string;
}

export interface CreateForwarderRequest {
  source_email: string;
  destinations: string[];
  domain_id: string;
  keep_copy?: boolean;
}

export interface CreateRateLimitRequest {
  mailbox_id?: string;
  rate_type: "outbound" | "inbound" | "relay";
  max_messages: number;
  window_minutes: number;
  max_size_mb?: number;
}

export interface ListResponse<_T> {
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// API keys for React Query
export const mailKeys = {
  all: ["mail"] as const,
  domains: () => [...mailKeys.all, "domains"] as const,
  domain: (id: string) => [...mailKeys.domains(), id] as const,
  mailboxes: () => [...mailKeys.all, "mailboxes"] as const,
  mailbox: (id: string) => [...mailKeys.mailboxes(), id] as const,
  mailboxQuota: (id: string) => [...mailKeys.mailbox(id), "quota"] as const,
  aliases: () => [...mailKeys.all, "aliases"] as const,
  forwarders: () => [...mailKeys.all, "forwarders"] as const,
  audit: () => [...mailKeys.all, "audit"] as const,
  stats: () => [...mailKeys.all, "stats"] as const,
  ratelimits: () => [...mailKeys.all, "ratelimits"] as const,
  dnsRecords: (id: string) => [...mailKeys.domain(id), "dns"] as const,
};

// Domain API
export async function listDomains(params?: {
  page?: number;
  page_size?: number;
}): Promise<{ domains: MailDomain[] } & ListResponse<MailDomain>> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.page_size) searchParams.set("page_size", String(params.page_size));
  return request<{ domains: MailDomain[] } & ListResponse<MailDomain>>(
    `/api/v1/mail/domains?${searchParams.toString()}`
  );
}

export async function getDomain(id: string): Promise<MailDomain> {
  return request<MailDomain>(`/api/v1/mail/domains/${id}`);
}

export async function createDomain(data: CreateDomainRequest): Promise<MailDomain> {
  return request<MailDomain>("/api/v1/mail/domains", { method: "POST", body: data });
}

export async function updateDomain(
  id: string,
  data: Partial<MailDomain>
): Promise<MailDomain> {
  return request<MailDomain>(`/api/v1/mail/domains/${id}`, { method: "PUT", body: data });
}

export async function deleteDomain(id: string): Promise<void> {
  await request(`/api/v1/mail/domains/${id}`, { method: "DELETE" });
}

export async function generateDKIM(
  id: string,
  selector?: string
): Promise<{ dkim_selector: string; dkim_public: string }> {
  return request<{ dkim_selector: string; dkim_public: string }>(
    `/api/v1/mail/domains/${id}/dkim`,
    { method: "POST", body: { selector } }
  );
}

export async function getDNSRecords(id: string): Promise<DNSRecords> {
  return request<DNSRecords>(`/api/v1/mail/domains/${id}/records`);
}

// Mailbox API
export async function listMailboxes(params?: {
  page?: number;
  page_size?: number;
  domain_id?: string;
}): Promise<{ mailboxes: Mailbox[] } & ListResponse<Mailbox>> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.page_size) searchParams.set("page_size", String(params.page_size));
  if (params?.domain_id) searchParams.set("domain_id", params.domain_id);
  return request<{ mailboxes: Mailbox[] } & ListResponse<Mailbox>>(
    `/api/v1/mail/mailboxes?${searchParams.toString()}`
  );
}

export async function getMailbox(id: string): Promise<Mailbox> {
  return request<Mailbox>(`/api/v1/mail/mailboxes/${id}`);
}

export async function createMailbox(data: CreateMailboxRequest): Promise<Mailbox> {
  return request<Mailbox>("/api/v1/mail/mailboxes", { method: "POST", body: data });
}

export async function updateMailbox(
  id: string,
  data: Partial<Mailbox>
): Promise<Mailbox> {
  return request<Mailbox>(`/api/v1/mail/mailboxes/${id}`, { method: "PUT", body: data });
}

export async function deleteMailbox(id: string): Promise<void> {
  await request(`/api/v1/mail/mailboxes/${id}`, { method: "DELETE" });
}

export async function changeMailboxPassword(
  id: string,
  password: string
): Promise<void> {
  await request(`/api/v1/mail/mailboxes/${id}/password`, {
    method: "POST",
    body: { password },
  });
}

export async function suspendMailbox(id: string): Promise<void> {
  await request(`/api/v1/mail/mailboxes/${id}/suspend`, { method: "POST" });
}

export async function reactivateMailbox(id: string): Promise<void> {
  await request(`/api/v1/mail/mailboxes/${id}/reactivate`, { method: "POST" });
}

export async function getMailboxQuota(id: string): Promise<QuotaStatus> {
  return request<QuotaStatus>(`/api/v1/mail/mailboxes/${id}/quota`);
}

// Alias API
export async function listAliases(params?: {
  page?: number;
  page_size?: number;
  domain_id?: string;
}): Promise<{ aliases: MailAlias[] } & ListResponse<MailAlias>> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.page_size) searchParams.set("page_size", String(params.page_size));
  if (params?.domain_id) searchParams.set("domain_id", params.domain_id);
  return request<{ aliases: MailAlias[] } & ListResponse<MailAlias>>(
    `/api/v1/mail/aliases?${searchParams.toString()}`
  );
}

export async function createAlias(data: CreateAliasRequest): Promise<MailAlias> {
  return request<MailAlias>("/api/v1/mail/aliases", { method: "POST", body: data });
}

export async function deleteAlias(id: string): Promise<void> {
  await request(`/api/v1/mail/aliases/${id}`, { method: "DELETE" });
}

// Forwarder API
export async function listForwarders(params?: {
  page?: number;
  page_size?: number;
  domain_id?: string;
}): Promise<{ forwarders: MailForwarder[] } & ListResponse<MailForwarder>> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.page_size) searchParams.set("page_size", String(params.page_size));
  if (params?.domain_id) searchParams.set("domain_id", params.domain_id);
  return request<{ forwarders: MailForwarder[] } & ListResponse<MailForwarder>>(
    `/api/v1/mail/forwarders?${searchParams.toString()}`
  );
}

export async function createForwarder(data: CreateForwarderRequest): Promise<MailForwarder> {
  return request<MailForwarder>("/api/v1/mail/forwarders", { method: "POST", body: data });
}

export async function deleteForwarder(id: string): Promise<void> {
  await request(`/api/v1/mail/forwarders/${id}`, { method: "DELETE" });
}

// Stats API
export async function getMailStats(): Promise<MailStats> {
  return request<MailStats>("/api/v1/mail/stats");
}

export async function getQuotaStats(): Promise<QuotaStats> {
  return request<QuotaStats>("/api/v1/mail/quota/stats");
}

// Audit API
export async function listAuditLogs(params?: {
  page?: number;
  page_size?: number;
  mailbox_id?: string;
  action?: string;
  direction?: string;
}): Promise<{ logs: MailAuditLog[]; total: number }> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.page_size) searchParams.set("page_size", String(params.page_size));
  if (params?.action) searchParams.set("action", params.action);
  if (params?.direction) searchParams.set("direction", params.direction);
  return request<{ logs: MailAuditLog[]; total: number }>(
    `/api/v1/mail/audit?${searchParams.toString()}`
  );
}

// Backward compatibility alias
export const getAuditLogs = listAuditLogs;

export async function getAuditStats(): Promise<{
  total_events: number;
  actions: Array<{ action: string; count: number }>;
}> {
  return request<{
    total_events: number;
    actions: Array<{ action: string; count: number }>;
  }>("/api/v1/mail/audit/stats");
}

// Rate Limit API
export async function listRateLimits(): Promise<{ ratelimits: MailRateLimit[] }> {
  return request<{ ratelimits: MailRateLimit[] }>("/api/v1/mail/ratelimits");
}

export async function createRateLimit(data: CreateRateLimitRequest): Promise<MailRateLimit> {
  return request<MailRateLimit>("/api/v1/mail/ratelimits", { method: "POST", body: data });
}

export async function deleteRateLimit(id: string): Promise<void> {
  await request(`/api/v1/mail/ratelimits/${id}`, { method: "DELETE" });
}

// Test endpoints (VPS required - returns stub data in preview)
export async function testSMTP(): Promise<{ status: string; message: string }> {
  return request<{ status: string; message: string }>("/api/v1/mail/test/smtp", {
    method: "POST",
  });
}

export async function testIMAP(): Promise<{ status: string; message: string }> {
  return request<{ status: string; message: string }>("/api/v1/mail/test/imap", {
    method: "POST",
  });
}

export async function testDelivery(): Promise<{ status: string; message: string }> {
  return request<{ status: string; message: string }>("/api/v1/mail/test/delivery", {
    method: "POST",
  });
}