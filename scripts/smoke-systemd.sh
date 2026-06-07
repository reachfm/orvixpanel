#!/usr/bin/env bash
# OrvixPanel v0.2.2 systemd-mode smoke.
#
# Exercises the FULL systemd install path (no --skip-systemd):
#   1. systemctl is-system-running -> running
#   2. install.sh (writes /etc/systemd/system/orvixpanel.service)
#   3. systemctl is-active orvixpanel -> active
#   4. systemctl is-enabled orvixpanel -> enabled
#   5. smoke-phase2 -> 16/16
#   6. cleanup: stop, disable, remove unit
#
# This catches regressions where the systemd unit is broken but the
# --skip-systemd path still works. The two paths share only the env
# file and the binary; the unit file is install.sh's only and
# deserves a smoke on its own.
#
# Usage:
#   sudo bash scripts/smoke-systemd.sh
#
# Requires:
#   - WSL Ubuntu 22.04+ with [boot] systemd=true, OR a bare VPS
#     with systemd.
#   - /etc/wsl.conf must have [boot] systemd=true on WSL.
#     Run `wsl --shutdown` after editing.
set -uo pipefail

REPO="${REPO:-/mnt/d/orvixpanel}"
BIND_ADDR="${SYSTEMD_BIND:-127.0.0.1:28448}"
PASS=0
FAIL=0
pass() { echo "  PASS: $*"; PASS=$((PASS+1)); }
fail() { echo "  FAIL: $*"; FAIL=$((FAIL+1)); }
step() { printf '\n=== %s ===\n' "$*"; }

[ -d "$REPO" ] || { echo "repo not found: $REPO"; exit 1; }

# 0. systemd is actually running.
step "0. systemd available"
if ! command -v systemctl >/dev/null 2>&1; then
  fail "systemctl not on PATH"
  exit 1
fi
running=$(systemctl is-system-running 2>&1 || true)
if [ "$running" = "running" ]; then
  pass "systemctl is-system-running => running"
else
  fail "systemctl is-system-running => $running (need [boot] systemd=true in /etc/wsl.conf + wsl --shutdown)"
  exit 1
fi

# 1. clean up any previous orvixpanel unit
step "1. clean prior unit (if any)"
if [ -f /etc/systemd/system/orvixpanel.service ]; then
  systemctl stop    orvixpanel 2>/dev/null || true
  systemctl disable orvixpanel 2>/dev/null || true
  rm -f /etc/systemd/system/orvixpanel.service
  systemctl daemon-reload
  pass "removed stale unit"
else
  pass "no prior unit"
fi

# 2. uninstall (just in case install.sh was last run with --skip-systemd)
pkill -9 -f "orvixpanel/bin/orvixpanel" 2>/dev/null || true
bash "$REPO/scripts/uninstall.sh" --keep-user --keep-nginx --keep-php-pkg --yes \
  >/tmp/orvix-systemd-uninstall.log 2>&1 || true
pass "uninstall reset"

# 3. free port
port=$(echo "$BIND_ADDR" | cut -d: -f2)
if ss -tln "( sport = :$port )" 2>/dev/null | grep -q ":$port\b"; then
  fail "port $port is busy"
  exit 1
fi

# 4. install (full systemd path)
step "2. install (no --skip-systemd)"
bash "$REPO/scripts/install.sh" --bind "$BIND_ADDR" --yes \
  >/tmp/orvix-systemd-install.log 2>&1 \
  && pass "install.sh" \
  || { fail "install.sh failed; log: /tmp/orvix-systemd-install.log"; exit 1; }

# 5. unit file present
step "3. systemd unit state"
[ -f /etc/systemd/system/orvixpanel.service ] \
  && pass "/etc/systemd/system/orvixpanel.service present" \
  || fail "/etc/systemd/system/orvixpanel.service missing"

# 6. active
active=$(systemctl is-active orvixpanel 2>&1 || echo unknown)
if [ "$active" = "active" ]; then
  pass "systemctl is-active orvixpanel => active"
else
  fail "systemctl is-active orvixpanel => $active"
fi

# 7. enabled
enabled=$(systemctl is-enabled orvixpanel 2>&1 || echo unknown)
if [ "$enabled" = "enabled" ]; then
  pass "systemctl is-enabled orvixpanel => enabled"
else
  fail "systemctl is-enabled orvixpanel => $enabled"
fi

# 8. healthz
host=$(echo "$BIND_ADDR" | cut -d: -f1)
[ "$host" = "0.0.0.0" ] && host=127.0.0.1
if curl -fsS --max-time 3 "http://${host}:${port}/healthz" 2>/dev/null | grep -q '"status":"ok"'; then
  pass "healthz OK at http://${host}:${port}/healthz"
else
  fail "healthz not reachable at http://${host}:${port}/healthz"
fi

# 9. full smoke
step "4. smoke-phase2 (under systemd)"
out=$(BASE="http://${host}:${port}" bash "$REPO/smoke-phase2.sh" 2>&1) || true
if echo "$out" | grep -qE 'PHASE 2 SMOKE: ALL GATES PASSED'; then
  pass "smoke-phase2 16/16"
else
  fail "smoke-phase2 failed under systemd"
  echo "$out" | tail -10
fi

# 10. cleanup
step "5. cleanup"
systemctl stop    orvixpanel 2>/dev/null && pass "systemctl stop"    || fail "stop"
systemctl disable orvixpanel 2>/dev/null && pass "systemctl disable" || fail "disable"
rm -f /etc/systemd/system/orvixpanel.service
systemctl daemon-reload
pass "unit file removed"

# ---------------------------------------------------------------------------
step "summary"
printf "  %d PASS, %d FAIL\n" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
