/**
 * AppLayout — the chrome shared by every authenticated page.
 *
 * Composed of:
 *   - Sidebar (left, persistent)
 *   - Topbar   (top, persistent) with breadcrumbs / search / status /
 *              license pill / theme / notifications / user menu
 *   - Outlet   (the page content)
 *
 * AuthGuard wraps the whole tree and bounces unauthenticated users
 * to /login. The login page is rendered WITHOUT this chrome (it has
 * its own minimal layout).
 */

import { Outlet, useRouterState } from "@tanstack/react-router";
import { useEffect } from "react";
import { Sidebar } from "@/lib/ui/Sidebar";
import { Topbar } from "@/lib/ui/Topbar";
import { Breadcrumbs, useBreadcrumbsFromPath } from "@/lib/ui/Breadcrumbs";
import { useAuthStore } from "@/lib/auth/store";

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

export function AppLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const accessToken = useAuthStore((s) => s.accessToken);
  const user = useAuthStore((s) => s.user);

  // If the user is not authenticated, push to /login.
  // We use an effect so we don't update state during render.
  useEffect(() => {
    if (!accessToken && !pathname.startsWith("/login")) {
      // Lazy import the router to avoid a cycle.
      void import("@/router").then((m) => m.router.navigate({ to: "/login" }));
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
