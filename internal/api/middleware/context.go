// Package middleware contains the Fiber v2 middleware used by the
// OrvixPanel API server.
//
// v1.0 keeps the middleware surface small — every middleware in this
// package compiles and is wired into the router. Some are stubbed
// (e.g. TenantMiddleware does a no-op lookup because v1.0 only has
// one tenant) and documented as such.
package middleware

// Local user-storage keys (Fiber v2 Locals is map[string]any).
const (
	LocalClaims = "claims"
)
