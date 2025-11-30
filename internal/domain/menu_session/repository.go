package menu_session

import (
	"context"
	"time"
)

// Repository defines interface for menu session storage
type Repository interface {
	// Get retrieves a session by telegram ID
	Get(ctx context.Context, telegramID int64) (*Session, error)

	// Save stores a session with TTL
	Save(ctx context.Context, session *Session, ttl time.Duration) error

	// Delete removes a session
	Delete(ctx context.Context, telegramID int64) error

	// Exists checks if session exists
	Exists(ctx context.Context, telegramID int64) (bool, error)
}
