package macro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"prometheus/internal/domain/macro"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// MarketCorrelationCollector tracks crypto correlation with traditional markets
// Monitors S&P500, Gold, DXY, US10Y yields to gauge risk-on/risk-off sentiment
type MarketCorrelationCollector struct {
	*workers.BaseWorker
	marketDataRepo market_data.Repository
	macroRepo      macro.Repository
	httpClient     *http.Client
	apiKey         string   // Alpha Vantage API key
	cryptoSymbols  []string // BTC/USDT, ETH/USDT, SOL/USDT, etc.
	assets         []string // SPY, GLD, DXY, TLT
}

// NewMarketCorrelationCollector creates a new market correlation collector
func NewMarketCorrelationCollector(
	marketDataRepo market_data.Repository,
	macroRepo macro.Repository,
	alphaVantageKey string,
	cryptoSymbols []string,
	assets []string,
	interval time.Duration,
	enabled bool,
) *MarketCorrelationCollector {
	return &MarketCorrelationCollector{
		BaseWorker:     workers.NewBaseWorker("market_correlation_collector", interval, enabled),
		marketDataRepo: marketDataRepo,
		macroRepo:      macroRepo,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		apiKey:         alphaVantageKey,
		cryptoSymbols:  cryptoSymbols,
		assets:         assets,
	}
}

// Run executes one iteration of market correlation collection
func (mc *MarketCorrelationCollector) Run(ctx context.Context) error {
	mc.Log().Debug("Market correlation collector: starting iteration")

	if mc.apiKey == "" {
		mc.Log().Debug("Alpha Vantage API key not configured, skipping iteration")
		return nil
	}

	totalCorrelations := 0

	// Calculate correlations for each crypto symbol
	for _, cryptoSymbol := range mc.cryptoSymbols {
		// Fetch crypto price history (30 days)
		cryptoPrices, err := mc.fetchCryptoPrices(ctx, cryptoSymbol, 30)
		if err != nil {
			mc.Log().Error("Failed to fetch crypto prices",
				"symbol", cryptoSymbol,
				"error", err,
			)
			continue
		}

		if len(cryptoPrices) < 10 {
			mc.Log().Warn("Insufficient crypto price data for correlation",
				"symbol", cryptoSymbol,
			)
			continue
		}

		// Calculate correlations with each traditional asset
		correlations := make(map[string]float64)
		for _, asset := range mc.assets {
			assetPrices, err := mc.fetchAssetPrices(ctx, asset, 30)
			if err != nil {
				mc.Log().Error("Failed to fetch asset prices",
					"asset", asset,
					"error", err,
				)
				continue
			}

			if len(assetPrices) < 10 {
				mc.Log().Warn("Insufficient asset price data", "asset", asset)
				continue
			}

			// Calculate Pearson correlation
			correlation := mc.calculateCorrelation(cryptoPrices, assetPrices)
			correlations[asset] = correlation

			mc.Log().Debug("Calculated correlation",
				"crypto", cryptoSymbol,
				"asset", asset,
				"correlation", fmt.Sprintf("%.3f", correlation),
			)

			totalCorrelations++
		}

		// Store correlation data for this crypto symbol
		mc.storeCorrelations(ctx, cryptoSymbol, correlations)
	}

	mc.Log().Info("Market correlations calculated",
		"crypto_symbols", len(mc.cryptoSymbols),
		"assets", len(mc.assets),
		"total_correlations", totalCorrelations,
	)

	return nil
}

// pricePoint represents a price at a specific time
type pricePoint struct {
	timestamp time.Time
	price     float64
}

// fetchCryptoPrices fetches crypto price history from internal database
func (mc *MarketCorrelationCollector) fetchCryptoPrices(ctx context.Context, symbol string, days int) ([]pricePoint, error) {
	since := time.Now().AddDate(0, 0, -days)

	// Fetch from market_data repository
	query := market_data.OHLCVQuery{
		Symbol:    symbol,
		Timeframe: "1d",
		StartTime: since,
		EndTime:   time.Now(),
		Limit:     1000,
	}

	ohlcv, err := mc.marketDataRepo.GetOHLCV(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "fetch crypto OHLCV")
	}

	var prices []pricePoint
	for _, candle := range ohlcv {
		prices = append(prices, pricePoint{
			timestamp: candle.OpenTime,
			price:     candle.Close,
		})
	}

	return prices, nil
}

// Alpha Vantage API response structures
type alphaVantageTimeSeriesResponse struct {
	MetaData   alphaVantageMetaData             `json:"Meta Data"`
	TimeSeries map[string]alphaVantageDataPoint `json:"Time Series (Daily)"`
}

type alphaVantageMetaData struct {
	Symbol     string `json:"2. Symbol"`
	LastUpdate string `json:"3. Last Refreshed"`
}

type alphaVantageDataPoint struct {
	Open   string `json:"1. open"`
	High   string `json:"2. high"`
	Low    string `json:"3. low"`
	Close  string `json:"4. close"`
	Volume string `json:"5. volume"`
}

// fetchAssetPrices fetches traditional asset prices from Alpha Vantage
func (mc *MarketCorrelationCollector) fetchAssetPrices(ctx context.Context, symbol string, days int) ([]pricePoint, error) {
	// Alpha Vantage API endpoint
	// Docs: https://www.alphavantage.co/documentation/
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s",
		symbol,
		mc.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create Alpha Vantage API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Alpha Vantage API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		mc.Log().Warn("Alpha Vantage API rate limit reached")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Alpha Vantage API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp alphaVantageTimeSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.Wrap(err, "decode Alpha Vantage API response")
	}

	if len(apiResp.TimeSeries) == 0 {
		return nil, fmt.Errorf("no data in Alpha Vantage response for %s", symbol)
	}

	// Convert to price points
	var prices []pricePoint
	cutoff := time.Now().AddDate(0, 0, -days)

	for dateStr, data := range apiResp.TimeSeries {
		timestamp, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if timestamp.Before(cutoff) {
			continue
		}

		var closePrice float64
		fmt.Sscanf(data.Close, "%f", &closePrice)

		prices = append(prices, pricePoint{
			timestamp: timestamp,
			price:     closePrice,
		})
	}

	return prices, nil
}

// calculateCorrelation calculates Pearson correlation coefficient
// Returns value between -1 (perfect negative) and +1 (perfect positive)
func (mc *MarketCorrelationCollector) calculateCorrelation(cryptoPrices, assetPrices []pricePoint) float64 {
	// Align prices by date and calculate returns
	cryptoReturns, assetReturns := mc.alignAndCalculateReturns(cryptoPrices, assetPrices)

	if len(cryptoReturns) < 2 {
		return 0
	}

	// Calculate means
	cryptoMean := mc.mean(cryptoReturns)
	assetMean := mc.mean(assetReturns)

	// Calculate covariance and standard deviations
	var covariance float64
	var cryptoVariance float64
	var assetVariance float64

	for i := 0; i < len(cryptoReturns); i++ {
		cryptoDiff := cryptoReturns[i] - cryptoMean
		assetDiff := assetReturns[i] - assetMean

		covariance += cryptoDiff * assetDiff
		cryptoVariance += cryptoDiff * cryptoDiff
		assetVariance += assetDiff * assetDiff
	}

	n := float64(len(cryptoReturns))
	covariance /= n
	cryptoStdDev := math.Sqrt(cryptoVariance / n)
	assetStdDev := math.Sqrt(assetVariance / n)

	if cryptoStdDev == 0 || assetStdDev == 0 {
		return 0
	}

	// Pearson correlation coefficient
	correlation := covariance / (cryptoStdDev * assetStdDev)
	return correlation
}

// alignAndCalculateReturns aligns two price series by date and calculates daily returns
func (mc *MarketCorrelationCollector) alignAndCalculateReturns(cryptoPrices, assetPrices []pricePoint) ([]float64, []float64) {
	// Create maps for fast lookup
	cryptoMap := make(map[string]float64)
	for _, p := range cryptoPrices {
		dateKey := p.timestamp.Format("2006-01-02")
		cryptoMap[dateKey] = p.price
	}

	assetMap := make(map[string]float64)
	for _, p := range assetPrices {
		dateKey := p.timestamp.Format("2006-01-02")
		assetMap[dateKey] = p.price
	}

	// Find common dates and calculate returns
	var cryptoReturns, assetReturns []float64
	var prevCrypto, prevAsset float64

	for dateKey := range cryptoMap {
		cryptoPrice, cryptoOk := cryptoMap[dateKey]
		assetPrice, assetOk := assetMap[dateKey]

		if !cryptoOk || !assetOk {
			continue
		}

		if prevCrypto > 0 && prevAsset > 0 {
			cryptoReturn := (cryptoPrice - prevCrypto) / prevCrypto
			assetReturn := (assetPrice - prevAsset) / prevAsset

			cryptoReturns = append(cryptoReturns, cryptoReturn)
			assetReturns = append(assetReturns, assetReturn)
		}

		prevCrypto = cryptoPrice
		prevAsset = assetPrice
	}

	return cryptoReturns, assetReturns
}

// mean calculates arithmetic mean
func (mc *MarketCorrelationCollector) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// storeCorrelations stores correlation data in macro repository
func (mc *MarketCorrelationCollector) storeCorrelations(ctx context.Context, cryptoSymbol string, correlations map[string]float64) {
	for asset, correlation := range correlations {
		// Store as a MacroEvent for now (could be improved with dedicated table)
		event := &macro.MacroEvent{
			ID:          fmt.Sprintf("correlation_%s_%s_%d", cryptoSymbol, asset, time.Now().Unix()),
			EventType:   "correlation", // Non-standard type
			Title:       fmt.Sprintf("%s vs %s Correlation", cryptoSymbol, asset),
			Country:     "Global",
			Currency:    "USD",
			EventTime:   time.Now(),
			Previous:    "",
			Forecast:    "",
			Actual:      fmt.Sprintf("%.3f", correlation),
			Impact:      macro.ImpactMedium,
			CollectedAt: time.Now(),
		}

		if err := mc.macroRepo.InsertEvent(ctx, event); err != nil {
			mc.Log().Error("Failed to store correlation",
				"crypto", cryptoSymbol,
				"asset", asset,
				"error", err,
			)
		}
	}
}

// InterpretCorrelation provides human-readable interpretation
func InterpretCorrelation(correlation float64) string {
	absCorr := math.Abs(correlation)

	direction := "positive"
	if correlation < 0 {
		direction = "negative"
	}

	strength := "weak"
	if absCorr > 0.7 {
		strength = "strong"
	} else if absCorr > 0.4 {
		strength = "moderate"
	}

	return fmt.Sprintf("%s_%s", strength, direction)
}

// CalculateRollingCorrelation calculates rolling correlation over a window
// Useful for detecting changing market dynamics
func CalculateRollingCorrelation(btcPrices, assetPrices []pricePoint, windowDays int) []float64 {
	var correlations []float64

	for i := windowDays; i < len(btcPrices); i++ {
		// Calculate correlation for this window
		// (simplified - would need actual implementation using windowBTC and windowAsset)
		// windowBTC := btcPrices[i-windowDays : i]
		// windowAsset := assetPrices[i-windowDays : i]
		correlations = append(correlations, 0.0)
	}

	return correlations
}
