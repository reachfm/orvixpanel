/**
 * Global Notification System - Toast notifications for the app.
 * Provides a hook to show notifications from anywhere in the app.
 */

import { create } from "zustand";
import { useEffect, useState, useCallback } from "react";

export type NotificationType = "success" | "error" | "warning" | "info";

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message?: string;
  duration?: number;
}

interface NotificationState {
  notifications: Notification[];
  add: (notification: Omit<Notification, "id">) => string;
  remove: (id: string) => void;
  clear: () => void;
}

export const useNotificationStore = create<NotificationState>((set) => ({
  notifications: [],
  add: (notification) => {
    const id = `notification-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    const newNotification = { ...notification, id };
    set((state) => ({
      notifications: [...state.notifications, newNotification],
    }));
    return id;
  },
  remove: (id) => {
    set((state) => ({
      notifications: state.notifications.filter((n) => n.id !== id),
    }));
  },
  clear: () => set({ notifications: [] }),
}));

// Hook to show notifications easily
export function useNotification() {
  const add = useNotificationStore((s) => s.add);

  return useCallback(
    (type: NotificationType, title: string, message?: string, duration = 5000) => {
      return add({ type, title, message, duration });
    },
    [add],
  );
}

// Notification container component - renders all active notifications
export function NotificationContainer() {
  const notifications = useNotificationStore((s) => s.notifications);
  const remove = useNotificationStore((s) => s.remove);

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {notifications.map((notification) => (
        <NotificationToast
          key={notification.id}
          notification={notification}
          onDismiss={() => remove(notification.id)}
        />
      ))}
    </div>
  );
}

function NotificationToast({
  notification,
  onDismiss,
}: {
  notification: Notification;
  onDismiss: () => void;
}) {
  const [isExiting, setIsExiting] = useState(false);

  useEffect(() => {
    if (notification.duration && notification.duration > 0) {
      const timer = setTimeout(() => {
        setIsExiting(true);
        setTimeout(onDismiss, 200);
      }, notification.duration);
      return () => clearTimeout(timer);
    }
  }, [notification.duration, onDismiss]);

  const handleDismiss = () => {
    setIsExiting(true);
    setTimeout(onDismiss, 200);
  };

  const iconColors = {
    success: "text-success",
    error: "text-danger",
    warning: "text-warning",
    info: "text-info",
  };

  const bgColors = {
    success: "bg-success/10 border-success/30",
    error: "bg-danger/10 border-danger/30",
    warning: "bg-warning/10 border-warning/30",
    info: "bg-info/10 border-info/30",
  };

  return (
    <div
      className={`rounded-lg border p-4 shadow-pop transition-all duration-200 ${
        isExiting ? "translate-x-full opacity-0" : "translate-x-0 opacity-100"
      } ${bgColors[notification.type]}`}
    >
      <div className="flex items-start gap-3">
        <div className={iconColors[notification.type]}>
          {notification.type === "success" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
              <polyline points="22 4 12 14.01 9 11.01" />
            </svg>
          )}
          {notification.type === "error" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <circle cx="12" cy="12" r="10" />
              <line x1="15" y1="9" x2="9" y2="15" />
              <line x1="9" y1="9" x2="15" y2="15" />
            </svg>
          )}
          {notification.type === "warning" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
          )}
          {notification.type === "info" && (
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-5 w-5">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
          )}
        </div>
        <div className="flex-1">
          <h4 className="text-sm font-medium text-ink-1">{notification.title}</h4>
          {notification.message && (
            <p className="mt-1 text-xs text-ink-3">{notification.message}</p>
          )}
        </div>
        <button
          onClick={handleDismiss}
          className="text-ink-3 hover:text-ink-2"
          aria-label="Close"
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="h-4 w-4">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
    </div>
  );
}