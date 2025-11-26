package orderflow

import (
	"context"
	"fmt"
	"time"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/events"
	"prometheus/internal/workers"
	"prometheus/pkg/logger"
)

const (
	// WhaleThresholdUSD defines minimum trade value to be considered whale trade
	WhaleThresholdUSD = 100000.0 // $100k
)

// WhaleAlertCollector detects and tracks large trades (whale activity)
// Runs every 10 seconds to analyze recent trades
type WhaleAlertCollector struct {
	*workers.BaseWorker
	mdRepo           market_data.Repository
	symbols          []string
	exchanges        []string
	thresholdUSD     float64
	kafka            *kafka.Producer
	eventPublisher   *events.WorkerPublisher
	whaleTradesCache map[string]bool // Prevents duplicate alerts
	log              *logger.Logger
}

// NewWhaleAlertCollector creates a new whale alert collector
func NewWhaleAlertCollector(
	mdRepo market_data.Repository,
	symbols []string,
	exchanges []string,
	thresholdUSD float64,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *WhaleAlertCollector {
	if thresholdUSD <= 0 {
		thresholdUSD = WhaleThresholdUSD
	}

	return &WhaleAlertCollector{
		BaseWorker:       workers.NewBaseWorker("whale_alert_collector", interval, enabled),
		mdRepo:           mdRepo,
		symbols:          symbols,
		exchanges:        exchanges,
		thresholdUSD:     thresholdUSD,
		kafka:            kafka,
		eventPublisher:   events.NewWorkerPublisher(kafka),
		whaleTradesCache: make(map[string]bool),
		log:              logger.Get().With("component", "whale_alert_collector"),
	}
}

// Run executes one iteration of whale alert detection
func (wac *WhaleAlertCollector) Run(ctx context.Context) error {
	wac.Log().Debug("Whale alert collector: starting iteration")

	totalWhales := 0
	errorCount := 0

	// Analyze trades from each exchange
	for _, exchangeName := range wac.exchanges {
		for _, symbol := range wac.symbols {
			whaleCount, err := wac.analyzeSymbol(ctx, exchangeName, symbol)
			if err != nil {
				wac.Log().Error("Failed to analyze symbol",
					"exchange", exchangeName,
					"symbol", symbol,
					"error", err,
				)
				errorCount++
				continue
			}

			totalWhales += whaleCount
		}
	}

	wac.Log().Info("Whale alert collection complete",
		"total_whales", totalWhales,
		"errors", errorCount,
	)

	// Clean cache periodically (keep only last hour)
	wac.cleanCache()

	return nil
}

// analyzeSymbol analyzes recent trades for a symbol and detects whales
func (wac *WhaleAlertCollector) analyzeSymbol(ctx context.Context, exchange, symbol string) (int, error) {
	// Fetch recent trades (last 100)
	trades, err := wac.mdRepo.GetRecentTrades(ctx, exchange, symbol, 100)
	if err != nil {
		return 0, err
	}

	if len(trades) == 0 {
		return 0, nil
	}

	whaleCount := 0

	for _, trade := range trades {
		// Calculate trade value in USD
		tradeValueUSD := trade.Price * trade.Quantity

		// Check if it's a whale trade
		if tradeValueUSD >= wac.thresholdUSD {
			// Check if we've already alerted on this trade
			cacheKey := fmt.Sprintf("%s:%s:%s", exchange, symbol, trade.TradeID)
			if wac.whaleTradesCache[cacheKey] {
				continue
			}

			// Mark as processed
			wac.whaleTradesCache[cacheKey] = true

			// Determine trade direction and sentiment
			direction := "BUY"
			sentiment := "bullish"
			if trade.Side == "sell" {
				direction = "SELL"
				sentiment = "bearish"
			}

			wac.Log().Infof("🐋 WHALE ALERT: %s %s $%.0f %s at $%.2f",
				symbol, direction, tradeValueUSD, exchange, trade.Price)

			// Publish alert to Kafka
			if err := wac.publishWhaleAlert(ctx, exchange, symbol, trade, tradeValueUSD, sentiment); err != nil {
				wac.Log().Error("Failed to publish whale alert", "error", err)
			}

			whaleCount++
		}
	}

	return whaleCount, nil
}

// publishWhaleAlert publishes whale trade alert to Kafka using protobuf
func (wac *WhaleAlertCollector) publishWhaleAlert(
	ctx context.Context,
	exchange, symbol string,
	trade market_data.Trade,
	valueUSD float64,
	sentiment string,
) error {
	if wac.eventPublisher == nil {
		return nil // Kafka not configured
	}

	// Publish using protobuf event
	return wac.eventPublisher.PublishWhaleAlert(
		ctx,
		exchange,
		symbol,
		trade.TradeID,
		trade.Side,
		sentiment,
		trade.Price,
		trade.Quantity,
		valueUSD,
	)
}

// cleanCache removes old entries from cache (keep only last hour)
func (wac *WhaleAlertCollector) cleanCache() {
	// Simple cache cleanup - in production, use Redis with TTL
	// For now, just clear cache if it gets too large
	if len(wac.whaleTradesCache) > 10000 {
		wac.whaleTradesCache = make(map[string]bool)
		wac.Log().Debug("Whale trades cache cleared")
	}
}

// CalculateWhaleImpact analyzes whale trade impact on orderbook
// Large buy = potential support, large sell = potential resistance
func (wac *WhaleAlertCollector) CalculateWhaleImpact(
	trade market_data.Trade,
	orderBook *market_data.OrderBookSnapshot,
) (impact string, priceImpact float64) {
	// TODO: Calculate actual orderbook impact
	// For now, simple heuristic based on trade size

	tradeValue := trade.Price * trade.Quantity

	switch {
	case tradeValue > 1000000: // $1M+
		impact = "extreme"
		priceImpact = 0.5 // Likely to move price 0.5%+
	case tradeValue > 500000: // $500k+
		impact = "high"
		priceImpact = 0.2
	case tradeValue > 200000: // $200k+
		impact = "medium"
		priceImpact = 0.1
	default:
		impact = "low"
		priceImpact = 0.05
	}

	return impact, priceImpact
}

