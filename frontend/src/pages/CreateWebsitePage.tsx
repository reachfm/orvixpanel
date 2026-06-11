/**
 * Create Website - Provisioning wizard.
 * Step-by-step form to provision a new website.
 */

import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardHeader } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { Spinner, ErrorState } from "@/lib/ui/Feedback";
import { Modal } from "@/lib/ui/Modal";
import { createWebsite, PHP_VERSIONS, type PhpVersion } from "@/lib/api/provisioning";
import { listAccounts, type Account } from "@/lib/api/accounts";
import { provisioningKeys } from "@/lib/query/keys";

const STEPS = [
  { id: 1, title: "Domain", description: "Enter the domain name" },
  { id: 2, title: "Owner", description: "Select account owner" },
  { id: 3, title: "PHP", description: "Choose PHP version" },
  { id: 4, title: "Resources", description: "Configure disk quota" },
  { id: 5, title: "Review", description: "Confirm settings" },
];

interface FormData {
  domain: string;
  account_id: string;
  php_version: PhpVersion;
  disk_quota_mb: number;
  enable_logs: boolean;
  create_default_index: boolean;
}

const DEFAULT_FORM: FormData = {
  domain: "",
  account_id: "",
  php_version: "8.2",
  disk_quota_mb: 1024,
  enable_logs: true,
  create_default_index: true,
};

export function CreateWebsitePage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [currentStep, setCurrentStep] = useState(1);
  const [formData, setFormData] = useState<FormData>(DEFAULT_FORM);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [confirmModal, setConfirmModal] = useState(false);

  // Fetch accounts for owner selection
  const accountsQuery = useQuery({
    queryKey: ["accounts", "all"],
    queryFn: () => listAccounts(),
  });

  const createMutation = useMutation({
    mutationFn: createWebsite,
    onSuccess: (job) => {
      qc.invalidateQueries({ queryKey: provisioningKeys.all() });
      navigate({ to: "/hosting/provisioning/jobs/$id", params: { id: job.id } });
    },
  });

  const updateField = <K extends keyof FormData>(key: K, value: FormData[K]) => {
    setFormData(prev => ({ ...prev, [key]: value }));
    if (errors[key]) {
      setErrors(prev => { const e = {...prev}; delete e[key]; return e; });
    }
  };

  const validateStep = (step: number): boolean => {
    const newErrors: Record<string, string> = {};

    if (step === 1) {
      if (!formData.domain.trim()) {
        newErrors.domain = "Domain is required";
      } else if (!/^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$/.test(formData.domain)) {
        newErrors.domain = "Invalid domain format";
      } else {
        // Basic TLD check
        const parts = formData.domain.split(".");
        if (parts.length < 2 || parts[parts.length - 1].length < 2) {
          newErrors.domain = "Invalid TLD";
        }
      }
    }

    if (step === 2) {
      if (!formData.account_id) {
        newErrors.account_id = "Account owner is required";
      }
    }

    if (step === 4) {
      if (formData.disk_quota_mb < 100) {
        newErrors.disk_quota_mb = "Minimum 100 MB required";
      } else if (formData.disk_quota_mb > 1024000) {
        newErrors.disk_quota_mb = "Maximum 1 TB allowed";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleNext = () => {
    if (validateStep(currentStep)) {
      setCurrentStep(prev => Math.min(prev + 1, 5));
    }
  };

  const handleBack = () => {
    setCurrentStep(prev => Math.max(prev - 1, 1));
  };

  const handleSubmit = () => {
    if (!validateStep(currentStep)) return;
    setConfirmModal(true);
  };

  const confirmSubmit = () => {
    setConfirmModal(false);
    createMutation.mutate(formData);
  };

  const canProceed = currentStep < 5;
  const canGoBack = currentStep > 1;

  const renderStepContent = () => {
    switch (currentStep) {
      case 1:
        return (
          <div className="space-y-6">
            <div className="text-sm text-ink-2">
              Enter the domain name for the new website. The domain should be registered
              and pointed to this server.
            </div>
            <Input
              label="Domain name"
              value={formData.domain}
              onChange={(e) => {
                updateField("domain", e.target.value.toLowerCase().trim());
              }}
              placeholder="example.com"
              hint={errors.domain || "Enter without https:// or www prefix"}
              error={errors.domain ?? null}
              autoFocus
            />
            {formData.domain && !errors.domain && (
              <div className="rounded-md bg-surface-2 p-3 text-sm">
                <div className="font-medium text-ink-1">Preview:</div>
                <div className="font-mono text-ink-2">
                  Document root: /var/www/{formData.domain}
                </div>
                <div className="font-mono text-ink-2">
                  Nginx config: /etc/nginx/sites-available/{formData.domain}
                </div>
              </div>
            )}
          </div>
        );

      case 2:
        return (
          <div className="space-y-6">
            <div className="text-sm text-ink-2">
              Select the account that will own this website.
            </div>
            {accountsQuery.isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Spinner size={20} />
              </div>
            ) : accountsQuery.isError ? (
              <ErrorState description="Failed to load accounts" onRetry={() => accountsQuery.refetch()} />
            ) : (
              <div className="space-y-2">
                {accountsQuery.data?.accounts.length === 0 ? (
                  <div className="rounded-md border border-danger/30 bg-danger/5 p-4 text-sm text-danger">
                    No active accounts found. Create an account first.
                  </div>
                ) : (
                  accountsQuery.data?.accounts.map((account: Account) => (
                    <label
                      key={account.id}
                      className={`flex cursor-pointer items-center gap-3 rounded-md border p-4 transition-all ${
                        formData.account_id === account.id
                          ? "border-brand-500 bg-brand-500/5"
                          : "border-surface-border bg-surface-1 hover:bg-surface-2"
                      }`}
                    >
                      <input
                        type="radio"
                        name="account"
                        value={account.id}
                        checked={formData.account_id === account.id}
                        onChange={() => updateField("account_id", account.id)}
                        className="h-4 w-4 text-brand-600 focus:ring-brand-500"
                      />
                      <div className="flex-1">
                        <div className="font-medium text-ink-1">{account.username}</div>
                        <div className="text-xs text-ink-3">{account.domain}</div>
                      </div>
                      <div className="text-xs text-ink-3">
                        {account.disk_used_mb && account.disk_quota_mb
                          ? `${Math.round(account.disk_used_mb / 1024)} / ${Math.round(account.disk_quota_mb / 1024)} GB`
                          : "No usage data"}
                      </div>
                    </label>
                  ))
                )}
              </div>
            )}
            {errors.account_id && (
              <div className="text-sm text-danger">{errors.account_id}</div>
            )}
          </div>
        );

      case 3:
        return (
          <div className="space-y-6">
            <div className="text-sm text-ink-2">
              Choose the PHP version for this website. PHP 8.2 is recommended for most applications.
            </div>
            <div className="grid grid-cols-3 gap-4">
              {PHP_VERSIONS.map((version) => (
                <label
                  key={version}
                  className={`flex cursor-pointer flex-col items-center rounded-lg border p-6 text-center transition-all ${
                    formData.php_version === version
                      ? "border-brand-500 bg-brand-500/5"
                      : "border-surface-border bg-surface-1 hover:bg-surface-2"
                  }`}
                >
                  <input
                    type="radio"
                    name="php_version"
                    value={version}
                    checked={formData.php_version === version}
                    onChange={() => updateField("php_version", version)}
                    className="sr-only"
                  />
                  <div className="text-2xl font-bold text-ink-1">PHP {version}</div>
                  <div className="mt-2 text-xs text-ink-3">
                    {version === "8.1" && "Stable, security support until 2025"}
                    {version === "8.2" && "Recommended, latest stable"}
                    {version === "8.3" && "Latest, may have compatibility issues"}
                  </div>
                  {formData.php_version === version && (
                    <div className="mt-2 rounded bg-brand-500 px-2 py-0.5 text-xs text-white">
                      Selected
                    </div>
                  )}
                </label>
              ))}
            </div>
            <div className="space-y-3">
              <label className="flex items-center gap-3 text-sm">
                <input
                  type="checkbox"
                  checked={formData.create_default_index}
                  onChange={(e) => updateField("create_default_index", e.target.checked)}
                  className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500"
                />
                <span>Create default index.html</span>
              </label>
              <label className="flex items-center gap-3 text-sm">
                <input
                  type="checkbox"
                  checked={formData.enable_logs}
                  onChange={(e) => updateField("enable_logs", e.target.checked)}
                  className="h-4 w-4 rounded border-ink-4 text-brand-600 focus:ring-brand-500"
                />
                <span>Enable access and error logs</span>
              </label>
            </div>
          </div>
        );

      case 4:
        return (
          <div className="space-y-6">
            <div className="text-sm text-ink-2">
              Configure disk quota and resource limits for this website.
            </div>
            <Input
              label="Disk quota (MB)"
              type="number"
              min={100}
              max={1024000}
              value={formData.disk_quota_mb}
              onChange={(e) => updateField("disk_quota_mb", parseInt(e.target.value) || 0)}
              error={errors.disk_quota_mb ?? null}
              hint={errors.disk_quota_mb || `~${Math.round(formData.disk_quota_mb / 1024)} GB`}
            />
            <div className="rounded-md bg-surface-2 p-4">
              <div className="text-sm font-medium text-ink-1">Resource Summary</div>
              <dl className="mt-3 space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-ink-3">Disk Quota</dt>
                  <dd className="font-mono text-ink-1">
                    {formData.disk_quota_mb >= 1024
                      ? `${(formData.disk_quota_mb / 1024).toFixed(1)} GB`
                      : `${formData.disk_quota_mb} MB`}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-3">PHP Version</dt>
                  <dd className="font-mono text-ink-1">{formData.php_version}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-ink-3">Logs</dt>
                  <dd className="text-ink-1">{formData.enable_logs ? "Enabled" : "Disabled"}</dd>
                </div>
              </dl>
            </div>
          </div>
        );

      case 5:
        const selectedAccount = accountsQuery.data?.accounts.find(
          (a: Account) => a.id === formData.account_id
        );
        return (
          <div className="space-y-6">
            <div className="text-sm text-ink-2">
              Review the configuration before creating the website.
            </div>
            <div className="rounded-lg border border-surface-border">
              <div className="border-b border-surface-border bg-surface-2 px-4 py-3">
                <div className="text-sm font-medium text-ink-1">Website Configuration</div>
              </div>
              <dl className="divide-y divide-surface-border p-4 text-sm">
                <div className="flex justify-between py-2">
                  <dt className="text-ink-3">Domain</dt>
                  <dd className="font-mono font-medium text-ink-1">{formData.domain}</dd>
                </div>
                <div className="flex justify-between py-2">
                  <dt className="text-ink-3">Owner</dt>
                  <dd className="text-ink-1">
                    {selectedAccount ? (
                      <span>{selectedAccount.username} ({selectedAccount.domain})</span>
                    ) : (
                      <span className="text-danger">Not selected</span>
                    )}
                  </dd>
                </div>
                <div className="flex justify-between py-2">
                  <dt className="text-ink-3">PHP Version</dt>
                  <dd className="font-mono text-ink-1">{formData.php_version}</dd>
                </div>
                <div className="flex justify-between py-2">
                  <dt className="text-ink-3">Disk Quota</dt>
                  <dd className="font-mono text-ink-1">
                    {formData.disk_quota_mb >= 1024
                      ? `${(formData.disk_quota_mb / 1024).toFixed(1)} GB`
                      : `${formData.disk_quota_mb} MB`}
                  </dd>
                </div>
                <div className="flex justify-between py-2">
                  <dt className="text-ink-3">Access Logs</dt>
                  <dd className="text-ink-1">{formData.enable_logs ? "Enabled" : "Disabled"}</dd>
                </div>
                <div className="flex justify-between py-2">
                  <dt className="text-ink-3">Default Index</dt>
                  <dd className="text-ink-1">{formData.create_default_index ? "Created" : "Not created"}</dd>
                </div>
              </dl>
            </div>
            <div className="rounded-md border border-brand-500/30 bg-brand-500/5 p-4 text-sm">
              <div className="font-medium text-brand-600">Ready to Provision</div>
              <p className="mt-1 text-ink-2">
                Click "Create Website" to start the provisioning process. This will:
              </p>
              <ul className="mt-2 list-inside list-disc space-y-1 text-ink-2">
                <li>Create document root directory</li>
                <li>Configure Nginx virtual host</li>
                <li>Set up PHP-FPM pool</li>
                <li>Configure logging (if enabled)</li>
              </ul>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Create Website"
        description="Provision a new website on this server"
      />

      {/* Progress Steps */}
      <div className="flex items-center justify-between">
        {STEPS.map((step, index) => (
          <div key={step.id} className="flex items-center">
            <div
              className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium transition-colors ${
                currentStep > step.id
                  ? "bg-brand-500 text-white"
                  : currentStep === step.id
                  ? "bg-brand-500/20 text-brand-600"
                  : "bg-surface-2 text-ink-3"
              }`}
            >
              {currentStep > step.id ? (
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                step.id
              )}
            </div>
            <div className="ml-3 hidden sm:block">
              <div className={`text-sm font-medium ${currentStep >= step.id ? "text-ink-1" : "text-ink-3"}`}>
                {step.title}
              </div>
              <div className="text-xs text-ink-3">{step.description}</div>
            </div>
            {index < STEPS.length - 1 && (
              <div className={`mx-4 h-px w-12 sm:w-24 ${
                currentStep > step.id ? "bg-brand-500" : "bg-surface-border"
              }`} />
            )}
          </div>
        ))}
      </div>

      {/* Step Content */}
      <Card>
        <CardHeader
          title={STEPS[currentStep - 1].title}
          description={`Step ${currentStep} of ${STEPS.length}`}
        />
        <div className="p-6">
          {renderStepContent()}
        </div>
        <div className="flex justify-between border-t border-surface-border bg-surface-2 px-6 py-4">
          <Button
            variant="secondary"
            onClick={handleBack}
            disabled={!canGoBack || createMutation.isPending}
          >
            Back
          </Button>
          {canProceed ? (
            <Button variant="primary" onClick={handleNext}>
              Continue
            </Button>
          ) : (
            <Button
              variant="primary"
              onClick={handleSubmit}
              loading={createMutation.isPending}
            >
              Create Website
            </Button>
          )}
        </div>
      </Card>

      {/* Error display */}
      {createMutation.isError && (
        <div className="rounded-md border border-danger/30 bg-danger/5 p-4 text-sm text-danger">
          {createMutation.error instanceof Error
            ? createMutation.error.message
            : "Failed to create website. Please try again."}
        </div>
      )}

      {/* Confirmation Modal */}
      <Modal
        open={confirmModal}
        onClose={() => setConfirmModal(false)}
        title="Confirm Website Creation"
        description={`Ready to provision ${formData.domain}?`}
        width="md"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmModal(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={confirmSubmit} loading={createMutation.isPending}>
              Create Website
            </Button>
          </>
        }
      >
        <p className="text-sm text-ink-2">
          The provisioning process will start immediately. You can monitor the progress
          on the next page.
        </p>
      </Modal>
    </div>
  );
}