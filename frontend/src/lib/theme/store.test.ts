/**
 * Tests for the theme store (Zustand).
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { useThemeStore, type Theme } from "@/lib/theme/store";

describe("ThemeStore", () => {
  beforeEach(() => {
    // Reset store state
    useThemeStore.setState({ theme: "light" });
    // Clear document classes
    document.documentElement.classList.remove("dark");
    document.documentElement.style.colorScheme = "";
  });

  it("should start with light theme", () => {
    const { theme } = useThemeStore.getState();
    expect(theme).toBe("light");
  });

  it("should set theme to dark", () => {
    const { setTheme } = useThemeStore.getState();
    setTheme("dark");

    const { theme } = useThemeStore.getState();
    expect(theme).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe("dark");
  });

  it("should set theme to light", () => {
    const { setTheme } = useThemeStore.getState();
    setTheme("dark"); // First set dark
    setTheme("light"); // Then set light

    const { theme } = useThemeStore.getState();
    expect(theme).toBe("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe("light");
  });

  it("should toggle theme from light to dark", () => {
    const { toggle } = useThemeStore.getState();
    expect(useThemeStore.getState().theme).toBe("light");

    toggle();
    expect(useThemeStore.getState().theme).toBe("dark");
  });

  it("should toggle theme from dark to light", () => {
    useThemeStore.setState({ theme: "dark" });
    document.documentElement.classList.add("dark");

    const { toggle } = useThemeStore.getState();
    toggle();

    expect(useThemeStore.getState().theme).toBe("light");
  });

  it("should apply theme to document on setTheme", () => {
    const { setTheme } = useThemeStore.getState();

    setTheme("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe("dark");

    setTheme("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe("light");
  });

  it("should have correct type", () => {
    const { theme } = useThemeStore.getState();
    expect(theme === "light" || theme === "dark").toBe(true);
  });
});