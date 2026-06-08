#!/bin/bash
# OrvixPanel Mail Module Local Smoke Test
# Tests mail components without requiring VPS/SMTP

set -e

echo "=========================================="
echo "MAIL MODULE LOCAL SMOKE TEST"
echo "=========================================="

cd /workspace

# Track results
PASS=0
FAIL=0

pass() {
    echo "  ✅ PASS: $1"
    ((PASS++))
}

fail() {
    echo "  ❌ FAIL: $1"
    ((FAIL++))
}

# 1. Check backend files exist
echo ""
echo "=== STEP 1: Backend Files ==="
FILES=(
    "internal/db/models/mail.go"
    "internal/mail/errors.go"
    "internal/mail/domain.go"
    "internal/mail/mailbox.go"
    "internal/mail/alias.go"
    "internal/mail/quota.go"
    "internal/mail/ratelimit.go"
    "internal/mail/audit.go"
    "internal/mail/config.go"
    "internal/mail/handlers.go"
    "internal/mail/installer.go"
    "internal/mail/mail_test.go"
)
for f in "${FILES[@]}"; do
    if [ -f "$f" ]; then
        pass "$f exists"
    else
        fail "$f missing"
    fi
done

# 2. Check frontend files exist
echo ""
echo "=== STEP 2: Frontend Files ==="
FRONTEND_FILES=(
    "frontend/src/lib/api/mail.ts"
    "frontend/src/pages/MailDomainsList.tsx"
    "frontend/src/pages/MailboxesList.tsx"
    "frontend/src/pages/MailAliases.tsx"
    "frontend/src/pages/MailForwarders.tsx"
    "frontend/src/pages/MailAuditLog.tsx"
    "frontend/src/pages/MailStats.tsx"
)
for f in "${FRONTEND_FILES[@]}"; do
    if [ -f "$f" ]; then
        pass "$f exists"
    else
        fail "$f missing"
    fi
done

# 3. Check router has mail routes
echo ""
echo "=== STEP 3: Router Configuration ==="
if grep -q "mailDomainsListRoute" frontend/src/router.tsx; then
    pass "Mail routes in router"
else
    fail "Mail routes missing from router"
fi

# 4. Check sidebar has mail link
echo ""
echo "=== STEP 4: Sidebar Navigation ==="
if grep -q '"/mail/domains"' frontend/src/lib/ui/Sidebar.tsx; then
    pass "Mail link in sidebar"
else
    fail "Mail link missing from sidebar"
fi

# 5. Go unit tests
echo ""
echo "=== STEP 5: Go Unit Tests ==="
if go test ./internal/mail/... -v 2>&1 | tee /tmp/mail_test.log; then
    pass "Mail unit tests passed"
else
    fail "Mail unit tests failed"
fi

# 6. Go build
echo ""
echo "=== STEP 6: Go Build ==="
if go build -o orvixpanel ./cmd/orvixpanel 2>&1; then
    pass "Go build successful"
    echo "  Binary size: $(du -h orvixpanel | cut -f1)"
else
    fail "Go build failed"
fi

# 7. Frontend typecheck
echo ""
echo "=== STEP 7: Frontend TypeScript ==="
cd frontend
if pnpm typecheck > /tmp/frontend_typecheck.log 2>&1; then
    pass "TypeScript typecheck passed"
else
    fail "TypeScript typecheck failed"
    cat /tmp/frontend_typecheck.log
fi
cd ..

# 8. Frontend build
echo ""
echo "=== STEP 8: Frontend Build ==="
if pnpm build > /tmp/frontend_build.log 2>&1; then
    pass "Frontend build successful"
else
    fail "Frontend build failed"
    tail -20 /tmp/frontend_build.log
fi

# Summary
echo ""
echo "=========================================="
echo "SMOKE TEST RESULTS"
echo "=========================================="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""

if [ $FAIL -eq 0 ]; then
    echo "✅ ALL CHECKS PASSED"
    echo ""
    echo "Mail module is ready for VPS integration testing."
    echo ""
    echo "To complete full mail testing, deploy to a VPS and run:"
    echo "  scripts/smoke-mail-vps.sh"
    exit 0
else
    echo "❌ SOME CHECKS FAILED"
    echo "Fix issues before proceeding."
    exit 1
fi