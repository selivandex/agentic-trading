package orderflow

import (
	"context"
	"time"

	"prometheus/internal/adapters/exchanges/websocket"
	"prometheus/internal/domain/liquidation"
	"prometheus/internal/workers"
)

// LiquidationCollector collects real-time liquidation events
// Runs every 5 seconds or uses WebSocket for real-time updates
type LiquidationCollector struct {
	*workers.BaseWorker
	liqRepo  liquidation.Repository
	wsClient websocket.WebSocketClient
	symbols  []string
	useWS    bool // Use WebSocket vs polling
}

// NewLiquidationCollector creates a new liquidation collector
func NewLiquidationCollector(
	liqRepo liquidation.Repository,
	wsClient websocket.WebSocketClient,
	symbols []string,
	useWS bool,
	interval time.Duration,
	enabled bool,
) *LiquidationCollector {
	return &LiquidationCollector{
		BaseWorker: workers.NewBaseWorker("liquidation_collector", interval, enabled),
		liqRepo:    liqRepo,
		wsClient:   wsClient,
		symbols:    symbols,
		useWS:      useWS,
	}
}

// Run executes one iteration of liquidation collection
func (lc *LiquidationCollector) Run(ctx context.Context) error {
	if lc.useWS && lc.wsClient != nil {
		// WebSocket mode - subscribe once and let callbacks handle updates
		return lc.runWebSocketMode(ctx)
	}

	// Polling mode - fetch recent liquidations via HTTP API
	return lc.runPollingMode(ctx)
}

// runWebSocketMode subscribes to liquidation stream via WebSocket
func (lc *LiquidationCollector) runWebSocketMode(ctx context.Context) error {
	lc.Log().Debug("Liquidation collector: WebSocket mode")

	// Subscribe to liquidations with callback
	err := lc.wsClient.SubscribeLiquidations(func(liq *websocket.Liquidation) {
		// Convert to domain liquidation
		domainLiq := &liquidation.Liquidation{
			Exchange:  liq.Exchange,
			Symbol:    liq.Symbol,
			Timestamp: liq.Timestamp,
			Side:      liq.Side,
			Price:     liq.Price,
			Quantity:  liq.Quantity,
			Value:     liq.Value,
		}

		// Store in ClickHouse
		if err := lc.liqRepo.InsertLiquidation(ctx, domainLiq); err != nil {
			lc.Log().Error("Failed to insert liquidation",
				"exchange", liq.Exchange,
				"symbol", liq.Symbol,
				"error", err,
			)
			return
		}

		lc.Log().Debugf("Liquidation recorded: %s %s $%.0f", liq.Symbol, liq.Side, liq.Value)

		// Publish large liquidations as alerts
		if liq.Value >= 100000 { // $100k+
			lc.publishLiquidationAlert(ctx, domainLiq)
		}
	})

	if err != nil {
		lc.Log().Errorf("Failed to subscribe to liquidations: %v", err)
		return err
	}

	return nil
}

// runPollingMode polls liquidation data from external APIs
func (lc *LiquidationCollector) runPollingMode(ctx context.Context) error {
	lc.Log().Debug("Liquidation collector: polling mode")

	// TODO: Implement polling from Coinglass/Hyblock APIs
	// For now, this is a placeholder

	lc.Log().Debug("Polling mode not yet implemented, use WebSocket mode")
	return nil
}

// publishLiquidationAlert publishes large liquidation event to Kafka
func (lc *LiquidationCollector) publishLiquidationAlert(ctx context.Context, liq *liquidation.Liquidation) {
	// TODO: Publish to Kafka topic "market.liquidations"
	lc.Log().Infof("Large liquidation alert: %s %s $%.0f",
		liq.Symbol, liq.Side, liq.Value)
}
