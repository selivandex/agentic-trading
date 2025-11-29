package marketdata

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// TickerCollector collects real-time ticker data (prices, volume) from exchanges
// This worker runs frequently (e.g., every 5-10 seconds) to get latest prices
type TickerCollector struct {
	*workers.BaseWorker
	mdRepo      market_data.Repository
	exchFactory exchanges.CentralFactory
	symbols     []string
	exchanges   []string // List of exchange names to collect from
}

// NewTickerCollector creates a new ticker collector worker
func NewTickerCollector(
	mdRepo market_data.Repository,
	exchFactory exchanges.CentralFactory,
	symbols []string,
	exchanges []string,
	interval time.Duration,
	enabled bool,
) *TickerCollector {
	return &TickerCollector{
		BaseWorker:  workers.NewBaseWorker("ticker_collector", interval, enabled),
		mdRepo:      mdRepo,
		exchFactory: exchFactory,
		symbols:     symbols,
		exchanges:   exchanges,
	}
}

// Run executes one iteration of ticker collection
func (tc *TickerCollector) Run(ctx context.Context) error {
	tc.Log().Debug("Ticker collector: starting iteration")

	totalTickers := 0
	errorCount := 0

	// Collect tickers from each exchange
	for _, exchangeName := range tc.exchanges {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			tc.Log().Info("Ticker collection interrupted by shutdown",
				"tickers_collected", totalTickers,
				"exchanges_remaining", len(tc.exchanges)-totalTickers/len(tc.symbols),
			)
			return ctx.Err()
		default:
		}

		exchangeClient, err := tc.exchFactory.GetClient(exchangeName)
		if err != nil {
			tc.Log().Error("Failed to get exchange client",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		// Collect tickers for all symbols on this exchange
		tickerCount, err := tc.collectExchangeTickers(ctx, exchangeClient, exchangeName)
		if err != nil {
			tc.Log().Error("Failed to collect exchange tickers",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		totalTickers += tickerCount
	}

	tc.Log().Info("Ticker collection complete",
		"total_tickers", totalTickers,
		"errors", errorCount,
	)

	return nil
}

// collectExchangeTickers collects tickers for all symbols from a specific exchange
func (tc *TickerCollector) collectExchangeTickers(
	ctx context.Context,
	exchange exchanges.Exchange,
	exchangeName string,
) (int, error) {
	successCount := 0

	// Collect ticker for each symbol
	for _, symbol := range tc.symbols {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			tc.Log().Debug("Ticker collection for exchange interrupted by shutdown",
				"exchange", exchangeName,
				"symbols_collected", successCount,
			)
			return successCount, ctx.Err()
		default:
		}

		ticker, err := tc.collectTicker(ctx, exchange, exchangeName, symbol)
		if err != nil {
			tc.Log().Error("Failed to collect ticker",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			// Continue with other symbols
			continue
		}

		// Insert ticker to ClickHouse
		if err := tc.mdRepo.InsertTicker(ctx, []market_data.Ticker{*ticker}); err != nil {
			tc.Log().Error("Failed to insert ticker",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		successCount++
	}

	tc.Log().Debug("Exchange tickers collected",
		"exchange", exchangeName,
		"ticker_count", successCount,
	)

	return successCount, nil
}

// collectTicker collects a single ticker from an exchange
func (tc *TickerCollector) collectTicker(
	ctx context.Context,
	exchange exchanges.Exchange,
	exchangeName, symbol string,
) (*market_data.Ticker, error) {
	// Get ticker from exchange
	exchangeTicker, err := exchange.GetTicker(ctx, symbol)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ticker for %s", symbol)
	}

	// Convert exchanges.Ticker to market_data.Ticker
	ticker := &market_data.Ticker{
		Exchange:           exchangeName,
		Symbol:             exchangeTicker.Symbol,
		MarketType:         "spot", // Default, would need to be determined from exchange
		Timestamp:          time.Now(),
		LastPrice:          exchangeTicker.LastPrice.InexactFloat64(),
		OpenPrice:          exchangeTicker.LastPrice.InexactFloat64(), // Approximation
		HighPrice:          exchangeTicker.High24h.InexactFloat64(),
		LowPrice:           exchangeTicker.Low24h.InexactFloat64(),
		Volume:             exchangeTicker.VolumeBase.InexactFloat64(),
		QuoteVolume:        exchangeTicker.VolumeQuote.InexactFloat64(),
		PriceChange:        0, // Would need calculation
		PriceChangePercent: exchangeTicker.Change24hPct.InexactFloat64(),
		WeightedAvgPrice:   exchangeTicker.LastPrice.InexactFloat64(), // Approximation
		TradeCount:         0,                                         // Not available in basic ticker
		EventTime:          time.Now(),
	}

	return ticker, nil
}
