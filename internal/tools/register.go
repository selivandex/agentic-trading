package tools

import (
	"prometheus/internal/adapters/kafka"
	"prometheus/internal/tools/indicators"
	"prometheus/internal/tools/market"
	toolmemory "prometheus/internal/tools/memory"
	"prometheus/internal/tools/orderflow"
	toolrisk "prometheus/internal/tools/risk"
	"prometheus/internal/tools/shared"
	"prometheus/internal/tools/smc"
	"prometheus/internal/tools/trading"
	"prometheus/pkg/logger"

	"google.golang.org/adk/tool"
)

// RegisterAllTools registers all available tools in the registry
// Note: kafkaProducer is passed separately because publish_opportunity needs the concrete type
func RegisterAllTools(registry *Registry, deps shared.Deps, kafkaProducer interface{}) {
	log := deps.Log.With("component", "tool_registration")
	// Note: All tools now use shared.NewToolBuilder() fluent API with built-in middleware:
	// - .WithRetry(attempts, backoff) - automatic retry with exponential backoff
	// - .WithTimeout(duration) - execution timeout enforcement
	//
	// Stats tracking is handled by ADK callbacks (see internal/agents/callbacks/tool.go)
	// Middleware is configured in each tool's NewXXXTool() constructor via ToolBuilder
	// Example: shared.NewToolBuilder(...).WithTimeout(10*time.Second).WithRetry(3, 500*time.Millisecond).Build()
	// ========================================
	// Market Data Tools
	// ========================================
	registry.Register("get_price", market.NewGetPriceTool(deps))
	registry.Register("get_ohlcv", market.NewGetOHLCVTool(deps))
	registry.Register("get_orderbook", market.NewGetOrderBookTool(deps))
	registry.Register("get_trades", market.NewGetTradesTool(deps))

	// publish_opportunity requires kafka producer (concrete type, not interface)
	if kafkaProducer != nil {
		if kp, ok := kafkaProducer.(*kafka.Producer); ok {
			registry.Register("publish_opportunity", market.NewPublishOpportunityTool(kp))
		}
	}

	log.Debug("Registered market data tools")
	// ========================================
	// Technical Indicators - Momentum
	// ========================================
	registry.Register("rsi", indicators.NewRSITool(deps))
	registry.Register("macd", indicators.NewMACDTool(deps))
	registry.Register("stochastic", indicators.NewStochasticTool(deps))
	registry.Register("cci", indicators.NewCCITool(deps))
	registry.Register("roc", indicators.NewROCTool(deps))
	log.Debug("Registered momentum indicator tools")
	// ========================================
	// Technical Indicators - Volatility
	// ========================================
	registry.Register("atr", indicators.NewATRTool(deps))
	registry.Register("bollinger", indicators.NewBollingerTool(deps))
	registry.Register("keltner", indicators.NewKeltnerTool(deps))
	log.Debug("Registered volatility indicator tools")
	// ========================================
	// Technical Indicators - Trend
	// ========================================
	registry.Register("ema", indicators.NewEMATool(deps))
	registry.Register("sma", indicators.NewSMATool(deps))
	registry.Register("ema_ribbon", indicators.NewEMARibbonTool(deps))
	registry.Register("supertrend", indicators.NewSupertrendTool(deps))
	registry.Register("ichimoku", indicators.NewIchimokuTool(deps))
	registry.Register("pivot_points", indicators.NewPivotPointsTool(deps))
	log.Debug("Registered trend indicator tools")
	// ========================================
	// Technical Indicators - Volume
	// ========================================
	registry.Register("vwap", indicators.NewVWAPTool(deps))
	registry.Register("obv", indicators.NewOBVTool(deps))
	registry.Register("volume_profile", indicators.NewVolumeProfileTool(deps))
	registry.Register("delta_volume", indicators.NewDeltaVolumeTool(deps))
	log.Debug("Registered volume indicator tools")
	// ========================================
	// Smart Money Concepts (SMC/ICT) Tools
	// ========================================
	registry.Register("detect_fvg", smc.NewDetectFVGTool(deps))
	registry.Register("detect_order_blocks", smc.NewDetectOrderBlocksTool(deps))
	registry.Register("get_swing_points", smc.NewGetSwingPointsTool(deps))
	registry.Register("detect_liquidity_zones", smc.NewDetectLiquidityZonesTool(deps))
	registry.Register("detect_stop_hunt", smc.NewDetectStopHuntTool(deps))
	registry.Register("detect_imbalances", smc.NewDetectImbalancesTool(deps))
	registry.Register("get_market_structure", smc.NewGetMarketStructureTool(deps))
	log.Debug("Registered SMC/ICT tools")
	// ========================================
	// Order Flow Tools
	// ========================================
	registry.Register("get_trade_imbalance", orderflow.NewGetTradeImbalanceTool(deps))
	registry.Register("get_cvd", orderflow.NewGetCVDTool(deps))
	registry.Register("get_whale_trades", orderflow.NewGetWhaleTradesTool(deps))
	registry.Register("get_orderbook_imbalance", orderflow.NewGetOrderbookImbalanceTool(deps))
	registry.Register("get_tick_speed", orderflow.NewGetTickSpeedTool(deps))
	log.Debug("Registered order flow tools")
	// ========================================
	// Trading Tools (user-specific, requires exchange account)
	// ========================================
	registry.Register("get_balance", trading.NewGetBalanceTool(deps))
	registry.Register("get_positions", trading.NewGetPositionsTool(deps))
	registry.Register("get_portfolio_summary", trading.NewGetPortfolioSummaryTool(deps))
	registry.Register("place_order", trading.NewPlaceOrderTool(deps))
	registry.Register("cancel_order", trading.NewCancelOrderTool(deps))
	log.Debug("Registered trading tools")
	// ========================================
	// Risk Management Tools
	// ========================================
	registry.Register("check_circuit_breaker", toolrisk.NewCheckCircuitBreakerTool(deps))
	registry.Register("validate_trade", toolrisk.NewValidateTradeTool(deps))
	registry.Register("emergency_close_all", toolrisk.NewEmergencyCloseAllTool(deps))
	registry.Register("get_user_risk_profile", toolrisk.NewGetUserRiskProfileTool(deps))
	log.Debug("Registered risk management tools")
	// ========================================
	// Memory Tools
	// ========================================
	registry.Register("search_memory", toolmemory.NewSearchMemoryTool(deps))
	registry.Register("save_analysis", toolmemory.NewSaveAnalysisTool(deps))
	registry.Register("save_insight", toolmemory.NewSaveInsightTool(deps))
	registry.Register("record_reasoning", toolmemory.NewRecordReasoningTool(deps))
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
