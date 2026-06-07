/**
 * Tests for the global notification system.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { renderHook } from "@testing-library/react";
import {
  useNotificationStore,
  useNotification,
  NotificationContainer,
} from "@/lib/ui/Notification";

describe("NotificationStore", () => {
  beforeEach(() => {
    // Reset the store before each test
    useNotificationStore.getState().clear();
  });

  it("should start with empty notifications", () => {
    const { notifications } = useNotificationStore.getState();
    expect(notifications).toEqual([]);
  });

  it("should add a notification with generated id", () => {
    const store = useNotificationStore.getState();
    const id = store.add({
      type: "success",
      title: "Test",
      message: "Test message",
    });

    expect(id).toBeDefined();
    expect(typeof id).toBe("string");
    expect(id.startsWith("notification-")).toBe(true);

    const { notifications } = useNotificationStore.getState();
    expect(notifications).toHaveLength(1);
    expect(notifications[0].title).toBe("Test");
    expect(notifications[0].type).toBe("success");
  });

  it("should remove a notification by id", () => {
    const store = useNotificationStore.getState();
    const id = store.add({ type: "info", title: "To remove" });

    expect(useNotificationStore.getState().notifications).toHaveLength(1);

    store.remove(id);

    expect(useNotificationStore.getState().notifications).toHaveLength(0);
  });

  it("should clear all notifications", () => {
    const store = useNotificationStore.getState();
    store.add({ type: "info", title: "One" });
    store.add({ type: "error", title: "Two" });
    store.add({ type: "warning", title: "Three" });

    expect(useNotificationStore.getState().notifications).toHaveLength(3);

    store.clear();

    expect(useNotificationStore.getState().notifications).toHaveLength(0);
  });

  it("should support custom duration", () => {
    const store = useNotificationStore.getState();
    store.add({ type: "info", title: "Custom duration", duration: 10000 });

    const { notifications } = useNotificationStore.getState();
    expect(notifications[0].duration).toBe(10000);
  });
});

describe("useNotification hook", () => {
  beforeEach(() => {
    useNotificationStore.getState().clear();
  });

  it("should return a function", () => {
    const { result } = renderHook(() => useNotification());
    expect(typeof result.current).toBe("function");
  });

  it("should add notification with correct parameters", () => {
    const TestComponent = () => {
      const notify = useNotification();
      notify("success", "Title", "Message", 5000);
      return null;
    };

    render(<TestComponent />);

    const { notifications } = useNotificationStore.getState();
    expect(notifications).toHaveLength(1);
    expect(notifications[0].type).toBe("success");
    expect(notifications[0].title).toBe("Title");
    expect(notifications[0].message).toBe("Message");
    expect(notifications[0].duration).toBe(5000);
  });
});

describe("NotificationContainer", () => {
  beforeEach(() => {
    useNotificationStore.getState().clear();
  });

  it("should render nothing when no notifications", () => {
    render(<NotificationContainer />);
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("should render notification when added", () => {
    useNotificationStore.getState().add({
      type: "success",
      title: "Success!",
      message: "Operation completed",
    });

    render(<NotificationContainer />);

    expect(screen.getByText("Success!")).toBeInTheDocument();
    expect(screen.getByText("Operation completed")).toBeInTheDocument();
  });

  it("should render all notification types", () => {
    const store = useNotificationStore.getState();
    store.add({ type: "success", title: "Success" });
    store.add({ type: "error", title: "Error" });
    store.add({ type: "warning", title: "Warning" });
    store.add({ type: "info", title: "Info" });

    render(<NotificationContainer />);

    expect(screen.getByText("Success")).toBeInTheDocument();
    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.getByText("Warning")).toBeInTheDocument();
    expect(screen.getByText("Info")).toBeInTheDocument();
  });

  it("should dismiss notification on close button click", () => {
    const store = useNotificationStore.getState();
    const id = store.add({ type: "info", title: "To dismiss" });

    render(<NotificationContainer />);

    expect(screen.getByText("To dismiss")).toBeInTheDocument();

    const closeButton = screen.getByRole("button", { name: /close/i });
    fireEvent.click(closeButton);

    // Wait for exit animation
    act(() => {
      vi.advanceTimersByTime(250);
    });

    expect(useNotificationStore.getState().notifications).toHaveLength(0);
  });
});