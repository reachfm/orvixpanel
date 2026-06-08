package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/api/v1"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/quota"
	"github.com/orvixpanel/orvixpanel/internal/rbac"
	"github.com/orvixpanel/orvixpanel/internal/ssl"
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
//
// v0.4.0 DNS Engine adds:
//   - /dns/zones/*            — DNS zone CRUD
//   - /dns/templates/*        — Zone template management
//   - /dns/validate           — Record validation
//   - /dns/lookup/:domain     — Local DNS lookup
//
// v0.5.0 SSL Engine adds:
//   - /ssl/*                 — SSL certificate management
//   - /ssl/certificates      — List/manage certificates
//   - /ssl/import            — Import existing certificates
//   - /ssl/health            — Certificate health scan
//   - /ssl/events            — SSL audit events
//   - /ssl/dashboard         — Dashboard statistics
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

	// DNS Engine (v0.4.0).
	dnsDeps := v1.DNSDeps{DB: d.DB, DNS: d.DNS, Audit: d.Audit}
	g.Get("/dns/zones", v1.ListZonesHandler(dnsDeps)).Name("dns.zone.read")
	g.Post("/dns/zones", v1.CreateZoneHandler(dnsDeps)).Name("dns.zone.write")
	g.Get("/dns/zones/:id", v1.GetZoneHandler(dnsDeps)).Name("dns.zone.read")
	g.Put("/dns/zones/:id", v1.UpdateZoneHandler(dnsDeps)).Name("dns.zone.write")
	g.Delete("/dns/zones/:id", v1.DeleteZoneHandler(dnsDeps)).Name("dns.zone.delete")
	g.Get("/dns/zones/:id/records", v1.ListRecordsHandler(dnsDeps)).Name("dns.record.read")
	g.Post("/dns/zones/:id/records", v1.CreateRecordHandler(dnsDeps)).Name("dns.record.write")
	g.Put("/dns/zones/:id/records/:recordId", v1.UpdateRecordHandler(dnsDeps)).Name("dns.record.write")
	g.Delete("/dns/zones/:id/records/:recordId", v1.DeleteRecordHandler(dnsDeps)).Name("dns.record.delete")
	g.Get("/dns/templates", v1.ListTemplatesHandler(dnsDeps)).Name("dns.template.read")
	g.Post("/dns/templates", v1.CreateTemplateHandler(dnsDeps)).Name("dns.template.write")
	g.Post("/dns/templates/:id/apply", v1.ApplyTemplateHandler(dnsDeps)).Name("dns.template.apply")
	g.Post("/dns/validate", v1.ValidateRecordHandler(dnsDeps)).Name("dns.validate")
	g.Get("/dns/lookup/:domain", v1.LookupHandler(dnsDeps)).Name("dns.lookup")

	// SSL Engine (v0.5.0) — Certificate lifecycle management.
	sslDeps := ssl.SSLDeps{DB: d.DB}
	sslGroup := g.Group("/ssl")

	// Certificate CRUD.
	sslGroup.Get("/certificates", ssl.ListCertificatesHandler(sslDeps)).Name("ssl.cert.read")
	sslGroup.Get("/certificates/:id", ssl.GetCertificateHandler(sslDeps)).Name("ssl.cert.read")
	sslGroup.Post("/certificates", ssl.IssueCertificateHandler(sslDeps, nil)).Name("ssl.cert.write")
	sslGroup.Post("/certificates/:id/renew", ssl.RenewCertificateHandler(sslDeps, nil)).Name("ssl.cert.write")
	sslGroup.Post("/certificates/:id/revoke", ssl.RevokeCertificateHandler(sslDeps, nil)).Name("ssl.cert.write")
	sslGroup.Delete("/certificates/:id", ssl.DeleteCertificateHandler(sslDeps, nil)).Name("ssl.cert.write")
	sslGroup.Post("/import", ssl.ImportCertificateHandler(sslDeps, nil)).Name("ssl.cert.write")

	// Health & events.
	sslGroup.Get("/health", ssl.GetHealthHandler(sslDeps)).Name("ssl.health.read")
	sslGroup.Get("/events", ssl.GetSSLEventsHandler(sslDeps)).Name("ssl.events.read")
	sslGroup.Get("/certificates/:id/events", ssl.GetCertificateEventsHandler(sslDeps)).Name("ssl.events.read")
	sslGroup.Get("/dashboard", ssl.GetDashboardStatsHandler(sslDeps)).Name("ssl.dashboard.read")
}
