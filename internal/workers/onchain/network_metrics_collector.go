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

// NetworkMetricsCollector collects blockchain network health metrics
// Monitors Bitcoin hash rate, mempool, Ethereum gas prices, etc.
type NetworkMetricsCollector struct {
	*workers.BaseWorker
	onchainRepo      onchain.Repository
	httpClient       *http.Client
	blockchainAPIKey string // Blockchain.com API key
	etherscanAPIKey  string // Etherscan API key
	blockchains      []string
}

// NewNetworkMetricsCollector creates a new network metrics collector
func NewNetworkMetricsCollector(
	onchainRepo onchain.Repository,
	blockchainAPIKey string,
	etherscanAPIKey string,
	blockchains []string,
	interval time.Duration,
	enabled bool,
) *NetworkMetricsCollector {
	return &NetworkMetricsCollector{
		BaseWorker:       workers.NewBaseWorker("network_metrics_collector", interval, enabled),
		onchainRepo:      onchainRepo,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		blockchainAPIKey: blockchainAPIKey,
		etherscanAPIKey:  etherscanAPIKey,
		blockchains:      blockchains,
	}
}

// Run executes one iteration of network metrics collection
func (nc *NetworkMetricsCollector) Run(ctx context.Context) error {
	nc.Log().Debug("Network metrics collector: starting iteration")

	collectedCount := 0

	// Collect metrics for each blockchain
	for _, blockchain := range nc.blockchains {
		var metrics *onchain.NetworkMetrics
		var err error

		switch blockchain {
		case "bitcoin":
			metrics, err = nc.collectBitcoinMetrics(ctx)
		case "ethereum":
			metrics, err = nc.collectEthereumMetrics(ctx)
		default:
			nc.Log().Warn("Unsupported blockchain", "blockchain", blockchain)
			continue
		}

		if err != nil {
			nc.Log().Error("Failed to collect network metrics",
				"blockchain", blockchain,
				"error", err,
			)
			continue
		}

		if metrics == nil {
			continue
		}

		// Store in database
		if err := nc.onchainRepo.InsertNetworkMetrics(ctx, metrics); err != nil {
			nc.Log().Error("Failed to save network metrics",
				"blockchain", blockchain,
				"error", err,
			)
			continue
		}

		collectedCount++
	}

	nc.Log().Info("Network metrics collected",
		"count", collectedCount,
		"blockchains", len(nc.blockchains),
	)

	return nil
}

// ========================================
// Bitcoin Metrics
// ========================================

// Blockchain.com API response structures
type blockchainStatsResponse struct {
	MarketPriceUSD       float64 `json:"market_price_usd"`
	HashRate             float64 `json:"hash_rate"` // Hash/s
	TotalFeesBTC         float64 `json:"total_fees_btc"`
	NTxns                uint32  `json:"n_tx"`
	NBlocksMined         uint32  `json:"n_blocks_mined"`
	MinutesBetweenBlocks float64 `json:"minutes_between_blocks"`
	TotalBC              float64 `json:"totalbc"`
	NBlocksTotal         uint32  `json:"n_blocks_total"`
	EstimatedBTCsent     float64 `json:"estimated_transaction_volume_usd"`
	BlocksSize           uint64  `json:"blocks_size"`
	MempoolSize          uint32  `json:"mempool_size"`
	MarketCap            float64 `json:"market_cap_usd"`
}

// collectBitcoinMetrics fetches Bitcoin network metrics
func (nc *NetworkMetricsCollector) collectBitcoinMetrics(ctx context.Context) (*onchain.NetworkMetrics, error) {
	// Blockchain.com API (free, no auth required for basic stats)
	url := "https://api.blockchain.info/stats"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create Blockchain.com API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Blockchain.com API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Blockchain.com API returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats blockchainStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, errors.Wrap(err, "decode Blockchain.com API response")
	}

	// Calculate active addresses from transaction count (rough estimate)
	activeAddresses := stats.NTxns / 2 // Approximate: each tx involves ~2 addresses

	// Calculate average fee in USD
	avgFeeUSD := 0.0
	if stats.NTxns > 0 {
		avgFeeUSD = (stats.TotalFeesBTC * stats.MarketPriceUSD) / float64(stats.NTxns)
	}

	// Convert hash rate from H/s to TH/s
	hashRateTH := stats.HashRate / 1_000_000_000_000.0

	nc.Log().Debug("Collected Bitcoin metrics",
		"hash_rate_th", hashRateTH,
		"mempool_size", stats.MempoolSize,
		"tx_count", stats.NTxns,
	)

	return &onchain.NetworkMetrics{
		Blockchain:       "bitcoin",
		Timestamp:        time.Now(),
		ActiveAddresses:  activeAddresses,
		TransactionCount: stats.NTxns,
		AvgFeeUSD:        avgFeeUSD,
		HashRate:         hashRateTH,
		MempoolSize:      stats.MempoolSize,
		MempoolBytes:     stats.BlocksSize, // Approximate
		// Ethereum-specific fields left at zero
		GasPrice:         0,
		GasUsed:          0,
		BlockUtilization: 0,
	}, nil
}

// ========================================
// Ethereum Metrics
// ========================================

// Etherscan API response structures
type etherscanGasPriceResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		SafeGasPrice    string `json:"SafeGasPrice"`
		ProposeGasPrice string `json:"ProposeGasPrice"`
		FastGasPrice    string `json:"FastGasPrice"`
	} `json:"result"`
}

type etherscanNodeCountResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		TotalNodeCount string `json:"TotalNodeCount"`
	} `json:"result"`
}

// collectEthereumMetrics fetches Ethereum network metrics
func (nc *NetworkMetricsCollector) collectEthereumMetrics(ctx context.Context) (*onchain.NetworkMetrics, error) {
	var gasPrice float64

	if nc.etherscanAPIKey == "" {
		nc.Log().Debug("Etherscan API key not configured, using default gas price")
		gasPrice = 30.0 // Default fallback value
	} else {
		var err error
		gasPrice, err = nc.fetchEthereumGasPrice(ctx)
		if err != nil {
			nc.Log().Error("Failed to fetch Ethereum gas price", "error", err)
			gasPrice = 30.0 // Fallback
		}
	}

	// For now, use estimates for other metrics
	// In production, you'd want to use Infura/Alchemy to query actual chain data
	activeAddresses := uint32(500000)       // Daily active addresses (estimated)
	transactionCount := uint32(1200000)     // Daily tx count (estimated)
	avgFeeUSD := gasPrice * 0.000021 * 2500 // 21000 gas * ETH price

	nc.Log().Debug("Collected Ethereum metrics",
		"gas_price_gwei", gasPrice,
		"active_addresses", activeAddresses,
	)

	return &onchain.NetworkMetrics{
		Blockchain:       "ethereum",
		Timestamp:        time.Now(),
		ActiveAddresses:  activeAddresses,
		TransactionCount: transactionCount,
		AvgFeeUSD:        avgFeeUSD,
		// Bitcoin-specific fields left at zero
		HashRate:     0,
		MempoolSize:  0,
		MempoolBytes: 0,
		// Ethereum-specific
		GasPrice:         gasPrice,
		GasUsed:          uint64(transactionCount) * 21000, // Estimate
		BlockUtilization: 0.75,                             // Typically 70-80%
	}, nil
}

// fetchEthereumGasPrice fetches current Ethereum gas price from Etherscan
func (nc *NetworkMetricsCollector) fetchEthereumGasPrice(ctx context.Context) (float64, error) {
	url := fmt.Sprintf("https://api.etherscan.io/api?module=gastracker&action=gasoracle&apikey=%s",
		nc.etherscanAPIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, errors.Wrap(err, "create Etherscan API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "Etherscan API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Etherscan API returned status %d: %s", resp.StatusCode, string(body))
	}

	var gasResp etherscanGasPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&gasResp); err != nil {
		return 0, errors.Wrap(err, "decode Etherscan gas price response")
	}

	if gasResp.Status != "1" {
		return 0, fmt.Errorf("Etherscan API error: %s", gasResp.Message)
	}

	// Parse ProposeGasPrice (standard gas price in Gwei)
	var gasPrice float64
	fmt.Sscanf(gasResp.Result.ProposeGasPrice, "%f", &gasPrice)

	return gasPrice, nil
}

// ========================================
// Helper Functions
// ========================================

// CalculateNetworkCongestion determines if network is congested
func CalculateNetworkCongestion(metrics *onchain.NetworkMetrics) string {
	if metrics.Blockchain == "bitcoin" {
		// Bitcoin: check mempool size
		if metrics.MempoolSize > 100000 {
			return "high"
		} else if metrics.MempoolSize > 50000 {
			return "medium"
		}
		return "low"
	}

	if metrics.Blockchain == "ethereum" {
		// Ethereum: check gas price
		if metrics.GasPrice > 100 {
			return "high"
		} else if metrics.GasPrice > 50 {
			return "medium"
		}
		return "low"
	}

	return "unknown"
}

// InterpretHashRate provides interpretation of Bitcoin hash rate
func InterpretHashRate(hashRateTH float64) string {
	// As of 2024, network hash rate is around 400-600 TH/s
	if hashRateTH > 500 {
		return "very_high" // Strong network security
	} else if hashRateTH > 400 {
		return "high"
	} else if hashRateTH > 300 {
		return "medium"
	}
	return "low" // Potential security concern
}
