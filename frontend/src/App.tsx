/**
 * App root. Wires the three global providers (TanStack Query,
 * TanStack Router, theme init) in the order they need to appear.
 *
 * The theme bootstrap (dark class on <html>) is applied by the
 * Zustand store on hydrate — see src/lib/theme/store.ts.
 */

import { useEffect } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import { router } from "@/router";
import { queryClient } from "@/lib/query/client";
import { useThemeStore } from "@/lib/theme/store";

export function App() {
  const theme = useThemeStore((s) => s.theme);

  // Apply the theme to <html> on first render + when it changes.
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "dark") root.classList.add("dark");
    else root.classList.remove("dark");
    root.style.colorScheme = theme;
  }, [theme]);

  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}
