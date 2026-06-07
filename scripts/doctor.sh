#!/usr/bin/env bash
# OrvixPanel v0.2.2 doctor — health check.
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
      sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
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

if [ "$JSON_MODE" -eq 0 ]; then
  bold "OrvixPanel doctor"; rst
  echo "  host: ${HOSTNAME_FQDN}"
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
if [ -r /etc/orvixpanel/orvixpanel.env ]; then
  bind=$(grep -E '^ORVIX_SERVER_BIND_ADDR=' /etc/orvixpanel/orvixpanel.env | head -1 | cut -d= -f2)
  fpmv=$(grep -E '^ORVIX_FPM_VERSION=' /etc/orvixpanel/orvixpanel.env | head -1 | cut -d= -f2)
  record OK "env file" "bind=${bind:-unset} fpm=${fpmv:-unset}"
else
  record FAIL "env file" "/etc/orvixpanel/orvixpanel.env missing"
fi

# -----------------------------------------------------------------------------
# 4. data dirs
# -----------------------------------------------------------------------------
for d in /var/lib/orvixpanel /var/log/orvixpanel /run/orvixpanel; do
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
# 7. php-fpm
# -----------------------------------------------------------------------------
FPM_VER=$(detect_fpm)
FPM_BIN="${FPM_VER:+php-fpm}${FPM_VER:+$FPM_VER}"
if [ -n "$FPM_VER" ] && command -v "$FPM_BIN" >/dev/null 2>&1; then
  if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
    if systemctl is-active "php${FPM_VER}-fpm" >/dev/null 2>&1; then
      record OK "php-fpm" "version=$FPM_VER active"
    else
      record FAIL "php-fpm" "version=$FPM_VER installed but not active"
    fi
  else
    record WARN "php-fpm" "version=$FPM_VER installed; systemd not running"
  fi
else
  record FAIL "php-fpm" "no php*-fpm binary found"
fi

# -----------------------------------------------------------------------------
# 8. ports
# -----------------------------------------------------------------------------
for p in 80 443 8443; do
  if port_in_use "$p"; then
    record WARN "port $p" "in use by another process"
  else
    record OK "port $p" "free"
  fi
done

# -----------------------------------------------------------------------------
# 9. healthz
# -----------------------------------------------------------------------------
bind=$(grep -E '^ORVIX_SERVER_BIND_ADDR=' /etc/orvixpanel/orvixpanel.env 2>/dev/null | head -1 | cut -d= -f2)
if [ -z "$bind" ]; then bind="0.0.0.0:8443"; fi
host=$(echo "$bind" | cut -d: -f1)
[ "$host" = "0.0.0.0" ] && host=127.0.0.1
port=$(echo "$bind" | cut -d: -f2)
if curl -fsS --max-time 3 "http://${host}:${port}/healthz" 2>/dev/null | grep -q '"status":"ok"'; then
  record OK "healthz" "http://${host}:${port}/healthz"
else
  record FAIL "healthz" "no response at http://${host}:${port}/healthz"
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
fi

case "$state" in
  OK)   exit 0 ;;
  WARN) exit 2 ;;
  FAIL) exit 1 ;;
esac
