package derivatives

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"prometheus/internal/domain/derivatives"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// OptionsFlowCollector tracks large options trades from Deribit
// Monitors unusual options activity that may signal directional bets
type OptionsFlowCollector struct {
	*workers.BaseWorker
	derivRepo     derivatives.Repository
	httpClient    *http.Client
	apiKey        string
	apiSecret     string
	symbols       []string
	minPremiumUSD float64
}

// NewOptionsFlowCollector creates a new options flow collector
func NewOptionsFlowCollector(
	derivRepo derivatives.Repository,
	deribitAPIKey string,
	deribitAPISecret string,
	symbols []string,
	minPremiumUSD float64,
	interval time.Duration,
	enabled bool,
) *OptionsFlowCollector {
	return &OptionsFlowCollector{
		BaseWorker:    workers.NewBaseWorker("options_flow_collector", interval, enabled),
		derivRepo:     derivRepo,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		apiKey:        deribitAPIKey,
		apiSecret:     deribitAPISecret,
		symbols:       symbols,
		minPremiumUSD: minPremiumUSD,
	}
}

// Run executes one iteration of options flow collection
func (oc *OptionsFlowCollector) Run(ctx context.Context) error {
	oc.Log().Debug("Options flow collector: starting iteration")

	if oc.apiKey == "" {
		oc.Log().Debug("Deribit API key not configured, skipping iteration")
		return nil
	}

	totalFlows := 0

	// Collect options flows for each symbol
	for _, symbol := range oc.symbols {
		flows, err := oc.fetchOptionsFlows(ctx, symbol)
		if err != nil {
			oc.Log().Error("Failed to fetch options flows",
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		if len(flows) == 0 {
			continue
		}

		// Store each flow
		for _, flow := range flows {
			if err := oc.derivRepo.InsertOptionsFlow(ctx, &flow); err != nil {
				oc.Log().Error("Failed to save options flow",
					"symbol", symbol,
					"error", err,
				)
				continue
			}
			totalFlows++
		}
	}

	oc.Log().Info("Options flows collected",
		"count", totalFlows,
		"symbols", len(oc.symbols),
		"min_premium", oc.minPremiumUSD,
	)

	return nil
}

// Deribit API response structures
type deribitTradesResponse struct {
	Jsonrpc string              `json:"jsonrpc"`
	ID      int                 `json:"id"`
	Result  deribitTradesResult `json:"result"`
}

type deribitTradesResult struct {
	Trades  []deribitTrade `json:"trades"`
	HasMore bool           `json:"has_more"`
}

type deribitTrade struct {
	TradeID        string  `json:"trade_id"`
	Timestamp      int64   `json:"timestamp"`
	InstrumentName string  `json:"instrument_name"`
	Amount         float64 `json:"amount"`
	Price          float64 `json:"price"`
	Direction      string  `json:"direction"` // buy, sell
	IndexPrice     float64 `json:"index_price"`
	IV             float64 `json:"iv"`
}

// fetchOptionsFlows fetches recent large options trades from Deribit
func (oc *OptionsFlowCollector) fetchOptionsFlows(ctx context.Context, symbol string) ([]derivatives.OptionsFlow, error) {
	// Deribit API endpoint
	// Docs: https://docs.deribit.com/

	// Get trades for options instruments (e.g., BTC-28DEC24-40000-C)
	// We'll query recent trades and filter by size

	baseSymbol := symbol
	if symbol == "BTC/USDT" {
		baseSymbol = "BTC"
	} else if symbol == "ETH/USDT" {
		baseSymbol = "ETH"
	}

	// For simplicity, get all instruments for this currency
	// In production, you'd want to be more specific about expiries/strikes
	url := fmt.Sprintf("https://www.deribit.com/api/v2/public/get_last_trades_by_currency?currency=%s&kind=option&count=100",
		baseSymbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create Deribit API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Deribit API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		oc.Log().Warn("Deribit API rate limit reached")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Deribit API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp deribitTradesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.Wrap(err, "decode Deribit API response")
	}

	oc.Log().Debug("Fetched options trades from Deribit",
		"symbol", symbol,
		"count", len(apiResp.Result.Trades),
	)

	// Convert to internal format and filter by premium
	var flows []derivatives.OptionsFlow
	for _, trade := range apiResp.Result.Trades {
		flow := oc.convertTrade(trade, symbol)
		if flow != nil && flow.Premium >= oc.minPremiumUSD {
			flows = append(flows, *flow)
		}
	}

	return flows, nil
}

// convertTrade converts Deribit trade to internal OptionsFlow format
func (oc *OptionsFlowCollector) convertTrade(trade deribitTrade, symbol string) *derivatives.OptionsFlow {
	// Parse instrument name: BTC-28DEC24-40000-C
	// Format: SYMBOL-EXPIRY-STRIKE-TYPE
	strike, expiry, side := oc.parseInstrumentName(trade.InstrumentName)
	if strike == 0 {
		return nil
	}

	// Calculate premium in USD
	premium := trade.Amount * trade.Price * trade.IndexPrice

	// Determine sentiment based on direction and option type
	sentiment := oc.determineSentiment(trade.Direction, side)

	timestamp := time.Unix(trade.Timestamp/1000, 0)

	return &derivatives.OptionsFlow{
		ID:        fmt.Sprintf("deribit_%s", trade.TradeID),
		Timestamp: timestamp,
		Symbol:    symbol,
		Side:      side,
		Strike:    strike,
		Expiry:    expiry,
		Premium:   premium,
		Size:      int(trade.Amount),
		Spot:      trade.IndexPrice,
		TradeType: trade.Direction,
		Sentiment: sentiment,
	}
}

// parseInstrumentName extracts strike, expiry, and side from Deribit instrument name
func (oc *OptionsFlowCollector) parseInstrumentName(instrument string) (float64, time.Time, string) {
	// Example: BTC-28DEC24-40000-C
	// Parse using simple string operations

	var strike float64
	var expiry time.Time
	var side string

	// Extract strike and side (last two parts after splitting by -)
	parts := oc.splitInstrument(instrument)
	if len(parts) < 4 {
		return 0, expiry, ""
	}

	// Parse strike
	fmt.Sscanf(parts[2], "%f", &strike)

	// Parse side (C=call, P=put)
	if parts[3] == "C" {
		side = "call"
	} else if parts[3] == "P" {
		side = "put"
	}

	// Parse expiry date (e.g., 28DEC24)
	expiry, _ = time.Parse("2JAN06", parts[1])

	return strike, expiry, side
}

// splitInstrument splits instrument name by hyphen
func (oc *OptionsFlowCollector) splitInstrument(instrument string) []string {
	var parts []string
	current := ""

	for _, char := range instrument {
		if char == '-' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// determineSentiment determines bullish/bearish sentiment from trade
func (oc *OptionsFlowCollector) determineSentiment(direction, side string) string {
	// Buy calls = bullish
	// Buy puts = bearish
	// Sell calls = bearish (covered calls or bearish spread)
	// Sell puts = bullish (cash-secured puts or bullish spread)

	if direction == "buy" && side == "call" {
		return "bullish"
	}
	if direction == "buy" && side == "put" {
		return "bearish"
	}
	if direction == "sell" && side == "call" {
		return "bearish"
	}
	if direction == "sell" && side == "put" {
		return "bullish"
	}

	return "neutral"
}

// CalculatePutCallRatio calculates put/call ratio from recent flows
func CalculatePutCallRatio(flows []derivatives.OptionsFlow) float64 {
	var callVolume, putVolume float64

	for _, flow := range flows {
		if flow.Side == "call" {
			callVolume += flow.Premium
		} else if flow.Side == "put" {
			putVolume += flow.Premium
		}
	}

	if callVolume == 0 {
		return 0
	}

	return putVolume / callVolume
}

// InterpretPutCallRatio provides interpretation
// >1.0 = more puts (bearish), <1.0 = more calls (bullish)
func InterpretPutCallRatio(ratio float64) string {
	if ratio > 1.5 {
		return "very_bearish"
	} else if ratio > 1.0 {
		return "bearish"
	} else if ratio < 0.5 {
		return "very_bullish"
	} else if ratio < 1.0 {
		return "bullish"
	}
	return "neutral"
}
