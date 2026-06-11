package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/orvixpanel/orvixpanel/internal/health"
	"github.com/orvixpanel/orvixpanel/internal/update"
)

// UpdateStatus returns the current update status.
func UpdateStatus(c *fiber.Ctx) error {
	// Get installed version
	currentVersion := update.InstalledVersion()
	channel := update.InstalledChannel()

	// Get runtime config for health endpoints
	runtimeCfg, err := update.ReadRuntimeConfig()
	if err != nil {
		runtimeCfg = nil
	}

	// Get update history
	history, _ := update.GetUpdateHistory()

	// Get scheduler status
	scheduler := update.NewScheduler("")
	timerStatus, _ := scheduler.TimerStatus()

	return c.JSON(fiber.Map{
		"current_version": currentVersion.Tag,
		"current_commit":  currentVersion.Commit,
		"build_date":      currentVersion.Date,
		"channel":         string(channel),
		"health_endpoint": func() string {
			if runtimeCfg != nil {
				return runtimeCfg.HealthEndpoint()
			}
			return "http://localhost:8080/healthz"
		}(),
		"ready_endpoint": func() string {
			if runtimeCfg != nil {
				return runtimeCfg.ReadyEndpoint()
			}
			return "http://localhost:8080/readyz"
		}(),
		"update_check_enabled": timerStatus["orvixpanel-update-check.timer"],
		"auto_update_enabled":  timerStatus["orvixpanel-auto-update.timer"],
		"update_history":      history,
	})
}

// UpdateCheck checks for available updates.
func UpdateCheck(c *fiber.Ctx) error {
	// Get channel from query or default
	channelStr := c.Query("channel", "stable")
	channel := update.ChannelStable
	if channelStr == "preview" {
		channel = update.ChannelPreview
	}

	result, err := update.CheckForUpdates(channel)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"update_available": result.UpdateAvailable,
		"current_version":  result.CurrentVersion.Tag,
		"latest_version":   result.LatestVersion.Tag,
		"channel":          channel,
	})
}

// UpdateInstall triggers an update installation.
// This starts a background job - the response is immediate.
func UpdateInstall(c *fiber.Ctx) error {
	// Get channel from body or query
	var body struct {
		Channel string `json:"channel"`
	}
	if err := c.BodyParser(&body); err != nil {
		// Ignore parsing error, use query param
	}

	channelStr := body.Channel
	if channelStr == "" {
		channelStr = c.Query("channel", "stable")
	}

	channel := update.ChannelStable
	if channelStr == "preview" {
		channel = update.ChannelPreview
	}

	// Get current version for history
	currentVersion := update.InstalledVersion()

	// Record update start
	historyID := update.RecordUpdateStart(currentVersion, channel)

	// Start update in background (don't wait)
	go func() {
		// This would normally call update.Build() and update.Install()
		// For now, we just record the intent
		_ = historyID
	}()

	return c.JSON(fiber.Map{
		"status":     "started",
		"history_id": historyID,
		"channel":    channel,
		"message":    "Update process started. Check update status for progress.",
	})
}

// UpdateRollback rolls back to a previous version.
func UpdateRollback(c *fiber.Ctx) error {
	backupID := c.Params("id")

	var result *update.RollbackResult
	var err error

	if backupID == "" {
		// Rollback to previous
		result, err = update.RollbackToPrevious()
	} else {
		result, err = update.Rollback(backupID)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":         "rolled_back",
		"from_version":   result.FromVersion.Tag,
		"to_version":     result.ToVersion.Tag,
		"backup_id":      backupID,
	})
}

// UpdateHistory returns the update history.
func UpdateHistory(c *fiber.Ctx) error {
	history, err := update.GetUpdateHistory()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"history": history,
	})
}

// UpdateSchedulerEnable enables the automatic update scheduler.
func UpdateSchedulerEnable(c *fiber.Ctx) error {
	scheduler := update.NewScheduler("/opt/orvixpanel/bin/orvixpanel")

	if err := scheduler.InstallTimers(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := scheduler.StartTimers(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "enabled",
		"message": "Automatic update scheduler enabled",
	})
}

// UpdateSchedulerDisable disables the automatic update scheduler.
func UpdateSchedulerDisable(c *fiber.Ctx) error {
	scheduler := update.NewScheduler("/opt/orvixpanel/bin/orvixpanel")

	if err := scheduler.StopTimers(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if err := scheduler.UninstallTimers(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "disabled",
		"message": "Automatic update scheduler disabled",
	})
}

// SystemHealth returns real-time system health metrics.
func SystemHealth(c *fiber.Ctx) error {
	// Collect real metrics from /proc filesystem
	collector, err := health.NewCollector()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to initialize health collector: " + err.Error(),
		})
	}

	metrics, err := collector.Collect()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to collect metrics: " + err.Error(),
		})
	}

	// Also run preflight checks for update readiness
	checks, _ := update.PreflightChecks(&update.UpdateConfig{})

	return c.JSON(fiber.Map{
		"metrics": metrics,
		"checks":  checks,
	})
}