# OrvixPanel v0.1.0 — Foundation Preview

**Release date:** 2026-06-07
**Codename:** Nova
**Tag:** v0.1.0 (NOT v0.1.0 — see the honest note below)
**Scope:** Foundation only. See `BUILD_REPORT.md` for the honest list
of what v0.1.0 actually delivers vs. what the spec called for.

## Honest summary

The previous v0.1.0 attempt claimed all 8 phases shipped. **That was
wrong.** The codebase was never compiled and contained stubbed
APIs that wouldn't have worked against the actual library versions
(Coraza v3 didn't exist on the proxy, Fiber v3 API differs from v2,
lego v4 API differs from what the code assumed).

This **v0.1.0 Foundation Preview** is a **honest rewrite** that ships
only the Foundation (Phase 1 of the spec):

- JWT auth (15m access, 30d refresh with rotation)
- bcrypt password hashing
- 12 default RBAC roles + permission middleware
- License key parsing + feature gating
- Append-only audit log with SHA-256 hash chain
- Token-bucket rate limiter
- Minimal `accounts` and `tenants` CRUD schema
- Bootstrap admin on first boot (prints password to log)
- Health/ready probes
- Embedded React frontend: **not shipped in v0.1.0**

Everything else (hosting engine, DNS, mail, SSL, WAF, firewall,
CrowdSec, file manager, backups, Guardian, reseller, white-label,
provisioning API, WHMCS module) is **deferred to v1.1+** and listed
explicitly in the BUILD_REPORT.

## What's in v0.1.0

### Endpoints
- `GET  /healthz`                            — 200 ok
- `GET  /readyz`                             — 200 ready
- `POST /auth/login`                         — email + password → JWT
- `POST /auth/refresh`                       — refresh token rotation
- `POST /auth/logout`                        — revoke current session
- `GET  /api/v1/me`                          — caller claims
- `GET  /api/v1/admin/system`                — build info
- `GET  /api/v1/admin/license`               — current license
- `GET  /api/v1/admin/audit-log?limit=N`    — recent audit rows
- `POST /api/v1/admin/audit-log/verify`     — chain integrity check
- everything else: `501 phase_X_pending`

### Build artifacts
- 12.0 MB binary (`bin/orvixpanel.exe`)
- SHA-256: see `BUILD_REPORT.md`
- Pure-Go SQLite driver (`github.com/glebarez/sqlite`) — no CGO required
- Single static binary, zero external runtime deps

### Tests
- `internal/auth/jwt_test.go` — bcrypt round-trip + password policy
  matrix
- All other packages: no tests yet (v1.1 will add them)

## What's NOT in v0.1.0 (v1.1+ backlog)

1. Linux system user provisioning (`useradd`, `setquota`, cgroups v2)
2. Nginx vhost generator + PHP-FPM pool generator
3. Multi-PHP version support
4. Git deploy with atomic symlink swap
5. Cron job manager
6. Node / Python / Ruby / Static app runtimes
7. PowerDNS client + DNS zone/record CRUD + BIND parser
8. Postlane mail bridge client (mailbox, alias, queue, DKIM)
9. ACME/Let's Encrypt cert manager
10. Coraza WAF (OWASP CRS 3.3, paranoia 1-4)
11. eBPF/XDP firewall (iptables in Phase 4 spec, deferred)
12. CrowdSec bouncer + brute-force protection (Redis-backed)
13. MySQL/Postgres manager
14. Web file manager (chunked upload, path-traversal protection)
15. Backup system (tar + zstd + AES-256-GCM; local / S3 / SFTP)
16. Metric collector (`/proc/stat`, `/proc/meminfo`, etc.)
17. Modified Z-score anomaly detector
18. Auto-heal engine (restart service, kill process, etc.)
19. LLM insights (OpenAI / Anthropic / Ollama)
20. WebSocket live metrics stream
21. Reseller CRUD + white-label theme engine
22. Provisioning API for WHMCS / Blesta
23. WHMCS PHP module
24. React frontend (login, dashboard, file manager)
25. `orvixpanel init` interactive setup wizard
26. Real TLS termination in the Go binary (v0.1.0 is HTTP only; put
    nginx in front for production)
27. Full unit test sweep (target: 80% coverage on security packages)

## How to verify

```bash
# Build
go build -ldflags="-s -w -X main.version=1.0.0" -o bin/orvixpanel ./cmd/orvixpanel

# Test
go test ./...

# Run
export ORVIX_ALLOW_DEV=1
./bin/orvixpanel
# Check the log for "BOOTSTRAP: ... password=<...>"

# Smoke test
curl http://localhost:8443/healthz
TOKEN=$(curl -s -X POST http://localhost:8443/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"admin@orvixpanel.local","password":"<paste>"}' | jq -r .access_token)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/v1/me
curl -X POST -H "Authorization: Bearer $TOKEN" \
    http://localhost:8443/api/v1/admin/audit-log/verify
# → {"tampered":false,"first_bad_row":-1}
```

## Breaking changes from "v0.1.0" (the previous hallucination)

Everything. The previous 8-phase build did not compile, did not
run, and contained phantom dependencies. The current v0.1.0 is the
first build that actually works.

## Why the honest downgrade

Because the previous v0.1.0 was a fake. The BUILD_REPORT has the
detailed timeline (PowerShell sed-style replace, ~70 files
collapsed to 1 line each, automated recovery failed, deliberate
trash + minimal rewrite was the only path to a verifiable build).

## License

OrvixPanel is **commercial open-core**. The v0.1.0 binary is released
under the **Business Source License 1.1** — free for non-production
use, converts to Apache 2.0 after 4 years. See the spec for the
tier matrix (SMB / ISP / Enterprise / White-Label).

For commercial deployment: https://orvixpanel.com/pricing
