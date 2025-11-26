package smc

import (
	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// LiquidityZone represents a clustered area of swing points
type LiquidityZone struct {
	Type        string  `json:"type"`         // buy_side, sell_side
	PriceHigh   float64 `json:"price_high"`   // Upper boundary
	PriceLow    float64 `json:"price_low"`    // Lower boundary
	PriceCenter float64 `json:"price_center"` // Middle of zone
	Density     int     `json:"density"`      // Number of swing points in zone
	Strength    string  `json:"strength"`     // weak, medium, strong
	Distance    float64 `json:"distance"`     // Distance from current price
}

// NewDetectLiquidityZonesTool detects liquidity zones - areas where stop losses cluster
// Buy-side liquidity = above swing highs (where shorts have stops)
// Sell-side liquidity = below swing lows (where longs have stops)
func NewDetectLiquidityZonesTool(deps shared.Deps) tool.Tool {
	t, _ := functiontool.New(
		functiontool.Config{
			Name:        "detect_liquidity_zones",
			Description: "Detect Liquidity Zones",
		},
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 200)
			if err != nil {
				return nil, err
			}

			if len(candles) < 20 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 20 candles")
			}

			lookback := parseLimit(args["lookback"], 5)
			clusterThreshold := parseFloat(args["cluster_threshold_pct"], 0.5) // 0.5% price range for clustering

			// Find swing points first
			swingHighs := findSwingHighs(candles, lookback)
			swingLows := findSwingLows(candles, lookback)

			// Cluster swing points into liquidity zones
			buySideLiquidity := clusterSwingPoints(swingHighs, clusterThreshold)
			sellSideLiquidity := clusterSwingPoints(swingLows, clusterThreshold)

			currentPrice := candles[0].Close

			// Add type and distance to zones
			for i := range buySideLiquidity {
				buySideLiquidity[i].Type = "buy_side"
				buySideLiquidity[i].Distance = buySideLiquidity[i].PriceCenter - currentPrice
			}

			for i := range sellSideLiquidity {
				sellSideLiquidity[i].Type = "sell_side"
				sellSideLiquidity[i].Distance = currentPrice - sellSideLiquidity[i].PriceCenter
			}

			// Find nearest zones
			var nearestBuySide, nearestSellSide *LiquidityZone

			for i := range buySideLiquidity {
				if buySideLiquidity[i].Distance > 0 {
					if nearestBuySide == nil || buySideLiquidity[i].Distance < nearestBuySide.Distance {
						nearestBuySide = &buySideLiquidity[i]
					}
				}
			}

			for i := range sellSideLiquidity {
				if sellSideLiquidity[i].Distance > 0 {
					if nearestSellSide == nil || sellSideLiquidity[i].Distance < nearestSellSide.Distance {
						nearestSellSide = &sellSideLiquidity[i]
					}
				}
			}

			// Generate signal - smart money often targets liquidity
			signal := "no_nearby_liquidity"
			targetZone := "none"

			if nearestSellSide != nil && nearestSellSide.Distance/currentPrice*100 < 2 {
				signal = "sell_side_liquidity_near"
				targetZone = "below"
			} else if nearestBuySide != nil && nearestBuySide.Distance/currentPrice*100 < 2 {
				signal = "buy_side_liquidity_near"
				targetZone = "above"
			}

			return map[string]interface{}{
				"buy_side_zones":    buySideLiquidity,
				"sell_side_zones":   sellSideLiquidity,
				"nearest_buy_side":  nearestBuySide,
				"nearest_sell_side": nearestSellSide,
				"signal":            signal,
				"target_zone":       targetZone,
				"current_price":     currentPrice,
			}, nil
		})
	return t
}

// findSwingHighs finds all swing highs in candles
func findSwingHighs(candles []market_data.OHLCV, lookback int) []float64 {
	highs := make([]float64, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		candle := candles[i]
		isSwingHigh := true

		for j := 1; j <= lookback; j++ {
			if candles[i+j].High >= candle.High || candles[i-j].High >= candle.High {
				isSwingHigh = false
				break
			}
		}

		if isSwingHigh {
			highs = append(highs, candle.High)
		}
	}

	return highs
}

// findSwingLows finds all swing lows in candles
func findSwingLows(candles []market_data.OHLCV, lookback int) []float64 {
	lows := make([]float64, 0)

	for i := lookback; i < len(candles)-lookback; i++ {
		candle := candles[i]
		isSwingLow := true

		for j := 1; j <= lookback; j++ {
			if candles[i+j].Low <= candle.Low || candles[i-j].Low <= candle.Low {
				isSwingLow = false
				break
			}
		}

		if isSwingLow {
			lows = append(lows, candle.Low)
		}
	}

	return lows
}

// clusterSwingPoints clusters swing points that are close together
func clusterSwingPoints(points []float64, thresholdPct float64) []LiquidityZone {
	if len(points) == 0 {
		return nil
	}

	zones := make([]LiquidityZone, 0)
	used := make([]bool, len(points))

	for i, point := range points {
		if used[i] {
			continue
		}

		// Start new cluster
		cluster := []float64{point}
		used[i] = true

		// Find nearby points
		for j, other := range points {
			if used[j] || i == j {
				continue
			}

			// Check if within threshold
			diff := absFloat(point - other)
			pct := (diff / point) * 100

			if pct <= thresholdPct {
				cluster = append(cluster, other)
				used[j] = true
			}
		}

		// Create zone from cluster
		if len(cluster) > 0 {
			high, low := minMaxFloat(cluster)
			center := (high + low) / 2

			strength := "weak"
			if len(cluster) >= 5 {
				strength = "strong"
			} else if len(cluster) >= 3 {
				strength = "medium"
			}

			zones = append(zones, LiquidityZone{
				PriceHigh:   high,
				PriceLow:    low,
				PriceCenter: center,
				Density:     len(cluster),
				Strength:    strength,
			})
		}
	}

	return zones
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func minMaxFloat(vals []float64) (max, min float64) {
	if len(vals) == 0 {
		return 0, 0
	}

	min = vals[0]
	max = vals[0]

	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return max, min
}
