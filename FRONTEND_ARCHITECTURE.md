# OrvixPanel Frontend Architecture

> Permanent architecture for the OrvixPanel Enterprise Admin UI.
> v0.3.0 ships the foundation. Future modules (DNS / Mail / SSL /
> WAF / AI / Reseller) plug into the same shell, component
> library, and state model without redesign.

## 1. Stack

| Concern              | Choice                            | Why                                                                 |
|----------------------|-----------------------------------|---------------------------------------------------------------------|
| Build                | **Vite 5**                        | fast cold start, native ESM in dev, predictable prod bundle        |
| Language             | **TypeScript 5** (strict)         | catch API contract drift at compile time                            |
| UI framework         | **React 18**                      | ecosystem maturity, the rest of the company speaks it               |
| Routing              | **TanStack Router 1**             | type-safe routes, route-level data loading, file-system agnostic    |
| Server state         | **TanStack Query 5**              | cache + dedupe + invalidation + retry out of the box                |
| Client state         | **Zustand 4** (with persist)     | tiny, no Provider hell, persisted slices for auth + theme          |
| Styling              | **Tailwind 3** + CSS variables    | design tokens in CSS, Tailwind utilities reference them            |
| Form / validation    | (v0.3.0) — native + ApiError map  | first-party only; React Hook Form lands when forms multiply        |
| Tests                | (v0.3.0) — none on the frontend   | backend integration is the load-bearing test surface                |
| Linting              | (v0.3.0) — `tsc --noEmit`         | ESLint + Prettier are next-once-the-team-stops-fighting              |

## 2. Folder layout

```
frontend/
├── index.html                    # bootstrap <script> applies theme pre-paint
├── package.json
├── tsconfig.json                 # solution file (project references)
├── tsconfig.app.json             # app-side TS config
├── tsconfig.node.json            # vite.config.ts side
├── vite.config.ts
├── tailwind.config.js            # reads CSS variables for theme tokens
├── postcss.config.js
├── public/                       # copied verbatim to dist/
│   └── favicon.svg
└── src/
    ├── main.tsx                  # React mount, StrictMode, global CSS
    ├── App.tsx                   # QueryClientProvider + RouterProvider
    ├── router.tsx                # code-based route tree
    ├── styles/
    │   └── globals.css           # @tailwind + CSS variable design tokens
    ├── lib/
    │   ├── api/                  # typed wrappers over every backend route
    │   │   ├── client.ts         # fetch + 401-refresh + ApiError
    │   │   ├── auth.ts
    │   │   ├── accounts.ts
    │   │   ├── domains.ts
    │   │   ├── deployments.ts
    │   │   └── system.ts
    │   ├── auth/
    │   │   └── store.ts          # Zustand: tokens + user, persisted
    │   ├── theme/
    │   │   └── store.ts          # Zustand: light/dark, persisted
    │   ├── query/
    │   │   ├── client.ts         # QueryClient + retry rules
    │   │   └── keys.ts           # per-module key factory
    │   ├── ui/                   # the component library
    │   │   ├── Badge.tsx
    │   │   ├── Breadcrumbs.tsx
    │   │   ├── Button.tsx
    │   │   ├── Card.tsx
    │   │   ├── cn.ts
    │   │   ├── Dropdown.tsx
    │   │   ├── Feedback.tsx       # Spinner / LoadingState / EmptyState / ErrorState / Skeleton
    │   │   ├── Input.tsx
    │   │   ├── Modal.tsx
    │   │   ├── PageHeader.tsx
    │   │   ├── Select.tsx
    │   │   ├── Sidebar.tsx
    │   │   ├── StatusPill.tsx
    │   │   ├── Table.tsx
    │   │   ├── Tabs.tsx
    │   │   └── Topbar.tsx
    │   └── layout/
    │       └── AppLayout.tsx     # sidebar + topbar + <Outlet/>
    └── pages/                    # one file per route
        ├── Login.tsx
        ├── Dashboard.tsx
        ├── AccountsList.tsx
        ├── AccountDetail.tsx
        ├── NewAccount.tsx
        ├── AddDomain.tsx
        ├── DomainsList.tsx
        ├── DeploymentsList.tsx
        ├── SystemHealth.tsx
        ├── AuditLog.tsx
        └── Settings.tsx
```

## 3. State model — the three layers

| Layer               | Lives in        | Persisted? | Examples                                                  |
|---------------------|-----------------|------------|-----------------------------------------------------------|
| **Server state**    | TanStack Query  | no (the server is the source of truth) | accounts list, license status, audit log entries |
| **Session / client state** | Zustand    | yes (localStorage)                      | JWT pair, current user, theme |
| **URL state**        | TanStack Router | yes (browser URL)                       | current path, search params, breadcrumbs |

The contract between the three is strict: **nothing in the server-state layer is duplicated in the client-state layer.** When the user submits a mutation we invalidate the relevant query keys via `qc.invalidateQueries({ queryKey: keys.all() })` and let the next render re-fetch. No manual caching, no storing accounts in Zustand.

## 4. Theming

CSS variables are the single source of truth for the design tokens. Tailwind's `tailwind.config.js` references them so every utility class (`bg-surface-2`, `text-ink-3`, `border-success/30`) resolves to the right RGB triplet for the current theme.

```css
/* light defaults (in :root) */
--surface-0: 255 255 255;   /* page bg  */
--surface-1: 248 250 252;   /* card     */
--ink-1:      15  23  42;   /* primary  */
--ink-3:     100 116 139;   /* muted    */
--success:    16 185 129;

/* dark overrides (in .dark) */
.dark {
  --surface-0:   9  11  16;
  --surface-1:  17  20  28;
  --ink-1:     226 232 240;
  --ink-3:     100 116 139;
}
```

A bootstrap script in `index.html` reads `localStorage.orvix.theme` and applies the `dark` class on `<html>` **before paint**, so refreshing never flashes the wrong theme. The Zustand `theme` store re-applies it on every navigation.

## 5. API client

`src/lib/api/client.ts` wraps `fetch` with:

- automatic `Authorization: Bearer <accessToken>` injection from the Zustand auth store
- automatic token refresh on `401` (single in-flight refresh promise; concurrent requests wait on it)
- typed `ApiError` with the stable `{ code, request_id, status }` shape the Go backend returns
- a single `request<T>(path, opts)` function every module uses

Modules own slices of the API surface:

```ts
// src/lib/api/accounts.ts
export function listAccounts(): Promise<{ accounts: Account[] }> { ... }
export function getAccount(id: string): Promise<Account> { ... }
// ...
```

Pages never construct a `fetch` call directly. If a new endpoint ships, add a function in the matching module; the call site gets a typed return value for free.

### Adding a new endpoint

1. Add the handler in `internal/api/v1/<thing>.go`.
2. Register it in `internal/api/router.go`.
3. Add the wrapper in `frontend/src/lib/api/<thing>.ts`.
4. Add a typed shape in the module's interface.
5. The page calls the new wrapper via `useQuery` / `useMutation`.

## 6. Component library

All components live in `frontend/src/lib/ui/`. The library has 17 components in v0.3.0:

| Component     | Variants / Props                                  |
|---------------|---------------------------------------------------|
| `Button`      | `variant: primary/secondary/ghost/danger/outline`, `size: sm/md/lg`, `loading`, `leftIcon`/`rightIcon` |
| `Badge`       | `tone: neutral/info/success/warning/danger/brand`, `variant: soft/solid/outline` |
| `Breadcrumbs` | `items: Crumb[]`                                  |
| `Card`        | `padding: none/sm/md/lg` + `CardHeader`           |
| `Dropdown`    | `trigger: (open) => ReactNode`, `items: DropdownItem[]`, keyboard-navigable, click-outside-to-close |
| `Feedback`    | `Spinner`, `LoadingState`, `EmptyState`, `ErrorState`, `Skeleton` |
| `Input`       | `label`, `hint`, `error`, `leftAddon`, `rightAddon` |
| `Modal`       | controlled, `width: sm/md/lg/xl`, Escape + click-outside to close |
| `PageHeader`  | `title`, `description`, `actions`, `breadcrumbs`   |
| `Select`      | native `<select>` styled, `label`/`hint`/`error`  |
| `Sidebar`     | persistent left nav with active-route highlight   |
| `StatusPill`  | `tone: neutral/success/warning/danger/info`, `dot` |
| `Table`       | generic over `T`, `columns: Column<T>[]`, optional `onRowClick`, `isLoading`, `emptyState` |
| `Tabs`        | controlled, panels are arbitrary JSX              |
| `Topbar`      | search + license pill + theme toggle + user menu  |
| `cn`          | the `clsx` re-export used everywhere               |

### Adding a new component

1. Drop a `Foo.tsx` into `src/lib/ui/`.
2. Use `forwardRef` for components that need to be referenced.
3. Use `cn` for class composition.
4. Use CSS variables (`bg-surface-2`, `text-ink-3`) for colors, never hardcoded hex.
5. Keep the component dumb — no TanStack Query, no router. Pages compose.

## 7. Routing

Code-based, in `src/router.tsx`. Adding a new page:

```ts
const fooRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/foo",
  component: FooPage,
});
// add to appLayoutRoute.addChildren([...])
```

Sidebar items live in `src/lib/ui/Sidebar.tsx` — add the link + icon there. Route-level data loading is not used in v0.3.0; pages do their own `useQuery`.

## 8. Auth flow

```
LoginPage  --POST /auth/login-->  store tokens in Zustand
            --navigate("/")-->  AppLayout (gated)

Any 401  -->  client tries /auth/refresh (single in-flight)
            -->  on success: store new tokens, retry original request
            -->  on failure: clear tokens, AppLayout redirects to /login
```

Tokens are persisted to `localStorage` via `zustand/persist` so a page refresh doesn't bounce the operator. Refresh runs transparently inside the API client.

## 9. Build & deploy

```bash
cd frontend
npm install
npm run build       # typechecks + bundles → dist/
cd ..

# copy dist to where the Go binary will look
mkdir -p /var/lib/orvixpanel/frontend
cp -r frontend/dist /var/lib/orvixpanel/frontend/

# build Go
go build -o bin/orvixpanel.linux ./cmd/orvixpanel
# install + restart
sudo systemctl restart orvixpanel  # or skip-systemd + nohup
```

`internal/web/frontend.go` looks for the dist in this order:

1. `$ORVIX_WEB_DIR` (operator override)
2. `./frontend/dist` (relative to the binary's CWD)
3. `../frontend/dist`
4. `./ui/dist`

The v0.3.0 install workflow puts the dist at `/var/lib/orvixpanel/frontend/dist` and cd's the binary into `/var/lib/orvixpanel`, so the second lookup wins.

### Why on-disk and not go:embed

`//go:embed` forbids `..` in the pattern, so embedding `frontend/dist` (which lives outside `internal/web/`) requires a build-time copy. On-disk serving is simpler and the DX is better: rebuild the frontend and refresh the browser without a Go rebuild.

## 10. Future modules — how DNS / Mail / SSL / etc. plug in

The architecture is intentionally one-shallow so future modules just **add a top-level nav item, a page, and a slice of `src/lib/api/`**. No redesign required.

| New module   | Files to add                                                                |
|--------------|-----------------------------------------------------------------------------|
| DNS zones    | `src/lib/api/dns.ts` + `src/pages/DnsZones.tsx` + sidebar entry + nav       |
| Mail        | `src/lib/api/mail.ts` + `src/pages/Mail.tsx` + sidebar entry + nav          |
| SSL         | `src/lib/api/ssl.ts` + `src/pages/Ssl.tsx` + sidebar entry + nav            |
| WAF         | `src/lib/api/waf.ts` + `src/pages/Waf.tsx` + sidebar entry + nav            |
| AI Guardian | `src/lib/api/ai.ts` + `src/pages/AiGuardian.tsx` + sidebar entry + nav      |
| Reseller    | `src/lib/api/reseller.ts` + `src/pages/Reseller.tsx` + sidebar entry + nav  |

Each new module's settings land as an additional tab in the existing `Settings.tsx` shell (which is already a `Tabs` component). No need to redesign the layout.

## 11. Non-goals for v0.3.0

- No fake charts (recharts / chart.js) — every metric is a real API value.
- No placeholder "coming soon" claims on the UI. Pages either call a real endpoint and render real data, or render a clean empty state with a 1-line description of what's needed for data to appear.
- No drag-drop file manager, no in-browser terminal, no real-time websocket dashboard. Those are out-of-scope until v0.6.x.
- No i18n. English only. (i18n design considerations: store strings in `lib/i18n/`, but the v0.3.0 commit doesn't ship this.)
