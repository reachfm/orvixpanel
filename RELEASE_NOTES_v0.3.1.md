# OrvixPanel v0.3.1 — Enterprise UI Quality Pass

**Release Date:** 2024
**Tag:** `v0.3.1-enterprise-ui-quality-pass`

## Summary

v0.3.1 is an enterprise UI quality pass focused on hardening security, improving user experience, and establishing a foundation for frontend testing. This release does **not** include Phase 3 features (DNS, Mail, SSL).

## Breaking Changes

None. This release is fully backward-compatible with v0.3.0.

---

## Backend Changes

### Security Hardening

#### Admin Bootstrap Hardening

- **Removed random credential generation**: Bootstrap no longer generates and logs random admin passwords in production mode
- **Added `--admin-email` flag**: Specify admin email during non-interactive bootstrap
- **Added `--admin-password` flag**: Specify admin password during non-interactive bootstrap
- **Interactive secure prompts**: Interactive mode now securely prompts for email and password with validation
- **Clear non-interactive failure**: Production mode fails with explicit error message when credentials not provided

```bash
# Non-interactive (production-safe)
orvixpanel --bootstrap --admin-email admin@example.com --admin-password "SecurePass123!"

# Interactive (with secure password prompt)
orvixpanel --bootstrap
```

#### Password Validation

- Minimum 8 characters
- Maximum 128 characters
- Requires uppercase letter
- Requires lowercase letter
- Requires digit

#### Email Validation

- Standard RFC 5322 email format validation
- Proper error messages for invalid formats

---

## Frontend Changes

### Enterprise Dashboard Upgrade

- **System Status Cards**: Top row showing API health, database status, license status, and system uptime
- **Resource Counts**: Account count, domain count, and deployment count with per-account breakdowns
- **Recent Activity**: Last 10 audit log entries displayed on dashboard
- **License Summary**: Tier, days remaining, grace period information

### Accounts UX Improvements

- **Search filtering**: Real-time search by account name or description
- **Status filtering**: Filter by active/suspended accounts
- **Pagination**: 20 accounts per page with page controls
- **Confirmation modals**: Modal dialogs for suspend, unsuspend, and delete operations
- **Improved empty state**: User-friendly empty state when no accounts exist
- **Better loading states**: Spinner during data fetch

### Domains UX Improvements

- **Cross-account view**: All domains across accounts in a single table with account links
- **Search filtering**: Real-time search by domain name, account username, or document root
- **Status filtering**: Filter by Active, Suspended, or Pending domains
- **Pagination**: 25 domains per page with Previous/Next navigation
- **Status badges**: Visual status pills for active/suspended/pending domains
- **Creation flow**: Dedicated AddDomain page with proper validation

### System Health Page

- **Professional status page**: Comprehensive system health display
- **Real API endpoints**: Polls actual `/healthz`, `/readyz`, `/api/v1/admin/system`, and license endpoints
- **Health metrics grid**: Card-based display of API, Database, License, and System Build status
- **Overall status banner**: Summary banner showing overall system health
- **Live polling**: 5-second refresh for health probes, 30-second for system info

### Audit Log UX Improvements

- **Search filtering**: Filter by actor, action, or resource
- **Result filtering**: Filter by success/failure
- **Pagination**: 25 entries per page with page controls
- **Verification status banner**: Visual indication of log integrity verification
- **Timestamp display**: Formatted date and time columns

### Frontend Architecture

#### ErrorBoundary

- React error boundary wrapping the entire application
- User-friendly error page with "Something went wrong" message
- "Refresh page" and "Try again" actions
- Development mode shows error details in expandable section

#### Global Notification System

- Zustand-based notification store
- `useNotification()` hook for easy toast creation
- `NotificationContainer` component for rendering toasts
- Support for success, error, warning, and info notifications
- Auto-dismiss with configurable duration
- Smooth exit animations

#### Session Management & Route Guards

- JWT expiry parsing and monitoring
- 30-second session check interval
- 5-minute warning before session expiry
- Auto-logout on token expiration
- User notification via toast on session events

### Dark Mode Improvements

- Polished CSS variable-based theming
- Improved focus ring styling for dark mode
- Removed hardcoded dark mode values in index.css
- Proper color scheme application via CSS variables

### Frontend Testing Infrastructure

#### Vitest Setup

- Test runner: **Vitest** with jsdom environment
- Testing utilities: **Testing Library** for React
- Configuration: `vitest.config.ts` with proper path aliases
- Setup file: Global mocks and jest-dom matchers

#### Test Files Added

- `Notification.test.tsx` - Notification store and component tests
- `ErrorBoundary.test.tsx` - Error boundary component tests
- `store.test.ts` (theme) - Theme store tests

#### Test Commands

```bash
npm test              # Run all tests
npm run test:watch    # Watch mode
npm run test:coverage # With coverage report
```

---

## Documentation

### New Documents

- `FRONTEND_TESTING.md` - Comprehensive testing guide covering:
  - Setup and configuration
  - Writing component tests
  - Writing store tests
  - Async testing patterns
  - Mocking strategies
  - Best practices

### Updated Documents

- `FRONTEND_ARCHITECTURE.md`:
  - Updated stack table with Vitest
  - Added test directory structure
  - Added ErrorBoundary and Notification components
  - Updated component count (17 → 19)
  - Added Route Guards & Session Management section
  - Updated test instructions for new components

---

## Verification Gates

All verification gates must pass before merging:

```bash
# Frontend
cd frontend
npm install
npm run typecheck
npm run build
npm test

# Backend
go test ./...
go build ./cmd/orvixpanel

# Scripts
./scripts/doctor.sh
./scripts/smoke-phase2.sh
```

---

## Migration Guide

### From v0.3.0 to v0.3.1

#### Bootstrap Usage Change

**Before (v0.3.0):**
```bash
# Would log random password to stdout
orvixpanel --bootstrap
```

**After (v0.3.1):**
```bash
# Non-interactive - explicit credentials required
orvixpanel --bootstrap --admin-email admin@example.com --admin-password "SecurePass123!"

# Interactive - secure password prompt
orvixpanel --bootstrap
```

#### Environment Variables

No new environment variables required. Existing `ORVIX_ALLOW_DEV=1` behavior preserved.

---

## Known Limitations

- No i18n support (English only)
- No fake data on any pages (real API data only)
- No in-browser terminal or file manager
- Session monitoring requires valid JWT tokens

---

## Credits

OrvixPanel Team

---

## Next Release (v0.3.2)

Planned features:
- DNS zone management
- Mail server configuration
- SSL certificate management
- Expanded frontend test coverage

---

## Links

- [Documentation](docs/)
- [Frontend Testing Guide](frontend/FRONTEND_TESTING.md)
- [Frontend Architecture](FRONTEND_ARCHITECTURE.md)
- [Enterprise Plan](ENTERPRISE_PLAN.md)