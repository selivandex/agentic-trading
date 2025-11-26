package smc

import (
	"context"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// Imbalance represents a price imbalance (similar to FVG but stricter)
type Imbalance struct {
	Type     string  `json:"type"` // bullish, bearish
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Size     float64 `json:"size"`
	SizePct  float64 `json:"size_pct"`
	FormTime int64   `json:"form_time"`
	Index    int     `json:"index"`
	Filled   bool    `json:"filled"`
}

// NewDetectImbalancesTool detects price imbalances
// Imbalance = Large candle with minimal overlap to previous candles
// Indicates strong directional move with inefficient price discovery
func NewDetectImbalancesTool(deps shared.Deps) tool.Tool {
	return functiontool.New("detect_imbalances", "Detect Price Imbalances", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		candles, err := loadCandles(ctx, deps, args, 100)
		if err != nil {
			return nil, err
		}

		if len(candles) < 3 {
			return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 3 candles")
		}

		minSizePct := parseFloat(args["min_size_pct"], 0.3)        // Min 0.3%
		maxOverlapPct := parseFloat(args["max_overlap_pct"], 25.0) // Max 25% overlap

		imbalances := make([]Imbalance, 0)
		currentPrice := candles[0].Close

		for i := 0; i < len(candles)-1; i++ {
			curr := candles[i]
			prev := candles[i+1]

			currBody := absFloat(curr.Close - curr.Open)
			currRange := curr.High - curr.Low

			// Check for large candle (strong move)
			if currRange == 0 {
				continue
			}

			bodyPct := (currBody / currRange) * 100

			// Strong candle = body > 70% of range
			if bodyPct < 70 {
				continue
			}

			// Check overlap with previous candle
			overlap := calculateOverlap(curr, prev)
			overlapPct := (overlap / currRange) * 100

			if overlapPct > maxOverlapPct {
				continue // Too much overlap
			}

			// Calculate gap size
			gapSize := 0.0
			gapStart := 0.0
			gapEnd := 0.0
			imbType := ""

			// Bullish imbalance (gap up)
			if curr.Low > prev.High {
				gapStart = prev.High
				gapEnd = curr.Low
				gapSize = gapEnd - gapStart
				imbType = "bullish"
			} else if curr.High < prev.Low {
				// Bearish imbalance (gap down)
				gapStart = curr.High
				gapEnd = prev.Low
				gapSize = gapEnd - gapStart
				imbType = "bearish"
			}

			if gapSize > 0 {
				sizePct := (gapSize / gapStart) * 100

				if sizePct >= minSizePct {
					filled := false
					if imbType == "bullish" {
						filled = currentPrice <= gapStart
					} else {
						filled = currentPrice >= gapEnd
					}

					imbalances = append(imbalances, Imbalance{
						Type:     imbType,
						Start:    gapStart,
						End:      gapEnd,
						Size:     gapSize,
						SizePct:  sizePct,
						FormTime: curr.OpenTime.Unix(),
						Index:    i,
						Filled:   filled,
					})
				}
			}
		}

		signal := "no_imbalance"
		if len(imbalances) > 0 && !imbalances[0].Filled {
			if imbalances[0].Type == "bullish" {
				signal = "bullish_imbalance"
			} else {
				signal = "bearish_imbalance"
			}
		}

		return map[string]interface{}{
			"imbalances":     imbalances,
			"unfilled_count": countUnfilledImb(imbalances),
			"signal":         signal,
			"current_price":  currentPrice,
		}, nil
	})
}

func calculateOverlap(c1, c2 market_data.OHLCV) float64 {
	overlapHigh := min(c1.High, c2.High)
	overlapLow := max(c1.Low, c2.Low)

	if overlapHigh > overlapLow {
		return overlapHigh - overlapLow
	}

	return 0
}

func countUnfilledImb(imbs []Imbalance) int {
	count := 0
	for _, imb := range imbs {
		if !imb.Filled {
			count++
		}
	}
	return count
}
