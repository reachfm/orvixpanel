/**
 * Theme store (Zustand).
 *
 * Persists the operator's light/dark preference to localStorage and
 * toggles the `dark` class on <html>. The bootstrap <script> in
 * index.html applies the class *before* paint to avoid a flash.
 */

import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Theme = "light" | "dark";

interface ThemeState {
  theme: Theme;
  setTheme: (t: Theme) => void;
  toggle: () => void;
}

function applyToDocument(theme: Theme) {
  const root = document.documentElement;
  if (theme === "dark") root.classList.add("dark");
  else root.classList.remove("dark");
  root.style.colorScheme = theme;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: "light",
      setTheme: (t) => {
        applyToDocument(t);
        set({ theme: t });
      },
      toggle: () => {
        const next: Theme = get().theme === "dark" ? "light" : "dark";
        applyToDocument(next);
        set({ theme: next });
      },
    }),
    {
      name: "orvix.theme",
      onRehydrateStorage: () => (state) => {
        if (state) applyToDocument(state.theme);
      },
    },
  ),
);
