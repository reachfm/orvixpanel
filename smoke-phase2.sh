#!/bin/bash
# OrvixPanel Phase 2 Core Hosting Engine — smoke test.
# Runs inside WSL Ubuntu 26.04. Exits non-zero on any failure.

set -uo pipefail

BASE="${BASE:-http://127.0.0.1:28444}"
PASSWORD="${PASSWORD:-U8gvIecWdzSpWa50}"
TEST_USER="testuser"
TEST_DOMAIN="test.local"
TEST_PORT="${TEST_PORT:-8080}"

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
blue()  { printf '\033[34m%s\033[0m\n' "$*"; }

step() { blue ""; blue "=== $* ==="; }
ok()   { green "PASS: $*"; }
fail() { red   "FAIL: $*"; exit 1; }

require() {
  local cmd="$1"
  command -v "$cmd" >/dev/null 2>&1 || fail "missing command: $cmd"
}

require curl
require python3
require nginx
require php-fpm8.5
require id
require getent
require systemctl
require nginx

# -----------------------------------------------------------------------------
# 0. Pre-flight
# -----------------------------------------------------------------------------
step "0. preflight"
curl -fsS "$BASE/healthz" >/dev/null || fail "binary not reachable at $BASE"
ok "binary up"

# 0a. Clean previous testuser state so the script is idempotent.
PASSWORD="${PASSWORD:-U8gvIecWdzSpWa50}"
LOGIN_JSON=$(curl -fsS -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"admin@orvixpanel.local\",\"password\":\"$PASSWORD\"}")
TOKEN=$(echo "$LOGIN_JSON" | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])')

# Drop the system user directly if it exists. This is more
# reliable than going through the API (which needs the account
# ID, and the ID is in a soft-deleted row).
if id "$TEST_USER" >/dev/null 2>&1; then
  # userdel without -r: we want the home dir gone too, but the
  # API is supposed to have done that. Fall back to -r if the
  # home still exists.
  pkill -u "$TEST_USER" 2>/dev/null || true
  if [ -d "/var/lib/orvixpanel/homes/$TEST_USER" ]; then
    userdel -r "$TEST_USER" 2>/dev/null || true
  else
    userdel "$TEST_USER" 2>/dev/null || true
  fi
fi
# Hard-clear the DB row using python (avoids the soft-delete trap).
if [ -f /var/lib/orvixpanel/data.db ]; then
  python3 -c "
import sqlite3
c = sqlite3.connect('/var/lib/orvixpanel/data.db')
c.execute(\"DELETE FROM accounts WHERE username='$TEST_USER' OR domain='$TEST_DOMAIN'\")
c.commit()
c.close()
" 2>/dev/null || true
fi
# Drop any stale config files from a previous run.
rm -f "/etc/nginx/conf.d/orvix/$TEST_USER-$TEST_DOMAIN.conf" 2>/dev/null || true
rm -f "/etc/php/8.5/fpm/pool.d/orvix-$TEST_USER-$TEST_DOMAIN.conf" 2>/dev/null || true
rm -rf "/var/lib/orvixpanel/homes/$TEST_USER" 2>/dev/null || true
systemctl reload nginx >/dev/null 2>&1 || true
systemctl reload php8.5-fpm >/dev/null 2>&1 || true
ok "preflight clean: testuser state reset"

# -----------------------------------------------------------------------------
# 1. Login (already done above; just re-confirm)
# -----------------------------------------------------------------------------
step "1. login"
[ ${#TOKEN} -gt 50 ] || fail "no access_token"
ok "got access_token (len=${#TOKEN})"

# -----------------------------------------------------------------------------
# 2. Create account
# -----------------------------------------------------------------------------
step "2. create account (user=$TEST_USER)"
ACCT_JSON=$(curl -fsS -X POST "$BASE/api/v1/accounts" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$TEST_USER\",\"domain\":\"$TEST_DOMAIN\",\"plan\":\"basic\"}")
ACCT_ID=$(echo "$ACCT_JSON" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("id",""))')
[ -n "$ACCT_ID" ] || fail "no account id in response: $ACCT_JSON"
ok "account created id=$ACCT_ID"

# -----------------------------------------------------------------------------
# 3. id testuser
# -----------------------------------------------------------------------------
step "3. id $TEST_USER"
id_out=$(id "$TEST_USER" 2>&1) || fail "id $TEST_USER failed: $id_out"
echo "  $id_out"
ok "id $TEST_USER works"

# -----------------------------------------------------------------------------
# 4. getent passwd testuser
# -----------------------------------------------------------------------------
step "4. getent passwd $TEST_USER"
gp_out=$(getent passwd "$TEST_USER" 2>&1) || fail "getent failed: $gp_out"
echo "  $gp_out"
ok "getent passwd works"

# -----------------------------------------------------------------------------
# 5. ls -la /var/lib/orvixpanel/homes/testuser
# -----------------------------------------------------------------------------
step "5. ls -la /var/lib/orvixpanel/homes/$TEST_USER"
home_dir="/var/lib/orvixpanel/homes/$TEST_USER"
ls_out=$(ls -la "$home_dir" 2>&1) || fail "ls failed: $ls_out"
echo "$ls_out"
echo "$ls_out" | grep -q "public_html" || fail "public_html not in home"
ok "home dir + public_html exist"

# -----------------------------------------------------------------------------
# 6. Create domain (nginx vhost + php-fpm pool + reload)
# -----------------------------------------------------------------------------
step "6. create domain (name=$TEST_DOMAIN, port=$TEST_PORT)"
DOM_JSON=$(curl -fsS -X POST "$BASE/api/v1/accounts/$ACCT_ID/domains" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"domain\":\"$TEST_DOMAIN\",\"port\":$TEST_PORT}")
echo "  $DOM_JSON"
ok "domain created"

# -----------------------------------------------------------------------------
# 7. nginx -t
# -----------------------------------------------------------------------------
step "7. nginx -t"
nginx_t_out=$(nginx -t 2>&1) || fail "nginx -t failed: $nginx_t_out"
echo "$nginx_t_out"
ok "nginx -t passes"

# -----------------------------------------------------------------------------
# 8. systemctl reload nginx
# -----------------------------------------------------------------------------
step "8. systemctl reload nginx"
systemctl reload nginx && ok "nginx reloaded" || fail "nginx reload failed"

# -----------------------------------------------------------------------------
# 9. systemctl status nginx
# -----------------------------------------------------------------------------
step "9. systemctl status nginx (active check)"
nginx_status=$(systemctl is-active nginx 2>&1) || true
echo "  is-active: $nginx_status"
[ "$nginx_status" = "active" ] || fail "nginx not active: $nginx_status"
ok "nginx active"

# -----------------------------------------------------------------------------
# 10. systemctl status php-fpm
# -----------------------------------------------------------------------------
step "10. systemctl status php8.5-fpm (active check)"
fpm_status=$(systemctl is-active php8.5-fpm 2>&1) || true
echo "  is-active: $fpm_status"
[ "$fpm_status" = "active" ] || fail "php-fpm not active: $fpm_status"
ok "php-fpm active"

# -----------------------------------------------------------------------------
# 11. vhost file exists
# -----------------------------------------------------------------------------
step "11. vhost file on disk"
vhost="/etc/nginx/conf.d/orvix/$TEST_USER-$TEST_DOMAIN.conf"
[ -f "$vhost" ] || fail "vhost not found: $vhost"
echo "  $(ls -la "$vhost")"
echo "  ---"
sed -n '1,30p' "$vhost"
ok "vhost exists and rendered"

# -----------------------------------------------------------------------------
# 12. php-fpm pool file exists
# -----------------------------------------------------------------------------
step "12. php-fpm pool file on disk"
pool="/etc/php/8.5/fpm/pool.d/orvix-$TEST_USER-$TEST_DOMAIN.conf"
[ -f "$pool" ] || fail "pool not found: $pool"
echo "  $(ls -la "$pool")"
echo "  ---"
sed -n '1,20p' "$pool"
ok "pool exists and rendered"

# -----------------------------------------------------------------------------
# 13. curl http://127.0.0.1:8080  →  HTTP 200
# -----------------------------------------------------------------------------
step "13. curl -H 'Host: $TEST_DOMAIN' http://127.0.0.1:$TEST_PORT"
http_out=$(curl -sS -i -H "Host: $TEST_DOMAIN" "http://127.0.0.1:$TEST_PORT/" 2>&1) || fail "curl failed: $http_out"
echo "$http_out" | head -10
status_line=$(echo "$http_out" | head -1)
echo "$status_line" | grep -q "200" || fail "expected HTTP 200, got: $status_line"
ok "HTTP 200 returned"

# -----------------------------------------------------------------------------
# 14. curl http://127.0.0.1:8080/info.php  →  HTTP 200 + PHP-FPM alive
# -----------------------------------------------------------------------------
step "14. curl -H 'Host: $TEST_DOMAIN' http://127.0.0.1:$TEST_PORT/info.php"
php_out=$(curl -sS -H "Host: $TEST_DOMAIN" "http://127.0.0.1:$TEST_PORT/info.php" 2>&1) || fail "php curl failed: $php_out"
echo "$php_out"
echo "$php_out" | grep -q "OrvixPanel PHP-FPM alive" || fail "info.php not served via PHP-FPM"
ok "PHP-FPM is serving info.php"

# -----------------------------------------------------------------------------
# 15. delete domain
# -----------------------------------------------------------------------------
step "15. delete domain (DELETE /accounts/$ACCT_ID/domains/$TEST_DOMAIN)"
curl -fsS -X DELETE -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/accounts/$ACCT_ID/domains/$TEST_DOMAIN" || fail "delete domain failed"
ok "domain deleted"

# -----------------------------------------------------------------------------
# 16. delete account
# -----------------------------------------------------------------------------
step "16. delete account (DELETE /accounts/$ACCT_ID)"
curl -fsS -X DELETE -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/accounts/$ACCT_ID" || fail "delete account failed"
id_post=$(id "$TEST_USER" 2>&1) || true
echo "  $id_post"
echo "$id_post" | grep -q "no such user" || fail "user still present: $id_post"
ok "account deleted, system user removed"

green ""
green "================================================="
green "PHASE 2 SMOKE: ALL GATES PASSED"
green "================================================="
