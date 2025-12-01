package strategy

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Strategy represents a user's trading strategy (portfolio)
// This is the main entity for tracking allocated capital, current equity, and PnL
type Strategy struct {
	ID     uuid.UUID `db:"id"`
	UserID uuid.UUID `db:"user_id"`

	// Strategy metadata
	Name        string         `db:"name"`
	Description string         `db:"description"`
	Status      StrategyStatus `db:"status"`

	// Capital allocation (key fields for PnL tracking)
	AllocatedCapital decimal.Decimal `db:"allocated_capital"` // Initial capital (fixed, never changes)
	CurrentEquity    decimal.Decimal `db:"current_equity"`    // Current total value (cash + positions)
	CashReserve      decimal.Decimal `db:"cash_reserve"`      // Unallocated cash (free balance)

	// Strategy configuration
	RiskTolerance      RiskTolerance      `db:"risk_tolerance"`
	RebalanceFrequency RebalanceFrequency `db:"rebalance_frequency"`
	TargetAllocations  json.RawMessage    `db:"target_allocations"` // {"BTC/USDT": 0.5, "ETH/USDT": 0.3}

	// Performance metrics (calculated periodically)
	TotalPnL        decimal.Decimal  `db:"total_pnl"`         // CurrentEquity - AllocatedCapital
	TotalPnLPercent decimal.Decimal  `db:"total_pnl_percent"` // (CurrentEquity / AllocatedCapital - 1) * 100
	SharpeRatio     *decimal.Decimal `db:"sharpe_ratio"`      // Risk-adjusted returns
	MaxDrawdown     *decimal.Decimal `db:"max_drawdown"`      // Maximum peak-to-trough decline
	WinRate         *decimal.Decimal `db:"win_rate"`          // Winning trades / total trades

	// Timestamps
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
	ClosedAt         *time.Time `db:"closed_at"`
	LastRebalancedAt *time.Time `db:"last_rebalanced_at"`

	// Reasoning log (for explainability)
	ReasoningLog json.RawMessage `db:"reasoning_log"` // Full CoT trace from portfolio creation
}

// StrategyStatus represents strategy state
type StrategyStatus string

const (
	StrategyActive StrategyStatus = "active"
	StrategyPaused StrategyStatus = "paused"
	StrategyClosed StrategyStatus = "closed"
)

// Valid checks if status is valid
func (s StrategyStatus) Valid() bool {
	return s == StrategyActive || s == StrategyPaused || s == StrategyClosed
}

// RiskTolerance represents risk profile
type RiskTolerance string

const (
	RiskConservative RiskTolerance = "conservative"
	RiskModerate     RiskTolerance = "moderate"
	RiskAggressive   RiskTolerance = "aggressive"
)

// Valid checks if risk tolerance is valid
func (r RiskTolerance) Valid() bool {
	return r == RiskConservative || r == RiskModerate || r == RiskAggressive
}

// RebalanceFrequency represents rebalancing schedule
type RebalanceFrequency string

const (
	RebalanceDaily   RebalanceFrequency = "daily"
	RebalanceWeekly  RebalanceFrequency = "weekly"
	RebalanceMonthly RebalanceFrequency = "monthly"
	RebalanceNever   RebalanceFrequency = "never"
)

// Valid checks if rebalance frequency is valid
func (r RebalanceFrequency) Valid() bool {
	return r == RebalanceDaily || r == RebalanceWeekly || r == RebalanceMonthly || r == RebalanceNever
}

// Business logic methods

// UpdateEquity recalculates current equity from cash + positions value
// Should be called after any position change
func (s *Strategy) UpdateEquity(positionsValue decimal.Decimal) {
	s.CurrentEquity = s.CashReserve.Add(positionsValue)
	s.calculatePnL()
	s.UpdatedAt = time.Now()
}

// calculatePnL calculates PnL metrics
func (s *Strategy) calculatePnL() {
	// Total PnL in USD
	s.TotalPnL = s.CurrentEquity.Sub(s.AllocatedCapital)

	// Total PnL in percentage
	if s.AllocatedCapital.IsPositive() {
		pnlPct := s.CurrentEquity.Div(s.AllocatedCapital).Sub(decimal.NewFromInt(1)).Mul(decimal.NewFromInt(100))
		s.TotalPnLPercent = pnlPct
	}
}

// AllocateCash moves cash from reserve to a position
// Called when opening a position
func (s *Strategy) AllocateCash(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}
	if s.CashReserve.LessThan(amount) {
		return ErrInsufficientCash
	}

	s.CashReserve = s.CashReserve.Sub(amount)
	s.UpdatedAt = time.Now()
	return nil
}

// ReturnCash moves cash back to reserve (from closed position)
// Called when closing a position
func (s *Strategy) ReturnCash(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	s.CashReserve = s.CashReserve.Add(amount)
	s.UpdatedAt = time.Now()
	return nil
}

// GetAvailableCash returns available cash for new positions
func (s *Strategy) GetAvailableCash() decimal.Decimal {
	return s.CashReserve
}

// GetROI returns Return on Investment as percentage
func (s *Strategy) GetROI() decimal.Decimal {
	return s.TotalPnLPercent
}

// IsActive checks if strategy is active
func (s *Strategy) IsActive() bool {
	return s.Status == StrategyActive
}

// Pause pauses the strategy
func (s *Strategy) Pause() {
	s.Status = StrategyPaused
	s.UpdatedAt = time.Now()
}

// Resume resumes the strategy
func (s *Strategy) Resume() {
	if s.Status == StrategyPaused {
		s.Status = StrategyActive
		s.UpdatedAt = time.Now()
	}
}

// Close closes the strategy
func (s *Strategy) Close() {
	now := time.Now()
	s.Status = StrategyClosed
	s.ClosedAt = &now
	s.UpdatedAt = now
}

// TargetAllocations represents asset allocation targets
type TargetAllocations map[string]float64 // {"BTC/USDT": 0.5, "ETH/USDT": 0.3}

// ParseTargetAllocations parses target allocations from JSONB
func (s *Strategy) ParseTargetAllocations() (TargetAllocations, error) {
	var allocations TargetAllocations
	if err := json.Unmarshal(s.TargetAllocations, &allocations); err != nil {
		return nil, err
	}
	return allocations, nil
}

// SetTargetAllocations sets target allocations
func (s *Strategy) SetTargetAllocations(allocations TargetAllocations) error {
	data, err := json.Marshal(allocations)
	if err != nil {
		return err
	}
	s.TargetAllocations = data
	return nil
}

// Errors
var (
	ErrInvalidAmount     = &StrategyError{Code: "invalid_amount", Message: "amount must be positive"}
	ErrInsufficientCash  = &StrategyError{Code: "insufficient_cash", Message: "not enough cash reserve"}
	ErrStrategyNotActive = &StrategyError{Code: "strategy_not_active", Message: "strategy is not active"}
)

// StrategyError represents a strategy-specific error
type StrategyError struct {
	Code    string
	Message string
}

func (e *StrategyError) Error() string {
	return e.Message
}
