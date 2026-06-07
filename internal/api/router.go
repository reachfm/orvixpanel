package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/api/v1"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/quota"
	"github.com/orvixpanel/orvixpanel/internal/rbac"
	"github.com/orvixpanel/orvixpanel/internal/vault"
)

// registerV1 wires every /api/v1/* route.
//
// v0.1.0:
//   - /me         — who am I (returns the JWT claims)
//   - /admin/*    — license + audit-log view (root_admin only)
//
// v0.3.0 Enterprise Edition adds:
//   - /admin/license/*        — renewal info + license upload
//   - /admin/audit-log/search — filtered audit log query
//   - /admin/audit-log/export — CEF export over file/syslog
//   - /admin/roles/*          — custom RBAC role CRUD
//   - /admin/users/:id/role   — assign custom/built-in role to user
//   - /admin/api-keys/*       — long-lived API key CRUD
//   - /admin/tenants/:id/quotas — per-tenant resource limits
//   - /vault/secrets/*        — encrypted secrets store
//   - /me/quotas              — caller's own quota
func registerV1(g fiber.Router, d Deps) {
	g.Get("/me", v1.MeHandler).Name("auth.me")
	g.Get("/me/quotas", quota.MeHandler(d.Quota))

	// Vault — every authenticated user in the tenant can read/write
	// their tenant's secrets. (Tenant isolation enforced by the
	// claims.TenantID inside the handler.)
	vaultGrp := g.Group("/vault")
	vaultGrp.Get("/secrets", vault.ListHandler(vault.Deps{
		Vault: d.Vault, Audit: d.Audit, Quota: d.Quota,
	})).Name("vault.read")
	vaultGrp.Post("/secrets", vault.PutHandler(vault.Deps{
		Vault: d.Vault, Audit: d.Audit, Quota: d.Quota,
	})).Name("vault.write")
	vaultGrp.Get("/secrets/:name", vault.GetHandler(vault.Deps{
		Vault: d.Vault, Audit: d.Audit, Quota: d.Quota,
	})).Name("vault.read")
	vaultGrp.Get("/secrets/:name/value", vault.ReadHandler(vault.Deps{
		Vault: d.Vault, Audit: d.Audit, Quota: d.Quota,
	})).Name("vault.read")
	vaultGrp.Delete("/secrets/:name", vault.DeleteHandler(vault.Deps{
		Vault: d.Vault, Audit: d.Audit, Quota: d.Quota,
	})).Name("vault.write")

	// Admin group — root_admin gets the full set; other roles get
	// the bits their permissions allow (the middleware enforces).
	admin := g.Group("/admin", middleware.RequirePermission("admin", "read"))
	admin.Get("/system", v1.AdminSystem).Name("admin.system")
	admin.Get("/license", v1.AdminLicense).Name("admin.license.read")
	admin.Get("/license/renewal-info", v1.AdminLicenseRenewal(d.LicenseStore)).Name("admin.license.read")
	admin.Put("/license", v1.AdminLicenseUpload(d.LicenseStore)).Name("admin.license.write")
	admin.Get("/audit-log", v1.AdminAuditLog).Name("admin.audit.read")
	admin.Post("/audit-log/verify", v1.AdminAuditVerify).Name("admin.audit.verify")
	admin.Post("/audit-log/search", v1.AuditSearchHandler(d.Audit)).Name("admin.audit.read")
	admin.Post("/audit-log/export", v1.AuditExportHandler(d.Audit)).Name("admin.audit.export")

	// Custom RBAC roles.
	rbacDeps := rbac.Deps{Service: d.RBAC, DB: d.DB, Audit: d.Audit}
	admin.Get("/roles", rbac.ListHandler(rbacDeps)).Name("rbac.custom")
	admin.Post("/roles", rbac.CreateHandler(rbacDeps)).Name("rbac.custom")
	admin.Put("/roles/:name", rbac.UpdateHandler(rbacDeps)).Name("rbac.custom")
	admin.Delete("/roles/:name", rbac.DeleteHandler(rbacDeps)).Name("rbac.custom")
	admin.Post("/users/:id/role", rbac.AssignRoleHandler(rbacDeps)).Name("rbac.custom")

	// API keys.
	apikeyDeps := auth.APIKeyDeps{Keys: d.APIKeys, Audit: d.Audit}
	admin.Get("/api-keys", auth.ListAPIKeyHandler(apikeyDeps)).Name("apikey.read")
	admin.Post("/api-keys", auth.CreateAPIKeyHandler(apikeyDeps)).Name("apikey.write")
	admin.Delete("/api-keys/:id", auth.RevokeAPIKeyHandler(apikeyDeps)).Name("apikey.write")

	// Tenant quotas — root_admin only.
	quotaAdmin := g.Group("/admin/tenants", middleware.RequirePermission("admin", "*"))
	quotaAdmin.Get("/:id/quotas", quota.GetHandler(d.Quota)).Name("quota.tenant")
	quotaAdmin.Put("/:id/quotas", quota.PutHandler(d.Quota)).Name("quota.tenant")

	// Accounts (Phase 2 Core Hosting Engine).
	acctDeps := v1.AccountDeps{DB: d.DB, Hosting: d.Hosting, Audit: d.Audit}
	g.Post("/accounts", v1.CreateAccountHandler(acctDeps)).Name("account.create")
	g.Get("/accounts", v1.ListAccountsHandler(acctDeps)).Name("account.read")
	g.Get("/accounts/:id", v1.GetAccountHandler(acctDeps)).Name("account.read")
	g.Delete("/accounts/:id", v1.DeleteAccountHandler(acctDeps)).Name("account.delete")
	g.Post("/accounts/:id/suspend", v1.SuspendAccountHandler(acctDeps)).Name("account.update")
	g.Post("/accounts/:id/unsuspend", v1.UnsuspendAccountHandler(acctDeps)).Name("account.update")
	g.Get("/accounts/:id/usage", v1.AccountUsageHandler(acctDeps)).Name("account.read")

	// Domains (Phase 2).
	domDeps := v1.DomainDeps{DB: d.DB, Hosting: d.Hosting, Audit: d.Audit}
	g.Post("/accounts/:id/domains", v1.CreateDomainHandler(domDeps)).Name("domain.create")
	g.Get("/accounts/:id/domains", v1.ListDomainsHandler(domDeps)).Name("domain.read")
	g.Delete("/accounts/:id/domains/:domain", v1.DeleteDomainHandler(domDeps)).Name("domain.delete")

	// Deployments (v0.3.0 Enterprise UI). Read-only list of release
	// directories on disk, scoped to a single account.
	g.Get("/accounts/:id/deployments", v1.ListDeploymentsHandler(domDeps)).Name("deployment.read")
}
