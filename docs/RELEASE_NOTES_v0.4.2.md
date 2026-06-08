# Release Notes — v0.4.2

**Tag**: `v0.4.2-powerdns-live-integration`
**Date**: 2026-06-08

PowerDNS Real Integration — verified end-to-end DNS management with PowerDNS server.

## Highlights

- **PowerDNS Live Integration**: Complete zone/record management synced to PowerDNS server
- **Transaction Safety**: Automatic rollback on PowerDNS failures
- **Installer Support**: `--with-powerdns` flag for automated PowerDNS setup
- **Smoke Testing**: Comprehensive `smoke-powerdns.sh` validation script

## New Features

### Installer PowerDNS Support

The `install.sh` script now accepts `--with-powerdns` to automatically:

1. Install PowerDNS packages (`pdns-server`, `pdns-backend-sqlite3`, `dnsutils`)
2. Configure PowerDNS with SQLite3 backend
3. Enable webserver and API on port 8081
4. Generate secure API key
5. Start PowerDNS service
6. Write required environment variables to `/etc/orvixpanel/orvixpanel.env`

```bash
sudo bash scripts/install.sh --with-powerdns
```

### DNS Mode Configuration

New `ORVIX_DNS_MODE` environment variable:

| Mode | Description |
|------|-------------|
| `local` (default) | DNS data stored in SQLite only |
| `powerdns` | Changes synced to PowerDNS server |

Environment variables written by installer:

```
ORVIX_DNS_MODE=powerdns
ORVIX_POWERDNS_URL=http://127.0.0.1:8081
ORVIX_POWERDNS_API_KEY=<generated-key>
```

### Transaction Rollback

The DNS Service now uses database transactions for PowerDNS operations:

- **Zone Creation**: If PowerDNS API call fails, the database insert is rolled back
- **Zone Deletion**: PowerDNS zone is deleted first; if it fails, the DB operation is aborted

This ensures data consistency between OrvixPanel and PowerDNS.

## Changes

### `internal/config/config.go`

- Added `DNSConfig` struct with `Mode`, `PowerDNSURL`, `PowerDNSAPIKey`, `PowerDNSServer` fields
- Added `dns.mode` default (`local`)
- Added environment variable bindings for `ORVIX_DNS_MODE`, `ORVIX_POWERDNS_URL`, `ORVIX_POWERDNS_API_KEY`, `ORVIX_POWERDNS_SERVER_ID`

### `internal/dns/service.go`

- `CreateZone()`: Uses transaction in PowerDNS mode; rolls back on PowerDNS failure
- `DeleteZone()`: Deletes from PowerDNS first, then DB; aborts on PowerDNS failure
- `syncRecordToPowerDNS()`: Called after record create/update/delete

### `scripts/install.sh`

- Added `--with-powerdns` flag parsing
- Added PowerDNS installation and configuration block
- Generates API key and writes to env file
- Starts PowerDNS service

### `scripts/smoke-powerdns.sh` (new)

Comprehensive smoke test that verifies:

1. PowerDNS server running
2. API reachable
3. Zone creation via Orvix API
4. Zone verification via PowerDNS API
5. Record creation
6. Record verification
7. DNS query with dig
8. Record deletion
9. Zone deletion
10. NXDOMAIN confirmation

## Breaking Changes

None. This release is fully backward-compatible.

## Deprecations

None.

## Bug Fixes

- Fixed transaction handling in `DeleteZone()` to delete from PowerDNS first (previously could leave orphaned zones in PowerDNS after DB deletion)

## Documentation

- Updated `docs/DNS_ENGINE.md` with v0.4.2 production status
- Added `--with-powerdns` flag to `INSTALL.md`
- Created `RELEASE_NOTES_v0.4.2.md` (this file)

## Verification

Run the verification gate:

```bash
# Go tests
go test ./...

# Build
go build ./cmd/orvixpanel

# PowerDNS smoke test (requires PowerDNS running)
bash scripts/smoke-powerdns.sh
```

## Upgrading

### From v0.4.0/v0.4.1

1. Pull the latest code
2. Rebuild: `go build ./cmd/orvixpanel`
3. If using PowerDNS, run installer with `--with-powerdns` or manually configure:
   - Set `ORVIX_DNS_MODE=powerdns` in `/etc/orvixpanel/orvixpanel.env`
   - Set `ORVIX_POWERDNS_URL` and `ORVIX_POWERDNS_API_KEY`
4. Restart: `systemctl restart orvixpanel`

## Credits

- OrvixPanel Core Team

---

**Next**: v0.5.0 will add WAF / eBPF firewall capabilities. See `NEXT_PHASE_PLAN.md`.