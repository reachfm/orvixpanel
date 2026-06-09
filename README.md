# OrvixPanel v0.7.4 — Production Hosting Control Panel

> **Latest release:** v0.7.4 with VPS runtime path fixes. See `CHANGELOG.md` for details.

OrvixPanel is a single-binary, zero-runtime-dep, AI-native server
control panel. v0.7.4 ships the **Core Hosting Engine** with self-update support.

## Installation

```bash
curl -fsSL https://github.com/reachfm/orvixpanel/releases/download/v0.7.4/orvixpanel-installer-v0.7.4.tar.gz | tar -xz && sudo bash scripts/install.sh
```

**Binary Checksum:** `0ac543d297ef1962367ca1b6e966b2d9d8267f7ae7196ecb3c8bce3f9107cc39`

**Supported OS:** Ubuntu 22.04 LTS, Ubuntu 24.04 LTS

## Features
- ✅ JWT auth (15m access, 30d refresh with rotation)
- ✅ bcrypt password hashing + 10-char policy
- ✅ 12 default RBAC roles + permission middleware
- ✅ License key parsing + feature gating (4 tiers)
- ✅ Append-only audit log with SHA-256 hash chain
- ✅ Token-bucket rate limiter (100/min, burst 30)
- ✅ Minimal `accounts` and `tenants` schema
- ✅ Bootstrap admin on first boot
- ✅ Health/ready probes + structured JSON logging

What v0.1.0 does **NOT** ship (deferred to v0.2.0+):

- Linux system user provisioning (useradd/setquota/cgroups)
- Nginx vhost generator + PHP-FPM pool generator
- Git deploy / cron / application runtimes
- PowerDNS / Postlane / Let's Encrypt
- WAF / eBPF firewall / CrowdSec
- Database / file manager / backup
- Guardian AI / LLM insights / live metrics WS
- Reseller / white-label / WHMCS
- React frontend (the binary is API-only)

The binary works. The tests pass. The audit chain verifies. **Do
not deploy this expecting a hosting panel** — it is auth + admin
API surface only.

## Build

```bash
go build -ldflags="-s -w -X main.version=0.1.0" -o bin/orvixpanel ./cmd/orvixpanel
```

## Run (dev mode)

```bash
export ORVIX_ALLOW_DEV=1
export ORVIX_SERVER_BIND_ADDR=127.0.0.1:18443
export ORVIX_DATABASE_DSN=/tmp/orvixpanel.db
./bin/orvixpanel
# Bootstrap admin: check the log for "BOOTSTRAP: ... password=<...>"
```

## Endpoints (v0.1.0)

| Method | Path                              | Result                          |
|--------|-----------------------------------|---------------------------------|
| GET    | `/healthz`                        | 200 ok                          |
| GET    | `/readyz`                         | 200 ready                       |
| POST   | `/auth/login`                     | JWT pair + user                 |
| POST   | `/auth/refresh`                   | new pair (rotation)             |
| POST   | `/auth/logout`                    | revoke current session          |
| GET    | `/api/v1/me`                      | caller claims                   |
| GET    | `/api/v1/admin/system`            | build info                      |
| GET    | `/api/v1/admin/license`           | current license                 |
| GET    | `/api/v1/admin/audit-log?limit=N` | recent audit rows               |
| POST   | `/api/v1/admin/audit-log/verify`  | chain integrity check           |
| *      | any other spec route              | 501 `phase_X_pending`           |

## Tests

```bash
go test ./...   # 1 test file (auth) — passes
```

## File map

```
cmd/orvixpanel/         # main.go, bootstrap.go (25 .go files total)
internal/api/           # server, router, deps, v1 handlers
internal/api/middleware # auth, rbac, tenant, ratelimit, audit, context
internal/audit/         # SHA-256 chain
internal/auth/          # JWT + bcrypt + roles + 1 test file
internal/config/        # Viper + TOML + env
internal/db/            # GORM + AutoMigrate
internal/db/models/     # Tenant, User, UserSession, Account, AuditEntry
internal/license/       # parse + feature gating
go.mod                   # 9 deps
bin/orvixpanel(.exe)     # 12 MB static binary
```

## Where to next

- `NEXT_PHASE_PLAN.md` — v0.2.0 Core Hosting Engine roadmap
- `BUILD_REPORT.md` — verification status (build + test + smoke)
- `RELEASE_NOTES.md` — what shipped + what didn't
- `audit.md` — phase log
- `OrvixPanel-MVP.md` — the original spec

## License

Commercial open-core (BSL 1.1 → Apache 2.0 after 4 years). See the
spec for the tier matrix.
