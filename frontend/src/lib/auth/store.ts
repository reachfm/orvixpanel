/**
 * Auth store (Zustand).
 *
 * Holds the JWT pair + the current user. The store is the *only* place
 * the access token is persisted (localStorage) and the only place the
 * API client reads it from. Pages never reach into localStorage
 * directly.
 *
 * On `clear()` the store is wiped and the route guard pushes the
 * browser to /login.
 */

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { User } from "@/lib/api/auth";

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  expiresAt: string | null;
  user: User | null;

  setTokens: (t: { accessToken: string; refreshToken: string; expiresAt: string }) => void;
  setUser: (u: User) => void;
  clear: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      user: null,

      setTokens: (t) => set({
        accessToken: t.accessToken,
        refreshToken: t.refreshToken,
        expiresAt: t.expiresAt,
      }),
      setUser: (u) => set({ user: u }),
      clear: () => set({
        accessToken: null,
        refreshToken: null,
        expiresAt: null,
        user: null,
      }),
    }),
    {
      name: "orvix.auth",
      // Persist only the tokens + user; nothing else.
      partialize: (s) => ({
        accessToken: s.accessToken,
        refreshToken: s.refreshToken,
        expiresAt: s.expiresAt,
        user: s.user,
      }),
    },
  ),
);
