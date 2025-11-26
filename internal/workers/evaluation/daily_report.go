package evaluation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/user"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// DailyReport generates daily performance reports for all users
// Runs once per day at midnight UTC
type DailyReport struct {
	*workers.BaseWorker
	userRepo    user.Repository
	posRepo     position.Repository
	journalRepo journal.Repository
	kafka       *kafka.Producer
}

// NewDailyReport creates a new daily report worker
func NewDailyReport(
	userRepo user.Repository,
	posRepo position.Repository,
	journalRepo journal.Repository,
	kafka *kafka.Producer,
	interval time.Duration,
	enabled bool,
) *DailyReport {
	return &DailyReport{
		BaseWorker:  workers.NewBaseWorker("daily_report", interval, enabled),
		userRepo:    userRepo,
		posRepo:     posRepo,
		journalRepo: journalRepo,
		kafka:       kafka,
	}
}

// Run executes one iteration of daily report generation
func (dr *DailyReport) Run(ctx context.Context) error {
	dr.Log().Info("Daily report: starting generation")

	// Get all active users (using large limit)
	users, err := dr.userRepo.List(ctx, 1000, 0)
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
		dr.Log().Debug("No active users for daily report")
		return nil
	}

	// Generate report for each user
	for _, usr := range users {
		if err := dr.generateUserReport(ctx, usr); err != nil {
			dr.Log().Error("Failed to generate user report",
				"user_id", usr.ID,
				"error", err,
			)
			// Continue with other users
			continue
		}
	}

	dr.Log().Info("Daily report: generation complete", "users_processed", len(users))

	return nil
}

// generateUserReport generates a daily report for a single user
func (dr *DailyReport) generateUserReport(ctx context.Context, usr *user.User) error {
	dr.Log().Debug("Generating daily report", "user_id", usr.ID)

	// Calculate date range (yesterday 00:00 to today 00:00 UTC)
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Get positions closed yesterday
	closedPositions, err := dr.posRepo.GetClosedInRange(ctx, usr.ID, startOfYesterday, startOfToday)
	if err != nil {
		return errors.Wrap(err, "failed to get closed positions")
	}

	// Calculate daily statistics
	stats := dr.calculateDailyStats(closedPositions)

	// Get journal entries from yesterday
	entries, err := dr.journalRepo.GetByUserAndDateRange(ctx, usr.ID, startOfYesterday, startOfToday)
	if err != nil {
		return errors.Wrap(err, "failed to get journal entries")
	}

	// Get strategy performance (for last 30 days)
	thirtyDaysAgo := startOfToday.AddDate(0, 0, -30)
	strategyStats, err := dr.journalRepo.GetStrategyStats(ctx, usr.ID, thirtyDaysAgo)
	if err != nil {
		return errors.Wrap(err, "failed to get strategy stats")
	}

	// Convert strategy stats slice to map
	strategyStatsMap := make(map[string]journal.StrategyStats)
	for _, stat := range strategyStats {
		strategyStatsMap[stat.StrategyName] = stat
	}

	// Build report
	report := DailyReportData{
		UserID:        usr.ID,
		Date:          startOfYesterday,
		TotalTrades:   stats.TotalTrades,
		WinningTrades: stats.WinningTrades,
		LosingTrades:  stats.LosingTrades,
		WinRate:       stats.WinRate,
		TotalPnL:      stats.TotalPnL,
		TotalPnLPct:   stats.TotalPnLPct,
		BestTrade:     stats.BestTrade,
		WorstTrade:    stats.WorstTrade,
		AvgWin:        stats.AvgWin,
		AvgLoss:       stats.AvgLoss,
		ProfitFactor:  stats.ProfitFactor,
		JournalCount:  len(entries),
		StrategyStats: strategyStatsMap,
	}

	dr.Log().Info("Daily report generated",
		"user_id", usr.ID,
		"total_trades", report.TotalTrades,
		"win_rate", report.WinRate,
		"total_pnl", report.TotalPnL,
	)

	// Publish report event (for Telegram notification)
	dr.publishReportEvent(ctx, report)

	return nil
}

// calculateDailyStats calculates daily trading statistics
func (dr *DailyReport) calculateDailyStats(positions []*position.Position) DailyStats {
	stats := DailyStats{}

	if len(positions) == 0 {
		return stats
	}

	stats.TotalTrades = len(positions)

	var totalWins decimal.Decimal
	var totalLosses decimal.Decimal
	var wins []decimal.Decimal
	var losses []decimal.Decimal

	for _, pos := range positions {
		if pos.RealizedPnL.IsPositive() {
			stats.WinningTrades++
			totalWins = totalWins.Add(pos.RealizedPnL)
			wins = append(wins, pos.RealizedPnL)

			if stats.BestTrade.IsZero() || pos.RealizedPnL.GreaterThan(stats.BestTrade) {
				stats.BestTrade = pos.RealizedPnL
			}
		} else if pos.RealizedPnL.IsNegative() {
			stats.LosingTrades++
			totalLosses = totalLosses.Add(pos.RealizedPnL.Abs())
			losses = append(losses, pos.RealizedPnL)

			if stats.WorstTrade.IsZero() || pos.RealizedPnL.LessThan(stats.WorstTrade) {
				stats.WorstTrade = pos.RealizedPnL
			}
		}

		stats.TotalPnL = stats.TotalPnL.Add(pos.RealizedPnL)
		stats.TotalPnLPct = stats.TotalPnLPct.Add(pos.UnrealizedPnLPct) // Use unrealized as approximation
	}

	// Calculate win rate
	if stats.TotalTrades > 0 {
		stats.WinRate = (float64(stats.WinningTrades) / float64(stats.TotalTrades)) * 100
	}

	// Calculate average win/loss
	if len(wins) > 0 {
		stats.AvgWin = totalWins.Div(decimal.NewFromInt(int64(len(wins))))
	}
	if len(losses) > 0 {
		stats.AvgLoss = totalLosses.Div(decimal.NewFromInt(int64(len(losses))))
	}

	// Calculate profit factor
	if !totalLosses.IsZero() {
		stats.ProfitFactor = totalWins.Div(totalLosses)
	}

	return stats
}

// Data structures

type DailyStats struct {
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalPnL      decimal.Decimal
	TotalPnLPct   decimal.Decimal
	BestTrade     decimal.Decimal
	WorstTrade    decimal.Decimal
	AvgWin        decimal.Decimal
	AvgLoss       decimal.Decimal
	ProfitFactor  decimal.Decimal
}

type DailyReportData struct {
	UserID        uuid.UUID
	Date          time.Time
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalPnL      decimal.Decimal
	TotalPnLPct   decimal.Decimal
	BestTrade     decimal.Decimal
	WorstTrade    decimal.Decimal
	AvgWin        decimal.Decimal
	AvgLoss       decimal.Decimal
	ProfitFactor  decimal.Decimal
	JournalCount  int
	StrategyStats map[string]journal.StrategyStats
}

// Event structure

type DailyReportEvent struct {
	UserID        string  `json:"user_id"`
	Date          string  `json:"date"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      string  `json:"total_pnl"`
	TotalPnLPct   string  `json:"total_pnl_pct"`
	BestTrade     string  `json:"best_trade"`
	WorstTrade    string  `json:"worst_trade"`
	AvgWin        string  `json:"avg_win"`
	AvgLoss       string  `json:"avg_loss"`
	ProfitFactor  string  `json:"profit_factor"`
	Summary       string  `json:"summary"`
	Timestamp     time.Time `json:"timestamp"`
}

func (dr *DailyReport) publishReportEvent(ctx context.Context, report DailyReportData) {
	// Generate summary text
	summary := fmt.Sprintf(
		"Daily Report for %s\n"+
			"Trades: %d (W: %d, L: %d)\n"+
			"Win Rate: %.1f%%\n"+
			"Total PnL: %s (%.2f%%)\n"+
			"Profit Factor: %s",
		report.Date.Format("2006-01-02"),
		report.TotalTrades,
		report.WinningTrades,
		report.LosingTrades,
		report.WinRate,
		report.TotalPnL.StringFixed(2),
		report.TotalPnLPct.InexactFloat64(),
		report.ProfitFactor.StringFixed(2),
	)

	event := DailyReportEvent{
		UserID:        report.UserID.String(),
		Date:          report.Date.Format("2006-01-02"),
		TotalTrades:   report.TotalTrades,
		WinningTrades: report.WinningTrades,
		LosingTrades:  report.LosingTrades,
		WinRate:       report.WinRate,
		TotalPnL:      report.TotalPnL.String(),
		TotalPnLPct:   report.TotalPnLPct.String(),
		BestTrade:     report.BestTrade.String(),
		WorstTrade:    report.WorstTrade.String(),
		AvgWin:        report.AvgWin.String(),
		AvgLoss:       report.AvgLoss.String(),
		ProfitFactor:  report.ProfitFactor.String(),
		Summary:       summary,
		Timestamp:     time.Now(),
	}

	if err := dr.kafka.Publish(ctx, "report.daily", report.UserID.String(), event); err != nil {
		dr.Log().Error("Failed to publish daily report event", "error", err)
	}
}

