package postgres

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	domainRisk "prometheus/internal/domain/risk"
	"prometheus/pkg/logger"
)

// CorrelationRepository reads correlation data from database
// Data is populated by workers/macro/market_correlation_collector.go
type CorrelationRepository struct {
	macroRepo MacroDataReader
	cache     map[string]map[string]decimal.Decimal // simple in-memory cache
	cacheTTL  int64                                 // unix timestamp
	log       *logger.Logger
}

// MacroDataReader provides access to macro/correlation data
// This should be implemented by internal/repository/postgres/macro_repository.go
type MacroDataReader interface {
	GetLatestCorrelation(ctx context.Context, cryptoSymbol, asset string) (float64, error)
	GetAllCorrelations(ctx context.Context, cryptoSymbol string) (map[string]float64, error)
}

// Ensure CorrelationRepository implements the interface
var _ domainRisk.CorrelationService = (*CorrelationRepository)(nil)

// NewCorrelationRepository creates a correlation repository that reads from database
func NewCorrelationRepository(macroRepo MacroDataReader, log *logger.Logger) *CorrelationRepository {
	return &CorrelationRepository{
		macroRepo: macroRepo,
		cache:     make(map[string]map[string]decimal.Decimal),
		log:       log,
	}
}

// GetCorrelations returns correlations from database (populated by worker)
func (r *CorrelationRepository) GetCorrelations(ctx context.Context, symbol string) (map[string]decimal.Decimal, error) {
	// Check cache (5 min TTL)
	now := time.Now().Unix()
	if r.cacheTTL > now {
		if cached, ok := r.cache[symbol]; ok {
			r.log.Debug("Correlation cache hit", "symbol", symbol)
			return cached, nil
		}
	}

	// Fetch from database
	correlations, err := r.macroRepo.GetAllCorrelations(ctx, symbol)
	if err != nil {
		r.log.Warnf("Failed to fetch correlations from DB for %s: %v, using defaults", symbol, err)
		// Return default correlations as fallback
		return r.getDefaultCorrelations(symbol), nil
	}

	if len(correlations) == 0 {
		r.log.Debug("No correlations found in DB", "symbol", symbol)
		return r.getDefaultCorrelations(symbol), nil
	}

	// Convert to decimal
	result := make(map[string]decimal.Decimal)
	for asset, corr := range correlations {
		result[asset] = decimal.NewFromFloat(corr)
	}

	// Update cache
	r.cache[symbol] = result
	r.cacheTTL = now + 300 // 5 minutes

	r.log.Debug("Correlations loaded from DB", "symbol", symbol, "count", len(result))
	return result, nil
}

// GetBTCCorrelation returns BTC correlation from database
func (r *CorrelationRepository) GetBTCCorrelation(ctx context.Context, symbol string) (decimal.Decimal, error) {
	if symbol == "BTC/USDT" {
		return decimal.NewFromFloat(1.0), nil
	}

	corr, err := r.macroRepo.GetLatestCorrelation(ctx, symbol, "BTC/USDT")
	if err != nil {
		r.log.Warnf("Failed to fetch BTC correlation for %s: %v, using default", symbol, err)
		// Fallback to default
		return decimal.NewFromFloat(0.70), nil
	}

	return decimal.NewFromFloat(corr), nil
}

// getDefaultCorrelations returns sensible default correlations when DB is empty
func (r *CorrelationRepository) getDefaultCorrelations(symbol string) map[string]decimal.Decimal {
	defaults := map[string]decimal.Decimal{
		"BTC/USDT": decimal.NewFromFloat(0.70), // most crypto correlated with BTC
	}

	// If this is a major coin, add more specific correlations
	switch symbol {
	case "BTC/USDT":
		defaults = map[string]decimal.Decimal{
			"ETH/USDT": decimal.NewFromFloat(0.85),
			"BNB/USDT": decimal.NewFromFloat(0.75),
			"SOL/USDT": decimal.NewFromFloat(0.70),
		}
	case "ETH/USDT":
		defaults = map[string]decimal.Decimal{
			"BTC/USDT": decimal.NewFromFloat(0.85),
			"BNB/USDT": decimal.NewFromFloat(0.80),
			"SOL/USDT": decimal.NewFromFloat(0.75),
		}
	}

	return defaults
}

// ClearCache clears the correlation cache
func (r *CorrelationRepository) ClearCache() {
	r.cache = make(map[string]map[string]decimal.Decimal)
	r.cacheTTL = 0
	r.log.Debug("Correlation cache cleared")
}

