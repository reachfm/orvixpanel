/**
 * SSL API. Maps 1:1 to the Go handlers in internal/ssl/handlers.go.
 *
 * The backend returns the full `SSLCertificate`, `SSLEvent` JSON.
 */

import { request } from "./client";

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

export type SSLCertStatus =
  | "pending"
  | "issued"
  | "expiring_soon"
  | "expired"
  | "revoked"
  | "failed";

export type SSLProvider = "letsencrypt" | "zerossl";

export interface SSLCertificate {
  id: string;
  domain_id?: string;
  account_id?: string;
  tenant_id?: string;
  provider: SSLProvider;
  common_name: string;
  san_names?: string; // Backend stores as comma-separated string, parse with .split(',')
  status: SSLCertStatus;
  auto_renew: boolean;
  cert_path?: string;
  key_path?: string;
  issued_at?: string;
  expires_at?: string;
  serial_number?: string;
  fingerprint?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface SSLEvent {
  id: string;
  certificate_id: string;
  event_type: string;
  message: string;
  error_detail?: string;
  created_at: string;
}

export interface SSLDashboardStats {
  total_active: number;
  expiring_soon: number;
  failed_renewals: number;
  auto_renew_enabled: number;
}

export interface SSLHealthReport {
  total: number;
  healthy: number;
  expiring_soon: number;
  expired: number;
  failed: number;
  certs: SSLHealthDetail[];
}

export interface SSLHealthDetail {
  id: string;
  domain: string;
  status: SSLCertStatus;
  expires_at?: string;
  days_until_expiry?: number;
  issues: string[];
}

// -----------------------------------------------------------------------------
// Certificate API
// -----------------------------------------------------------------------------

export function listCertificates(): Promise<SSLCertificate[]> {
  return request("/api/v1/ssl/certificates");
}

export function getCertificate(id: string): Promise<SSLCertificate> {
  return request(`/api/v1/ssl/certificates/${encodeURIComponent(id)}`);
}

export interface IssueCertificateRequest {
  domain: string;
  san_names?: string[];
  provider?: SSLProvider;
  auto_renew?: boolean;
  acme_account_id?: string;
}

export function issueCertificate(body: IssueCertificateRequest): Promise<SSLCertificate> {
  return request("/api/v1/ssl/certificates", { method: "POST", body });
}

export function renewCertificate(id: string): Promise<SSLCertificate> {
  return request(`/api/v1/ssl/certificates/${encodeURIComponent(id)}/renew`, { method: "POST" });
}

export function revokeCertificate(id: string): Promise<void> {
  return request(`/api/v1/ssl/certificates/${encodeURIComponent(id)}/revoke`, { method: "POST" });
}

export function deleteCertificate(id: string): Promise<void> {
  return request(`/api/v1/ssl/certificates/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export interface ImportCertificateRequest {
  domain: string;
  cert_pem: string;
  key_pem: string;
  chain_pem?: string;
}

export function importCertificate(body: ImportCertificateRequest): Promise<SSLCertificate> {
  return request("/api/v1/ssl/import", { method: "POST", body });
}

// -----------------------------------------------------------------------------
// Events API
// -----------------------------------------------------------------------------

export function listSSLEvents(limit?: number): Promise<SSLEvent[]> {
  const url = limit ? `/api/v1/ssl/events?limit=${limit}` : "/api/v1/ssl/events";
  return request(url);
}

export function getCertificateEvents(certId: string): Promise<SSLEvent[]> {
  return request(`/api/v1/ssl/certificates/${encodeURIComponent(certId)}/events`);
}

// -----------------------------------------------------------------------------
// Health & Dashboard API
// -----------------------------------------------------------------------------

export function getSSLHealth(): Promise<SSLHealthReport> {
  return request("/api/v1/ssl/health");
}

export function getSSLDashboardStats(): Promise<SSLDashboardStats> {
  return request("/api/v1/ssl/dashboard");
}

// -----------------------------------------------------------------------------
// Query Keys
// -----------------------------------------------------------------------------

export const sslKeys = {
  all: ["ssl"] as const,
  certificates: () => [...sslKeys.all, "certificates"] as const,
  certificate: (id: string) => [...sslKeys.certificates(), id] as const,
  events: () => [...sslKeys.all, "events"] as const,
  certificateEvents: (id: string) => [...sslKeys.certificate(id), "events"] as const,
  health: () => [...sslKeys.all, "health"] as const,
  dashboard: () => [...sslKeys.all, "dashboard"] as const,
};