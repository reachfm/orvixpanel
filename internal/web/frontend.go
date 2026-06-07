// Package web serves the OrvixPanel Enterprise UI. v0.3.0 ships a
// built frontend in ../frontend/dist; this file serves it directly
// from the on-disk directory.
//
// Why disk and not //go:embed?
//   - go:embed restricts the pattern to the same package directory
//     (no .. in the path). The frontend lives at frontend/dist and
//     the binary is at cmd/orvixpanel, so the embed would have to
//     copy the dist into the Go tree at build time — fragile.
//   - Serving from disk means an operator can rebuild the frontend
//     and refresh the browser without a Go rebuild. The dev
//     workflow is: cd frontend && npm run dev.
//
// Build flow (run by the operator once per release):
//
//	cd frontend
//	npm install
//	npm run build      # writes frontend/dist
//	cd ..
//	go build ./cmd/orvixpanel
//
// At runtime the binary serves ./frontend/dist relative to its
// working directory. The install.sh copies the dist next to the
// binary at /opt/orvixpanel/ui/dist and cd's into /var/lib/orvixpanel
// before exec; the v0.3.0 install steps ensure that path is correct.
package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RegisterRoutes mounts the UI onto the given Fiber app. Looks for
// the dist directory in this order:
//
//  1. $ORVIX_WEB_DIR environment variable (operator override)
//  2. ./frontend/dist  (relative to the binary's CWD)
//  3. ../frontend/dist (relative to the binary, for `go run` from
//     the repo root)
//
// Falls back to a clear 404 page if no dist is found — the API and
// health endpoints keep working.
func RegisterRoutes(app *fiber.App) {
	dir, ok := findDistDir()
	if !ok {
		log.Warn().Str("dir", dirOrEmpty(dir)).Msg("frontend dist not found; UI will return 404 (API + healthz still work)")
		app.Get("/*", func(c *fiber.Ctx) error {
			return c.Status(http.StatusNotFound).SendString(
				"OrvixPanel UI not built. Run `cd frontend && npm install && npm run build` " +
					"or set $ORVIX_WEB_DIR to the built dist directory.",
			)
		})
		return
	}
	log.Info().Str("dir", dir).Msg("serving OrvixPanel UI")

	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}

	// /assets/* — long-cache immutable
	app.Use("/assets", func(c *fiber.Ctx) error {
		rel := strings.TrimPrefix(c.Path(), "/")
		full := filepath.Join(abs, filepath.FromSlash(rel))
		if !isInside(abs, full) {
			return c.Status(http.StatusBadRequest).SendString("bad path")
		}
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			c.Set("Cache-Control", "public, max-age=31536000, immutable")
			return c.SendFile(full)
		}
		return c.Status(http.StatusNotFound).SendString("not found")
	})

	// /favicon.svg
	app.Get("/favicon.svg", func(c *fiber.Ctx) error {
		return c.SendFile(filepath.Join(abs, "favicon.svg"))
	})

	// Catch-all SPA handler. Tries the literal file, then index.html.
	app.Get("/*", func(c *fiber.Ctx) error {
		rel := strings.TrimPrefix(c.Path(), "/")
		if rel == "" {
			rel = "index.html"
		}
		full := filepath.Join(abs, filepath.FromSlash(rel))
		if !isInside(abs, full) {
			return c.Status(http.StatusBadRequest).SendString("bad path")
		}
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			c.Set("Cache-Control", "no-cache")
			return c.SendFile(full)
		}
		// Fall back to index.html for client-side routing.
		idx := filepath.Join(abs, "index.html")
		if _, err := os.Stat(idx); err == nil {
			c.Set("Cache-Control", "no-cache")
			return c.SendFile(idx)
		}
		return c.Status(http.StatusNotFound).SendString("OrvixPanel UI: index.html missing in dist")
	})
}

func findDistDir() (string, bool) {
	if v := os.Getenv("ORVIX_WEB_DIR"); v != "" {
		if st, err := os.Stat(v); err == nil && st.IsDir() {
			return v, true
		}
	}
	for _, candidate := range []string{
		"./frontend/dist",
		"../frontend/dist",
		"./ui/dist",
	} {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func dirOrEmpty(d string) string { if d == "" { return "(unset)" }; return d }

// isInside reports whether child is lexically inside parent. Both
// paths are cleaned before comparison.
func isInside(parent, child string) bool {
	cleanP := filepath.Clean(parent)
	cleanC := filepath.Clean(child)
	rel, err := filepath.Rel(cleanP, cleanC)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") {
		return false
	}
	return true
}
