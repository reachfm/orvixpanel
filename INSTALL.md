# OrvixPanel — Installation Guide

OrvixPanel v0.2.1 ships a one-shot `install.sh` that turns a
clean Ubuntu 22.04+ (or WSL Ubuntu 26.04) into a working
hosting panel in one command.

## TL;DR

```bash
# Build the binary first (Windows or Linux)
go build -ldflags="-s -w -X main.version=0.2.1" -o bin/orvixpanel.linux ./cmd/orvixpanel

# Install (on the target Linux box)
sudo bash scripts/install.sh

# Verify
sudo bash scripts/doctor.sh
```

## What the installer does

1. `apt-get install nginx` + auto-detects and installs
   the highest available `php*-fpm` (currently `php8.5-fpm` on
   Ubuntu 26.04). Override with `--fpm-version 8.3` if you need
   an older PHP.
2. Creates the `orvixpanel` system user (no login, no home).
3. Creates the directory layout:
   - `/opt/orvixpanel/bin/` — the binary
   - `/var/lib/orvixpanel/{homes,releases,data.db}` — runtime data
   - `/var/log/orvixpanel/` — logs
   - `/run/orvixpanel/` — pidfile, sockets
   - `/etc/orvixpanel/orvixpanel.env` — runtime config (sourced by systemd)
   - `/etc/nginx/conf.d/orvix/` — generated vhost files
4. Installs the binary from `./bin/orvixpanel.linux`.
5. Generates two 256-bit random secrets and writes them to
   the env file:
   - `ORVIX_MASTER_KEY` — vault + license encryption key
   - `ORVIX_SERVER_SECRET_KEY` — JWT signing key
6. Writes `/etc/systemd/system/orvixpanel.service` and
   `systemctl enable`s it (skippable with `--skip-systemd`).
7. Writes `/etc/nginx/conf.d/00-orvixpanel-include.conf` so
   every `*.conf` we drop into `/etc/nginx/conf.d/orvix/` is
   auto-loaded.
8. `nginx -t` + `systemctl reload nginx`.
9. Boots the binary + probes `/healthz` 5×.
10. Runs `doctor.sh` to print the install-time health matrix.

## Environment variables written to `/etc/orvixpanel/orvixpanel.env`

```
ORVIX_SERVER_BIND_ADDR=0.0.0.0:8443
ORVIX_DATABASE_DSN=/var/lib/orvixpanel/data.db
ORVIX_FPM_VERSION=8.5
ORVIX_MASTER_KEY=<32 random bytes, base64>
ORVIX_SERVER_SECRET_KEY=<32 random bytes, base64>
ORVIX_ALLOW_DEV=1                          # flip to 0 in production
ORVIX_DEV_LICENSE_EXPIRES_AT=2030-01-01    # dev license only
ORVIX_ALLOW_LOCAL_TLD=0
```

For production, **set `ORVIX_ALLOW_DEV=0` and provide a real
license key** via the API or by appending `ORVIX_LICENSE_KEY=…`
to the env file.

## Installer flags

```bash
sudo bash scripts/install.sh --help
```

| Flag | Default | Purpose |
|------|---------|---------|
| `--bind HOST:PORT` | `0.0.0.0:8443` | listen address |
| `--db PATH` | `/var/lib/orvixpanel/data.db` | SQLite DSN |
| `--bin PATH` | `./bin/orvixpanel.linux` | source binary |
| `--master-key B64` | random | vault + license encryption key |
| `--license KEY` | (empty) | production license key |
| `--fpm-version VER` | auto-detect | override PHP version |
| `--skip-systemd` | off | don't install the unit |
| `--no-start` | off | install but don't start |
| `--dry-run` | off | print commands without executing |
| `-y` / `--yes` | off | for symmetry with uninstall.sh |

## Common operations

```bash
# Service management (systemd)
sudo systemctl start   orvixpanel
sudo systemctl stop    orvixpanel
sudo systemctl restart orvixpanel
sudo systemctl status  orvixpanel
sudo journalctl -u orvixpanel -f

# Logs
tail -f /var/log/orvixpanel/orvixpanel.out
ls -la /var/log/nginx/orvix-*.{access,error}.log

# Health
sudo bash scripts/doctor.sh
curl -sS http://localhost:8443/healthz

# Uninstall (keeps account data by default; --purge wipes it)
sudo bash scripts/uninstall.sh
```

## WSL notes

WSL on Windows has Docker Desktop on port 80 — port 80 is
unreachable from inside WSL. v0.2.1's default `--bind
0.0.0.0:8443` avoids that. If you need port 80, free it by
stopping the docker-proxy on the Windows side. The Phase 2
smoke (smoke-phase2.sh) uses port 8080 for the same reason.

For systemd inside WSL, ensure `/etc/wsl.conf` has
`[boot] systemd=true`, then `wsl --shutdown` and reopen.

## Verifying the install

The installer smoke is `smoke-phase2.sh` and it expects:
- WSL Ubuntu 22.04+ or a fresh VPS
- `nginx` + `php8.3-fpm`/`php8.4-fpm`/`php8.5-fpm` installed
- the binary running at `BASE`

```bash
BASE=http://127.0.0.1:28444 bash smoke-phase2.sh
```

16/16 gates must pass. The gates (id, getent, ls, nginx -t,
systemctl reload, curl 200) match the Phase 2 verification
matrix in the project spec.

## Upgrading from v0.2.0

`install.sh` is **idempotent**: re-run it. The systemd unit
is re-written, the binary is replaced, the env file is
overwritten with fresh secrets, and the service restarts.

Your data (`/var/lib/orvixpanel/`) is preserved unless you
also pass `--purge` to `uninstall.sh` before reinstalling.

## What's not in v0.2.1 (still on the roadmap)

- TLS termination in the Go binary (operators put nginx in
  front; the vhost listens plaintext on 80/8080)
- DNS / Mail / SSL cert automation (v0.4.0)
- WAF / eBPF firewall (v0.5.0)
- DB / File manager / Backups (v0.6.0)
- AI Guardian (v0.7.0)
- White-label (v0.8.0)

See `NEXT_PHASE_PLAN.md` and `ENTERPRISE_PLAN.md` for the
detailed scope.

## Troubleshooting

- **`nginx: the configuration file /etc/nginx/nginx.conf syntax is ok`** on install but `curl` returns 500 →
  the vhost file was written but the worker can't reach the
  account's home. `ls -la /var/lib/orvixpanel/homes/testuser/`
  — make sure the mode is `0751` (owner) and the parent dir
  is `0755` (world-traversable).
- **`port 80 in use`** → use `--bind 0.0.0.0:8443` (the
  default) or free port 80 on the host.
- **`ORVIX_ALLOW_DEV=1` in production** → set
  `ORVIX_ALLOW_DEV=0` and provide `ORVIX_LICENSE_KEY` before
  starting the service.
- **Service won't start** → `journalctl -u orvixpanel -n 50` or
  `/var/log/orvixpanel/orvixpanel.out`.
