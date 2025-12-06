package strategy

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FilterOptions defines filter criteria for strategy queries
type FilterOptions struct {
	UserID   *uuid.UUID
	Status   *StrategyStatus
	Search   *string          // Search by name
	Statuses []StrategyStatus // Filter by multiple statuses

	// Additional filters
	RiskTolerance        *RiskTolerance
	RiskTolerances       []RiskTolerance
	MarketType           *MarketType
	MarketTypes          []MarketType
	RebalanceFrequency   *RebalanceFrequency
	RebalanceFrequencies []RebalanceFrequency
	MinCapital           *decimal.Decimal
	MaxCapital           *decimal.Decimal
	MinPnLPercent        *decimal.Decimal
	MaxPnLPercent        *decimal.Decimal
}

// Repository defines operations for strategy persistence
type Repository interface {
	// Create creates a new strategy
	Create(ctx context.Context, strategy *Strategy) error

	// GetByID retrieves a strategy by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Strategy, error)

	// GetByUserID retrieves all strategies for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Strategy, error)

	// GetActiveByUserID retrieves all active strategies for a user
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*Strategy, error)

	// GetAllActive retrieves all active strategies across all users
	GetAllActive(ctx context.Context) ([]*Strategy, error)

	// GetWithFilter retrieves strategies with filter options (SQL WHERE)
	GetWithFilter(ctx context.Context, filter FilterOptions) ([]*Strategy, error)

	// CountByStatus returns count of strategies grouped by status for a user
	CountByStatus(ctx context.Context, userID uuid.UUID) (map[StrategyStatus]int, error)

	// Update updates an existing strategy
	Update(ctx context.Context, strategy *Strategy) error

	// UpdateEquity updates strategy equity and PnL fields
	// Optimized method to avoid loading full strategy
	UpdateEquity(ctx context.Context, strategyID uuid.UUID, currentEquity, cashReserve, totalPnL, totalPnLPercent decimal.Decimal) error

	// Delete soft-deletes a strategy (sets status to closed)
	Delete(ctx context.Context, id uuid.UUID) error

	// GetTotalAllocatedCapital returns sum of allocated capital across all active strategies
	GetTotalAllocatedCapital(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)

	// GetTotalCurrentEquity returns sum of current equity across all active strategies
	GetTotalCurrentEquity(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)

	// GetRangeStats returns min/max values for numeric fields across all strategies
	// Used for dynamic filter ranges
	GetRangeStats(ctx context.Context, userID *uuid.UUID) (*RangeStats, error)
}

// TransactionRepository defines operations for strategy transaction persistence
type TransactionRepository interface {
	// Create creates a new transaction
	Create(ctx context.Context, tx *Transaction) error

	// GetByID retrieves a transaction by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)

	// GetByStrategyID retrieves all transactions for a strategy
	GetByStrategyID(ctx context.Context, strategyID uuid.UUID, limit int) ([]*Transaction, error)

	// GetByUserID retrieves all transactions for a user
	GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Transaction, error)

	// GetByPositionID retrieves all transactions related to a position
	GetByPositionID(ctx context.Context, positionID uuid.UUID) ([]*Transaction, error)

	// GetByDateRange retrieves transactions in a date range
	GetByDateRange(ctx context.Context, strategyID uuid.UUID, start, end time.Time) ([]*Transaction, error)

	// GetLatestBalance gets the most recent balance_after for a strategy
	// Used for reconciliation and validation
	GetLatestBalance(ctx context.Context, strategyID uuid.UUID) (decimal.Decimal, error)

	// GetTransactionStats returns aggregate stats for a strategy
	GetTransactionStats(ctx context.Context, strategyID uuid.UUID) (*TransactionStats, error)
}

// TransactionStats contains aggregate transaction statistics
type TransactionStats struct {
	TotalDeposits    decimal.Decimal
	TotalWithdrawals decimal.Decimal
	TotalFees        decimal.Decimal
	TotalPnL         decimal.Decimal // From position closes
	TransactionCount int
}

// RangeStats contains min/max values for numeric fields
// Used for dynamic filter ranges in the frontend
type RangeStats struct {
	MinCapital    decimal.Decimal `db:"min_capital"`
	MaxCapital    decimal.Decimal `db:"max_capital"`
	MinPnLPercent decimal.Decimal `db:"min_pnl_percent"`
	MaxPnLPercent decimal.Decimal `db:"max_pnl_percent"`
}
