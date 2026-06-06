package v1

import "github.com/gofiber/fiber/v2"

// NotImplemented is the placeholder for routes the spec calls for
// but v1.0 doesn't ship. Returns 501 with a stable `phase_X_pending`
// error code so the frontend can render a "Coming soon" message and
// the audit chain still records the denial.
func NotImplemented(name string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotImplemented, "phase_"+name+"_pending")
	}
}
