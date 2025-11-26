package indicators

import (
	"context"
	"sort"

	"prometheus/internal/tools/shared"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// VolumeProfileBin represents a price level with accumulated volume
type VolumeProfileBin struct {
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

// NewVolumeProfileTool computes Volume Profile (Volume at Price levels)
// Shows where most trading activity occurred
func NewVolumeProfileTool(deps shared.Deps) tool.Tool {
	return functiontool.New("volume_profile", "Volume Profile (Volume by Price)", func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
		// Load candles
		candles, err := loadCandles(ctx, deps, args, 100)
		if err != nil {
			return nil, err
		}

		if err := ValidateMinLength(candles, 10, "Volume Profile"); err != nil {
			return nil, err
		}

		// Number of price bins
		bins := parseLimit(args["bins"], 20)

		// Find price range
		minPrice := candles[0].Low
		maxPrice := candles[0].High
		for _, candle := range candles {
			if candle.Low < minPrice {
				minPrice = candle.Low
			}
			if candle.High > maxPrice {
				maxPrice = candle.High
			}
		}

		// Create bins
		priceRange := maxPrice - minPrice
		binSize := priceRange / float64(bins)
		volumeByPrice := make([]float64, bins)

		// Distribute volume across bins
		for _, candle := range candles {
			// Simple approach: distribute candle volume to its price bin
			// In reality, we'd need tick data for precise volume profile
			candlePrice := (candle.High + candle.Low + candle.Close) / 3
			binIndex := int((candlePrice - minPrice) / binSize)
			
			if binIndex < 0 {
				binIndex = 0
			}
			if binIndex >= bins {
				binIndex = bins - 1
			}

			volumeByPrice[binIndex] += candle.Volume
		}

		// Find POC (Point of Control) - price level with highest volume
		maxVolume := 0.0
		pocIndex := 0
		for i, vol := range volumeByPrice {
			if vol > maxVolume {
				maxVolume = vol
				pocIndex = i
			}
		}

		pocPrice := minPrice + (float64(pocIndex)+0.5)*binSize

		// Find Value Area (70% of volume)
		totalVolume := 0.0
		for _, vol := range volumeByPrice {
			totalVolume += vol
		}

		targetVolume := totalVolume * 0.70
		valueAreaVolume := volumeByPrice[pocIndex]
		vaHigh := pocIndex
		vaLow := pocIndex

		// Expand value area from POC until we reach 70% of volume
		for valueAreaVolume < targetVolume && (vaHigh < bins-1 || vaLow > 0) {
			nextHighVol := 0.0
			if vaHigh < bins-1 {
				nextHighVol = volumeByPrice[vaHigh+1]
			}

			nextLowVol := 0.0
			if vaLow > 0 {
				nextLowVol = volumeByPrice[vaLow-1]
			}

			if nextHighVol > nextLowVol {
				vaHigh++
				valueAreaVolume += nextHighVol
			} else if nextLowVol > 0 {
				vaLow--
				valueAreaVolume += nextLowVol
			} else if vaHigh < bins-1 {
				vaHigh++
				valueAreaVolume += nextHighVol
			} else {
				break
			}
		}

		valueAreaHigh := minPrice + (float64(vaHigh)+1)*binSize
		valueAreaLow := minPrice + float64(vaLow)*binSize

		// Build profile data
		profile := make([]VolumeProfileBin, 0, bins)
		for i, vol := range volumeByPrice {
			if vol > 0 {
				price := minPrice + (float64(i)+0.5)*binSize
				profile = append(profile, VolumeProfileBin{
					Price:  price,
					Volume: vol,
				})
			}
		}

		// Sort by volume (descending)
		sort.Slice(profile, func(i, j int) bool {
			return profile[i].Volume > profile[j].Volume
		})

		// Get top 5 high volume nodes
		topNodes := profile
		if len(topNodes) > 5 {
			topNodes = topNodes[:5]
		}

		currentPrice := candles[0].Close

		// Determine position relative to value area
		position := "in_value_area"
		if currentPrice > valueAreaHigh {
			position = "above_value_area"
		} else if currentPrice < valueAreaLow {
			position = "below_value_area"
		}

		// Signal based on position
		signal := "neutral"
		if position == "above_value_area" {
			signal = "bullish" // Buyers in control
		} else if position == "below_value_area" {
			signal = "bearish" // Sellers in control
		}

		return map[string]interface{}{
			"poc":             pocPrice,
			"value_area_high": valueAreaHigh,
			"value_area_low":  valueAreaLow,
			"current_price":   currentPrice,
			"position":        position,
			"signal":          signal,
			"top_nodes":       topNodes,
			"total_volume":    totalVolume,
		}, nil
	})
}

