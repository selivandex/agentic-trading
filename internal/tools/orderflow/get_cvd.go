package orderflow

import (
	"context"

	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// NewGetCVDTool calculates Cumulative Volume Delta
// CVD = running total of (buy volume - sell volume)
// Rising CVD = accumulation, Falling CVD = distribution
func NewGetCVDTool(deps shared.Deps) tool.Tool {
	return functiontool.New("get_cvd", "Get Cumulative Volume Delta (CVD)", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		if !deps.HasMarketData() {
			return nil, errors.Wrapf(errors.ErrInternal, "market data repository not configured")
		}

		exchange, _ := args["exchange"].(string)
		symbol, _ := args["symbol"].(string)
		limit := parseLimit(args["limit"], 500) // More trades for CVD

		if exchange == "" || symbol == "" {
			return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange and symbol required")
		}

		// Get recent trades
		trades, err := deps.MarketDataRepo.GetRecentTrades(ctx, exchange, symbol, limit)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get recent trades")
		}

		if len(trades) == 0 {
			return nil, errors.Wrapf(errors.ErrNotFound, "no trades found")
		}

		// Calculate CVD (cumulative)
		cvd := 0.0
		cvdSeries := make([]float64, 0, len(trades))

		// Process trades in chronological order (reverse our DESC order)
		for i := len(trades) - 1; i >= 0; i-- {
			trade := trades[i]
			
			if trade.Side == "buy" {
				cvd += trade.Quantity
			} else {
				cvd -= trade.Quantity
			}

			cvdSeries = append(cvdSeries, cvd)
		}

		// Analyze CVD trend
		cvdStart := cvdSeries[0]
		cvdEnd := cvd
		cvdChange := cvdEnd - cvdStart
		cvdChangePct := 0.0
		if cvdStart != 0 {
			cvdChangePct = (cvdChange / absFloat(cvdStart)) * 100
		}

		// Determine CVD trend
		trend := "accumulation"
		if cvdChange < 0 {
			trend = "distribution"
		} else if absFloat(cvdChange) < absFloat(cvdStart)*0.1 {
			trend = "neutral"
		}

		// Price-CVD divergence analysis
		priceChange := trades[0].Price - trades[len(trades)-1].Price

		divergence := "none"
		if priceChange > 0 && cvdChange < 0 {
			divergence = "bearish" // Price up, CVD down = weak rally
		} else if priceChange < 0 && cvdChange > 0 {
			divergence = "bullish" // Price down, CVD up = weak dump
		}

		// Signal
		signal := "neutral"
		if trend == "accumulation" && divergence == "none" {
			signal = "bullish"
		} else if trend == "distribution" && divergence == "none" {
			signal = "bearish"
		} else if divergence == "bullish" {
			signal = "bullish_reversal"
		} else if divergence == "bearish" {
			signal = "bearish_reversal"
		}

		return map[string]interface{}{
			"cvd":            cvd,
			"cvd_change":     cvdChange,
			"cvd_change_pct": cvdChangePct,
			"trend":          trend,
			"divergence":     divergence,
			"signal":         signal,
			"trades_analyzed": len(trades),
		}, nil
	})
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

