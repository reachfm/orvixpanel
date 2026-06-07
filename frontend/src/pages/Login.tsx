/**
 * Login page. Renders outside the AppLayout (no sidebar / topbar).
 *
 * Calls /auth/login, stores the JWT pair in the Zustand auth store,
 * and pushes to / on success. The /auth/login route in main.go
 * returns a stable error code on failure (invalid_credentials,
 * account_locked, account_suspended, login_failed) which we map to
 * a per-code message in the UI.
 */

import { useState, type FormEvent } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/lib/ui/Button";
import { Input } from "@/lib/ui/Input";
import { login } from "@/lib/api/auth";
import { ApiError } from "@/lib/api/client";
import { useAuthStore } from "@/lib/auth/store";
import { useThemeStore } from "@/lib/theme/store";

const errorMessages: Record<string, string> = {
  invalid_credentials: "Email or password is incorrect.",
  account_locked: "Account is temporarily locked. Try again in a few minutes.",
  account_suspended: "Account is suspended. Contact your administrator.",
  login_failed: "Login failed. Please try again.",
  missing_credentials: "Email and password are required.",
  invalid_body: "Invalid request. Please try again.",
};

export function LoginPage() {
  const navigate = useNavigate();
  const setTokens = useAuthStore((s) => s.setTokens);
  const setUser = useAuthStore((s) => s.setUser);
  const theme = useThemeStore((s) => s.theme);
  const toggleTheme = useThemeStore((s) => s.toggle);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const mut = useMutation({
    mutationFn: (vars: { email: string; password: string }) => login(vars.email, vars.password),
    onSuccess: (res) => {
      setTokens({
        accessToken: res.access_token,
        refreshToken: res.refresh_token,
        expiresAt: res.expires_at,
      });
      setUser(res.user);
      navigate({ to: "/" });
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    mut.mutate({ email, password });
  };

  const errMsg =
    mut.error instanceof ApiError
      ? errorMessages[mut.error.code] ?? `Login failed (${mut.error.code}).`
      : mut.error
        ? "Network error. Please try again."
        : null;

  return (
    <div className="grid h-full place-items-center bg-surface-0">
      <div className="absolute right-4 top-4">
        <Button variant="ghost" size="sm" onClick={toggleTheme}>
          {theme === "dark" ? "Light" : "Dark"} theme
        </Button>
      </div>

      <div className="w-full max-w-sm rounded-lg border border-surface-border bg-surface-1 p-6 shadow-card">
        <div className="mb-5 flex items-center gap-2">
          <div className="grid h-8 w-8 place-items-center rounded-md bg-brand-600 text-sm font-bold text-white">O</div>
          <div>
            <div className="text-base font-semibold text-ink-1">OrvixPanel</div>
            <div className="text-xs text-ink-3">Sign in to continue</div>
          </div>
        </div>

        <form onSubmit={onSubmit} className="space-y-3">
          <Input
            label="Email"
            name="email"
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="admin@orvixpanel.local"
            required
          />
          <Input
            label="Password"
            name="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />

          {errMsg && (
            <div className="rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
              {errMsg}
            </div>
          )}

          <Button
            type="submit"
            variant="primary"
            size="md"
            className="w-full"
            loading={mut.isPending}
          >
            Sign in
          </Button>
        </form>
      </div>
    </div>
  );
}
