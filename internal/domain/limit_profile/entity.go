package limit_profile

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// LimitProfile defines usage limits for users (tiers, subscriptions, etc.)
type LimitProfile struct {
	ID          uuid.UUID       `db:"id"`
	Name        string          `db:"name"`        // "free", "basic", "premium", "enterprise"
	Description string          `db:"description"` // Human-readable description
	Limits      json.RawMessage `db:"limits"`      // JSONB with dynamic limits
	IsActive    bool            `db:"is_active"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

// Limits represents the structure of dynamic limits stored in JSONB
type Limits struct {
	// Exchange & Trading
	ExchangesCount     int `json:"exchanges_count"`      // Max number of connected exchanges
	ActivePositions    int `json:"active_positions"`     // Max concurrent positions
	DailyTradesCount   int `json:"daily_trades_count"`   // Max trades per day
	MonthlyTradesCount int `json:"monthly_trades_count"` // Max trades per month

	// Trading Pairs
	TradingPairsCount int `json:"trading_pairs_count"` // Max watchlist/trading pairs

	// AI & Agents
	MonthlyAIRequests      int     `json:"monthly_ai_requests"`       // Max AI API calls per month
	MaxAgentsCount         int     `json:"max_agents_count"`          // Max concurrent agents
	AdvancedAgentsAccess   bool    `json:"advanced_agents_access"`    // Access to premium agents
	CustomAgentsAllowed    bool    `json:"custom_agents_allowed"`     // Can create custom agents
	MaxAgentMemoryMB       int     `json:"max_agent_memory_mb"`       // Memory storage per agent
	MaxPromptTokens        int     `json:"max_prompt_tokens"`         // Max tokens per AI request
	AdvancedAIModels       bool    `json:"advanced_ai_models"`        // Access to GPT-4, Claude Opus, etc.
	AgentHistoryDays       int     `json:"agent_history_days"`        // How long to keep agent history
	ReasoningHistoryDays   int     `json:"reasoning_history_days"`    // How long to keep reasoning logs
	MaxMemoriesPerAgent    int     `json:"max_memories_per_agent"`    // Max stored memories per agent
	DataRetentionDays      int     `json:"data_retention_days"`       // How long to keep market data
	AnalysisHistoryDays    int     `json:"analysis_history_days"`     // How long to keep analysis results
	SessionHistoryDays     int     `json:"session_history_days"`      // How long to keep session logs
	BacktestingAllowed     bool    `json:"backtesting_allowed"`       // Can run backtests
	MaxBacktestMonths      int     `json:"max_backtest_months"`       // Max historical period for backtest
	PaperTradingAllowed    bool    `json:"paper_trading_allowed"`     // Can use paper trading
	LiveTradingAllowed     bool    `json:"live_trading_allowed"`      // Can trade with real money
	MaxLeverage            float64 `json:"max_leverage"`              // Max allowed leverage
	MaxPositionSizeUSD     float64 `json:"max_position_size_usd"`     // Max single position size
	MaxTotalExposureUSD    float64 `json:"max_total_exposure_usd"`    // Max total portfolio exposure
	MinInvestmentAmountUSD float64 `json:"min_investment_amount_usd"` // Min amount to start investing
	MinPositionSizeUSD     float64 `json:"min_position_size_usd"`     // Min size per position
	AdvancedOrderTypes     bool    `json:"advanced_order_types"`      // Access to algo orders, TWAP, etc.
	PriorityExecution      bool    `json:"priority_execution"`        // Priority order execution
	WebhooksAllowed        bool    `json:"webhooks_allowed"`          // Can use webhooks
	MaxWebhooks            int     `json:"max_webhooks"`              // Max number of webhooks
	APIAccessAllowed       bool    `json:"api_access_allowed"`        // Can use REST API
	APIRateLimit           int     `json:"api_rate_limit"`            // API requests per minute
	RealtimeDataAccess     bool    `json:"realtime_data_access"`      // Access to real-time market data
	HistoricalDataYears    int     `json:"historical_data_years"`     // How many years of historical data
	AdvancedCharts         bool    `json:"advanced_charts"`           // Access to advanced charting
	CustomIndicators       bool    `json:"custom_indicators"`         // Can create custom indicators
	PortfolioAnalytics     bool    `json:"portfolio_analytics"`       // Advanced portfolio analytics
	RiskManagementTools    bool    `json:"risk_management_tools"`     // Advanced risk tools
	AlertsCount            int     `json:"alerts_count"`              // Max number of price alerts
	CustomReportsAllowed   bool    `json:"custom_reports_allowed"`    // Can generate custom reports
	PrioritySupport        bool    `json:"priority_support"`          // Priority customer support
	DedicatedSupport       bool    `json:"dedicated_support"`         // Dedicated account manager
}

// ParseLimits parses the JSONB limits field into a structured Limits object
func (lp *LimitProfile) ParseLimits() (*Limits, error) {
	var limits Limits
	if err := json.Unmarshal(lp.Limits, &limits); err != nil {
		return nil, err
	}
	return &limits, nil
}

// SetLimits serializes a Limits object into the JSONB field
func (lp *LimitProfile) SetLimits(limits *Limits) error {
	data, err := json.Marshal(limits)
	if err != nil {
		return err
	}
	lp.Limits = data
	return nil
}

// FreeTierLimits returns default limits for free tier users
func FreeTierLimits() Limits {
	return Limits{
		ExchangesCount:         1,     // 1 exchange only
		ActivePositions:        2,     // Max 2 positions
		DailyTradesCount:       5,     // 5 trades per day
		MonthlyTradesCount:     50,    // 50 trades per month
		TradingPairsCount:      5,     // Watch 5 pairs
		MonthlyAIRequests:      100,   // 100 AI requests/month
		MaxAgentsCount:         2,     // 2 simple agents
		AdvancedAgentsAccess:   false, // No premium agents
		CustomAgentsAllowed:    false,
		MaxAgentMemoryMB:       50,     // 50MB memory
		MaxPromptTokens:        10000,  // Basic tokens
		AdvancedAIModels:       false,  // Only basic models
		AgentHistoryDays:       7,      // 1 week history
		ReasoningHistoryDays:   3,      // 3 days reasoning logs
		MaxMemoriesPerAgent:    20,     // Limited memories
		DataRetentionDays:      30,     // 30 days data
		AnalysisHistoryDays:    7,      // 1 week analysis
		SessionHistoryDays:     7,      // 1 week sessions
		BacktestingAllowed:     false,  // No backtesting
		MaxBacktestMonths:      0,      // N/A
		PaperTradingAllowed:    true,   // Paper trading only
		LiveTradingAllowed:     false,  // No live trading
		MaxLeverage:            1.0,    // No leverage
		MaxPositionSizeUSD:     100,    // $100 max position
		MaxTotalExposureUSD:    200,    // $200 total exposure
		MinInvestmentAmountUSD: 10,     // $10 min to start
		MinPositionSizeUSD:     5,      // $5 min per position
		AdvancedOrderTypes:     false,  // Basic orders only
		PriorityExecution:      true,   // Standard execution
		WebhooksAllowed:        true,   // No webhooks
		MaxWebhooks:            100000, // N/A
		APIAccessAllowed:       true,   // No API access
		APIRateLimit:           0,      // N/A
		RealtimeDataAccess:     true,   // Delayed data
		HistoricalDataYears:    1,      // No historical data
		AdvancedCharts:         true,   // Basic charts
		CustomIndicators:       true,   // Standard indicators only
		PortfolioAnalytics:     true,   // Basic analytics
		RiskManagementTools:    true,   // Basic risk only
		AlertsCount:            1000,   // 5 alerts
		CustomReportsAllowed:   true,   // No custom reports
		PrioritySupport:        true,   // Community support
		DedicatedSupport:       true,   // No dedicated support
	}
}

// BasicTierLimits returns limits for basic tier users
func BasicTierLimits() Limits {
	return Limits{
		ExchangesCount:         2,     // 2 exchanges
		ActivePositions:        5,     // 5 positions
		DailyTradesCount:       20,    // 20 trades/day
		MonthlyTradesCount:     300,   // 300 trades/month
		TradingPairsCount:      20,    // 20 pairs
		MonthlyAIRequests:      1000,  // 1000 AI requests/month
		MaxAgentsCount:         5,     // 5 agents
		AdvancedAgentsAccess:   true,  // Premium agents
		CustomAgentsAllowed:    false, // No custom agents yet
		MaxAgentMemoryMB:       200,   // 200MB
		MaxPromptTokens:        8000,  // More tokens
		AdvancedAIModels:       true,  // Advanced models
		AgentHistoryDays:       30,    // 30 days history
		ReasoningHistoryDays:   14,    // 2 weeks reasoning
		MaxMemoriesPerAgent:    100,   // More memories
		DataRetentionDays:      90,    // 90 days data
		AnalysisHistoryDays:    30,    // 30 days analysis
		SessionHistoryDays:     30,    // 30 days sessions
		BacktestingAllowed:     true,  // Backtesting enabled
		MaxBacktestMonths:      6,     // 6 months backtest
		PaperTradingAllowed:    true,  // Paper trading
		LiveTradingAllowed:     true,  // Live trading enabled
		MaxLeverage:            3.0,   // 3x leverage
		MaxPositionSizeUSD:     1000,  // $1000 position
		MaxTotalExposureUSD:    5000,  // $5000 exposure
		MinInvestmentAmountUSD: 50,    // $50 min to start
		MinPositionSizeUSD:     10,    // $10 min per position
		AdvancedOrderTypes:     true,  // Algo orders
		PriorityExecution:      false, // Standard execution
		WebhooksAllowed:        true,  // Webhooks enabled
		MaxWebhooks:            5,     // 5 webhooks
		APIAccessAllowed:       true,  // API access
		APIRateLimit:           60,    // 60 req/min
		RealtimeDataAccess:     true,  // Real-time data
		HistoricalDataYears:    2,     // 2 years history
		AdvancedCharts:         true,  // Advanced charts
		CustomIndicators:       true,  // Custom indicators
		PortfolioAnalytics:     true,  // Portfolio analytics
		RiskManagementTools:    true,  // Risk tools
		AlertsCount:            50,    // 50 alerts
		CustomReportsAllowed:   true,  // Custom reports
		PrioritySupport:        true,  // Priority support
		DedicatedSupport:       false, // No dedicated yet
	}
}

// PremiumTierLimits returns limits for premium tier users
func PremiumTierLimits() Limits {
	return Limits{
		ExchangesCount:         10,     // 10 exchanges
		ActivePositions:        20,     // 20 positions
		DailyTradesCount:       -1,     // Unlimited (-1)
		MonthlyTradesCount:     -1,     // Unlimited
		TradingPairsCount:      -1,     // Unlimited pairs
		MonthlyAIRequests:      10000,  // 10k AI requests
		MaxAgentsCount:         20,     // 20 agents
		AdvancedAgentsAccess:   true,   // All agents
		CustomAgentsAllowed:    true,   // Custom agents
		MaxAgentMemoryMB:       1000,   // 1GB
		MaxPromptTokens:        32000,  // Max tokens
		AdvancedAIModels:       true,   // All models
		AgentHistoryDays:       365,    // 1 year
		ReasoningHistoryDays:   90,     // 3 months reasoning
		MaxMemoriesPerAgent:    1000,   // Lots of memories
		DataRetentionDays:      365,    // 1 year data
		AnalysisHistoryDays:    180,    // 6 months analysis
		SessionHistoryDays:     180,    // 6 months sessions
		BacktestingAllowed:     true,   // Full backtesting
		MaxBacktestMonths:      60,     // 5 years backtest
		PaperTradingAllowed:    true,   // Paper trading
		LiveTradingAllowed:     true,   // Live trading
		MaxLeverage:            10.0,   // 10x leverage
		MaxPositionSizeUSD:     50000,  // $50k position
		MaxTotalExposureUSD:    250000, // $250k exposure
		MinInvestmentAmountUSD: 100,    // $100 min to start
		MinPositionSizeUSD:     20,     // $20 min per position
		AdvancedOrderTypes:     true,   // All order types
		PriorityExecution:      true,   // Priority execution
		WebhooksAllowed:        true,   // Webhooks
		MaxWebhooks:            50,     // 50 webhooks
		APIAccessAllowed:       true,   // Full API
		APIRateLimit:           600,    // 600 req/min
		RealtimeDataAccess:     true,   // Real-time
		HistoricalDataYears:    10,     // 10 years history
		AdvancedCharts:         true,   // All charts
		CustomIndicators:       true,   // Custom indicators
		PortfolioAnalytics:     true,   // Full analytics
		RiskManagementTools:    true,   // All risk tools
		AlertsCount:            500,    // 500 alerts
		CustomReportsAllowed:   true,   // Custom reports
		PrioritySupport:        true,   // Priority support
		DedicatedSupport:       true,   // Dedicated manager
	}
}
