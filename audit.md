# OrvixPanel — Phase Audit Log

> This log was originally written as an 8-phase "RELEASED" narrative
> by the previous chat session. **That was a hallucination.** The
> previous v1.0 never compiled and the 70-file Go tree was trashed
> during the verification pass.
>
> The honest v1.0 ships only the **Foundation** (Phase 1 of the spec).
> The v1.1+ backlog is enumerated in `BUILD_REPORT.md`.

---

## v1.0 — RELEASED (revised)

**Closed at:** 2026-06-07 (Asia/Dubai)
**Verdict:** `go test ./...` passes · `go build ./cmd/orvixpanel` succeeds
· binary boots · login + audit chain + RBAC verified live

### What shipped

| Area | File(s) | Verified |
|------|---------|----------|
| Entry point | `cmd/orvixpanel/main.go`, `cmd/orvixpanel/bootstrap.go` | ✅ builds + runs |
| Config | `internal/config/config.go`, `internal/config/rand.go` | ✅ compiles |
| DB | `internal/db/db.go` (glebarez/sqlite, no CGO) | ✅ opens + migrates |
| Models | `internal/db/models/models.go` (Tenant, User, UserSession, Account, AuditEntry) | ✅ AutoMigrate |
| Auth | `internal/auth/jwt.go`, `roles.go`, `id.go`, `jwt_test.go` | ✅ `go test` passes |
| Audit | `internal/audit/audit.go` (SHA-256 chain) | ✅ `VerifyChain` returns clean |
| License | `internal/license/license.go` (parse + feature gate) | ✅ compiles |
| Middleware | `internal/api/middleware/{auth,rbac,tenant,ratelimit,audit,context}.go` | ✅ wired |
| API | `internal/api/{server,router,deps}.go` | ✅ listens |
| Handlers | `internal/api/v1/{auth,admin,stubs}.go` | ✅ 200/400/401 returned |
| Bootstrap admin | prints password to log on first boot | ✅ smoke-tested |

### What's deferred (v1.1+)

Everything from Phases 2-8 of the spec. See `BUILD_REPORT.md` for the
full 27-item backlog.

### Why this looks so different from the prior audit.md

The prior `audit.md` (pre-trash) was an aspirational narrative. The
trashed 70-file tree contained:

- Phantom dependency `corazawaf/coraza/v3 v3.0.0-20231110091329-15d97f3a02eb`
  (does not exist)
- Fiber v3 API calls (`c.Method`, `c.Path`, `c.BodyParser`) that
  don't exist on the actual Fiber v2.52.13 we depend on
- lego v4 SSL code targeting a moved API
- BackupPolicy / BackupJob references without the model definitions
- Reseller theme code referencing fields not on the Reseller model
- A `//go:build linux` filename suffix on `provision_linux.go` that
  silently excluded the file on Windows builds
- A broken `ws.go` in the guardian package
- Plus: every file was destroyed by a PowerShell sed that stripped
  all newlines

None of that tree could be repaired cleanly. The v1.0 you have
now is the first verifiable build, and it is honest about what it
does and doesn't do.

---

*Last updated: 2026-06-07 (Asia/Dubai) — v1.0 verified & released (Foundation only).*
