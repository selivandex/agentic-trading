package fund_watchlist

import (
	"context"

	"github.com/google/uuid"

	"prometheus/internal/domain/fundwatchlist"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles fund watchlist business logic (Application Service)
type Service struct {
	repo fundwatchlist.Repository
	log  *logger.Logger
}

// NewService creates a new fund watchlist service
func NewService(repo fundwatchlist.Repository, log *logger.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log.With("service", "fund_watchlist"),
	}
}

// GetByID retrieves a watchlist entry by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*fundwatchlist.Watchlist, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySymbol retrieves a watchlist entry by symbol and market type
func (s *Service) GetBySymbol(ctx context.Context, symbol, marketType string) (*fundwatchlist.Watchlist, error) {
	return s.repo.GetBySymbol(ctx, symbol, marketType)
}

// GetAll retrieves all watchlist entries
func (s *Service) GetAll(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	return s.repo.GetAll(ctx)
}

// GetActive retrieves all active watchlist entries
func (s *Service) GetActive(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	return s.repo.GetActive(ctx)
}

// GetMonitored retrieves all monitored symbols (active and not paused)
func (s *Service) GetMonitored(ctx context.Context, marketType *string) ([]*fundwatchlist.Watchlist, error) {
	all, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by market type if provided and not paused
	monitored := make([]*fundwatchlist.Watchlist, 0)
	for _, entry := range all {
		if entry.IsMonitored() {
			if marketType == nil || entry.MarketType == *marketType {
				monitored = append(monitored, entry)
			}
		}
	}

	return monitored, nil
}

// Create creates a new watchlist entry
func (s *Service) Create(ctx context.Context, entry *fundwatchlist.Watchlist) error {
	if err := s.repo.Create(ctx, entry); err != nil {
		return errors.Wrap(err, "failed to create fund watchlist entry")
	}

	s.log.Infow("Fund watchlist entry created",
		"symbol", entry.Symbol,
		"market_type", entry.MarketType,
		"tier", entry.Tier,
	)

	return nil
}

// Update updates a watchlist entry
func (s *Service) Update(ctx context.Context, entry *fundwatchlist.Watchlist) error {
	if err := s.repo.Update(ctx, entry); err != nil {
		return errors.Wrap(err, "failed to update fund watchlist entry")
	}

	s.log.Debugw("Fund watchlist entry updated", "id", entry.ID, "symbol", entry.Symbol)

	return nil
}

// Delete deletes a watchlist entry
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "failed to delete fund watchlist entry")
	}

	s.log.Infow("Fund watchlist entry deleted", "id", id)

	return nil
}

// TogglePause pauses or unpauses monitoring for a symbol
func (s *Service) TogglePause(ctx context.Context, id uuid.UUID, isPaused bool, reason *string) (*fundwatchlist.Watchlist, error) {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	entry.IsPaused = isPaused
	if isPaused && reason != nil {
		entry.PausedReason = reason
	} else {
		entry.PausedReason = nil
	}

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, errors.Wrap(err, "failed to toggle pause")
	}

	s.log.Infow("Fund watchlist pause toggled",
		"id", id,
		"symbol", entry.Symbol,
		"paused", isPaused,
	)

	return entry, nil
}
