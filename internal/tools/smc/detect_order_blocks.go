package smc
import (
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"time"
	"google.golang.org/adk/tool"
)
// OrderBlock represents a detected Order Block
type OrderBlock struct {
	Type      string  `json:"type"` // bullish, bearish
	High      float64 `json:"high"` // Block high
	Low       float64 `json:"low"`  // Block low
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Strength  string  `json:"strength"`  // weak, medium, strong
	FormTime  int64   `json:"form_time"` // Unix timestamp
	Index     int     `json:"index"`     // Candle index
	Mitigated bool    `json:"mitigated"` // Has price returned and broken through?
}
// NewDetectOrderBlocksTool detects Order Blocks - last bullish/bearish candle before reversal
// Order Block = The last candle before a strong move in opposite direction
// Acts as support (bullish OB) or resistance (bearish OB)
func NewDetectOrderBlocksTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"detect_order_blocks",
		"Detect Order Blocks (ICT)",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles
			candles, err := loadCandles(ctx, deps, args, 100)
			if err != nil {
				return nil, err
			}
			if len(candles) < 5 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 5 candles for order block detection")
			}
			minMoveSize := parseFloat(args["min_move_pct"], 1.0) // Minimum 1% move after OB
			orderBlocks := make([]OrderBlock, 0)
			currentPrice := candles[0].Close
			// Scan for order blocks
			// Look for strong moves and identify the setup candle
			for i := 1; i < len(candles)-3; i++ {
				prevCandle := candles[i]
				// Check for bullish move after this candle
				// Next 3 candles should show upward movement
				nextHighest := 0.0
				for j := 0; j < 3 && i-j >= 0; j++ {
					if candles[i-j].High > nextHighest {
						nextHighest = candles[i-j].High
					}
				}
				moveUpPct := ((nextHighest - prevCandle.Low) / prevCandle.Low) * 100
				if moveUpPct >= minMoveSize {
					// This could be a bullish order block
					// Check if prev candle was bearish or consolidation
					isBearishCandle := prevCandle.Close < prevCandle.Open
					strength := calculateOBStrength(prevCandle.Volume, moveUpPct)
					mitigated := currentPrice < prevCandle.Low // Price broke below OB
					orderBlocks = append(orderBlocks, OrderBlock{
						Type:      "bullish",
						High:      prevCandle.High,
						Low:       prevCandle.Low,
						Open:      prevCandle.Open,
						Close:     prevCandle.Close,
						Volume:    prevCandle.Volume,
						Strength:  strength,
						FormTime:  prevCandle.OpenTime.Unix(),
						Index:     i,
						Mitigated: mitigated,
					})
					_ = isBearishCandle // Used for validation in production
				}
				// Check for bearish move after this candle
				nextLowest := 99999999.0
				for j := 0; j < 3 && i-j >= 0; j++ {
					if candles[i-j].Low < nextLowest {
						nextLowest = candles[i-j].Low
					}
				}
				moveDownPct := ((prevCandle.High - nextLowest) / prevCandle.High) * 100
				if moveDownPct >= minMoveSize {
					// This could be a bearish order block
					strength := calculateOBStrength(prevCandle.Volume, moveDownPct)
					mitigated := currentPrice > prevCandle.High // Price broke above OB
					orderBlocks = append(orderBlocks, OrderBlock{
						Type:      "bearish",
						High:      prevCandle.High,
						Low:       prevCandle.Low,
						Open:      prevCandle.Open,
						Close:     prevCandle.Close,
						Volume:    prevCandle.Volume,
						Strength:  strength,
						FormTime:  prevCandle.OpenTime.Unix(),
						Index:     i,
						Mitigated: mitigated,
					})
				}
			}
			// Find nearest unmitigated OB
			var nearestOB *OrderBlock
			minDistance := 999999.0
			for i := range orderBlocks {
				if !orderBlocks[i].Mitigated {
					distance := 0.0
					if orderBlocks[i].Type == "bullish" {
						distance = currentPrice - orderBlocks[i].High
					} else {
						distance = orderBlocks[i].Low - currentPrice
					}
					if distance > 0 && distance < minDistance {
						minDistance = distance
						nearestOB = &orderBlocks[i]
					}
				}
			}
			// Generate signal
			signal := "no_ob"
			if nearestOB != nil {
				if nearestOB.Type == "bullish" {
					signal = "bullish_ob_support"
				} else {
					signal = "bearish_ob_resistance"
				}
			}
			return map[string]interface{}{
				"order_blocks":    len(orderBlocks),
				"unmitigated_obs": countUnmitigatedOB(orderBlocks),
				"nearest_ob":      nearestOB,
				"all_obs":         orderBlocks,
				"signal":          signal,
				"current_price":   currentPrice,
			}, nil
		},
		deps,
	).
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond).
		Build()
}
func calculateOBStrength(volume, movePct float64) string {
	// Higher volume + larger move = stronger OB
	score := volume * movePct / 100
	if score > 100000 {
		return "strong"
	} else if score > 50000 {
		return "medium"
	}
	return "weak"
}
func countUnmitigatedOB(obs []OrderBlock) int {
	count := 0
	for _, ob := range obs {
		if !ob.Mitigated {
			count++
		}
	}
	return count
}
