/**
 * DNS API. Maps 1:1 to the Go handlers in internal/api/v1/dns.go.
 *
 * The backend returns the full `DNSZone`, `DNSRecord`, `DNSZoneTemplate` JSON.
 */

import { request } from "./client";

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

export type DNSZoneStatus = "active" | "suspended" | "pending";
export type DNSZoneType = "native" | "master" | "slave";
export type DNSRecordType = "A" | "AAAA" | "CNAME" | "MX" | "TXT" | "NS" | "SRV" | "CAA";

export interface DNSZone {
  id: string;
  account_id: string;
  tenant_id: string;
  domain: string;
  type: DNSZoneType;
  masters?: string;
  soa_refresh: number;
  soa_retry: number;
  soa_expire: number;
  soa_minimum: number;
  status: DNSZoneStatus;
  created_at: string;
  updated_at: string;
}

export interface DNSRecord {
  id: string;
  zone_id: string;
  name: string;
  type: DNSRecordType;
  content: string;
  ttl: number;
  priority: number;
  disabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface DNSZoneTemplate {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  records: string; // JSON array of record definitions
  created_at: string;
  updated_at: string;
}

// Template record definition (for creating templates)
export interface TemplateRecordDefinition {
  name: string;
  type: DNSRecordType;
  content: string;
  ttl?: number;
  priority?: number;
}

// Validation result
export interface ValidationResult {
  valid: boolean;
  errors?: string[];
}

// -----------------------------------------------------------------------------
// Zone API
// -----------------------------------------------------------------------------

export interface ListZonesResponse {
  zones: DNSZone[];
  count: number;
}

export function listZones(): Promise<ListZonesResponse> {
  return request("/api/v1/dns/zones");
}

export function getZone(id: string): Promise<DNSZone> {
  return request(`/api/v1/dns/zones/${encodeURIComponent(id)}`);
}

export interface CreateZoneRequest {
  domain: string;
  account_id?: string;
  type?: DNSZoneType;
  masters?: string[];
}

export function createZone(body: CreateZoneRequest): Promise<DNSZone> {
  return request("/api/v1/dns/zones", { method: "POST", body });
}

export interface UpdateZoneRequest {
  domain?: string;
  type?: DNSZoneType;
  status?: DNSZoneStatus;
  soa_refresh?: number;
  soa_retry?: number;
  soa_expire?: number;
  soa_minimum?: number;
}

export function updateZone(id: string, body: UpdateZoneRequest): Promise<DNSZone> {
  return request(`/api/v1/dns/zones/${encodeURIComponent(id)}`, { method: "PUT", body });
}

export function deleteZone(id: string): Promise<void> {
  return request(`/api/v1/dns/zones/${encodeURIComponent(id)}`, { method: "DELETE" });
}

// -----------------------------------------------------------------------------
// Record API
// -----------------------------------------------------------------------------

export interface ListRecordsResponse {
  records: DNSRecord[];
  count: number;
}

export function listRecords(zoneId: string): Promise<ListRecordsResponse> {
  return request(`/api/v1/dns/zones/${encodeURIComponent(zoneId)}/records`);
}

export interface CreateRecordRequest {
  name: string;
  type: DNSRecordType;
  content: string;
  ttl?: number;
  priority?: number;
  disabled?: boolean;
}

export function createRecord(zoneId: string, body: CreateRecordRequest): Promise<DNSRecord> {
  return request(`/api/v1/dns/zones/${encodeURIComponent(zoneId)}/records`, {
    method: "POST",
    body,
  });
}

export interface UpdateRecordRequest {
  name?: string;
  type?: DNSRecordType;
  content?: string;
  ttl?: number;
  priority?: number;
  disabled?: boolean;
}

export function updateRecord(
  zoneId: string,
  recordId: string,
  body: UpdateRecordRequest
): Promise<DNSRecord> {
  return request(
    `/api/v1/dns/zones/${encodeURIComponent(zoneId)}/records/${encodeURIComponent(recordId)}`,
    { method: "PUT", body }
  );
}

export function deleteRecord(zoneId: string, recordId: string): Promise<void> {
  return request(
    `/api/v1/dns/zones/${encodeURIComponent(zoneId)}/records/${encodeURIComponent(recordId)}`,
    { method: "DELETE" }
  );
}

// -----------------------------------------------------------------------------
// Template API
// -----------------------------------------------------------------------------

export interface ListTemplatesResponse {
  templates: DNSZoneTemplate[];
  count: number;
}

export function listTemplates(): Promise<ListTemplatesResponse> {
  return request("/api/v1/dns/templates");
}

export interface CreateTemplateRequest {
  name: string;
  description?: string;
  records: TemplateRecordDefinition[];
}

export function createTemplate(body: CreateTemplateRequest): Promise<DNSZoneTemplate> {
  return request("/api/v1/dns/templates", { method: "POST", body });
}

export function deleteTemplate(id: string): Promise<void> {
  return request(`/api/v1/dns/templates/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export interface ApplyTemplateRequest {
  zone_id: string;
}

export function applyTemplate(
  templateId: string,
  body: ApplyTemplateRequest
): Promise<{ records_created: number }> {
  return request(`/api/v1/dns/templates/${encodeURIComponent(templateId)}/apply`, {
    method: "POST",
    body,
  });
}

// -----------------------------------------------------------------------------
// Validation API
// -----------------------------------------------------------------------------

export interface ValidateRecordRequest {
  name?: string;
  type?: DNSRecordType;
  content?: string;
  ttl?: number;
}

export function validateRecord(body: ValidateRecordRequest): Promise<ValidationResult> {
  return request("/api/v1/dns/validate", { method: "POST", body });
}

// -----------------------------------------------------------------------------
// Lookup API
// -----------------------------------------------------------------------------

export interface LookupResponse {
  domain: string;
  found: boolean;
  records?: DNSRecord[];
}

export function lookupDomain(domain: string): Promise<LookupResponse> {
  return request(`/api/v1/dns/lookup/${encodeURIComponent(domain)}`);
}