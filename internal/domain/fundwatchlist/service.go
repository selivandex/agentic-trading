package fundwatchlist

import (
	"context"

	"github.com/google/uuid"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service coordinates fund watchlist operations
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a fund watchlist service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		log:  logger.Get().With("component", "fundwatchlist_service"),
	}
}

// Create adds a new symbol to the fund watchlist
func (s *Service) Create(ctx context.Context, entry *Watchlist) error {
	if entry == nil {
		return errors.ErrInvalidInput
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.Symbol == "" {
		return errors.New("symbol is required")
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return errors.Wrap(err, "create fund watchlist entry")
	}

	s.log.Infow("Symbol added to fund watchlist",
		"symbol", entry.Symbol,
		"market_type", entry.MarketType,
		"category", entry.Category,
	)

	return nil
}

// GetActive returns all active symbols
func (s *Service) GetActive(ctx context.Context) ([]*Watchlist, error) {
	entries, err := s.repo.GetActive(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get active fund watchlist")
	}
	return entries, nil
}

// IsMonitored checks if a symbol is actively monitored
func (s *Service) IsMonitored(ctx context.Context, symbol, marketType string) (bool, error) {
	return s.repo.IsActive(ctx, symbol, marketType)
}

// Pause pauses monitoring for a symbol
func (s *Service) Pause(ctx context.Context, symbol, marketType, reason string) error {
	entry, err := s.repo.GetBySymbol(ctx, symbol, marketType)
	if err != nil {
		return errors.Wrap(err, "get fund watchlist entry")
	}

	entry.IsPaused = true
	entry.PausedReason = &reason

	if err := s.repo.Update(ctx, entry); err != nil {
		return errors.Wrap(err, "pause fund watchlist entry")
	}

	s.log.Infow("Symbol paused in fund watchlist",
		"symbol", symbol,
		"reason", reason,
	)

	return nil
}

// Resume resumes monitoring for a symbol
func (s *Service) Resume(ctx context.Context, symbol, marketType string) error {
	entry, err := s.repo.GetBySymbol(ctx, symbol, marketType)
	if err != nil {
		return errors.Wrap(err, "get fund watchlist entry")
	}

	entry.IsPaused = false
	entry.PausedReason = nil

	if err := s.repo.Update(ctx, entry); err != nil {
		return errors.Wrap(err, "resume fund watchlist entry")
	}

	s.log.Infow("Symbol resumed in fund watchlist", "symbol", symbol)

	return nil
}

// IsActive checks if a symbol is actively monitored (delegates to repository)
func (s *Service) IsActive(ctx context.Context, symbol, marketType string) (bool, error) {
	return s.repo.IsActive(ctx, symbol, marketType)
}

// GetByID retrieves a watchlist entry by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Watchlist, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	return s.repo.GetByID(ctx, id)
}

// GetBySymbol retrieves a watchlist entry by symbol and market type
func (s *Service) GetBySymbol(ctx context.Context, symbol, marketType string) (*Watchlist, error) {
	if symbol == "" {
		return nil, errors.ErrInvalidInput
	}
	return s.repo.GetBySymbol(ctx, symbol, marketType)
}

// GetAll retrieves all watchlist entries
func (s *Service) GetAll(ctx context.Context) ([]*Watchlist, error) {
	return s.repo.GetAll(ctx)
}

// Update updates a watchlist entry
func (s *Service) Update(ctx context.Context, entry *Watchlist) error {
	if entry == nil || entry.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if entry.Symbol == "" {
		return errors.New("symbol is required")
	}

	if err := s.repo.Update(ctx, entry); err != nil {
		return errors.Wrap(err, "update fund watchlist entry")
	}

	s.log.Debugw("Fund watchlist entry updated",
		"id", entry.ID,
		"symbol", entry.Symbol,
	)

	return nil
}

// Delete deletes a watchlist entry
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete fund watchlist entry")
	}

	s.log.Infow("Fund watchlist entry deleted", "id", id)

	return nil
}
