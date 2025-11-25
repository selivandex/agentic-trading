package journal

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for journal data access
type Repository interface {
	Create(ctx context.Context, entry *JournalEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	GetEntriesSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]JournalEntry, error)
	GetStrategyStats(ctx context.Context, userID uuid.UUID, since time.Time) ([]StrategyStats, error)
	GetByStrategy(ctx context.Context, userID uuid.UUID, strategy string, limit int) ([]JournalEntry, error)
}


import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for journal data access
type Repository interface {
	Create(ctx context.Context, entry *JournalEntry) error
	GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	GetEntriesSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]JournalEntry, error)
	GetStrategyStats(ctx context.Context, userID uuid.UUID, since time.Time) ([]StrategyStats, error)
	GetByStrategy(ctx context.Context, userID uuid.UUID, strategy string, limit int) ([]JournalEntry, error)
}

