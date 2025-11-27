package tools

import (
	"prometheus/internal/events"
	"prometheus/internal/tools/indicators"
	"prometheus/internal/tools/market"
	toolmemory "prometheus/internal/tools/memory"
	toolrisk "prometheus/internal/tools/risk"
	"prometheus/internal/tools/shared"
	"prometheus/internal/tools/smc"
	"prometheus/internal/tools/trading"
	"prometheus/pkg/logger"

	"google.golang.org/adk/tool"
)

// RegisterAllTools registers all available tools in the registry
// Note: workerPublisher is passed separately for publish_opportunity tool
func RegisterAllTools(registry *Registry, deps shared.Deps, workerPublisher *events.WorkerPublisher) {
	log := deps.Log.With("component", "tool_registration")
	// Note: All tools now use shared.NewToolBuilder() fluent API with built-in middleware:
	// - .WithRetry(attempts, backoff) - automatic retry with exponential backoff
	// - .WithTimeout(duration) - execution timeout enforcement
	//
	// Stats tracking is handled by ADK callbacks (see internal/agents/callbacks/tool.go)
	// Middleware is configured in each tool's NewXXXTool() constructor via ToolBuilder
	// Example: shared.NewToolBuilder(...).WithTimeout(10*time.Second).WithRetry(3, 500*time.Millisecond).Build()
	// ========================================
	// Market Data & Analysis (ALL-IN-ONE)
	// Replaces: get_price, get_ohlcv, get_trades, get_orderbook, get_order_flow_analysis
	// ========================================
	registry.Register("get_market_analysis", market.NewMarketAnalysisTool(deps))

	// publish_opportunity requires worker publisher for event publishing
	if workerPublisher != nil {
		registry.Register("publish_opportunity", market.NewPublishOpportunityTool(workerPublisher))
	}

	log.Debug("Registered market data tools")
	// ========================================
	// Technical Analysis (ALL-IN-ONE)
	// Replaces: rsi, macd, stochastic, cci, roc, atr, bollinger, keltner,
	//           ema, sma, ema_ribbon, supertrend, ichimoku, pivot_points,
	//           vwap, obv, volume_profile, delta_volume
	// ========================================
	registry.Register("get_technical_analysis", indicators.NewTechnicalAnalysisTool(deps))
	log.Debug("Registered comprehensive technical analysis tool")

	// ========================================
	// Smart Money Concepts (ALL-IN-ONE)
	// Replaces: detect_fvg, detect_order_blocks, get_swing_points,
	//           detect_liquidity_zones, detect_stop_hunt, detect_imbalances,
	//           get_market_structure
	// ========================================
	registry.Register("get_smc_analysis", smc.NewSMCAnalysisTool(deps))
	log.Debug("Registered comprehensive SMC analysis tool")
	// ========================================
	// Trading Tools (user-specific, requires exchange account)
	// ========================================
	// Account Status (ALL-IN-ONE) - replaces get_balance, get_positions, get_portfolio_summary
	registry.Register("get_account_status", trading.NewAccountStatusTool(deps))
	registry.Register("place_order", trading.NewPlaceOrderTool(deps))
	registry.Register("cancel_order", trading.NewCancelOrderTool(deps))
	log.Debug("Registered trading tools")

	// ========================================
	// Risk Management Tools
	// ========================================
	// Pre-Trade Check (ALL-IN-ONE) - replaces check_circuit_breaker, validate_trade
	registry.Register("pre_trade_check", toolrisk.NewPreTradeCheckTool(deps))
	registry.Register("emergency_close_all", toolrisk.NewEmergencyCloseAllTool(deps))
	log.Debug("Registered risk management tools")
	// ========================================
	// Memory Tools
	// ========================================
	registry.Register("search_memory", toolmemory.NewSearchMemoryTool(deps))
	registry.Register("save_memory", toolmemory.NewSaveMemoryTool(deps)) // Universal: analysis, insights, observations
	log.Debug("Registered memory tools")
	// Expert agent tools will be registered separately after agent factory initialization
	log.Infof("Tool registration complete: %d tools available", len(registry.List()))
}

// RegisterExpertTools registers agent-as-tool expert consultants
// Must be called after agent factory is created
func RegisterExpertTools(registry *Registry, expertTools map[string]tool.Tool, log *logger.Logger) {
	if len(expertTools) == 0 {
		log.Warn("No expert tools provided for registration")
		return
	}
	for name, t := range expertTools {
		registry.Register(name, t)
	}
	log.Infof("✓ Registered %d expert agent tools", len(expertTools))
}
