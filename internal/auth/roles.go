// Package auth — RBAC role constants. The matching permission map
// lives in api/middleware/rbac.go; this file only defines the
// canonical role names so the rest of the system can refer to them
// without typing string literals.
package auth

const (
	RoleRootAdmin     = "root_admin"
	RoleResellerAdmin = "reseller_admin"
	RoleResellerAgent = "reseller_agent"
	RoleAccountOwner  = "account_owner"
	RoleAccountDev    = "account_dev"
	RoleAccountViewer = "account_viewer"
	RoleMailAdmin     = "mail_admin"
	RoleDBAdmin       = "db_admin"
	RoleSecurityAdmin = "security_admin"
	RoleMonitor       = "monitor"
	RoleSupport       = "support"
	RoleBilling       = "billing"
)

// AllRoles is the canonical ordering used by the UI and validation.
var AllRoles = []string{
	RoleRootAdmin,
	RoleResellerAdmin,
	RoleResellerAgent,
	RoleAccountOwner,
	RoleAccountDev,
	RoleAccountViewer,
	RoleMailAdmin,
	RoleDBAdmin,
	RoleSecurityAdmin,
	RoleMonitor,
	RoleSupport,
	RoleBilling,
}

// IsValidRole reports whether s is one of the known roles.
func IsValidRole(s string) bool {
	for _, r := range AllRoles {
		if r == s {
			return true
		}
	}
	return false
}
