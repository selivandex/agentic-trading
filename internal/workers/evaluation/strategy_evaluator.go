package evaluation

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// StrategyEvaluator evaluates strategy performance and auto-disables underperformers
// Runs every 6 hours to assess recent performance
type StrategyEvaluator struct {
	*workers.BaseWorker
	userRepo        user.Repository
	tradingPairRepo trading_pair.Repository
	journalRepo     journal.Repository
	kafka           *kafka.Producer

	// Thresholds for strategy disabling
	minTrades          int     // Minimum trades before evaluation (default: 10)
	minWinRate         float64 // Minimum win rate % (default: 35)
	minProfitFactor    float64 // Minimum profit factor (default: 0.8)
	maxDrawdownPercent float64 // Maximum drawdown % before warning (default: 20)
}

// NewStrategyEvaluator creates a new strategy evaluator worker
func NewStrategyEvaluator(
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	journalRepo journal.Repository,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *StrategyEvaluator {
	return &StrategyEvaluator{
		BaseWorker:         workers.NewBaseWorker("strategy_evaluator", interval, enabled),
		userRepo:           userRepo,
		tradingPairRepo:    tradingPairRepo,
		journalRepo:        journalRepo,
		kafka:              kafka,
		minTrades:          10,
		minWinRate:         35.0,
		minProfitFactor:    0.8,
		maxDrawdownPercent: 20.0,
	}
}

// Run executes one iteration of strategy evaluation
func (se *StrategyEvaluator) Run(ctx context.Context) error {
	se.Log().Info("Strategy evaluator: starting evaluation")

	// Get all active users (using large limit)
	users, err := se.userRepo.List(ctx, 1000, 0)
	if err != nil {
		return errors.Wrap(err, "failed to list users")
	}

	// Filter only active users
	var activeUsers []*user.User
	for _, usr := range users {
		if usr.IsActive {
			activeUsers = append(activeUsers, usr)
		}
	}
	users = activeUsers

	if len(users) == 0 {
		se.Log().Debug("No active users for strategy evaluation")
		return nil
	}

	disabledStrategies := 0
	warnedStrategies := 0

	// Evaluate strategies for each user
	for _, usr := range users {
		disabled, warned, err := se.evaluateUserStrategies(ctx, usr)
		if err != nil {
			se.Log().Error("Failed to evaluate user strategies",
				"user_id", usr.ID,
				"error", err,
			)
			// Continue with other users
			continue
		}

		disabledStrategies += disabled
		warnedStrategies += warned
	}

	se.Log().Info("Strategy evaluation: complete",
		"users_processed", len(users),
		"strategies_disabled", disabledStrategies,
		"strategies_warned", warnedStrategies,
	)

	return nil
}

// evaluateUserStrategies evaluates all strategies for a single user
func (se *StrategyEvaluator) evaluateUserStrategies(ctx context.Context, usr *user.User) (int, int, error) {
	se.Log().Debug("Evaluating user strategies", "user_id", usr.ID)

	// Get strategy statistics for last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	stats, err := se.journalRepo.GetStrategyStats(ctx, usr.ID, thirtyDaysAgo)
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to get strategy stats")
	}

	if len(stats) == 0 {
		se.Log().Debug("No strategy stats available", "user_id", usr.ID)
		return 0, 0, nil
	}

	disabledCount := 0
	warnedCount := 0

	// Evaluate each strategy
	for _, stratStat := range stats {
		strategyName := stratStat.StrategyName
		action, reason := se.evaluateStrategy(&stratStat)

		switch action {
		case "disable":
			// Disable all trading pairs using this strategy
			if err := se.disableStrategy(ctx, usr.ID, strategyName); err != nil {
				se.Log().Error("Failed to disable strategy",
					"user_id", usr.ID,
					"strategy", strategyName,
					"error", err,
				)
				continue
			}

			se.Log().Warn("Strategy disabled due to poor performance",
				"user_id", usr.ID,
				"strategy", strategyName,
				"reason", reason,
				"win_rate", stratStat.WinRate,
				"profit_factor", stratStat.ProfitFactor,
				"total_trades", stratStat.TotalTrades,
			)

			se.publishStrategyDisabledEvent(ctx, usr.ID, strategyName, reason, &stratStat)
			disabledCount++

		case "warn":
			se.Log().Warn("Strategy performance warning",
				"user_id", usr.ID,
				"strategy", strategyName,
				"reason", reason,
				"win_rate", stratStat.WinRate,
				"profit_factor", stratStat.ProfitFactor,
			)

			se.publishStrategyWarningEvent(ctx, usr.ID, strategyName, reason, &stratStat)
			warnedCount++

		case "ok":
			// Strategy performing well
			se.Log().Debug("Strategy performing well",
				"user_id", usr.ID,
				"strategy", strategyName,
				"win_rate", stratStat.WinRate,
				"profit_factor", stratStat.ProfitFactor,
			)
		}
	}

	return disabledCount, warnedCount, nil
}

// evaluateStrategy evaluates a single strategy and returns action to take
func (se *StrategyEvaluator) evaluateStrategy(stats *journal.StrategyStats) (string, string) {
	// Not enough data yet
	if stats.TotalTrades < se.minTrades {
		return "ok", ""
	}

	// Check win rate
	if stats.WinRate < se.minWinRate {
		return "disable", "Win rate below minimum threshold"
	}

	// Check profit factor
	profitFactorFloat, _ := stats.ProfitFactor.Float64()
	if profitFactorFloat < se.minProfitFactor {
		return "disable", "Profit factor below minimum threshold"
	}

	// Check for extreme drawdown
	maxDrawdownFloat, _ := stats.MaxDrawdown.Float64()
	maxDrawdownPct := (maxDrawdownFloat / 100.0) * 100.0 // Convert to percentage if needed
	if maxDrawdownPct > se.maxDrawdownPercent {
		return "disable", "Maximum drawdown exceeded"
	}

	// Warning conditions (not critical but concerning)
	if stats.WinRate < se.minWinRate+5 {
		return "warn", "Win rate approaching minimum threshold"
	}

	if profitFactorFloat < se.minProfitFactor+0.2 {
		return "warn", "Profit factor approaching minimum threshold"
	}

	return "ok", ""
}

// disableStrategy disables all trading pairs using a specific strategy for a user
func (se *StrategyEvaluator) disableStrategy(ctx context.Context, userID uuid.UUID, strategy string) error {
	// Get all active trading pairs for this user with this strategy
	pairs, err := se.tradingPairRepo.GetActiveByUser(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "failed to get trading pairs")
	}

	disabledCount := 0

	for _, pair := range pairs {
		if pair.StrategyMode.String() == strategy {
			// Update trading pair to disabled
			if err := se.tradingPairRepo.Disable(ctx, pair.ID); err != nil {
				se.Log().Error("Failed to disable trading pair",
					"pair_id", pair.ID,
					"symbol", pair.Symbol,
					"error", err,
				)
				continue
			}

			disabledCount++
		}
	}

	se.Log().Info("Disabled trading pairs for strategy",
		"user_id", userID,
		"strategy", strategy,
		"pairs_disabled", disabledCount,
	)

	return nil
}

// Event structures

type StrategyDisabledEvent struct {
	UserID         string  `json:"user_id"`
	Strategy       string  `json:"strategy"`
	Reason         string  `json:"reason"`
	TotalTrades    int     `json:"total_trades"`
	WinRate        float64 `json:"win_rate"`
	ProfitFactor   string  `json:"profit_factor"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	TotalPnL       string  `json:"total_pnl"`
	Timestamp      time.Time `json:"timestamp"`
}

type StrategyWarningEvent struct {
	UserID         string  `json:"user_id"`
	Strategy       string  `json:"strategy"`
	Reason         string  `json:"reason"`
	TotalTrades    int     `json:"total_trades"`
	WinRate        float64 `json:"win_rate"`
	ProfitFactor   string  `json:"profit_factor"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	Timestamp      time.Time `json:"timestamp"`
}

func (se *StrategyEvaluator) publishStrategyDisabledEvent(
	ctx context.Context,
	userID uuid.UUID,
	strategy, reason string,
	stats *journal.StrategyStats,
) {
	maxDrawdownFloat, _ := stats.MaxDrawdown.Float64()
	event := StrategyDisabledEvent{
		UserID:         userID.String(),
		Strategy:       strategy,
		Reason:         reason,
		TotalTrades:    stats.TotalTrades,
		WinRate:        stats.WinRate,
		ProfitFactor:   stats.ProfitFactor.String(),
		MaxDrawdownPct: maxDrawdownFloat,
		TotalPnL:       decimal.Zero.String(), // TODO: Calculate total PnL
		Timestamp:      time.Now(),
	}

	if err := se.kafka.Publish(ctx, "strategy.disabled", userID.String(), event); err != nil {
		se.Log().Error("Failed to publish strategy disabled event", "error", err)
	}
}

func (se *StrategyEvaluator) publishStrategyWarningEvent(
	ctx context.Context,
	userID uuid.UUID,
	strategy, reason string,
	stats *journal.StrategyStats,
) {
	maxDrawdownFloat, _ := stats.MaxDrawdown.Float64()
	event := StrategyWarningEvent{
		UserID:         userID.String(),
		Strategy:       strategy,
		Reason:         reason,
		TotalTrades:    stats.TotalTrades,
		WinRate:        stats.WinRate,
		ProfitFactor:   stats.ProfitFactor.String(),
		MaxDrawdownPct: maxDrawdownFloat,
		Timestamp:      time.Now(),
	}

	if err := se.kafka.Publish(ctx, "strategy.warning", userID.String(), event); err != nil {
		se.Log().Error("Failed to publish strategy warning event", "error", err)
	}
}

