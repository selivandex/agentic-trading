package journal

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// JournalEntry represents a trade journal entry with reflection
type JournalEntry struct {
	ID     uuid.UUID `db:"id"`
	UserID uuid.UUID `db:"user_id"`
	TradeID uuid.UUID `db:"trade_id"`

	// Trade details
	Symbol     string          `db:"symbol"`
	Side       string          `db:"side"`
	EntryPrice decimal.Decimal `db:"entry_price"`
	ExitPrice  decimal.Decimal `db:"exit_price"`
	Size       decimal.Decimal `db:"size"`
	PnL        decimal.Decimal `db:"pnl"`
	PnLPercent decimal.Decimal `db:"pnl_percent"`

	// Strategy info
	StrategyUsed string `db:"strategy_used"`
	Timeframe    string `db:"timeframe"`
	SetupType    string `db:"setup_type"` // breakout, reversal, trend_follow

	// Decision context
	MarketRegime    string  `db:"market_regime"`    // trend, range, volatile
	EntryReasoning  string  `db:"entry_reasoning"`
	ExitReasoning   string  `db:"exit_reasoning"`
	ConfidenceScore float64 `db:"confidence_score"`

	// Indicators at entry
	RSIAtEntry    float64 `db:"rsi_at_entry"`
	ATRAtEntry    float64 `db:"atr_at_entry"`
	VolumeAtEntry float64 `db:"volume_at_entry"`

	// Outcome analysis
	WasCorrectEntry bool            `db:"was_correct_entry"`
	WasCorrectExit  bool            `db:"was_correct_exit"`
	MaxDrawdown     decimal.Decimal `db:"max_drawdown"`
	MaxProfit       decimal.Decimal `db:"max_profit"`
	HoldDuration    int64           `db:"hold_duration"` // Duration in seconds

	// Lessons learned (AI generated)
	LessonsLearned  string `db:"lessons_learned"`
	ImprovementTips string `db:"improvement_tips"`

	CreatedAt time.Time `db:"created_at"`
}

// StrategyStats represents aggregated strategy performance
type StrategyStats struct {
	StrategyName  string          `db:"strategy_name"`
	TotalTrades   int             `db:"total_trades"`
	WinningTrades int             `db:"winning_trades"`
	LosingTrades  int             `db:"losing_trades"`
	WinRate       float64         `db:"win_rate"`
	AvgWin        decimal.Decimal `db:"avg_win"`
	AvgLoss       decimal.Decimal `db:"avg_loss"`
	ProfitFactor  decimal.Decimal `db:"profit_factor"`
	ExpectedValue decimal.Decimal `db:"expected_value"`
	MaxDrawdown   decimal.Decimal `db:"max_drawdown"`
	SharpeRatio   float64         `db:"sharpe_ratio"`
	IsActive      bool            `db:"is_active"` // Can be disabled if underperforming
	LastUpdated   time.Time       `db:"last_updated"`
}


import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// JournalEntry represents a trade journal entry with reflection
type JournalEntry struct {
	ID     uuid.UUID `db:"id"`
	UserID uuid.UUID `db:"user_id"`
	TradeID uuid.UUID `db:"trade_id"`

	// Trade details
	Symbol     string          `db:"symbol"`
	Side       string          `db:"side"`
	EntryPrice decimal.Decimal `db:"entry_price"`
	ExitPrice  decimal.Decimal `db:"exit_price"`
	Size       decimal.Decimal `db:"size"`
	PnL        decimal.Decimal `db:"pnl"`
	PnLPercent decimal.Decimal `db:"pnl_percent"`

	// Strategy info
	StrategyUsed string `db:"strategy_used"`
	Timeframe    string `db:"timeframe"`
	SetupType    string `db:"setup_type"` // breakout, reversal, trend_follow

	// Decision context
	MarketRegime    string  `db:"market_regime"`    // trend, range, volatile
	EntryReasoning  string  `db:"entry_reasoning"`
	ExitReasoning   string  `db:"exit_reasoning"`
	ConfidenceScore float64 `db:"confidence_score"`

	// Indicators at entry
	RSIAtEntry    float64 `db:"rsi_at_entry"`
	ATRAtEntry    float64 `db:"atr_at_entry"`
	VolumeAtEntry float64 `db:"volume_at_entry"`

	// Outcome analysis
	WasCorrectEntry bool            `db:"was_correct_entry"`
	WasCorrectExit  bool            `db:"was_correct_exit"`
	MaxDrawdown     decimal.Decimal `db:"max_drawdown"`
	MaxProfit       decimal.Decimal `db:"max_profit"`
	HoldDuration    int64           `db:"hold_duration"` // Duration in seconds

	// Lessons learned (AI generated)
	LessonsLearned  string `db:"lessons_learned"`
	ImprovementTips string `db:"improvement_tips"`

	CreatedAt time.Time `db:"created_at"`
}

// StrategyStats represents aggregated strategy performance
type StrategyStats struct {
	StrategyName  string          `db:"strategy_name"`
	TotalTrades   int             `db:"total_trades"`
	WinningTrades int             `db:"winning_trades"`
	LosingTrades  int             `db:"losing_trades"`
	WinRate       float64         `db:"win_rate"`
	AvgWin        decimal.Decimal `db:"avg_win"`
	AvgLoss       decimal.Decimal `db:"avg_loss"`
	ProfitFactor  decimal.Decimal `db:"profit_factor"`
	ExpectedValue decimal.Decimal `db:"expected_value"`
	MaxDrawdown   decimal.Decimal `db:"max_drawdown"`
	SharpeRatio   float64         `db:"sharpe_ratio"`
	IsActive      bool            `db:"is_active"` // Can be disabled if underperforming
	LastUpdated   time.Time       `db:"last_updated"`
}

