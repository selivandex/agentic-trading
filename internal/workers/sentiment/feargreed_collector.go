package sentiment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"prometheus/internal/domain/sentiment"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// FearGreedCollector collects the Crypto Fear & Greed Index
// Free API from Alternative.me - no authentication required
type FearGreedCollector struct {
	*workers.BaseWorker
	sentimentRepo sentiment.Repository
	httpClient    *http.Client
}

// NewFearGreedCollector creates a new Fear & Greed index collector
func NewFearGreedCollector(
	sentimentRepo sentiment.Repository,
	interval time.Duration,
	enabled bool,
) *FearGreedCollector {
	return &FearGreedCollector{
		BaseWorker:    workers.NewBaseWorker("feargreed_collector", interval, enabled),
		sentimentRepo: sentimentRepo,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Run executes one iteration of Fear & Greed index collection
func (fg *FearGreedCollector) Run(ctx context.Context) error {
	fg.Log().Debug("Fear & Greed collector: starting iteration")

	// Fetch latest index value
	index, err := fg.fetchFearGreedIndex(ctx)
	if err != nil {
		fg.Log().Error("Failed to fetch Fear & Greed index", "error", err)
		return errors.Wrap(err, "fetch fear greed index")
	}

	if index == nil {
		fg.Log().Warn("No Fear & Greed data returned")
		return nil
	}

	// Store in database
	if err := fg.sentimentRepo.InsertFearGreed(ctx, index); err != nil {
		fg.Log().Error("Failed to save Fear & Greed index", "error", err)
		return errors.Wrap(err, "save fear greed index")
	}

	fg.Log().Info("Fear & Greed index collected",
		"value", index.Value,
		"rating", index.Rating,
	)

	return nil
}

// Alternative.me API response structure
type fearGreedAPIResponse struct {
	Name     string              `json:"name"`
	Data     []fearGreedDataItem `json:"data"`
	Metadata fearGreedMetadata   `json:"metadata"`
}

type fearGreedDataItem struct {
	Value               string `json:"value"`
	ValueClassification string `json:"value_classification"`
	Timestamp           string `json:"timestamp"`
	TimeUntilUpdate     string `json:"time_until_update"`
}

type fearGreedMetadata struct {
	Error string `json:"error"`
}

// fetchFearGreedIndex fetches the current Fear & Greed index from Alternative.me
func (fg *FearGreedCollector) fetchFearGreedIndex(ctx context.Context) (*sentiment.FearGreedIndex, error) {
	// Alternative.me API endpoint (free, no auth required)
	url := "https://api.alternative.me/fng/?limit=1"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create API request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := fg.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp fearGreedAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.Wrap(err, "decode API response")
	}

	// Check for API errors
	if apiResp.Metadata.Error != "" {
		return nil, fmt.Errorf("API error: %s", apiResp.Metadata.Error)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("no data in API response")
	}

	dataItem := apiResp.Data[0]

	// Parse value (should be 0-100)
	value, err := strconv.Atoi(dataItem.Value)
	if err != nil {
		return nil, errors.Wrap(err, "parse fear greed value")
	}

	// Parse timestamp (Unix timestamp)
	timestampInt, err := strconv.ParseInt(dataItem.Timestamp, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "parse timestamp")
	}
	timestamp := time.Unix(timestampInt, 0)

	// Map value_classification to our rating enum
	rating := fg.mapValueClassification(dataItem.ValueClassification)

	fg.Log().Debug("Fetched Fear & Greed index",
		"value", value,
		"classification", dataItem.ValueClassification,
		"rating", rating,
		"timestamp", timestamp,
	)

	return &sentiment.FearGreedIndex{
		Timestamp: timestamp,
		Value:     value,
		Rating:    rating,
	}, nil
}

// mapValueClassification maps API classification to our rating enum
// API classifications: "Extreme Fear", "Fear", "Neutral", "Greed", "Extreme Greed"
func (fg *FearGreedCollector) mapValueClassification(classification string) string {
	switch classification {
	case "Extreme Fear":
		return "extreme_fear"
	case "Fear":
		return "fear"
	case "Neutral":
		return "neutral"
	case "Greed":
		return "greed"
	case "Extreme Greed":
		return "extreme_greed"
	default:
		// Fallback based on naming patterns
		lower := ""
		for _, r := range classification {
			if r >= 'A' && r <= 'Z' {
				if lower != "" {
					lower += "_"
				}
				lower += string(r + 32) // Convert to lowercase
			} else {
				lower += string(r)
			}
		}
		return lower
	}
}


