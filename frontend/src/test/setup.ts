/**
 * Vitest test setup.
 *
 * Configures jsdom environment and extends expect with
 * @testing-library/jest-dom matchers for DOM assertions.
 */

import "@testing-library/jest-dom";

// Mock window.matchMedia for theme tests
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Mock localStorage for Zustand persist middleware
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
vi.stubGlobal("localStorage", localStorageMock);

// Mock fetch globally
global.fetch = vi.fn();

// Mock window.location
Object.defineProperty(window, "location", {
  value: {
    href: "http://localhost:3000",
    pathname: "/",
    search: "",
    hash: "",
    host: "localhost:3000",
    hostname: "localhost",
    port: "3000",
    protocol: "http:",
    reload: vi.fn(),
    replace: vi.fn(),
    assign: vi.fn(),
  },
  writable: true,
});

// Suppress console errors in tests (optional - comment out for debugging)
// vi.spyOn(console, "error").mockImplementation(() => {});
// vi.spyOn(console, "warn").mockImplementation(() => {});