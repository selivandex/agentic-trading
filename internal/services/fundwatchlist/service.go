package fundwatchlist

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"prometheus/internal/domain/fundwatchlist"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles fund watchlist business logic (Application Service)
// Following Clean Architecture: uses domain service, not repository directly
type Service struct {
	domainService fundwatchlist.DomainService // Domain service interface
	repo          fundwatchlist.Repository    // Direct access for complex queries
	log           *logger.Logger
}

// NewService creates a new fund watchlist service
func NewService(domainService fundwatchlist.DomainService, repo fundwatchlist.Repository, log *logger.Logger) *Service {
	return &Service{
		domainService: domainService,
		repo:          repo,
		log:           log.With("service", "fundwatchlist_app"),
	}
}

// GetByID retrieves a watchlist entry by ID via domain service
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*fundwatchlist.Watchlist, error) {
	return s.domainService.GetByID(ctx, id)
}

// GetBySymbol retrieves a watchlist entry by symbol and market type via domain service
func (s *Service) GetBySymbol(ctx context.Context, symbol, marketType string) (*fundwatchlist.Watchlist, error) {
	return s.domainService.GetBySymbol(ctx, symbol, marketType)
}

// GetAll retrieves all watchlist entries via domain service
func (s *Service) GetAll(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	return s.domainService.GetAll(ctx)
}

// GetActive retrieves all active watchlist entries via domain service
func (s *Service) GetActive(ctx context.Context) ([]*fundwatchlist.Watchlist, error) {
	return s.domainService.GetActive(ctx)
}

// GetMonitored retrieves all monitored symbols (active and not paused)
func (s *Service) GetMonitored(ctx context.Context, marketType *string) ([]*fundwatchlist.Watchlist, error) {
	all, err := s.domainService.GetActive(ctx)
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

// CreateWatchlistParams contains parameters for creating a watchlist entry
type CreateWatchlistParams struct {
	Symbol     string
	MarketType string
	Category   string
	Tier       int
}

// CreateWatchlist creates a new watchlist entry (admin function)
func (s *Service) CreateWatchlist(ctx context.Context, params CreateWatchlistParams) (*fundwatchlist.Watchlist, error) {
	s.log.Infow("Creating fund watchlist entry",
		"symbol", params.Symbol,
		"market_type", params.MarketType,
	)

	// Validate
	if params.Symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if params.MarketType == "" {
		return nil, errors.New("market_type is required")
	}

	entry := &fundwatchlist.Watchlist{
		ID:         uuid.New(),
		Symbol:     params.Symbol,
		MarketType: params.MarketType,
		Category:   params.Category,
		Tier:       params.Tier,
		IsActive:   true,
		IsPaused:   false,
	}

	// Create via domain service
	if err := s.domainService.Create(ctx, entry); err != nil {
		return nil, errors.Wrap(err, "failed to create fund watchlist entry")
	}

	s.log.Infow("Fund watchlist entry created",
		"id", entry.ID,
		"symbol", entry.Symbol,
		"market_type", entry.MarketType,
		"tier", entry.Tier,
	)

	return entry, nil
}

// UpdateWatchlistParams contains parameters for updating a watchlist entry
type UpdateWatchlistParams struct {
	Category     *string
	Tier         *int
	IsActive     *bool
	IsPaused     *bool
	PausedReason *string
}

// UpdateWatchlist updates a watchlist entry (admin function)
func (s *Service) UpdateWatchlist(ctx context.Context, id uuid.UUID, params UpdateWatchlistParams) (*fundwatchlist.Watchlist, error) {
	s.log.Infow("Updating fund watchlist entry", "id", id)

	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get existing entry via domain service
	entry, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get fund watchlist entry")
	}

	// Apply updates
	if params.Category != nil {
		entry.Category = *params.Category
	}
	if params.Tier != nil {
		entry.Tier = *params.Tier
	}
	if params.IsActive != nil {
		entry.IsActive = *params.IsActive
	}
	if params.IsPaused != nil {
		entry.IsPaused = *params.IsPaused
		if !*params.IsPaused {
			entry.PausedReason = nil // Clear reason when unpausing
		}
	}
	if params.PausedReason != nil {
		entry.PausedReason = params.PausedReason
	}

	// Update via domain service
	if err := s.domainService.Update(ctx, entry); err != nil {
		return nil, errors.Wrap(err, "failed to update fund watchlist entry")
	}

	s.log.Infow("Fund watchlist entry updated",
		"id", id,
		"symbol", entry.Symbol,
	)

	return entry, nil
}

// DeleteWatchlist deletes a watchlist entry (admin function, hard delete)
func (s *Service) DeleteWatchlist(ctx context.Context, id uuid.UUID) error {
	s.log.Warnw("Deleting fund watchlist entry", "id", id)

	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}

	// Use domain service to delete
	if err := s.domainService.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "failed to delete fund watchlist entry")
	}

	s.log.Infow("Fund watchlist entry deleted", "id", id)

	return nil
}

// BatchDeleteWatchlists deletes multiple watchlist entries in batch (admin function)
// Returns the number of successfully deleted entries
func (s *Service) BatchDeleteWatchlists(ctx context.Context, ids []uuid.UUID) (int, error) {
	s.log.Infow("Batch deleting fund watchlist entries",
		"count", len(ids),
	)

	if len(ids) == 0 {
		return 0, errors.ErrInvalidInput
	}

	deleted := 0
	var lastErr error

	for _, id := range ids {
		if err := s.DeleteWatchlist(ctx, id); err != nil {
			s.log.Warnw("Failed to delete watchlist entry in batch",
				"id", id,
				"error", err,
			)
			lastErr = err
			continue
		}
		deleted++
	}

	s.log.Infow("Batch delete completed",
		"deleted", deleted,
		"total", len(ids),
	)

	// Return error only if all deletions failed
	if deleted == 0 && lastErr != nil {
		return 0, lastErr
	}

	return deleted, nil
}

// TogglePause pauses or unpauses monitoring for a symbol
func (s *Service) TogglePause(ctx context.Context, id uuid.UUID, isPaused bool, reason *string) (*fundwatchlist.Watchlist, error) {
	s.log.Infow("Toggling fund watchlist pause",
		"id", id,
		"is_paused", isPaused,
	)

	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get entry via domain service
	entry, err := s.domainService.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get fund watchlist entry")
	}

	entry.IsPaused = isPaused
	if isPaused && reason != nil {
		entry.PausedReason = reason
	} else {
		entry.PausedReason = nil
	}

	// Update via domain service
	if err := s.domainService.Update(ctx, entry); err != nil {
		return nil, errors.Wrap(err, "failed to toggle pause")
	}

	s.log.Infow("Fund watchlist pause toggled",
		"id", id,
		"symbol", entry.Symbol,
		"paused", isPaused,
	)

	return entry, nil
}

// GetWatchlistsWithScope returns watchlist items filtered by scope and additional filters
// Scope is translated to status filter at repository level (SQL WHERE)
// filters is a map of filter_id -> filter_value from GraphQL JSONObject input
func (s *Service) GetWatchlistsWithScope(ctx context.Context, scopeID *string, search *string, filters map[string]interface{}) ([]*fundwatchlist.Watchlist, error) {
	filter := fundwatchlist.FilterOptions{
		Scope:  scopeID,
		Search: search,
	}

	// Process dynamic filters from GraphQL
	if filters != nil {
		s.processFilters(&filter, filters)
	}

	// Use repository directly for complex filtering
	return s.repo.GetWithFilters(ctx, filter)
}

// GetWatchlistsScopes returns counts for each scope
// Uses SQL GROUP BY for efficiency
func (s *Service) GetWatchlistsScopes(ctx context.Context) (map[string]int, error) {
	// Get counts by scope from repository (SQL GROUP BY)
	scopeCounts, err := s.repo.CountByScope(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count watchlists by scope")
	}

	return scopeCounts, nil
}

// processFilters converts GraphQL filter map to repository filter options
func (s *Service) processFilters(filter *fundwatchlist.FilterOptions, filters map[string]interface{}) {
	for filterID, filterValue := range filters {
		switch filterID {
		case "market_type":
			if val, ok := filterValue.(string); ok {
				filter.MarketType = &val
			}

		case "category":
			if val, ok := filterValue.(string); ok {
				filter.Category = &val
			}

		case "tier":
			// tier filter is multiselect - array of strings
			if val, ok := filterValue.([]interface{}); ok {
				tiers := make([]int, 0, len(val))
				for _, v := range val {
					// Try string first (from GraphQL)
					if strVal, ok := v.(string); ok {
						// Convert string to int
						var tier int
						if _, err := fmt.Sscanf(strVal, "%d", &tier); err == nil {
							tiers = append(tiers, tier)
						}
					} else if intVal, ok := v.(float64); ok {
						// Also support direct int values
						tiers = append(tiers, int(intVal))
					}
				}
				if len(tiers) > 0 {
					filter.Tiers = tiers
				}
			}
		}
	}
}
