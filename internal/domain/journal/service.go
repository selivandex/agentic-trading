package journal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"prometheus/pkg/logger"
)

// Service encapsulates journal operations.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a journal service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Create writes a new journal entry.
func (s *Service) Create(ctx context.Context, entry *JournalEntry) error {
	if entry == nil {
		return fmt.Errorf("create journal entry: entry is nil")
	}
	if entry.UserID == uuid.Nil {
		return fmt.Errorf("create journal entry: user id is required")
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return fmt.Errorf("create journal entry: %w", err)
	}
	return nil
}

// GetByID retrieves a single entry.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("get journal entry: id is required")
	}
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get journal entry: %w", err)
	}
	return entry, nil
}

// GetEntriesSince returns entries from a timestamp.
func (s *Service) GetEntriesSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]JournalEntry, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("get entries since: user id is required")
	}
	entries, err := s.repo.GetEntriesSince(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("get entries since: %w", err)
	}
	return entries, nil
}

// GetStrategyStats returns aggregated stats for a strategy.
func (s *Service) GetStrategyStats(ctx context.Context, userID uuid.UUID, since time.Time) ([]StrategyStats, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("get strategy stats: user id is required")
	}
	stats, err := s.repo.GetStrategyStats(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("get strategy stats: %w", err)
	}
	return stats, nil
}

// GetByStrategy returns journal entries filtered by strategy.
func (s *Service) GetByStrategy(ctx context.Context, userID uuid.UUID, strategy string, limit int) ([]JournalEntry, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("get journal by strategy: user id is required")
	}
	if limit <= 0 {
		limit = 20
	}
	entries, err := s.repo.GetByStrategy(ctx, userID, strategy, limit)
	if err != nil {
		return nil, fmt.Errorf("get journal by strategy: %w", err)
	}
	return entries, nil
}
