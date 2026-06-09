# Changelog

All notable changes to this project will be documented in this file.

## [v0.7.4] - Runtime Path Fixes for VPS

**Release date:** 2026-06-10
**Tag:** v0.7.4
**Commit:** d907e52

### What's Changed

- Fixed VPS directory paths for installer
- Installer creates: `/etc/orvixpanel`, `/var/lib/orvixpanel`, `/var/log/orvixpanel`, `/run/orvixpanel`
- Installer updated to v0.7.4
- All 11 Go test packages passing
- Self-update preflight now uses correct runtime paths from `/etc/orvixpanel/orvixpanel.env`
- Health endpoint fallback updated to `127.0.0.1:8443`

### Installation

```bash
# Download and run installer
curl -fsSL https://github.com/reachfm/orvixpanel/releases/download/v0.7.4/orvixpanel-installer-v0.7.4.tar.gz | tar -xz
sudo bash scripts/install.sh
```

### Binary

- **File:** `orvixpanel.linux`
- **Size:** 17.4 MB
- **SHA256:** `0ac543d297ef1962367ca1b6e966b2d9d8267f7ae7196ecb3c8bce3f9107cc39`

### Files in Release Package

| File | SHA256 |
|------|--------|
| bin/orvixpanel.linux | `0ac543d297ef1962367ca1b6e966b2d9d8267f7ae7196ecb3c8bce3f9107cc39` |
| scripts/install.sh | `48c8e6c61f608369ced8be698fe14b5422f8210e6f2051bbfece5fe15fe8f397` |
| scripts/doctor.sh | `9aec3d8875ae92227853ca62f1ef293ff771191d014295e3a27fc0bbd6ccc998` |

---

## [v0.7.3] - Atomic VERSION File Writing

**Release date:** 2026-06-10
**Tag:** v0.7.3
**Commit:** c100925

### What's Changed

- Atomic VERSION file writing (write to .tmp, fsync, rename)
- VERSION written only after health verification passes
- VERSION rollback on health failure
- Stale VERSION detection (warns when VERSION commit != local HEAD)

---

## [v0.7.4-preview] - First Real Self-Update Proof

**Release date:** 2026-06-10
**Tag:** v0.7.4
**Commit:** 8a38b81

### What's Changed

- First production self-update with atomic VERSION writing
- Proves update system writes VERSION only after health passes
- Validates rollback restores VERSION on health failure
- Proves update history is written correctly