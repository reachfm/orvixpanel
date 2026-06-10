/**
 * Notification Settings page.
 * Allows users to configure their notification preferences.
 * v0.3.1 Phase F: Notification preferences page.
 */

import { useState } from "react";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Select } from "@/lib/ui/Select";
import { useNotificationStore, type NotificationPreferences } from "@/lib/ui/Notification";
import { useNotification } from "@/lib/ui/Notification";

export function NotificationSettingsPage() {
  const preferences = useNotificationStore((s) => s.preferences);
  const updatePreferences = useNotificationStore((s) => s.updatePreferences);
  const resetPreferences = useNotificationStore((s) => s.resetPreferences);
  const notify = useNotification();

  const [localPrefs, setLocalPrefs] = useState<NotificationPreferences>(preferences);
  const [hasChanges, setHasChanges] = useState(false);

  const handleChange = <K extends keyof NotificationPreferences>(key: K, value: NotificationPreferences[K]) => {
    setLocalPrefs((prev) => ({ ...prev, [key]: value }));
    setHasChanges(true);
  };

  const handleTypeToggle = (type: keyof NotificationPreferences["typesEnabled"]) => {
    setLocalPrefs((prev) => ({
      ...prev,
      typesEnabled: { ...prev.typesEnabled, [type]: !prev.typesEnabled[type] },
    }));
    setHasChanges(true);
  };

  const handleSave = () => {
    updatePreferences(localPrefs);
    setHasChanges(false);
    notify("success", "Preferences saved", "Your notification settings have been updated.");
  };

  const handleReset = () => {
    resetPreferences();
    setLocalPrefs(preferences);
    setHasChanges(false);
    notify("info", "Preferences reset", "Notification settings have been restored to defaults.");
  };

  const handleTestNotification = () => {
    notify("info", "Test notification", "This is a test notification to preview your settings.");
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Notification Settings"
        description="Configure how and when you receive notifications from the panel."
        actions={
          <Button variant="secondary" size="sm" onClick={handleTestNotification}>
            Test Notification
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader
            title="General Settings"
            description="Control overall notification behavior"
          />
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-ink-1">Enable Notifications</div>
                <div className="text-xs text-ink-3">Show toast notifications in the panel</div>
              </div>
              <ToggleSwitch
                checked={localPrefs.enabled}
                onChange={(v) => handleChange("enabled", v)}
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-ink-1">Sound Effects</div>
                <div className="text-xs text-ink-3">Play a sound when notifications appear</div>
              </div>
              <ToggleSwitch
                checked={localPrefs.soundEnabled}
                onChange={(v) => handleChange("soundEnabled", v)}
                disabled={!localPrefs.enabled}
              />
            </div>
          </div>
        </Card>

        <Card>
          <CardHeader
            title="Position"
            description="Where notifications appear on screen"
          />
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              {(["top-right", "bottom-right", "top-left", "bottom-left"] as const).map((pos) => (
                <button
                  key={pos}
                  className={`rounded-md border p-3 text-center text-sm transition-colors ${
                    localPrefs.position === pos
                      ? "border-brand-500 bg-brand-500/10 text-ink-1"
                      : "border-surface-border bg-surface-1 text-ink-2 hover:bg-surface-2"
                  }`}
                  onClick={() => handleChange("position", pos)}
                >
                  {pos.replace("-", " ")}
                </button>
              ))}
            </div>
          </div>
        </Card>

        <Card>
          <CardHeader
            title="Duration"
            description="How long notifications stay visible"
          />
          <div className="space-y-4">
            <Select
              label="Default Duration"
              value={String(localPrefs.defaultDuration)}
              onChange={(e) => handleChange("defaultDuration", parseInt(e.target.value, 10))}
              disabled={!localPrefs.enabled}
            >
              <option value={3000}>3 seconds</option>
              <option value={5000}>5 seconds</option>
              <option value={10000}>10 seconds</option>
              <option value={15000}>15 seconds</option>
              <option value={30000}>30 seconds</option>
              <option value={0}>Never auto-dismiss</option>
            </Select>
          </div>
        </Card>

        <Card>
          <CardHeader
            title="Notification Types"
            description="Choose which types of notifications to show"
          />
          <div className="space-y-4">
            {(["success", "error", "warning", "info"] as const).map((type) => (
              <div key={type} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className={`h-3 w-3 rounded-full bg-${type === "error" ? "danger" : type === "warning" ? "warning" : type}-500`} />
                  <div className="text-sm font-medium text-ink-1 capitalize">{type}</div>
                </div>
                <ToggleSwitch
                  checked={localPrefs.typesEnabled[type]}
                  onChange={() => handleTypeToggle(type)}
                  disabled={!localPrefs.enabled}
                />
              </div>
            ))}
          </div>
        </Card>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between rounded-lg border border-surface-border bg-surface-1 p-4">
        <div className="text-sm text-ink-2">
          {hasChanges ? "You have unsaved changes" : "No unsaved changes"}
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={handleReset}>
            Reset to Defaults
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={!hasChanges}>
            Save Preferences
          </Button>
        </div>
      </div>

      {/* Preview section */}
      <Card>
        <CardHeader
          title="Preview"
          description="See how your notifications will look"
        />
        <div className="space-y-3">
          {localPrefs.enabled && (
            <>
              {localPrefs.typesEnabled.success && (
                <PreviewToast type="success" title="Success notification" message="Your changes have been saved." />
              )}
              {localPrefs.typesEnabled.error && (
                <PreviewToast type="error" title="Error notification" message="Something went wrong. Please try again." />
              )}
              {localPrefs.typesEnabled.warning && (
                <PreviewToast type="warning" title="Warning notification" message="Your license is expiring soon." />
              )}
              {localPrefs.typesEnabled.info && (
                <PreviewToast type="info" title="Info notification" message="New features are available." />
              )}
            </>
          )}
          {!localPrefs.enabled && (
            <div className="py-4 text-center text-sm text-ink-3">
              Notifications are currently disabled
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}

// Toggle switch component
function ToggleSwitch({
  checked,
  onChange,
  disabled = false,
}: {
  checked: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
        disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"
      } ${checked ? "bg-brand-600" : "bg-ink-6"}`}
      onClick={() => !disabled && onChange(!checked)}
    >
      <span
        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
          checked ? "translate-x-6" : "translate-x-1"
        }`}
      />
    </button>
  );
}

// Preview toast component
function PreviewToast({
  type,
  title,
  message,
}: {
  type: "success" | "error" | "warning" | "info";
  title: string;
  message: string;
}) {
  const bgColors = {
    success: "bg-success/10 border-success/30",
    error: "bg-danger/10 border-danger/30",
    warning: "bg-warning/10 border-warning/30",
    info: "bg-info/10 border-info/30",
  };

  const iconColors = {
    success: "text-success",
    error: "text-danger",
    warning: "text-warning",
    info: "text-info",
  };

  return (
    <div className={`rounded-lg border p-4 ${bgColors[type]}`}>
      <div className="flex items-start gap-3">
        <div className={iconColors[type]}>
          {type === "success" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
          )}
          {type === "error" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <circle cx="12" cy="12" r="10" />
              <line x1="15" y1="9" x2="9" y2="15" />
              <line x1="9" y1="9" x2="15" y2="15" />
            </svg>
          )}
          {type === "warning" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
          )}
          {type === "info" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
          )}
        </div>
        <div className="flex-1">
          <h4 className="text-sm font-medium text-ink-1">{title}</h4>
          <p className="mt-1 text-xs text-ink-3">{message}</p>
        </div>
      </div>
    </div>
  );
}