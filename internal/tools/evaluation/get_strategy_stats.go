package evaluation

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"prometheus/pkg/templates"

	"google.golang.org/adk/tool"
)

// NewGetStrategyStatsTool returns statistics about trading strategies for a user
// MVP version: calculates basic statistics from closed positions
func NewGetStrategyStatsTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_strategy_stats",
		"Get trading statistics for a user. Returns win rate, P&L, profit factor, and other metrics. Useful for evaluating trading performance.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Parse user_id
			userIDStr, ok := args["user_id"].(string)
			if !ok || userIDStr == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "user_id is required")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid user_id format: %v", err)
			}

			// Parse timeframe (optional, default: all)
			timeframe, _ := args["timeframe"].(string)
			if timeframe == "" {
				timeframe = "all"
			}

			// Calculate time range based on timeframe
			now := time.Now()
			var startTime time.Time

			switch timeframe {
			case "7d":
				startTime = now.AddDate(0, 0, -7)
			case "30d":
				startTime = now.AddDate(0, 0, -30)
			case "90d":
				startTime = now.AddDate(0, 0, -90)
			case "all":
				startTime = time.Time{} // Zero time (all history)
			default:
				return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid timeframe: %s (valid: 7d, 30d, 90d, all)", timeframe)
			}

			// Get closed positions
			positions, err := deps.PositionRepo.GetClosedInRange(ctx, userID, startTime, now)
			if err != nil {
				return nil, errors.Wrap(err, "failed to fetch closed positions")
			}

			// Calculate statistics
			stats := calculateStatistics(positions)

			// Prepare template data
			templateData := map[string]interface{}{
				"UserID":        userID.String(),
				"Timeframe":     timeframe,
				"CalculatedAt":  now.Format(time.RFC3339),
				"TotalTrades":   stats["total_trades"],
				"WinningTrades": stats["winning_trades"],
				"LosingTrades":  stats["losing_trades"],
				"WinRate":       stats["win_rate"],
				"AvgWin":        stats["avg_win"],
				"AvgLoss":       stats["avg_loss"],
				"ProfitFactor":  stats["profit_factor"],
				"TotalPnL":      stats["total_pnl"],
				"AvgPnL":        stats["avg_pnl"],
				"MaxWin":        stats["max_win"],
				"MaxLoss":       stats["max_loss"],
				"AvgHoldTime":   stats["avg_hold_time"],
			}

			// Render using template
			tmpl := templates.Get()
			analysis, err := tmpl.Render("tools/strategy_stats", templateData)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render strategy stats template")
			}

			return map[string]interface{}{
				"analysis": analysis,
			}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(2, 1*time.Second).
		Build()
}

// calculateStatistics computes trading statistics from closed positions
func calculateStatistics(positions []*position.Position) map[string]interface{} {
	if len(positions) == 0 {
		return map[string]interface{}{
			"total_trades":   0,
			"winning_trades": 0,
			"losing_trades":  0,
			"win_rate":       0.0,
			"avg_win":        0.0,
			"avg_loss":       0.0,
			"profit_factor":  0.0,
			"total_pnl":      0.0,
			"avg_pnl":        0.0,
			"max_win":        0.0,
			"max_loss":       0.0,
			"avg_hold_time":  "0h 0m",
		}
	}

	var (
		winningTrades int
		losingTrades  int
		totalWins     = decimal.Zero
		totalLosses   = decimal.Zero
		totalPnL      = decimal.Zero
		maxWin        = decimal.Zero
		maxLoss       = decimal.Zero
		totalHoldTime time.Duration
	)

	for _, pos := range positions {
		pnl := pos.RealizedPnL

		// Count wins/losses
		if pnl.GreaterThan(decimal.Zero) {
			winningTrades++
			totalWins = totalWins.Add(pnl)
			if pnl.GreaterThan(maxWin) {
				maxWin = pnl
			}
		} else if pnl.LessThan(decimal.Zero) {
			losingTrades++
			totalLosses = totalLosses.Add(pnl.Abs())
			if pnl.Abs().GreaterThan(maxLoss) {
				maxLoss = pnl.Abs()
			}
		}

		totalPnL = totalPnL.Add(pnl)

		// Calculate hold time
		if pos.ClosedAt != nil {
			holdTime := pos.ClosedAt.Sub(pos.OpenedAt)
			totalHoldTime += holdTime
		}
	}

	totalTrades := len(positions)

	// Win rate
	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winningTrades) / float64(totalTrades)
	}

	// Average win
	avgWin := 0.0
	if winningTrades > 0 {
		avgWin, _ = totalWins.Div(decimal.NewFromInt(int64(winningTrades))).Float64()
	}

	// Average loss
	avgLoss := 0.0
	if losingTrades > 0 {
		avgLoss, _ = totalLosses.Div(decimal.NewFromInt(int64(losingTrades))).Float64()
	}

	// Profit factor
	profitFactor := 0.0
	if totalLosses.GreaterThan(decimal.Zero) {
		profitFactor, _ = totalWins.Div(totalLosses).Float64()
	}

	// Average P&L
	avgPnL, _ := totalPnL.Div(decimal.NewFromInt(int64(totalTrades))).Float64()

	// Average hold time
	avgHoldTime := totalHoldTime / time.Duration(totalTrades)
	avgHoldTimeStr := formatDuration(avgHoldTime)

	// Convert to float64 for JSON
	totalPnLFloat, _ := totalPnL.Float64()
	maxWinFloat, _ := maxWin.Float64()
	maxLossFloat, _ := maxLoss.Float64()

	return map[string]interface{}{
		"total_trades":   totalTrades,
		"winning_trades": winningTrades,
		"losing_trades":  losingTrades,
		"win_rate":       winRate,
		"avg_win":        avgWin,
		"avg_loss":       avgLoss,
		"profit_factor":  profitFactor,
		"total_pnl":      totalPnLFloat,
		"avg_pnl":        avgPnL,
		"max_win":        maxWinFloat,
		"max_loss":       maxLossFloat,
		"avg_hold_time":  avgHoldTimeStr,
	}
}

// formatDuration formats duration as "Xh Ym"
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
