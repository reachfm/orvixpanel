/**
 * Add Domain page. Wires the POST /api/v1/accounts/:id/domains route
 * to a single form. The route's "name" must be a valid FQDN per
 * internal/hosting.ValidateDomain; the WSL dev environment accepts
 * .local via ORVIX_ALLOW_LOCAL_TLD=1, the install sets this for dev.
 */

import { useState, type FormEvent } from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Card } from "@/lib/ui/Card";
import { PageHeader } from "@/lib/ui/PageHeader";
import { Input } from "@/lib/ui/Input";
import { Button } from "@/lib/ui/Button";
import { ApiError } from "@/lib/api/client";
import { createDomain } from "@/lib/api/domains";
import { getAccount } from "@/lib/api/accounts";
import { accountKeys, domainKeys } from "@/lib/query/keys";

const errorMap: Record<string, string> = {
  invalid_body: "Invalid request. Please review your inputs.",
  invalid_domain: "Domain name is not valid. Use a real FQDN, e.g. example.com.",
  domain_already_owned: "This domain is already owned by another account on this panel.",
  domain_owned_by: "This domain is already owned by another account on this panel.",
  create_failed: "Failed to create the domain. Check server logs.",
  nginx_invalid: "Generated nginx vhost failed validation. Check the nginx -t output in the server log.",
  php_invalid: "Generated php-fpm pool failed validation. Check php-fpm -t output in the server log.",
  reload_failed: "Domain was created but nginx/php-fpm reload failed. Check the server log.",
};

export function AddDomainPage() {
  const { id } = useParams({ strict: false }) as { id: string };
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [name, setName] = useState("");

  const acct = useQuery({ queryKey: accountKeys.detail(id), queryFn: () => getAccount(id) });

  const mut = useMutation({
    mutationFn: () => createDomain(id, { name: name.toLowerCase() }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: domainKeys.byAccount(id) });
      qc.invalidateQueries({ queryKey: accountKeys.detail(id) });
      navigate({ to: "/accounts/$id", params: { id } });
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    mut.mutate();
  };

  const errMsg =
    mut.error instanceof ApiError
      ? (mut.error.code.startsWith("domain_")
          ? (errorMap[mut.error.code] ?? mut.error.code)
          : (errorMap[mut.error.code] ?? `Failed (${mut.error.code})`))
      : null;

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Add domain to ${acct.data?.username ?? "…"}`}
        description={
          <span>
            <span className="text-ink-2">Adds a new vhost + php-fpm pool to the account.</span>
          </span>
        }
      />

      <Card>
        <form onSubmit={onSubmit} className="grid max-w-xl grid-cols-1 gap-4">
          <Input
            label="Domain"
            value={name}
            onChange={(e) => setName(e.target.value.toLowerCase())}
            placeholder="alice.example.com"
            required
            hint="Fully qualified domain name. Must resolve to this server's IP in production; the v0.2.x dev environment accepts .local."
          />

          {errMsg && (
            <div className="rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
              {errMsg}
            </div>
          )}

          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => navigate({ to: "/accounts/$id", params: { id } })}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={mut.isPending}>
              Add domain
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
