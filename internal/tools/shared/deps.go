package shared

import (
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/risk"
	"prometheus/pkg/logger"
)

// Deps bundles dependencies required by concrete tool implementations.
type Deps struct {
	MarketDataRepo      market_data.Repository
	OrderRepo           order.Repository
	PositionRepo        position.Repository
	ExchangeAccountRepo exchange_account.Repository
	MemoryRepo          memory.Repository
	RiskRepo            risk.Repository
	Log                 *logger.Logger
}

// HasMarketData reports whether the market data repository is available.
func (d Deps) HasMarketData() bool {
	return d.MarketDataRepo != nil
}

// HasTradingData reports whether trading repositories are wired.
func (d Deps) HasTradingData() bool {
	return d.OrderRepo != nil && d.PositionRepo != nil && d.ExchangeAccountRepo != nil
}
