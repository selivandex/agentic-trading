package shared

import (
	"context"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/risk"
	"prometheus/pkg/logger"
)

// RiskEngine interface to avoid circular dependency
type RiskEngine interface {
	CanTrade(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Deps bundles dependencies required by concrete tool implementations.
type Deps struct {
	MarketDataRepo      market_data.Repository
	OrderRepo           order.Repository
	PositionRepo        position.Repository
	ExchangeAccountRepo exchange_account.Repository
	MemoryRepo          memory.Repository
	RiskRepo            risk.Repository
	RiskEngine          RiskEngine
	Redis               RedisClient
	Log                 *logger.Logger
}

// RedisClient interface to avoid circular dependency
type RedisClient interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// HasMarketData reports whether the market data repository is available.
func (d Deps) HasMarketData() bool {
	return d.MarketDataRepo != nil
}

// HasTradingData reports whether trading repositories are wired.
func (d Deps) HasTradingData() bool {
	return d.OrderRepo != nil && d.PositionRepo != nil && d.ExchangeAccountRepo != nil
}

// HasRiskEngine reports whether the risk engine is available.
func (d Deps) HasRiskEngine() bool {
	return d.RiskEngine != nil
}
