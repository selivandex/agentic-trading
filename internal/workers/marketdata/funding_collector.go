package marketdata

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// FundingCollector collects funding rates from perpetual futures exchanges
// This worker runs every 1 minute to track funding rate changes
type FundingCollector struct {
	*workers.BaseWorker
	mdRepo      market_data.Repository
	exchFactory exchanges.CentralFactory
	symbols     []string
	exchanges   []string // List of exchange names to collect from
}

// NewFundingCollector creates a new funding rate collector worker
func NewFundingCollector(
	mdRepo market_data.Repository,
	exchFactory exchanges.CentralFactory,
	symbols []string,
	exchanges []string,
	interval time.Duration,
	enabled bool,
) *FundingCollector {
	return &FundingCollector{
		BaseWorker:  workers.NewBaseWorker("funding_collector", interval, enabled),
		mdRepo:      mdRepo,
		exchFactory: exchFactory,
		symbols:     symbols,
		exchanges:   exchanges,
	}
}

// Run executes one iteration of funding rate collection
func (fc *FundingCollector) Run(ctx context.Context) error {
	fc.Log().Debug("Funding collector: starting iteration")

	totalRates := 0
	errorCount := 0

	// Collect funding rates from each exchange
	for _, exchangeName := range fc.exchanges {
		exchangeClient, err := fc.exchFactory.GetClient(exchangeName)
		if err != nil {
			fc.Log().Error("Failed to get exchange client",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		// Check if exchange supports futures
		futuresExchange, ok := exchangeClient.(exchanges.FuturesExchange)
		if !ok {
			fc.Log().Debug("Exchange does not support futures, skipping",
				"exchange", exchangeName,
			)
			continue
		}

		// Collect funding rates for all symbols on this exchange
		ratesCount, err := fc.collectExchangeFundingRates(ctx, futuresExchange, exchangeName)
		if err != nil {
			fc.Log().Error("Failed to collect exchange funding rates",
				"exchange", exchangeName,
				"error", err,
			)
			errorCount++
			continue
		}

		totalRates += ratesCount
	}

	fc.Log().Info("Funding rate collection complete",
		"total_rates", totalRates,
		"errors", errorCount,
	)

	return nil
}

// collectExchangeFundingRates collects funding rates for all symbols from a specific exchange
func (fc *FundingCollector) collectExchangeFundingRates(
	ctx context.Context,
	exchange exchanges.FuturesExchange,
	exchangeName string,
) (int, error) {
	successCount := 0

	// Collect funding rate for each symbol
	for _, symbol := range fc.symbols {
		fundingData, err := fc.collectFundingRate(ctx, exchange, exchangeName, symbol)
		if err != nil {
			fc.Log().Error("Failed to collect funding rate",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			// Continue with other symbols
			continue
		}

		// Insert funding rate to ClickHouse
		if err := fc.mdRepo.InsertFundingRate(ctx, fundingData); err != nil {
			fc.Log().Error("Failed to insert funding rate",
				"exchange", exchangeName,
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		successCount++
	}

	fc.Log().Debug("Exchange funding rates collected",
		"exchange", exchangeName,
		"rate_count", successCount,
	)

	return successCount, nil
}

// collectFundingRate collects funding rate for a specific symbol
func (fc *FundingCollector) collectFundingRate(
	ctx context.Context,
	exchange exchanges.FuturesExchange,
	exchangeName, symbol string,
) (*market_data.FundingRate, error) {
	// Get funding rate from exchange
	fundingRate, err := exchange.GetFundingRate(ctx, symbol)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get funding rate for %s", symbol)
	}

	// Convert to domain FundingRate
	fundingData := &market_data.FundingRate{
		Exchange:        exchangeName,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		FundingRate:     fundingRate.Rate.InexactFloat64(),
		NextFundingTime: fundingRate.NextTime,
		MarkPrice:       0, // Would need separate API call for mark price
		IndexPrice:      0, // Would need separate API call for index price
	}

	return fundingData, nil
}

// AnalyzeFundingRateTrend analyzes funding rate trends for sentiment analysis
// Positive funding rate = longs pay shorts (bullish sentiment)
// Negative funding rate = shorts pay longs (bearish sentiment)
func (fc *FundingCollector) AnalyzeFundingRateTrend(rates []*market_data.FundingRate) (avgRate float64, sentiment string) {
	if len(rates) == 0 {
		return 0, "neutral"
	}

	sum := 0.0
	for _, rate := range rates {
		sum += rate.FundingRate
	}

	avgRate = sum / float64(len(rates))

	// Classify sentiment based on average funding rate
	switch {
	case avgRate > 0.01: // More than 1% (very bullish - longs paying high premium)
		sentiment = "very_bullish"
	case avgRate > 0.001: // More than 0.1% (bullish)
		sentiment = "bullish"
	case avgRate < -0.01: // Less than -1% (very bearish - shorts paying high premium)
		sentiment = "very_bearish"
	case avgRate < -0.001: // Less than -0.1% (bearish)
		sentiment = "bearish"
	default:
		sentiment = "neutral"
	}

	return avgRate, sentiment
}

// DetectFundingRateAnomaly detects abnormal funding rates that might indicate extreme market conditions
// Returns true if funding rate is outside normal range (typically -0.05% to 0.05% per 8h)
func (fc *FundingCollector) DetectFundingRateAnomaly(rate float64) (isAnomalous bool, severity string) {
	absRate := rate
	if absRate < 0 {
		absRate = -absRate
	}

	// Funding rates are typically expressed per 8 hours
	// Normal range: -0.05% to 0.05%
	// Warning: >0.1%
	// Critical: >0.3%
	switch {
	case absRate > 0.003: // More than 0.3%
		return true, "critical"
	case absRate > 0.001: // More than 0.1%
		return true, "warning"
	default:
		return false, "normal"
	}
}
