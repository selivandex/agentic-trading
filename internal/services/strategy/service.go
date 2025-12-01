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
	UserID           uuid.UUID
	Name             string
	Description      string
	AllocatedCapital decimal.Decimal
	RiskTolerance    strategy.RiskTolerance
	ReasoningLog     []byte // JSON from portfolio architect agent
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
	if !params.RiskTolerance.Valid() {
		return nil, errors.New("invalid risk_tolerance")
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
		RiskTolerance:      params.RiskTolerance,
		RebalanceFrequency: strategy.RebalanceWeekly,
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
