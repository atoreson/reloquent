export type WSMessageType =
  | "state_changed"
  | "discovery_complete"
  | "migration_progress"
  | "validation_check"
  | "index_progress"
  | "error"
  | "sync"
  | "full_state";

export interface WSMessage {
  type: WSMessageType;
  payload?: unknown;
}

export type WSMessageHandler = (msg: WSMessage) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private handlers: Set<WSMessageHandler> = new Set();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private maxReconnectDelay = 30000;
  private closed = false;

  constructor(url: string) {
    this.url = url;
  }

  connect() {
    if (this.closed) return;

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        this.reconnectDelay = 1000;
      };

      this.ws.onmessage = (event) => {
        try {
          const msg: WSMessage = JSON.parse(event.data);
          this.handlers.forEach((h) => h(msg));
        } catch {
          // ignore parse errors
        }
      };

      this.ws.onclose = () => {
        this.scheduleReconnect();
      };

      this.ws.onerror = () => {
        this.ws?.close();
      };
    } catch {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.closed) return;
    if (this.reconnectTimer) return;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.reconnectDelay = Math.min(
        this.reconnectDelay * 2,
        this.maxReconnectDelay,
      );
      this.connect();
    }, this.reconnectDelay);
  }

  subscribe(handler: WSMessageHandler) {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  send(msg: WSMessage) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  requestSync() {
    this.send({ type: "sync" });
  }

  close() {
    this.closed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }
    this.ws?.close();
  }
}
