package risk

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"prometheus/internal/domain/position"
	"prometheus/pkg/logger"
)

// TradingStats contains user trading statistics for Kelly Criterion
type TradingStats struct {
	WinRate    decimal.Decimal // 0-1
	AvgWin     decimal.Decimal // average winning trade % return
	AvgLoss    decimal.Decimal // average losing trade % return (positive)
	SampleSize int             // number of closed trades
	Expectancy decimal.Decimal // (WinRate * AvgWin) - ((1-WinRate) * AvgLoss)
}

// StatsCalculator calculates trading statistics from closed positions
type StatsCalculator struct {
	posRepo position.Repository
	log     *logger.Logger
}

// NewStatsCalculator creates a new stats calculator
func NewStatsCalculator(posRepo position.Repository, log *logger.Logger) *StatsCalculator {
	return &StatsCalculator{
		posRepo: posRepo,
		log:     log,
	}
}

// CalculateStats calculates trading statistics for a user
// Minimum 20 trades required for reliable Kelly Criterion
func (c *StatsCalculator) CalculateStats(ctx context.Context, userID uuid.UUID) (*TradingStats, error) {
	// Get all closed positions for user
	// TODO: Add method to position.Repository to get closed positions
	// For now, return empty stats
	stats := &TradingStats{
		WinRate:    decimal.Zero,
		AvgWin:     decimal.Zero,
		AvgLoss:    decimal.Zero,
		SampleSize: 0,
		Expectancy: decimal.Zero,
	}

	// TODO: Implement actual calculation:
	// 1. Get all closed positions for user
	// 2. Calculate win/loss counts
	// 3. Calculate average % return for wins and losses
	// 4. Calculate win rate
	// 5. Calculate expectancy

	return stats, nil
}

// CalculateStatsFromPositions calculates stats from a list of closed positions
func (c *StatsCalculator) CalculateStatsFromPositions(positions []position.Position) *TradingStats {
	if len(positions) == 0 {
		return &TradingStats{}
	}

	wins := 0
	losses := 0
	totalWinPct := decimal.Zero
	totalLossPct := decimal.Zero

	for _, pos := range positions {
		// Skip open positions
		if pos.Status == position.PositionOpen {
			continue
		}

		// Calculate % return
		// Use CurrentPrice as exit price for closed positions
		// Return % = (ExitPrice - EntryPrice) / EntryPrice * 100
		exitPrice := pos.CurrentPrice

		var returnPct decimal.Decimal
		if pos.Side == position.PositionLong {
			returnPct = exitPrice.Sub(pos.EntryPrice).Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
		} else {
			// Short position: inverted
			returnPct = pos.EntryPrice.Sub(exitPrice).Div(pos.EntryPrice).Mul(decimal.NewFromInt(100))
		}

		// Classify as win or loss
		if returnPct.GreaterThan(decimal.Zero) {
			wins++
			totalWinPct = totalWinPct.Add(returnPct)
		} else if returnPct.LessThan(decimal.Zero) {
			losses++
			totalLossPct = totalLossPct.Add(returnPct.Abs()) // store as positive
		}
	}

	totalTrades := wins + losses
	if totalTrades == 0 {
		return &TradingStats{SampleSize: 0}
	}

	// Calculate averages
	avgWin := decimal.Zero
	if wins > 0 {
		avgWin = totalWinPct.Div(decimal.NewFromInt(int64(wins)))
	}

	avgLoss := decimal.Zero
	if losses > 0 {
		avgLoss = totalLossPct.Div(decimal.NewFromInt(int64(losses)))
	}

	// Calculate win rate
	winRate := decimal.NewFromInt(int64(wins)).Div(decimal.NewFromInt(int64(totalTrades)))

	// Calculate expectancy: (WinRate * AvgWin) - ((1-WinRate) * AvgLoss)
	expectancy := winRate.Mul(avgWin).Sub(
		decimal.NewFromInt(1).Sub(winRate).Mul(avgLoss),
	)

	return &TradingStats{
		WinRate:    winRate,
		AvgWin:     avgWin,
		AvgLoss:    avgLoss,
		SampleSize: totalTrades,
		Expectancy: expectancy,
	}
}

// HasSufficientData checks if user has enough trades for Kelly Criterion
func (s *TradingStats) HasSufficientData() bool {
	return s.SampleSize >= 20
}

// IsPositiveExpectancy checks if system has positive expectancy
func (s *TradingStats) IsPositiveExpectancy() bool {
	return s.Expectancy.GreaterThan(decimal.Zero)
}
