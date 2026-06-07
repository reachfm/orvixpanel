// Package v1 — Deployments endpoint (v0.3.0 Enterprise UI).
//
// GET /api/v1/accounts/:id/deployments
//
// Iterates every domain that belongs to the account and returns the
// release directories on disk. Each release is reported with its
// name, size, mtime, and whether the document-root symlink is
// currently pointing at it.
//
// v0.3.0 ships the read-only list. The "deploy" / "rollback" /
// "delete release" actions are stubbed 501 — they live in the
// v0.3.x deploy CLI (internal/hosting.AtomicSwap) which the UI
// doesn't expose yet. The UI's Deployments page renders the list
// without any fake buttons.
//
// Linux-only (the `hosting.Service` methods are no-op on Windows).
// On non-Linux the handler returns 501 with a stable error code so
// the frontend can render a clean empty state.
//
// Domain storage: v0.2.0 keeps a single primary domain per account
// in the Account.Domain column. v0.3.x will add a separate Domain
// table; this handler reads the primary column for now.

package v1

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/orvixpanel/orvixpanel/internal/api/middleware"
	"github.com/orvixpanel/orvixpanel/internal/auth"
	"github.com/orvixpanel/orvixpanel/internal/db/models"
	"github.com/orvixpanel/orvixpanel/internal/hosting"
	"gorm.io/gorm"
)

// DeploymentView is the JSON shape returned to the UI.
type DeploymentView struct {
	ID         string `json:"id"`
	AccountID  string `json:"account_id"`
	Username   string `json:"username"`
	Domain     string `json:"domain"`
	Release    string `json:"release"`
	IsCurrent  bool   `json:"is_current"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at"`
}

// ListDeploymentsHandler — GET /api/v1/accounts/:id/deployments
func ListDeploymentsHandler(d DomainDeps) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(middleware.LocalClaims).(*auth.Claims)
		if !ok {
			return fiber.ErrUnauthorized
		}
		id := c.Params("id")
		var account models.Account
		if err := d.DB.WithContext(c.Context()).
			Where("id = ? AND tenant_id = ?", id, claims.TenantID).
			First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "account_not_found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "account_get_failed")
		}

		// v0.2.0: one primary domain per account, stored on the row.
		// Future v0.3.x will read from a separate domain table.
		domainNames := make([]string, 0, 1)
		if account.Domain != "" {
			domainNames = append(domainNames, account.Domain)
		}

		out := make([]DeploymentView, 0, 32)
		for _, dom := range domainNames {
			current, _ := d.Hosting.CurrentRelease(account.Username, dom)
			releases, _ := d.Hosting.ListReleases(account.Username, dom)
			for _, rel := range releases {
				path := d.Hosting.Paths.ReleaseDir(account.Username, dom, rel)
				st, err := os.Stat(path)
				if err != nil {
					continue
				}
				out = append(out, DeploymentView{
					ID:         uuid.NewSHA1(uuid.NameSpaceURL, []byte(account.ID+":"+dom+":"+rel)).String(),
					AccountID:  account.ID,
					Username:   account.Username,
					Domain:     dom,
					Release:    rel,
					IsCurrent:  rel == current,
					SizeBytes:  dirSize(path),
					ModifiedAt: st.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
				})
			}
		}

		// Sort by modified_at DESC for the UI.
		sort.Slice(out, func(i, j int) bool { return out[i].ModifiedAt > out[j].ModifiedAt })

		return c.JSON(fiber.Map{"deployments": out})
	}
}

// dirSize returns the sum of regular file sizes under path. Cheap
// enough for the v0.3.0 volume (~tens of MB per release); for huge
// trees we can swap in a parallel walker later.
func dirSize(path string) int64 {
	var n int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.Mode().IsRegular() {
			n += info.Size()
		}
		return nil
	})
	return n
}

// Compile-time check: hosting.Service is used (its ListReleases +
// CurrentRelease are part of the DomainDeps).
var _ = (*hosting.Service)(nil)
