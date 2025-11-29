package marketdata

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// OHLCVCollector collects OHLCV data from exchanges
type OHLCVCollector struct {
	*workers.BaseWorker
	mdRepo        market_data.Repository
	exchFactory   exchanges.Factory
	baseAssets    []string // Base assets like BTC, ETH, SOL
	quoteCurrency string   // Quote currency like USDT
	timeframes    []string
}

// NewOHLCVCollector creates a new OHLCV collector worker
// baseAssets: list of base assets to track (e.g., ["BTC", "ETH", "SOL"])
// quoteCurrency: quote currency for pairs (e.g., "USDT")
// Each exchange will format symbols according to their own requirements
func NewOHLCVCollector(
	mdRepo market_data.Repository,
	exchFactory exchanges.Factory,
	baseAssets []string,
	quoteCurrency string,
	timeframes []string,
	interval time.Duration,
	enabled bool,
) *OHLCVCollector {
	return &OHLCVCollector{
		BaseWorker:    workers.NewBaseWorker("ohlcv_collector", interval, enabled),
		mdRepo:        mdRepo,
		exchFactory:   exchFactory,
		baseAssets:    baseAssets,
		quoteCurrency: quoteCurrency,
		timeframes:    timeframes,
	}
}

// formatSymbol formats base asset and quote currency for the given exchange
// Each exchange has its own symbol format:
// - Binance: BTCUSDT (no slash)
// - Others: BTC/USDT (with slash)
func (oc *OHLCVCollector) formatSymbol(exchangeName, baseAsset string) string {
	// Most exchanges use slash notation (BTC/USDT)
	// Binance normalizeSymbol will handle removing the slash
	return baseAsset + "/" + oc.quoteCurrency
}

// Run executes one iteration of OHLCV collection
func (oc *OHLCVCollector) Run(ctx context.Context) error {
	oc.Log().Debug("OHLCV collector: starting iteration")

	// This is a simplified implementation
	// In production, we need:
	// 1. Central exchange account (or public API access)
	// 2. Rate limiting per exchange
	// 3. Batch inserts to ClickHouse
	// 4. Retry logic with exponential backoff

	oc.Log().Warn("OHLCV collector: simplified implementation - need central exchange account")

	totalCandles := 0

	// For each base asset and timeframe, collect OHLCV data
	for assetIdx, baseAsset := range oc.baseAssets {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			oc.Log().Info("OHLCV collection interrupted by shutdown",
				"candles_collected", totalCandles,
				"assets_processed", assetIdx,
				"assets_remaining", len(oc.baseAssets)-assetIdx,
				"current_asset", baseAsset,
			)
			return ctx.Err()
		default:
		}

		// Format symbol for the exchange
		// Note: Each exchange will normalize the symbol according to its requirements
		// e.g., Binance will convert "BTC/USDT" to "BTCUSDT"
		symbol := oc.formatSymbol("", baseAsset)

		for tfIdx, timeframe := range oc.timeframes {
			// Check for context cancellation within nested loop
			select {
			case <-ctx.Done():
				oc.Log().Info("OHLCV collection interrupted by shutdown",
					"candles_collected", totalCandles,
					"assets_processed", assetIdx,
					"current_asset", baseAsset,
					"current_timeframe", timeframe,
					"timeframes_completed_for_asset", tfIdx,
				)
				return ctx.Err()
			default:
			}

			candles, err := oc.collectOHLCV(ctx, symbol, timeframe)
			if err != nil {
				oc.Log().Error("Failed to collect OHLCV",
					"base_asset", baseAsset,
					"symbol", symbol,
					"timeframe", timeframe,
					"error", err,
				)
				continue
			}
			totalCandles += candles
		}
	}

	oc.Log().Info("OHLCV collection complete", "total_candles", totalCandles)

	return nil
}

// collectOHLCV collects OHLCV data for a specific symbol and timeframe
func (oc *OHLCVCollector) collectOHLCV(ctx context.Context, symbol, timeframe string) (int, error) {
	// In production, we'd:
	// 1. Get exchange client (using central/public API keys)
	// 2. Fetch latest candles from exchange
	// 3. Save to ClickHouse

	oc.Log().Debug("Collecting OHLCV",
		"symbol", symbol,
		"timeframe", timeframe,
	)

	// TODO: Implement actual collection logic
	// For now, return 0 candles collected

	return 0, nil
}

// collectOHLCVFromExchange collects OHLCV from a specific exchange
func (oc *OHLCVCollector) collectOHLCVFromExchange(
	ctx context.Context,
	exchange exchanges.Exchange,
	symbol, timeframe string,
	limit int,
) ([]market_data.OHLCV, error) {
	// Fetch OHLCV data from exchange
	exchangeCandles, err := exchange.GetOHLCV(ctx, symbol, timeframe, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch OHLCV from exchange")
	}

	// Convert exchange OHLCV to domain OHLCV
	exchangeName := exchange.Name()
	candles := make([]market_data.OHLCV, 0, len(exchangeCandles))
	for _, ec := range exchangeCandles {
		candle := market_data.OHLCV{
			Exchange:              exchangeName,
			Symbol:                symbol,
			Timeframe:             timeframe,
			MarketType:            "spot", // TODO: detect market type from exchange
			OpenTime:              ec.OpenTime,
			CloseTime:             ec.CloseTime,
			Open:                  ec.Open.InexactFloat64(),
			High:                  ec.High.InexactFloat64(),
			Low:                   ec.Low.InexactFloat64(),
			Close:                 ec.Close.InexactFloat64(),
			Volume:                ec.Volume.InexactFloat64(),
			QuoteVolume:           ec.QuoteVolume.InexactFloat64(),
			Trades:                ec.Trades,
			TakerBuyBaseVolume:    ec.TakerBuyBaseVolume.InexactFloat64(),
			TakerBuyQuoteVolume:   ec.TakerBuyQuoteVolume.InexactFloat64(),
			IsClosed:              true,      // Historical candles from REST API are always closed
			EventTime:             time.Now(), // Use current time as event_time for historical data
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// saveOHLCVBatch saves a batch of OHLCV candles to ClickHouse
func (oc *OHLCVCollector) saveOHLCVBatch(ctx context.Context, candles []market_data.OHLCV) error {
	if len(candles) == 0 {
		return nil
	}

	// TODO: Implement batch insert to ClickHouse
	// For now, we just log that we would save them
	oc.Log().Debug("Would save OHLCV batch", "count", len(candles))

	return nil
}

// retryWithBackoff implements exponential backoff retry logic
func (oc *OHLCVCollector) retryWithBackoff(ctx context.Context, fn func() error, maxRetries int) error {
	var err error
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		oc.Log().Warn("Retry attempt failed",
			"attempt", i+1,
			"max_retries", maxRetries,
			"backoff", backoff,
			"error", err,
		)

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Exponential backoff with cap at 30 seconds
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}

	return errors.Wrap(err, "max retries exceeded")
}

// RateLimiter implements a simple rate limiter with graceful shutdown support
type RateLimiter struct {
	tokens     chan struct{}
	refillRate time.Duration
	capacity   int
	stop       chan struct{} // Channel to signal shutdown to refill goroutine
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(capacity int, refillRate time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:     make(chan struct{}, capacity),
		refillRate: refillRate,
		capacity:   capacity,
		stop:       make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < capacity; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// Acquire waits for a token to become available
func (rl *RateLimiter) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	}
}

// Close stops the rate limiter and cleans up the refill goroutine
func (rl *RateLimiter) Close() {
	close(rl.stop)
}

// refill refills tokens at the specified rate
// Exits gracefully when Close() is called
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.refillRate)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			// Graceful shutdown requested
			return
		case <-ticker.C:
			// Try to add a token
			select {
			case rl.tokens <- struct{}{}:
				// Token added
			default:
				// Channel full, skip
			}
		}
	}
}
