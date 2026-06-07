/**
 * Domains API. Maps to internal/api/v1/domains.go.
 */

import { request } from "./client";

export interface Domain {
  id: string;
  account_id: string;
  tenant_id: string;
  username: string;
  name: string;
  document_root: string;
  status: "active" | "suspended" | "deleted";
  created_at: string;
  updated_at: string;
}

export interface CreateDomainRequest {
  name: string;
  document_root?: string;
  port?: number;
}

export function listDomains(accountId: string): Promise<{ domains: Domain[] }> {
  return request(`/api/v1/accounts/${encodeURIComponent(accountId)}/domains`);
}

export function createDomain(accountId: string, body: CreateDomainRequest): Promise<Domain> {
  return request(`/api/v1/accounts/${encodeURIComponent(accountId)}/domains`, {
    method: "POST",
    body,
  });
}

export function deleteDomain(accountId: string, domain: string): Promise<void> {
  return request(
    `/api/v1/accounts/${encodeURIComponent(accountId)}/domains/${encodeURIComponent(domain)}`,
    { method: "DELETE" },
  );
}
