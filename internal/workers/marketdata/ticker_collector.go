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
		if err := tc.mdRepo.InsertTicker(ctx, ticker); err != nil {
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
		Exchange:  exchangeName,
		Symbol:    exchangeTicker.Symbol,
		Timestamp: time.Now(),
		Price:     exchangeTicker.LastPrice.InexactFloat64(),
		Bid:       exchangeTicker.BidPrice.InexactFloat64(),
		Ask:       exchangeTicker.AskPrice.InexactFloat64(),
		Volume24h: exchangeTicker.VolumeBase.InexactFloat64(),
		Change24h: exchangeTicker.Change24hPct.InexactFloat64(),
		High24h:   exchangeTicker.High24h.InexactFloat64(),
		Low24h:    exchangeTicker.Low24h.InexactFloat64(),
		// FundingRate and OpenInterest would need additional API calls for futures
		FundingRate:  0,
		OpenInterest: 0,
	}

	return ticker, nil
}
