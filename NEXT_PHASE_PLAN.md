# OrvixPanel v0.2.0 — Core Hosting Engine (NEXT PHASE PLAN)

> **Target:** v0.2.0 = spec Phase 2 (Weeks 3-4 in the original timeline)
> **Goal:** "Domain added, nginx config generated, PHP site live"
> **Effort:** ~2-3 weeks of focused work
> **Released:** TBD

This document is the v0.1.0 → v0.2.0 roadmap. It enumerates every
file we will create, every API we will ship, and every verification
step we will pass before tagging v0.2.0. Everything here will be
done through git commits on the `main` branch — no more local-only
builds.

## Scope

v0.2.0 ships **Phase 2 of the spec** end-to-end:

1. Linux system user provisioning (`useradd`, `setquota`, cgroup v2)
2. Per-domain nginx vhost generator
3. Per-domain PHP-FPM pool generator (multi-version)
4. Domain CRUD (full v1.0 stubs replaced)
5. Resource quota tracking (disk / inodes / bandwidth)
6. Git deploy with atomic symlink swap
7. Cron job manager
8. Application runtimes: Node, Python, Ruby, Static

What v0.2.0 does **NOT** ship (deferred):

- DNS, Mail, SSL (v0.3.0)
- WAF, eBPF firewall, CrowdSec (v0.4.0)
- DB / file manager / backup (v0.5.0)
- Guardian AI (v0.6.0)
- Reseller / white-label / WHMCS (v0.7.0)
- React frontend (separate stream)

## File plan

```
internal/
├── hosting/                          # new package
│   ├── provision.go                  # ProvisionAccount / DeprovisionAccount / Suspend
│   ├── platform_linux.go             # //go:build linux
│   ├── platform_other.go             # //go:build !linux (returns ErrUnsupported)
│   ├── nginx.go                      # GenerateNginxVHost + nginxMainSnippet
│   ├── phpfpm.go                     # GeneratePHPFPMPool + PHPConfig struct
│   ├── quota.go                      # QuotaTracker + bandwidth collector
│   ├── gitdeploy.go                  # GitDeploy.Execute (atomic symlink swap)
│   ├── cron.go                       # InstallCron / RemoveCron
│   ├── runtime.go                    # NodeManager / PythonManager / RubyManager
│   ├── provision_test.go             # integration test (skips on non-linux)
│   └── quota_test.go
│
├── api/v1/
│   ├── accounts.go                   # rewrite: wire to hosting.ProvisionAccount
│   ├── domains.go                    # rewrite: full CRUD, license-gated
│   ├── files.go                      # rename of the hosting's web file manager (deferred to v0.5)
│   └── account_test.go               # table-driven tests for the provisioning path
│
├── embed/
│   └── dist/
│       └── index.html                 # server-rendered admin shell (no React yet)
│
└── scheduler/
    └── scheduler.go                  # upgrade stub → real Asynq wire-up

configs/
└── orvixpanel.example.toml           # add [quota] [php] [runtime] sections

scripts/
├── install.sh                        # rewrite for v0.2.0 (no fake phases)
├── php-fpm-pool-test.sh              # verify a generated pool config is valid
└── nginx-vhost-test.sh               # verify a generated vhost is `nginx -t` clean

docs/
├── ARCHITECTURE.md                   # one-pager: requests → middleware → handlers
├── DEPLOY.md                         # end-to-end Linux install + smoke test
└── SECURITY_AUDIT.md                 # rewrite for v0.2.0 (we have WAF? no. firewall? no.)

Makefile                              # add `make test` `make lint` `make smoke`
```

## API additions (v0.2.0)

All routes are gated by license feature `hosting.*` (SMB tier and up).
Returns 501 `phase_X_pending` outside the gate.

```
POST   /accounts                            # create account (now provisions)
GET    /accounts                            # list
GET    /accounts/:id                        # detail (with disk + bandwidth usage)
PUT    /accounts/:id                        # update (re-runs quota apply)
DELETE /accounts/:id                        # deprovision + drop
POST   /accounts/:id/suspend                # usermod -L + pkill
POST   /accounts/:id/unsuspend              # usermod -U
GET    /accounts/:id/usage                  # disk + inodes + bandwidth

POST   /accounts/:id/domains                # create domain (now generates vhost + pool)
GET    /accounts/:id/domains                # list
GET    /domains/:id                         # detail
PUT    /domains/:id                         # update (e.g. PHP version)
DELETE /domains/:id                         # drop (removes vhost + pool)

POST   /domains/:id/deploy                  # trigger a git deploy
GET    /domains/:id/deploy/history          # list past deploys
POST   /domains/:id/deploy/webhook          # GitHub/GitLab webhook receiver

POST   /accounts/:id/cron                   # install a cron job
GET    /accounts/:id/cron                   # list
DELETE /accounts/:id/cron/:jobId            # remove
```

## License feature gating

| Feature id        | SMB | ISP | Enterprise | White-Label |
|-------------------|-----|-----|------------|--------------|
| `hosting.*`       | ✅  | ✅  | ✅         | ✅           |
| `hosting.php`     | ✅  | ✅  | ✅         | ✅           |
| `hosting.node`    | ❌  | ✅  | ✅         | ✅           |
| `hosting.ruby`    | ❌  | ✅  | ✅         | ✅           |
| `hosting.git`     | ✅  | ✅  | ✅         | ✅           |
| `hosting.cron`    | ✅  | ✅  | ✅         | ✅           |
| `quota.disk`      | ✅  | ✅  | ✅         | ✅           |
| `quota.bandwidth` | ✅  | ✅  | ✅         | ✅           |

The v0.1.0 `license` package already has the engine; v0.2.0 just
adds the new feature ids to `TierFeatures` and wires the gates into
the new routes.

## Verification gate

Before tagging v0.2.0, ALL of these must be true on a clean Linux
VPS (Ubuntu 24.04, Go 1.22+, MySQL/Postgres optional):

```bash
# 1. Clean build
go build -ldflags="-s -w -X main.version=0.2.0" -o bin/orvixpanel ./cmd/orvixpanel
# → exit 0

# 2. All tests pass on Linux
go test -race -coverprofile=coverage.out ./...
# → ok for all packages
# → coverage on internal/hosting, internal/api/v1, internal/auth >= 80%

# 3. Installer works on a fresh Ubuntu
curl -fsSL install.sh | sudo bash
systemctl status orvixpanel
# → active (running)

# 4. End-to-end provisioning on a real box
TOKEN=$(curl -s -X POST https://panel.local/auth/login ... | jq -r .access_token)
ACCT=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  https://panel.local/api/v1/accounts \
  -d '{"username":"alice","domain":"alice.test"}' | jq -r .account.id)
# Verify the system user exists, the home dir is created, the vhost
# is generated, php-fpm pool is loaded, and `curl alice.test` returns 200
id alice.test
getent passwd alice
ls -la /home/alice/public_html/alice.test
nginx -t
systemctl status php8.3-fpm
curl -H "Host: alice.test" http://127.0.0.1/  # → 200

# 5. Quota tracking works
# Create 50MB of files in /home/alice, wait 5 min, check the usage endpoint
# → disk_used_mb ≈ 50

# 6. Git deploy roundtrip
# Push to a test repo, trigger the deploy, verify the symlink swap
ls -la /home/alice/current  # → releases/{timestamp}
curl -H "Host: alice.test" http://127.0.0.1/  # → new content

# 7. Audit chain still clean
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://panel.local/api/v1/admin/audit-log/verify
# → {"tampered":false,"first_bad_row":-1}
```

## Risks

1. **Linux-only code paths** — Phase 2 only runs on Linux. The CI
   must have a Linux runner; the Windows dev box is for editing
   only. We'll add `//go:build linux` to every file that imports
   `os/exec` with system commands.
2. **Nginx config templating** — the spec calls for "production-grade
   security headers". The v0.1.0 vhost template has placeholders; the
   v0.2.0 template must pass `nginx -t` clean on a real install. We
   need an integration test that renders a vhost and pipes it to
   `nginx -t`.
3. **PHP-FPM pool reload** — reloading php-fpm after every
   domain create is slow. We'll batch by minute.
4. **Resource quotas** — `setquota` requires the filesystem to be
   mounted with `usrquota`/`grpquota`. We document this in the
   installer; the panel still works without quotas enabled (the
   `QuotaTracker` reports whatever `du` sees).
5. **Git deploy secrets** — for HTTPS repos with token auth, the
   token lives in the deploy row. We encrypt it at rest using
   AES-256-GCM with a key derived from `server.secret_key`. v0.1.0
   already has the encryption primitives; v0.2.0 wires them.

## Out-of-scope for v0.2.0

These are real features, but v0.2.0 specifically does NOT do them:

- **Frontend** — the React app is a v0.8+ task. v0.2.0 serves a
  server-rendered admin shell that shows the v0.1.0 JSON over plain
  HTML tables.
- **TLS** — operators must put nginx in front. Same as v0.1.0.
- **High availability** — single binary, single instance. Multi-node
  is a post-1.0 problem.

## Done means

- [ ] All file paths above exist + compile on Linux
- [ ] `go test ./...` passes with ≥80% coverage on `internal/hosting`
      and `internal/api/v1`
- [ ] End-to-end smoke test on a real Linux VPS passes (steps 1-7
      above)
- [ ] `NEXT_PHASE_PLAN.md` v0.3.0 written
- [ ] git tag `v0.2.0` cut
- [ ] GitHub release created with the verification log
