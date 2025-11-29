package analysis

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"time"

	"prometheus/internal/adapters/redis"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// CacheConfig contains configuration for analysis caching
type CacheConfig struct {
	Enabled                  bool
	TTL                      time.Duration // Default TTL
	TTLVolatile              time.Duration // TTL for high volatility
	TTLNormal                time.Duration // TTL for normal volatility
	TTLQuiet                 time.Duration // TTL for low volatility
	PriceBucketPct           float64       // Price bucket size (default: 0.1%)
	VolumeTolerancePct       float64       // Volume tolerance (default: 20%)
	OrderBookTolerancePct    float64       // Orderbook tolerance (default: 10%)
	InvalidationPriceMovePct float64       // Invalidate on price move (default: 0.5%)
}

// DefaultCacheConfig returns default configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:                  true,
		TTL:                      3 * time.Minute,
		TTLVolatile:              1 * time.Minute,
		TTLNormal:                3 * time.Minute,
		TTLQuiet:                 5 * time.Minute,
		PriceBucketPct:           0.001, // 0.1%
		VolumeTolerancePct:       0.20,  // 20%
		OrderBookTolerancePct:    0.10,  // 10%
		InvalidationPriceMovePct: 0.005, // 0.5%
	}
}

// CachedAnalysis represents a cached analysis result
type CachedAnalysis struct {
	Tool      string                 `json:"tool"`
	Symbol    string                 `json:"symbol"`
	Exchange  string                 `json:"exchange"`
	Timeframe string                 `json:"timeframe"`
	Result    string                 `json:"result"`
	Price     float64                `json:"price"`
	Volume    float64                `json:"volume"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AnalysisCache provides caching for analysis tool results
type AnalysisCache struct {
	config      CacheConfig
	redisClient *redis.Client
	log         *logger.Logger

	// Metrics
	hits      int64
	misses    int64
	sets      int64
	evictions int64
}

// NewAnalysisCache creates a new analysis cache
func NewAnalysisCache(config CacheConfig, redisClient *redis.Client) *AnalysisCache {
	return &AnalysisCache{
		config:      config,
		redisClient: redisClient,
		log:         logger.Get().With("component", "analysis_cache"),
	}
}

// Get retrieves a cached analysis result
func (ac *AnalysisCache) Get(
	ctx context.Context,
	tool, exchange, symbol, timeframe string,
	currentPrice, currentVolume float64,
) (*CachedAnalysis, error) {
	if !ac.config.Enabled {
		return nil, nil
	}

	key := ac.buildCacheKey(tool, exchange, symbol, timeframe, currentPrice)

	var cached CachedAnalysis
	err := ac.redisClient.Get(ctx, key, &cached)
	if err != nil {
		if err.Error() == "redis: nil" {
			ac.misses++
			return nil, nil // Cache miss
		}
		return nil, errors.Wrap(err, "failed to get from cache")
	}

	// Validate cached entry is still relevant
	if !ac.isEntryValid(&cached, currentPrice, currentVolume) {
		// Invalidate stale entry
		_ = ac.redisClient.Delete(ctx, key)
		ac.evictions++
		ac.misses++
		return nil, nil
	}

	ac.hits++
	ac.log.Debug("Cache hit",
		"tool", tool,
		"symbol", symbol,
		"age", time.Since(cached.Timestamp),
	)

	return &cached, nil
}

// Set stores an analysis result in cache
func (ac *AnalysisCache) Set(
	ctx context.Context,
	tool, exchange, symbol, timeframe string,
	result string,
	price, volume float64,
	metadata map[string]interface{},
) error {
	if !ac.config.Enabled {
		return nil
	}

	cached := CachedAnalysis{
		Tool:      tool,
		Symbol:    symbol,
		Exchange:  exchange,
		Timeframe: timeframe,
		Result:    result,
		Price:     price,
		Volume:    volume,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	key := ac.buildCacheKey(tool, exchange, symbol, timeframe, price)

	// Determine TTL based on volatility (if provided in metadata)
	ttl := ac.config.TTL
	if metadata != nil {
		if volatility, ok := metadata["volatility"]; ok {
			if vol, ok := volatility.(float64); ok {
				ttl = ac.getTTLForVolatility(vol)
			}
		}
	}

	err := ac.redisClient.Set(ctx, key, cached, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to set cache")
	}

	ac.sets++
	ac.log.Debug("Cache set",
		"tool", tool,
		"symbol", symbol,
		"ttl", ttl,
	)

	return nil
}

// Invalidate removes a cached entry
func (ac *AnalysisCache) Invalidate(
	ctx context.Context,
	tool, exchange, symbol, timeframe string,
	price float64,
) error {
	if !ac.config.Enabled {
		return nil
	}

	key := ac.buildCacheKey(tool, exchange, symbol, timeframe, price)
	err := ac.redisClient.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "failed to invalidate cache")
	}

	ac.evictions++
	return nil
}

// InvalidateSymbol removes all cached entries for a symbol
func (ac *AnalysisCache) InvalidateSymbol(ctx context.Context, exchange, symbol string) error {
	if !ac.config.Enabled {
		return nil
	}

	// Pattern: analysis:{exchange}:{symbol}:*
	pattern := fmt.Sprintf("analysis:%s:%s:*", exchange, symbol)

	// Note: This requires SCAN command which is safe for production
	// Implementation would use redis SCAN to avoid blocking
	// For now, we'll log and let entries expire naturally
	ac.log.Info("Symbol invalidation requested (entries will expire naturally)",
		"exchange", exchange,
		"symbol", symbol,
		"pattern", pattern,
	)

	return nil
}

// buildCacheKey generates a cache key with price bucketing
func (ac *AnalysisCache) buildCacheKey(
	tool, exchange, symbol, timeframe string,
	price float64,
) string {
	// Round price to bucket
	priceBucket := ac.roundToBucket(price)

	// Create deterministic key
	keyData := fmt.Sprintf("%s:%s:%s:%s:%.8f", tool, exchange, symbol, timeframe, priceBucket)

	// Hash for consistent key length
	hash := sha256.Sum256([]byte(keyData))
	hashStr := fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes

	return fmt.Sprintf("analysis:%s:%s:%s:%s", exchange, symbol, tool, hashStr)
}

// roundToBucket rounds price to the nearest bucket
func (ac *AnalysisCache) roundToBucket(price float64) float64 {
	if price == 0 {
		return 0
	}

	bucketSize := price * ac.config.PriceBucketPct
	return math.Round(price/bucketSize) * bucketSize
}

// isEntryValid checks if a cached entry is still valid
func (ac *AnalysisCache) isEntryValid(cached *CachedAnalysis, currentPrice, currentVolume float64) bool {
	// Check age
	age := time.Since(cached.Timestamp)
	if age > ac.config.TTL*2 {
		// Double TTL safety check
		return false
	}

	// Check price deviation
	if cached.Price > 0 && currentPrice > 0 {
		priceChange := math.Abs(currentPrice-cached.Price) / cached.Price
		if priceChange > ac.config.InvalidationPriceMovePct {
			ac.log.Debug("Cache entry invalidated by price move",
				"symbol", cached.Symbol,
				"cached_price", cached.Price,
				"current_price", currentPrice,
				"change_pct", priceChange*100,
			)
			return false
		}
	}

	// Check volume deviation (if significant)
	if cached.Volume > 0 && currentVolume > 0 {
		volumeChange := math.Abs(currentVolume-cached.Volume) / cached.Volume
		if volumeChange > ac.config.VolumeTolerancePct*2 {
			// Only invalidate on very large volume changes (2x tolerance)
			ac.log.Debug("Cache entry invalidated by volume spike",
				"symbol", cached.Symbol,
				"cached_volume", cached.Volume,
				"current_volume", currentVolume,
				"change_pct", volumeChange*100,
			)
			return false
		}
	}

	return true
}

// getTTLForVolatility returns appropriate TTL based on volatility
func (ac *AnalysisCache) getTTLForVolatility(volatility float64) time.Duration {
	switch {
	case volatility > 0.03: // >3% ATR
		return ac.config.TTLVolatile
	case volatility < 0.01: // <1% ATR
		return ac.config.TTLQuiet
	default:
		return ac.config.TTLNormal
	}
}

// GetMetrics returns cache metrics for monitoring
func (ac *AnalysisCache) GetMetrics() map[string]interface{} {
	total := ac.hits + ac.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(ac.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"enabled":   ac.config.Enabled,
		"hits":      ac.hits,
		"misses":    ac.misses,
		"sets":      ac.sets,
		"evictions": ac.evictions,
		"hit_rate":  hitRate,
		"total":     total,
		"ttl":       ac.config.TTL.String(),
		"ttl_config": map[string]string{
			"volatile": ac.config.TTLVolatile.String(),
			"normal":   ac.config.TTLNormal.String(),
			"quiet":    ac.config.TTLQuiet.String(),
		},
	}
}

// ResetMetrics resets cache metrics
func (ac *AnalysisCache) ResetMetrics() {
	ac.hits = 0
	ac.misses = 0
	ac.sets = 0
	ac.evictions = 0
}
