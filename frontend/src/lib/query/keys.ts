/**
 * TanStack Query keys — a single source of truth for cache invalidation.
 *
 * Convention: each module exports a `xxxKeys` object whose values are
 * functions returning a tuple. `invalidateQueries(xxxKeys.all)` clears
 * everything for that module; `invalidateQueries(xxxKeys.detail(id))`
 * clears one record.
 *
 * Note: each function returns a plain `unknown[]` so TanStack Query's
 * type inference stays loose (we don't want to over-constrain the
 * key shape — invalidation works either way at runtime).
 */

export const authKeys = {
  me: (): readonly unknown[] => ["auth", "me"],
};

export const accountKeys = {
  all: (): readonly unknown[] => ["accounts"],
  list: (): readonly unknown[] => ["accounts", "list"],
  detail: (id: string): readonly unknown[] => ["accounts", "detail", id],
  usage: (id: string): readonly unknown[] => ["accounts", "usage", id],
};

export const domainKeys = {
  all: (): readonly unknown[] => ["domains"],
  byAccount: (accountId: string): readonly unknown[] => ["domains", "by-account", accountId],
};

export const deploymentKeys = {
  all: (): readonly unknown[] => ["deployments"],
  byAccount: (accountId: string): readonly unknown[] => ["deployments", "by-account", accountId],
};

export const systemKeys = {
  healthz: (): readonly unknown[] => ["system", "healthz"],
  readyz:  (): readonly unknown[] => ["system", "readyz"],
  info:    (): readonly unknown[] => ["system", "info"],
  license: (): readonly unknown[] => ["system", "license"],
  licenseRenewal: (): readonly unknown[] => ["system", "license-renewal"],
  health:  (): readonly unknown[] => ["system", "health"],
};

export const updateKeys = {
  all:     (): readonly unknown[] => ["update"],
  status:  (): readonly unknown[] => ["update", "status"],
  history: (): readonly unknown[] => ["update", "history"],
  health:  (): readonly unknown[] => ["update", "health"],
};

export const auditKeys = {
  all:    (): readonly unknown[] => ["audit"],
  list:   (limit: number): readonly unknown[] => ["audit", "list", limit],
  search: (params: object): readonly unknown[] => ["audit", "search", params],
};

// DNS
export const dnsZoneKeys = {
  all:     (): readonly unknown[] => ["dns", "zones"],
  list:    (): readonly unknown[] => ["dns", "zones", "list"],
  detail:  (id: string): readonly unknown[] => ["dns", "zones", "detail", id],
  records: (zoneId: string): readonly unknown[] => ["dns", "zones", "records", zoneId],
};

export const dnsTemplateKeys = {
  all:    (): readonly unknown[] => ["dns", "templates"],
  list:   (): readonly unknown[] => ["dns", "templates", "list"],
  detail: (id: string): readonly unknown[] => ["dns", "templates", "detail", id],
};

// Hosting Plans
export const planKeys = {
  all:    (): readonly unknown[] => ["plans"],
  list:   (): readonly unknown[] => ["plans", "list"],
  detail: (id: string): readonly unknown[] => ["plans", "detail", id],
};
