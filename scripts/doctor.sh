#!/usr/bin/env bash
# OrvixPanel v0.2.1 doctor — health check.
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
set -uo pipefail

green() { printf '\033[32m%s\033[0m' "✓"; }
yellow(){ printf '\033[33m%s\033[0m' "!"; }
red()   { printf '\033[31m%s\033[0m' "✗"; }
bold()  { printf '\033[1m%s\033[0m' "$*"; }
rst()   { printf '\033[0m'; }

state_ok=0
state_warn=0
state_fail=0

record() {
  local status="$1" label="$2" detail="${3:-}"
  case "$status" in
    OK)   printf ' %s %-40s %s\n' "$(green ok)" "$label" "${detail}"; state_ok=$((state_ok+1)) ;;
    WARN) printf ' %s %-40s %s\n' "$(yellow warn)" "$label" "${detail}"; state_warn=$((state_warn+1)) ;;
    FAIL) printf ' %s %-40s %s\n' "$(red fail)" "$label" "${detail}"; state_fail=$((state_fail+1)) ;;
  esac
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
bold "OrvixPanel doctor"; rst
echo "  host: $(hostname 2>/dev/null || echo unknown)"
echo "  date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo

# 1. binary
if [ -x /opt/orvixpanel/bin/orvixpanel ]; then
  sha=$(sha256sum /opt/orvixpanel/bin/orvixpanel | awk '{print $1}' | cut -c1-12)
  size=$(stat -c%s /opt/orvixpanel/bin/orvixpanel 2>/dev/null || stat -f%z /opt/orvixpanel/bin/orvixpanel)
  record OK "binary" "/opt/orvixpanel/bin/orvixpanel (${size}B, sha256=${sha}…)"
else
  record FAIL "binary" "/opt/orvixpanel/bin/orvixpanel not found"
fi

# 2. systemd unit
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

# 3. env file
if [ -r /etc/orvixpanel/orvixpanel.env ]; then
  bind=$(grep -E '^ORVIX_SERVER_BIND_ADDR=' /etc/orvixpanel/orvixpanel.env | head -1 | cut -d= -f2)
  fpmv=$(grep -E '^ORVIX_FPM_VERSION=' /etc/orvixpanel/orvixpanel.env | head -1 | cut -d= -f2)
  record OK "env file" "bind=$bind fpm=$fpmv"
else
  record FAIL "env file" "/etc/orvixpanel/orvixpanel.env missing"
fi

# 4. data dirs
for d in /var/lib/orvixpanel /var/log/orvixpanel /run/orvixpanel; do
  if [ -d "$d" ]; then
    perms=$(stat -c%a "$d" 2>/dev/null || stat -f%Lp "$d")
    owner=$(stat -c%U "$d" 2>/dev/null || stat -f%Su "$d")
    record OK "dir" "$d (mode $perms owner $owner)"
  else
    record FAIL "dir" "$d missing"
  fi
done

# 5. nginx include
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

# 6. nginx
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
  # nginx -t (may fail if our include is missing, that's WARN not FAIL here)
  if nginx -t >/dev/null 2>&1; then
    record OK "nginx -t" "config valid"
  else
    record FAIL "nginx -t" "config invalid — run nginx -t to see"
  fi
else
  record FAIL "nginx" "not installed"
fi

# 7. php-fpm
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

# 8. ports
for p in 80 443 8443; do
  if port_in_use "$p"; then
    record WARN "port $p" "in use by another process"
  else
    record OK "port $p" "free"
  fi
done

# 9. healthz
bind=$(grep -E '^ORVIX_SERVER_BIND_ADDR=' /etc/orvixpanel/orvixpanel.env 2>/dev/null | head -1 | cut -d= -f2)
if [ -z "$bind" ]; then bind="0.0.0.0:8443"; fi
host=$(echo "$bind" | cut -d: -f1)
[ "$host" = "0.0.0.0" ] && host=127.0.0.1
port=$(echo "$bind" | cut -d: -f2)
if curl -fsS "http://${host}:${port}/healthz" 2>/dev/null | grep -q '"status":"ok"'; then
  record OK "healthz" "http://${host}:${port}/healthz"
else
  record FAIL "healthz" "no response at http://${host}:${port}/healthz"
fi

# -----------------------------------------------------------------------------
echo
bold "Summary"; rst
printf "  %d OK, %d WARN, %d FAIL\n" "$state_ok" "$state_warn" "$state_fail"

if [ "$state_fail" -gt 0 ]; then
  red "  OrvixPanel is NOT healthy."; rst
  exit 1
fi
if [ "$state_warn" -gt 0 ]; then
  yellow "  OrvixPanel is healthy with warnings."; rst
  exit 2
fi
green "  OrvixPanel is fully healthy."; rst
exit 0
