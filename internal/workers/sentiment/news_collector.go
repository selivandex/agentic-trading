package sentiment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"prometheus/internal/domain/sentiment"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// NewsCollector collects crypto news from external sources
// Currently supports CryptoPanic API as a simple, free news aggregator
// In production, you'd want multiple sources: CoinGecko, Twitter, Reddit, etc.
type NewsCollector struct {
	*workers.BaseWorker
	sentimentRepo sentiment.Repository
	httpClient    *http.Client
	apiKey        string // CryptoPanic API key (optional for public endpoint)
	currencies    []string
}

// NewNewsCollector creates a new news collector worker
func NewNewsCollector(
	sentimentRepo sentiment.Repository,
	apiKey string,
	currencies []string,
	interval time.Duration,
	enabled bool,
) *NewsCollector {
	return &NewsCollector{
		BaseWorker:    workers.NewBaseWorker("news_collector", interval, enabled),
		sentimentRepo: sentimentRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:     apiKey,
		currencies: currencies,
	}
}

// Run executes one iteration of news collection
func (nc *NewsCollector) Run(ctx context.Context) error {
	nc.Log().Debug("News collector: starting iteration")

	// Collect news from CryptoPanic
	articles, err := nc.collectFromCryptoPanic(ctx)
	if err != nil {
		nc.Log().Error("Failed to collect news from CryptoPanic", "error", err)
		// Don't fail the whole worker if one source fails
		return nil
	}

	// Save to database
	savedCount := 0
	for _, article := range articles {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			nc.Log().Info("News saving interrupted by shutdown",
				"total_articles", len(articles),
				"saved_articles", savedCount,
				"remaining", len(articles)-savedCount,
			)
			return ctx.Err()
		default:
		}

		if err := nc.sentimentRepo.InsertNews(ctx, article); err != nil {
			nc.Log().Error("Failed to save news article",
				"title", article.Title,
				"error", err,
			)
			continue
		}
		savedCount++
	}

	nc.Log().Info("News collection complete",
		"total_articles", len(articles),
		"saved_articles", savedCount,
	)

	return nil
}

// CryptoPanic API response structures
type cryptoPanicResponse struct {
	Count   int                  `json:"count"`
	Next    string               `json:"next"`
	Results []cryptoPanicArticle `json:"results"`
}

type cryptoPanicArticle struct {
	Kind        string                `json:"kind"`
	Domain      string                `json:"domain"`
	Title       string                `json:"title"`
	URL         string                `json:"url"`
	Source      cryptoPanicSource     `json:"source"`
	PublishedAt string                `json:"published_at"`
	CreatedAt   string                `json:"created_at"`
	Votes       cryptoPanicVotes      `json:"votes"`
	Currencies  []cryptoPanicCurrency `json:"currencies,omitempty"`
}

type cryptoPanicSource struct {
	Title  string `json:"title"`
	Region string `json:"region"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

type cryptoPanicVotes struct {
	Negative  int `json:"negative"`
	Positive  int `json:"positive"`
	Important int `json:"important"`
	Liked     int `json:"liked"`
	Disliked  int `json:"disliked"`
	Lol       int `json:"lol"`
	Toxic     int `json:"toxic"`
	Saved     int `json:"saved"`
}

type cryptoPanicCurrency struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
	URL   string `json:"url"`
}

// collectFromCryptoPanic fetches news from CryptoPanic API
func (nc *NewsCollector) collectFromCryptoPanic(ctx context.Context) ([]*sentiment.News, error) {
	// Build URL - using public endpoint (no auth required for basic access)
	// For production, use authenticated endpoint with API key
	url := "https://cryptopanic.com/api/v1/posts/"

	// Add filters if we have specific currencies
	if len(nc.currencies) > 0 {
		// CryptoPanic uses currency codes like BTC, ETH
		// Join currencies with comma
		currencyFilter := ""
		for i, curr := range nc.currencies {
			if i > 0 {
				currencyFilter += ","
			}
			currencyFilter += curr
		}
		url += "?currencies=" + currencyFilter
	}

	// Add API key if available
	if nc.apiKey != "" {
		if len(nc.currencies) > 0 {
			url += "&auth_token=" + nc.apiKey
		} else {
			url += "?auth_token=" + nc.apiKey
		}
	}

	nc.Log().Debug("Fetching news from CryptoPanic", "url", url)

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch news")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CryptoPanic API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response cryptoPanicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	nc.Log().Debug("Fetched news articles", "count", len(response.Results))

	// Convert to internal format
	var articles []*sentiment.News
	for _, item := range response.Results {
		article := nc.convertCryptoPanicArticle(item)
		if article != nil {
			articles = append(articles, article)
		}
	}

	return articles, nil
}

// convertCryptoPanicArticle converts CryptoPanic article to internal format
func (nc *NewsCollector) convertCryptoPanicArticle(item cryptoPanicArticle) *sentiment.News {
	// Parse published time
	publishedAt, err := time.Parse(time.RFC3339, item.PublishedAt)
	if err != nil {
		nc.Log().Error("Failed to parse published_at", "value", item.PublishedAt, "error", err)
		publishedAt = time.Now()
	}

	// Calculate sentiment score from votes
	// Simple heuristic: (positive - negative) / total_votes
	// Range: -1.0 (very negative) to +1.0 (very positive)
	totalVotes := item.Votes.Positive + item.Votes.Negative + item.Votes.Important
	sentimentScore := 0.0
	if totalVotes > 0 {
		sentimentScore = float64(item.Votes.Positive-item.Votes.Negative) / float64(totalVotes)
	}

	// Extract currency symbols
	var symbols []string
	for _, curr := range item.Currencies {
		symbols = append(symbols, curr.Code)
	}

	return &sentiment.News{
		ID:          item.URL, // Use URL as unique ID
		Source:      "cryptopanic",
		Title:       item.Title,
		Content:     "", // CryptoPanic doesn't provide full content in API response
		URL:         item.URL,
		Sentiment:   sentimentScore,
		Symbols:     symbols,
		PublishedAt: publishedAt,
		CollectedAt: time.Now(),
	}
}
