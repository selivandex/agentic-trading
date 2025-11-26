package smc

import (
	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// MarketStructure represents the current market structure
type MarketStructure struct {
	Trend       string          `json:"trend"`        // uptrend, downtrend, ranging
	HigherHighs []float64       `json:"higher_highs"` // HH in uptrend
	HigherLows  []float64       `json:"higher_lows"`  // HL in uptrend
	LowerHighs  []float64       `json:"lower_highs"`  // LH in downtrend
	LowerLows   []float64       `json:"lower_lows"`   // LL in downtrend
	LastBOS     *StructureBreak `json:"last_bos"`     // Break of Structure
	LastCHoCH   *StructureBreak `json:"last_choch"`   // Change of Character
	CurrentHigh float64         `json:"current_high"`
	CurrentLow  float64         `json:"current_low"`
}

// StructureBreak represents a BOS or CHoCH event
type StructureBreak struct {
	Type      string  `json:"type"`      // BOS, CHoCH
	Direction string  `json:"direction"` // bullish, bearish
	Price     float64 `json:"price"`     // Price level broken
	Timestamp int64   `json:"timestamp"`
	Index     int     `json:"index"`
}

// NewGetMarketStructureTool analyzes market structure (BOS/CHoCH)
// BOS (Break of Structure) = Continuation of trend
// CHoCH (Change of Character) = Potential trend reversal
func NewGetMarketStructureTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "get_market_structure",
			Description: "Get Market Structure (BOS/CHoCH)",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			candles, err := loadCandles(ctx, deps, args, 100)
			if err != nil {
				return nil, err
			}

			if len(candles) < 20 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 20 candles")
			}

			lookback := parseLimit(args["lookback"], 5)

			// Find swing points
			swingHighs := findSwingHighsWithIndex(candles, lookback)
			swingLows := findSwingLowsWithIndex(candles, lookback)

			// Determine trend by analyzing swing point progression
			trend, hh, hl, lh, ll := analyzeSwingProgression(swingHighs, swingLows)

			// Detect BOS and CHoCH
			var lastBOS, lastCHoCH *StructureBreak

			// BOS: price breaks previous high (in uptrend) or previous low (in downtrend)
			// CHoCH: price fails to make HH/HL (uptrend) or LH/LL (downtrend)

			if trend == "uptrend" && len(hh) >= 2 {
				// Last BOS in uptrend = breaking previous high
				lastBOS = &StructureBreak{
					Type:      "BOS",
					Direction: "bullish",
					Price:     hh[len(hh)-2],
					Index:     0, // Simplified
				}
			} else if trend == "downtrend" && len(ll) >= 2 {
				// Last BOS in downtrend = breaking previous low
				lastBOS = &StructureBreak{
					Type:      "BOS",
					Direction: "bearish",
					Price:     ll[len(ll)-2],
					Index:     0,
				}
			}

			structure := MarketStructure{
				Trend:       trend,
				HigherHighs: hh,
				HigherLows:  hl,
				LowerHighs:  lh,
				LowerLows:   ll,
				LastBOS:     lastBOS,
				LastCHoCH:   lastCHoCH,
				CurrentHigh: candles[0].High,
				CurrentLow:  candles[0].Low,
			}

			signal := "neutral"
			if trend == "uptrend" && lastBOS != nil {
				signal = "bullish_bos"
			} else if trend == "downtrend" && lastBOS != nil {
				signal = "bearish_bos"
			}

			// CHoCH detection: trend changed from up to down or vice versa
			if trend == "ranging" && (len(hh) > 0 || len(ll) > 0) {
				// Possible trend change
				signal = "potential_trend_change"
			}

			return map[string]interface{}{
				"structure": structure,
				"signal":    signal,
				"trend":     trend,
			}, nil
		})
	return t
}

type swingPoint struct {
	price float64
	index int
}

func findSwingHighsWithIndex(candles []market_data.OHLCV, lookback int) []swingPoint {
	points := make([]swingPoint, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		candle := candles[i]
		isSwing := true

		for j := 1; j <= lookback; j++ {
			if candles[i+j].High >= candle.High || candles[i-j].High >= candle.High {
				isSwing = false
				break
			}
		}

		if isSwing {
			points = append(points, swingPoint{price: candle.High, index: i})
		}
	}

	return points
}

func findSwingLowsWithIndex(candles []market_data.OHLCV, lookback int) []swingPoint {
	points := make([]swingPoint, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		candle := candles[i]
		isSwing := true

		for j := 1; j <= lookback; j++ {
			if candles[i+j].Low <= candle.Low || candles[i-j].Low <= candle.Low {
				isSwing = false
				break
			}
		}

		if isSwing {
			points = append(points, swingPoint{price: candle.Low, index: i})
		}
	}

	return points
}

func analyzeSwingProgression(highs, lows []swingPoint) (trend string, hh, hl, lh, ll []float64) {
	hh = make([]float64, 0)
	hl = make([]float64, 0)
	lh = make([]float64, 0)
	ll = make([]float64, 0)

	// Analyze highs
	for i := 1; i < len(highs); i++ {
		if highs[i].price > highs[i-1].price {
			hh = append(hh, highs[i].price) // Higher high
		} else {
			lh = append(lh, highs[i].price) // Lower high
		}
	}

	// Analyze lows
	for i := 1; i < len(lows); i++ {
		if lows[i].price > lows[i-1].price {
			hl = append(hl, lows[i].price) // Higher low
		} else {
			ll = append(ll, lows[i].price) // Lower low
		}
	}

	// Determine trend
	if len(hh) >= 2 && len(hl) >= 2 {
		trend = "uptrend" // Making HH and HL
	} else if len(lh) >= 2 && len(ll) >= 2 {
		trend = "downtrend" // Making LH and LL
	} else {
		trend = "ranging"
	}

	return trend, hh, hl, lh, ll
}
