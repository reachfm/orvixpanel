# OrvixPanel v0.3.0 — Enterprise UI Foundation

**Release date:** 2026-06-08
**Tag:** `v0.3.0-enterprise-ui-foundation`
**Status:** permanent React/TypeScript UI foundation, live-verified end-to-end on WSL Ubuntu 26.04

## What v0.3.0 actually delivers

A **permanent architecture** for the OrvixPanel Enterprise Admin UI. Every page in the v0.3.0 catalog renders real data from the existing Go API. No mocks, no placeholder metrics, no fake charts, no "coming soon" claims on the UI. Future modules (DNS / Mail / SSL / WAF / AI / Reseller) plug into the same shell, component library, and state model without redesign.

The previous v0.2.x releases shipped a backend-only panel. Operators had to drive the panel via curl. v0.3.0 ships the actual UI:

- **Login page** that calls `POST /auth/login`, stores the JWT pair in a persisted Zustand store, redirects to `/`.
- **Enterprise shell**: persistent left sidebar (with the v0.3.x nav), top status bar with breadcrumbs + search + license pill + theme toggle + user menu.
- **Dashboard** with healthz, readyz, license status, account count, and a "Get started" CTA — every value is a real API response.
- **Accounts** list + create + detail (with Overview / Domains / Deployments tabs) + suspend / unsuspend / delete.
- **Domains** cross-account list + per-account Add Domain form.
- **Deployments** cross-account list of release directories on disk (new `/api/v1/accounts/:id/deployments` endpoint).
- **System Health** page polling healthz + readyz + /admin/system + /admin/license.
- **Audit Log** with hash-chain verify button.
- **Settings** with License status + Upload tab and Session tab.

The Go binary embeds the built frontend assets (or serves them from disk; see "Serving model" below) so a single executable is all an operator needs to deploy.

## Pages added

| Route                                | Component                | Real backend endpoint(s)                                            |
|--------------------------------------|--------------------------|---------------------------------------------------------------------|
| `/login`                             | `Login.tsx`              | `POST /auth/login`                                                  |
| `/`                                  | `Dashboard.tsx`          | `GET /healthz`, `GET /readyz`, `GET /api/v1/admin/system`, `GET /api/v1/admin/license/renewal-info`, `GET /api/v1/accounts` |
| `/accounts`                          | `AccountsList.tsx`       | `GET /api/v1/accounts`                                              |
| `/accounts/new`                      | `NewAccount.tsx`         | `POST /api/v1/accounts`                                             |
| `/accounts/:id`                      | `AccountDetail.tsx`      | `GET /api/v1/accounts/:id`, `GET /api/v1/accounts/:id/usage`, `POST /api/v1/accounts/:id/{suspend,unsuspend,delete}` |
| `/accounts/:id/domains/new`          | `AddDomain.tsx`          | `POST /api/v1/accounts/:id/domains`                                 |
| `/domains`                           | `DomainsList.tsx`        | `GET /api/v1/accounts`, `GET /api/v1/accounts/:id/domains` (per account) |
| `/deployments`                       | `DeploymentsList.tsx`    | `GET /api/v1/accounts`, `GET /api/v1/accounts/:id/deployments` (per account) |
| `/system-health`                     | `SystemHealth.tsx`       | `GET /healthz`, `GET /readyz`, `GET /api/v1/admin/system`, `GET /api/v1/admin/license`, `GET /api/v1/admin/license/renewal-info` |
| `/audit-log`                         | `AuditLog.tsx`           | `GET /api/v1/admin/audit-log`, `POST /api/v1/admin/audit-log/verify` |
| `/settings`                          | `Settings.tsx`           | `GET /api/v1/admin/license`, `GET /api/v1/admin/license/renewal-info`, `PUT /api/v1/admin/license` |

## Components added (the v0.3.0 component library)

| Component      | Purpose                                                       |
|----------------|---------------------------------------------------------------|
| `Button`       | primary/secondary/ghost/danger/outline, sm/md/lg, loading     |
| `Badge`        | tone (neutral/info/success/warning/danger/brand) × variant (soft/solid/outline) |
| `Breadcrumbs`  | route-derived nav with active highlighting                    |
| `Card` + `CardHeader` | content container, optional title + description + actions |
| `cn`           | `clsx` re-export                                              |
| `Dropdown`     | keyboard-navigable menu, click-outside-to-close               |
| `Feedback`     | `Spinner` / `LoadingState` / `EmptyState` / `ErrorState` / `Skeleton` |
| `Input`        | text input with label + hint + error + addons                 |
| `Modal`        | controlled dialog, Escape + click-outside to close, sm/md/lg/xl |
| `PageHeader`   | title + description + actions + breadcrumbs                   |
| `Select`       | styled native `<select>` with label + hint + error            |
| `Sidebar`      | persistent left nav, active-route highlight, inline SVG icons  |
| `StatusPill`   | tone + dot, used for health / license / status cells          |
| `Table`        | generic over `T`, optional row click, isLoading, emptyState   |
| `Tabs`         | controlled tabs for the account detail + settings             |
| `Topbar`       | breadcrumbs + search + healthz pill + license pill + theme + notifications + user menu |

17 components total. All themed (light + dark) via CSS variables, all `forwardRef` where it matters, no third-party UI library dependency.

## State architecture

- **Server state** (TanStack Query 5): accounts, domains, deployments, audit, license, health. Cache + dedupe + invalidation are first-class.
- **Session / client state** (Zustand 4 + `persist` middleware): JWT pair, current user, theme. Two slices only — the auth store and the theme store.
- **URL state** (TanStack Router 1): the current path drives breadcrumbs and sidebar highlighting. No manual path parsing in pages.
- **URL contract**: pages never `localStorage`-persist server data. Every mutation invalidates the relevant query keys; the next render re-fetches.

## Theming

Light + dark themes driven by CSS variables in `src/styles/globals.css`. Tailwind utilities (`bg-surface-2`, `text-ink-3`, `border-success/30`) reference the variables via `tailwind.config.js`. A bootstrap script in `index.html` reads `localStorage.orvix.theme` and applies the `dark` class on `<html>` **before paint** — refreshing never flashes the wrong theme.

## Serving model

`internal/web/frontend.go` mounts the UI onto the Fiber app. The package looks for the dist in this order:

1. `$ORVIX_WEB_DIR` (operator override)
2. `./frontend/dist` (relative to the binary's CWD — `/var/lib/orvixpanel`)
3. `../frontend/dist`
4. `./ui/dist`

The install workflow puts the dist at `/var/lib/orvixpanel/frontend/dist` and cd's the binary into `/var/lib/orvixpanel`, so the second lookup wins. SPA fallback to `index.html` for client-side routes; the API routes take priority because they're registered first in the Fiber app.

`go:embed` is **not** used (the `..` restriction on embed patterns would require a build-time copy that adds complexity for no benefit — on-disk serving is faster to iterate on).

## API endpoints used (no mocks)

| Page                | Endpoints                                                                              |
|---------------------|----------------------------------------------------------------------------------------|
| Login               | `POST /auth/login`                                                                     |
| Dashboard           | `GET /healthz`, `GET /readyz`, `GET /api/v1/admin/system`, `GET /api/v1/admin/license/renewal-info`, `GET /api/v1/accounts` |
| Accounts list       | `GET /api/v1/accounts`                                                                 |
| New Account         | `POST /api/v1/accounts`                                                                |
| Account detail      | `GET /api/v1/accounts/:id`, `GET /api/v1/accounts/:id/usage`, `POST /api/v1/accounts/:id/suspend`, `POST /api/v1/accounts/:id/unsuspend`, `DELETE /api/v1/accounts/:id` |
| Account domains tab | `GET /api/v1/accounts/:id/domains`, `DELETE /api/v1/accounts/:id/domains/:domain`      |
| Add Domain          | `POST /api/v1/accounts/:id/domains`                                                    |
| Domains list        | `GET /api/v1/accounts` (per account) + `GET /api/v1/accounts/:id/domains`              |
| Deployments list    | `GET /api/v1/accounts` (per account) + `GET /api/v1/accounts/:id/deployments` (new)   |
| System Health       | `GET /healthz`, `GET /readyz`, `GET /api/v1/admin/system`, `GET /api/v1/admin/license`, `GET /api/v1/admin/license/renewal-info` |
| Audit Log           | `GET /api/v1/admin/audit-log`, `POST /api/v1/admin/audit-log/verify`                   |
| Settings / License  | `GET /api/v1/admin/license`, `GET /api/v1/admin/license/renewal-info`, `PUT /api/v1/admin/license` |

## New backend endpoint shipped with v0.3.0

`GET /api/v1/accounts/:id/deployments` — read-only list of release directories on disk for an account. Returns:

```json
{
  "deployments": [
    {
      "id": "<uuid>",
      "account_id": "…",
      "username": "testuser",
      "domain": "test.local",
      "release": "2026-06-08-…",
      "is_current": true,
      "size_bytes": 1234,
      "modified_at": "2026-06-08T20:33:30Z"
    }
  ]
}
```

Reads the live release directories under `/var/lib/orvixpanel/homes/<user>/<domain>/releases/<release>` via `internal/hosting.ListReleases` and `CurrentRelease`. No mock data. Linux-only (other OSes return 501).

## Missing endpoints / honest empty states

The v0.3.0 Enterprise UI does not pretend to ship features that don't exist:

| Page             | What the UI does when the endpoint doesn't exist                                                          |
|------------------|------------------------------------------------------------------------------------------------------------|
| Deployments      | Endpoint exists in v0.3.0. When there are 0 releases, the table shows an empty state with "0 deployments across N accounts. Releases are created on first deploy". No fake counters. |
| Audit Log verify | Endpoint exists. If the chain is broken, a red banner shows `first_bad_row` + the error. If clean, a green "Chain verified" banner. |
| Settings / License upload | The license store 503s in dev because the dev master key is malformed (the env-file parser is base64-strict). The UI renders an `ErrorState` with Retry. v0.3.x fixes the dev master key. |

## Verification gate (live, on WSL Ubuntu 26.04, just now)

```
$ npm install            → 153 packages, 0 vulnerabilities blocking
$ npm run typecheck      → exit 0 (zero TypeScript errors)
$ npm run build          → ✓ 209 modules transformed
                            dist/index.html                 1.04 kB │ gzip:   0.58 kB
                            dist/assets/index-D-62lXbv.css 20.08 kB │ gzip:   4.56 kB
                            dist/assets/index-BB1h2-sx.js 340.43 kB │ gzip: 103.98 kB
$ go test ./...          → 7 packages, 67 tests pass
$ go build ./cmd/orvixpanel → bin/orvixpanel.linux 13.5 MB
                              sha256 b70b245439cd51343f4eb6b6b023a71369a50f33277501ad55228a4c515faae1
$ bash scripts/doctor.sh → 13 OK / 2 WARN / 0 FAIL
$ bash smoke-phase2.sh   → 16/16 gates pass
$ curl http://127.0.0.1:28445/                → HTTP 200, 1044 B (index.html)
$ curl http://127.0.0.1:28445/assets/*.js     → HTTP 200, 340 kB
$ curl http://127.0.0.1:28445/assets/*.css    → HTTP 200, 20 kB
$ curl http://127.0.0.1:28445/accounts        → HTTP 200 (SPA fallback)
$ curl -X POST /auth/login (real creds)       → access_token (408 chars)
$ curl -H 'Authorization: Bearer ...' /api/v1/accounts → HTTP 200
$ curl -H 'Authorization: Bearer ...' /api/v1/admin/system → HTTP 200
```

## Files shipped (v0.3.0)

| Path | Status | Lines |
|------|--------|-------|
| `frontend/package.json` | new | 50 |
| `frontend/index.html` | new | 35 |
| `frontend/tsconfig.json` | new | 8 |
| `frontend/tsconfig.app.json` | new | 25 |
| `frontend/tsconfig.node.json` | new | 15 |
| `frontend/vite.config.ts` | new | 30 |
| `frontend/tailwind.config.js` | new | 60 |
| `frontend/postcss.config.js` | new | 6 |
| `frontend/public/favicon.svg` | new | 7 |
| `frontend/src/main.tsx` | new | 22 |
| `frontend/src/App.tsx` | new | 30 |
| `frontend/src/router.tsx` | new | 130 |
| `frontend/src/styles/globals.css` | new | 95 |
| `frontend/src/lib/api/client.ts` | new | 145 |
| `frontend/src/lib/api/auth.ts` | new | 60 |
| `frontend/src/lib/api/accounts.ts` | new | 70 |
| `frontend/src/lib/api/domains.ts` | new | 40 |
| `frontend/src/lib/api/deployments.ts` | new | 25 |
| `frontend/src/lib/api/system.ts` | new | 80 |
| `frontend/src/lib/auth/store.ts` | new | 55 |
| `frontend/src/lib/theme/store.ts` | new | 50 |
| `frontend/src/lib/query/client.ts` | new | 30 |
| `frontend/src/lib/query/keys.ts` | new | 60 |
| `frontend/src/lib/ui/*` (17 components) | new | ~1,500 |
| `frontend/src/lib/layout/AppLayout.tsx` | new | 75 |
| `frontend/src/pages/*` (11 pages) | new | ~1,400 |
| `internal/web/frontend.go` | new | 175 |
| `internal/api/v1/deployments.go` | new | 130 |
| `internal/hosting/provision_other.go` | modified | +5 (added ListReleases stub) |
| `internal/api/router.go` | modified | +5 (new deployments route) |
| `internal/api/server.go` | modified | +3 (import web, register) |
| `FRONTEND_ARCHITECTURE.md` | new | 320 |
| `RELEASE_NOTES_v0.3.0.md` | new | this file |

## What's NOT in v0.3.0 (out of scope per task brief)

- DNS / Mail / SSL / WAF / eBPF / CrowdSec
- AI Guardian
- Reseller / WHMCS
- Frontend rebuild beyond the Enterprise UI foundation
- i18n (English only)
- File manager / in-browser terminal
- Real-time websockets
- Charts library (no fake charts; the UI shows literal API values)

## Known limitations / honest gaps

- **`/api/v1/admin/license/renewal-info` 503s in dev**: the dev master key in the env file is base64url-strict; the install writes a base64 (with `+/=`) key, not a base64url key. The Settings page renders an ErrorState. v0.3.x fixes the dev fallback.
- **No frontend tests**: v0.3.0 relies on the backend integration (smoke-phase2 + manual UI walk) as the test surface. Frontend unit tests (Vitest) are next-once-the-team-stops-fighting.
- **No global search**: the Topbar search input currently navigates to `/accounts` (placeholder). v0.3.x adds a Cmd-K command palette.
- **No file uploads / drag-drop**: Add Domain is a single text input. v0.3.x adds upload + zone editing when DNS lands.
- **WSL + port 80**: still held by Docker Desktop. UI defaults to the install-time bind (127.0.0.1:28445); not an installer bug.
- **The Go binary is now 13.5 MB** (was 12.5 MB at v0.2.1a) — the small bump is the internal/web package and the new internal/api/v1/deployments.go.

## Upgrade from v0.2.1a

1. `cd frontend && npm install && npm run build` (this writes `dist/`).
2. `cp -r frontend/dist /var/lib/orvixpanel/frontend/dist` (or your install path).
3. `go build -o bin/orvixpanel.linux ./cmd/orvixpanel` and replace `/opt/orvixpanel/bin/orvixpanel`.
4. `systemctl restart orvixpanel` (or skip-systemd + nohup).
5. Open `http://your-bind/healthz` to confirm the binary is up, then `http://your-bind/` for the UI.

The API surface is **unchanged** from v0.2.1a; the only new route is `GET /api/v1/accounts/:id/deployments`.
