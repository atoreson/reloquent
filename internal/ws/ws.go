package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"nhooyr.io/websocket"
)

// StateProviderFunc returns the current wizard state as JSON bytes.
type StateProviderFunc func() ([]byte, error)

// Hub manages WebSocket connections and broadcasts messages to all clients.
type Hub struct {
	clients       map[*Client]bool
	broadcast     chan []byte
	register      chan *Client
	unregister    chan *Client
	logger        *slog.Logger
	mu            sync.RWMutex
	stateProvider StateProviderFunc
}

// Client represents a single WebSocket connection.
type Client struct {
	hub  *Hub
	send chan []byte
	conn *websocket.Conn
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// SetStateProvider sets the function called to get current state for new/reconnecting clients.
func (h *Hub) SetStateProvider(fn StateProviderFunc) {
	h.stateProvider = fn
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("websocket client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Debug("websocket client disconnected")

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastStateChanged broadcasts a state change event.
func (h *Hub) BroadcastStateChanged() {
	msg, err := NewMessage(MsgStateChanged, nil)
	if err != nil {
		h.logger.Error("failed to create state_changed message", "error", err)
		return
	}
	h.Broadcast(msg)
}

// BroadcastMigrationProgress broadcasts migration progress.
func (h *Hub) BroadcastMigrationProgress(payload any) {
	msg, err := NewMessage(MsgMigrationProgress, payload)
	if err != nil {
		return
	}
	h.Broadcast(msg)
}

// BroadcastValidationCheck broadcasts a validation check result.
func (h *Hub) BroadcastValidationCheck(payload any) {
	msg, err := NewMessage(MsgValidationCheck, payload)
	if err != nil {
		return
	}
	h.Broadcast(msg)
}

// BroadcastIndexProgress broadcasts index build progress.
func (h *Hub) BroadcastIndexProgress(payload any) {
	msg, err := NewMessage(MsgIndexProgress, payload)
	if err != nil {
		return
	}
	h.Broadcast(msg)
}

// BroadcastError broadcasts an error to all clients.
func (h *Hub) BroadcastError(errMsg string) {
	msg, err := NewMessage(MsgError, map[string]string{"message": errMsg})
	if err != nil {
		return
	}
	h.Broadcast(msg)
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastJSON broadcasts any JSON-serializable payload with the given message type.
func (h *Hub) BroadcastJSON(msgType MessageType, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("failed to marshal broadcast payload", "error", err)
		return
	}
	msg, err := NewMessage(msgType, json.RawMessage(data))
	if err != nil {
		h.logger.Error("failed to create broadcast message", "error", err)
		return
	}
	h.Broadcast(msg)
}
