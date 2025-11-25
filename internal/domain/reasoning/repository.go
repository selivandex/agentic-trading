package reasoning

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for reasoning log data access
type Repository interface {
	Create(ctx context.Context, entry *LogEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*LogEntry, error)
	GetBySession(ctx context.Context, sessionID string) ([]*LogEntry, error)
	GetByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*LogEntry, error)
	GetByAgent(ctx context.Context, agentID string, limit int) ([]*LogEntry, error)
}
