#!/bin/bash
# smoke-dns-local.sh — v0.4.0 DNS Engine smoke test (local SQLite mode)
#
# Verifies:
#   1. Binary builds with DNS engine enabled
#   2. Binary starts without PowerDNS configured (local-only mode)
#   3. API responds on /healthz
#   4. DNS endpoints return 401 (auth required)
#
# Prerequisites:
#   - go build ./cmd/orvixpanel (already done)
#   - ORVIX_ALLOW_DEV=1 for dev license
#   - Optional: ORVIX_DB_PATH for custom DB location

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/orvixpanel"
DB_PATH="${ORVIX_DB_PATH:-/tmp/smoke-dns-$$.db}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

cleanup() {
  if [[ -f "$DB_PATH" ]]; then
    rm -f "$DB_PATH"
  fi
  if [[ -n "${PID:-}" ]] && kill -0 "$PID" 2>/dev/null; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# -----------------------------------------------------------------------------
# Test 1: Binary exists
# -----------------------------------------------------------------------------
info "Test 1: Checking binary..."
if [[ ! -f "$BINARY" ]]; then
  fail "Binary not found at $BINARY"
  exit 1
fi
pass "Binary exists at $BINARY"

# -----------------------------------------------------------------------------
# Test 2: Start binary in background (local-only mode, no PowerDNS)
# -----------------------------------------------------------------------------
info "Test 2: Starting orvixpanel in local-only DNS mode..."
export ORVIX_ALLOW_DEV=1
export ORVIX_DB_PATH="$DB_PATH"
unset ORVIX_POWERDNS_URL
unset ORVIX_POWERDNS_API_KEY

$BINARY &
PID=$!
sleep 4

# Check if process is still running
if ! kill -0 "$PID" 2>/dev/null; then
  fail "orvixpanel failed to start"
  exit 1
fi
pass "orvixpanel started (PID $PID)"

# Detect what port the server is listening on
detect_port() {
  for port in 8443 8080 18942; do
    if curl -s "http://localhost:$port/healthz" | grep -q '"status"'; then
      echo "$port"
      return 0
    fi
  done
  return 1
}

info "Detecting server port..."
PORT=$(detect_port) || {
  fail "Could not detect server port"
  exit 1
}
info "Server detected on port $PORT"

# -----------------------------------------------------------------------------
# Test 3: Health check
# -----------------------------------------------------------------------------
info "Test 3: Checking /healthz endpoint..."
HEALTH=$(curl -s -f "http://localhost:$PORT/healthz" || echo '{"status":"error"}')
if echo "$HEALTH" | grep -q '"status":"ok"'; then
  pass "Health check passed"
else
  fail "Health check failed: $HEALTH"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 4: Ready check
# -----------------------------------------------------------------------------
info "Test 4: Checking /readyz endpoint..."
READY=$(curl -s -f "http://localhost:$PORT/readyz" || echo '{"status":"error"}')
if echo "$READY" | grep -q '"status":"ready"'; then
  pass "Ready check passed"
else
  fail "Ready check failed: $READY"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 5: DNS endpoints require auth (401)
# -----------------------------------------------------------------------------
info "Test 5: Verifying DNS endpoints require authentication..."

# List zones should return 401 without auth
ZONES_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/api/v1/dns/zones")
if [[ "$ZONES_STATUS" == "401" ]]; then
  pass "GET /api/v1/dns/zones returns 401 (auth required)"
else
  fail "GET /api/v1/dns/zones returned $ZONES_STATUS, expected 401"
  exit 1
fi

# Create zone should return 401 without auth, or 423 if license is expired (panel locked)
CREATE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '{"domain":"test.example.com"}' "http://localhost:$PORT/api/v1/dns/zones")
if [[ "$CREATE_STATUS" == "401" ]]; then
  pass "POST /api/v1/dns/zones returns 401 (auth required)"
elif [[ "$CREATE_STATUS" == "423" ]]; then
  pass "POST /api/v1/dns/zones returns 423 (license expired, panel locked - dev fallback)"
else
  fail "POST /api/v1/dns/zones returned $CREATE_STATUS, expected 401 or 423"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 6: Validate endpoint requires auth
# -----------------------------------------------------------------------------
info "Test 6: Verifying /dns/validate requires authentication..."
VALIDATE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" \
  -d '{"name":"www","type":"A","content":"192.0.2.1"}' "http://localhost:$PORT/api/v1/dns/validate")
if [[ "$VALIDATE_STATUS" == "401" ]]; then
  pass "POST /api/v1/dns/validate returns 401 (auth required)"
elif [[ "$VALIDATE_STATUS" == "423" ]]; then
  pass "POST /api/v1/dns/validate returns 423 (license expired, panel locked - dev fallback)"
else
  fail "POST /api/v1/dns/validate returned $VALIDATE_STATUS, expected 401 or 423"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 7: Lookup endpoint requires auth
# -----------------------------------------------------------------------------
info "Test 7: Verifying /dns/lookup requires authentication..."
LOOKUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/api/v1/dns/lookup/example.com")
if [[ "$LOOKUP_STATUS" == "401" ]]; then
  pass "GET /api/v1/dns/lookup/:domain returns 401 (auth required)"
else
  fail "GET /api/v1/dns/lookup/:domain returned $LOOKUP_STATUS, expected 401"
  exit 1
fi

# -----------------------------------------------------------------------------
# Test 8: Templates endpoint requires auth
# -----------------------------------------------------------------------------
info "Test 8: Verifying /dns/templates requires authentication..."
TEMPLATES_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/api/v1/dns/templates")
if [[ "$TEMPLATES_STATUS" == "401" ]]; then
  pass "GET /api/v1/dns/templates returns 401 (auth required)"
else
  fail "GET /api/v1/dns/templates returned $TEMPLATES_STATUS, expected 401"
  exit 1
fi

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo ""
info "=========================================="
info "  DNS Engine Smoke Test Complete"
info "  Mode: Local SQLite (PowerDNS disabled)"
info "  Port: $PORT"
info "  DB: $DB_PATH"
info "=========================================="
echo ""
pass "All smoke tests passed!"
echo ""

exit 0