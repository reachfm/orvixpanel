# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.0.x   | ✅                 |
| < 1.0   | ❌ (pre-release)   |

## Reporting a Vulnerability

**Please do not file public GitHub issues for security bugs.**

Email **security@orvixpanel.com** (PGP key below) with:

- A description of the vulnerability
- Steps to reproduce
- The impact you believe it has
- Any mitigation workarounds you've found

We commit to:

- **Acknowledge** within 48 hours
- **Triage** within 7 days
- **Patch** critical issues within 30 days
- **Credit** the reporter in the release notes (unless you ask to remain anonymous)

## Coordinated Vulnerability Disclosure (CVD)

We follow the [disclose.io](https://disclose.io/) CVD principles:

1. Reporter contacts us privately.
2. We confirm and start a fix.
3. We negotiate a 90-day disclosure window (extendable for complex fixes).
4. We release the patch + advisory on the same day.
5. Reporter publishes their writeup.

We will not pursue legal action against researchers who:

- Make a good-faith effort to avoid privacy violations
- Only interact with accounts they own (or have explicit permission for)
- Stop testing immediately if they encounter user data
- Do not exploit a vulnerability beyond what is necessary to demonstrate it

## PGP Key

```
pub   rsa4096 2026-01-01 [SC]
      0000 0000 0000 0000 0000 0000 0000 0000 0000 0000
uid   security@orvixpanel.com
```

(Fingerprint will be published at <https://orvixpanel.com/security.asc> with the v1.0 release.)

## Scope

In scope:

- The `orvixpanel` Go binary
- The `whmcs-orvixpanel` PHP module
- The React frontend bundled into the binary
- The installer + systemd unit
- The documentation in this repository

Out of scope:

- Third-party services we integrate with (PowerDNS, Postlane,
  CrowdSec, Let's Encrypt). Report to those projects directly.
- The host OS — keep it patched.
- The default nginx/PHP-FPM config we ship in `nginx.go` /
  `phpfpm.go` — these are templates; the operator's job to harden
  further.
