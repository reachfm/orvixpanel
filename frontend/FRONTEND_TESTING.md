# Frontend Testing Guide

## Overview

The OrvixPanel frontend uses **Vitest** as its test runner, **@testing-library/react** for DOM testing, **@testing-library/user-event** for user interactions, and **jsdom** for browser environment simulation.

## Quick Start

```bash
# Install dependencies (includes test deps)
pnpm install

# Run all tests
pnpm test

# Run tests in watch mode
pnpm test:watch

# Run with coverage
pnpm test:coverage
```

## Test Structure

```
frontend/src/
├── lib/
│   ├── ui/
│   │   ├── ErrorBoundary.test.tsx     # Error boundary behavior
│   │   ├── Feedback.test.tsx          # Error state components
│   │   └── Notification.test.tsx      # Toast notification system
│   └── theme/
│       └── store.test.ts              # Theme state management
└── pages/
    └── *.test.tsx                     # Page-level tests (future)
```

## Test Suites

### ErrorBoundary.test.tsx (7 tests)
Tests the React error boundary component:
- Renders children when no error occurs
- Renders fallback when error occurs
- Shows error details in development mode
- Has refresh page button
- Has try again button
- Renders custom fallback when provided
- Recovers after trying again

### Feedback.test.tsx (19 tests)
Tests all error state components:
- **ErrorState**: Default/custom rendering, retry button
- **NetworkError**: Connection error display, try again button
- **AuthError**: Authentication required display, log in button
- **InlineError**: Inline error styling
- **FormError**: Error list rendering, empty state
- **WarningBanner**: Warning display with action
- **ToastError**: Toast error with dismiss
- **Spinner**: Loading spinner with size variants
- **LoadingState**: Loading state display
- **EmptyState**: Empty state with icon/action
- **Skeleton**: Skeleton loading placeholder

### Notification.test.tsx (18 tests)
Tests the notification system:
- **NotificationContainer**: Renders notifications, handles empty state
- **useNotification hook**: Basic notifications
- **NotificationPreferences**: Enabled/disabled, type filtering, duration
- **Auto-dismiss**: Timed removal of notifications

### store.test.ts (21 tests)
Tests the theme store:
- Light/dark mode toggle
- Theme persistence
- CSS class application
- Initial state

## Writing Tests

### Component Tests

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MyComponent } from "./MyComponent";

describe("MyComponent", () => {
  it("should render", () => {
    render(<MyComponent />);
    expect(screen.getByText("Hello")).toBeInTheDocument();
  });

  it("should call onClick when button is clicked", async () => {
    const onClick = vi.fn();
    render(<MyComponent onClick={onClick} />);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalled();
  });
});
```

### Store Tests

Test Zustand stores by getting/setting state directly:

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { useMyStore } from "./store";

describe("MyStore", () => {
  beforeEach(() => {
    // Reset store before each test
    useMyStore.setState({ items: [] });
  });

  it("should add items", () => {
    useMyStore.getState().addItem("test");
    expect(useMyStore.getState().items).toContain("test");
  });
});
```

### Testing User Interactions

Use `@testing-library/user-event` for realistic user interactions:

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

it("should submit form with user input", async () => {
  render(<LoginForm />);

  await userEvent.type(screen.getByLabelText("Email"), "user@example.com");
  await userEvent.type(screen.getByLabelText("Password"), "password123");
  await userEvent.click(screen.getByRole("button", { name: /submit/i }));

  expect(screen.getByText("Welcome!")).toBeInTheDocument();
});
```

### Testing with Timers

For components with timed behavior (like auto-dismiss notifications):

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

it("should auto-dismiss after duration", () => {
  const onDismiss = vi.fn();
  render(<Toast duration={5000} onDismiss={onDismiss} />);

  // Fast-forward time
  vi.advanceTimersByTime(5000);

  expect(onDismiss).toHaveBeenCalled();
});
```

### Testing with Persistence

For stores using Zustand's persist middleware:

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { useNotificationStore } from "./Notification";

// Use localStorage mock
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
vi.stubGlobal("localStorage", localStorageMock);

describe("NotificationStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useNotificationStore.setState({
      notifications: [],
      preferences: {
        enabled: true,
        soundEnabled: false,
        position: "bottom-right",
        defaultDuration: 5000,
        typesEnabled: { success: true, error: true, warning: true, info: true },
      },
    });
  });

  it("should persist preferences to localStorage", () => {
    useNotificationStore.getState().updatePreferences({ enabled: false });
    expect(localStorageMock.setItem).toHaveBeenCalled();
  });
});
```

## Best Practices

### DO

- Test behavior, not implementation details
- Use `screen` for queries (better debugging)
- Use `findBy*` queries for async content
- Reset state in `beforeEach`
- Mock external dependencies (API calls, timers)
- Use `userEvent` instead of `fireEvent` for realistic interactions
- Test the public API (rendered output, user actions)

### DON'T

- Test internal state directly
- Use `container.querySelector()` (use screen queries instead)
- Forget to reset mocks/timers between tests
- Test implementation details (class names, styles)
- Use `fireEvent` when `userEvent` is available

## Coverage

Coverage reports are generated in `frontend/coverage/`:

```bash
pnpm test:coverage
open coverage/index.html
```

Current coverage targets:
- Statements: 70%
- Branches: 70%
- Functions: 70%
- Lines: 70%

## Running Tests

```bash
# All tests
pnpm test

# Watch mode
pnpm test:watch

# Specific file
pnpm test -- src/lib/ui/Notification.test.tsx

# Specific test pattern
pnpm test -- --grep "Notification"
```

## Debugging

```bash
# Run a specific test file
npx vitest run src/lib/ui/Notification.test.tsx

# Run tests matching a pattern
npx vitest run --grep "Notification"

# Open UI mode (if available)
npx vitest --ui
```

## Troubleshooting

### "Cannot find module" errors

Ensure path aliases are configured in `tsconfig.json` and `vitest.config.ts`.

### "React state updates" warnings

Wrap state updates in `act()` from React or use `waitFor`.

### Timer issues

Always use `vi.useFakeTimers()` and clean up with `vi.useRealTimers()` in `afterEach`.

### Persistence mocks

For stores with Zustand persist middleware, mock localStorage before tests:

```typescript
const localStorageMock = {
  getItem: vi.fn().mockReturnValue(null),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
vi.stubGlobal("localStorage", localStorageMock);
```

## CI Integration

Tests run automatically on:
- Every pull request
- Push to `main` branch

Failed tests block merges.

## Mock Patterns

### Fetch Mocks

```typescript
global.fetch = vi.fn().mockResolvedValue({
  ok: true,
  json: () => Promise.resolve({ data: "test" }),
});
```

### LocalStorage Mocks

```typescript
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
vi.stubGlobal("localStorage", localStorageMock);
```

### Router Mocks

For components using TanStack Router, the router context needs to be provided. Check individual test files for setup patterns.