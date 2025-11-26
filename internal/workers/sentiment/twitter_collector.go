package sentiment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"prometheus/internal/domain/sentiment"
	"prometheus/internal/workers"
	"prometheus/pkg/errors"
)

// TwitterCollector collects tweets from crypto influencers
// Uses Twitter API v2 to track sentiment from key accounts
type TwitterCollector struct {
	*workers.BaseWorker
	sentimentRepo   sentiment.Repository
	httpClient      *http.Client
	apiKey          string
	apiSecret       string
	bearerToken     string
	trackedAccounts []string // Twitter usernames without @
	symbols         []string
}

// NewTwitterCollector creates a new Twitter collector worker
func NewTwitterCollector(
	sentimentRepo sentiment.Repository,
	twitterAPIKey string,
	twitterAPISecret string,
	trackedAccounts []string,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *TwitterCollector {
	return &TwitterCollector{
		BaseWorker:      workers.NewBaseWorker("twitter_collector", interval, enabled),
		sentimentRepo:   sentimentRepo,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		apiKey:          twitterAPIKey,
		apiSecret:       twitterAPISecret,
		bearerToken:     twitterAPIKey, // For simplicity, assume API key is bearer token
		trackedAccounts: trackedAccounts,
		symbols:         symbols,
	}
}

// Run executes one iteration of Twitter sentiment collection
func (tc *TwitterCollector) Run(ctx context.Context) error {
	tc.Log().Debug("Twitter collector: starting iteration")

	if tc.bearerToken == "" {
		tc.Log().Warn("Twitter API bearer token not configured, skipping collection")
		return nil
	}

	totalMentions := 0
	
	// Collect tweets for each tracked symbol
	for _, symbol := range tc.symbols {
		tweets, err := tc.collectTweets(ctx, symbol)
		if err != nil {
			tc.Log().Error("Failed to collect tweets", "symbol", symbol, "error", err)
			continue
		}

		if len(tweets) == 0 {
			continue
		}

		// Aggregate sentiment for this symbol
		aggregated := tc.aggregateSentiment(symbol, tweets)
		if aggregated == nil {
			continue
		}

		// Store in database
		if err := tc.sentimentRepo.InsertSocialSentiment(ctx, aggregated); err != nil {
			tc.Log().Error("Failed to save Twitter sentiment",
				"symbol", symbol,
				"error", err,
			)
			continue
		}

		totalMentions += int(aggregated.Mentions)
	}

	tc.Log().Info("Twitter collection complete",
		"symbols", len(tc.symbols),
		"total_mentions", totalMentions,
	)

	return nil
}

// Twitter API v2 response structures
type twitterSearchResponse struct {
	Data []twitterTweet       `json:"data"`
	Meta twitterResponseMeta  `json:"meta"`
}

type twitterTweet struct {
	ID        string               `json:"id"`
	Text      string               `json:"text"`
	AuthorID  string               `json:"author_id"`
	CreatedAt string               `json:"created_at"`
	Metrics   twitterPublicMetrics `json:"public_metrics"`
}

type twitterPublicMetrics struct {
	RetweetCount int `json:"retweet_count"`
	LikeCount    int `json:"like_count"`
	ReplyCount   int `json:"reply_count"`
	QuoteCount   int `json:"quote_count"`
}

type twitterResponseMeta struct {
	ResultCount int    `json:"result_count"`
	NextToken   string `json:"next_token"`
}

// collectTweets fetches recent tweets mentioning a symbol
func (tc *TwitterCollector) collectTweets(ctx context.Context, symbol string) ([]twitterTweet, error) {
	// Build search query
	// Example: "(BTC OR Bitcoin OR $BTC) -is:retweet lang:en"
	query := tc.buildSearchQuery(symbol)
	
	// Twitter API v2 endpoint
	url := fmt.Sprintf("https://api.twitter.com/2/tweets/search/recent?query=%s&max_results=100&tweet.fields=created_at,public_metrics,author_id",
		query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create Twitter API request")
	}

	// Add bearer token authentication
	req.Header.Set("Authorization", "Bearer "+tc.bearerToken)
	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Twitter API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		// Rate limited
		tc.Log().Warn("Twitter API rate limit reached")
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Twitter API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response twitterSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "decode Twitter API response")
	}

	tc.Log().Debug("Fetched tweets from Twitter",
		"symbol", symbol,
		"count", len(response.Data),
	)

	return response.Data, nil
}

// buildSearchQuery creates a Twitter search query for a symbol
func (tc *TwitterCollector) buildSearchQuery(symbol string) string {
	// Remove /USDT suffix if present
	baseSymbol := strings.Split(symbol, "/")[0]
	
	// Common crypto terms for the symbol
	var terms []string
	switch baseSymbol {
	case "BTC":
		terms = []string{"BTC", "Bitcoin", "$BTC"}
	case "ETH":
		terms = []string{"ETH", "Ethereum", "$ETH"}
	case "SOL":
		terms = []string{"SOL", "Solana", "$SOL"}
	default:
		terms = []string{baseSymbol, "$" + baseSymbol}
	}

	// Build query: "(term1 OR term2) -is:retweet lang:en"
	query := "(" + strings.Join(terms, " OR ") + ") -is:retweet lang:en"
	
	return query
}

// aggregateSentiment analyzes tweets and creates aggregated sentiment
func (tc *TwitterCollector) aggregateSentiment(symbol string, tweets []twitterTweet) *sentiment.SocialSentiment {
	if len(tweets) == 0 {
		return nil
	}

	var positiveCount, negativeCount, neutralCount uint32
	var totalEngagement uint32

	// Analyze each tweet
	for _, tweet := range tweets {
		sentiment := tc.analyzeTweetSentiment(tweet.Text)
		
		switch {
		case sentiment > 0.2:
			positiveCount++
		case sentiment < -0.2:
			negativeCount++
		default:
			neutralCount++
		}

		// Weight by engagement (retweets + likes)
		engagement := uint32(tweet.Metrics.RetweetCount + tweet.Metrics.LikeCount)
		totalEngagement += engagement
	}

	totalMentions := uint32(len(tweets))
	
	// Calculate overall sentiment score: (positive - negative) / total
	// Range: -1.0 (very bearish) to +1.0 (very bullish)
	sentimentScore := 0.0
	if totalMentions > 0 {
		sentimentScore = float64(int32(positiveCount)-int32(negativeCount)) / float64(totalMentions)
	}

	// Influencer sentiment (same for now, could weight by follower count)
	influencerSentiment := sentimentScore

	return &sentiment.SocialSentiment{
		Platform:            "twitter",
		Symbol:              symbol,
		Timestamp:           time.Now(),
		Mentions:            totalMentions,
		SentimentScore:      sentimentScore,
		PositiveCount:       positiveCount,
		NegativeCount:       negativeCount,
		NeutralCount:        neutralCount,
		InfluencerSentiment: influencerSentiment,
		TrendingRank:        0, // Not tracked from API
	}
}

// analyzeTweetSentiment performs simple keyword-based sentiment analysis
// Returns: positive (>0), negative (<0), or neutral (0)
func (tc *TwitterCollector) analyzeTweetSentiment(text string) float64 {
	textLower := strings.ToLower(text)

	// Bullish keywords
	bullishWords := []string{
		"bullish", "moon", "pump", "rally", "breakout", "long",
		"buy", "calls", "accumulate", "strong", "bullrun",
		"ATH", "all-time high", "parabolic", "rocket", "🚀",
		"📈", "💎", "WAGMI", "up", "gain",
	}

	// Bearish keywords
	bearishWords := []string{
		"bearish", "dump", "crash", "drop", "fall", "short",
		"sell", "puts", "reject", "weak", "bearmarket",
		"resistance", "breakdown", "liquidation", "NGMI",
		"down", "loss", "fear", "panic", "📉",
	}

	bullishCount := 0
	for _, word := range bullishWords {
		if strings.Contains(textLower, word) {
			bullishCount++
		}
	}

	bearishCount := 0
	for _, word := range bearishWords {
		if strings.Contains(textLower, word) {
			bearishCount++
		}
	}

	total := bullishCount + bearishCount
	if total == 0 {
		return 0 // Neutral
	}

	// Normalize to -1.0 to +1.0
	score := float64(bullishCount-bearishCount) / float64(total)
	return score
}

