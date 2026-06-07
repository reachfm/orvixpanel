/**
 * Auth API. Maps 1:1 to the Go handlers in internal/api/v1/auth.go.
 *
 * Login returns the full envelope (access + refresh + expires_at + user).
 * Refresh returns just the new pair.
 */

import { request } from "./client";

export interface User {
  id: string;
  email: string;
  role: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user: User;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

export function login(email: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>("/auth/login", {
    method: "POST",
    body: { email, password },
    noAuth: true,
  });
}

export function refresh(refreshToken: string): Promise<RefreshResponse> {
  return request<RefreshResponse>("/auth/refresh", {
    method: "POST",
    body: { refresh_token: refreshToken },
    noAuth: true,
  });
}

export function logout(): Promise<{ logged_out: boolean }> {
  return request<{ logged_out: boolean }>("/auth/logout", { method: "POST" });
}

export interface MeResponse {
  user_id: string;
  email: string;
  role: string;
  tenant_id: string;
  account_id: string | null;
  session_id: string;
}
export function me(): Promise<MeResponse> {
  return request<MeResponse>("/api/v1/me");
}
