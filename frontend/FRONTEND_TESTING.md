# Frontend Testing Guide

## Overview

The OrvixPanel frontend uses **Vitest** as its test runner, **Testing Library** for DOM testing, and **jsdom** for browser environment simulation.

## Quick Start

```bash
# Install dependencies (includes test deps)
npm install

# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run with coverage
npm run test:coverage
```

## Test Structure

```
frontend/src/
├── test/
│   └── setup.ts          # Global test setup and mocks
├── lib/
│   ├── ui/
│   │   ├── ErrorBoundary.test.tsx
│   │   └── Notification.test.tsx
│   └── theme/
│       └── store.test.ts
└── pages/
    └── *.test.tsx         # Page-level tests
```

## Writing Tests

### Component Tests

Use `@testing-library/react` for testing React components:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MyComponent } from "./MyComponent";

describe("MyComponent", () => {
  it("should render", () => {
    render(<MyComponent />);
    expect(screen.getByText("Hello")).toBeInTheDocument();
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

### Async Tests

Use `waitFor` from Testing Library for async operations:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { fetchData } from "./api";

it("should display data after loading", async () => {
  render(<DataComponent />);

  await waitFor(() => {
    expect(screen.getByText("Loaded Data")).toBeInTheDocument();
  });
});
```

### Mocking

#### API Mocks

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MyComponent } from "./MyComponent";

// Mock the API module
vi.mock("@/lib/api/myapi", () => ({
  fetchData: vi.fn().mockResolvedValue({ name: "Test" }),
}));

it("should display fetched data", async () => {
  render(<MyComponent />);
  expect(await screen.findByText("Test")).toBeInTheDocument();
});
```

#### Timer Mocks

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

beforeEach(() => {
  vi.useFakeTimers();
});

it("should auto-dismiss after duration", () => {
  const onDismiss = vi.fn();
  render(<Toast duration={5000} onDismiss={onDismiss} />);

  // Fast-forward time
  vi.advanceTimersByTime(5000);

  expect(onDismiss).toHaveBeenCalled();
});
```

## Best Practices

### DO

- Test behavior, not implementation details
- Use `screen` for queries (better debugging)
- Use `findBy*` queries for async content
- Reset state in `beforeEach`
- Mock external dependencies (API calls, timers)

### DON'T

- Test internal state directly
- Use `container.querySelector()` (use screen queries instead)
- Forget to reset mocks/timers between tests
- Test implementation details (class names, styles)

## Coverage

Coverage reports are generated in `frontend/coverage/`:

```bash
npm run test:coverage
open coverage/index.html
```

Minimum coverage targets:
- Statements: 70%
- Branches: 70%
- Functions: 70%
- Lines: 70%

## CI Integration

Tests run automatically on:
- Every pull request
- Push to `main` branch

Failed tests block merges.

## Debugging

```bash
# Run a specific test file
npx vitest run src/lib/ui/Notification.test.tsx

# Run tests matching a pattern
npx vitest run --grep "Notification"

# Open UI mode
npx vitest --ui
```

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

For components using TanStack Router, mock the router context:

```tsx
import { RouterContext } from "@tanstack/react-router";

const renderWithRouter = (ui: ReactElement) => {
  const memoryHistory = createMemoryHistory();

  return render(
    <RouterContext.Provider
      value={{
        router: mockRouter,
        navigate: vi.fn(),
        location: { pathname: "/" },
      }}
    >
      {ui}
    </RouterContext.Provider>
  );
};
```

## Running Specific Test Suites

```bash
# Unit tests only
npm test -- --testPathPattern="store|Notification"

# Integration tests
npm test -- --testPathPattern="pages"

# All tests
npm test
```

## Troubleshooting

### "Cannot find module" errors

Ensure path aliases are configured in `tsconfig.json` and `vitest.config.ts`.

### "React state updates" warnings

Wrap state updates in `act()` from React or use `waitFor`.

### Timer issues

Always use `vi.useFakeTimers()` and clean up with `vi.useRealTimers()` in `afterEach`.

### Jest compatibility

If using Jest matchers, import `@testing-library/jest-dom` in setup file.