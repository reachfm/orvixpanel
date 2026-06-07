/**
 * Deployments API — backed by the v0.2.3 GET /api/v1/accounts/:id/deployments
 * route, which iterates the account's domains and returns the live
 * release directories on disk. No mock data; the route reads the
 * filesystem via internal/hosting.ListReleases + CurrentRelease.
 */

import { request } from "./client";

export interface Deployment {
  account_id: string;
  username: string;
  domain: string;
  release: string;          // release directory name (e.g. 2026-06-08-12-34-56)
  is_current: boolean;      // whether this is the symlink target
  size_bytes: number;       // sum of file sizes under the release dir
  modified_at: string;      // ISO timestamp of the release dir mtime
}

export function listDeployments(accountId: string): Promise<{ deployments: Deployment[] }> {
  return request(`/api/v1/accounts/${encodeURIComponent(accountId)}/deployments`);
}
