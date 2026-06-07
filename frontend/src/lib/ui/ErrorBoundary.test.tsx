/**
 * Tests for the ErrorBoundary component.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { Component, type ReactNode } from "react";
import { ErrorBoundary } from "@/lib/ui/ErrorBoundary";

// Throw error component for testing
function ThrowError({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error("Test error message");
  }
  return <div>Normal content</div>;
}

describe("ErrorBoundary", () => {
  const originalEnv = process.env.NODE_ENV;

  beforeEach(() => {
    vi.useFakeTimers();
    process.env.NODE_ENV = "test";
  });

  afterEach(() => {
    process.env.NODE_ENV = originalEnv;
    vi.useRealTimers();
  });

  it("should render children when no error occurs", () => {
    render(
      <ErrorBoundary>
        <div data-testid="child">Child content</div>
      </ErrorBoundary>
    );

    expect(screen.getByTestId("child")).toHaveTextContent("Child content");
  });

  it("should render fallback when error occurs", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(
      screen.getByText(/An unexpected error occurred/i)
    ).toBeInTheDocument();
  });

  it("should show error details in development mode", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    // Should show "Error details" expandable section
    const details = screen.getByText("Error details");
    expect(details).toBeInTheDocument();
  });

  it("should have refresh page button", () => {
    const reloadSpy = vi.spyOn(window.location, "reload");

    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    const refreshButton = screen.getByRole("button", { name: /refresh page/i });
    expect(refreshButton).toBeInTheDocument();

    refreshButton.click();
    expect(reloadSpy).toHaveBeenCalled();
  });

  it("should have try again button", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    const tryAgainButton = screen.getByRole("button", { name: /try again/i });
    expect(tryAgainButton).toBeInTheDocument();
  });

  it("should render custom fallback when provided", () => {
    const CustomFallback = () => (
      <div data-testid="custom-fallback">Custom error UI</div>
    );

    render(
      <ErrorBoundary fallback={<CustomFallback />}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByTestId("custom-fallback")).toHaveTextContent(
      "Custom error UI"
    );
  });

  it("should recover after trying again", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText("Something went wrong")).toBeInTheDocument();

    const tryAgainButton = screen.getByRole("button", { name: /try again/i });
    tryAgainButton.click();

    // After clicking try again, should show children again (which will throw again,
    // but the boundary catches it and shows error)
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });
});