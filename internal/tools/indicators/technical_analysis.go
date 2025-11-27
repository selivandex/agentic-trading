package indicators

import (
	"math"
	"time"

	"github.com/markcheno/go-talib"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/errors"
	"prometheus/pkg/templates"

	"google.golang.org/adk/tool"
)

// NewTechnicalAnalysisTool returns comprehensive technical analysis in one call.
// This tool calculates ALL common indicators at once, avoiding multiple API calls
// and reducing LLM token usage.
//
// Output includes:
// - Price context (current, high, low, change)
// - Momentum (RSI, MACD, Stochastic, CCI, ROC)
// - Volatility (ATR, Bollinger, Keltner)
// - Trend (EMA ribbon, Supertrend, Ichimoku summary)
// - Volume (VWAP, OBV trend, volume ratio)
// - Overall signal assessment
func NewTechnicalAnalysisTool(deps shared.Deps) tool.Tool {
	return shared.NewToolBuilder(
		"get_technical_analysis",
		"Get comprehensive technical analysis with all indicators in one call. Returns momentum, volatility, trend, and volume indicators plus overall signal.",
		func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Load candles ONCE (enough for 200 EMA)
			candles, err := loadCandles(ctx, deps, args, 250)
			if err != nil {
				return nil, err
			}

			if len(candles) < 55 {
				return nil, errors.Wrapf(errors.ErrInvalidInput, "need at least 55 candles, got %d", len(candles))
			}

			// Prepare data for ta-lib (chronological order)
			data, err := PrepareData(candles)
			if err != nil {
				return nil, err
			}

			closes, err := PrepareCloses(candles)
			if err != nil {
				return nil, err
			}

			currentPrice := candles[0].Close

			// Calculate all indicators
			priceCtx := calculatePriceContext(candles, currentPrice)
			momentum := calculateMomentum(data, closes)
			volatility := calculateVolatility(data, closes, currentPrice)
			trend := calculateTrend(data, closes, currentPrice)
			volume := calculateVolumeIndicators(candles, data)

			// Build result map for signal generation
			result := map[string]interface{}{
				"momentum":   momentum,
				"trend":      trend,
				"volume":     volume,
				"volatility": volatility,
			}

			// Generate overall signal
			signal := generateOverallSignal(result)

			// Prepare data for template
			templateData := map[string]interface{}{
				"Symbol":     args["symbol"],
				"Exchange":   args["exchange"],
				"Timeframe":  args["timeframe"],
				"Timestamp":  time.Now().UTC().Format(time.RFC3339),
				"Price":      priceCtx,
				"Momentum":   momentum,
				"Volatility": volatility,
				"Trend":      trend,
				"Volume":     volume,
				"Signal":     signal,
			}

			// Render using template
			tmpl := templates.Get()
			analysis, err := tmpl.Render("tools/technical_analysis", templateData)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render technical analysis template")
			}

			return map[string]interface{}{
				"analysis": analysis,
			}, nil
		},
		deps,
	).
		WithTimeout(45*time.Second). // Longer timeout for comprehensive analysis
		WithRetry(2, 1*time.Second).
		Build()
}

// calculatePriceContext returns price-related metrics
func calculatePriceContext(candles []market_data.OHLCV, currentPrice float64) map[string]interface{} {
	// Find 24h high/low (assume hourly candles, take 24)
	lookback := min(24, len(candles))
	high24h := candles[0].High
	low24h := candles[0].Low

	for i := 0; i < lookback; i++ {
		if candles[i].High > high24h {
			high24h = candles[i].High
		}
		if candles[i].Low < low24h {
			low24h = candles[i].Low
		}
	}

	// Calculate price change
	prevClose := candles[1].Close
	change := ((currentPrice - prevClose) / prevClose) * 100

	// Position in range
	rangeSize := high24h - low24h
	positionInRange := 0.5
	if rangeSize > 0 {
		positionInRange = (currentPrice - low24h) / rangeSize
	}

	return map[string]interface{}{
		"current":           currentPrice,
		"high_24h":          high24h,
		"low_24h":           low24h,
		"change_pct":        math.Round(change*100) / 100,
		"position_in_range": math.Round(positionInRange*100) / 100, // 0 = at low, 1 = at high
	}
}

// calculateMomentum returns all momentum indicators
func calculateMomentum(data *TalibData, closes []float64) map[string]interface{} {
	result := make(map[string]interface{})

	// RSI (14)
	rsiValues := talib.Rsi(closes, 14)
	if rsi, err := GetLastValue(rsiValues); err == nil {
		rsiSignal := "neutral"
		if rsi < 30 {
			rsiSignal = "oversold"
		} else if rsi > 70 {
			rsiSignal = "overbought"
		} else if rsi > 50 {
			rsiSignal = "bullish"
		} else {
			rsiSignal = "bearish"
		}
		result["rsi"] = map[string]interface{}{
			"value":  math.Round(rsi*100) / 100,
			"signal": rsiSignal,
		}
	}

	// MACD (12, 26, 9)
	macdLine, signalLine, histogram := talib.Macd(closes, 12, 26, 9)
	if macd, err := GetLastValue(macdLine); err == nil {
		signal, _ := GetLastValue(signalLine)
		hist, _ := GetLastValue(histogram)

		macdSignal := "neutral"
		if macd > signal && hist > 0 {
			macdSignal = "bullish"
		} else if macd < signal && hist < 0 {
			macdSignal = "bearish"
		} else if macd > signal {
			macdSignal = "bullish_cross"
		} else {
			macdSignal = "bearish_cross"
		}

		result["macd"] = map[string]interface{}{
			"line":      math.Round(macd*100) / 100,
			"signal":    math.Round(signal*100) / 100,
			"histogram": math.Round(hist*100) / 100,
			"direction": macdSignal,
		}
	}

	// Stochastic (14, 3, 3)
	slowK, slowD := talib.Stoch(data.High, data.Low, data.Close, 14, 3, talib.SMA, 3, talib.SMA)
	if k, err := GetLastValue(slowK); err == nil {
		d, _ := GetLastValue(slowD)
		stochSignal := "neutral"
		if k < 20 && d < 20 {
			stochSignal = "oversold"
		} else if k > 80 && d > 80 {
			stochSignal = "overbought"
		} else if k > d {
			stochSignal = "bullish"
		} else {
			stochSignal = "bearish"
		}

		result["stochastic"] = map[string]interface{}{
			"k":      math.Round(k*100) / 100,
			"d":      math.Round(d*100) / 100,
			"signal": stochSignal,
		}
	}

	// CCI (20)
	cciValues := talib.Cci(data.High, data.Low, data.Close, 20)
	if cci, err := GetLastValue(cciValues); err == nil {
		cciSignal := "neutral"
		if cci < -100 {
			cciSignal = "oversold"
		} else if cci > 100 {
			cciSignal = "overbought"
		} else if cci > 0 {
			cciSignal = "bullish"
		} else {
			cciSignal = "bearish"
		}

		result["cci"] = map[string]interface{}{
			"value":  math.Round(cci*100) / 100,
			"signal": cciSignal,
		}
	}

	// ROC (12)
	rocValues := talib.Roc(closes, 12)
	if roc, err := GetLastValue(rocValues); err == nil {
		rocSignal := "neutral"
		if roc > 5 {
			rocSignal = "strong_bullish"
		} else if roc > 0 {
			rocSignal = "bullish"
		} else if roc < -5 {
			rocSignal = "strong_bearish"
		} else {
			rocSignal = "bearish"
		}

		result["roc"] = map[string]interface{}{
			"value":  math.Round(roc*100) / 100,
			"signal": rocSignal,
		}
	}

	return result
}

// calculateVolatility returns volatility indicators
func calculateVolatility(data *TalibData, closes []float64, currentPrice float64) map[string]interface{} {
	result := make(map[string]interface{})

	// ATR (14)
	atrValues := talib.Atr(data.High, data.Low, data.Close, 14)
	if atr, err := GetLastValue(atrValues); err == nil {
		atrPct := (atr / currentPrice) * 100
		volatility := "low"
		if atrPct > 5 {
			volatility = "extreme"
		} else if atrPct > 3 {
			volatility = "high"
		} else if atrPct > 1.5 {
			volatility = "moderate"
		}

		result["atr"] = map[string]interface{}{
			"value":      math.Round(atr*100) / 100,
			"pct":        math.Round(atrPct*100) / 100,
			"volatility": volatility,
		}
	}

	// Bollinger Bands (20, 2)
	upperBand, middleBand, lowerBand := talib.BBands(closes, 20, 2.0, 2.0, talib.SMA)
	if upper, err := GetLastValue(upperBand); err == nil {
		middle, _ := GetLastValue(middleBand)
		lower, _ := GetLastValue(lowerBand)
		bandwidth := ((upper - lower) / middle) * 100

		position := "middle"
		if currentPrice >= upper {
			position = "above_upper"
		} else if currentPrice <= lower {
			position = "below_lower"
		} else if currentPrice > middle {
			position = "upper_half"
		} else {
			position = "lower_half"
		}

		bbSignal := "neutral"
		switch position {
		case "below_lower":
			bbSignal = "oversold"
		case "above_upper":
			bbSignal = "overbought"
		}

		result["bollinger"] = map[string]interface{}{
			"upper":     math.Round(upper*100) / 100,
			"middle":    math.Round(middle*100) / 100,
			"lower":     math.Round(lower*100) / 100,
			"bandwidth": math.Round(bandwidth*100) / 100,
			"position":  position,
			"signal":    bbSignal,
		}
	}

	// Keltner Channels (20, 2)
	keltnerMiddle := talib.Ema(closes, 20)
	if kMiddle, err := GetLastValue(keltnerMiddle); err == nil {
		atr, _ := GetLastValue(atrValues)
		kUpper := kMiddle + (2 * atr)
		kLower := kMiddle - (2 * atr)

		result["keltner"] = map[string]interface{}{
			"upper":  math.Round(kUpper*100) / 100,
			"middle": math.Round(kMiddle*100) / 100,
			"lower":  math.Round(kLower*100) / 100,
		}
	}

	return result
}

// calculateTrend returns trend indicators
func calculateTrend(data *TalibData, closes []float64, currentPrice float64) map[string]interface{} {
	result := make(map[string]interface{})

	// EMA Ribbon (9, 21, 55, 200)
	ema9 := talib.Ema(closes, 9)
	ema21 := talib.Ema(closes, 21)
	ema55 := talib.Ema(closes, 55)

	e9, _ := GetLastValue(ema9)
	e21, _ := GetLastValue(ema21)
	e55, _ := GetLastValue(ema55)

	// Try 200 EMA if enough data
	var e200 float64
	if len(closes) >= 200 {
		ema200 := talib.Ema(closes, 200)
		e200, _ = GetLastValue(ema200)
	}

	// EMA alignment analysis
	alignment := "neutral"
	if e9 > e21 && e21 > e55 {
		if e200 > 0 && e55 > e200 {
			alignment = "strong_bullish"
		} else {
			alignment = "bullish"
		}
	} else if e9 < e21 && e21 < e55 {
		if e200 > 0 && e55 < e200 {
			alignment = "strong_bearish"
		} else {
			alignment = "bearish"
		}
	}

	priceVsEMA := "neutral"
	if currentPrice > e9 && currentPrice > e21 {
		priceVsEMA = "above_fast"
	} else if currentPrice < e9 && currentPrice < e21 {
		priceVsEMA = "below_fast"
	}

	result["ema_ribbon"] = map[string]interface{}{
		"ema_9":       math.Round(e9*100) / 100,
		"ema_21":      math.Round(e21*100) / 100,
		"ema_55":      math.Round(e55*100) / 100,
		"ema_200":     math.Round(e200*100) / 100,
		"alignment":   alignment,
		"price_vs_ma": priceVsEMA,
	}

	// Supertrend (10, 3)
	atrValues := talib.Atr(data.High, data.Low, data.Close, 10)
	supertrend, trend := calculateSupertrendValues(data, atrValues, 3.0)
	if st, err := GetLastValue(supertrend); err == nil {
		trendDir, _ := GetLastValue(trend)
		stSignal := "neutral"
		trendName := "ranging"
		if trendDir > 0 {
			trendName = "uptrend"
			stSignal = "bullish"
		} else if trendDir < 0 {
			trendName = "downtrend"
			stSignal = "bearish"
		}

		result["supertrend"] = map[string]interface{}{
			"value":  math.Round(st*100) / 100,
			"trend":  trendName,
			"signal": stSignal,
		}
	}

	// Simple trend direction based on higher highs/lows
	trendDirection := detectSimpleTrend(data)
	result["direction"] = trendDirection

	// Ichimoku Cloud (simplified)
	result["ichimoku"] = calculateIchimokuSummary(data)

	// Pivot Points (Classic)
	result["pivot_points"] = calculatePivotPointsSummary(data, currentPrice)

	return result
}

// calculateSupertrendValues computes supertrend (reuse from supertrend.go logic)
func calculateSupertrendValues(data *TalibData, atr []float64, multiplier float64) ([]float64, []float64) {
	n := len(data.Close)
	supertrend := make([]float64, n)
	trend := make([]float64, n)

	supertrend[0] = data.Close[0]
	trend[0] = 1

	for i := 1; i < n; i++ {
		hl2 := (data.High[i] + data.Low[i]) / 2
		upperBand := hl2 + (multiplier * atr[i])
		lowerBand := hl2 - (multiplier * atr[i])

		if data.Close[i] > supertrend[i-1] {
			supertrend[i] = lowerBand
			if supertrend[i] < supertrend[i-1] {
				supertrend[i] = supertrend[i-1]
			}
			trend[i] = 1
		} else {
			supertrend[i] = upperBand
			if supertrend[i] > supertrend[i-1] {
				supertrend[i] = supertrend[i-1]
			}
			trend[i] = -1
		}
	}
	return supertrend, trend
}

// detectSimpleTrend analyzes recent price action for trend direction
func detectSimpleTrend(data *TalibData) string {
	n := len(data.Close)
	if n < 20 {
		return "unknown"
	}

	// Check last 20 bars for higher highs/lows or lower highs/lows
	higherHighs := 0
	lowerLows := 0

	for i := n - 20; i < n-1; i++ {
		if data.High[i+1] > data.High[i] {
			higherHighs++
		}
		if data.Low[i+1] < data.Low[i] {
			lowerLows++
		}
	}

	if higherHighs > 12 {
		return "uptrend"
	} else if lowerLows > 12 {
		return "downtrend"
	}
	return "ranging"
}

// calculateVolumeIndicators returns volume-based indicators
func calculateVolumeIndicators(candles []market_data.OHLCV, data *TalibData) map[string]interface{} {
	result := make(map[string]interface{})

	// VWAP (simplified - intraday only)
	vwap := calculateSimpleVWAP(candles)
	currentPrice := candles[0].Close
	vwapSignal := "neutral"
	if currentPrice > vwap*1.01 {
		vwapSignal = "above_vwap"
	} else if currentPrice < vwap*0.99 {
		vwapSignal = "below_vwap"
	}

	result["vwap"] = map[string]interface{}{
		"value":  math.Round(vwap*100) / 100,
		"signal": vwapSignal,
	}

	// Volume ratio (current vs average)
	avgVolume := calculateAverageVolume(data.Volume, 20)
	currentVolume := data.Volume[len(data.Volume)-1]
	volumeRatio := 1.0
	if avgVolume > 0 {
		volumeRatio = currentVolume / avgVolume
	}

	volumeSignal := "normal"
	if volumeRatio > 2.0 {
		volumeSignal = "very_high"
	} else if volumeRatio > 1.5 {
		volumeSignal = "high"
	} else if volumeRatio < 0.5 {
		volumeSignal = "low"
	}

	result["volume"] = map[string]interface{}{
		"current": math.Round(currentVolume*100) / 100,
		"average": math.Round(avgVolume*100) / 100,
		"ratio":   math.Round(volumeRatio*100) / 100,
		"signal":  volumeSignal,
	}

	// OBV trend
	obvValues := talib.Obv(data.Close, data.Volume)
	if len(obvValues) >= 10 {
		obvRecent := obvValues[len(obvValues)-1]
		obv10Ago := obvValues[len(obvValues)-10]
		obvTrend := "flat"
		if obvRecent > obv10Ago*1.05 {
			obvTrend = "rising"
		} else if obvRecent < obv10Ago*0.95 {
			obvTrend = "falling"
		}
		result["obv_trend"] = obvTrend
	}

	// Volume Profile (simplified POC and Value Area)
	result["volume_profile"] = calculateVolumeProfileSummary(candles)

	// Delta Volume (buy vs sell pressure estimate)
	result["delta"] = calculateDeltaVolumeSummary(candles)

	return result
}

// calculateSimpleVWAP calculates a simple VWAP approximation
func calculateSimpleVWAP(candles []market_data.OHLCV) float64 {
	var sumPV, sumV float64
	lookback := min(24, len(candles))

	for i := 0; i < lookback; i++ {
		typicalPrice := (candles[i].High + candles[i].Low + candles[i].Close) / 3
		sumPV += typicalPrice * candles[i].Volume
		sumV += candles[i].Volume
	}

	if sumV == 0 {
		return candles[0].Close
	}
	return sumPV / sumV
}

// calculateAverageVolume calculates average volume over period
func calculateAverageVolume(volumes []float64, period int) float64 {
	if len(volumes) < period {
		period = len(volumes)
	}
	if period == 0 {
		return 0
	}

	sum := 0.0
	start := len(volumes) - period
	for i := start; i < len(volumes); i++ {
		sum += volumes[i]
	}
	return sum / float64(period)
}

// calculateIchimokuSummary returns Ichimoku Cloud summary
func calculateIchimokuSummary(data *TalibData) map[string]interface{} {
	n := len(data.Close)
	if n < 52 {
		return map[string]interface{}{
			"signal": "insufficient_data",
		}
	}

	// Tenkan-sen (9-period)
	tenkan := calculateIchimokuMidpoint(data.High, data.Low, n-9, n)

	// Kijun-sen (26-period)
	kijun := calculateIchimokuMidpoint(data.High, data.Low, n-26, n)

	// Senkou Span A
	senkouA := (tenkan + kijun) / 2

	// Senkou Span B (52-period)
	senkouB := calculateIchimokuMidpoint(data.High, data.Low, n-52, n)

	currentPrice := data.Close[n-1]

	// Cloud color
	cloudColor := "green"
	if senkouB > senkouA {
		cloudColor = "red"
	}

	// Price position
	pricePosition := "in_cloud"
	if currentPrice > senkouA && currentPrice > senkouB {
		pricePosition = "above_cloud"
	} else if currentPrice < senkouA && currentPrice < senkouB {
		pricePosition = "below_cloud"
	}

	// Signal
	signal := "neutral"
	if pricePosition == "above_cloud" && cloudColor == "green" {
		signal = "strong_bullish"
	} else if pricePosition == "above_cloud" {
		signal = "bullish"
	} else if pricePosition == "below_cloud" && cloudColor == "red" {
		signal = "strong_bearish"
	} else if pricePosition == "below_cloud" {
		signal = "bearish"
	}

	return map[string]interface{}{
		"tenkan":         math.Round(tenkan*100) / 100,
		"kijun":          math.Round(kijun*100) / 100,
		"senkou_a":       math.Round(senkouA*100) / 100,
		"senkou_b":       math.Round(senkouB*100) / 100,
		"cloud_color":    cloudColor,
		"price_position": pricePosition,
		"signal":         signal,
	}
}

// calculateIchimokuMidpoint calculates (highest high + lowest low) / 2
func calculateIchimokuMidpoint(highs, lows []float64, start, end int) float64 {
	if start < 0 {
		start = 0
	}
	if end > len(highs) {
		end = len(highs)
	}

	highest := highs[start]
	lowest := lows[start]

	for i := start; i < end; i++ {
		if highs[i] > highest {
			highest = highs[i]
		}
		if lows[i] < lowest {
			lowest = lows[i]
		}
	}

	return (highest + lowest) / 2
}

// calculateVolumeProfileSummary returns simplified volume profile with POC and Value Area
func calculateVolumeProfileSummary(candles []market_data.OHLCV) map[string]interface{} {
	if len(candles) < 10 {
		return map[string]interface{}{
			"signal": "insufficient_data",
		}
	}

	// Find price range
	minPrice := candles[0].Low
	maxPrice := candles[0].High
	for _, c := range candles {
		if c.Low < minPrice {
			minPrice = c.Low
		}
		if c.High > maxPrice {
			maxPrice = c.High
		}
	}

	// Create 20 bins
	bins := 20
	priceRange := maxPrice - minPrice
	if priceRange == 0 {
		return map[string]interface{}{
			"signal": "no_range",
		}
	}
	binSize := priceRange / float64(bins)
	volumeByPrice := make([]float64, bins)
	totalVolume := 0.0

	// Distribute volume
	for _, c := range candles {
		typicalPrice := (c.High + c.Low + c.Close) / 3
		binIndex := int((typicalPrice - minPrice) / binSize)
		if binIndex < 0 {
			binIndex = 0
		}
		if binIndex >= bins {
			binIndex = bins - 1
		}
		volumeByPrice[binIndex] += c.Volume
		totalVolume += c.Volume
	}

	// Find POC (Point of Control)
	maxVol := 0.0
	pocIndex := 0
	for i, vol := range volumeByPrice {
		if vol > maxVol {
			maxVol = vol
			pocIndex = i
		}
	}
	poc := minPrice + (float64(pocIndex)+0.5)*binSize

	// Find Value Area (70% of volume)
	targetVol := totalVolume * 0.70
	vaVol := volumeByPrice[pocIndex]
	vaHigh, vaLow := pocIndex, pocIndex

	for vaVol < targetVol && (vaHigh < bins-1 || vaLow > 0) {
		nextHigh := 0.0
		if vaHigh < bins-1 {
			nextHigh = volumeByPrice[vaHigh+1]
		}
		nextLow := 0.0
		if vaLow > 0 {
			nextLow = volumeByPrice[vaLow-1]
		}

		if nextHigh > nextLow && vaHigh < bins-1 {
			vaHigh++
			vaVol += nextHigh
		} else if vaLow > 0 {
			vaLow--
			vaVol += nextLow
		} else {
			break
		}
	}

	vah := minPrice + float64(vaHigh+1)*binSize
	val := minPrice + float64(vaLow)*binSize
	currentPrice := candles[0].Close

	// Position
	position := "in_value_area"
	if currentPrice > vah {
		position = "above_value_area"
	} else if currentPrice < val {
		position = "below_value_area"
	}

	signal := "neutral"
	switch position {
	case "above_value_area":
		signal = "bullish"
	case "below_value_area":
		signal = "bearish"
	}

	return map[string]interface{}{
		"poc":             math.Round(poc*100) / 100,
		"value_area_high": math.Round(vah*100) / 100,
		"value_area_low":  math.Round(val*100) / 100,
		"position":        position,
		"signal":          signal,
	}
}

// calculateDeltaVolumeSummary estimates buy vs sell pressure
func calculateDeltaVolumeSummary(candles []market_data.OHLCV) map[string]interface{} {
	if len(candles) < 5 {
		return map[string]interface{}{
			"signal": "insufficient_data",
		}
	}

	// Estimate delta using candle direction and volume
	// Bullish candle (close > open) = buy volume
	// Bearish candle (close < open) = sell volume
	var buyVolume, sellVolume float64

	lookback := 20
	if len(candles) < lookback {
		lookback = len(candles)
	}

	for i := 0; i < lookback; i++ {
		c := candles[i]
		if c.Close > c.Open {
			buyVolume += c.Volume
		} else {
			sellVolume += c.Volume
		}
	}

	totalVolume := buyVolume + sellVolume
	delta := buyVolume - sellVolume
	deltaRatio := 0.5
	if totalVolume > 0 {
		deltaRatio = buyVolume / totalVolume
	}

	signal := "neutral"
	if deltaRatio > 0.65 {
		signal = "strong_buying"
	} else if deltaRatio > 0.55 {
		signal = "buying"
	} else if deltaRatio < 0.35 {
		signal = "strong_selling"
	} else if deltaRatio < 0.45 {
		signal = "selling"
	}

	return map[string]interface{}{
		"buy_volume":  math.Round(buyVolume*100) / 100,
		"sell_volume": math.Round(sellVolume*100) / 100,
		"delta":       math.Round(delta*100) / 100,
		"ratio":       math.Round(deltaRatio*100) / 100,
		"signal":      signal,
	}
}

// calculatePivotPointsSummary returns pivot points from previous period
func calculatePivotPointsSummary(data *TalibData, currentPrice float64) map[string]interface{} {
	n := len(data.Close)
	if n < 2 {
		return map[string]interface{}{
			"signal": "insufficient_data",
		}
	}

	// Use previous candle for pivot calculation
	prevHigh := data.High[n-2]
	prevLow := data.Low[n-2]
	prevClose := data.Close[n-2]

	// Classic Pivot Points
	pivot := (prevHigh + prevLow + prevClose) / 3
	r1 := 2*pivot - prevLow
	s1 := 2*pivot - prevHigh
	r2 := pivot + (prevHigh - prevLow)
	s2 := pivot - (prevHigh - prevLow)

	// Current level
	level := "at_pivot"
	if currentPrice >= r2 {
		level = "above_r2"
	} else if currentPrice >= r1 {
		level = "above_r1"
	} else if currentPrice > pivot {
		level = "above_pivot"
	} else if currentPrice <= s2 {
		level = "below_s2"
	} else if currentPrice <= s1 {
		level = "below_s1"
	} else if currentPrice < pivot {
		level = "below_pivot"
	}

	// Signal
	signal := "neutral"
	if currentPrice > pivot && currentPrice < r1 {
		signal = "bullish"
	} else if currentPrice < pivot && currentPrice > s1 {
		signal = "bearish"
	} else if currentPrice >= r2 {
		signal = "strong_bullish"
	} else if currentPrice <= s2 {
		signal = "strong_bearish"
	}

	return map[string]interface{}{
		"pivot":  math.Round(pivot*100) / 100,
		"r1":     math.Round(r1*100) / 100,
		"r2":     math.Round(r2*100) / 100,
		"s1":     math.Round(s1*100) / 100,
		"s2":     math.Round(s2*100) / 100,
		"level":  level,
		"signal": signal,
	}
}

// generateOverallSignal produces a summary signal based on all indicators
func generateOverallSignal(result map[string]interface{}) map[string]interface{} {
	bullishCount := 0
	bearishCount := 0
	total := 0

	// Count signals from momentum
	if momentum, ok := result["momentum"].(map[string]interface{}); ok {
		for _, v := range momentum {
			if ind, ok := v.(map[string]interface{}); ok {
				if sig, ok := ind["signal"].(string); ok {
					total++
					switch sig {
					case "bullish", "strong_bullish", "oversold", "bullish_cross":
						bullishCount++
					case "bearish", "strong_bearish", "overbought", "bearish_cross":
						bearishCount++
					}
				}
			}
		}
	}

	// Count from trend
	if trend, ok := result["trend"].(map[string]interface{}); ok {
		if emaRibbon, ok := trend["ema_ribbon"].(map[string]interface{}); ok {
			if align, ok := emaRibbon["alignment"].(string); ok {
				total++
				switch align {
				case "bullish", "strong_bullish":
					bullishCount++
				case "bearish", "strong_bearish":
					bearishCount++
				}
			}
		}
		if supertrend, ok := trend["supertrend"].(map[string]interface{}); ok {
			if sig, ok := supertrend["signal"].(string); ok {
				total++
				switch sig {
				case "bullish":
					bullishCount++
				case "bearish":
					bearishCount++
				}
			}
		}
		// Ichimoku
		if ichimoku, ok := trend["ichimoku"].(map[string]interface{}); ok {
			if sig, ok := ichimoku["signal"].(string); ok && sig != "insufficient_data" {
				total++
				switch sig {
				case "bullish", "strong_bullish":
					bullishCount++
				case "bearish", "strong_bearish":
					bearishCount++
				}
			}
		}
		// Pivot Points
		if pivots, ok := trend["pivot_points"].(map[string]interface{}); ok {
			if sig, ok := pivots["signal"].(string); ok && sig != "insufficient_data" {
				total++
				switch sig {
				case "bullish", "strong_bullish":
					bullishCount++
				case "bearish", "strong_bearish":
					bearishCount++
				}
			}
		}
	}

	// Count from volume
	if volume, ok := result["volume"].(map[string]interface{}); ok {
		// Volume Profile
		if vp, ok := volume["volume_profile"].(map[string]interface{}); ok {
			if sig, ok := vp["signal"].(string); ok && sig != "insufficient_data" && sig != "no_range" {
				total++
				switch sig {
				case "bullish":
					bullishCount++
				case "bearish":
					bearishCount++
				}
			}
		}
		// Delta Volume
		if delta, ok := volume["delta"].(map[string]interface{}); ok {
			if sig, ok := delta["signal"].(string); ok && sig != "insufficient_data" {
				total++
				switch sig {
				case "buying", "strong_buying":
					bullishCount++
				case "selling", "strong_selling":
					bearishCount++
				}
			}
		}
	}

	// Determine overall
	direction := "neutral"
	strength := "weak"
	confidence := 0.5

	if total > 0 {
		bullishRatio := float64(bullishCount) / float64(total)
		bearishRatio := float64(bearishCount) / float64(total)

		if bullishRatio > 0.7 {
			direction = "bullish"
			strength = "strong"
			confidence = bullishRatio
		} else if bullishRatio > 0.5 {
			direction = "bullish"
			strength = "moderate"
			confidence = bullishRatio
		} else if bearishRatio > 0.7 {
			direction = "bearish"
			strength = "strong"
			confidence = bearishRatio
		} else if bearishRatio > 0.5 {
			direction = "bearish"
			strength = "moderate"
			confidence = bearishRatio
		}
	}

	return map[string]interface{}{
		"direction":     direction,
		"strength":      strength,
		"confidence":    math.Round(confidence*100) / 100,
		"bullish_count": bullishCount,
		"bearish_count": bearishCount,
		"total_signals": total,
	}
}
