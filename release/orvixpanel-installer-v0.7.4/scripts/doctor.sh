#!/usr/bin/env bash
# OrvixPanel doctor v0.7.2 — Runtime-discovered health check.
#
# Reads configuration from /etc/orvixpanel/orvixpanel.env
# Supports env keys:
#   ORVIX_SERVER_BIND_ADDR  - bind address (e.g., 127.0.0.1:8443)
#   ORVIX_DATABASE_DSN       - database path (e.g., /var/lib/orvixpanel/data.db)
#   ORVIX_FRONTEND_DIST      - frontend dist path
#   ORVIX_LOG_DIR            - log directory
#   ORVIX_FPM_VERSION        - PHP-FPM version
#
# Falls back to legacy paths ONLY if env file is missing.
#
# Exit code: 0 if all checks OK, 1 if any FAIL, 2 if only WARNs.
set -uo pipefail

# ----------------------------------------------------------------------------
# Flag parsing
# ----------------------------------------------------------------------------
JSON_MODE=0
for arg in "$@"; do
  case "$arg" in
    --json) JSON_MODE=1 ;;
    --help|-h)
      head -25 "$0" | tail -24 | sed 's/^# //'
      exit 0
      ;;
    *)
      echo "unknown flag: $arg" >&2
      exit 64
      ;;
  esac
done

# ----------------------------------------------------------------------------
# Runtime config discovery
# ----------------------------------------------------------------------------
ENV_FILE="/etc/orvixpanel/orvixpanel.env"

# Defaults (fallback only)
DEFAULT_BIND="127.0.0.1:8080"
DEFAULT_DATA_DIR="/opt/orvixpanel/var"
DEFAULT_DB_PATH="${DEFAULT_DATA_DIR}/orvixpanel.db"
DEFAULT_LOG_DIR="/var/log/orvixpanel"
DEFAULT_BACKUP_DIR="${DEFAULT_DATA_DIR}/backups"

# Runtime-discovered values
BIND_ADDR=""
DB_PATH=""
DATA_DIR=""
LOG_DIR=""
BACKUP_DIR=""
FPM_VERSION=""
FRONTEND_DIST=""

if [ -r "$ENV_FILE" ]; then
  # Parse bind address
  BIND_ADDR=$(grep -E '^ORVIX_SERVER_BIND_ADDR=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 | tr -d '"' | tr -d "'")

  # Parse database DSN (extract path from sqlite:///path/to/db)
  # Use parameter expansion to preserve leading slash
  while IFS= read -r line; do
    DB_DSN="${line#*=}"  # Remove everything up to and including the first =
    DB_DSN="${DB_DSN#\"}"  # Remove leading double quote if present
    DB_DSN="${DB_DSN%\"}"  # Remove trailing double quote if present
    DB_DSN="${DB_DSN#\'}"  # Remove leading single quote if present
    DB_DSN="${DB_DSN%\'}"  # Remove trailing single quote if present
    break
  done < <(grep -E '^ORVIX_DATABASE_DSN=' "$ENV_FILE" 2>/dev/null)

  if [ -n "$DB_DSN" ]; then
    # Handle sqlite:///path format - preserve leading slash
    DB_PATH="${DB_DSN#sqlite://}"
  fi

  # Derive data dir from DB path
  if [ -n "$DB_PATH" ]; then
    DATA_DIR=$(dirname "$DB_PATH")
  fi

  # Parse log directory
  LOG_DIR=$(grep -E '^ORVIX_LOG_DIR=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 | tr -d '"' | tr -d "'")

  # Parse frontend dist
  FRONTEND_DIST=$(grep -E '^ORVIX_FRONTEND_DIST=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 | tr -d '"' | tr -d "'")

  # Parse FPM version
  FPM_VERSION=$(grep -E '^ORVIX_FPM_VERSION=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 | tr -d '"' | tr -d "'")

  # Parse backup directory
  BACKUP_DIR=$(grep -E '^ORVIX_BACKUP_DIR=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2 | tr -d '"' | tr -d "'")

  # Detect FPM version from system if not in env
  if [ -z "$FPM_VERSION" ]; then
    for f in /etc/php/*/fpm; do
      [ -d "$f" ] && { FPM_VERSION=$(basename "$(dirname "$f")"); break; }
    done
  fi
fi

# Apply defaults if not set
BIND_ADDR="${BIND_ADDR:-${DEFAULT_BIND}}"
DB_PATH="${DB_PATH:-${DEFAULT_DB_PATH}}"
DATA_DIR="${DATA_DIR:-${DEFAULT_DATA_DIR}}"
LOG_DIR="${LOG_DIR:-${DEFAULT_LOG_DIR}}"
BACKUP_DIR="${BACKUP_DIR:-${DEFAULT_BACKUP_DIR}}"

# Extract host and port from bind address
HOST=$(echo "$BIND_ADDR" | cut -d: -f1)
PORT=$(echo "$BIND_ADDR" | cut -d: -f2)
[ "$HOST" = "0.0.0.0" ] && HOST="127.0.0.1"

HEALTHZ_URL="http://${HOST}:${PORT}/healthz"
READYZ_URL="http://${HOST}:${PORT}/readyz"

# ----------------------------------------------------------------------------
# Output helpers
# ----------------------------------------------------------------------------
green() { printf '\033[32m%s\033[0m' "✓"; }
yellow(){ printf '\033[33m%s\033[0m' "!"; }
red()   { printf '\033[31m%s\033[0m' "✗"; }
bold()  { printf '\033[1m%s\033[0m' "$*"; }
rst()   { printf '\033[0m'; }

state_ok=0
state_warn=0
state_fail=0

JSON_RECORDS=""
add_json_record() {
  local status="$1" label="$2" detail="$3"
  local esc_status esc_label esc_detail
  esc_status=$(printf '%s' "$status"  | sed 's/"/\\"/g')
  esc_label=$(printf '%s'   "$label"   | sed 's/"/\\"/g')
  esc_detail=$(printf '%s'  "$detail"  | sed 's/"/\\"/g')
  local entry
  entry=$(printf '{"status":"%s","check":"%s","detail":"%s"}' "$esc_status" "$esc_label" "$esc_detail")
  if [ -z "$JSON_RECORDS" ]; then
    JSON_RECORDS="$entry"
  else
    JSON_RECORDS="${JSON_RECORDS},${entry}"
  fi
}

record() {
  local status="$1" label="$2" detail="${3:-}"
  case "$status" in
    OK)   state_ok=$((state_ok+1)) ;;
    WARN) state_warn=$((state_warn+1)) ;;
    FAIL) state_fail=$((state_fail+1)) ;;
  esac
  if [ "$JSON_MODE" -eq 1 ]; then
    add_json_record "$status" "$label" "$detail"
  else
    case "$status" in
      OK)   printf ' %s %-40s %s\n' "$(green ok)"   "$label" "$detail" ;;
      WARN) printf ' %s %-40s %s\n' "$(yellow warn)" "$label" "$detail" ;;
      FAIL) printf ' %s %-40s %s\n' "$(red fail)"   "$label" "$detail" ;;
    esac
  fi
}

# ----------------------------------------------------------------------------
# Header
# ----------------------------------------------------------------------------
HOSTNAME_FQDN=$(hostname 2>/dev/null || echo unknown)
DATE_ISO=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ORVIX_VERSION=$(/opt/orvixpanel/bin/orvixpanel version 2>/dev/null | head -1 || echo "unknown")

if [ "$JSON_MODE" -eq 0 ]; then
  bold "OrvixPanel doctor v0.7.2"; rst
  echo "  host: ${HOSTNAME_FQDN}"
  echo "  version: ${ORVIX_VERSION}"
  echo "  date: ${DATE_ISO}"
  echo "  env: ${ENV_FILE}"
  echo "  bind: ${BIND_ADDR}"
  echo "  db: ${DB_PATH}"
  echo "  data: ${DATA_DIR}"
  echo
fi

# ----------------------------------------------------------------------------
# 1. env file
# ----------------------------------------------------------------------------
if [ -r "$ENV_FILE" ]; then
  record OK "env file" "$ENV_FILE"
else
  record FAIL "env file" "not found at $ENV_FILE"
fi

# ----------------------------------------------------------------------------
# 2. binary
# ----------------------------------------------------------------------------
if [ -x /opt/orvixpanel/bin/orvixpanel ]; then
  sha=$(sha256sum /opt/orvixpanel/bin/orvixpanel | awk '{print $1}' | cut -c1-12)
  size=$(stat -c%s /opt/orvixpanel/bin/orvixpanel 2>/dev/null || stat -f%z /opt/orvixpanel/bin/orvixpanel)
  record OK "binary" "/opt/orvixpanel/bin/orvixpanel (${size}B, sha256=${sha}…)"
else
  record FAIL "binary" "/opt/orvixpanel/bin/orvixpanel not found"
fi

# ----------------------------------------------------------------------------
# 3. systemd unit
# ----------------------------------------------------------------------------
if [ -f /etc/systemd/system/orvixpanel.service ]; then
  if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    active=$(systemctl is-active orvixpanel 2>&1 || echo unknown)
    enabled=$(systemctl is-enabled orvixpanel 2>&1 || echo unknown)
    if [ "$active" = "active" ]; then
      record OK "systemd unit" "active=$active enabled=$enabled"
    else
      record FAIL "systemd unit" "active=$active enabled=$enabled"
    fi
  else
    record WARN "systemd unit" "unit file present, systemd not running"
  fi
else
  record WARN "systemd unit" "/etc/systemd/system/orvixpanel.service not present"
fi

# ----------------------------------------------------------------------------
# 4. data dirs (runtime-discovered)
# ----------------------------------------------------------------------------
# Check data directory (derived from DB path)
if [ -d "$DATA_DIR" ]; then
  perms=$(stat -c%a "$DATA_DIR" 2>/dev/null || stat -f%Lp "$DATA_DIR")
  owner=$(stat -c%U "$DATA_DIR" 2>/dev/null || stat -f%Su "$DATA_DIR")
  record OK "data dir" "$DATA_DIR (mode $perms owner $owner)"
else
  record FAIL "data dir" "$DATA_DIR missing"
fi

# Check log directory
if [ -d "$LOG_DIR" ]; then
  record OK "log dir" "$LOG_DIR"
else
  record WARN "log dir" "$LOG_DIR missing"
fi

# Check runtime directory
if [ -d /run/orvixpanel ]; then
  record OK "runtime dir" "/run/orvixpanel"
else
  record WARN "runtime dir" "/run/orvixpanel missing"
fi

# ----------------------------------------------------------------------------
# 5. nginx
# ----------------------------------------------------------------------------
if command -v nginx >/dev/null 2>&1; then
  if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    if systemctl is-active nginx >/dev/null 2>&1; then
      record OK "nginx" "active"
    else
      record FAIL "nginx" "installed but not active"
    fi
  else
    record WARN "nginx" "installed; systemd not running"
  fi
  if nginx -t >/dev/null 2>&1; then
    record OK "nginx -t" "config valid"
  else
    record FAIL "nginx -t" "config invalid — run nginx -t to see"
  fi
else
  record FAIL "nginx" "not installed"
fi

# ----------------------------------------------------------------------------
# 6. nginx include
# ----------------------------------------------------------------------------
if [ -f /etc/nginx/conf.d/00-orvixpanel-include.conf ]; then
  record OK "nginx include" "/etc/nginx/conf.d/00-orvixpanel-include.conf"
else
  record WARN "nginx include" "00-orvixpanel-include.conf not present"
fi
if [ -d /etc/nginx/conf.d/orvix ]; then
  n=$(find /etc/nginx/conf.d/orvix -maxdepth 1 -type f -name '*.conf' | wc -l)
  record OK "vhost dir" "/etc/nginx/conf.d/orvix ($n vhost files)"
else
  record WARN "vhost dir" "/etc/nginx/conf.d/orvix missing (no accounts yet)"
fi

# ----------------------------------------------------------------------------
# 7. php-fpm (optional)
# ----------------------------------------------------------------------------
if [ -n "$FPM_VERSION" ] && command -v "php-fpm${FPM_VERSION}" >/dev/null 2>&1; then
  if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    if systemctl is-active "php${FPM_VERSION}-fpm" >/dev/null 2>&1; then
      record OK "php-fpm" "version=$FPM_VERSION active"
    else
      record WARN "php-fpm" "version=$FPM_VERSION installed but not active"
    fi
  else
    record WARN "php-fpm" "version=$FPM_VERSION installed; systemd not running"
  fi
else
  record OK "php-fpm" "not installed (optional for v0.7+)"
fi

# ----------------------------------------------------------------------------
# 8. ports (check the actual port from env)
# ----------------------------------------------------------------------------
port_in_use() {
  local p="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -tln "( sport = :$p )" 2>/dev/null | grep -q ":$p\b"
  elif command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$p" -sTCP:LISTEN 2>/dev/null | tail -n +2 | grep -q .
  elif command -v netstat >/dev/null 2>&1; then
    netstat -tln 2>/dev/null | grep -q ":$p\b"
  else
    return 1
  fi
}

# Check port from env (8443)
if port_in_use "$PORT"; then
  proc=$(ss -tlnp "( sport = :$PORT )" 2>/dev/null | grep ":$PORT" | awk '{print $6}' | head -1)
  record OK "port $PORT" "in use${proc:+ by $proc}"
else
  record WARN "port $PORT" "not in use"
fi

# ----------------------------------------------------------------------------
# 9. healthz (runtime-discovered URL)
# ----------------------------------------------------------------------------
if curl -fsS --max-time 3 "$HEALTHZ_URL" 2>/dev/null | grep -q '"status":"ok"'; then
  record OK "healthz" "$HEALTHZ_URL"
else
  record FAIL "healthz" "no response at $HEALTHZ_URL"
fi

# ----------------------------------------------------------------------------
# 10. readyz (runtime-discovered URL)
# ----------------------------------------------------------------------------
if curl -fsS --max-time 3 "$READYZ_URL" 2>/dev/null | grep -q '"status":"ready"'; then
  record OK "readyz" "$READYZ_URL"
else
  record FAIL "readyz" "no response at $READYZ_URL"
fi

# ----------------------------------------------------------------------------
# 11. frontend (if configured)
# ----------------------------------------------------------------------------
if [ -n "$FRONTEND_DIST" ] && [ -d "$FRONTEND_DIST" ]; then
  record OK "frontend" "$FRONTEND_DIST"
else
  record WARN "frontend" "dist not found at $FRONTEND_DIST"
fi

# ----------------------------------------------------------------------------
# 12. database (runtime-discovered path)
# ----------------------------------------------------------------------------
if [ -f "$DB_PATH" ]; then
  db_size=$(stat -c%s "$DB_PATH" 2>/dev/null || stat -f%z "$DB_PATH")
  record OK "database" "$DB_PATH (${db_size}B)"

  # Check for key tables
  required_tables=(
    "tenants" "users" "accounts" "audit_entries"
    "api_keys" "custom_roles" "secrets"
    "dns_zones" "dns_records"
    "ssl_certificates" "ssl_events"
    "mail_domains" "mailboxes"
    "backup_jobs" "restore_points"
  )

  missing_tables=()
  for table in "${required_tables[@]}"; do
    if ! sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' AND name='${table}';" 2>/dev/null | grep -q "$table"; then
      missing_tables+=("$table")
    fi
  done

  if [ ${#missing_tables[@]} -eq 0 ]; then
    record OK "db tables" "all ${#required_tables[@]} expected tables present"
  else
    record WARN "db tables" "${#missing_tables[@]} tables missing: ${missing_tables[*]}"
  fi
else
  record FAIL "database" "not found at $DB_PATH"
fi

# ----------------------------------------------------------------------------
# 13. error logs
# ----------------------------------------------------------------------------
ERROR_LOG="${LOG_DIR}/orvixpanel_error.log"
if [ -f "$ERROR_LOG" ]; then
  recent_errors=$(tail -50 "$ERROR_LOG" 2>/dev/null | grep -c "ERROR\|FATAL\|PANIC" || echo 0)
  if [ "$recent_errors" -gt 0 ]; then
    last_error=$(tail -5 "$ERROR_LOG" 2>/dev/null | grep -m1 "ERROR\|FATAL\|PANIC" | head -c 100)
    record WARN "error log" "$recent_errors recent errors (last: ${last_error}…)"
  else
    record OK "error log" "no recent errors"
  fi
else
  record OK "error log" "no error log file (healthy)"
fi

# ----------------------------------------------------------------------------
# 14. systemd service logs
# ----------------------------------------------------------------------------
if command -v journalctl >/dev/null 2>&1; then
  recent_failures=$(journalctl -u orvixpanel --since "1 hour ago" --priority err -n 50 2>/dev/null | wc -l)
  if [ "$recent_failures" -gt 0 ]; then
    record WARN "journal" "$recent_failures error lines in last hour"
  else
    record OK "journal" "no recent errors in journal"
  fi
fi

# ----------------------------------------------------------------------------
# 15. backups (runtime-discovered)
# ----------------------------------------------------------------------------
if [ -d "$BACKUP_DIR" ]; then
  backup_count=$(find "$BACKUP_DIR" -maxdepth 1 -type d 2>/dev/null | wc -l)
  backup_count=$((backup_count - 1))
  if [ "$backup_count" -gt 0 ]; then
    record OK "backups" "$backup_count backup(s) in $BACKUP_DIR"
  else
    record WARN "backups" "no backups found"
  fi
else
  record WARN "backups" "directory not present at $BACKUP_DIR"
fi

# ----------------------------------------------------------------------------
# 16. version file
# ----------------------------------------------------------------------------
VERSION_FILE="/opt/orvixpanel/VERSION"
if [ -f "$VERSION_FILE" ]; then
  version_info=$(cat "$VERSION_FILE" 2>/dev/null | head -3 | tr '\n' ' ')
  record OK "version file" "$version_info"
else
  record WARN "version file" "not found"
fi

# ----------------------------------------------------------------------------
# 17. update timers
# ----------------------------------------------------------------------------
if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
  for timer in orvixpanel-update-check.timer orvixpanel-auto-update.timer; do
    if systemctl list-unit-files | grep -q "^${timer}"; then
      status=$(systemctl is-active "$timer" 2>/dev/null || echo inactive)
      if [ "$status" = "active" ]; then
        record OK "timer" "$timer active"
      else
        record WARN "timer" "$timer $status"
      fi
    fi
  done
fi

# ----------------------------------------------------------------------------
# Final report
# ----------------------------------------------------------------------------
if [ "$state_fail" -gt 0 ]; then
  state="FAIL"
elif [ "$state_warn" -gt 0 ]; then
  state="WARN"
else
  state="OK"
fi

if [ "$JSON_MODE" -eq 1 ]; then
  printf '{"host":"%s","date":"%s","state":"%s","ok":%d,"warn":%d,"fail":%d,"checks":[%s]}\n' \
    "$HOSTNAME_FQDN" "$DATE_ISO" "$state" \
    "$state_ok" "$state_warn" "$state_fail" \
    "$JSON_RECORDS"
else
  echo
  bold "Summary"; rst
  printf "  %d OK, %d WARN, %d FAIL\n" "$state_ok" "$state_warn" "$state_fail"
  case "$state" in
    OK)   green "  OrvixPanel is fully healthy."; rst ;;
    WARN) yellow "  OrvixPanel is healthy with warnings."; rst ;;
    FAIL) red    "  OrvixPanel is NOT healthy."; rst ;;
  esac
  echo
  echo "  Run 'orvixpanel update --check' to check for updates"
fi

case "$state" in
  OK)   exit 0 ;;
  WARN) exit 2 ;;
  FAIL) exit 1 ;;
esac