package ws

import "encoding/json"

// MessageType identifies the kind of WebSocket message.
type MessageType string

const (
	MsgStateChanged       MessageType = "state_changed"
	MsgDiscoveryComplete  MessageType = "discovery_complete"
	MsgMigrationProgress  MessageType = "migration_progress"
	MsgValidationCheck    MessageType = "validation_check"
	MsgIndexProgress      MessageType = "index_progress"
	MsgError              MessageType = "error"
	MsgSync               MessageType = "sync"
	MsgFullState          MessageType = "full_state"
)

// Message is the envelope for all WebSocket messages.
type Message struct {
	Type    MessageType    `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewMessage creates a new Message with the given type and payload.
func NewMessage(typ MessageType, payload any) ([]byte, error) {
	var p json.RawMessage
	if payload != nil {
		var err error
		p, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(Message{Type: typ, Payload: p})
}
