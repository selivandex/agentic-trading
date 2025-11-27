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
	"prometheus/internal/domain/reasoning"
	"prometheus/internal/domain/risk"
	"prometheus/internal/domain/user"
	"prometheus/pkg/logger"
)
// RiskEngine interface to avoid circular dependency
type RiskEngine interface {
	CanTrade(ctx context.Context, userID uuid.UUID) (bool, error)
}
// EmbeddingProvider interface for generating text embeddings
// Supports multiple backends: OpenAI, Cohere, local models, etc.
type EmbeddingProvider interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	Dimensions() int
	Name() string
}
// Deps bundles dependencies required by concrete tool implementations
// Note: Stats tracking is now handled by ADK callbacks, not tool middleware
// Note: KafkaProducer is not included here - tools that need it receive it directly
type Deps struct {
	MarketDataRepo      market_data.Repository
	OrderRepo           order.Repository
	PositionRepo        position.Repository
	ExchangeAccountRepo exchange_account.Repository
	MemoryRepo          memory.Repository
	ReasoningRepo       reasoning.Repository
	RiskRepo            risk.Repository
	UserRepo            user.Repository
	RiskEngine          RiskEngine
	EmbeddingProvider   EmbeddingProvider
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
// HasMarketData reports whether the market data repository is available
func (d Deps) HasMarketData() bool {
	return d.MarketDataRepo != nil
}
// HasTradingData reports whether trading repositories are wired
func (d Deps) HasTradingData() bool {
	return d.OrderRepo != nil && d.PositionRepo != nil && d.ExchangeAccountRepo != nil
}
// HasRiskEngine reports whether the risk engine is available
func (d Deps) HasRiskEngine() bool {
	return d.RiskEngine != nil
}
