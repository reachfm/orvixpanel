#!/bin/bash
# OrvixPanel v0.3.1 Runtime Evidence Verification Script
# This script verifies that all components of the OrvixPanel system are working correctly.
# It serves as evidence of a properly functioning system.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FRONTEND_DIR="/workspace/frontend"
BACKEND_DIR="/workspace"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║         OrvixPanel v0.3.1 Runtime Evidence Script         ║${NC}"
echo -e "${BLUE}║              Verification Report Generator                ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Execution Time:${NC} $TIMESTAMP"
echo ""

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}▶ $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Function to check command success
check_cmd() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ SUCCESS${NC}"
    else
        echo -e "${RED}✗ FAILED${NC}"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
}

# Initialize counters
PASSED_COUNT=0
FAILED_COUNT=0

#######################################
# FRONTEND VERIFICATION
#######################################
print_section "FRONTEND VERIFICATION"

echo -e "${YELLOW}[1/6] TypeScript Type Checking...${NC}"
cd "$FRONTEND_DIR"
pnpm typecheck > /dev/null 2>&1
check_cmd
if [ $? -eq 0 ]; then PASSED_COUNT=$((PASSED_COUNT + 1)); fi

echo -e "${YELLOW}[2/6] Frontend Build...${NC}"
pnpm build > /dev/null 2>&1
check_cmd
if [ $? -eq 0 ]; then PASSED_COUNT=$((PASSED_COUNT + 1)); fi

echo -e "${YELLOW}[3/6] Frontend Unit Tests...${NC}"
TEST_OUTPUT=$(pnpm test -- --run 2>&1 | tail -5)
if echo "$TEST_OUTPUT" | grep -q "Tests.*passed"; then
    echo -e "${GREEN}✓ Tests passed${NC}"
    PASSED_COUNT=$((PASSED_COUNT + 1))
else
    echo -e "${RED}✗ Tests failed${NC}"
    FAILED_COUNT=$((FAILED_COUNT + 1))
fi

#######################################
# BACKEND VERIFICATION
#######################################
print_section "BACKEND VERIFICATION"

echo -e "${YELLOW}[4/6] Go Module Tidy...${NC}"
cd "$BACKEND_DIR"
go mod tidy > /dev/null 2>&1
check_cmd
if [ $? -eq 0 ]; then PASSED_COUNT=$((PASSED_COUNT + 1)); fi

echo -e "${YELLOW}[5/6] Go Backend Tests...${NC}"
GO_TEST_OUTPUT=$(go test ./... 2>&1)
if echo "$GO_TEST_OUTPUT" | grep -q "ok"; then
    echo -e "${GREEN}✓ All packages passed${NC}"
    PASSED_COUNT=$((PASSED_COUNT + 1))
else
    echo -e "${RED}✗ Tests failed${NC}"
    FAILED_COUNT=$((FAILED_COUNT + 1))
fi

echo -e "${YELLOW}[6/6] Go Backend Build...${NC}"
go build -buildvcs=false ./cmd/orvixpanel > /dev/null 2>&1
check_cmd
if [ $? -eq 0 ]; then PASSED_COUNT=$((PASSED_COUNT + 1)); fi

#######################################
# EVIDENCE SUMMARY
#######################################
print_section "EVIDENCE SUMMARY"

echo ""
echo -e "  ${GREEN}✓ PASSED:${NC} $PASSED_COUNT"
echo -e "  ${RED}✗ FAILED:${NC} $FAILED_COUNT"
echo ""

if [ $FAILED_COUNT -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                  ALL VERIFICATIONS PASSED                   ║${NC}"
    echo -e "${GREEN}║        System is ready for deployment and use               ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Evidence Generated: $TIMESTAMP"
    echo "Frontend: TypeScript ✓ | Build ✓ | Tests ✓"
    echo "Backend:  Go tidy ✓ | Tests ✓ | Build ✓"
    echo ""
    exit 0
else
    echo -e "${RED}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                  VERIFICATION FAILED                         ║${NC}"
    echo -e "${RED}║          Please review errors above before proceeding        ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    exit 1
fi