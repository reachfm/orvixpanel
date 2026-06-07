# OrvixPanel v0.2.2 — Reliability Hardening

**Release date:** 2026-06-08
**Codename:** Aegis (cont, the hardening)
**Tag:** `v0.2.2-reliability-hardening`
**Status:** install + uninstall + reinstall + systemd-mode + license-fail-safe + port-collision — all live-verified on Linux (WSL Ubuntu 26.04).

## What v0.2.2 actually delivers

v0.2.1 proved the install path works for the `--skip-systemd` (developer) mode. v0.2.2 closes the gap on every other install/reliability angle that v0.2.1 left open:

| Scope item | Before v0.2.2 | After v0.2.2 |
|------------|---------------|--------------|
| systemd-mode install | unit existed but failed: `groupadd` blocked by `ProtectSystem=full` | unit hardened correctly: `ProtectSystem=yes` (keeps /etc writable) + `RuntimeDirectory=orvixpanel` (systemd creates the runtime dir) + explicit `ReadWritePaths` |
| install / uninstall / reinstall round trip | not tested | `smoke-reinstall-cycle.sh` proves a clean reinstall boots and the smoke contract holds on the second install (12/12 gates pass) |
| production-mode license boot | only dev fallback was tested | `test-license-mode.sh` proves `ORVIX_ALLOW_DEV=0` + no key fails closed (3/3 gates) |
| port collision on install | failed with a cryptic Go `address already in use` | install.sh refuses in **preflight** with a clear message naming the conflicting process |
| doctor output | human-readable only | `doctor.sh --json` outputs a single JSON document for tooling / dashboards |
| uninstall leaves /opt/orvixpanel | empty parent dir was left behind | `rm -rf /opt/orvixpanel` (cleanup completeness) |

## What ships in v0.2.2

### `scripts/install.sh` — port-collision precheck + systemd unit hardening
- **Preflight port check** (new step 0). If the bind port is in use, the installer dies before touching apt, with a message naming the conflicting process:
  ```
  FAIL: bind port 28445 is already in use by 'orvixpanel' (detected via ss).
  Free it (e.g. 'systemctl stop orvixpanel' or kill the pid) or re-run with
  a different port: --bind 0.0.0.0:<other>
  ```
- **systemd unit hardening** (rewritten):
  - `RuntimeDirectory=orvixpanel` + `RuntimeDirectoryMode=0755` — systemd creates `/run/orvixpanel` at every service start, eliminates the v0.2.1a race where the binary had to self-heal the dir under systemd.
  - `ProtectSystem=yes` (not `=full`) — keeps `/usr`, `/boot`, `/efi` read-only but leaves `/etc` writable for `useradd`/`groupadd` (which need to update `/etc/{passwd,shadow,group,gshadow}`) and for the vhost / fpm-pool / env-file writes.
  - Explicit `ReadWritePaths=/var/lib/orvixpanel /var/log/orvixpanel /run/orvixpanel /etc/orvixpanel /etc/nginx/conf.d/orvix /etc/php/<ver>/fpm/pool.d` — the only dirs the binary needs to mutate.

### `scripts/doctor.sh` — `--json` output mode
- New `--json` flag emits a single document:
  ```json
  {
    "host":"DESKTOP-2CPOJJD",
    "date":"2026-06-07T21:22:34Z",
    "state":"WARN",
    "ok":13,"warn":2,"fail":0,
    "checks":[
      {"status":"OK","check":"binary","detail":"..."},
      {"status":"WARN","check":"systemd unit","detail":"..."},
      ...
    ]
  }
  ```
  Human-readable output (default) is unchanged. Exit code is identical in both modes (`0`/`1`/`2`).
- `python3 -m json.tool` validates the output.

### `scripts/uninstall.sh` — full /opt cleanup
- `rm -rf /opt/orvixpanel` (was `rm -f /opt/orvixpanel/bin/orvixpanel`, which left the empty parent). Required for the reinstall-cycle smoke to assert a clean slate.

### `scripts/test-license-mode.sh` (new) — production-mode license contract
- Three checks:
  1. `ORVIX_ALLOW_DEV=0` + no `ORVIX_LICENSE_KEY` → binary refuses to boot with a license error
  2. `ORVIX_ALLOW_DEV=1` + no `ORVIX_LICENSE_KEY` → binary boots in dev fallback, `/healthz` returns ok
  3. `ORVIX_ALLOW_DEV` unset/0 + no `ORVIX_LICENSE_KEY` → fail-closed (same as #1)
- Uses `env -i` to start the binary with a CLEAN environment, so the test isn't poisoned by the install-time `/etc/orvixpanel/orvixpanel.env`.
- The error message is preserved (verifies the operator sees a real license error, not a generic crash):
  ```
  FTL orvixpanel exited with error
      error="load config: license.key is required (or set ORVIX_ALLOW_DEV=1)"
  ```

### `scripts/test-port-collision.sh` (new) — preflight port-collision contract
- Five checks:
  1. install.sh exits non-zero when the bind port is busy
  2. install.sh's error message names the port and the conflicting process
  3. install.sh fails fast in preflight (no apt installs)
  4. the port is correctly freed
  5. install.sh succeeds when the port is free (`--no-start` so it doesn't race the already-running production binary)

### `scripts/smoke-reinstall-cycle.sh` (new) — install/uninstall/reinstall round trip
- Twelve gates:
  1. uninstall `--keep-user --keep-nginx --keep-php-pkg --yes` succeeds
  2. `/var/lib/orvixpanel` preserved
  3. `/opt/orvixpanel` removed
  4. `/run/orvixpanel` removed
  5. install (cycle 1) succeeds
  6. doctor (cycle 1) reports 0 FAIL
  7. smoke-phase2 (cycle 1) 16/16
  8. uninstall (cycle 1) succeeds without --purge
  9. `/var/lib/orvixpanel` still preserved (reinstall-grade data survives)
  10. install (cycle 2, on preserved data) succeeds
  11. doctor (cycle 2) reports 0 FAIL
  12. smoke-phase2 (cycle 2) 16/16 — proves reinstall is fully functional

### `scripts/smoke-systemd.sh` (new) — full systemd path
- Twelve gates:
  1. `systemctl is-system-running` => `running`
  2. clean prior unit (if any)
  3. uninstall reset
  4. install (no `--skip-systemd`) succeeds
  5. `/etc/systemd/system/orvixpanel.service` present
  6. `systemctl is-active orvixpanel` => `active`
  7. `systemctl is-enabled orvixpanel` => `enabled`
  8. `/healthz` reachable
  9. smoke-phase2 16/16 (full end-to-end under systemd)
  10. `systemctl stop` succeeds
  11. `systemctl disable` succeeds
  12. unit file removed (cleanup)

## How to verify on a clean Linux box

```bash
# 1. clone + build
git clone https://github.com/reachfm/orvixpanel
cd orvixpanel
go build -ldflags="-s -w -X main.version=0.2.2" \
  -o bin/orvixpanel.linux ./cmd/orvixpanel

# 2. install
sudo bash scripts/install.sh
sudo bash scripts/doctor.sh              # human-readable
sudo bash scripts/doctor.sh --json       # JSON for tooling

# 3. full smoke
BASE=http://localhost:8443 bash smoke-phase2.sh
sudo bash scripts/smoke-reinstall-cycle.sh
sudo bash scripts/smoke-systemd.sh
sudo bash scripts/test-license-mode.sh
sudo bash scripts/test-port-collision.sh
```

## Test summary

`go test -count=1 ./...` — 7 packages, 67 tests, all green (unchanged from v0.2.1a).

v0.2.2 ships **three new bash smokes** (license, port, reinstall) and **one new bash smoke** (systemd). All live-verified end-to-end on WSL Ubuntu 26.04 with the actual binary, actual `useradd`/`groupadd`, actual nginx, actual php-fpm 8.5, and a real port collision.

| Smoke | Gates | Status |
|-------|-------|--------|
| `bash scripts/doctor.sh` | 13 OK / 2 WARN / 0 FAIL | PASS |
| `bash scripts/doctor.sh --json` | valid JSON, same 13/2/0 | PASS |
| `bash smoke-phase2.sh` (--skip-systemd) | 16/16 | PASS |
| `bash scripts/smoke-systemd.sh` | 12/12 | PASS |
| `bash scripts/smoke-reinstall-cycle.sh` | 12/12 | PASS |
| `bash scripts/test-license-mode.sh` | 3/3 | PASS |
| `bash scripts/test-port-collision.sh` | 5/5 | PASS |

## Files changed (v0.2.2)

| File | Change | Lines |
|------|--------|-------|
| `scripts/install.sh` | port-collision precheck + systemd unit hardening | +30 |
| `scripts/uninstall.sh` | `rm -rf /opt/orvixpanel` | +1 |
| `scripts/doctor.sh` | `--json` mode + `add_json_record` accumulator | +60 / -10 |
| `scripts/test-license-mode.sh` | NEW | 110 |
| `scripts/test-port-collision.sh` | NEW | 105 |
| `scripts/smoke-reinstall-cycle.sh` | NEW | 145 |
| `scripts/smoke-systemd.sh` | NEW | 140 |
| `RELEASE_NOTES_v0.2.2.md` | NEW | this file |

## What's NOT in v0.2.2 (out of scope per task brief)

- DNS / Mail / SSL / WAF / eBPF / CrowdSec
- AI Guardian
- Reseller / WHMCS
- Frontend rebuild
- TLS termination
- Phase 3 (anything beyond install / lifecycle / reliability)

## Known v0.2.2 caveats

- **WSL port 80** is held by Docker Desktop; not an installer bug, the smoke skips it.
- **systemd inside WSL** requires `[boot] systemd=true` in `/etc/wsl.conf` + `wsl --shutdown`. The smoke refuses to run otherwise (it checks `is-system-running` up front).
- **Binary in v0.2.2 is unchanged** from v0.2.1a. v0.2.2 is purely script + unit + smoke work; the runtime is the same Go binary (sha256 `25C54340…`).
- **`test-port-collision.sh` requires `python3`** to occupy a port; available on all Ubuntu 22.04+ images.
- **The 2 WARNs on doctor** are environment-only (systemd unit not present when running `--skip-systemd`, port 80 held by Docker on WSL). On a bare VPS both become OK and the doctor returns `0 0 0`.

## Upgrade path from v0.2.1 / v0.2.1a

`install.sh` is still idempotent. Re-run to pick up the v0.2.2 systemd unit template. The new preflight and `--json` doctor are opt-in (the preflight is always run; `--json` is a flag).

## License

OrvixPanel v0.2.2 binary is unchanged from v0.2.1a (Business Source License 1.1 — free for non-production use, converts to Apache 2.0 after 4 years).
