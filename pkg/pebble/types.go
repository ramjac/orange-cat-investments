package pebble

import "time"

// AppMessage Dictionary Key Constants for Pebble Watch C SDK
const (
	PebbleKeyCmd       uint32 = 0 // Command Type (1 = ALERT, 2 = ACK_CONFIRM)
	PebbleKeyTicketID  uint32 = 1 // Ticket ID UUID
	PebbleKeyTitle     uint32 = 2 // Alert Title
	PebbleKeyBody      uint32 = 3 // Alert Body / Details
	PebbleKeyPriority  uint32 = 4 // Alert Priority
	PebbleKeyAuthor    uint32 = 5 // Author / Source
	PebbleKeyTimestamp uint32 = 6 // UTC Timestamp String
)

// PebbleAlertPayload represents a push alert broadcast to paired Pebble watches.
type PebbleAlertPayload struct {
	TicketID       string            `json:"ticket_id"`
	Title          string            `json:"title"`
	Priority       string            `json:"priority"`
	Body           string            `json:"body"`
	Author         string            `json:"author"`
	CreatedAt      time.Time         `json:"created_at"`
	AppMessageDict map[uint32]string `json:"app_message_dict"`
}

// PebbleAckRequest represents an acknowledgment received from a Pebble watch.
type PebbleAckRequest struct {
	TicketID       string `json:"ticket_id"`
	AcknowledgedBy string `json:"acknowledged_by"`
	WatchID        string `json:"watch_id,omitempty"`
	DeviceID       string `json:"device_id,omitempty"`
	ButtonID       uint8  `json:"button_id,omitempty"` // 1 = SELECT (ACK)
}

// PebbleAckResponse represents the response to a Pebble ACK confirmation.
type PebbleAckResponse struct {
	TicketID       string    `json:"ticket_id"`
	State          string    `json:"state"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
	AcknowledgedBy string    `json:"acknowledged_by"`
	StatusMessage  string    `json:"status_message"`
}
