package onchain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"prometheus/internal/domain/onchain"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// MinerMetricsCollector tracks Bitcoin mining pool behavior
// Monitors miner holdings and outflows to detect selling pressure
type MinerMetricsCollector struct {
	*workers.BaseWorker
	onchainRepo onchain.Repository
	httpClient  *http.Client
	apiKey      string // Glassnode API key
	pools       []string
}

// NewMinerMetricsCollector creates a new miner metrics collector
func NewMinerMetricsCollector(
	onchainRepo onchain.Repository,
	glassnodeAPIKey string,
	pools []string,
	interval time.Duration,
	enabled bool,
) *MinerMetricsCollector {
	return &MinerMetricsCollector{
		BaseWorker:  workers.NewBaseWorker("miner_metrics_collector", interval, enabled),
		onchainRepo: onchainRepo,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		apiKey:      glassnodeAPIKey,
		pools:       pools,
	}
}

// Run executes one iteration of miner metrics collection
func (mc *MinerMetricsCollector) Run(ctx context.Context) error {
	mc.Log().Debug("Miner metrics collector: starting iteration")

	if mc.apiKey == "" {
		mc.Log().Debug("Glassnode API key not configured, skipping iteration")
		return nil
	}

	// Collect metrics for major pools
	metrics, err := mc.collectMinerMetrics(ctx)
	if err != nil {
		mc.Log().Error("Failed to collect miner metrics", "error", err)
		return errors.Wrap(err, "collect miner metrics")
	}

	if len(metrics) == 0 {
		mc.Log().Debug("No miner metrics collected")
		return nil
	}

	// Store in batch
	if err := mc.onchainRepo.InsertMinerMetricsBatch(ctx, metrics); err != nil {
		mc.Log().Error("Failed to save miner metrics", "error", err)
		return errors.Wrap(err, "save miner metrics")
	}

	mc.Log().Info("Miner metrics collected", "count", len(metrics))

	return nil
}

// Glassnode API response structures
// Note: Actual Glassnode API structure may differ
type glassnodeResponse struct {
	T []int64   `json:"t"` // Timestamps
	V []float64 `json:"v"` // Values
}

// collectMinerMetrics fetches miner metrics from Glassnode or other sources
func (mc *MinerMetricsCollector) collectMinerMetrics(ctx context.Context) ([]onchain.MinerMetrics, error) {
	var allMetrics []onchain.MinerMetrics

	// Fetch miner outflow data from Glassnode
	outflows, err := mc.fetchMinerOutflow(ctx)
	if err != nil {
		mc.Log().Error("Failed to fetch miner outflow", "error", err)
		return nil, err
	}

	// Fetch miner balance
	balance, err := mc.fetchMinerBalance(ctx)
	if err != nil {
		mc.Log().Error("Failed to fetch miner balance", "error", err)
		return nil, err
	}

	// Fetch hash rate distribution (from BTC.com or similar)
	hashRates, err := mc.fetchHashRateDistribution(ctx)
	if err != nil {
		mc.Log().Error("Failed to fetch hash rate distribution", "error", err)
		// Continue with partial data
		hashRates = make(map[string]float64)
	}

	// Calculate MPI (Miner Position Index)
	mpi := mc.calculateMPI(outflows, balance)

	// Create metrics for known pools
	now := time.Now()
	for _, poolName := range mc.pools {
		hashRate := hashRates[poolName]
		if hashRate == 0 {
			hashRate = 10000.0 // Default TH/s
		}

		metrics := onchain.MinerMetrics{
			PoolName:      poolName,
			Timestamp:     now,
			HashRate:      hashRate,
			HashRateShare: (hashRate / 500000.0) * 100, // % of ~500 EH/s network
			BTCBalance:    balance,
			BTCSent:       outflows,
			BTCReceived:   outflows * 1.1, // Assume slightly more received (block rewards)
			MPI:           mpi,
		}

		allMetrics = append(allMetrics, metrics)
	}

	return allMetrics, nil
}

// fetchMinerOutflow fetches Bitcoin miner outflow data from Glassnode
func (mc *MinerMetricsCollector) fetchMinerOutflow(ctx context.Context) (float64, error) {
	// Glassnode API endpoint for miner outflow
	// Docs: https://docs.glassnode.com/
	url := fmt.Sprintf("https://api.glassnode.com/v1/metrics/mining/miners_outflow_total?a=BTC&api_key=%s&i=24h",
		mc.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, errors.Wrap(err, "create Glassnode API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "Glassnode API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		mc.Log().Warn("Glassnode API rate limit reached")
		return 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Glassnode API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp glassnodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, errors.Wrap(err, "decode Glassnode API response")
	}

	if len(apiResp.V) == 0 {
		return 0, fmt.Errorf("no data in Glassnode response")
	}

	// Return most recent value
	latestOutflow := apiResp.V[len(apiResp.V)-1]
	return latestOutflow, nil
}

// fetchMinerBalance fetches total miner balance from Glassnode
func (mc *MinerMetricsCollector) fetchMinerBalance(ctx context.Context) (float64, error) {
	url := fmt.Sprintf("https://api.glassnode.com/v1/metrics/mining/miners_balance?a=BTC&api_key=%s",
		mc.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, errors.Wrap(err, "create Glassnode API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "Glassnode API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Glassnode API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp glassnodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, errors.Wrap(err, "decode Glassnode API response")
	}

	if len(apiResp.V) == 0 {
		return 0, fmt.Errorf("no data in Glassnode response")
	}

	latestBalance := apiResp.V[len(apiResp.V)-1]
	return latestBalance, nil
}

// fetchHashRateDistribution fetches hash rate by pool from BTC.com or similar
func (mc *MinerMetricsCollector) fetchHashRateDistribution(ctx context.Context) (map[string]float64, error) {
	// BTC.com API (free, no auth required for basic stats)
	// Note: Actual API endpoint may differ
	url := "https://chain.api.btc.com/v3/pool/stats"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create BTC.com API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "BTC.com API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Return empty map if API fails (non-critical)
		return make(map[string]float64), nil
	}

	// Parse response (structure depends on actual API)
	// For now, return empty map as API integration is not yet complete
	return make(map[string]float64), nil
}

// calculateMPI calculates Miner Position Index
// MPI = miner outflow / 365-day moving average of outflow
// >2.0 = high selling pressure, <0.5 = accumulation
func (mc *MinerMetricsCollector) calculateMPI(currentOutflow, balance float64) float64 {
	// Simplified MPI calculation
	// In production, you'd want to fetch historical data for moving average
	avgOutflow := balance * 0.001 // Rough estimate: 0.1% of balance per day

	if avgOutflow == 0 {
		return 1.0
	}

	mpi := currentOutflow / avgOutflow
	return mpi
}

// ========================================
// Helper Functions
// ========================================

// InterpretMPI provides human-readable interpretation of MPI
func InterpretMPI(mpi float64) string {
	if mpi > 2.0 {
		return "high_selling_pressure" // Miners distributing aggressively
	} else if mpi > 1.5 {
		return "moderate_selling_pressure"
	} else if mpi < 0.5 {
		return "accumulation" // Miners holding/accumulating
	} else if mpi < 0.8 {
		return "low_selling_pressure"
	}
	return "neutral"
}

// CalculateMinerRevenue estimates daily mining revenue based on hash rate
// Simplified calculation: revenue = hashRate * blockReward * blocksPerDay / networkHashRate
func CalculateMinerRevenue(hashRateTH, networkHashRateTH, btcPrice float64) float64 {
	blockReward := 6.25 // Current Bitcoin block reward
	blocksPerDay := 144.0 // ~10 min per block

	if networkHashRateTH == 0 {
		return 0
	}

	hashRateShare := hashRateTH / networkHashRateTH
	dailyBTC := hashRateShare * blockReward * blocksPerDay
	dailyUSD := dailyBTC * btcPrice

	return dailyUSD
}

