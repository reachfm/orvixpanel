#!/usr/bin/env bash
# OrvixPanel v0.7.1 doctor — comprehensive health check.
#
# Checks (each is OK / WARN / FAIL):
#   - binary installed
#   - systemd unit present + active
#   - env file readable
#   - /var/lib/orvixpanel, /var/log/orvixpanel, /run/orvixpanel
#   - /etc/nginx/conf.d/orvix + include line
#   - nginx installed + active + config valid
#   - php-fpm installed + active
#   - port 80, 443, 8443 availability
#   - /healthz endpoint reachable
#   - database tables (AutoMigrate verification)
#   - recent error logs
#   - public proxy test
#
# Exit code: 0 if all checks OK, 1 if any FAIL, 2 if only WARNs.
#
# Flags:
#   --json   output the report as a single JSON document on stdout
#            (also sets process exit code based on the health state)
#   --help   show usage
set -uo pipefail

# ----------------------------------------------------------------------------
# Flag parsing
# ----------------------------------------------------------------------------
JSON_MODE=0
for arg in "$@"; do
  case "$arg" in
    --json) JSON_MODE=1 ;;
    --help|-h)
      sed -n '2,35p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "unknown flag: $arg" >&2
      exit 64
      ;;
  esac
done

green() { printf '\033[32m%s\033[0m' "✓"; }
yellow(){ printf '\033[33m%s\033[0m' "!"; }
red()   { printf '\033[31m%s\033[0m' "✗"; }
bold()  { printf '\033[1m%s\033[0m' "$*"; }
rst()   { printf '\033[0m'; }

# State accumulators.
state_ok=0
state_warn=0
state_fail=0

# In JSON mode we collect all records and emit a single document at the end.
# In human mode we print as we go.
JSON_RECORDS=""
add_json_record() {
  local status="$1" label="$2" detail="$3"
  # JSON-escape the fields.
  local esc_status esc_label esc_detail
  esc_status=$(printf '%s' "$status"  | sed 's/"/\\"/g')
  esc_label=$(printf '%s'   "$label"   | sed 's/"/\\"/g')
  esc_detail=$(printf '%s'  "$detail"  | sed 's/"/\\"/g')
  local entry
  entry=$(printf '{"status":"%s","check":"%s","detail":"%s"}' \
    "$esc_status" "$esc_label" "$esc_detail")
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

# Detect php-fpm version (best effort)
detect_fpm() {
  for f in /etc/php/*/fpm; do
    [ -d "$f" ] && { basename "$(dirname "$f")"; return; }
  done
  echo ""
}

# port_in_use <port>
port_in_use() {
  local p="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -tln "( sport = :$p )" 2>/dev/null | grep -q ":$p\b"
  elif command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$p" -sTCP:LISTEN 2>/dev/null | tail -n +2 | grep -q .
  elif command -v netstat >/dev/null 2>&1; then
    netstat -tln 2>/dev/null | grep -q ":$p\b"
  else
    return 1   # unknown
  fi
}

# -----------------------------------------------------------------------------
# Header
# -----------------------------------------------------------------------------
HOSTNAME_FQDN=$(hostname 2>/dev/null || echo unknown)
DATE_ISO=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ORVIX_VERSION=$(/opt/orvixpanel/bin/orvixpanel version 2>/dev/null | head -1 || echo "unknown")

if [ "$JSON_MODE" -eq 0 ]; then
  bold "OrvixPanel doctor v0.7.1"; rst
  echo "  host: ${HOSTNAME_FQDN}"
  echo "  version: ${ORVIX_VERSION}"
  echo "  date: ${DATE_ISO}"
  echo
fi

# -----------------------------------------------------------------------------
# 1. binary
# -----------------------------------------------------------------------------
if [ -x /opt/orvixpanel/bin/orvixpanel ]; then
  sha=$(sha256sum /opt/orvixpanel/bin/orvixpanel | awk '{print $1}' | cut -c1-12)
  size=$(stat -c%s /opt/orvixpanel/bin/orvixpanel 2>/dev/null || stat -f%z /opt/orvixpanel/bin/orvixpanel)
  record OK "binary" "/opt/orvixpanel/bin/orvixpanel (${size}B, sha256=${sha}…)"
else
  record FAIL "binary" "/opt/orvixpanel/bin/orvixpanel not found"
fi

# -----------------------------------------------------------------------------
# 2. systemd unit
# -----------------------------------------------------------------------------
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

# -----------------------------------------------------------------------------
# 3. env file
# -----------------------------------------------------------------------------
ENV_FILE="/opt/orvixpanel/etc/orvixpanel.env"
if [ -r "$ENV_FILE" ]; then
  bind=$(grep -E '^ORVIX_BIND=' "$ENV_FILE" | head -1 | cut -d= -f2)
  record OK "env file" "bind=${bind:-unset} (${ENV_FILE})"
else
  # Fallback to old location
  if [ -r /etc/orvixpanel/orvixpanel.env ]; then
    bind=$(grep -E '^ORVIX_BIND=' /etc/orvixpanel/orvixpanel.env | head -1 | cut -d= -f2)
    record OK "env file" "bind=${bind:-unset} (/etc/orvixpanel/orvixpanel.env)"
  else
    record FAIL "env file" "no env file found"
  fi
fi

# -----------------------------------------------------------------------------
# 4. data dirs
# -----------------------------------------------------------------------------
for d in /opt/orvixpanel/var /opt/orvixpanel/var/log /run/orvixpanel; do
  if [ -d "$d" ]; then
    perms=$(stat -c%a "$d" 2>/dev/null || stat -f%Lp "$d")
    owner=$(stat -c%U "$d" 2>/dev/null || stat -f%Su "$d")
    record OK "dir" "$d (mode $perms owner $owner)"
  else
    record FAIL "dir" "$d missing"
  fi
done

# -----------------------------------------------------------------------------
# 5. nginx include
# -----------------------------------------------------------------------------
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

# -----------------------------------------------------------------------------
# 6. nginx
# -----------------------------------------------------------------------------
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

# -----------------------------------------------------------------------------
# 7. php-fpm (optional, for legacy PHP sites)
# -----------------------------------------------------------------------------
FPM_VER=$(detect_fpm)
FPM_BIN="${FPM_VER:+php-fpm}${FPM_VER:+$FPM_VER}"
if [ -n "$FPM_VER" ] && command -v "$FPM_BIN" >/dev/null 2>&1; then
  if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    if systemctl is-active "php${FPM_VER}-fpm" >/dev/null 2>&1; then
      record OK "php-fpm" "version=$FPM_VER active"
    else
      record WARN "php-fpm" "version=$FPM_VER installed but not active"
    fi
  else
    record WARN "php-fpm" "version=$FPM_VER installed; systemd not running"
  fi
else
  record OK "php-fpm" "not installed (optional for v0.7+)"
fi

# -----------------------------------------------------------------------------
# 8. ports
# -----------------------------------------------------------------------------
for p in 80 443 8443; do
  if port_in_use "$p"; then
    # Check if it's our service or something else
    proc=$(ss -tlnp "( sport = :$p )" 2>/dev/null | grep ":$p" | awk '{print $6}' | head -1)
    record WARN "port $p" "in use${proc:+ by $proc}"
  else
    record OK "port $p" "free"
  fi
done

# -----------------------------------------------------------------------------
# 9. healthz
# -----------------------------------------------------------------------------
bind=$(grep -E '^ORVIX_BIND=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2)
if [ -z "$bind" ]; then bind="0.0.0.0:8080"; fi
host=$(echo "$bind" | cut -d: -f1)
[ "$host" = "0.0.0.0" ] && host=127.0.0.1
port=$(echo "$bind" | cut -d: -f2)
if curl -fsS --max-time 3 "http://${host}:${port}/healthz" 2>/dev/null | grep -q '"status":"ok"'; then
  record OK "healthz" "http://${host}:${port}/healthz"
else
  record FAIL "healthz" "no response at http://${host}:${port}/healthz"
fi

# -----------------------------------------------------------------------------
# 10. database tables
# -----------------------------------------------------------------------------
DB_PATH=$(grep -E '^ORVIX_DB_PATH=' "$ENV_FILE" 2>/dev/null | cut -d= -f2)
if [ -z "$DB_PATH" ]; then
  DB_PATH="/opt/orvixpanel/var/orvixpanel.db"
fi

if [ -f "$DB_PATH" ]; then
  # Check for key tables from AutoMigrate
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
  record WARN "db" "database not found at $DB_PATH"
fi

# -----------------------------------------------------------------------------
# 11. recent error logs
# -----------------------------------------------------------------------------
ERROR_LOG="/opt/orvixpanel/var/log/orvixpanel_error.log"
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

# -----------------------------------------------------------------------------
# 12. systemd service logs (last 10 lines)
# -----------------------------------------------------------------------------
if command -v journalctl >/dev/null 2>&1; then
  recent_failures=$(journalctl -u orvixpanel --since "1 hour ago" --priority err -n 50 2>/dev/null | wc -l)
  if [ "$recent_failures" -gt 0 ]; then
    record WARN "journal" "$recent_failures error lines in last hour"
  else
    record OK "journal" "no recent errors in journal"
  fi
fi

# -----------------------------------------------------------------------------
# 13. public proxy test (if nginx is configured for public)
# -----------------------------------------------------------------------------
PUBLIC_IP=$(curl -s --max-time 5 https://api.ipify.org 2>/dev/null || echo "")
if [ -n "$PUBLIC_IP" ]; then
  if curl -fsS --max-time 5 "http://${PUBLIC_IP}/" -o /dev/null 2>/dev/null; then
    record OK "public proxy" "port 80 publicly accessible"
  else
    record WARN "public proxy" "port 80 not accessible from public (may be firewalled)"
  fi
else
  record OK "public proxy" "could not determine public IP"
fi

# -----------------------------------------------------------------------------
# 14. backup directory
# -----------------------------------------------------------------------------
BACKUP_DIR="/opt/orvixpanel/var/backups"
if [ -d "$BACKUP_DIR" ]; then
  backup_count=$(find "$BACKUP_DIR" -maxdepth 1 -type d 2>/dev/null | wc -l)
  backup_count=$((backup_count - 1))  # subtract the dir itself
  if [ "$backup_count" -gt 0 ]; then
    latest_backup=$(find "$BACKUP_DIR" -maxdepth 1 -type d -printf '%T@ %p\n' 2>/dev/null | sort -n | tail -1 | cut -d' ' -f2-)
    record OK "backups" "$backup_count backup(s) available"
  else
    record WARN "backups" "no backups found"
  fi
else
  record WARN "backups" "backup directory not present"
fi

# -----------------------------------------------------------------------------
# 15. version file
# -----------------------------------------------------------------------------
VERSION_FILE="/opt/orvixpanel/VERSION"
if [ -f "$VERSION_FILE" ]; then
  version_info=$(cat "$VERSION_FILE" 2>/dev/null | head -3 | tr '\n' ' ')
  record OK "version file" "$version_info"
else
  record WARN "version file" "VERSION file not found"
fi

# -----------------------------------------------------------------------------
# Final report
# -----------------------------------------------------------------------------
if [ "$state_fail" -gt 0 ]; then
  state="FAIL"
elif [ "$state_warn" -gt 0 ]; then
  state="WARN"
else
  state="OK"
fi

if [ "$JSON_MODE" -eq 1 ]; then
  # Single JSON document, machine-friendly.
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