package riskservice

import (
	"time"

	"prometheus/internal/domain/position"
)

// EvaluationRequest contains all information needed for risk evaluation
type EvaluationRequest struct {
	UserID       string
	Plan         *TradePlan
	CurrentPrice float64
	MarketData   *MarketData
}

// TradePlan represents a proposed trade to be evaluated
type TradePlan struct {
	Symbol     string
	Exchange   string
	Direction  string  // "long" or "short"
	Entry      float64 // entry price
	StopLoss   float64 // stop-loss price
	TakeProfit float64 // take-profit price
	Size       float64 // position size (in base currency)
	Leverage   float64 // leverage (1.0 = no leverage)
	OrderType  string  // "market", "limit", "stop_limit"
	Confidence float64 // 0-1, from opportunity signal
	Strategy   string  // "breakout", "reversal", etc (for stats)
}

// RiskDecision is the result of risk evaluation
type RiskDecision struct {
	UserID       string
	OriginalPlan *TradePlan
	FinalPlan    *TradePlan // adjusted plan (if approved)
	Approved     bool
	FinalReason  string // if rejected, why

	// Adjustments made
	AdjustedSize   float64
	SizeWasReduced bool

	// Risk metrics
	RiskAmount        float64
	RiskPercent       float64
	RiskRewardRatio   float64
	PortfolioMetrics  PortfolioMetrics
	MarginRequired    float64

	// Check results (for debugging and explanation)
	Checks map[string]CheckResult

	// Warnings (non-blocking issues)
	Warnings []string

	Timestamp      time.Time
	ProcessingTime time.Duration
}

// CheckResult represents result of a single risk check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
	Details interface{} // check-specific details
}

// MarketData contains market context for risk evaluation
type MarketData struct {
	CurrentPrice      float64
	Volume24h         float64
	Volatility        float64 // current volatility (e.g., ATR)
	AvgVolatility     float64 // average volatility
	LiquidityScore    float64 // 0-1, based on orderbook depth
	BidAskSpread      float64
	LastUpdateTime    time.Time
}

// Portfolio represents current portfolio state
type Portfolio struct {
	TotalValue       float64 // total portfolio value
	AvailableBalance float64 // available balance (not in positions)
	TotalExposure    float64 // sum of all position values
	AssetExposure    map[string]float64 // exposure per asset
	UsedMargin       float64 // margin used (for futures)
	PeakValue        float64 // all-time high value (for drawdown)
	Positions        []position.Position
}

// PortfolioMetrics contains calculated portfolio risk metrics
type PortfolioMetrics struct {
	TotalExposurePct      float64 // total exposure / total value
	AssetConcentrationPct float64 // largest single asset %
	CorrelatedExposurePct float64 // correlated assets %
	CurrentDrawdownPct    float64 // (peak - current) / peak
	MarginUsagePct        float64 // used margin / available margin
	NumPositions          int
}

// ValidationResult from pre-trade validator
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
	RR       float64 // calculated R:R ratio
}

// SizeResult from position sizer
type SizeResult struct {
	Size          float64 // final position size
	SizeUSD       float64 // size in USD
	RiskAmount    float64 // dollar amount at risk
	RiskPercent   float64 // % of portfolio at risk
	WasReduced    bool    // was size reduced from original?
	ReductionPct  float64 // by how much (%)
	Method        string  // "fixed_risk", "kelly", "fixed_pct"
}

// PortfolioCheckResult from portfolio checker
type PortfolioCheckResult struct {
	Approved bool
	Errors   []string
	Warnings []string
	Metrics  PortfolioMetrics
}

// CircuitBreakerResult from circuit breaker check
type CircuitBreakerResult struct {
	Triggered bool
	Reason    string // "daily_loss", "consecutive_losses", "manual"
	Message   string
	ResumeAt  time.Time
}

// CircuitBreakerStats contains circuit breaker statistics
type CircuitBreakerStats struct {
	DailyPnL          float64
	DailyPnLPercent   float64
	ConsecutiveLosses int
	ConsecutiveWins   int
	LastTradeTime     time.Time
	TradesToday       int
}

// MarketConditionResult from market conditioner
type MarketConditionResult struct {
	Acceptable bool   // are conditions acceptable?
	Critical   bool   // is this a critical failure?
	Message    string
	VolatilityMultiple float64
	LiquidityScore     float64
	HighImpactEvent    *HighImpactEvent // if event detected
}

// HighImpactEvent represents an upcoming high-impact economic event
type HighImpactEvent struct {
	Name       string
	Impact     string // "high", "medium", "low"
	Timestamp  time.Time
	MinutesTil int
}

// MarginCheckResult from margin calculator
type MarginCheckResult struct {
	Sufficient      bool
	Required        float64 // margin required for this trade
	Available       float64 // margin available
	UsageAfterTrade float64 // % of margin used after trade
	Message         string
}

// TimeRuleResult from time-based rules check
type TimeRuleResult struct {
	Allowed  bool
	Critical bool   // if true, blocks trade; if false, just warning
	Message  string
	Event    *HighImpactEvent // if blocked by event
}

// RiskStatus represents current risk status for a user
type RiskStatus struct {
	UserID            string
	CircuitBreaker    *CircuitBreakerResult
	Portfolio         *Portfolio
	DailyPnL          float64
	DailyPnLPercent   float64
	ConsecutiveLosses int
	TradesRemaining   int  // trades remaining before circuit breaker
	CanTrade          bool // overall trading allowed?
	Timestamp         time.Time
}

// TradeOutcome records the result of a trade for statistics
type TradeOutcome struct {
	UserID    string
	Symbol    string
	Direction string
	EntryPrice float64
	ExitPrice  float64
	Size       float64
	PnL        float64
	PnLPercent float64
	Winner     bool
	Duration   time.Duration
	Strategy   string
	Timestamp  time.Time
}

// Impulse represents an impulse move (for Order Block detection, etc.)
type Impulse struct {
	Direction string  // "up" or "down"
	Magnitude float64 // % move
}

