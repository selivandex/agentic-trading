package session

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an ADK agent session
type Session struct {
	ID        uuid.UUID
	AppName   string
	UserID    string
	SessionID string
	State     map[string]interface{}
	Events    []Event
	UpdatedAt time.Time
	CreatedAt time.Time
}

// Event represents a session event (message/tool call/response)
type Event struct {
	ID            uuid.UUID
	SessionID     uuid.UUID
	EventID       string // ADK event ID
	Author        string // Agent name or "user"
	Content       map[string]interface{}
	Timestamp     time.Time
	Branch        string
	Partial       bool
	TurnComplete  bool
	Actions       EventActions
	UsageMetadata *UsageMetadata
}

// EventActions contains actions that can be performed with an event
type EventActions struct {
	TransferToAgent   string
	Escalate          bool
	SkipSummarization bool
	StateDelta        map[string]interface{}
}

// UsageMetadata tracks token usage for an event
type UsageMetadata struct {
	PromptTokenCount     int32
	CandidatesTokenCount int32
	TotalTokenCount      int32
}

// AppState represents application-level state shared across all users
type AppState struct {
	AppName string
	State   map[string]interface{}
}

// UserState represents user-level state shared across all user's sessions
type UserState struct {
	AppName string
	UserID  string
	State   map[string]interface{}
}

// State key prefixes for multi-level state management
const (
	KeyPrefixApp  = "_app_"
	KeyPrefixUser = "_user_"
	KeyPrefixTemp = "_temp_"
)

