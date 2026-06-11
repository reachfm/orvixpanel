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
 *     /dns/zones                      ZonesList
 *     /dns/zones/:id                  ZoneDetail
 *     /dns/templates                  DNSTemplates
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
import { ZonesListPage } from "@/pages/ZonesList";
import { ZoneDetailPage } from "@/pages/ZoneDetail";
import { DNSTemplatesPage } from "@/pages/DNSTemplates";
import { CertificatesListPage } from "@/pages/CertificatesList";
import { CertificateDetailPage } from "@/pages/CertificateDetail";
import { BackupsListPage } from "@/pages/BackupsList";
import { BackupDetailPage } from "@/pages/BackupDetail";
import { MailDomainsListPage } from "@/pages/MailDomainsList";
import { MailboxesListPage } from "@/pages/MailboxesList";
import { MailAliasesPage } from "@/pages/MailAliases";
import { MailForwardersPage } from "@/pages/MailForwarders";
import { MailAuditLogPage } from "@/pages/MailAuditLog";
import { MailStatsPage } from "@/pages/MailStats";
import { MailDomainDetailPage } from "@/pages/MailDomainDetail";
import { UpdateCenterPage } from "@/pages/UpdateCenter";
import { NotificationSettingsPage } from "@/pages/NotificationSettings";
import { PlansListPage } from "@/pages/PlansList";
import { NewPlanPage } from "@/pages/NewPlanPage";
import { PlanDetailPage } from "@/pages/PlanDetail";

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

const updateCenterRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/updates",
  component: UpdateCenterPage,
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

const notificationSettingsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/settings/notifications",
  component: NotificationSettingsPage,
});

// DNS routes
const dnsZonesListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/dns/zones",
  component: ZonesListPage,
});

const dnsZoneDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/dns/zones/$id",
  component: ZoneDetailPage,
});

const dnsTemplatesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/dns/templates",
  component: DNSTemplatesPage,
});

// SSL routes
const sslCertificatesListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/ssl/certificates",
  component: CertificatesListPage,
});

const sslCertificateDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/ssl/certificates/$id",
  component: CertificateDetailPage,
});

// Backup routes
const backupsListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/backup",
  component: BackupsListPage,
});

const backupDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/backup/$id",
  component: BackupDetailPage,
});

// Mail routes
const mailDomainsListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/domains",
  component: MailDomainsListPage,
});

const mailDomainDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/domains/$id",
  component: MailDomainDetailPage,
});

const mailboxesListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/mailboxes",
  component: MailboxesListPage,
});

const mailAliasesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/aliases",
  component: MailAliasesPage,
});

const mailForwardersRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/forwarders",
  component: MailForwardersPage,
});

const mailAuditLogRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/audit",
  component: MailAuditLogPage,
});

const mailStatsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/mail/stats",
  component: MailStatsPage,
});

// Hosting Plans routes
const plansListRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/hosting/plans",
  component: PlansListPage,
});

const newPlanRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/hosting/plans/new",
  component: NewPlanPage,
});

const planDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: "/hosting/plans/$id",
  component: PlanDetailPage,
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
    updateCenterRoute,
    auditLogRoute,
    settingsRoute,
    notificationSettingsRoute,
    dnsZonesListRoute,
    dnsZoneDetailRoute,
    dnsTemplatesRoute,
    sslCertificatesListRoute,
    sslCertificateDetailRoute,
    backupsListRoute,
    backupDetailRoute,
    mailDomainsListRoute,
    mailDomainDetailRoute,
    mailboxesListRoute,
    mailAliasesRoute,
    mailForwardersRoute,
    mailAuditLogRoute,
    mailStatsRoute,
    plansListRoute,
    newPlanRoute,
    planDetailRoute,
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
