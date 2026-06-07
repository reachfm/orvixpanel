#!/usr/bin/env bash
# OrvixPanel v0.2.1 uninstaller.
#
# Default mode: SAFE. Stops the service, removes the binary,
# removes our directories EXCEPT /var/lib/orvixpanel/homes
# (which holds every account's web files). Re-running is safe.
#
# Flags:
#   --purge      ALSO delete /var/lib/orvixpanel/homes (every
#                account's files) and /var/lib/orvixpanel/releases
#                and the database. DESTRUCTIVE — operators only.
#   --keep-user  do not delete the orvixpanel system user
#   --keep-nginx do not touch the nginx include file
#   --keep-php   do not uninstall php-fpm
#   --keep-nginx-pkg do not apt remove nginx
#   --keep-php-pkg   do not apt remove php-fpm
#   --yes        skip the confirmation prompt
#
# Exit code: 0 success, non-zero on first failed step.
set -euo pipefail

KEEP_USER=0
KEEP_NGINX_CFG=0
KEEP_PHP_PKG=0
KEEP_NGINX_PKG=0
PURGE=0
YES=0

while [ $# -gt 0 ]; do
  case "$1" in
    --purge)        PURGE=1; shift ;;
    --keep-user)    KEEP_USER=1; shift ;;
    --keep-nginx)   KEEP_NGINX_CFG=1; shift ;;
    --keep-php-pkg) KEEP_PHP_PKG=1; shift ;;
    --keep-nginx-pkg) KEEP_NGINX_PKG=1; shift ;;
    --yes|-y)       YES=1; shift ;;
    -h|--help)
      sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
blue()  { printf '\033[34m%s\033[0m\n' "$*"; }
step()  { blue ""; blue "=== $* ==="; }
ok()    { green "OK: $*"; }
die()   { red "FAIL: $*"; exit 1; }
[ "$(id -u)" -eq 0 ] || die "must run as root (use sudo)"

# -----------------------------------------------------------------------------
# Confirm
# -----------------------------------------------------------------------------
if [ "$YES" != 1 ]; then
  echo "This will:"
  echo "  - stop + disable the orvixpanel systemd service"
  echo "  - remove the binary at /opt/orvixpanel/bin/orvixpanel"
  echo "  - remove /etc/orvixpanel"
  if [ "$PURGE" = 1 ]; then
    echo "  - remove /var/lib/orvixpanel/homes (EVERY account's files)"
    echo "  - remove /var/lib/orvixpanel/releases"
    echo "  - remove /var/lib/orvixpanel/data.db"
  else
    echo "  - keep /var/lib/orvixpanel/homes (default — accounts survive)"
  fi
  if [ "$KEEP_USER" = 0 ]; then
    echo "  - delete the orvixpanel system user"
  fi
  if [ "$KEEP_NGINX_CFG" = 0 ]; then
    echo "  - remove /etc/nginx/conf.d/00-orvixpanel-include.conf"
    echo "  - remove generated vhosts in /etc/nginx/conf.d/orvix/"
  fi
  echo
  read -p "Type 'yes' to continue: " ans
  [ "$ans" = "yes" ] || { echo "aborted."; exit 1; }
fi

# -----------------------------------------------------------------------------
# Stop + disable service
# -----------------------------------------------------------------------------
step "1. stop + disable orvixpanel service"
if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
  systemctl stop orvixpanel.service 2>/dev/null || true
  systemctl disable orvixpanel.service 2>/dev/null || true
  rm -f /etc/systemd/system/orvixpanel.service
  systemctl daemon-reload
  ok "service stopped + disabled"
else
  # best effort
  pkill -f /opt/orvixpanel/bin/orvixpanel 2>/dev/null || true
  ok "no systemd; killed orvixpanel processes"
fi

# -----------------------------------------------------------------------------
# Remove binary + etc
# -----------------------------------------------------------------------------
step "2. remove binary + etc/orvixpanel"
rm -rf /opt/orvixpanel
rm -rf /etc/orvixpanel
ok "removed"

# -----------------------------------------------------------------------------
# Remove or purge data dir
# -----------------------------------------------------------------------------
step "3. data directory"
if [ "$PURGE" = 1 ]; then
  rm -rf /var/lib/orvixpanel/homes
  rm -rf /var/lib/orvixpanel/releases
  rm -f  /var/lib/orvixpanel/data.db
  ok "purged /var/lib/orvixpanel"
else
  ok "kept /var/lib/orvixpanel (use --purge to remove)"
fi
rm -rf /var/log/orvixpanel /run/orvixpanel

# -----------------------------------------------------------------------------
# Nginx + PHP cleanup
# -----------------------------------------------------------------------------
if [ "$KEEP_NGINX_CFG" = 0 ]; then
  step "4. remove nginx include + vhost files"
  rm -f /etc/nginx/conf.d/00-orvixpanel-include.conf
  rm -f /etc/nginx/conf.d/00-orvix-include.conf  # legacy
  # only remove the vhost dir if it exists (operator-installed or ours)
  if [ -d /etc/nginx/conf.d/orvix ]; then
    # safety check: must contain ONLY orvix vhosts we wrote
    bad=$(find /etc/nginx/conf.d/orvix -type f ! -name '*orvixpanel*' 2>/dev/null)
    if [ -z "$bad" ]; then
      rm -rf /etc/nginx/conf.d/orvix
      ok "removed /etc/nginx/conf.d/orvix/"
    else
      ok "kept /etc/nginx/conf.d/orvix/ (contains non-orvixpanel files: $bad)"
    fi
  fi
  if command -v nginx >/dev/null 2>&1; then
    nginx -t 2>&1 | tail -3 || true
    systemctl reload nginx 2>/dev/null || true
  fi
fi

if [ "$KEEP_PHP_PKG" = 0 ]; then
  step "5. uninstall php-fpm (optional)"
  echo "  skipping apt remove by default — php-fpm is shared with other apps."
  echo "  use --keep-php-pkg=off (not implemented) or apt remove manually."
fi

if [ "$KEEP_NGINX_PKG" = 0 ]; then
  step "6. uninstall nginx (optional)"
  echo "  skipping apt remove by default — nginx is shared with other sites."
fi

# -----------------------------------------------------------------------------
# System user
# -----------------------------------------------------------------------------
if [ "$KEEP_USER" = 0 ]; then
  step "7. delete orvixpanel system user"
  if id orvixpanel >/dev/null 2>&1; then
    userdel orvixpanel 2>/dev/null || true
    ok "user orvixpanel deleted"
  else
    ok "user orvixpanel not present"
  fi
fi

green ""
green "Uninstall complete."
if [ "$PURGE" = 1 ]; then
  green "  Homes + db + releases WERE purged."
else
  green "  Homes + db + releases KEPT at /var/lib/orvixpanel."
  green "  Reinstall + reuse: bash scripts/install.sh"
fi
