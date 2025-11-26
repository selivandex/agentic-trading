package orderflow

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// OICollector collects open interest data from futures exchanges
// Runs every 1 minute to track open interest changes
type OICollector struct {
	*workers.BaseWorker
	mdRepo      market_data.Repository
	exchFactory exchanges.CentralFactory
	symbols     []string
	exchanges   []string
}

// NewOICollector creates a new open interest collector
func NewOICollector(
	mdRepo market_data.Repository,
	exchFactory exchanges.CentralFactory,
	symbols []string,
	exchanges []string,
	interval time.Duration,
	enabled bool,
) *OICollector {
	return &OICollector{
		BaseWorker:  workers.NewBaseWorker("oi_collector", interval, enabled),
		mdRepo:      mdRepo,
		exchFactory: exchFactory,
		symbols:     symbols,
		exchanges:   exchanges,
	}
}

// Run executes one iteration of OI collection
func (oic *OICollector) Run(ctx context.Context) error {
	oic.Log().Debug("OI collector: starting iteration")

	totalOI := 0
	errorCount := 0

	// Collect OI from each exchange
	for _, exchangeName := range oic.exchanges {
		exchangeClient, err := oic.exchFactory.GetClient(exchangeName)
		if err != nil {
			oic.Log().Error("Failed to get exchange client",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		// Check if exchange supports futures
		futuresExchange, ok := exchangeClient.(exchanges.FuturesExchange)
		if !ok {
			oic.Log().Debug("Exchange does not support futures, skipping",
				"exchange", exchangeName,
			)
			continue
		}

		// Collect OI for all symbols
		oiCount, err := oic.collectExchangeOI(ctx, futuresExchange, exchangeName)
		if err != nil {
			oic.Log().Error("Failed to collect exchange OI",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		totalOI += oiCount
	}

	oic.Log().Info("OI collection complete",
		"total_oi_snapshots", totalOI,
		"errors", errorCount,
	)

	return nil
}

// collectExchangeOI collects open interest for all symbols from an exchange
func (oic *OICollector) collectExchangeOI(
	ctx context.Context,
	exchange exchanges.FuturesExchange,
	exchangeName string,
) (int, error) {
	successCount := 0

	for _, symbol := range oic.symbols {
		oi, err := oic.collectOI(ctx, exchange, exchangeName, symbol)
		if err != nil {
			oic.Log().Error("Failed to collect OI",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		// Store OI snapshot
		if err := oic.mdRepo.InsertOpenInterest(ctx, oi); err != nil {
			oic.Log().Error("Failed to insert OI",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		successCount++
	}

	oic.Log().Debug("Exchange OI collected",
		"exchange", exchangeName,
		"oi_count", successCount,
	)

	return successCount, nil
}

// collectOI collects open interest for a specific symbol
func (oic *OICollector) collectOI(
	ctx context.Context,
	exchange exchanges.FuturesExchange,
	exchangeName, symbol string,
) (*market_data.OpenInterest, error) {
	// Get open interest from exchange
	exchangeOI, err := exchange.GetOpenInterest(ctx, symbol)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get open interest for %s", symbol)
	}

	// Convert to domain OpenInterest
	oi := &market_data.OpenInterest{
		Exchange:  exchangeName,
		Symbol:    symbol,
		Timestamp: time.Now(),
		Amount:    exchangeOI.Amount.InexactFloat64(),
	}

	return oi, nil
}

// AnalyzeOITrend analyzes open interest trend for trading signals
// Rising OI + rising price = bullish (new longs)
// Rising OI + falling price = bearish (new shorts)
// Falling OI = position closing
func (oic *OICollector) AnalyzeOITrend(current, previous *market_data.OpenInterest) (trend string, signal string) {
	if previous == nil {
		return "unknown", "neutral"
	}

	oiChange := ((current.Amount - previous.Amount) / previous.Amount) * 100

	switch {
	case oiChange > 5:
		trend = "rising_fast"
		signal = "strong_interest"
	case oiChange > 1:
		trend = "rising"
		signal = "increasing_interest"
	case oiChange < -5:
		trend = "falling_fast"
		signal = "closing_positions"
	case oiChange < -1:
		trend = "falling"
		signal = "decreasing_interest"
	default:
		trend = "stable"
		signal = "neutral"
	}

	return trend, signal
}

