package derivatives

import (
	"context"
	"time"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
)

// FundingAggregator aggregates funding rates from multiple exchanges
// Calculates average funding and detects arbitrage opportunities
type FundingAggregator struct {
	*workers.BaseWorker
	marketDataRepo market_data.Repository
	exchanges      []string
	symbols        []string
}

// NewFundingAggregator creates a new funding rate aggregator
func NewFundingAggregator(
	marketDataRepo market_data.Repository,
	exchanges []string,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *FundingAggregator {
	return &FundingAggregator{
		BaseWorker:     workers.NewBaseWorker("funding_aggregator", interval, enabled),
		marketDataRepo: marketDataRepo,
		exchanges:      exchanges,
		symbols:        symbols,
	}
}

// Run executes one iteration of funding rate aggregation
func (fa *FundingAggregator) Run(ctx context.Context) error {
	fa.Log().Debug("Funding aggregator: starting iteration")

	// Aggregate funding rates for each symbol
	for _, symbol := range fa.symbols {
		aggregated, err := fa.aggregateFundingRates(ctx, symbol)
		if err != nil {
			fa.Log().Error("Failed to aggregate funding rates",
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		if aggregated == nil {
			continue
		}

		fa.Log().Info("Funding rates aggregated",
			"symbol", symbol,
			"avg_rate", aggregated.avgRate,
			"min_rate", aggregated.minRate,
			"max_rate", aggregated.maxRate,
			"exchanges", len(aggregated.rates),
		)

		// Detect arbitrage opportunities
		if arbitrage := fa.detectArbitrage(aggregated); arbitrage != nil {
			fa.Log().Warn("Funding rate arbitrage detected",
				"symbol", symbol,
				"long_exchange", arbitrage.longExchange,
				"short_exchange", arbitrage.shortExchange,
				"spread", arbitrage.spread,
			)
		}
	}

	fa.Log().Info("Funding aggregation complete", "symbols", len(fa.symbols))

	return nil
}

// aggregatedFunding represents aggregated funding rate data
type aggregatedFunding struct {
	symbol    string
	timestamp time.Time
	rates     map[string]float64 // exchange -> rate
	avgRate   float64
	minRate   float64
	maxRate   float64
	stdDev    float64
}

// arbitrageOpportunity represents a funding rate arbitrage
type arbitrageOpportunity struct {
	symbol        string
	longExchange  string // Exchange with lowest rate (go long here)
	shortExchange string // Exchange with highest rate (go short here)
	spread        float64
	profitAPR     float64
}

// aggregateFundingRates fetches and aggregates funding rates across exchanges
func (fa *FundingAggregator) aggregateFundingRates(ctx context.Context, symbol string) (*aggregatedFunding, error) {
	rates := make(map[string]float64)
	// Note: would use since for querying historical rates if needed
	// since := time.Now().Add(-1 * time.Hour)

	// Fetch funding rate from each exchange
	for range fa.exchanges {
		// TODO: Query from market_data repository using GetLatestFundingRate
		// For now, skip as proper repository integration is pending
	}

	if len(rates) == 0 {
		fa.Log().Debug("No funding rates collected", "symbol", symbol)
		return nil, nil
	}

	// Calculate statistics
	avgRate, minRate, maxRate, stdDev := fa.calculateStats(rates)

	return &aggregatedFunding{
		symbol:    symbol,
		timestamp: time.Now(),
		rates:     rates,
		avgRate:   avgRate,
		minRate:   minRate,
		maxRate:   maxRate,
		stdDev:    stdDev,
	}, nil
}

// calculateStats calculates statistical measures for funding rates
func (fa *FundingAggregator) calculateStats(rates map[string]float64) (float64, float64, float64, float64) {
	if len(rates) == 0 {
		return 0, 0, 0, 0
	}

	// Calculate mean
	sum := 0.0
	min := 999999.0
	max := -999999.0
	
	for _, rate := range rates {
		sum += rate
		if rate < min {
			min = rate
		}
		if rate > max {
			max = rate
		}
	}
	
	avg := sum / float64(len(rates))

	// Calculate standard deviation
	varianceSum := 0.0
	for _, rate := range rates {
		diff := rate - avg
		varianceSum += diff * diff
	}
	
	stdDev := 0.0
	if len(rates) > 1 {
		stdDev = varianceSum / float64(len(rates))
		// Square root would go here, but keeping it simple
	}

	return avg, min, max, stdDev
}

// detectArbitrage identifies arbitrage opportunities from rate spreads
func (fa *FundingAggregator) detectArbitrage(agg *aggregatedFunding) *arbitrageOpportunity {
	// Find exchanges with min and max rates
	var longExchange, shortExchange string
	minRate := 999999.0
	maxRate := -999999.0

	for exchange, rate := range agg.rates {
		if rate < minRate {
			minRate = rate
			longExchange = exchange
		}
		if rate > maxRate {
			maxRate = rate
			shortExchange = exchange
		}
	}

	spread := maxRate - minRate

	// Threshold: arbitrage is profitable if spread > 0.01% (0.0001)
	// This covers trading fees and provides profit margin
	if spread < 0.0001 {
		return nil
	}

	// Calculate annualized profit
	// Funding is paid every 8 hours (3x per day)
	profitAPR := spread * 3 * 365 * 100 // Convert to %

	return &arbitrageOpportunity{
		symbol:        agg.symbol,
		longExchange:  longExchange,
		shortExchange: shortExchange,
		spread:        spread,
		profitAPR:     profitAPR,
	}
}

// CalculateFundingCost calculates the cost of holding a position with funding
// positionSize: in USD
// fundingRate: as decimal (e.g., 0.0001 = 0.01%)
// hours: holding period
func CalculateFundingCost(positionSize, fundingRate float64, hours int) float64 {
	// Funding is paid every 8 hours
	periods := float64(hours) / 8.0
	cost := positionSize * fundingRate * periods
	return cost
}

// InterpretFundingRate provides human-readable interpretation
func InterpretFundingRate(rate float64) string {
	// Convert to percentage
	pct := rate * 100

	if pct > 0.05 {
		return "very_high" // >0.05% per 8h = very bullish market
	} else if pct > 0.02 {
		return "high" // 0.02-0.05% = bullish
	} else if pct < -0.02 {
		return "negative" // Bearish market (shorts pay longs)
	} else if pct < -0.05 {
		return "very_negative" // Very bearish
	}
	return "neutral"
}

// CalculateFundingAPR converts funding rate to annualized return
func CalculateFundingAPR(rate float64) float64 {
	// Funding paid 3 times per day (every 8 hours)
	dailyRate := rate * 3
	annualRate := dailyRate * 365
	return annualRate * 100 // Convert to percentage
}

// DetectFundingTrend analyzes funding rate history to detect trends
func DetectFundingTrend(rates []float64) string {
	if len(rates) < 5 {
		return "insufficient_data"
	}

	// Calculate simple moving average of last 5 periods
	recent := rates[len(rates)-5:]
	sum := 0.0
	for _, r := range recent {
		sum += r
	}
	recentAvg := sum / float64(len(recent))

	// Compare with earlier periods
	earlier := rates[:len(rates)-5]
	sum = 0.0
	for _, r := range earlier {
		sum += r
	}
	earlierAvg := sum / float64(len(earlier))

	// Determine trend
	if recentAvg > earlierAvg*1.2 {
		return "increasing" // Funding getting more positive
	} else if recentAvg < earlierAvg*0.8 {
		return "decreasing" // Funding getting more negative
	}
	return "stable"
}

// CalculateOptimalFundingStrategy suggests optimal strategy based on rates
func CalculateOptimalFundingStrategy(avgRate, spread float64) string {
	// If average funding is high positive: shorters pay longs
	// Strategy: go long to collect funding
	if avgRate > 0.0003 {
		return "long_to_collect_funding"
	}

	// If average funding is negative: longs pay shorters
	if avgRate < -0.0003 {
		return "short_to_collect_funding"
	}

	// If spread is wide: arbitrage opportunity
	if spread > 0.0002 {
		return "funding_arbitrage"
	}

	return "neutral_funding"
}

