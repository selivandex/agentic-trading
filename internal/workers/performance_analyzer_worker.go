package workers

import (
	"context"
	"time"

	"google.golang.org/adk/agent"

	"prometheus/pkg/logger"
)

// PerformanceAnalyzerWorker periodically analyzes trade outcomes and extracts learnings.
type PerformanceAnalyzerWorker struct {
	agent    agent.Agent // PerformanceAnalyzer agent
	schedule string      // Cron schedule (e.g., "0 0 * * 0" for Sunday midnight)
	log      *logger.Logger
}

// NewPerformanceAnalyzerWorker creates a new performance analysis worker.
func NewPerformanceAnalyzerWorker(
	performanceAnalyzerAgent agent.Agent,
	schedule string,
) *PerformanceAnalyzerWorker {
	return &PerformanceAnalyzerWorker{
		agent:    performanceAnalyzerAgent,
		schedule: schedule,
		log:      logger.Get().With("component", "performance_analyzer_worker"),
	}
}

// Run starts the performance analysis loop.
// Runs weekly to analyze trade outcomes and extract patterns.
func (w *PerformanceAnalyzerWorker) Run(ctx context.Context) error {
	w.log.Infow("Performance analyzer worker started", "schedule", w.schedule)

	// Run immediately on startup
	if err := w.analyzePerformance(ctx); err != nil {
		w.log.Errorw("Initial performance analysis failed", "error", err)
	}

	// Run weekly (every Sunday at 00:00 UTC)
	ticker := time.NewTicker(7 * 24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.analyzePerformance(ctx); err != nil {
				w.log.Errorw("Performance analysis failed", "error", err)
				// Continue running despite errors
			}

		case <-ctx.Done():
			w.log.Info("Performance analyzer worker stopped")
			return ctx.Err()
		}
	}
}

// analyzePerformance calls PerformanceAnalyzer agent to extract learnings from trade history.
func (w *PerformanceAnalyzerWorker) analyzePerformance(ctx context.Context) error {
	w.log.Info("Running weekly performance analysis")

	// TODO: Call agent with trade history from past week
	// For now, this is a placeholder - actual agent invocation
	// will be wired in cmd/main.go when integrating with ADK session system

	// Agent should:
	// 1. Call get_trade_journal(period: "1w") to fetch all trades
	// 2. Call get_strategy_stats to get aggregated metrics
	// 3. Group trades by patterns (setup type, regime, timeframe)
	// 4. Calculate success rates for each pattern
	// 5. Extract validated lessons (sample_size >=10, clear pattern)
	// 6. Save lessons to collective memory via save_memory
	// 7. Optionally update system parameters (weights, thresholds)

	w.log.Info("Performance analysis completed")
	return nil
}

// RunNow triggers an immediate analysis (for on-demand execution).
func (w *PerformanceAnalyzerWorker) RunNow(ctx context.Context) error {
	w.log.Info("Running on-demand performance analysis")
	return w.analyzePerformance(ctx)
}
