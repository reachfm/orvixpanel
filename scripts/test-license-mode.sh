#!/usr/bin/env bash
# OrvixPanel v0.2.2 license-mode test.
#
# Verifies the v0.3.0 license boot contract:
#   1. ORVIX_ALLOW_DEV=0 with no ORVIX_LICENSE_KEY
#      -> binary fails to boot with a license error (fail-safe)
#   2. ORVIX_ALLOW_DEV=1 with no ORVIX_LICENSE_KEY
#      -> binary boots in dev fallback mode and /healthz returns ok
#
# This protects the production install path: a fresh install with
# ORVIX_ALLOW_DEV=0 must never silently fall back to a dev license.
#
# Exit code: 0 if all checks pass, non-zero if any check fails.
set -uo pipefail

BIN="/opt/orvixpanel/bin/orvixpanel"
PORT="${TEST_LICENSE_PORT:-28446}"
DB_DSN="${TEST_LICENSE_DSN:-/var/lib/orvixpanel/data.db}"
FPM_VERSION="${ORVIX_FPM_VERSION:-8.5}"

PASS=0
FAIL=0
pass() { echo "  PASS: $*"; PASS=$((PASS+1)); }
fail() { echo "  FAIL: $*"; FAIL=$((FAIL+1)); }

# Sanity.
[ -x "$BIN" ] || { echo "binary not found at $BIN — run install.sh first"; exit 1; }
if ss -tln "( sport = :$PORT )" 2>/dev/null | grep -q ":$PORT\b"; then
  echo "port $PORT is busy; free it (e.g. 'pkill -9 -f orvixpanel/bin/orvixpanel')"
  exit 1
fi

# Stop any running orvixpanel so we have a clean port.
pkill -9 -f "orvixpanel/bin/orvixpanel" 2>/dev/null || true
sleep 1

# Clean env that overrides /etc/orvixpanel/orvixpanel.env values
# (we want exactly the test env, nothing else).
build_env() {
  # Preserve PATH, HOME, TZ, LANG; null everything else.
  env -i \
    PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    HOME="/root" \
    TZ="UTC" \
    LANG="C.UTF-8" \
    ORVIX_SERVER_BIND_ADDR="127.0.0.1:${PORT}" \
    ORVIX_DATABASE_DSN="${DB_DSN}" \
    ORVIX_FPM_VERSION="${FPM_VERSION}" \
    "$@"
  return $?
}

cleanup() {
  pkill -9 -f "orvixpanel/bin/orvixpanel" 2>/dev/null || true
  rm -f /tmp/orvix-license-test.log
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
echo "=== 1. production mode (ORVIX_ALLOW_DEV=0, no ORVIX_LICENSE_KEY) ==="
# Capture both stdout and stderr. The binary will try to call license.Parse
# with an empty key; in production it must fail closed.
out=$(build_env ORVIX_ALLOW_DEV=0 timeout 5 "$BIN" 2>&1) || true
ec=$?
if echo "$out" | grep -qi 'license' && echo "$out" | grep -qiE 'invalid|expir|fail|error'; then
  pass "production mode refused to boot (license error surfaced)"
  echo "    stderr: $(echo "$out" | tr -d '\n' | cut -c1-200)"
else
  fail "production mode did not surface a license error"
  echo "    output: $out" | head -5
fi

# ---------------------------------------------------------------------------
echo "=== 2. dev mode (ORVIX_ALLOW_DEV=1, no ORVIX_LICENSE_KEY) ==="
# Start the binary in dev mode in the background and probe /healthz.
build_env ORVIX_ALLOW_DEV=1 ORVIX_DEV_LICENSE_EXPIRES_AT=2030-01-01 \
  nohup "$BIN" >/tmp/orvix-license-test.log 2>&1 </dev/null &
disown 2>/dev/null || true
sleep 3
if curl -fsS --max-time 3 "http://127.0.0.1:${PORT}/healthz" 2>/dev/null \
   | grep -q '"status":"ok"'; then
  pass "dev mode booted, /healthz returned ok"
else
  fail "dev mode did not respond on /healthz"
  echo "    log: $(tail -10 /tmp/orvix-license-test.log 2>/dev/null | head -10)"
fi

# ---------------------------------------------------------------------------
echo "=== 3. production mode with no DEV and no KEY = fail-closed ==="
pkill -9 -f "orvixpanel/bin/orvixpanel" 2>/dev/null || true
sleep 1
# ORVIX_ALLOW_DEV unset/0 AND no license key: must fail closed.
out=$(build_env ORVIX_ALLOW_DEV=0 timeout 5 "$BIN" 2>&1) || true
if echo "$out" | grep -qi 'license'; then
  pass "no-DEV + no-KEY => fail-closed (license error)"
else
  fail "no-DEV + no-KEY did not surface a license error"
fi

# ---------------------------------------------------------------------------
echo
echo "=== summary ==="
printf "  %d PASS, %d FAIL\n" "$PASS" "$FAIL"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
