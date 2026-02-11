package ws

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	logger := slog.Default()
	hub := NewHub(logger)
	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map not initialized")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}
	if hub.register == nil {
		t.Error("register channel not initialized")
	}
	if hub.unregister == nil {
		t.Error("unregister channel not initialized")
	}
}

func TestClientCount_Empty(t *testing.T) {
	hub := NewHub(slog.Default())
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("ClientCount() = %d, want 0", got)
	}
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	// Register
	hub.register <- client
	time.Sleep(50 * time.Millisecond)
	if got := hub.ClientCount(); got != 1 {
		t.Errorf("after register: ClientCount() = %d, want 1", got)
	}

	// Unregister
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("after unregister: ClientCount() = %d, want 0", got)
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	c1 := &Client{hub: hub, send: make(chan []byte, 256)}
	c2 := &Client{hub: hub, send: make(chan []byte, 256)}

	hub.register <- c1
	hub.register <- c2
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"type":"test"}`)
	hub.Broadcast(msg)

	select {
	case got := <-c1.send:
		if string(got) != string(msg) {
			t.Errorf("c1 got %q, want %q", got, msg)
		}
	case <-time.After(time.Second):
		t.Error("c1 did not receive broadcast")
	}

	select {
	case got := <-c2.send:
		if string(got) != string(msg) {
			t.Errorf("c2 got %q, want %q", got, msg)
		}
	case <-time.After(time.Second):
		t.Error("c2 did not receive broadcast")
	}
}

func TestHubBroadcast_DropsSlowClient(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	// Client with buffer size 1
	slow := &Client{hub: hub, send: make(chan []byte, 1)}
	hub.register <- slow
	time.Sleep(50 * time.Millisecond)

	// Fill the buffer
	slow.send <- []byte("filler")

	// This broadcast should close the slow client's channel
	hub.Broadcast([]byte("overflow"))
	time.Sleep(50 * time.Millisecond)

	if got := hub.ClientCount(); got != 0 {
		t.Errorf("slow client should be dropped, ClientCount() = %d, want 0", got)
	}
}

func TestSetStateProvider(t *testing.T) {
	hub := NewHub(slog.Default())
	called := false
	hub.SetStateProvider(func() ([]byte, error) {
		called = true
		return []byte(`{"step":"source_connection"}`), nil
	})
	if hub.stateProvider == nil {
		t.Error("stateProvider not set")
	}
	data, err := hub.stateProvider()
	if err != nil {
		t.Fatalf("stateProvider returned error: %v", err)
	}
	if !called {
		t.Error("stateProvider was not called")
	}
	if string(data) != `{"step":"source_connection"}` {
		t.Errorf("stateProvider returned %q", data)
	}
}

func TestNewMessage_NilPayload(t *testing.T) {
	data, err := NewMessage(MsgStateChanged, nil)
	if err != nil {
		t.Fatalf("NewMessage error: %v", err)
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Type != MsgStateChanged {
		t.Errorf("type = %q, want %q", msg.Type, MsgStateChanged)
	}
	if msg.Payload != nil {
		t.Errorf("payload should be nil, got %s", msg.Payload)
	}
}

func TestNewMessage_WithPayload(t *testing.T) {
	payload := map[string]string{"message": "test error"}
	data, err := NewMessage(MsgError, payload)
	if err != nil {
		t.Fatalf("NewMessage error: %v", err)
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Type != MsgError {
		t.Errorf("type = %q, want %q", msg.Type, MsgError)
	}

	var p map[string]string
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload error: %v", err)
	}
	if p["message"] != "test error" {
		t.Errorf("payload message = %q, want %q", p["message"], "test error")
	}
}

func TestNewMessage_AllTypes(t *testing.T) {
	types := []MessageType{
		MsgStateChanged, MsgDiscoveryComplete, MsgMigrationProgress,
		MsgValidationCheck, MsgIndexProgress, MsgError, MsgSync, MsgFullState,
	}
	for _, mt := range types {
		data, err := NewMessage(mt, nil)
		if err != nil {
			t.Errorf("NewMessage(%q) error: %v", mt, err)
			continue
		}
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Errorf("unmarshal %q error: %v", mt, err)
			continue
		}
		if msg.Type != mt {
			t.Errorf("type = %q, want %q", msg.Type, mt)
		}
	}
}

func TestBroadcastStateChanged(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	client := &Client{hub: hub, send: make(chan []byte, 256)}
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastStateChanged()

	select {
	case data := <-client.send:
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != MsgStateChanged {
			t.Errorf("type = %q, want %q", msg.Type, MsgStateChanged)
		}
	case <-time.After(time.Second):
		t.Error("did not receive state_changed broadcast")
	}
}

func TestBroadcastError(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	client := &Client{hub: hub, send: make(chan []byte, 256)}
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastError("something went wrong")

	select {
	case data := <-client.send:
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != MsgError {
			t.Errorf("type = %q, want %q", msg.Type, MsgError)
		}
		var p map[string]string
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			t.Fatalf("unmarshal payload error: %v", err)
		}
		if p["message"] != "something went wrong" {
			t.Errorf("error message = %q", p["message"])
		}
	case <-time.After(time.Second):
		t.Error("did not receive error broadcast")
	}
}

func TestBroadcastMigrationProgress(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	client := &Client{hub: hub, send: make(chan []byte, 256)}
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastMigrationProgress(map[string]any{
		"phase":   "running",
		"percent": 42,
	})

	select {
	case data := <-client.send:
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != MsgMigrationProgress {
			t.Errorf("type = %q, want %q", msg.Type, MsgMigrationProgress)
		}
	case <-time.After(time.Second):
		t.Error("did not receive migration_progress broadcast")
	}
}

func TestBroadcastJSON(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	client := &Client{hub: hub, send: make(chan []byte, 256)}
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.BroadcastJSON(MsgValidationCheck, map[string]string{"collection": "users", "status": "passed"})

	select {
	case data := <-client.send:
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != MsgValidationCheck {
			t.Errorf("type = %q, want %q", msg.Type, MsgValidationCheck)
		}
	case <-time.After(time.Second):
		t.Error("did not receive validation_check broadcast")
	}
}

func TestHubMultipleClients(t *testing.T) {
	hub := NewHub(slog.Default())
	go hub.Run()

	const n = 5
	clients := make([]*Client, n)
	for i := range clients {
		clients[i] = &Client{hub: hub, send: make(chan []byte, 256)}
		hub.register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	if got := hub.ClientCount(); got != n {
		t.Fatalf("ClientCount() = %d, want %d", got, n)
	}

	hub.Broadcast([]byte("hello"))

	for i, c := range clients {
		select {
		case msg := <-c.send:
			if string(msg) != "hello" {
				t.Errorf("client %d got %q", i, msg)
			}
		case <-time.After(time.Second):
			t.Errorf("client %d did not receive message", i)
		}
	}

	// Unregister all
	for _, c := range clients {
		hub.unregister <- c
	}
	time.Sleep(50 * time.Millisecond)
	if got := hub.ClientCount(); got != 0 {
		t.Errorf("after unregister all: ClientCount() = %d", got)
	}
}
