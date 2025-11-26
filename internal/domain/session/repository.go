package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for session storage operations
type Repository interface {
	// Session CRUD operations
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, appName, userID, sessionID string, opts *GetOptions) (*Session, error)
	List(ctx context.Context, appName, userID string) ([]*Session, error)
	Delete(ctx context.Context, appName, userID, sessionID string) error
	UpdateState(ctx context.Context, appName, userID, sessionID string, state map[string]interface{}) error

	// Event operations
	AppendEvent(ctx context.Context, sessionUUID uuid.UUID, event *Event) error
	GetEvents(ctx context.Context, sessionUUID uuid.UUID, opts *GetEventsOptions) ([]*Event, error)

	// State operations
	GetAppState(ctx context.Context, appName string) (*AppState, error)
	SetAppState(ctx context.Context, appName string, state map[string]interface{}) error
	GetUserState(ctx context.Context, appName, userID string) (*UserState, error)
	SetUserState(ctx context.Context, appName, userID string, state map[string]interface{}) error
}

// GetOptions provides filtering options for Get operation
type GetOptions struct {
	NumRecentEvents int
	After           time.Time
}

// GetEventsOptions provides filtering options for event retrieval
type GetEventsOptions struct {
	Limit int
	After time.Time
}
