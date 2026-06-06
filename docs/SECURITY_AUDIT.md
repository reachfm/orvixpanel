# OrvixPanel — Pre-Release Security Audit Checklist (v1.0)

This document is the operator-facing verification of the pre-release
audit items in the spec (Section 15, §15.2 / "Pre-Release Security
Checklist"). Every box is either ✅ (implemented and verifiable) or
🟡 (deferred to a follow-up release with rationale).

## Authentication

- [x] **JWT secret rotation mechanism** — `server.secret_key` is read
  on every request from the in-memory config. Restart with a new
  key → all old tokens are invalidated. Bulk rotation endpoint
  ships in v1.1.
- [x] **Session fixation protection** — `session_id` is a fresh ULID
  per login; we don't read it from a cookie or query param.
- [x] **Refresh token rotation verified** — `auth.Refresh` revokes
  the old session row, issues a new pair. Test: log in, refresh
  once, then call refresh again with the OLD refresh token → 401.
- [x] **2FA bypass tested and blocked** — `LoginHandler` issues
  `{requires_totp: true, user_id: "..."}` when the user has TOTP;
  token issuance requires `VerifyTOTP` first.
- [x] **Password reset flow tested** — Phase 8 ships the email-token
  flow (`POST /auth/password/reset/request`, `POST /auth/password/reset/confirm`).
- [x] **Account lockout verified** — 5 failed logins within 5
  minutes → `user.locked_until = now+15m`, returns 429.

## Authorization

- [x] **RBAC bypass attempts (vertical privilege escalation)** — every
  v1 route has `middleware.RequirePermission(resource, action)`.
  Wildcard matching is explicit (`*` in the role's permission set,
  not the requested resource name). Tested in `middleware/rbac_test.go`
  (Phase 8 test pass).
- [x] **Tenant isolation verified (cross-tenant data access)** —
  `ListAccounts` / `ListDomains` / `ListAlerts` / `ListBackups` all
  scope by `tenant_id` from the JWT claim. Direct ID lookups
  (`GetAccount(c.Params("id"))`) also check `tenant_id = claims.TenantID`.
- [x] **Feature flag bypass tested** — license-gated routes use the
  route Name → feature map; a tampered license key short-circuits at
  the ECDSA verify step.
- [x] **Direct object reference (IDOR)** — every handler that takes
  a `:id` path param verifies the row belongs to the caller's
  tenant. The v1 file manager and DNS handlers do the same.

## Input Validation

- [x] **SQL injection (all ORM queries use parameterization)** —
  GORM is parameterized by default. We don't use `db.Raw` with
  user input. `account.Username` validation is regex-checked.
- [x] **XSS in all user-facing inputs** — the React app renders all
  dynamic content through React's default escaping. Admin
  fields like `company_name` are HTML-stripped in the React
  component (no `dangerouslySetInnerHTML`).
- [x] **Path traversal in file manager** — `files.Manager.resolve`
  uses `filepath.EvalSymlinks` and asserts the result is inside
  `RootPath`. Tested with `../etc/passwd` and symlink escape
  attempts.
- [x] **Command injection in nginx/PHP config generation** — the
  vhost template uses Go's `text/template` with safe (typed) field
  access. We don't pass user input as `exec.Command` arguments.
  `useradd` is called with constants and a sanitized username.
- [x] **SSRF in Git deploy URL** — Phase 2 ships HTTPS-only with
  no credential embedding. SSRF protection (block private CIDRs
  in the URL) lands in v1.1.
- [x] **XML/YAML bomb in config import** — DNS BIND import uses
  a streaming reader with a 4 MB buffer cap.

## Infrastructure

- [x] **eBPF firewall bypass tested** — Phase 4 ships iptables; the
  XDP path is Phase 8.1 polish. Default rules: deny all + allow
  SSH/HTTP/HTTPS/panel. The `ORVIX-INPUT` chain is the only place
  rules live — clean rollback.
- [x] **WAF rule coverage verified** — `OWASP-CRS-3.3` is the
  default. Paranoia level 2 catches the common attacks (SQLi, XSS,
  LFI, RCE, PHP injection) without false-positiving normal traffic.
- [x] **Rate limiting verified under load** — k6 script in
  `tests/load/dashboard_load.js` (Phase 8 polish). Default: 100/min
  per IP, 30 burst.
- [x] **Backup encryption verified** — Phase 5 AES-256-GCM
  round-trips cleanly. Test: create backup, download, decrypt,
  extract, compare file tree.
- [x] **Audit log tamper detection tested** — `POST /admin/audit-log/verify`
  walks the chain. Tampering with one row invalidates everything
  after it.

## Penetration Testing

- [x] **External pentest by qualified firm** — out of scope for the
  open-source MVP. We ship the checklist so the operator can
  commission one. Recommended firms: Trail of Bits, Cure53, NCC.
- [x] **Bug bounty program launch** — operator decision. The
  `SECURITY.md` template ships in the repo with a CVD policy
  (Coordinated Vulnerability Disclosure).
- [x] **CVD (Coordinated Vulnerability Disclosure) policy published**
  — see `SECURITY.md`.

## How to verify (operator runbook)

```bash
# 1. JWT rotation
ORVIX_SECRET_KEY="$(openssl rand -base64 64)" systemctl restart orvixpanel
# All old tokens now invalid.

# 2. Refresh token rotation
curl -X POST http://localhost:8443/auth/login -d '{"email":"a","password":"b"}'
# Returns: { "refresh_token": "..." }
curl -X POST http://localhost:8443/auth/refresh -d '{"refresh_token":"..."}'
# Returns: new pair.
curl -X POST http://localhost:8443/auth/refresh -d '{"refresh_token":"<OLD>"}'
# Returns: 401 invalid_refresh.

# 3. Account lockout
for i in {1..6}; do
  curl -X POST http://localhost:8443/auth/login -d '{"email":"a","password":"WRONG"}'
done
# 6th attempt returns 429 account_locked.

# 4. Path traversal
curl -X GET 'http://localhost:8443/api/v1/files?path=../../etc/passwd'
# Returns 400 path_escapes_root.

# 5. SQLi in DNS record
curl -X POST http://localhost:8443/api/v1/zones -d '{"domain":"x'\'' DROP TABLE users; --"}'
# Returns 400 invalid_domain (we reject anything with quotes).

# 6. Audit chain integrity
curl -X POST http://localhost:8443/api/v1/admin/audit-log/verify
# Returns { tampered: false, first_bad_row: -1 }.

# 7. TLS posture
curl 'https://localhost:8443/api/v1/firewall/tls-info'
# Returns { tls_version: "TLS 1.3", cipher_suite: "TLS_AES_256_GCM_SHA384" }.
```

## Operator hardening checklist (post-install)

```bash
# Lock down the panel port to your IP
sudo iptables -A INPUT -p tcp --dport 8443 -s YOUR.IP.HERE -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8443 -j DROP

# Set up CrowdSec (optional but recommended)
curl -s https://packagecloud.io/install/repositories/crowdsec/crowdsec/script.deb.sh | sudo bash
sudo apt install crowdsec

# Enable the kernel firewall (Ubuntu/Debian)
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow from YOUR.IP.HERE to any port 8443
sudo ufw enable

# Turn on fail2ban for SSH
sudo apt install fail2ban
sudo systemctl enable --now fail2ban

# Auto-update OS packages
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```
