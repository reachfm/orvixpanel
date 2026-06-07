# OrvixPanel v0.3.0 — Enterprise Edition (PLAN)

> **Target:** v0.3.0 = spec Phase "Enterprise tier" — the actual feature set
> that justifies the $999/mo Enterprise license. v0.1.0 had the engine
> with the tier *names*; v0.3.0 wires the features.
>
> **Status:** in progress
> **Codename:** Aegis
> **Effort estimate:** multi-day focused build; this is real work, not docs
>
> Everything here gets built, tested, and verified on the Windows dev box
> before any commit. No `// TODO: implement` stubs. No fake completions.
> No faked audit entries. If a feature can't be verified, it doesn't ship.

## Scope — what v0.3.0 Enterprise Edition actually ships

### 1. Real license expiry + read-only mode (replaces the v0.1.0 stub)

`internal/license/license.go` currently bakes `ExpiresAt = 1735689600`
(2025-01-01) into every parsed key. v0.3.0:

- License payload gets real `IssuedAt`, `ExpiresAt`, `GracePeriodDays`
- ECDSA-signed payload (key from `ORVIX_LICENSE_PUBKEY` env, dev-mode
  short-circuit behind `ORVIX_ALLOW_DEV=1` preserved)
- New middleware `ReadOnlyEnforcer`: if license expired + grace exhausted,
  all `POST/PUT/PATCH/DELETE` return `423 locked` (RFC 4918) with
  `license_expired` code; `GET` still works
- New endpoint `GET /api/v1/admin/license/renewal-info` returns days
  remaining, grace days left, current mode (`active` | `grace` | `locked`)
- Audit every mode transition

### 2. Encrypted license persistence

- License file `/etc/orvixpanel/license.key` (or env override) stored
  in DB as `encrypted_license` (BLOB column) using AES-256-GCM
- Master key from `ORVIX_MASTER_KEY` env (32 bytes hex), or derived
  from `server.secret_key` in dev
- Key versioning prefix: `v1:` so we can rotate the cipher
- Audit every read/write of the license row

### 3. Encrypted secrets vault

`internal/vault/vault.go` — per-tenant secret store. Spec §6 says
"Custom vault with memory encryption" — we deliver it.

- Model: `Secret { id, tenant_id, name, ciphertext, nonce, created_by,
  created_at, rotated_at, version }`
- AES-256-GCM, master key from same `ORVIX_MASTER_KEY`
- API: `GET/POST /api/v1/vault/secrets`, `GET /api/v1/vault/secrets/:id`,
  `POST /api/v1/vault/secrets/:id/rotate`, `DELETE /api/v1/vault/secrets/:id`
- Every read/write/rotate/delete audited
- Vault write requires `vault.write` permission; read requires `vault.read`

### 4. Custom RBAC roles

`internal/rbac/custom.go` — admin-defined roles, not just the 12
hardcoded ones.

- Model: `CustomRole { id, tenant_id, name, permissions jsonb,
  is_builtin, created_at, updated_at }`
- Built-in 12 stay; new roles compose permissions like
  `[{resource:"domain", actions:["read","create"]}, ...]`
- `RequirePermission` middleware extended to check custom roles when
  the user has one assigned
- API: `GET/POST /api/v1/admin/roles`, `PUT/DELETE /api/v1/admin/roles/:id`,
  `POST /api/v1/admin/users/:id/role` (assign custom role)
- Audit every role CRUD + assignment

### 5. API key auth

`internal/auth/apikey.go` — long-lived keys for automation, separate
from JWT.

- Model: `APIKey { id, tenant_id, name, key_hash, prefix, role, scopes,
  expires_at, last_used_at, last_used_ip, created_by, created_at,
  revoked_at }`
- Format: `orx_live_<prefix>_<secret>` — prefix is the public 8-char
  identifier; full key never stored, only SHA-256 hash
- Auth: `Authorization: Bearer orx_live_...` or `X-Orvix-Api-Key: orx_live_...`
- Scope check: each key has `scopes []string` (subset of role's perms)
- Audit every key use (subject to rate limit so we don't audit-spam)
- API: `GET/POST /api/v1/admin/api-keys`, `DELETE /api/v1/admin/api-keys/:id`

### 6. Audit search/filter

`internal/audit/search.go` — replaces the v0.1.0 list-all endpoint.

- `POST /api/v1/admin/audit-log/search` with body:
  ```json
  {
    "tenant_id": "...",
    "user_id": "...",
    "action": "domain.create",
    "result": "success",
    "since": "2026-06-01T00:00:00Z",
    "until": "2026-06-07T23:59:59Z",
    "limit": 100,
    "offset": 0
  }
  ```
- Returns: `{rows: [...], total: 1234, filters: {...}, next_offset: 100}`
- Indexes on `(tenant_id, timestamp)`, `(user_id, timestamp)`,
  `(action, timestamp)` for the search path
- Tenant isolation: non-root users can only search their own tenant

### 7. Audit CEF export over syslog

`internal/audit/export.go` — push to SIEM.

- `POST /api/v1/admin/audit-log/export` with body:
  ```json
  {
    "transport": "syslog_udp",   // or "syslog_tcp", "file"
    "host": "siem.example.com",
    "port": 514,
    "since": "...",
    "until": "...",
    "device_vendor": "Orvix",
    "device_product": "Panel"
  }
  ```
- CEF format: `CEF:0|Orvix|Panel|0.3.0|domain.create|User created domain|5|...`
- Returns: `{exported: 42, transport: "syslog_udp", started_at, finished_at, errors: 0}`
- Audit the export itself

### 8. Tenant-level quotas

`internal/quota/tenant.go` — bound the resources a tenant can consume.

- Model: `TenantQuota { tenant_id, max_accounts, max_users, max_domains,
  max_storage_mb, max_bandwidth_gb, max_api_keys, max_custom_roles,
  updated_at }`
- API: `GET/PUT /api/v1/admin/tenants/:id/quotas`
- Default quota per tier (enterprise: 1000/5000/5000/1TB/10TB/100/50)
- Enforce on account create, user create, etc. — return `403 quota_exceeded`
- Audit every quota check that blocks

## File plan (v0.3.0 adds)

```
internal/
├── license/
│   ├── license.go                  # REWRITE: real expiry + grace
│   ├── license_test.go             # NEW
│   └── store.go                    # NEW: encrypted license persistence
├── vault/
│   ├── vault.go                    # NEW: AES-256-GCM secret store
│   ├── vault_test.go               # NEW
│   └── handlers.go                 # NEW: Fiber handlers
├── rbac/
│   ├── custom.go                   # NEW: custom role service
│   ├── custom_test.go              # NEW
│   └── handlers.go                 # NEW
├── auth/
│   ├── apikey.go                   # NEW: API key model + service
│   ├── apikey_test.go              # NEW
│   ├── apikey_handlers.go          # NEW
│   └── jwt.go                      # unchanged
├── audit/
│   ├── audit.go                    # unchanged (chain still works)
│   ├── search.go                   # NEW: filter + paginate
│   ├── search_test.go              # NEW
│   ├── export.go                   # NEW: CEF + syslog
│   ├── export_test.go              # NEW
│   └── handlers.go                 # NEW: search + export endpoints
├── quota/
│   ├── quota.go                    # NEW: per-tenant quota service
│   ├── quota_test.go               # NEW
│   └── handlers.go                 # NEW
├── api/
│   ├── middleware/
│   │   ├── readonly.go             # NEW: 423 on expired license
│   │   ├── apikey.go               # NEW: X-Orvix-Api-Key auth
│   │   └── rbac.go                 # UPDATE: check custom roles
│   ├── router.go                   # UPDATE: new routes
│   └── server.go                   # UPDATE: wire new services
├── db/
│   ├── db.go                       # UPDATE: migrate new tables
│   └── models/
│       ├── models.go               # UPDATE: 5 new models
│       └── apikey.go               # NEW
└── config/
    └── config.go                   # UPDATE: master key + grace period config

configs/orvixpanel.example.toml     # UPDATE: new [license] [vault] [quota] sections
README.md                            # UPDATE: v0.3.0 section
RELEASE_NOTES.md                     # UPDATE: v0.3.0 release
BUILD_REPORT.md                      # UPDATE: full verification log
```

## API additions (v0.3.0)

All routes gated by license tier **enterprise** (or **whitelabel**).
Returns 423 `license_expired` for write methods after grace.
Returns 403 `feature_not_licensed` outside the gate.

```
# License + read-only
GET    /api/v1/admin/license/renewal-info

# Encrypted license persistence
PUT    /api/v1/admin/license                            # upload signed license

# Secrets vault
GET    /api/v1/vault/secrets
POST   /api/v1/vault/secrets
GET    /api/v1/vault/secrets/:id
POST   /api/v1/vault/secrets/:id/rotate
DELETE /api/v1/vault/secrets/:id

# Custom RBAC roles
GET    /api/v1/admin/roles
POST   /api/v1/admin/roles
PUT    /api/v1/admin/roles/:id
DELETE /api/v1/admin/roles/:id
POST   /api/v1/admin/users/:id/role                     # assign role (builtin or custom)

# API keys
GET    /api/v1/admin/api-keys
POST   /api/v1/admin/api-keys
DELETE /api/v1/admin/api-keys/:id

# Audit search + export
POST   /api/v1/admin/audit-log/search
POST   /api/v1/admin/audit-log/export

# Tenant quotas
GET    /api/v1/admin/tenants/:id/quotas
PUT    /api/v1/admin/tenants/:id/quotas
```

## License feature gating

| Feature id           | SMB | ISP | Enterprise | White-Label |
|----------------------|-----|-----|------------|--------------|
| `vault.*`            | ❌  | ❌  | ✅         | ✅           |
| `rbac.custom`        | ❌  | ❌  | ✅         | ✅           |
| `apikey.*`           | ❌  | ❌  | ✅         | ✅           |
| `audit.search`       | ❌  | ✅  | ✅         | ✅           |
| `audit.export`       | ❌  | ❌  | ✅         | ✅           |
| `quota.tenant`       | ❌  | ✅  | ✅         | ✅           |
| `readonly.enforce`   | ✅  | ✅  | ✅         | ✅           |

## Verification gate (everything must pass before v0.3.0 tag)

```bash
# 1. Clean build
go build -ldflags="-s -w -X main.version=0.3.0" -o bin/orvixpanel ./cmd/orvixpanel
# → exit 0

# 2. All tests pass with race detector
go test -race -coverprofile=coverage.out ./...
# → all packages ok
# → coverage on internal/license, internal/vault, internal/rbac,
#   internal/auth, internal/audit, internal/quota >= 80%

# 3. Live smoke test
# - Boot the binary with ORVIX_ALLOW_DEV=1
# - Verify license renewal-info endpoint shows mode=active
# - Set ORVIX_FAKE_EXPIRED=1, verify POST /me returns 423
# - Reset, create a vault secret, read it back, verify ciphertext
# - Create a custom role, assign to user, verify permissions
# - Create an API key, use it to call /me, verify it works
# - Search audit log with filters, verify count
# - Export audit log via CEF, verify syslog receives
# - Set tenant quota to max_accounts=0, verify account create returns 403 quota_exceeded
# - Verify audit chain still clean at the end

# 4. Cross-platform build
GOOS=linux go build -o bin/orvixpanel.linux ./cmd/orvixpanel
# → exit 0 (verifies our code is portable; runtime verification is on Linux VPS)
```

## Risks

1. **License key ECDSA verification** — the v0.1.0 code says it's "bypassed
   when ORVIX_ALLOW_DEV=1". I don't have a real public key from the
   license server. v0.3.0 will: (a) keep the dev bypass, (b) wire the
   ECDSA path so an operator can drop in a public key, (c) document this
   clearly in RELEASE_NOTES.
2. **AES-256-GCM nonce reuse** — every encryption generates a fresh
   12-byte nonce from `crypto/rand`. Never reused by design. Test in
   `internal/vault/vault_test.go` and `internal/license/store_test.go`.
3. **Audit export volume** — exporting 1M rows over syslog could blow
   up. v0.3.0 caps at 10k rows per export + requires `since`/`until`.
   Documented in handler docstring.
4. **Custom roles can over-permission** — if an admin creates a role
   with `["*","*"]`, it has root power. That's intentional (root admin
   is responsible) but we log a warning when such a role is created.
5. **API key leakage in logs** — full key is never logged; we log the
   8-char prefix only.

## Out of scope for v0.3.0

- v0.2.0 Core Hosting Engine (provisioning, nginx, php-fpm) — separate work
- DNS, Mail, SSL — v0.4.0
- WAF, eBPF firewall — v0.5.0
- Database, File Manager, Backups — v0.6.0
- Guardian AI — v0.7.0
- White-label (custom domain, logo, color) — v0.8.0
- SAML, OIDC SSO — v0.4.0 (basic auth via API key covers v0.3.0)
- HA / clustering — post-1.0

## Done means

- [ ] All file paths above exist + compile
- [ ] `go test ./...` passes with race detector
- [ ] Coverage on each new package >= 80%
- [ ] Live smoke test passes all 8 steps above
- [ ] `ENTERPRISE_PLAN.md` v0.4.0 written (next phase)
- [ ] git tag `v0.3.0-enterprise-preview` cut and pushed
- [ ] GitHub release with verification log
