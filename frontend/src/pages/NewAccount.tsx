/**
 * New account form. Single page, single mutation. On success, redirect
 * to the new account's detail page.
 */

import { useState, type FormEvent } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Input } from "@/lib/ui/Input";
import { Button } from "@/lib/ui/Button";
import { ApiError } from "@/lib/api/client";
import { createAccount } from "@/lib/api/accounts";
import { accountKeys } from "@/lib/query/keys";

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
  const [username, setUsername] = useState("");
  const [domain, setDomain] = useState("");
  const [plan, setPlan] = useState<"basic" | "pro" | "unlimited">("basic");
  const [diskQuotaMB, setDiskQuotaMB] = useState(10240);
  const [bandwidthGB, setBandwidthGB] = useState(100);

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
    mut.mutate();
  };

  const errMsg =
    mut.error instanceof ApiError
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
        <form onSubmit={onSubmit} className="grid max-w-2xl grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, ""))}
            placeholder="alice"
            required
            hint="Lowercase letters, digits, _ and -. Used as the Linux system user."
          />
          <Input
            label="Primary domain"
            value={domain}
            onChange={(e) => setDomain(e.target.value.toLowerCase())}
            placeholder="alice.example.com"
            required
            hint="First domain on the account. You can add more later."
          />
          <div className="md:col-span-2">
            <label className="text-xs font-medium text-ink-2">Plan</label>
            <div className="mt-1.5 grid grid-cols-3 gap-2">
              {(["basic", "pro", "unlimited"] as const).map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setPlan(p)}
                  className={
                    "rounded-md border px-3 py-2 text-sm font-medium capitalize " +
                    (plan === p
                      ? "border-brand-600 bg-brand-600/10 text-brand-600"
                      : "border-surface-border bg-surface-1 text-ink-2 hover:bg-surface-2")
                  }
                >
                  {p}
                </button>
              ))}
            </div>
          </div>
          <Input
            label="Disk quota (MB)"
            type="number"
            min={100}
            value={diskQuotaMB}
            onChange={(e) => setDiskQuotaMB(parseInt(e.target.value || "0", 10))}
          />
          <Input
            label="Bandwidth (GB)"
            type="number"
            min={0}
            value={bandwidthGB}
            onChange={(e) => setBandwidthGB(parseInt(e.target.value || "0", 10))}
          />

          {errMsg && (
            <div className="md:col-span-2 rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
              {errMsg}
            </div>
          )}

          <div className="md:col-span-2 flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => navigate({ to: "/accounts" })}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={mut.isPending}>
              Create account
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
