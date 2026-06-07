/**
 * Error Boundary - Catches React errors in the component tree.
 * Shows a user-friendly error page instead of a blank screen.
 */

import { Component, type ReactNode, type ErrorInfo } from "react";
import { Button } from "@/lib/ui/Button";
import { Card } from "@/lib/ui/Card";

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("ErrorBoundary caught an error:", error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="flex min-h-[60vh] items-center justify-center p-6">
          <Card className="max-w-md text-center">
            <div className="mb-4 grid h-12 w-12 place-items-center rounded-full bg-danger/10 mx-auto">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className="h-6 w-6 text-danger"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <h2 className="text-lg font-semibold text-ink-1">Something went wrong</h2>
            <p className="mt-2 text-sm text-ink-3">
              An unexpected error occurred. Please try refreshing the page.
            </p>
            {import.meta.env.DEV && this.state.error && (
              <details className="mt-4 text-left">
                <summary className="cursor-pointer text-sm text-danger">Error details</summary>
                <pre className="mt-2 overflow-x-auto rounded bg-surface-2 p-2 text-xs text-ink-2">
                  {this.state.error.message}
                  {"\n\n"}
                  {this.state.error.stack}
                </pre>
              </details>
            )}
            <div className="mt-6 flex justify-center gap-3">
              <Button variant="secondary" onClick={() => window.location.reload()}>
                Refresh page
              </Button>
              <Button variant="primary" onClick={this.handleReset}>
                Try again
              </Button>
            </div>
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}