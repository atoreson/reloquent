import { useEffect, useRef } from "react";

/**
 * Requests browser notification permission on mount.
 * Notifications are dispatched from the WebSocket handler in useWebSocket.
 */
export function NotificationManager() {
  const asked = useRef(false);

  useEffect(() => {
    if (!asked.current && "Notification" in window) {
      asked.current = true;
      if (Notification.permission === "default") {
        Notification.requestPermission();
      }
    }
  }, []);

  return null;
}
