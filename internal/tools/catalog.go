package tools

import (
	"encoding/json"
	"sync"
)

// RiskLevel defines the risk level of a tool operation
type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "none"     // Read-only operations with no side effects
	RiskLevelLow      RiskLevel = "low"      // Low-risk operations (queries, analysis)
	RiskLevelMedium   RiskLevel = "medium"   // Medium-risk operations (single order placement)
	RiskLevelHigh     RiskLevel = "high"     // High-risk operations (large orders, multiple actions)
	RiskLevelCritical RiskLevel = "critical" // Critical operations (emergency actions, kill switches)
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
	RiskLevel    RiskLevel              `json:"risk_level"`    // Risk level of the operation
}

// Definition is an alias for backward compatibility
type Definition = ToolDefinition

// ToolCategory represents a category of tools
type ToolCategory string

const (
	CategoryMarketData  ToolCategory = "market_data" // Real-time market analysis, OHLCV
	CategorySMC         ToolCategory = "smc"         // Smart Money Concepts analysis
	CategorySentiment   ToolCategory = "sentiment"   // Fear & Greed, news
	CategoryOnChain     ToolCategory = "onchain"     // Whale movements, exchange flows
	CategoryMacro       ToolCategory = "macro"       // Economic calendar
	CategoryDerivatives ToolCategory = "derivatives" // Funding rates, open interest
	CategoryCorrelation ToolCategory = "correlation" // Cross-asset correlations
	CategoryAccount     ToolCategory = "account"     // Account status
	CategoryExecution   ToolCategory = "execution"   // Place/cancel orders
	CategoryRisk        ToolCategory = "risk"        // Pre-trade checks, emergency actions
	CategoryMemory      ToolCategory = "memory"      // Memory search/save
	CategoryEvaluation  ToolCategory = "evaluation"  // Strategy stats, trade journal
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
		// ========================================
		// COMPREHENSIVE ANALYSIS (ALL-IN-ONE) - USE THESE!
		// ========================================
		{
			Name:         "get_technical_analysis",
			Category:     string(CategoryMarketData),
			Description:  "Get comprehensive technical analysis with ALL indicators in one call: momentum (RSI, MACD, Stochastic, CCI, ROC), volatility (ATR, Bollinger, Keltner), trend (EMA ribbon, Supertrend, Ichimoku, Pivot Points), volume (VWAP, OBV, Volume Profile, Delta). Returns overall signal assessment.",
			RequiresAuth: false,
			RateLimit:    30,
			RiskLevel:    RiskLevelNone,
		},
		{
			Name:         "get_smc_analysis",
			Category:     string(CategorySMC),
			Description:  "Get comprehensive Smart Money Concepts analysis in one call: market structure (trend, BOS, CHoCH), Fair Value Gaps, Order Blocks, liquidity zones, swing points. Returns overall SMC signal.",
			RequiresAuth: false,
			RateLimit:    30,
			RiskLevel:    RiskLevelNone,
		},

		// ========================================
		// Market Data & Analysis (ALL-IN-ONE)
		// Replaces: get_price, get_ohlcv, get_trades, get_orderbook, get_order_flow_analysis
		{
			Name:         "get_market_analysis",
			Category:     string(CategoryMarketData),
			Description:  "Get comprehensive real-time market analysis. Returns: price (bid/ask/spread/change%), volume (total/hourly estimate), order flow (CVD/trade imbalance), whale activity, orderbook (pressure/walls), tick speed, overall signal. Human-readable text format.",
			RequiresAuth: false,
			RateLimit:    30,
			RiskLevel:    RiskLevelNone,
		},
		// Sentiment
		{
			Name:         "get_fear_greed",
			Category:     string(CategorySentiment),
			Description:  "Get Fear & Greed Index",
			RequiresAuth: false,
			RateLimit:    10,
			RiskLevel:    RiskLevelNone,
		},
		{
			Name:         "get_news",
			Category:     string(CategorySentiment),
			Description:  "Get latest crypto news",
			RequiresAuth: false,
			RateLimit:    30,
			RiskLevel:    RiskLevelNone,
		},
		// On-Chain
		{
			Name:         "get_whale_movements",
			Category:     string(CategoryOnChain),
			Description:  "Get large wallet transfers",
			RequiresAuth: false,
			RateLimit:    30,
			RiskLevel:    RiskLevelNone,
		},
		{
			Name:         "get_exchange_flows",
			Category:     string(CategoryOnChain),
			Description:  "Get exchange inflow/outflow",
			RequiresAuth: false,
			RateLimit:    30,
			RiskLevel:    RiskLevelNone,
		},
		// Macro
		{
			Name:         "get_economic_calendar",
			Category:     string(CategoryMacro),
			Description:  "Get economic events calendar",
			RequiresAuth: false,
			RateLimit:    10,
			RiskLevel:    RiskLevelNone,
		},
		// Trading (requires user auth)
		// Account Status (ALL-IN-ONE) - replaces get_balance, get_positions, get_portfolio_summary, get_user_risk_profile
		{
			Name:         "get_account_status",
			Category:     string(CategoryAccount),
			Description:  "Get comprehensive account status. Returns connected exchanges, open positions, portfolio summary (PnL/exposure), health warnings, and user risk profile. Human-readable text format.",
			RequiresAuth: true,
			RateLimit:    30,
			RiskLevel:    RiskLevelLow,
		},
		{
			Name:         "place_order",
			Category:     string(CategoryExecution),
			Description:  "Place market/limit/stop order",
			RequiresAuth: true,
			RateLimit:    30,
			RiskLevel:    RiskLevelMedium,
		},
		{
			Name:         "cancel_order",
			Category:     string(CategoryExecution),
			Description:  "Cancel an order",
			RequiresAuth: true,
			RateLimit:    60,
			RiskLevel:    RiskLevelLow,
		},
		// Risk Management
		// Pre-Trade Check (ALL-IN-ONE) - replaces check_circuit_breaker, validate_trade
		{
			Name:         "pre_trade_check",
			Category:     string(CategoryRisk),
			Description:  "Comprehensive pre-trade validation. Checks circuit breaker status, validates trade parameters against user risk limits. USE THIS BEFORE PLACING ANY ORDER.",
			RequiresAuth: true,
			RateLimit:    0,
			RiskLevel:    RiskLevelLow,
		},
		{
			Name:         "emergency_close_all",
			Category:     string(CategoryRisk),
			Description:  "Emergency close all positions (kill switch)",
			RequiresAuth: true,
			RateLimit:    0,
			RiskLevel:    RiskLevelCritical,
		},
		{
			Name:         "publish_opportunity",
			Category:     string(CategoryMarketData),
			Description:  "Publish validated trading opportunity to event stream",
			RequiresAuth: false,
			RateLimit:    10,
			RiskLevel:    RiskLevelLow,
		},
		// Memory
		{
			Name:         "search_memory",
			Category:     string(CategoryMemory),
			Description:  "Search past memories using text query. Finds similar memories (analysis, insights, patterns, observations) using semantic search.",
			RequiresAuth: true,
			RateLimit:    60,
			RiskLevel:    RiskLevelLow,
		},
		{
			Name:         "save_memory",
			Category:     string(CategoryMemory),
			Description:  "Save to long-term memory. Types: 'analysis' (structured JSON results), 'insight' (learnings/patterns as text), 'observation' (market notes). Replaces save_analysis and save_insight.",
			RequiresAuth: true,
			RateLimit:    120,
			RiskLevel:    RiskLevelLow,
		},
		// Evaluation
		{
			Name:         "get_strategy_stats",
			Category:     string(CategoryEvaluation),
			Description:  "Get strategy statistics",
			RequiresAuth: true,
			RateLimit:    30,
			RiskLevel:    RiskLevelLow,
		},
		{
			Name:         "get_trade_journal",
			Category:     string(CategoryEvaluation),
			Description:  "Get trade journal entries",
			RequiresAuth: true,
			RateLimit:    30,
			RiskLevel:    RiskLevelLow,
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
		CategoryMarketData, // get_market_analysis, get_technical_analysis, get_ohlcv
		CategorySMC,        // get_smc_analysis
		CategorySentiment,
		CategoryOnChain,
		CategoryMacro,
		CategoryDerivatives,
		CategoryCorrelation,
		CategoryAccount,   // get_account_status
		CategoryExecution, // place_order, cancel_order
		CategoryRisk,      // pre_trade_check, emergency_close_all
		CategoryMemory,
		CategoryEvaluation,
	}
}
