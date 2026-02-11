package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

const (
	writeTimeout = 10 * time.Second
	readTimeout  = 60 * time.Second
	pingPeriod   = 30 * time.Second
)

// HandleWebSocket upgrades the HTTP connection to a WebSocket and manages
// the read/write pumps for the client.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin (dev mode)
	})
	if err != nil {
		h.logger.Error("websocket accept failed", "error", err)
		return
	}

	client := &Client{
		hub:  h,
		send: make(chan []byte, 256),
		conn: conn,
	}

	h.register <- client

	// Send initial full state on connect
	if h.stateProvider != nil {
		data, err := h.stateProvider()
		if err == nil {
			msg, _ := NewMessage(MsgFullState, json.RawMessage(data))
			select {
			case client.send <- msg:
			default:
			}
		}
	}

	go client.writePump(r.Context())
	client.readPump(r.Context())
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				c.hub.logger.Debug("websocket client disconnected normally")
			}
			return
		}

		// Parse incoming messages
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case MsgSync:
			// Client requests full state re-sync
			if c.hub.stateProvider != nil {
				state, err := c.hub.stateProvider()
				if err == nil {
					reply, _ := NewMessage(MsgFullState, json.RawMessage(state))
					select {
					case c.send <- reply:
					default:
					}
				}
			}
		}
	}
}

func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(writeCtx, websocket.MessageText, message)
			cancel()
			if err != nil {
				return
			}

		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}
