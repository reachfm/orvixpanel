# OrvixPanel Frontend

Enterprise Admin UI for the OrvixPanel hosting control panel.

## Tech Stack

- **React 18** + **TypeScript 5** — Core framework
- **Vite 5** — Build tool and dev server
- **Tailwind CSS 3** — Utility-first CSS with CSS variable design tokens
- **TanStack Query 5** — Server state management
- **TanStack Router 1** — Type-safe routing
- **Zustand 4** — Client state management (theme, notifications)
- **Vitest 2** — Test runner
- **@testing-library/react** — Component testing

## Features (v0.3.1)

- Dark/Light mode with persistent preference
- Responsive sidebar navigation
- Real-time system health monitoring
- Account and domain management
- Audit log with timeline view and CSV export
- Toast notification system with preferences
- Comprehensive error handling components
- 65+ passing tests

## Getting Started

```bash
# Install dependencies
pnpm install

# Start dev server
pnpm dev

# Type check
pnpm typecheck

# Build for production
pnpm build

# Run tests
pnpm test

# Run tests in watch mode
pnpm test:watch

# Run with coverage
pnpm test:coverage
```

## Project Structure

```
frontend/
├── src/
│   ├── components/         # Reusable UI components
│   │   └── ui/             # Base components (Button, Card, Modal, etc.)
│   ├── lib/
│   │   ├── api/            # API client functions
│   │   ├── query/          # TanStack Query hooks and keys
│   │   ├── ui/             # UI components (Feedback, Notification, etc.)
│   │   ├── utils/          # Utility functions
│   │   └── theme/          # Theme store (Zustand)
│   ├── pages/              # Route-level page components
│   ├── routes/             # TanStack Router configuration
│   └── styles/             # Global styles (Tailwind + CSS variables)
├── public/                 # Static assets
└── dist/                   # Production build output
```

## Design System

### Color Tokens

CSS variables defined in `src/styles/globals.css`:

| Token | Light Mode | Dark Mode | Usage |
|-------|-----------|-----------|-------|
| `--ink-1` to `--ink-6` | Various grays | Inverted | Text hierarchy |
| `--surface-1` to `--surface-3` | Light backgrounds | Dark backgrounds | Card/panel backgrounds |
| `--brand-500` | Primary blue | Primary blue | Brand color |
| `--success` | Green | Green | Success states |
| `--danger` | Red | Red | Error states |
| `--warning` | Amber | Amber | Warning states |
| `--info` | Blue | Blue | Info states |

### Typography

- **Sans**: System font stack (SF Pro, Segoe UI, Roboto)
- **Mono**: JetBrains Mono, Fira Code, monospace

### Spacing

Uses Tailwind's 4px grid system (1 unit = 0.25rem)

## Testing

See [FRONTEND_TESTING.md](./FRONTEND_TESTING.md) for comprehensive testing guide.

### Test Files (4 files, 65 tests)

- `src/lib/ui/ErrorBoundary.test.tsx` — Error boundary behavior
- `src/lib/ui/Feedback.test.tsx` — Error state components
- `src/lib/ui/Notification.test.tsx` — Toast notification system
- `src/lib/theme/store.test.ts` — Theme state management

### Running Tests

```bash
# All tests
pnpm test

# Watch mode
pnpm test:watch

# Coverage report
pnpm test:coverage
```

## Pages

| Route | Component | Description |
|-------|-----------|-------------|
| `/` | Dashboard | Enterprise overview with stats and activity |
| `/accounts` | Accounts | Account management with CRUD |
| `/accounts/:id` | AccountDetail | Single account with domains/deployments tabs |
| `/domains` | Domains | Domain listing with search/filter |
| `/audit-log` | AuditLog | Audit trail with timeline view and export |
| `/system-health` | SystemHealth | Live health metrics and checks |
| `/settings/notifications` | NotificationSettings | Notification preferences |

## API Integration

The frontend communicates with the OrvixPanel backend via REST API:

- Authentication: JWT tokens (15m access, 30d refresh)
- Base URL: Configurable via environment
- Error handling: Consistent error states across all API calls

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## License

Part of OrvixPanel commercial open-core (BSL 1.1)