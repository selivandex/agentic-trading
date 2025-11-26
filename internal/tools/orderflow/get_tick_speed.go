package orderflow

import (
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetTickSpeedTool calculates trade velocity (trades per minute)
// High tick speed = high activity, potential breakout
func NewGetTickSpeedTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "get_tick_speed",
			Description: "Get Tick Speed (Trade Velocity)",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			if !deps.HasMarketData() {
				return nil, errors.Wrapf(errors.ErrInternal, "market data repository not configured")
			}

			exchange, _ := args["exchange"].(string)
			symbol, _ := args["symbol"].(string)
			limit := parseLimit(args["limit"], 200)

			if exchange == "" || symbol == "" {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange and symbol required")
			}

			// Get recent trades
			trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, limit)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get recent trades")
			}

			if len(trades) < 2 {
				return nil, errors.Wrapf(errors.ErrNotFound, "not enough trades")
			}

			// Calculate time span
			oldest := trades[len(trades)-1].Timestamp
			newest := trades[0].Timestamp
			timeSpan := newest.Sub(oldest)

			if timeSpan.Seconds() == 0 {
				return nil, errors.Wrapf(errors.ErrInternal, "zero time span")
			}

			// Calculate tick speed (trades per minute)
			tradesPerMinute := float64(len(trades)) / timeSpan.Minutes()

			// Calculate volume velocity (USD volume per minute)
			totalVolumeUSD := 0.0
			for _, trade := range trades {
				totalVolumeUSD += trade.Price * trade.Quantity
			}

			volumePerMinute := totalVolumeUSD / timeSpan.Minutes()

			// Analyze acceleration (recent vs older trades)
			midPoint := len(trades) / 2
			recentTrades := trades[:midPoint]
			olderTrades := trades[midPoint:]

			recentTime := recentTrades[0].Timestamp.Sub(recentTrades[len(recentTrades)-1].Timestamp)
			olderTime := olderTrades[0].Timestamp.Sub(olderTrades[len(olderTrades)-1].Timestamp)

			recentSpeed := 0.0
			olderSpeed := 0.0

			if recentTime.Minutes() > 0 {
				recentSpeed = float64(len(recentTrades)) / recentTime.Minutes()
			}

			if olderTime.Minutes() > 0 {
				olderSpeed = float64(len(olderTrades)) / olderTime.Minutes()
			}

			acceleration := "stable"
			accelerationPct := 0.0

			if olderSpeed > 0 {
				accelerationPct = ((recentSpeed - olderSpeed) / olderSpeed) * 100

				if accelerationPct > 50 {
					acceleration = "increasing_fast"
				} else if accelerationPct > 20 {
					acceleration = "increasing"
				} else if accelerationPct < -50 {
					acceleration = "decreasing_fast"
				} else if accelerationPct < -20 {
					acceleration = "decreasing"
				}
			}

			// Determine activity level
			activity := "low"
			if tradesPerMinute > 100 {
				activity = "extreme"
			} else if tradesPerMinute > 50 {
				activity = "high"
			} else if tradesPerMinute > 20 {
				activity = "moderate"
			}

			// Signal
			signal := "neutral"
			if acceleration == "increasing_fast" && activity == "high" {
				signal = "breakout_likely" // High velocity + acceleration
			} else if acceleration == "decreasing_fast" {
				signal = "exhaustion" // Slowing down
			}

			return map[string]interface{}{
				"trades_per_minute": tradesPerMinute,
				"volume_per_minute": volumePerMinute,
				"acceleration":      acceleration,
				"acceleration_pct":  accelerationPct,
				"activity_level":    activity,
				"signal":            signal,
				"time_span_minutes": timeSpan.Minutes(),
				"trades_analyzed":   len(trades),
			}, nil
		})
	return t
}
