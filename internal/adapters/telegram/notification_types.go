package telegram

import "time"

// Notification data types for Telegram messages
// These are used by notification consumer to send structured notifications

// PositionOpenedData represents data for position opened notification
type PositionOpenedData struct {
	Symbol     string
	Side       string
	EntryPrice float64
	Amount     float64
	StopLoss   float64
	TakeProfit float64
	Exchange   string
	Reasoning  string
	Timestamp  time.Time
}

// PositionClosedData represents data for position closed notification
type PositionClosedData struct {
	Symbol      string
	Side        string
	EntryPrice  float64
	ExitPrice   float64
	PnL         float64
	PnLPercent  float64
	Duration    time.Duration
	CloseReason string
	Timestamp   time.Time
}

// StopLossHitData represents data for stop loss hit notification
type StopLossHitData struct {
	Symbol      string
	Side        string
	EntryPrice  float64
	StopPrice   float64
	Loss        float64
	LossPercent float64
	Timestamp   time.Time
}

// TakeProfitHitData represents data for take profit hit notification
type TakeProfitHitData struct {
	Symbol        string
	Side          string
	EntryPrice    float64
	TargetPrice   float64
	Profit        float64
	ProfitPercent float64
	Timestamp     time.Time
}

// CircuitBreakerData represents data for circuit breaker notification
type CircuitBreakerData struct {
	Reason          string
	DailyLoss       float64
	LossPercent     float64
	ConsecutiveLoss int
	Timestamp       time.Time
	Action          string
}

// ExchangeDeactivatedData represents data for exchange deactivated notification
type ExchangeDeactivatedData struct {
	Exchange     string
	Label        string
	Reason       string
	ErrorMessage string
	IsTestnet    bool
}

// InvestmentAcceptedData represents data for investment accepted notification
type InvestmentAcceptedData struct {
	Capital     float64
	RiskProfile string
	Exchange    string
}

// PortfolioCreatedData represents data for portfolio created notification
type PortfolioCreatedData struct {
	StrategyName   string
	Invested       float64
	PositionsCount int
}
