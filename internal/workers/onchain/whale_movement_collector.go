package onchain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"prometheus/internal/domain/onchain"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// WhaleMovementCollector tracks large wallet transfers
// Monitors significant on-chain movements that may indicate market-moving activity
type WhaleMovementCollector struct {
	*workers.BaseWorker
	onchainRepo  onchain.Repository
	httpClient   *http.Client
	apiKey       string
	blockchains  []string
	minAmountUSD float64
}

// NewWhaleMovementCollector creates a new whale movement collector
func NewWhaleMovementCollector(
	onchainRepo onchain.Repository,
	whaleAlertAPIKey string,
	blockchains []string,
	minAmountUSD float64,
	interval time.Duration,
	enabled bool,
) *WhaleMovementCollector {
	return &WhaleMovementCollector{
		BaseWorker:   workers.NewBaseWorker("whale_movement_collector", interval, enabled),
		onchainRepo:  onchainRepo,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		apiKey:       whaleAlertAPIKey,
		blockchains:  blockchains,
		minAmountUSD: minAmountUSD,
	}
}

// Run executes one iteration of whale movement collection
func (wc *WhaleMovementCollector) Run(ctx context.Context) error {
	wc.Log().Debug("Whale movement collector: starting iteration")

	if wc.apiKey == "" {
		wc.Log().Warn("Whale Alert API key not configured, skipping collection")
		return nil
	}

	// Collect recent whale movements
	movements, err := wc.fetchWhaleMovements(ctx)
	if err != nil {
		wc.Log().Error("Failed to fetch whale movements", "error", err)
		return errors.Wrap(err, "fetch whale movements")
	}

	if len(movements) == 0 {
		wc.Log().Debug("No whale movements found")
		return nil
	}

	// Store in batch
	if err := wc.onchainRepo.InsertWhaleMovementBatch(ctx, movements); err != nil {
		wc.Log().Error("Failed to save whale movements", "error", err)
		return errors.Wrap(err, "save whale movements")
	}

	wc.Log().Info("Whale movements collected",
		"count", len(movements),
		"min_amount_usd", wc.minAmountUSD,
	)

	return nil
}

// Whale Alert API response structures
type whaleAlertResponse struct {
	Result       string                  `json:"result"`
	Count        int                     `json:"count"`
	Cursor       string                  `json:"cursor"`
	Transactions []whaleAlertTransaction `json:"transactions"`
}

type whaleAlertTransaction struct {
	Blockchain      string            `json:"blockchain"`
	Symbol          string            `json:"symbol"`
	ID              string            `json:"id"`
	TransactionType string            `json:"transaction_type"`
	Hash            string            `json:"hash"`
	From            whaleAlertAddress `json:"from"`
	To              whaleAlertAddress `json:"to"`
	Timestamp       int64             `json:"timestamp"`
	Amount          float64           `json:"amount"`
	AmountUSD       float64           `json:"amount_usd"`
}

type whaleAlertAddress struct {
	Address   string `json:"address"`
	Owner     string `json:"owner"`
	OwnerType string `json:"owner_type"`
}

// fetchWhaleMovements fetches recent whale movements from Whale Alert API
func (wc *WhaleMovementCollector) fetchWhaleMovements(ctx context.Context) ([]onchain.WhaleMovement, error) {
	// Whale Alert API endpoint
	// Docs: https://docs.whale-alert.io/
	url := fmt.Sprintf("https://api.whale-alert.io/v1/transactions?api_key=%s&min_value=%d&limit=100",
		wc.apiKey,
		int(wc.minAmountUSD),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create Whale Alert API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := wc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Whale Alert API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		wc.Log().Warn("Whale Alert API rate limit reached")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Whale Alert API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp whaleAlertResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.Wrap(err, "decode Whale Alert API response")
	}

	if apiResp.Result != "success" {
		return nil, fmt.Errorf("Whale Alert API returned result: %s", apiResp.Result)
	}

	wc.Log().Debug("Fetched whale movements from API",
		"count", len(apiResp.Transactions),
	)

	// Convert to internal format
	var movements []onchain.WhaleMovement
	for _, tx := range apiResp.Transactions {
		// Filter by configured blockchains if specified
		if len(wc.blockchains) > 0 && !wc.containsBlockchain(tx.Blockchain) {
			continue
		}

		movement := wc.convertTransaction(tx)
		movements = append(movements, movement)
	}

	return movements, nil
}

// containsBlockchain checks if a blockchain is in the configured list
func (wc *WhaleMovementCollector) containsBlockchain(blockchain string) bool {
	for _, bc := range wc.blockchains {
		if bc == blockchain {
			return true
		}
	}
	return false
}

// convertTransaction converts Whale Alert transaction to internal format
func (wc *WhaleMovementCollector) convertTransaction(tx whaleAlertTransaction) onchain.WhaleMovement {
	timestamp := time.Unix(tx.Timestamp, 0)

	return onchain.WhaleMovement{
		TxHash:      tx.Hash,
		Blockchain:  tx.Blockchain,
		FromAddress: tx.From.Address,
		ToAddress:   tx.To.Address,
		Token:       tx.Symbol,
		Amount:      tx.Amount,
		AmountUSD:   tx.AmountUSD,
		Timestamp:   timestamp,
		FromLabel:   wc.formatLabel(tx.From),
		ToLabel:     wc.formatLabel(tx.To),
	}
}

// formatLabel creates a human-readable label for an address
func (wc *WhaleMovementCollector) formatLabel(addr whaleAlertAddress) string {
	if addr.Owner != "" {
		return addr.Owner
	}
	if addr.OwnerType != "" {
		return addr.OwnerType
	}
	return ""
}

// calculateMovementImpact estimates the market impact of a whale movement
// Helper function for future analytics
func (wc *WhaleMovementCollector) calculateMovementImpact(movement onchain.WhaleMovement) float64 {
	// Simple heuristic: impact = amount_usd / market_cap_factor
	// >$10M = high impact, >$5M = medium, <$5M = low
	impact := movement.AmountUSD / 10_000_000.0

	// Adjust based on direction (to exchange = potential sell pressure)
	if movement.ToLabel != "" && wc.isExchangeLabel(movement.ToLabel) {
		impact *= 1.5 // Increased impact for exchange deposits
	}

	return impact
}

// isExchangeLabel checks if a label represents an exchange
func (wc *WhaleMovementCollector) isExchangeLabel(label string) bool {
	exchanges := []string{
		"Binance", "Coinbase", "Kraken", "Bitfinex", "Huobi",
		"OKX", "Bybit", "FTX", "Gemini", "Bitstamp",
	}

	for _, exchange := range exchanges {
		if label == exchange {
			return true
		}
	}

	return false
}

// ParseBlockchainFromTxHash attempts to determine blockchain from transaction hash format
// Helper function for transactions without explicit blockchain field
func ParseBlockchainFromTxHash(txHash string) string {
	if len(txHash) == 0 {
		return "unknown"
	}

	// Ethereum-style (0x prefix, 66 chars)
	if txHash[:2] == "0x" && len(txHash) == 66 {
		return "ethereum"
	}

	// Bitcoin-style (64 hex chars, no prefix)
	if len(txHash) == 64 {
		_, err := strconv.ParseUint(txHash[:8], 16, 64)
		if err == nil {
			return "bitcoin"
		}
	}

	return "unknown"
}
