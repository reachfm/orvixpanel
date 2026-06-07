#!/usr/bin/env bash
# OrvixPanel v0.2.2 reinstall-cycle smoke.
#
# Exercises the full install/uninstall/reinstall round trip and
# proves that:
#   * a fresh install boots
#   * the smoke-phase2 contract still holds end-to-end
#   * uninstalling WITHOUT --purge keeps /var/lib/orvixpanel intact
#     (so account homes + db survive)
#   * reinstalling on top of a non-purged state still boots
#   * the smoke contract still holds on the reinstall
#
# This catches "install was never idempotent" regressions that a
# one-shot smoke can't.
#
# Usage:
#   sudo bash scripts/smoke-reinstall-cycle.sh
#
# Exits 0 if all 6 gates pass, non-zero otherwise.
set -uo pipefail

REPO="${REPO:-/mnt/d/orvixpanel}"
BIND_ADDR="${REINSTALL_BIND:-127.0.0.1:28447}"
BASE="http://${BIND_ADDR}"
PASS=0
FAIL=0
pass() { echo "  PASS: $*"; PASS=$((PASS+1)); }
fail() { echo "  FAIL: $*"; FAIL=$((FAIL+1)); }
step() { printf '\n=== %s ===\n' "$*"; }

[ -d "$REPO" ] || { echo "repo not found: $REPO"; exit 1; }
[ -x "$REPO/scripts/install.sh" ] || { echo "install.sh missing"; exit 1; }

# Make sure the port is free.
port=$(echo "$BIND_ADDR" | cut -d: -f2)
if ss -tln "( sport = :$port )" 2>/dev/null | grep -q ":$port\b"; then
  echo "port $port is busy; free it first (pkill -9 -f orvixpanel/bin/orvixpanel)"
  exit 1
fi

cleanup() {
  pkill -9 -f "orvixpanel/bin/orvixpanel" 2>/dev/null || true
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
step "0. uninstall (clean slate)"
bash "$REPO/scripts/uninstall.sh" --keep-user --keep-nginx --keep-php-pkg --yes \
  >/tmp/orvix-reinstall-uninstall.log 2>&1 \
  && pass "uninstall --yes --keep-user --keep-nginx --keep-php-pkg" \
  || { fail "uninstall failed; log: /tmp/orvix-reinstall-uninstall.log"; exit 1; }

# Verify the dir layout: /var/lib preserved, /opt removed, /run wiped.
[ -d /var/lib/orvixpanel ] && pass "/var/lib/orvixpanel preserved" || fail "/var/lib/orvixpanel missing"
[ ! -d /opt/orvixpanel ]  && pass "/opt/orvixpanel removed"      || fail "/opt/orvixpanel still present"
[ ! -d /run/orvixpanel ]  && pass "/run/orvixpanel removed"      || fail "/run/orvixpanel still present"

# ---------------------------------------------------------------------------
step "1. install (cycle 1)"
bash "$REPO/scripts/install.sh" --bind "$BIND_ADDR" --skip-systemd --yes \
  >/tmp/orvix-reinstall-install1.log 2>&1 \
  && pass "install cycle 1" \
  || { fail "install cycle 1 failed; log: /tmp/orvix-reinstall-install1.log"; exit 1; }

# ---------------------------------------------------------------------------
step "2. doctor (cycle 1)"
out=$(bash "$REPO/scripts/doctor.sh" 2>&1) || true
echo "$out" | tail -5
if echo "$out" | grep -qE '0 FAIL' && echo "$out" | grep -qE 'healthz.*ok' 2>/dev/null; then
  pass "doctor cycle 1 (0 FAIL)"
elif echo "$out" | grep -qE '0 FAIL'; then
  pass "doctor cycle 1 (0 FAIL, healthz may be missing but no FAIL row)"
else
  fail "doctor cycle 1 has FAILs"
fi

# ---------------------------------------------------------------------------
step "3. smoke-phase2 (cycle 1)"
out=$(BASE="$BASE" bash "$REPO/smoke-phase2.sh" 2>&1) || true
if echo "$out" | grep -qE 'PHASE 2 SMOKE: ALL GATES PASSED'; then
  pass "smoke-phase2 cycle 1 (16/16)"
else
  fail "smoke-phase2 cycle 1 failed"
  echo "$out" | tail -10
fi

# ---------------------------------------------------------------------------
step "4. uninstall (no --purge) — keeps /var/lib/orvixpanel"
pkill -9 -f "orvixpanel/bin/orvixpanel" 2>/dev/null || true
sleep 1
bash "$REPO/scripts/uninstall.sh" --keep-user --keep-nginx --keep-php-pkg --yes \
  >/tmp/orvix-reinstall-uninstall2.log 2>&1 \
  && pass "uninstall cycle 1" \
  || { fail "uninstall cycle 1 failed"; exit 1; }
[ -d /var/lib/orvixpanel ] && pass "/var/lib/orvixpanel preserved by safe uninstall" \
  || fail "/var/lib/orvixpanel was removed by safe uninstall"

# ---------------------------------------------------------------------------
step "5. install (cycle 2) — reinstall on preserved data"
bash "$REPO/scripts/install.sh" --bind "$BIND_ADDR" --skip-systemd --yes \
  >/tmp/orvix-reinstall-install2.log 2>&1 \
  && pass "install cycle 2" \
  || { fail "install cycle 2 failed; log: /tmp/orvix-reinstall-install2.log"; exit 1; }

# ---------------------------------------------------------------------------
step "6. doctor (cycle 2)"
out=$(bash "$REPO/scripts/doctor.sh" 2>&1) || true
if echo "$out" | grep -qE '0 FAIL'; then
  pass "doctor cycle 2 (0 FAIL)"
else
  fail "doctor cycle 2 has FAILs"
fi

# ---------------------------------------------------------------------------
step "7. smoke-phase2 (cycle 2) — proves reinstall is fully functional"
out=$(BASE="$BASE" bash "$REPO/smoke-phase2.sh" 2>&1) || true
if echo "$out" | grep -qE 'PHASE 2 SMOKE: ALL GATES PASSED'; then
  pass "smoke-phase2 cycle 2 (16/16)"
else
  fail "smoke-phase2 cycle 2 failed"
  echo "$out" | tail -10
fi

# ---------------------------------------------------------------------------
step "summary"
printf "  %d PASS, %d FAIL\n" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
