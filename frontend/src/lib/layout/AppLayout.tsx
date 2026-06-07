/**
 * AppLayout — the chrome shared by every authenticated page.
 *
 * Composed of:
 *   - Sidebar (left, persistent)
 *   - Topbar   (top, persistent) with breadcrumbs / search / status /
 *              license pill / theme / notifications / user menu
 *   - Outlet   (the page content)
 *
 * Route Guard:
 *   - Checks for valid access token on every navigation
 *   - Redirects unauthenticated users to /login
 *   - Warns users with expiring tokens (5 minutes before expiry)
 *   - Handles session expiration gracefully with notification
 *
 * The login page is rendered WITHOUT this chrome (it has
 * its own minimal layout).
 */

import { Outlet, useRouterState } from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { Sidebar } from "@/lib/ui/Sidebar";
import { Topbar } from "@/lib/ui/Topbar";
import { Breadcrumbs, useBreadcrumbsFromPath } from "@/lib/ui/Breadcrumbs";
import { useAuthStore } from "@/lib/auth/store";
import { useNotification } from "@/lib/ui/Notification";

const labelMap: Record<string, string> = {
  accounts: "Accounts",
  domains: "Domains",
  deployments: "Deployments",
  "system-health": "System Health",
  "audit-log": "Audit Log",
  settings: "Settings",
  new: "New",
  usage: "Usage",
};

// JWT token expiry check interval (30 seconds)
const SESSION_CHECK_INTERVAL = 30_000;
// Warn user 5 minutes before token expires
const EXPIRY_WARNING_MS = 5 * 60 * 1_000;

function parseJWTExpiry(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.exp ? payload.exp * 1_000 : null;
  } catch {
    return null;
  }
}

export function AppLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const accessToken = useAuthStore((s) => s.accessToken);
  const user = useAuthStore((s) => s.user);
  const clear = useAuthStore((s) => s.clear);
  const notify = useNotification();

  const hasWarnedRef = useRef(false);
  const expiryWarningShownRef = useRef(false);

  // Session expiry monitoring
  useEffect(() => {
    if (!accessToken) return;

    const checkSession = () => {
      const expiry = parseJWTExpiry(accessToken);
      if (!expiry) return;

      const now = Date.now();
      const timeUntilExpiry = expiry - now;

      // Token is expired or about to expire
      if (timeUntilExpiry <= 0) {
        clear();
        notify("warning", "Session Expired", "Your session has expired. Please log in again.");
        void import("@/router").then((m) => m.router.navigate({ to: "/login" }));
        return;
      }

      // Warn user 5 minutes before expiry (only once)
      if (timeUntilExpiry <= EXPIRY_WARNING_MS && !expiryWarningShownRef.current) {
        expiryWarningShownRef.current = true;
        const minutesLeft = Math.ceil(timeUntilExpiry / 60_000);
        notify("warning", "Session Expiring Soon", `Your session will expire in ${minutesLeft} minute${minutesLeft !== 1 ? "s" : ""}. Save your work.`);
      }
    };

    // Check immediately
    checkSession();

    // Set up interval for periodic checks
    const intervalId = setInterval(checkSession, SESSION_CHECK_INTERVAL);
    return () => clearInterval(intervalId);
  }, [accessToken, clear, notify]);

  // Auth guard: redirect to login if not authenticated.
  useEffect(() => {
    if (!accessToken && !pathname.startsWith("/login")) {
      // Only warn once to avoid notification spam
      if (!hasWarnedRef.current) {
        hasWarnedRef.current = true;
      }
      void import("@/router").then((m) => m.router.navigate({ to: "/login" }));
    } else if (accessToken) {
      hasWarnedRef.current = false;
      expiryWarningShownRef.current = false;
    }
  }, [accessToken, pathname]);

  if (!accessToken) {
    return null; // The effect above will redirect.
  }

  const crumbs = useBreadcrumbsFromPath(pathname, labelMap);
  // Inject the user email as a terminal crumb only on the dashboard.
  if (pathname === "/" && user?.email) {
    crumbs.push({ label: user.email });
  }

  return (
    <div className="flex h-full">
      <Sidebar />
      <div className="flex h-full min-w-0 flex-1 flex-col">
        <Topbar breadcrumbs={<Breadcrumbs items={crumbs} />} />
        <main className="flex-1 overflow-y-auto p-6 scroll-surface">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
