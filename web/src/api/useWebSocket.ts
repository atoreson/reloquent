import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { WebSocketClient } from "./websocket";
import type { WSMessage } from "./websocket";

function getWSUrl(): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/api/ws`;
}

export function useWebSocket() {
  const queryClient = useQueryClient();
  const clientRef = useRef<WebSocketClient | null>(null);

  useEffect(() => {
    const wsClient = new WebSocketClient(getWSUrl());
    clientRef.current = wsClient;

    const unsubscribe = wsClient.subscribe((msg: WSMessage) => {
      switch (msg.type) {
        case "state_changed":
        case "full_state":
          queryClient.invalidateQueries({ queryKey: ["state"] });
          break;
        case "discovery_complete":
          queryClient.invalidateQueries({ queryKey: ["schema"] });
          queryClient.invalidateQueries({ queryKey: ["tables"] });
          break;
        case "migration_progress":
          queryClient.invalidateQueries({ queryKey: ["migration-status"] });
          break;
        case "validation_check":
          queryClient.invalidateQueries({ queryKey: ["validation-results"] });
          break;
        case "index_progress":
          queryClient.invalidateQueries({ queryKey: ["index-status"] });
          break;
      }

      // Browser notifications when tab is hidden
      if (
        document.hidden &&
        "Notification" in window &&
        Notification.permission === "granted"
      ) {
        const payload = msg.payload as Record<string, unknown> | undefined;
        if (msg.type === "migration_progress" && payload?.phase === "complete") {
          new Notification("Reloquent", { body: "Migration complete!" });
        }
        if (msg.type === "error") {
          new Notification("Reloquent", {
            body: `Error: ${(payload as Record<string, string>)?.message || "Unknown"}`,
          });
        }
      }
    });

    wsClient.connect();

    return () => {
      unsubscribe();
      wsClient.close();
    };
  }, [queryClient]);

  return clientRef;
}
