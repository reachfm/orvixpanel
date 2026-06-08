#!/usr/bin/env bash
# OrvixPanel v0.4.2 production installer.
#
# Idempotent — re-run safely. Targets Ubuntu 22.04+ (incl. WSL
# Ubuntu 26.04) with systemd. On non-systemd environments (e.g.
# WSL without [boot] systemd=true) the script prints a clear
# fallback at the end.
#
# What this does:
#   1. apt install nginx + php-fpm (auto-detects the php version)
#   2. optionally apt install PowerDNS (--with-powerdns)
#   3. creates orvixpanel system user (no login, no home)
#   4. creates /opt/orvixpanel, /var/lib/orvixpanel, /etc/orvixpanel,
#      /var/log/orvixpanel, /run/orvixpanel, /etc/nginx/conf.d/orvix
#   5. installs the binary from $BIN_SRC (default: ./bin/orvixpanel.linux)
#   6. writes /etc/systemd/system/orvixpanel.service
#   7. writes /etc/orvixpanel/orvixpanel.env (operator-editable)
#   8. writes /etc/nginx/conf.d/00-orvixpanel-include.conf
#   9. systemctl enable --now orvixpanel (best effort)
#  10. nginx -t + reload
#  11. doctor.sh report
#
# Usage:
#   sudo bash scripts/install.sh                          # defaults
#   sudo bash scripts/install.sh --bind 0.0.0.0:8443     # custom listen
#   sudo bash scripts/install.sh --master-key <base64>    # inject key
#   sudo bash scripts/install.sh --skip-systemd           # don't touch systemd
#   sudo bash scripts/install.sh --no-start              # install only
#   sudo bash scripts/install.sh --with-powerdns          # install PowerDNS
#
# Exit code: 0 success, non-zero on first failed step.
set -euo pipefail
shopt -s nullglob

# -----------------------------------------------------------------------------
# Args
# -----------------------------------------------------------------------------
BIN_SRC="${BIN_SRC:-./bin/orvixpanel.linux}"
BIND_ADDR="${BIND_ADDR:-0.0.0.0:8443}"
DB_DSN="${DB_DSN:-/var/lib/orvixpanel/data.db}"
MASTER_KEY="${ORVIX_MASTER_KEY:-}"
LICENSE_KEY="${ORVIX_LICENSE_KEY:-}"
SKIP_SYSTEMD=0
NO_START=0
DRY_RUN=0
FPM_VERSION=""
WITH_POWERDNS=0

while [ $# -gt 0 ]; do
  case "$1" in
    --bind)         BIND_ADDR="$2"; shift 2 ;;
    --db)           DB_DSN="$2"; shift 2 ;;
    --master-key)   MASTER_KEY="$2"; shift 2 ;;
    --license)      LICENSE_KEY="$2"; shift 2 ;;
    --bin)          BIN_SRC="$2"; shift 2 ;;
    --skip-systemd) SKIP_SYSTEMD=1; shift ;;
    --no-start)     NO_START=1; shift ;;
    --yes|-y)       shift ;;  # for symmetry with uninstall.sh
    --dry-run)      DRY_RUN=1; shift ;;
    --fpm-version)  FPM_VERSION="$2"; shift 2 ;;
    --with-powerdns) WITH_POWERDNS=1; shift ;;
    -h|--help)
      sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
blue()  { printf '\033[34m%s\033[0m\n' "$*"; }
step()  { blue ""; blue "=== $* ==="; }
ok()    { green "OK: $*"; }
die()   { red "FAIL: $*"; exit 1; }
need_root() { [ "$(id -u)" -eq 0 ] || die "must run as root (use sudo)"; }

run() {
  if [ "$DRY_RUN" = 1 ]; then
    echo "  [dry-run] $*"
  else
    "$@"
  fi
}

# -----------------------------------------------------------------------------
# Preflight
# -----------------------------------------------------------------------------
need_root
step "preflight"
command -v apt-get >/dev/null 2>&1 || die "apt-get not found — installer targets Debian/Ubuntu"
[ -r /etc/os-release ] && . /etc/os-release && echo "  OS: $PRETTY_NAME"

# Port-collision precheck. v0.2.2: the binary used to fail with a
# cryptic "address already in use" when something else (Docker on
# WSL, another panel, a leftover dev server) was already bound to
# the chosen port. We refuse up front with a clear message.
BIND_PORT=$(echo "$BIND_ADDR" | cut -d: -f2)
if [ -n "$BIND_PORT" ] && [ "$BIND_PORT" != "0" ]; then
  port_busy=""
  if command -v ss >/dev/null 2>&1; then
    if ss -tln "( sport = :$BIND_PORT )" 2>/dev/null | grep -q ":$BIND_PORT\b"; then
      port_busy="ss"
    fi
  elif command -v lsof >/dev/null 2>&1; then
    if lsof -iTCP:"$BIND_PORT" -sTCP:LISTEN 2>/dev/null | tail -n +2 | grep -q .; then
      port_busy="lsof"
    fi
  elif command -v netstat >/dev/null 2>&1; then
    if netstat -tln 2>/dev/null | grep -q ":$BIND_PORT\b"; then
      port_busy="netstat"
    fi
  fi
  if [ -n "$port_busy" ]; then
    # Find the conflicting process for a more actionable error message.
    conflict_owner=$(ss -tlnp "( sport = :$BIND_PORT )" 2>/dev/null | grep ":$BIND_PORT\b" | head -1 | sed -E 's/.*users:\(\("([^"]+)".*/\1/')
    if [ -z "$conflict_owner" ] && command -v lsof >/dev/null 2>&1; then
      conflict_owner=$(lsof -iTCP:"$BIND_PORT" -sTCP:LISTEN 2>/dev/null | tail -n +2 | awk '{print $1}' | head -1)
    fi
    if [ -n "$conflict_owner" ]; then
      die "bind port $BIND_PORT is already in use by '$conflict_owner' (detected via $port_busy). Free it (e.g. 'systemctl stop $conflict_owner' or kill the pid) or re-run with a different port: --bind 0.0.0.0:<other>"
    else
      die "bind port $BIND_PORT is already in use (detected via $port_busy). Free it, or re-run with --bind 0.0.0.0:<other>"
    fi
  fi
  echo "  bind port $BIND_PORT: free"
fi

ok "preflight passed"

# -----------------------------------------------------------------------------
# 1. apt install nginx + php-fpm (auto-detect version)
# -----------------------------------------------------------------------------
step "1. install nginx + php-fpm"
export DEBIAN_FRONTEND=noninteractive
run apt-get update -qq
run apt-get install -y -qq nginx curl
if [ -z "$FPM_VERSION" ]; then
  # pick the highest installed-or-available php*-fpm package
  candidate=$(apt-cache search '^php[0-9.]+-fpm$' 2>/dev/null | awk '{print $1}' | sort -V | tail -1)
  if [ -z "$candidate" ]; then
    # fall back to the default php-fpm meta-package
    FPM_VERSION="8.5"
  else
    # "php8.5-fpm" -> "8.5"
    FPM_VERSION="${candidate#php}"   # strip leading "php"
    FPM_VERSION="${FPM_VERSION%-fpm}"  # strip trailing "-fpm"
  fi
fi
echo "  detected php version: php${FPM_VERSION}"
run apt-get install -y -qq "php${FPM_VERSION}-fpm" "php${FPM_VERSION}-cli"
# verify — Debian binary is "php-fpm<ver>"
PHP_BIN="php-fpm${FPM_VERSION}"
command -v "$PHP_BIN" >/dev/null 2>&1 || die "php-fpm binary not found: $PHP_BIN (tried: $PHP_BIN)"
ok "nginx + php${FPM_VERSION}-fpm installed"

# -----------------------------------------------------------------------------
# 2. optionally install PowerDNS
# -----------------------------------------------------------------------------
if [ "$WITH_POWERDNS" = 1 ]; then
  step "2. install PowerDNS"
  export DEBIAN_FRONTEND=noninteractive
  run apt-get update -qq
  # Install PowerDNS Server and SQLite backend
  run apt-get install -y -qq pdns-server pdns-backend-sqlite3 dnsutils
  # Create PowerDNS database
  POWERDNS_DB="/var/lib/orvixpanel/powerdns.sqlite3"
  run mkdir -p "$(dirname "$POWERDNS_DB")"
  run chown pdns:pdns "$(dirname "$POWERDNS_DB")" 2>/dev/null || true

  # Generate API key
  POWERDNS_API_KEY=$(head -c 32 /dev/urandom | base64 -w 0)

  # Configure PowerDNS
  cat > /etc/powerdns/pdns.conf <<EOF
# PowerDNS Configuration for OrvixPanel
# Generated by install.sh --with-powerdns

# Webserver (for API access)
webserver=yes
webserver-address=127.0.0.1
webserver-port=8081
webserver-api-key=${POWERDNS_API_KEY}
webserver-allow-from=127.0.0.1

# SQLite3 Backend
launch=gsqlite3
gsqlite3-database=${POWERDNS_DB}

# API Configuration
api=yes
api-key=${POWERDNS_API_KEY}

# Logging
log-dns-queries=no
loglevel=warning
EOF
  run chown pdns:pdns /etc/powerdns/pdns.conf 2>/dev/null || true

  # Start PowerDNS
  if command -v systemctl >/dev/null 2>&1; then
    run systemctl enable pdns
    run systemctl restart pdns
  fi

  # Store PowerDNS config for env file
  POWERDNS_URL="http://127.0.0.1:8081"
  POWERDNS_SERVER_ID="localhost"

  ok "PowerDNS installed with API key configured"
else
  step "2. skip PowerDNS (use --with-powerdns to install)"
fi

# -----------------------------------------------------------------------------
# 3. orvixpanel system user (no login)
# -----------------------------------------------------------------------------
step "3. create orvixpanel system user"
if ! id orvixpanel >/dev/null 2>&1; then
  run useradd --system --no-create-home --shell /usr/sbin/nologin --user-group orvixpanel
  ok "user orvixpanel created"
else
  ok "user orvixpanel already exists"
fi

# -----------------------------------------------------------------------------
# 4. directories
# -----------------------------------------------------------------------------
step "4. create directories"
for d in /opt/orvixpanel /opt/orvixpanel/bin \
         /var/lib/orvixpanel /var/lib/orvixpanel/homes \
         /var/lib/orvixpanel/releases \
         /var/log/orvixpanel \
         /run/orvixpanel \
         /etc/orvixpanel \
         /etc/nginx/conf.d/orvix; do
  run mkdir -p "$d"
done
run chown -R orvixpanel:orvixpanel /var/lib/orvixpanel /var/log/orvixpanel /run/orvixpanel
# /var/lib/orvixpanel must be world-traversable so the nginx
# worker (www-data) can reach account homes. Same for homes/
# and releases/. We do NOT make the db file writable by world.
run chmod 0755 /var/lib/orvixpanel
run chmod 0755 /var/lib/orvixpanel/homes /var/lib/orvixpanel/releases
run chmod 0755 /opt/orvixpanel /etc/orvixpanel
ok "directories created"

# -----------------------------------------------------------------------------
# 5. install binary
# -----------------------------------------------------------------------------
step "5. install binary"
[ -f "$BIN_SRC" ] || die "binary not found: $BIN_SRC (build it first: GOOS=linux go build -o $BIN_SRC ./cmd/orvixpanel)"
run install -m 0755 "$BIN_SRC" /opt/orvixpanel/bin/orvixpanel
BIN_SHA=$(sha256sum /opt/orvixpanel/bin/orvixpanel | awk '{print $1}')
ok "binary installed sha256=${BIN_SHA:0:16}…"

# -----------------------------------------------------------------------------
# 6. env file
# -----------------------------------------------------------------------------
step "6. write /etc/orvixpanel/orvixpanel.env"
if [ -z "$MASTER_KEY" ]; then
  MASTER_KEY=$(head -c 32 /dev/urandom | base64 -w 0)
  echo "  generated fresh master key"
fi
# JWT signing key — separate from the master key. 32 random bytes,
# base64'd. 64 chars min.
JWT_KEY=$(head -c 32 /dev/urandom | base64 -w 0)
cat > /etc/orvixpanel/orvixpanel.env <<EOF
# OrvixPanel runtime config. Edit + restart: systemctl restart orvixpanel
ORVIX_SERVER_BIND_ADDR=${BIND_ADDR}
ORVIX_DATABASE_DSN=${DB_DSN}
ORVIX_FPM_VERSION=${FPM_VERSION}
ORVIX_MASTER_KEY=${MASTER_KEY}
ORVIX_SERVER_SECRET_KEY=${JWT_KEY}
# ORVIX_ALLOW_DEV=1 enables a development fallback license (SMB tier,
# expires 2030-01-01) so the panel boots without a real license key.
# For production, set this to 0 and provide ORVIX_LICENSE_KEY.
ORVIX_ALLOW_DEV=1
ORVIX_DEV_LICENSE_EXPIRES_AT=2030-01-01
ORVIX_ALLOW_LOCAL_TLD=0

# DNS Configuration
# ORVIX_DNS_MODE: local (SQLite only) or powerdns (sync to PowerDNS)
ORVIX_DNS_MODE=${ORVIX_DNS_MODE:-local}
EOF
# Optional license key
if [ -n "$LICENSE_KEY" ]; then
  echo "ORVIX_LICENSE_KEY=${LICENSE_KEY}" >> /etc/orvixpanel/orvixpanel.env
fi

# Optional PowerDNS config
if [ "$WITH_POWERDNS" = 1 ]; then
  echo "" >> /etc/orvixpanel/orvixpanel.env
  echo "# PowerDNS Integration (installed with --with-powerdns)" >> /etc/orvixpanel/orvixpanel.env
  echo "ORVIX_POWERDNS_URL=${POWERDNS_URL}" >> /etc/orvixpanel/orvixpanel.env
  echo "ORVIX_POWERDNS_API_KEY=${POWERDNS_API_KEY}" >> /etc/orvixpanel/orvixpanel.env
  echo "ORVIX_POWERDNS_SERVER_ID=${POWERDNS_SERVER_ID}" >> /etc/orvixpanel/orvixpanel.env
  echo "ORVIX_DNS_MODE=powerdns" >> /etc/orvixpanel/orvixpanel.env
fi
run chmod 0640 /etc/orvixpanel/orvixpanel.env
run chown root:orvixpanel /etc/orvixpanel/orvixpanel.env
ok "env file written"

# -----------------------------------------------------------------------------
# 7. systemd unit
# -----------------------------------------------------------------------------
step "7. write systemd unit"
if [ "$SKIP_SYSTEMD" = 1 ]; then
  ok "skip-systemd requested; not writing the unit"
else
  cat > /etc/systemd/system/orvixpanel.service <<EOF
[Unit]
Description=OrvixPanel Core Hosting Engine
Documentation=https://github.com/reachfm/orvixpanel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
EnvironmentFile=-/etc/orvixpanel/orvixpanel.env
ExecStart=/opt/orvixpanel/bin/orvixpanel
WorkingDirectory=/var/lib/orvixpanel
RuntimeDirectory=orvixpanel
RuntimeDirectoryMode=0755
Restart=always
RestartSec=5
TimeoutStopSec=20
LimitNOFILE=65535

# Hardening
# Note: ProtectSystem=yes (not =full) keeps /etc writable. groupadd /
# useradd need to write /etc/{passwd,shadow,group,gshadow} and the
# panel needs to write /etc/nginx/conf.d/orvix/ (vhosts) +
# /etc/php/<ver>/fpm/pool.d/ (fpm pools) + /etc/orvixpanel/orvixpanel.env.
# =full would block all of those; =yes protects /usr, /boot, /efi only.
NoNewPrivileges=false
ProtectSystem=yes
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/orvixpanel /var/log/orvixpanel /run/orvixpanel /etc/orvixpanel /etc/nginx/conf.d/orvix /etc/php/${FPM_VERSION}/fpm/pool.d
CapabilityBoundingSet=CAP_DAC_OVERRIDE CAP_CHOWN CAP_FOWNER CAP_FSETID CAP_KILL CAP_SETGID CAP_SETUID CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF
  run systemctl daemon-reload
  run systemctl enable orvixpanel.service
  ok "systemd unit installed + enabled"
fi

# -----------------------------------------------------------------------------
# 8. nginx include for our generated vhosts
# -----------------------------------------------------------------------------
step "8. nginx include for /etc/nginx/conf.d/orvix/*.conf"
cat > /etc/nginx/conf.d/00-orvixpanel-include.conf <<'EOF'
# OrvixPanel — auto-include all generated vhosts.
include /etc/nginx/conf.d/orvix/*.conf;
EOF
# Remove the v0.2.0 scratch file we wrote during smoke.
rm -f /etc/nginx/conf.d/00-orvix-include.conf 2>/dev/null || true
ok "include file written"

# -----------------------------------------------------------------------------
# 9. start + validate
# -----------------------------------------------------------------------------
step "9. start + validate"
run nginx -t
run systemctl reload nginx
if [ "$NO_START" = 1 ]; then
  ok "no-start requested; not starting the service"
elif [ "$SKIP_SYSTEMD" = 1 ]; then
  # skip-systemd: still start the binary directly so the install
  # is verifiable, but don't install the unit
  blue "  skip-systemd: starting binary directly (no unit installed)"
  nohup /opt/orvixpanel/bin/orvixpanel </dev/null >/var/log/orvixpanel/orvixpanel.out 2>&1 &
  disown 2>/dev/null || true
  sleep 2
else
  if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    run systemctl enable --now orvixpanel.service
    sleep 2
  else
    blue "  systemd not running; starting binary directly"
    nohup /opt/orvixpanel/bin/orvixpanel </dev/null >/var/log/orvixpanel/orvixpanel.out 2>&1 &
    disown 2>/dev/null || true
    sleep 2
  fi
fi

# -----------------------------------------------------------------------------
# 10. healthz probe
# -----------------------------------------------------------------------------
step "10. healthz probe"
# extract host:port from BIND_ADDR
HEALTH_HOST=$(echo "$BIND_ADDR" | cut -d: -f1)
[ "$HEALTH_HOST" = "0.0.0.0" ] && HEALTH_HOST=127.0.0.1
HEALTH_PORT=$(echo "$BIND_ADDR" | cut -d: -f2)
for i in 1 2 3 4 5; do
  if curl -fsS "http://${HEALTH_HOST}:${HEALTH_PORT}/healthz" 2>/dev/null | grep -q '"status":"ok"'; then
    ok "binary up at http://${HEALTH_HOST}:${HEALTH_PORT}/healthz"
    break
  fi
  sleep 1
  [ "$i" = 5 ] && { red "binary did not respond on healthz after 5s — check: journalctl -u orvixpanel -n 50"; }
done

# -----------------------------------------------------------------------------
# 11. doctor
# -----------------------------------------------------------------------------
step "11. doctor.sh report"
DOCTOR="$(dirname "$0")/doctor.sh"
[ -x "$DOCTOR" ] && bash "$DOCTOR" || echo "  (doctor.sh not present in repo, skipping)"

green ""
green "==========================================="
green "OrvixPanel v0.4.2 installed."
green "Binary: /opt/orvixpanel/bin/orvixpanel"
green "Listen: ${BIND_ADDR}"
green "Env:    /etc/orvixpanel/orvixpanel.env"
if [ "$WITH_POWERDNS" = 1 ]; then
  green "PowerDNS: enabled (http://127.0.0.1:8081)"
fi
green "Logs:   journalctl -u orvixpanel -f"
green "Doctor: bash scripts/doctor.sh"
green "==========================================="
