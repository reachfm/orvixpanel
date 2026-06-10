/**
 * New account form. Single page, single mutation. On success, redirect
 * to the new account's detail page.
 * v0.3.1 Phase B: Enhanced with inline validation, plan descriptions, and disk quota preview.
 */

import { useState, type FormEvent, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Input } from "@/lib/ui/Input";
import { Button } from "@/lib/ui/Button";
import { StatusPill } from "@/lib/ui/StatusPill";
import { ApiError } from "@/lib/api/client";
import { createAccount } from "@/lib/api/accounts";
import { accountKeys } from "@/lib/query/keys";

const PLAN_INFO = {
  basic: { label: "Basic", desc: "10 GB disk, 100 GB bandwidth", color: "neutral" },
  pro: { label: "Pro", desc: "50 GB disk, 500 GB bandwidth", color: "info" },
  unlimited: { label: "Unlimited", desc: "Unlimited disk & bandwidth", color: "success" },
} as const;

const errorMap: Record<string, string> = {
  missing_username: "Username is required.",
  account_username_taken: "That username is already taken.",
  hosting_linux_only: "Account provisioning is only supported on Linux.",
  user_provision_failed: "The system user could not be provisioned. Check server logs.",
  account_create_failed: "Database insert failed. Check server logs.",
  invalid_body: "Invalid request. Please review your inputs.",
};

export function NewAccountPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();

  // Form state
  const [username, setUsername] = useState("");
  const [domain, setDomain] = useState("");
  const [plan, setPlan] = useState<"basic" | "pro" | "unlimited">("basic");
  const [diskQuotaMB, setDiskQuotaMB] = useState(10240);
  const [bandwidthGB, setBandwidthGB] = useState(100);

  // Validation state
  const [touched, setTouched] = useState({ username: false, domain: false });

  // Inline validation
  const usernameError = useMemo(() => {
    if (!touched.username || username.length === 0) return null;
    if (username.length < 3) return "Username must be at least 3 characters";
    if (!/^[a-z0-9_-]+$/.test(username)) return "Only lowercase letters, digits, _ and - allowed";
    return null;
  }, [username, touched.username]);

  const domainError = useMemo(() => {
    if (!touched.domain || domain.length === 0) return null;
    const domainRegex = /^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$/;
    if (!domainRegex.test(domain)) return "Enter a valid domain (e.g. example.com)";
    return null;
  }, [domain, touched.domain]);

  const isFormValid = useMemo(() => {
    return username.length >= 3 &&
      /^[a-z0-9_-]+$/.test(username) &&
      domain.length > 0 &&
      domainError === null &&
      diskQuotaMB >= 100;
  }, [username, domain, domainError, diskQuotaMB]);

  // Mutation
  const mut = useMutation({
    mutationFn: () => createAccount({
      username, domain, plan,
      disk_quota_mb: diskQuotaMB,
      bandwidth_gb: bandwidthGB,
    }),
    onSuccess: (account) => {
      qc.invalidateQueries({ queryKey: accountKeys.all() });
      navigate({ to: "/accounts/$id", params: { id: account.id } });
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    // Mark all fields as touched
    setTouched({ username: true, domain: true });
    if (!isFormValid) return;
    mut.mutate();
  };

  const errMsg = mut.error instanceof ApiError
    ? (mut.error.code.startsWith("user_provision_failed:")
        ? mut.error.code.replace("user_provision_failed:", "user_provision_failed: ")
        : errorMap[mut.error.code] ?? `Failed (${mut.error.code})`)
    : null;

  return (
    <div className="space-y-6">
      <PageHeader
        title="New account"
        description="Provisions a system user + a primary domain on this panel."
      />

      <Card>
        <form onSubmit={onSubmit} className="grid max-w-2xl grid-cols-1 gap-6 md:grid-cols-2">
          {/* Username field */}
          <div className="md:col-span-2">
            <Input
              label="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, ""))}
              onBlur={() => setTouched(t => ({ ...t, username: true }))}
              placeholder="alice"
              required
              error={usernameError ?? undefined}
              hint="Lowercase letters, digits, _ and -. Used as the Linux system user."
            />
            {username.length > 0 && !usernameError && (
              <div className="mt-1.5 flex items-center gap-1.5 text-xs text-success">
                <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
                <span>Username format is valid</span>
              </div>
            )}
          </div>

          {/* Domain field */}
          <div className="md:col-span-2">
            <Input
              label="Primary domain"
              value={domain}
              onChange={(e) => setDomain(e.target.value.toLowerCase())}
              onBlur={() => setTouched(t => ({ ...t, domain: true }))}
              placeholder="alice.example.com"
              required
              error={domainError ?? undefined}
              hint="First domain on the account. You can add more later."
            />
          </div>

          {/* Plan selector */}
          <div className="md:col-span-2">
            <label className="text-xs font-medium text-ink-2">Plan</label>
            <div className="mt-1.5 grid grid-cols-3 gap-3">
              {(Object.keys(PLAN_INFO) as Array<keyof typeof PLAN_INFO>).map((p) => {
                const info = PLAN_INFO[p];
                return (
                  <button
                    key={p}
                    type="button"
                    onClick={() => {
                      setPlan(p);
                      // Auto-fill default values based on plan
                      if (p === "basic") { setDiskQuotaMB(10240); setBandwidthGB(100); }
                      else if (p === "pro") { setDiskQuotaMB(51200); setBandwidthGB(500); }
                      else { setDiskQuotaMB(1024000); setBandwidthGB(0); }
                    }}
                    className={
                      "rounded-md border px-3 py-3 text-center transition-all " +
                      (plan === p
                        ? "border-brand-600 bg-brand-600/10 shadow-sm"
                        : "border-surface-border bg-surface-1 text-ink-2 hover:bg-surface-2 hover:border-ink-3")
                    }
                  >
                    <div className={`text-sm font-semibold capitalize ${
                      plan === p ? "text-brand-600" : "text-ink-1"
                    }`}>{info.label}</div>
                    <div className="mt-1 text-[11px] text-ink-3">{info.desc}</div>
                    {plan === p && (
                      <div className="mt-2 flex justify-center">
                        <StatusPill tone="success" className="text-[10px]">Selected</StatusPill>
                      </div>
                    )}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Quota fields */}
          <div>
            <Input
              label="Disk quota (MB)"
              type="number"
              min={100}
              max={plan === "unlimited" ? undefined : 999999}
              value={diskQuotaMB}
              onChange={(e) => setDiskQuotaMB(parseInt(e.target.value || "0", 10))}
              hint={plan === "unlimited" ? "Unlimited (0 = no limit)" : undefined}
            />
            {diskQuotaMB > 0 && (
              <div className="mt-1.5">
                <div className="flex items-center justify-between text-[11px] text-ink-3">
                  <span>~{Math.round(diskQuotaMB / 1024)} GB</span>
                </div>
                <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-ink-6">
                  <div className="h-full rounded-full bg-brand-500" style={{ width: "100%" }} />
                </div>
              </div>
            )}
          </div>
          <div>
            <Input
              label="Bandwidth (GB/month)"
              type="number"
              min={0}
              value={bandwidthGB}
              onChange={(e) => setBandwidthGB(parseInt(e.target.value || "0", 10))}
              hint={bandwidthGB === 0 ? "Unlimited" : undefined}
            />
          </div>

          {/* Error message */}
          {errMsg && (
            <div className="md:col-span-2 rounded-md border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger">
              {errMsg}
            </div>
          )}

          {/* Form actions */}
          <div className="md:col-span-2 flex items-center justify-between">
            <div className="text-xs text-ink-3">
              Review before creating — this action cannot be undone.
            </div>
            <div className="flex gap-2">
              <Button type="button" variant="ghost" onClick={() => navigate({ to: "/accounts" })}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                loading={mut.isPending}
                disabled={!isFormValid}
              >
                Create account
              </Button>
            </div>
          </div>
        </form>
      </Card>
    </div>
  );
}