package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/strategy"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles complex business logic for strategy management (Application Layer)
// Uses domain service for simple operations and handles transactions/orchestration
type Service struct {
	domainService   *strategy.Service   // Domain service for CRUD
	strategyRepo    strategy.Repository // Direct repo access for transactions
	transactionRepo strategy.TransactionRepository
	db              DB // Database for transactions
	log             *logger.Logger
}

// DB interface for transaction support
type DB interface {
	BeginTx(ctx context.Context) (Tx, error)
}

// Tx interface for database transaction
type Tx interface {
	Commit() error
	Rollback() error
}

// NewService creates a new strategy application service
func NewService(
	domainService *strategy.Service,
	strategyRepo strategy.Repository,
	transactionRepo strategy.TransactionRepository,
	db DB,
	log *logger.Logger,
) *Service {
	return &Service{
		domainService:   domainService,
		strategyRepo:    strategyRepo,
		transactionRepo: transactionRepo,
		db:              db,
		log:             log.With("component", "strategy_app_service"),
	}
}

// CreateStrategyParams contains parameters for creating a new strategy
type CreateStrategyParams struct {
	UserID             uuid.UUID
	Name               string
	Description        string
	AllocatedCapital   decimal.Decimal
	MarketType         strategy.MarketType
	RiskTolerance      strategy.RiskTolerance
	RebalanceFrequency strategy.RebalanceFrequency
	TargetAllocations  []byte // JSON
	ReasoningLog       []byte // JSON from portfolio architect agent
}

// CreateStrategy creates a new strategy with initial deposit transaction
// Uses database transaction to ensure atomicity
func (s *Service) CreateStrategy(ctx context.Context, params CreateStrategyParams) (*strategy.Strategy, error) {
	s.log.Infow("Creating new strategy",
		"user_id", params.UserID,
		"allocated_capital", params.AllocatedCapital,
		"risk_tolerance", params.RiskTolerance,
	)

	// Validate
	if params.UserID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	if params.AllocatedCapital.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("allocated_capital must be positive")
	}
	if !params.MarketType.Valid() {
		return nil, errors.New("invalid market_type")
	}
	if !params.RiskTolerance.Valid() {
		return nil, errors.New("invalid risk_tolerance")
	}
	if !params.RebalanceFrequency.Valid() {
		return nil, errors.New("invalid rebalance_frequency")
	}

	// Begin database transaction
	dbTx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			_ = dbTx.Rollback()
		}
	}()

	// Create strategy entity using domain service (ensures proper initialization)
	newStrategy := &strategy.Strategy{
		UserID:             params.UserID,
		Name:               params.Name,
		Description:        params.Description,
		AllocatedCapital:   params.AllocatedCapital,
		MarketType:         params.MarketType,
		RiskTolerance:      params.RiskTolerance,
		RebalanceFrequency: params.RebalanceFrequency,
		TargetAllocations:  params.TargetAllocations,
		ReasoningLog:       params.ReasoningLog,
	}

	// Use domain service to create (handles validation + JSON initialization)
	if err := s.domainService.Create(ctx, newStrategy); err != nil {
		return nil, errors.Wrap(err, "failed to create strategy")
	}

	// Create initial deposit transaction
	depositTx := strategy.NewTransaction(
		newStrategy.ID,
		params.UserID,
		strategy.TransactionDeposit,
		params.AllocatedCapital, // amount (positive for deposit)
		decimal.Zero,            // balance_before (empty strategy)
		fmt.Sprintf("Initial deposit: %s", params.Name),
	)

	if err := s.transactionRepo.Create(ctx, depositTx); err != nil {
		return nil, errors.Wrap(err, "failed to create deposit transaction")
	}

	// Commit database transaction
	if err := dbTx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	s.log.Infow("Strategy created successfully",
		"strategy_id", newStrategy.ID,
		"user_id", params.UserID,
		"allocated_capital", params.AllocatedCapital,
	)

	return newStrategy, nil
}

// AllocateCashToPosition allocates cash from strategy reserve to open a position
// Creates a transaction for audit trail
// Uses database transaction to ensure atomicity
func (s *Service) AllocateCashToPosition(ctx context.Context, strategyID, positionID uuid.UUID, amount decimal.Decimal) error {
	s.log.Debugw("Allocating cash to position",
		"strategy_id", strategyID,
		"position_id", positionID,
		"amount", amount,
	)

	// Begin database transaction
	dbTx, err := s.db.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			_ = dbTx.Rollback()
		}
	}()

	// Get strategy
	strat, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return errors.Wrap(err, "failed to get strategy")
	}

	// Record balance before allocation
	balanceBefore := strat.CashReserve

	// Allocate cash (validates sufficient balance)
	if err := strat.AllocateCash(amount); err != nil {
		return errors.Wrap(err, "failed to allocate cash")
	}

	// Update strategy in DB
	if err := s.strategyRepo.Update(ctx, strat); err != nil {
		return errors.Wrap(err, "failed to update strategy")
	}

	// Create transaction record
	tx := strategy.NewTransaction(
		strategyID,
		strat.UserID,
		strategy.TransactionPositionOpen,
		amount.Neg(),  // Negative amount (debit)
		balanceBefore, // Balance before this operation
		fmt.Sprintf("Position opened: %s", positionID.String()),
	)
	tx.PositionID = &positionID

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return errors.Wrap(err, "failed to create transaction")
	}

	// Commit database transaction
	if err := dbTx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.log.Infow("Cash allocated to position",
		"strategy_id", strategyID,
		"position_id", positionID,
		"amount", amount,
		"new_cash_reserve", strat.CashReserve,
	)

	return nil
}

// ReturnCashFromPosition returns cash to strategy reserve when position closes
// Creates a transaction for audit trail
// Uses database transaction to ensure atomicity
func (s *Service) ReturnCashFromPosition(ctx context.Context, strategyID, positionID uuid.UUID, amount decimal.Decimal, pnl decimal.Decimal) error {
	s.log.Debugw("Returning cash from position",
		"strategy_id", strategyID,
		"position_id", positionID,
		"amount", amount,
		"pnl", pnl,
	)

	// Begin database transaction
	dbTx, err := s.db.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() {
		if err != nil {
			_ = dbTx.Rollback()
		}
	}()

	// Get strategy
	strat, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return errors.Wrap(err, "failed to get strategy")
	}

	// Record balance before return
	balanceBefore := strat.CashReserve

	// Return cash (amount includes PnL)
	if err := strat.ReturnCash(amount); err != nil {
		return errors.Wrap(err, "failed to return cash")
	}

	// Update strategy in DB
	if err := s.strategyRepo.Update(ctx, strat); err != nil {
		return errors.Wrap(err, "failed to update strategy")
	}

	// Create transaction record
	description := fmt.Sprintf("Position closed: %s (PnL: %s)", positionID.String(), pnl.String())

	tx := strategy.NewTransaction(
		strategyID,
		strat.UserID,
		strategy.TransactionPositionClose,
		amount,        // Positive amount (credit)
		balanceBefore, // Balance before this operation
		description,
	)
	tx.PositionID = &positionID

	if err := s.transactionRepo.Create(ctx, tx); err != nil {
		return errors.Wrap(err, "failed to create transaction")
	}

	// Commit database transaction
	if err := dbTx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.log.Infow("Cash returned from position",
		"strategy_id", strategyID,
		"position_id", positionID,
		"amount", amount,
		"pnl", pnl,
		"new_cash_reserve", strat.CashReserve,
	)

	return nil
}

// UpdateStrategyEquity recalculates strategy equity from positions + cash
// Should be called periodically or when positions update
func (s *Service) UpdateStrategyEquity(ctx context.Context, strategyID uuid.UUID, positionsValue decimal.Decimal) error {
	// Get strategy
	strat, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return errors.Wrap(err, "failed to get strategy")
	}

	// Update equity
	strat.UpdateEquity(positionsValue)

	// Optimized update (only equity fields)
	if err := s.strategyRepo.UpdateEquity(ctx, strategyID, strat.CurrentEquity, strat.CashReserve, strat.TotalPnL, strat.TotalPnLPercent); err != nil {
		return errors.Wrap(err, "failed to update strategy equity")
	}

	s.log.Debugw("Strategy equity updated",
		"strategy_id", strategyID,
		"current_equity", strat.CurrentEquity,
		"total_pnl", strat.TotalPnL,
		"total_pnl_percent", strat.TotalPnLPercent,
	)

	return nil
}

// GetActiveStrategies returns all active strategies for a user
// Delegates to domain service
func (s *Service) GetActiveStrategies(ctx context.Context, userID uuid.UUID) ([]*strategy.Strategy, error) {
	return s.domainService.GetActiveByUserID(ctx, userID)
}

// GetStrategiesWithScope returns strategies filtered by scope and additional filters
// Scope is translated to status filter at repository level (SQL WHERE)
// filters is a map of filter_id -> filter_value from GraphQL JSONObject input
func (s *Service) GetStrategiesWithScope(ctx context.Context, userID uuid.UUID, scopeID *string, search *string, filters map[string]interface{}) ([]*strategy.Strategy, error) {
	filter := strategy.FilterOptions{
		UserID: &userID,
		Search: search,
	}

	// Map scope to status filter
	if scopeID != nil && *scopeID != "" && *scopeID != "all" {
		switch *scopeID {
		case "active":
			status := strategy.StrategyActive
			filter.Status = &status
		case "paused":
			status := strategy.StrategyPaused
			filter.Status = &status
		case "closed":
			status := strategy.StrategyClosed
			filter.Status = &status
		default:
			// Unknown scope, return all
		}
	}

	// Apply additional filters from map
	if filters != nil {
		s.applyFiltersToOptions(&filter, filters)
	}

	return s.strategyRepo.GetWithFilter(ctx, filter)
}

// applyFiltersToOptions parses filters map and applies them to FilterOptions
func (s *Service) applyFiltersToOptions(filter *strategy.FilterOptions, filters map[string]interface{}) {
	for filterID, filterValue := range filters {
		if filterValue == nil {
			continue
		}

		switch filterID {
		case "risk_tolerance":
			if val, ok := filterValue.(string); ok {
				rt := strategy.RiskTolerance(val)
				filter.RiskTolerance = &rt
			}

		case "market_type":
			if val, ok := filterValue.(string); ok {
				mt := strategy.MarketType(val)
				filter.MarketType = &mt
			}

		case "rebalance_frequency":
			// Can be single value or array for multiselect
			switch v := filterValue.(type) {
			case string:
				rf := strategy.RebalanceFrequency(v)
				filter.RebalanceFrequency = &rf
			case []interface{}:
				filter.RebalanceFrequencies = make([]strategy.RebalanceFrequency, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						filter.RebalanceFrequencies = append(filter.RebalanceFrequencies, strategy.RebalanceFrequency(str))
					}
				}
			}

		case "min_capital":
			if val, ok := filterValue.(float64); ok {
				minCap := decimal.NewFromFloat(val)
				filter.MinCapital = &minCap
			}

		case "max_capital":
			if val, ok := filterValue.(float64); ok {
				maxCap := decimal.NewFromFloat(val)
				filter.MaxCapital = &maxCap
			}

		case "min_pnl_percent":
			if val, ok := filterValue.(float64); ok {
				minPnL := decimal.NewFromFloat(val)
				filter.MinPnLPercent = &minPnL
			}

		case "max_pnl_percent":
			if val, ok := filterValue.(float64); ok {
				maxPnL := decimal.NewFromFloat(val)
				filter.MaxPnLPercent = &maxPnL
			}
		}
	}
}

// GetStrategiesScopes returns counts for each scope/status
// Uses SQL GROUP BY for efficiency
func (s *Service) GetStrategiesScopes(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	// Get counts by status from repository (SQL GROUP BY)
	statusCounts, err := s.strategyRepo.CountByStatus(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count strategies by status")
	}

	// Calculate total count for "all" scope
	totalCount := 0
	for _, count := range statusCounts {
		totalCount += count
	}

	// Map status counts to scope IDs
	result := map[string]int{
		"all":    totalCount,
		"active": statusCounts[strategy.StrategyActive],
		"paused": statusCounts[strategy.StrategyPaused],
		"closed": statusCounts[strategy.StrategyClosed],
	}

	return result, nil
}

// GetStrategyByID retrieves a strategy by ID
// Delegates to domain service
func (s *Service) GetStrategyByID(ctx context.Context, strategyID uuid.UUID) (*strategy.Strategy, error) {
	return s.domainService.GetByID(ctx, strategyID)
}

// GetTotalExposure returns total allocated capital + current equity across all active strategies
func (s *Service) GetTotalExposure(ctx context.Context, userID uuid.UUID) (allocated, equity decimal.Decimal, err error) {
	allocated, err = s.strategyRepo.GetTotalAllocatedCapital(ctx, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, errors.Wrap(err, "failed to get total allocated capital")
	}

	equity, err = s.strategyRepo.GetTotalCurrentEquity(ctx, userID)
	if err != nil {
		return decimal.Zero, decimal.Zero, errors.Wrap(err, "failed to get total current equity")
	}

	return allocated, equity, nil
}

// ReconcileStrategy verifies strategy balance matches transaction history
// Returns true if balanced, false if mismatch (requires investigation)
func (s *Service) ReconcileStrategy(ctx context.Context, strategyID uuid.UUID) (bool, error) {
	// Get strategy
	strat, err := s.strategyRepo.GetByID(ctx, strategyID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get strategy")
	}

	// Get latest balance from transactions
	latestBalance, err := s.transactionRepo.GetLatestBalance(ctx, strategyID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get latest balance")
	}

	// Compare
	if !strat.CashReserve.Equal(latestBalance) {
		s.log.Warnw("Strategy balance mismatch detected",
			"strategy_id", strategyID,
			"strategy_cash_reserve", strat.CashReserve,
			"transactions_latest_balance", latestBalance,
			"difference", strat.CashReserve.Sub(latestBalance),
		)
		return false, nil
	}

	s.log.Debugw("Strategy balance reconciled successfully",
		"strategy_id", strategyID,
		"balance", strat.CashReserve,
	)

	return true, nil
}

// UpdateStrategyParams contains parameters for updating a strategy
type UpdateStrategyParams struct {
	Name               *string
	Description        *string
	RiskTolerance      *strategy.RiskTolerance
	RebalanceFrequency *strategy.RebalanceFrequency
	TargetAllocations  []byte // Optional JSON
}

// UpdateStrategy updates strategy configuration fields
// Does NOT update financial fields (capital, equity, cash) - those are managed separately
func (s *Service) UpdateStrategy(ctx context.Context, strategyID uuid.UUID, params UpdateStrategyParams) (*strategy.Strategy, error) {
	s.log.Infow("Updating strategy",
		"strategy_id", strategyID,
	)

	// Validate
	if strategyID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get existing strategy via domain service (Clean Architecture)
	strat, err := s.domainService.GetByID(ctx, strategyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strategy")
	}

	// Only active or paused strategies can be updated
	if strat.Status == strategy.StrategyClosed {
		return nil, errors.New("cannot update closed strategy")
	}

	// Apply updates
	if params.Name != nil {
		strat.Name = *params.Name
	}
	if params.Description != nil {
		strat.Description = *params.Description
	}
	if params.RiskTolerance != nil {
		if !params.RiskTolerance.Valid() {
			return nil, errors.New("invalid risk_tolerance")
		}
		strat.RiskTolerance = *params.RiskTolerance
	}
	if params.RebalanceFrequency != nil {
		if !params.RebalanceFrequency.Valid() {
			return nil, errors.New("invalid rebalance_frequency")
		}
		strat.RebalanceFrequency = *params.RebalanceFrequency
	}
	if params.TargetAllocations != nil {
		strat.TargetAllocations = params.TargetAllocations
	}

	// Use domain service to update
	if err := s.domainService.Update(ctx, strat); err != nil {
		return nil, errors.Wrap(err, "failed to update strategy")
	}

	s.log.Infow("Strategy updated successfully",
		"strategy_id", strategyID,
	)

	return strat, nil
}

// PauseStrategy pauses an active strategy
// Paused strategies stop generating new trades but maintain positions
func (s *Service) PauseStrategy(ctx context.Context, strategyID uuid.UUID) (*strategy.Strategy, error) {
	s.log.Infow("Pausing strategy", "strategy_id", strategyID)

	if strategyID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get strategy via domain service (Clean Architecture)
	strat, err := s.domainService.GetByID(ctx, strategyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strategy")
	}

	// Can only pause active strategies
	if strat.Status != strategy.StrategyActive {
		return nil, errors.New("can only pause active strategies")
	}

	// Use entity method
	strat.Pause()

	// Update in DB via domain service
	if err := s.domainService.Update(ctx, strat); err != nil {
		return nil, errors.Wrap(err, "failed to pause strategy")
	}

	s.log.Infow("Strategy paused successfully", "strategy_id", strategyID)
	return strat, nil
}

// ResumeStrategy resumes a paused strategy
func (s *Service) ResumeStrategy(ctx context.Context, strategyID uuid.UUID) (*strategy.Strategy, error) {
	s.log.Infow("Resuming strategy", "strategy_id", strategyID)

	if strategyID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get strategy via domain service (Clean Architecture)
	strat, err := s.domainService.GetByID(ctx, strategyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strategy")
	}

	// Can only resume paused strategies
	if strat.Status != strategy.StrategyPaused {
		return nil, errors.New("can only resume paused strategies")
	}

	// Use entity method
	strat.Resume()

	// Update in DB via domain service
	if err := s.domainService.Update(ctx, strat); err != nil {
		return nil, errors.Wrap(err, "failed to resume strategy")
	}

	s.log.Infow("Strategy resumed successfully", "strategy_id", strategyID)
	return strat, nil
}

// CloseStrategy closes a strategy
// Closed strategies cannot be reopened
// Note: This should only be called after all positions are closed
func (s *Service) CloseStrategy(ctx context.Context, strategyID uuid.UUID) (*strategy.Strategy, error) {
	s.log.Infow("Closing strategy", "strategy_id", strategyID)

	if strategyID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get strategy via domain service (Clean Architecture)
	strat, err := s.domainService.GetByID(ctx, strategyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get strategy")
	}

	// Cannot close already closed strategy
	if strat.Status == strategy.StrategyClosed {
		return nil, errors.New("strategy is already closed")
	}

	// Use domain service close method
	if err := s.domainService.Close(ctx, strategyID); err != nil {
		return nil, errors.Wrap(err, "failed to close strategy")
	}

	// Reload strategy to get updated state via domain service
	strat, err = s.domainService.GetByID(ctx, strategyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reload strategy")
	}

	s.log.Infow("Strategy closed successfully",
		"strategy_id", strategyID,
		"final_equity", strat.CurrentEquity,
		"total_pnl", strat.TotalPnL,
	)

	return strat, nil
}

// DeleteStrategy permanently deletes a strategy
// This is a hard delete and should only be used in special cases
// Normal workflow is to close strategies, not delete them
func (s *Service) DeleteStrategy(ctx context.Context, strategyID uuid.UUID) error {
	s.log.Warnw("Deleting strategy (hard delete)", "strategy_id", strategyID)

	if strategyID == uuid.Nil {
		return errors.ErrInvalidInput
	}

	// Get strategy to check status via domain service (Clean Architecture)
	strat, err := s.domainService.GetByID(ctx, strategyID)
	if err != nil {
		return errors.Wrap(err, "failed to get strategy")
	}

	// Only allow deletion of closed strategies to prevent data loss
	if strat.Status != strategy.StrategyClosed {
		return errors.New("can only delete closed strategies")
	}

	// Use domain service to delete
	if err := s.domainService.Delete(ctx, strategyID); err != nil {
		return errors.Wrap(err, "failed to delete strategy")
	}

	s.log.Infow("Strategy deleted successfully", "strategy_id", strategyID)
	return nil
}

// BatchDeleteStrategies deletes multiple strategies in batch
// Returns the number of successfully deleted strategies
// Only closed strategies can be deleted
func (s *Service) BatchDeleteStrategies(ctx context.Context, strategyIDs []uuid.UUID) (int, error) {
	s.log.Infow("Batch deleting strategies",
		"count", len(strategyIDs),
	)

	if len(strategyIDs) == 0 {
		return 0, errors.ErrInvalidInput
	}

	deleted := 0
	var lastErr error

	for _, strategyID := range strategyIDs {
		if err := s.DeleteStrategy(ctx, strategyID); err != nil {
			s.log.Warnw("Failed to delete strategy in batch",
				"strategy_id", strategyID,
				"error", err,
			)
			lastErr = err
			continue
		}
		deleted++
	}

	s.log.Infow("Batch delete completed",
		"deleted", deleted,
		"total", len(strategyIDs),
	)

	// Return error only if all deletions failed
	if deleted == 0 && lastErr != nil {
		return 0, lastErr
	}

	return deleted, nil
}

// BatchPauseStrategies pauses multiple strategies in batch
// Returns the number of successfully paused strategies
func (s *Service) BatchPauseStrategies(ctx context.Context, strategyIDs []uuid.UUID) (int, error) {
	s.log.Infow("Batch pausing strategies",
		"count", len(strategyIDs),
	)

	if len(strategyIDs) == 0 {
		return 0, errors.ErrInvalidInput
	}

	paused := 0
	var lastErr error

	for _, strategyID := range strategyIDs {
		if _, err := s.PauseStrategy(ctx, strategyID); err != nil {
			s.log.Warnw("Failed to pause strategy in batch",
				"strategy_id", strategyID,
				"error", err,
			)
			lastErr = err
			continue
		}
		paused++
	}

	s.log.Infow("Batch pause completed",
		"paused", paused,
		"total", len(strategyIDs),
	)

	// Return error only if all pauses failed
	if paused == 0 && lastErr != nil {
		return 0, lastErr
	}

	return paused, nil
}

// BatchResumeStrategies resumes multiple strategies in batch
// Returns the number of successfully resumed strategies
func (s *Service) BatchResumeStrategies(ctx context.Context, strategyIDs []uuid.UUID) (int, error) {
	s.log.Infow("Batch resuming strategies",
		"count", len(strategyIDs),
	)

	if len(strategyIDs) == 0 {
		return 0, errors.ErrInvalidInput
	}

	resumed := 0
	var lastErr error

	for _, strategyID := range strategyIDs {
		if _, err := s.ResumeStrategy(ctx, strategyID); err != nil {
			s.log.Warnw("Failed to resume strategy in batch",
				"strategy_id", strategyID,
				"error", err,
			)
			lastErr = err
			continue
		}
		resumed++
	}

	s.log.Infow("Batch resume completed",
		"resumed", resumed,
		"total", len(strategyIDs),
	)

	// Return error only if all resumes failed
	if resumed == 0 && lastErr != nil {
		return 0, lastErr
	}

	return resumed, nil
}

// BatchCloseStrategies closes multiple strategies in batch
// Returns the number of successfully closed strategies
func (s *Service) BatchCloseStrategies(ctx context.Context, strategyIDs []uuid.UUID) (int, error) {
	s.log.Infow("Batch closing strategies",
		"count", len(strategyIDs),
	)

	if len(strategyIDs) == 0 {
		return 0, errors.ErrInvalidInput
	}

	closed := 0
	var lastErr error

	for _, strategyID := range strategyIDs {
		if _, err := s.CloseStrategy(ctx, strategyID); err != nil {
			s.log.Warnw("Failed to close strategy in batch",
				"strategy_id", strategyID,
				"error", err,
			)
			lastErr = err
			continue
		}
		closed++
	}

	s.log.Infow("Batch close completed",
		"closed", closed,
		"total", len(strategyIDs),
	)

	// Return error only if all closes failed
	if closed == 0 && lastErr != nil {
		return 0, lastErr
	}

	return closed, nil
}
