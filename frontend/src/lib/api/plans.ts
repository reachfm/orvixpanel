/**
 * Hosting Plans API. Maps 1:1 to the Go handlers in internal/api/v1/hosting_plans.go.
 *
 * The backend returns the full Plan JSON for create / get / update, and
 * `{plans: [...]}` for list.
 */

import { request } from "./client";

export interface Plan {
  id: string;
  name: string;
  display_name: string;
  description: string;
  disk_quota_mb: number;
  bandwidth_gb: number;
  max_domains: number;
  max_users: number;
  max_ssl: number;
  features: string[];
  monthly_price: number;
  is_active: boolean;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePlanRequest {
  name: string;
  display_name?: string;
  description?: string;
  disk_quota_mb?: number;
  bandwidth_gb?: number;
  max_domains?: number;
  max_users?: number;
  max_ssl?: number;
  features?: string[];
  monthly_price?: number;
  is_active?: boolean;
  is_default?: boolean;
}

export interface UpdatePlanRequest extends CreatePlanRequest {}

export function listPlans(params?: { search?: string; status?: string }): Promise<{ plans: Plan[] }> {
  const searchParams = new URLSearchParams();
  if (params?.search) searchParams.set("search", params.search);
  if (params?.status) searchParams.set("status", params.status);
  const query = searchParams.toString();
  return request(`/api/v1/hosting/plans${query ? `?${query}` : ""}`);
}

export function getPlan(id: string): Promise<Plan> {
  return request(`/api/v1/hosting/plans/${encodeURIComponent(id)}`);
}

export function createPlan(body: CreatePlanRequest): Promise<Plan> {
  return request("/api/v1/hosting/plans", { method: "POST", body });
}

export function updatePlan(id: string, body: UpdatePlanRequest): Promise<Plan> {
  return request(`/api/v1/hosting/plans/${encodeURIComponent(id)}`, { method: "PUT", body });
}

export function deletePlan(id: string): Promise<void> {
  return request(`/api/v1/hosting/plans/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function activatePlan(id: string): Promise<Plan> {
  return request(`/api/v1/hosting/plans/${encodeURIComponent(id)}/activate`, { method: "POST" });
}

export function deactivatePlan(id: string): Promise<Plan> {
  return request(`/api/v1/hosting/plans/${encodeURIComponent(id)}/deactivate`, { method: "POST" });
}