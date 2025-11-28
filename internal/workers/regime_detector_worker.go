package workers

import (
	"context"
	"time"

	"google.golang.org/adk/agent"

	"prometheus/pkg/logger"
)

// RegimeDetectorWorker periodically detects market regime and adjusts system parameters.
type RegimeDetectorWorker struct {
	agent    agent.Agent // RegimeDetector agent
	interval time.Duration
	log      *logger.Logger
}

// NewRegimeDetectorWorker creates a new regime detection worker.
func NewRegimeDetectorWorker(
	regimeDetectorAgent agent.Agent,
	interval time.Duration,
) *RegimeDetectorWorker {
	return &RegimeDetectorWorker{
		agent:    regimeDetectorAgent,
		interval: interval,
		log:      logger.Get().With("component", "regime_detector_worker"),
	}
}

// Run starts the regime detection loop.
func (w *RegimeDetectorWorker) Run(ctx context.Context) error {
	w.log.Info("Regime detector worker started", "interval", w.interval)

	// Run immediately on startup
	if err := w.detectRegime(ctx); err != nil {
		w.log.Error("Initial regime detection failed", "error", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.detectRegime(ctx); err != nil {
				w.log.Error("Regime detection failed", "error", err)
				// Continue running despite errors
			}

		case <-ctx.Done():
			w.log.Info("Regime detector worker stopped")
			return ctx.Err()
		}
	}
}

// detectRegime calls RegimeDetector agent to classify market and adjust parameters.
func (w *RegimeDetectorWorker) detectRegime(ctx context.Context) error {
	w.log.Info("Running regime detection")

	// TODO: Call agent with market data
	// For now, this is a placeholder - actual agent invocation
	// will be wired in cmd/main.go when integrating with ADK session system

	// Agent should:
	// 1. Call get_technical_analysis for BTC (reference asset)
	// 2. Analyze trend, volatility, volume patterns
	// 3. Classify regime (bull_trending / bull_euphoric / bear / range / high_vol)
	// 4. Save regime to memory
	// 5. Update system parameters (weights, multipliers, thresholds)

	w.log.Info("Regime detection completed")
	return nil
}
