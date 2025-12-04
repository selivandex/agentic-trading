package seeds

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/strategy"
)

// StrategyBuilder provides a fluent API for creating Strategy entities
type StrategyBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *strategy.Strategy
}

// NewStrategyBuilder creates a new StrategyBuilder with sensible defaults
func NewStrategyBuilder(db DBTX, ctx context.Context) *StrategyBuilder {
	now := time.Now()
	allocations := strategy.TargetAllocations{
		"BTC/USDT": 0.5,
		"ETH/USDT": 0.3,
		"SOL/USDT": 0.2,
	}
	allocationsJSON, _ := json.Marshal(allocations)

	return &StrategyBuilder{
		db:  db,
		ctx: ctx,
		entity: &strategy.Strategy{
			ID:                 uuid.New(),
			UserID:             uuid.Nil, // Must be set
			Name:               "Test Strategy",
			Description:        "Test trading strategy",
			Status:             strategy.StrategyActive,
			AllocatedCapital:   decimal.NewFromInt(10000),
			CurrentEquity:      decimal.NewFromInt(10000),
			CashReserve:        decimal.NewFromInt(5000),
			MarketType:         strategy.MarketSpot,
			RiskTolerance:      strategy.RiskModerate,
			RebalanceFrequency: strategy.RebalanceWeekly,
			TargetAllocations:  allocationsJSON,
			TotalPnL:           decimal.Zero,
			TotalPnLPercent:    decimal.Zero,
			SharpeRatio:        nil,
			MaxDrawdown:        nil,
			WinRate:            nil,
			CreatedAt:          now,
			UpdatedAt:          now,
			ClosedAt:           nil,
			LastRebalancedAt:   nil,
			ReasoningLog:       json.RawMessage("{}"),
		},
	}
}

// WithID sets a specific ID
func (b *StrategyBuilder) WithID(id uuid.UUID) *StrategyBuilder {
	b.entity.ID = id
	return b
}

// WithUserID sets the user ID (required)
func (b *StrategyBuilder) WithUserID(userID uuid.UUID) *StrategyBuilder {
	b.entity.UserID = userID
	return b
}

// WithName sets the strategy name
func (b *StrategyBuilder) WithName(name string) *StrategyBuilder {
	b.entity.Name = name
	return b
}

// WithDescription sets the description
func (b *StrategyBuilder) WithDescription(desc string) *StrategyBuilder {
	b.entity.Description = desc
	return b
}

// WithStatus sets the strategy status
func (b *StrategyBuilder) WithStatus(status strategy.StrategyStatus) *StrategyBuilder {
	b.entity.Status = status
	return b
}

// WithActive sets the strategy to active
func (b *StrategyBuilder) WithActive() *StrategyBuilder {
	b.entity.Status = strategy.StrategyActive
	return b
}

// WithPaused sets the strategy to paused
func (b *StrategyBuilder) WithPaused() *StrategyBuilder {
	b.entity.Status = strategy.StrategyPaused
	return b
}

// WithClosed sets the strategy to closed
func (b *StrategyBuilder) WithClosed() *StrategyBuilder {
	now := time.Now()
	b.entity.Status = strategy.StrategyClosed
	b.entity.ClosedAt = &now
	return b
}

// WithCapital sets the allocated capital
func (b *StrategyBuilder) WithCapital(amount decimal.Decimal) *StrategyBuilder {
	b.entity.AllocatedCapital = amount
	b.entity.CurrentEquity = amount
	b.entity.CashReserve = amount.Div(decimal.NewFromInt(2)) // 50% cash by default
	return b
}

// WithCashReserve sets the cash reserve
func (b *StrategyBuilder) WithCashReserve(amount decimal.Decimal) *StrategyBuilder {
	b.entity.CashReserve = amount
	return b
}

// WithCurrentEquity sets the current equity
func (b *StrategyBuilder) WithCurrentEquity(amount decimal.Decimal) *StrategyBuilder {
	b.entity.CurrentEquity = amount
	return b
}

// WithMarketType sets the market type
func (b *StrategyBuilder) WithMarketType(marketType strategy.MarketType) *StrategyBuilder {
	b.entity.MarketType = marketType
	return b
}

// WithSpot sets market type to spot
func (b *StrategyBuilder) WithSpot() *StrategyBuilder {
	b.entity.MarketType = strategy.MarketSpot
	return b
}

// WithFutures sets market type to futures
func (b *StrategyBuilder) WithFutures() *StrategyBuilder {
	b.entity.MarketType = strategy.MarketFutures
	return b
}

// WithRiskTolerance sets the risk tolerance
func (b *StrategyBuilder) WithRiskTolerance(risk strategy.RiskTolerance) *StrategyBuilder {
	b.entity.RiskTolerance = risk
	return b
}

// WithConservativeRisk sets conservative risk tolerance
func (b *StrategyBuilder) WithConservativeRisk() *StrategyBuilder {
	b.entity.RiskTolerance = strategy.RiskConservative
	return b
}

// WithModerateRisk sets moderate risk tolerance
func (b *StrategyBuilder) WithModerateRisk() *StrategyBuilder {
	b.entity.RiskTolerance = strategy.RiskModerate
	return b
}

// WithAggressiveRisk sets aggressive risk tolerance
func (b *StrategyBuilder) WithAggressiveRisk() *StrategyBuilder {
	b.entity.RiskTolerance = strategy.RiskAggressive
	return b
}

// WithRebalanceFrequency sets the rebalance frequency
func (b *StrategyBuilder) WithRebalanceFrequency(freq strategy.RebalanceFrequency) *StrategyBuilder {
	b.entity.RebalanceFrequency = freq
	return b
}

// WithTargetAllocations sets the target allocations
func (b *StrategyBuilder) WithTargetAllocations(allocations strategy.TargetAllocations) *StrategyBuilder {
	allocationsJSON, _ := json.Marshal(allocations)
	b.entity.TargetAllocations = allocationsJSON
	return b
}

// WithPnL sets PnL metrics
func (b *StrategyBuilder) WithPnL(totalPnL, pnlPercent decimal.Decimal) *StrategyBuilder {
	b.entity.TotalPnL = totalPnL
	b.entity.TotalPnLPercent = pnlPercent
	return b
}

// WithPerformanceMetrics sets all performance metrics
func (b *StrategyBuilder) WithPerformanceMetrics(sharpe, maxDrawdown, winRate decimal.Decimal) *StrategyBuilder {
	b.entity.SharpeRatio = &sharpe
	b.entity.MaxDrawdown = &maxDrawdown
	b.entity.WinRate = &winRate
	return b
}

// Build returns the built entity without inserting to DB
func (b *StrategyBuilder) Build() *strategy.Strategy {
	return b.entity
}

// Insert inserts the strategy into the database and returns the entity
func (b *StrategyBuilder) Insert() (*strategy.Strategy, error) {
	if b.entity.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}

	query := `
		INSERT INTO user_strategies (
			id, user_id, name, description, status,
			allocated_capital, current_equity, cash_reserve,
			market_type, risk_tolerance, rebalance_frequency, target_allocations,
			total_pnl, total_pnl_percent, sharpe_ratio, max_drawdown, win_rate,
			created_at, updated_at, closed_at, last_rebalanced_at, reasoning_log
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.UserID,
		b.entity.Name,
		b.entity.Description,
		b.entity.Status,
		b.entity.AllocatedCapital,
		b.entity.CurrentEquity,
		b.entity.CashReserve,
		b.entity.MarketType,
		b.entity.RiskTolerance,
		b.entity.RebalanceFrequency,
		b.entity.TargetAllocations,
		b.entity.TotalPnL,
		b.entity.TotalPnLPercent,
		b.entity.SharpeRatio,
		b.entity.MaxDrawdown,
		b.entity.WinRate,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
		b.entity.ClosedAt,
		b.entity.LastRebalancedAt,
		b.entity.ReasoningLog,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert strategy: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the strategy and panics on error (useful for tests)
func (b *StrategyBuilder) MustInsert() *strategy.Strategy {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}

