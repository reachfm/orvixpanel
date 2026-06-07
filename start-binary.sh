#!/bin/bash
# start-orvixpanel.sh — robust daemonizer.
# 1. forks via python3 (so the parent can exit cleanly)
# 2. the python child re-execs the binary
# 3. writes pid to /var/run/orvixpanel.pid
set -e

BIN="/mnt/d/orvixpanel/bin/orvixpanel.linux"
LOG="/var/log/orvixpanel/binary.out"
PIDFILE="/var/run/orvixpanel.pid"

# kill old
if [ -f "$PIDFILE" ]; then
  kill "$(cat $PIDFILE)" 2>/dev/null || true
  sleep 1
fi
pgrep -f 'orvixpanel/bin/orvixpanel\|orvixpanel.linux' | xargs -r kill 2>/dev/null || true
sleep 1

mkdir -p /var/log/orvixpanel /var/run

export ORVIX_DEV_LICENSE_EXPIRES_AT=2030-01-01
export ORVIX_ALLOW_DEV=1
export ORVIX_MASTER_KEY=MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI
export ORVIX_DATABASE_DSN=/var/lib/orvixpanel/data.db
export ORVIX_SERVER_BIND_ADDR=127.0.0.1:28444
export ORVIX_LICENSE_KEY=ORVIX-ENTERPRISE-2025-DEV01-DEVPLACE
export ORVIX_ALLOW_LOCAL_TLD=1
export ORVIX_FPM_VERSION=8.5

# Fork via python: parent exits 0 immediately, child re-execs.
python3 -c "
import os, sys
pid = os.fork()
if pid > 0:
    # parent
    with open('$PIDFILE', 'w') as f:
        f.write(str(pid))
    sys.exit(0)
# child: detach
os.setsid()
os.umask(0)
os.close(0); os.close(1); os.close(2)
fd = os.open('$LOG', os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o644)
os.dup2(fd, 1); os.dup2(fd, 2)
os.execvpe('$BIN', ['$BIN'], dict(os.environ))
"

sleep 4

# healthz
for i in 1 2 3 4 5; do
  if curl -fsS http://127.0.0.1:28444/healthz >/dev/null 2>&1; then
    echo "binary up on attempt $i (pid=$(cat $PIDFILE 2>/dev/null || echo unknown))"
    exit 0
  fi
  sleep 1
done
echo "binary did not start in 10s"
tail -30 "$LOG"
exit 1