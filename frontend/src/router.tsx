/**
 * TanStack Router config.
 *
 * The router is a *code-based* route tree (no file-system routing)
 * so we can keep the route definition colocated with the AppLayout.
 * Adding a new page = add a route object + a link from the sidebar.
 *
 * Routes:
 *   /login                            LoginPage (no chrome)
 *   /                                 AppLayout
 *     /                               Dashboard
 *     /accounts                       AccountsList
 *     /accounts/new                   NewAccount
 *     /accounts/:id                   AccountDetail
 *     /accounts/:id/domains/new       AddDomain
 *     /domains                        DomainsList
 *     /deployments                    DeploymentsList
 *     /system-health                  SystemHealth
 *     /audit-log                      AuditLog
 *     /settings                       Settings
 */

import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from "@tanstack/react-router";
import { AppLayout } from "@/lib/layout/AppLayout";
import { LoginPage } from "@/pages/Login";
import { DashboardPage } from "@/pages/Dashboard";
import { AccountsListPage } from "@/pages/AccountsList";
import { NewAccountPage } from "@/pages/NewAccount";
import { AccountDetailPage } from "@/pages/AccountDetail";
import { AddDomainPage } from "@/pages/AddDomain";
import { DomainsListPage } from "@/pages/DomainsList";
import { DeploymentsListPage } from "@/pages/DeploymentsList";
import { SystemHealthPage } from "@/pages/SystemHealth";
import { AuditLogPage } from "@/pages/AuditLog";
import { SettingsPage } from "@/pages/Settings";

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

// /login renders WITHOUT the AppLayout.
const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: LoginPage,
});

// Everything else lives inside the authenticated AppLayout.
const appLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "app",
  component: AppLayout,
});

const dashboardRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/",
  component: DashboardPage,
});

const accountsListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/accounts",
  component: AccountsListPage,
});

const newAccountRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/accounts/new",
  component: NewAccountPage,
});

const accountDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/accounts/$id",
  component: AccountDetailPage,
});

const addDomainRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/accounts/$id/domains/new",
  component: AddDomainPage,
});

const domainsListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/domains",
  component: DomainsListPage,
});

const deploymentsListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/deployments",
  component: DeploymentsListPage,
});

const systemHealthRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/system-health",
  component: SystemHealthPage,
});

const auditLogRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/audit-log",
  component: AuditLogPage,
});

const settingsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/settings",
  component: SettingsPage,
});

const routeTree = rootRoute.addChildren([
  loginRoute,
  appLayoutRoute.addChildren([
    dashboardRoute,
    accountsListRoute,
    newAccountRoute,
    accountDetailRoute,
    addDomainRoute,
    domainsListRoute,
    deploymentsListRoute,
    systemHealthRoute,
    auditLogRoute,
    settingsRoute,
  ]),
]);

export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
