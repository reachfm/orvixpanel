#!/usr/bin/env bash
# OrvixPanel v0.2.2 port-collision behavior test.
#
# Verifies install.sh's preflight refuses with a clear error if the
# bind port is already in use, before doing any apt installs or
# service starts.
set -uo pipefail
REPO="/mnt/d/orvixpanel"
PORT=28449
PASS=0
FAIL=0
pass() { echo "  PASS: $*"; PASS=$((PASS+1)); }
fail() { echo "  FAIL: $*"; FAIL=$((FAIL+1)); }

# 0. free the port
if ss -tln "( sport = :$PORT )" 2>/dev/null | grep -q ":$PORT\b"; then
  echo "port $PORT is busy; free it first"
  exit 1
fi

# 1. occupy the port with a python listener
python3 -c '
import socket, time
s = socket.socket()
s.bind(("127.0.0.1", '$PORT'))
s.listen(1)
time.sleep(120)
' >/dev/null 2>&1 &
LISTENER_PID=$!
sleep 1

# verify it's actually busy
if ! ss -tln "( sport = :$PORT )" 2>/dev/null | grep -q ":$PORT\b"; then
  kill $LISTENER_PID 2>/dev/null
  echo "failed to occupy port $PORT; aborting"
  exit 1
fi

# 2. run install.sh with the busy port
echo "=== install.sh with busy port ==="
# Capture exit code BEFORE the `|| true` masks it.
set +e
bash "$REPO/scripts/install.sh" --bind "127.0.0.1:$PORT" --skip-systemd --yes >/tmp/orvix-pc-out.log 2>&1
ec=$?
set -e
out=$(cat /tmp/orvix-pc-out.log)

# expect non-zero exit
if [ "$ec" -ne 0 ]; then
  pass "install.sh exited non-zero ($ec)"
else
  fail "install.sh exited 0 with busy port (expected non-zero)"
fi

# expect a clear error message naming the port
if echo "$out" | grep -q "port $PORT" && echo "$out" | grep -qi "in use\|already\|busy"; then
  pass "clear error message naming port $PORT"
  echo "    message: $(echo "$out" | grep -iE 'port|fail|busy' | head -2 | tr '\n' ' ')"
else
  fail "error message did not clearly name port $PORT"
  echo "    output:"
  echo "$out" | head -5 | sed 's/^/      /'
fi

# expect install.sh did NOT install apt packages (fail fast in preflight)
# (heuristic: the output should not contain "OK: nginx" or "OK: php")
if echo "$out" | grep -qE "OK: nginx|OK: php[0-9.]+"; then
  fail "install.sh proceeded past preflight before failing"
else
  pass "install.sh failed fast (preflight, no apt installs)"
fi

# 3. free the port
kill $LISTENER_PID 2>/dev/null
wait 2>/dev/null
sleep 1

# verify the port is free again
if ss -tln "( sport = :$PORT )" 2>/dev/null | grep -q ":$PORT\b"; then
  fail "port $PORT still busy after listener killed"
else
  pass "port $PORT freed"
fi

# 4. install.sh should now succeed (with the now-free port)
# We use --no-start so it doesn't actually try to start a binary on top
# of the already-running 28445; we just verify the port precheck passes.
echo "=== install.sh after port freed ==="
set +e
bash "$REPO/scripts/install.sh" --bind "127.0.0.1:$PORT" --skip-systemd --no-start --yes >/tmp/orvix-pc-out2.log 2>&1
ec=$?
set -e
out=$(cat /tmp/orvix-pc-out2.log)
if [ "$ec" -eq 0 ]; then
  pass "install.sh succeeded with port free (--no-start)"
else
  fail "install.sh failed with port free (exit $ec)"
  echo "    output:"
  echo "$out" | tail -10 | sed 's/^/      /'
fi

# clean up: uninstall the test install
bash "$REPO/scripts/uninstall.sh" --keep-user --keep-nginx --keep-php-pkg --yes \
  >/dev/null 2>&1 || true

echo
echo "=== summary ==="
printf "  %d PASS, %d FAIL\n" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
