package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
)

// PositionBuilder provides a fluent API for creating Position entities
type PositionBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *position.Position
}

// NewPositionBuilder creates a new PositionBuilder with sensible defaults
func NewPositionBuilder(db DBTX, ctx context.Context) *PositionBuilder {
	now := time.Now()
	return &PositionBuilder{
		db:  db,
		ctx: ctx,
		entity: &position.Position{
			ID:                uuid.New(),
			UserID:            uuid.Nil, // Must be set
			StrategyID:        nil,
			ExchangeAccountID: uuid.Nil, // Must be set
			Symbol:            "BTC/USDT",
			MarketType:        "spot",
			Side:              position.PositionLong,
			Size:              decimal.NewFromFloat(1.0),
			EntryPrice:        decimal.NewFromFloat(40000.0),
			CurrentPrice:      decimal.NewFromFloat(40000.0),
			LiquidationPrice:  decimal.Zero,
			Leverage:          1,
			MarginMode:        "cross",
			UnrealizedPnL:     decimal.Zero,
			UnrealizedPnLPct:  decimal.Zero,
			RealizedPnL:       decimal.Zero,
			StopLossPrice:     decimal.Zero,
			TakeProfitPrice:   decimal.Zero,
			TrailingStopPct:   decimal.Zero,
			StopLossOrderID:   nil,
			TakeProfitOrderID: nil,
			OpenReasoning:     "Test position",
			Status:            position.PositionOpen,
			OpenedAt:          now,
			ClosedAt:          nil,
			UpdatedAt:         now,
		},
	}
}

// WithID sets a specific ID
func (b *PositionBuilder) WithID(id uuid.UUID) *PositionBuilder {
	b.entity.ID = id
	return b
}

// WithUserID sets the user ID (required)
func (b *PositionBuilder) WithUserID(userID uuid.UUID) *PositionBuilder {
	b.entity.UserID = userID
	return b
}

// WithStrategyID sets the strategy ID
func (b *PositionBuilder) WithStrategyID(strategyID uuid.UUID) *PositionBuilder {
	b.entity.StrategyID = &strategyID
	return b
}

// WithExchangeAccountID sets the exchange account ID (required)
func (b *PositionBuilder) WithExchangeAccountID(accountID uuid.UUID) *PositionBuilder {
	b.entity.ExchangeAccountID = accountID
	return b
}

// WithSymbol sets the trading symbol
func (b *PositionBuilder) WithSymbol(symbol string) *PositionBuilder {
	b.entity.Symbol = symbol
	return b
}

// WithMarketType sets the market type
func (b *PositionBuilder) WithMarketType(marketType string) *PositionBuilder {
	b.entity.MarketType = marketType
	return b
}

// WithSide sets the position side
func (b *PositionBuilder) WithSide(side position.PositionSide) *PositionBuilder {
	b.entity.Side = side
	return b
}

// WithLong sets position to long
func (b *PositionBuilder) WithLong() *PositionBuilder {
	b.entity.Side = position.PositionLong
	return b
}

// WithShort sets position to short
func (b *PositionBuilder) WithShort() *PositionBuilder {
	b.entity.Side = position.PositionShort
	return b
}

// WithSize sets the position size
func (b *PositionBuilder) WithSize(size decimal.Decimal) *PositionBuilder {
	b.entity.Size = size
	return b
}

// WithEntryPrice sets the entry price
func (b *PositionBuilder) WithEntryPrice(price decimal.Decimal) *PositionBuilder {
	b.entity.EntryPrice = price
	b.entity.CurrentPrice = price // Default current price to entry price
	return b
}

// WithCurrentPrice sets the current price
func (b *PositionBuilder) WithCurrentPrice(price decimal.Decimal) *PositionBuilder {
	b.entity.CurrentPrice = price
	return b
}

// WithLeverage sets the leverage
func (b *PositionBuilder) WithLeverage(leverage int) *PositionBuilder {
	b.entity.Leverage = leverage
	return b
}

// WithMarginMode sets the margin mode
func (b *PositionBuilder) WithMarginMode(mode string) *PositionBuilder {
	b.entity.MarginMode = mode
	return b
}

// WithPnL sets PnL values
func (b *PositionBuilder) WithPnL(unrealized, unrealizedPct, realized decimal.Decimal) *PositionBuilder {
	b.entity.UnrealizedPnL = unrealized
	b.entity.UnrealizedPnLPct = unrealizedPct
	b.entity.RealizedPnL = realized
	return b
}

// WithStopLoss sets stop loss price
func (b *PositionBuilder) WithStopLoss(price decimal.Decimal) *PositionBuilder {
	b.entity.StopLossPrice = price
	return b
}

// WithTakeProfit sets take profit price
func (b *PositionBuilder) WithTakeProfit(price decimal.Decimal) *PositionBuilder {
	b.entity.TakeProfitPrice = price
	return b
}

// WithTrailingStop sets trailing stop percentage
func (b *PositionBuilder) WithTrailingStop(pct decimal.Decimal) *PositionBuilder {
	b.entity.TrailingStopPct = pct
	return b
}

// WithStatus sets the position status
func (b *PositionBuilder) WithStatus(status position.PositionStatus) *PositionBuilder {
	b.entity.Status = status
	return b
}

// WithClosed sets the position to closed
func (b *PositionBuilder) WithClosed() *PositionBuilder {
	now := time.Now()
	b.entity.Status = position.PositionClosed
	b.entity.ClosedAt = &now
	return b
}

// WithReasoning sets the open reasoning
func (b *PositionBuilder) WithReasoning(reasoning string) *PositionBuilder {
	b.entity.OpenReasoning = reasoning
	return b
}

// Build returns the built entity without inserting to DB
func (b *PositionBuilder) Build() *position.Position {
	return b.entity
}

// Insert inserts the position into the database and returns the entity
func (b *PositionBuilder) Insert() (*position.Position, error) {
	if b.entity.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if b.entity.ExchangeAccountID == uuid.Nil {
		return nil, fmt.Errorf("exchange_account_id is required")
	}

	query := `
		INSERT INTO positions (
			id, user_id, strategy_id, exchange_account_id, symbol, market_type, side,
			size, entry_price, current_price, liquidation_price, leverage, margin_mode,
			unrealized_pnl, unrealized_pnl_pct, realized_pnl,
			stop_loss_price, take_profit_price, trailing_stop_pct,
			stop_loss_order_id, take_profit_order_id,
			open_reasoning, status, opened_at, closed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.UserID,
		b.entity.StrategyID,
		b.entity.ExchangeAccountID,
		b.entity.Symbol,
		b.entity.MarketType,
		b.entity.Side,
		b.entity.Size,
		b.entity.EntryPrice,
		b.entity.CurrentPrice,
		b.entity.LiquidationPrice,
		b.entity.Leverage,
		b.entity.MarginMode,
		b.entity.UnrealizedPnL,
		b.entity.UnrealizedPnLPct,
		b.entity.RealizedPnL,
		b.entity.StopLossPrice,
		b.entity.TakeProfitPrice,
		b.entity.TrailingStopPct,
		b.entity.StopLossOrderID,
		b.entity.TakeProfitOrderID,
		b.entity.OpenReasoning,
		b.entity.Status,
		b.entity.OpenedAt,
		b.entity.ClosedAt,
		b.entity.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert position: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the position and panics on error (useful for tests)
func (b *PositionBuilder) MustInsert() *position.Position {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}

