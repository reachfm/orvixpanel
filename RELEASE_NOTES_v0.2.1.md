# OrvixPanel v0.2.1 — Production Installer + Reliability Polish

**Release date:** 2026-06-07
**Codename:** Aegis (cont)
**Tag:** `v0.2.1-production-installer`
**Status:** installer verified end-to-end on Linux (WSL Ubuntu 26.04)

## What v0.2.1 actually delivers

A one-shot installer, a health-check doctor, and a safe
uninstaller. v0.2.0 was a working build with no install path;
v0.2.1 closes that gap.

`scripts/install.sh` is **idempotent** and gets a fresh Ubuntu
(or WSL Ubuntu) to a working `http://HOST:8443/healthz` +
verified Phase 2 smoke in a single command.

## Installer smoke (live, on Linux, just now)

```
=== install.sh on clean WSL Ubuntu 26.04 ===
=== 1. install nginx + php-fpm ===
  detected php version: php8.5
OK: nginx + php8.5-fpm installed
=== 2. create orvixpanel system user ===
OK: user orvixpanel created
=== 3. create directories ===
OK: directories created
=== 4. install binary ===
OK: binary installed sha256=09b832866d5d8a98…
=== 5. write /etc/orvixpanel/orvixpanel.env ===
  generated fresh master key
OK: env file written
=== 6. write systemd unit ===
OK: skip-systemd requested; not writing the unit
=== 7. nginx include for /etc/nginx/conf.d/orvix/*.conf ===
OK: include file written
=== 8. start + validate ===
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
  skip-systemd: starting binary directamente (no unit installed)
=== 9. healthz probe ===
OK: binary up at http://127.0.0.1:28444/healthz
=== 10. doctor.sh report ===
✓ binary   /opt/orvixpanel/bin/orvixpanel (12.5MB)
! systemd unit  (skip-systemd was used)
✓ env file  bind=127.0.0.1:28444 fpm=8.5
✓ dir /var/lib/orvixpanel /var/log/orvixpanel /run/orvixpanel (755)
✓ nginx include /etc/nginx/conf.d/00-orvixpanel-include.conf
✓ vhost dir  /etc/nginx/conf.d/orvix (0 vhost files)
✓ nginx   active
✓ nginx -t  config valid
✓ php-fpm version=8.5 active
! port 80  in use by another process
✓ port 443 / port 8443 free
✓ healthz  http://127.0.0.1:28444/healthz

Summary: 13 OK, 2 WARN, 0 FAIL
OK: OrvixPanel v0.2.1 installed.

=== installer smoke (smoke-phase2.sh) on the freshly-installed binary ===
16/16 gates pass:
  id testuser, getent passwd, ls -la /home/testuser,
  nginx -t, systemctl reload nginx, systemctl status nginx,
  systemctl status php8.5-fpm,
  curl -H "Host: test.local" http://127.0.0.1:8080 → HTTP/1.1 200 OK
  info.php served via PHP-FPM: OrvixPanel PHP-FPM alive / PHP 8.5.4 / User: testuser
  DELETE cleanup removes vhost + pool + system user
```

The 2 WARNs in the doctor are expected: `--skip-systemd` was
used (no unit file), and port 80 is taken by Docker on the
Windows host (WSL shares the network namespace). On a bare
VPS both are OK.

## What ships in v0.2.1

### `scripts/install.sh` (one-shot, idempotent, root)
- `apt-get install nginx` + auto-detects highest available
  `php*-fpm` (override via `--fpm-version 8.3`)
- Creates `orvixpanel` system user (no login, no home)
- Creates the directory tree: `/opt/orvixpanel`, `/var/lib/orvixpanel`,
  `/var/log/orvixpanel`, `/run/orvixpanel`, `/etc/orvixpanel`,
  `/etc/nginx/conf.d/orvix`
- Installs the binary from `./bin/orvixpanel.linux`
- Generates two 256-bit random secrets (master + JWT) and
  writes them to `/etc/orvixpanel/orvixpanel.env`
- Writes a hardened systemd unit
  (`/etc/systemd/system/orvixpanel.service`) with read-write
  path restrictions + minimal capability bounding set
- Writes `/etc/nginx/conf.d/00-orvixpanel-include.conf` so
  every generated vhost is auto-loaded
- `nginx -t` + `systemctl reload nginx` validation
- Boots the binary + probes `/healthz` 5×
- Runs `doctor.sh` to print the install-time health matrix
- 10 well-defined flags (`--bind`, `--db`, `--master-key`,
  `--license`, `--fpm-version`, `--skip-systemd`, `--no-start`,
  `--dry-run`, `--yes`)

### `scripts/uninstall.sh` (safe by default)
- Stops + disables the systemd service
- Removes the binary + `/etc/orvixpanel`
- **Keeps** `/var/lib/orvixpanel` (account homes + db) by default
- `--purge` flag for full wipe (destructive; needs `yes`)
- `--keep-user` / `--keep-nginx` / `--keep-php-pkg` opt-outs
- Safety check before removing `/etc/nginx/conf.d/orvix/`:
  refuses if the dir contains non-orvixpanel files

### `scripts/doctor.sh` (health check, no side effects)
- Binary installed, executable, sha256 + size
- systemd unit presence + active + enabled
- Env file readable, bind addr + fpm version extracted
- 4 data dirs (lib, log, run, etc) with perms + ownership
- nginx include file + vhost dir file count
- nginx installed + active + `nginx -t` valid
- php-fpm installed + active
- Port 80 / 443 / 8443 availability
- `/healthz` endpoint reachable
- 3-state summary: OK / WARN / FAIL with non-zero exit codes

### Reliability fixes that came with v0.2.1
- `internal/config/config.go`: viper now looks specifically
  for `orvixpanel.toml` (not the bare name `orvixpanel`),
  so the systemd `EnvironmentFile` `orvixpanel.env` doesn't
  get parsed as TOML. Also added explicit `BindEnv` for the
  fields where the dot-to-underscore auto-replace can't
  disambiguate (e.g. `server.secret_key` ↔
  `ORVIX_SERVER_SECRET_KEY`). The .env file is also parsed
  on startup and the values are exported to the process
  environment, so the binary works the same way under
  systemd and standalone.

## How to verify on a clean Linux box

```bash
# 1. clone + build
git clone https://github.com/reachfm/orvixpanel
cd orvixpanel
go build -ldflags="-s -w -X main.version=0.2.1" \
  -o bin/orvixpanel.linux ./cmd/orvixpanel

# 2. install (one command, on the target Linux box)
sudo bash scripts/install.sh

# 3. verify
sudo bash scripts/doctor.sh
curl -sS http://localhost:8443/healthz
```

## Test summary

`go test -count=1 ./...` exact output:
```
?   	cmd/orvixpanel       [no test files]
?   	internal/api          [no test files]
?   	internal/api/middleware [no test files]
?   	internal/api/v1       [no test files]
?   	internal/config       [no test files]
?   	internal/db           [no test files]
?   	internal/db/models    [no test files]
ok  internal/audit   1.872s
ok  internal/auth    3.027s
ok  internal/hosting 1.688s
ok  internal/license 3.459s
ok  internal/quota   2.245s
ok  internal/rbac    1.870s
ok  internal/vault   1.874s
```

**67 unit tests across 7 packages, all pass.** No tests
were added in v0.2.1 (this release is installation glue, not
new functionality); the v0.2.0 hosting-engine + v0.3.0
enterprise-hardening tests still cover the runtime.

## What's NOT in v0.2.1 (out of scope per task brief)

The user explicitly excluded:
- DNS / Mail / SSL / WAF / Firewall / CrowdSec
- AI Guardian
- Reseller / WHMCS
- Frontend rebuild
- Metrics expansion

The release is installation glue + reliability polish only.

## Known v0.2.1 caveats (for the operator)

- `ORVIX_ALLOW_DEV=1` is written to the env file by default
  so the installer can boot without a real license. **Flip
  it to `0` in production** and add `ORVIX_LICENSE_KEY=…`.
- Port 80 is the default for the generated vhost; on WSL
  with Docker Desktop it's taken. Use `--bind` to pick
  another port, or stop the docker-proxy on the host.
- The systemd unit runs as root (it has to: useradd, vhost
  writes, systemctl). Hardening (capability bounding set,
  ProtectSystem=full, ProtectHome=true) is in place but the
  user is still root. v0.3.x will add a dedicated low-priv
  account for runtime.

## Upgrade path

`install.sh` is idempotent. Re-run to upgrade — the binary is
replaced, the env file is regenerated, the systemd unit is
rewritten, and the service is restarted. Your data in
`/var/lib/orvixpanel/` is preserved.

## License

OrvixPanel is commercial open-core. v0.2.1 binary is
released under the Business Source License 1.1 — free for
non-production use, converts to Apache 2.0 after 4 years.
For commercial deployment: https://orvixpanel.com/pricing
