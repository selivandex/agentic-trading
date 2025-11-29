package analysis

import (
	"testing"
	"time"

	"prometheus/internal/domain/market_data"
)

// TestFeatureExtractor_HelperMethods tests individual feature calculation methods
func TestFeatureExtractor_HelperMethods(t *testing.T) {
	t.Run("HistoricalVolatility", func(t *testing.T) {
		fe := &FeatureExtractor{}

		// Sample price data with some volatility
		closes := []float64{100, 102, 98, 103, 97, 105, 99, 101, 104, 96}

		vol := fe.calculateHistoricalVolatility(closes)

		if vol == 0 {
			t.Error("Historical volatility should not be zero")
		}

		if vol < 0 || vol > 100 {
			t.Errorf("Invalid volatility value: %.2f%%", vol)
		}

		t.Logf("Historical volatility: %.2f%%", vol)
	})

	t.Run("EMAAlignment", func(t *testing.T) {
		fe := &FeatureExtractor{}

		// Bullish alignment: 9 > 21 > 55 > 200
		alignment := fe.determineEMAAlignment(105, 103, 100, 98)
		if alignment != "strong_bullish" {
			t.Errorf("Expected strong_bullish, got %s", alignment)
		}

		// Bearish alignment: 9 < 21 < 55 < 200
		alignment = fe.determineEMAAlignment(95, 97, 100, 102)
		if alignment != "strong_bearish" {
			t.Errorf("Expected strong_bearish, got %s", alignment)
		}

		// Neutral
		alignment = fe.determineEMAAlignment(100, 99, 101, 98)
		if alignment != "neutral" {
			t.Errorf("Expected neutral, got %s", alignment)
		}
	})

	t.Run("SwingCounting", func(t *testing.T) {
		fe := &FeatureExtractor{}

		// Uptrend: higher highs, fewer lower lows
		highs := []float64{100, 102, 103, 105, 104, 106}
		lows := []float64{98, 99, 101, 102, 101, 103}

		higherHighs, lowerLows := fe.countSwings(highs, lows)

		if higherHighs == 0 {
			t.Error("Should have detected higher highs")
		}

		t.Logf("Higher highs: %d, Lower lows: %d", higherHighs, lowerLows)
	})
}

// Helper functions

func generateSampleCandles(count int) []market_data.OHLCV {
	candles := make([]market_data.OHLCV, count)
	basePrice := 43000.0
	baseTime := time.Now().Add(-time.Duration(count) * time.Hour)

	for i := 0; i < count; i++ {
		// Simple sine wave pattern for realistic price movement
		priceVariation := basePrice * 0.02 * float64(i%20-10) / 10.0
		price := basePrice + priceVariation

		candles[i] = market_data.OHLCV{
			Exchange:    "binance",
			Symbol:      "BTC-USDT",
			Timeframe:   "1h",
			OpenTime:    baseTime.Add(time.Duration(i) * time.Hour),
			Open:        price * 0.998,
			High:        price * 1.005,
			Low:         price * 0.995,
			Close:       price,
			Volume:      1000.0 + float64(i)*10,
			QuoteVolume: price * (1000.0 + float64(i)*10),
			Trades:      1000 + uint64(i),
		}
	}

	return candles
}
