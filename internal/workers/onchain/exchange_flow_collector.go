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

// ExchangeFlowCollector tracks BTC/ETH flows to/from exchanges
// Monitors exchange reserves to detect accumulation or distribution
type ExchangeFlowCollector struct {
	*workers.BaseWorker
	onchainRepo onchain.Repository
	httpClient  *http.Client
	apiKey      string
	exchanges   []string
	tokens      []string
}

// NewExchangeFlowCollector creates a new exchange flow collector
func NewExchangeFlowCollector(
	onchainRepo onchain.Repository,
	cryptoquantAPIKey string,
	exchanges []string,
	tokens []string,
	interval time.Duration,
	enabled bool,
) *ExchangeFlowCollector {
	return &ExchangeFlowCollector{
		BaseWorker:  workers.NewBaseWorker("exchange_flow_collector", interval, enabled),
		onchainRepo: onchainRepo,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		apiKey:      cryptoquantAPIKey,
		exchanges:   exchanges,
		tokens:      tokens,
	}
}

// Run executes one iteration of exchange flow collection
func (ec *ExchangeFlowCollector) Run(ctx context.Context) error {
	ec.Log().Debug("Exchange flow collector: starting iteration")

	if ec.apiKey == "" {
		ec.Log().Debug("CryptoQuant API key not configured, skipping iteration")
		return nil
	}

	totalFlows := 0
	processedTokens := 0

	// Collect flows for each token and exchange
	for _, token := range ec.tokens {
		// Check for context cancellation (graceful shutdown) - outer loop
		select {
		case <-ctx.Done():
			ec.Log().Info("Exchange flow collector interrupted by shutdown",
				"tokens_processed", processedTokens,
				"tokens_remaining", len(ec.tokens)-processedTokens,
				"flows_saved", totalFlows,
			)
			return ctx.Err()
		default:
		}

		for _, exchange := range ec.exchanges {
			// Check for context cancellation in inner loop (if many exchanges)
			select {
			case <-ctx.Done():
				ec.Log().Info("Exchange flow collector interrupted by shutdown",
					"current_token", token,
					"flows_saved", totalFlows,
				)
				return ctx.Err()
			default:
			}

			flow, err := ec.fetchExchangeFlow(ctx, exchange, token)
			if err != nil {
				ec.Log().Error("Failed to fetch exchange flow",
					"exchange", exchange,
					"token", token,
					"error", err,
				)
				continue
			}

			if flow == nil {
				continue
			}

			// Store in database
			if err := ec.onchainRepo.InsertExchangeFlow(ctx, flow); err != nil {
				ec.Log().Error("Failed to save exchange flow",
					"exchange", exchange,
					"token", token,
					"error", err,
				)
				continue
			}

			totalFlows++
		}
		processedTokens++
	}

	ec.Log().Info("Exchange flows collected",
		"count", totalFlows,
		"exchanges", len(ec.exchanges),
		"tokens", len(ec.tokens),
	)

	return nil
}

// CryptoQuant API response structures
// Note: Actual CryptoQuant API structure may differ - this is a placeholder
type cryptoquantFlowResponse struct {
	Status string              `json:"status"`
	Data   cryptoquantFlowData `json:"data"`
}

type cryptoquantFlowData struct {
	Inflow   float64 `json:"inflow"`
	Outflow  float64 `json:"outflow"`
	NetFlow  float64 `json:"net_flow"`
	Datetime string  `json:"datetime"`
}

// fetchExchangeFlow fetches flow data for a specific exchange and token
func (ec *ExchangeFlowCollector) fetchExchangeFlow(ctx context.Context, exchange, token string) (*onchain.ExchangeFlow, error) {
	// CryptoQuant API endpoint (placeholder - actual API structure may differ)
	// Docs: https://docs.cryptoquant.com/
	url := fmt.Sprintf("https://api.cryptoquant.com/v1/exchange-flows/%s/%s/latest?key=%s",
		exchange,
		token,
		ec.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create CryptoQuant API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := ec.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "CryptoQuant API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		ec.Log().Warn("CryptoQuant API rate limit reached")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CryptoQuant API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp cryptoquantFlowResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.Wrap(err, "decode CryptoQuant API response")
	}

	if apiResp.Status != "success" {
		return nil, fmt.Errorf("CryptoQuant API returned status: %s", apiResp.Status)
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, apiResp.Data.Datetime)
	if err != nil {
		timestamp = time.Now()
	}

	// Determine blockchain
	blockchain := "bitcoin"
	if token == "ETH" || token == "USDT" || token == "USDC" {
		blockchain = "ethereum"
	}

	return &onchain.ExchangeFlow{
		Exchange:   exchange,
		Blockchain: blockchain,
		Token:      token,
		InflowUSD:  apiResp.Data.Inflow,
		OutflowUSD: apiResp.Data.Outflow,
		NetFlowUSD: apiResp.Data.NetFlow,
		Timestamp:  timestamp,
	}, nil
}

// CalculateNetFlowTrend calculates the trend of net flows over time
// Helper function for analytics
func (ec *ExchangeFlowCollector) CalculateNetFlowTrend(ctx context.Context, exchange, token string, days int) (float64, error) {
	since := time.Now().AddDate(0, 0, -days)
	flows, err := ec.onchainRepo.GetExchangeFlows(ctx, exchange, since)
	if err != nil {
		return 0, errors.Wrap(err, "get exchange flows for trend")
	}

	if len(flows) == 0 {
		return 0, nil
	}

	// Filter by token
	var tokenFlows []onchain.ExchangeFlow
	for _, flow := range flows {
		if flow.Token == token {
			tokenFlows = append(tokenFlows, flow)
		}
	}

	if len(tokenFlows) == 0 {
		return 0, nil
	}

	// Calculate average net flow
	totalNetFlow := 0.0
	for _, flow := range tokenFlows {
		totalNetFlow += flow.NetFlowUSD
	}

	avgNetFlow := totalNetFlow / float64(len(tokenFlows))
	return avgNetFlow, nil
}

// InterpretNetFlow provides human-readable interpretation of net flow
func InterpretNetFlow(netFlowUSD float64, token string) string {
	// Thresholds vary by token market cap
	highThreshold := 10_000_000.0 // $10M
	if token == "ETH" {
		highThreshold = 5_000_000.0 // $5M
	}

	if netFlowUSD > highThreshold {
		return "strong_inflow" // Potential accumulation, bearish for price
	} else if netFlowUSD > 0 {
		return "moderate_inflow"
	} else if netFlowUSD < -highThreshold {
		return "strong_outflow" // Potential distribution, bullish for price
	} else if netFlowUSD < 0 {
		return "moderate_outflow"
	}

	return "neutral"
}
