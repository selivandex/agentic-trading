package tools

import (
	"encoding/json"
	"sync"
)

// ToolDefinition describes a tool's metadata
type ToolDefinition struct {
	Name         string                 `json:"name"`
	Category     string                 `json:"category"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema,omitempty"`
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"`
	RequiresAuth bool                   `json:"requires_auth"` // If tool needs user-specific credentials
	RateLimit    int                    `json:"rate_limit"`    // Calls per minute (0 = unlimited)
}

// Definition is an alias for backward compatibility
type Definition = ToolDefinition

// ToolCategory represents a category of tools
type ToolCategory string

const (
	CategoryMarketData  ToolCategory = "market_data"
	CategoryMomentum    ToolCategory = "momentum"
	CategoryVolatility  ToolCategory = "volatility"
	CategoryTrend       ToolCategory = "trend"
	CategoryVolume      ToolCategory = "volume"
	CategorySMC         ToolCategory = "smc"
	CategoryOrderFlow   ToolCategory = "order_flow"
	CategorySentiment   ToolCategory = "sentiment"
	CategoryOnChain     ToolCategory = "onchain"
	CategoryMacro       ToolCategory = "macro"
	CategoryDerivatives ToolCategory = "derivatives"
	CategoryCorrelation ToolCategory = "correlation"
	CategoryAccount     ToolCategory = "account"
	CategoryExecution   ToolCategory = "execution"
	CategoryRisk        ToolCategory = "risk"
	CategoryMemory      ToolCategory = "memory"
	CategoryEvaluation  ToolCategory = "evaluation"
)

var (
	catalog     []ToolDefinition
	catalogOnce sync.Once
)

// Definitions returns all tool definitions
func Definitions() []ToolDefinition {
	catalogOnce.Do(initCatalog)
	return catalog
}

// DefinitionsByCategory returns tools filtered by category
func DefinitionsByCategory(category ToolCategory) []ToolDefinition {
	catalogOnce.Do(initCatalog)

	var filtered []ToolDefinition
	for _, def := range catalog {
		if def.Category == string(category) {
			filtered = append(filtered, def)
		}
	}
	return filtered
}

// DefinitionByName returns a tool definition by name
func DefinitionByName(name string) (*ToolDefinition, bool) {
	catalogOnce.Do(initCatalog)

	for _, def := range catalog {
		if def.Name == name {
			return &def, true
		}
	}
	return nil, false
}

// initCatalog initializes the tool catalog
func initCatalog() {
	catalog = []ToolDefinition{
		// Market Data Tools
		{
			Name:         "get_price",
			Category:     string(CategoryMarketData),
			Description:  "Get current price with bid/ask spread",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_ohlcv",
			Category:     string(CategoryMarketData),
			Description:  "Get historical OHLCV candles",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_orderbook",
			Category:     string(CategoryMarketData),
			Description:  "Get order book depth",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_trades",
			Category:     string(CategoryMarketData),
			Description:  "Get recent trades (tape)",
			RequiresAuth: false,
			RateLimit:    60,
		},

		// Momentum Indicators
		{
			Name:         "rsi",
			Category:     string(CategoryMomentum),
			Description:  "Calculate Relative Strength Index",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "macd",
			Category:     string(CategoryMomentum),
			Description:  "Calculate MACD indicator",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "stochastic",
			Category:     string(CategoryMomentum),
			Description:  "Calculate Stochastic Oscillator",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "cci",
			Category:     string(CategoryMomentum),
			Description:  "Calculate Commodity Channel Index",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "roc",
			Category:     string(CategoryMomentum),
			Description:  "Calculate Rate of Change",
			RequiresAuth: false,
			RateLimit:    120,
		},

		// Volatility Indicators
		{
			Name:         "atr",
			Category:     string(CategoryVolatility),
			Description:  "Calculate Average True Range",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "bollinger",
			Category:     string(CategoryVolatility),
			Description:  "Calculate Bollinger Bands",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "keltner",
			Category:     string(CategoryVolatility),
			Description:  "Calculate Keltner Channels",
			RequiresAuth: false,
			RateLimit:    120,
		},

		// Trend Indicators
		{
			Name:         "ema",
			Category:     string(CategoryTrend),
			Description:  "Calculate Exponential Moving Average",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "sma",
			Category:     string(CategoryTrend),
			Description:  "Calculate Simple Moving Average",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "ema_ribbon",
			Category:     string(CategoryTrend),
			Description:  "Calculate multiple EMAs (9, 21, 55, 200)",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "supertrend",
			Category:     string(CategoryTrend),
			Description:  "Calculate Supertrend indicator",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "ichimoku",
			Category:     string(CategoryTrend),
			Description:  "Calculate Ichimoku Cloud",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "pivot_points",
			Category:     string(CategoryTrend),
			Description:  "Calculate pivot points and support/resistance",
			RequiresAuth: false,
			RateLimit:    120,
		},

		// Volume Indicators
		{
			Name:         "vwap",
			Category:     string(CategoryVolume),
			Description:  "Calculate Volume Weighted Average Price",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "obv",
			Category:     string(CategoryVolume),
			Description:  "Calculate On-Balance Volume",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "volume_profile",
			Category:     string(CategoryVolume),
			Description:  "Calculate Volume Profile histogram",
			RequiresAuth: false,
			RateLimit:    120,
		},
		{
			Name:         "delta_volume",
			Category:     string(CategoryVolume),
			Description:  "Calculate Buy vs Sell volume delta",
			RequiresAuth: false,
			RateLimit:    120,
		},

		// Smart Money Concepts
		{
			Name:         "detect_fvg",
			Category:     string(CategorySMC),
			Description:  "Detect Fair Value Gaps",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "detect_order_blocks",
			Category:     string(CategorySMC),
			Description:  "Detect Order Blocks",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_swing_points",
			Category:     string(CategorySMC),
			Description:  "Identify swing highs and lows",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "detect_liquidity_zones",
			Category:     string(CategorySMC),
			Description:  "Detect liquidity zones and pools",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "detect_stop_hunt",
			Category:     string(CategorySMC),
			Description:  "Detect stop hunts and liquidity sweeps",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "detect_imbalances",
			Category:     string(CategorySMC),
			Description:  "Detect price imbalances",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_market_structure",
			Category:     string(CategorySMC),
			Description:  "Analyze market structure (BOS, CHoCH)",
			RequiresAuth: false,
			RateLimit:    60,
		},

		// Order Flow
		{
			Name:         "get_trade_imbalance",
			Category:     string(CategoryOrderFlow),
			Description:  "Get buy vs sell pressure",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_cvd",
			Category:     string(CategoryOrderFlow),
			Description:  "Calculate Cumulative Volume Delta",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_whale_trades",
			Category:     string(CategoryOrderFlow),
			Description:  "Detect large trades (whales)",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_orderbook_imbalance",
			Category:     string(CategoryOrderFlow),
			Description:  "Calculate orderbook bid/ask imbalance",
			RequiresAuth: false,
			RateLimit:    60,
		},
		{
			Name:         "get_tick_speed",
			Category:     string(CategoryOrderFlow),
			Description:  "Calculate trade velocity (ticks per second)",
			RequiresAuth: false,
			RateLimit:    60,
		},

		// Sentiment
		{
			Name:         "get_fear_greed",
			Category:     string(CategorySentiment),
			Description:  "Get Fear & Greed Index",
			RequiresAuth: false,
			RateLimit:    10,
		},
		{
			Name:         "get_news",
			Category:     string(CategorySentiment),
			Description:  "Get latest crypto news",
			RequiresAuth: false,
			RateLimit:    30,
		},

		// On-Chain
		{
			Name:         "get_whale_movements",
			Category:     string(CategoryOnChain),
			Description:  "Get large wallet transfers",
			RequiresAuth: false,
			RateLimit:    30,
		},
		{
			Name:         "get_exchange_flows",
			Category:     string(CategoryOnChain),
			Description:  "Get exchange inflow/outflow",
			RequiresAuth: false,
			RateLimit:    30,
		},

		// Macro
		{
			Name:         "get_economic_calendar",
			Category:     string(CategoryMacro),
			Description:  "Get economic events calendar",
			RequiresAuth: false,
			RateLimit:    10,
		},

		// Trading (requires user auth)
		{
			Name:         "get_balance",
			Category:     string(CategoryAccount),
			Description:  "Get account balance",
			RequiresAuth: true,
			RateLimit:    60,
		},
		{
			Name:         "get_positions",
			Category:     string(CategoryAccount),
			Description:  "Get open positions",
			RequiresAuth: true,
			RateLimit:    60,
		},
		{
			Name:         "place_order",
			Category:     string(CategoryExecution),
			Description:  "Place market/limit/stop order",
			RequiresAuth: true,
			RateLimit:    30,
		},
		{
			Name:         "cancel_order",
			Category:     string(CategoryExecution),
			Description:  "Cancel an order",
			RequiresAuth: true,
			RateLimit:    60,
		},

		// Risk Management
		{
			Name:         "check_circuit_breaker",
			Category:     string(CategoryRisk),
			Description:  "Check if trading is allowed",
			RequiresAuth: true,
			RateLimit:    0,
		},
		{
			Name:         "validate_trade",
			Category:     string(CategoryRisk),
			Description:  "Validate trade before execution",
			RequiresAuth: true,
			RateLimit:    0,
		},
		{
			Name:         "emergency_close_all",
			Category:     string(CategoryRisk),
			Description:  "Emergency close all positions (kill switch)",
			RequiresAuth: true,
			RateLimit:    0,
		},

		// Memory
		{
			Name:         "search_memory",
			Category:     string(CategoryMemory),
			Description:  "Semantic search past memories",
			RequiresAuth: true,
			RateLimit:    60,
		},
		{
			Name:         "store_memory",
			Category:     string(CategoryMemory),
			Description:  "Store observation/decision",
			RequiresAuth: true,
			RateLimit:    120,
		},

		// Evaluation
		{
			Name:         "get_strategy_stats",
			Category:     string(CategoryEvaluation),
			Description:  "Get strategy statistics",
			RequiresAuth: true,
			RateLimit:    30,
		},
		{
			Name:         "get_trade_journal",
			Category:     string(CategoryEvaluation),
			Description:  "Get trade journal entries",
			RequiresAuth: true,
			RateLimit:    30,
		},
	}
}

// ToJSON exports catalog to JSON
func ToJSON() ([]byte, error) {
	catalogOnce.Do(initCatalog)
	return json.MarshalIndent(catalog, "", "  ")
}

// Categories returns all unique categories
func Categories() []ToolCategory {
	return []ToolCategory{
		CategoryMarketData,
		CategoryMomentum,
		CategoryVolatility,
		CategoryTrend,
		CategoryVolume,
		CategorySMC,
		CategoryOrderFlow,
		CategorySentiment,
		CategoryOnChain,
		CategoryMacro,
		CategoryDerivatives,
		CategoryCorrelation,
		CategoryAccount,
		CategoryExecution,
		CategoryRisk,
		CategoryMemory,
		CategoryEvaluation,
	}
}
