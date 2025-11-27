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

// RedditCollector collects crypto sentiment from Reddit
// Monitors r/CryptoCurrency, r/Bitcoin, r/ethereum and other crypto subreddits
type RedditCollector struct {
	*workers.BaseWorker
	sentimentRepo sentiment.Repository
	httpClient    *http.Client
	clientID      string
	clientSecret  string
	accessToken   string
	tokenExpiry   time.Time
	subreddits    []string
	symbols       []string
}

// NewRedditCollector creates a new Reddit collector worker
func NewRedditCollector(
	sentimentRepo sentiment.Repository,
	redditClientID string,
	redditClientSecret string,
	subreddits []string,
	symbols []string,
	interval time.Duration,
	enabled bool,
) *RedditCollector {
	return &RedditCollector{
		BaseWorker:    workers.NewBaseWorker("reddit_collector", interval, enabled),
		sentimentRepo: sentimentRepo,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		clientID:      redditClientID,
		clientSecret:  redditClientSecret,
		subreddits:    subreddits,
		symbols:       symbols,
	}
}

// Run executes one iteration of Reddit sentiment collection
func (rc *RedditCollector) Run(ctx context.Context) error {
	rc.Log().Debug("Reddit collector: starting iteration")

	if rc.clientID == "" || rc.clientSecret == "" {
		rc.Log().Warn("Reddit API credentials not configured, skipping collection")
		return nil
	}

	// Refresh access token if needed
	if time.Now().After(rc.tokenExpiry) {
		if err := rc.refreshAccessToken(ctx); err != nil {
			return errors.Wrap(err, "refresh Reddit access token")
		}
	}

	totalPosts := 0
	processedCount := 0

	// Collect posts for each symbol
	for _, symbol := range rc.symbols {
		// Check for context cancellation (graceful shutdown)
		select {
		case <-ctx.Done():
			rc.Log().Info("Reddit collector interrupted by shutdown",
				"symbols_processed", processedCount,
				"symbols_remaining", len(rc.symbols)-processedCount,
				"total_posts", totalPosts,
			)
			return ctx.Err()
		default:
		}

		posts, err := rc.collectPosts(ctx, symbol)
		if err != nil {
			rc.Log().Error("Failed to collect Reddit posts", "symbol", symbol, "error", err)
			processedCount++
			continue
		}

		if len(posts) == 0 {
			processedCount++
			continue
		}

		// Aggregate sentiment
		aggregated := rc.aggregateSentiment(symbol, posts)
		if aggregated == nil {
			processedCount++
			continue
		}

		// Store in database
		if err := rc.sentimentRepo.InsertSocialSentiment(ctx, aggregated); err != nil {
			rc.Log().Error("Failed to save Reddit sentiment",
				"symbol", symbol,
				"error", err,
			)
			processedCount++
			continue
		}

		totalPosts += len(posts)
		processedCount++
	}

	rc.Log().Info("Reddit collection complete",
		"symbols", len(rc.symbols),
		"total_posts", totalPosts,
	)

	return nil
}

// Reddit API OAuth response
type redditOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// Reddit API post listing response
type redditListingResponse struct {
	Data redditListingData `json:"data"`
}

type redditListingData struct {
	Children []redditPost `json:"children"`
	After    string       `json:"after"`
}

type redditPost struct {
	Data redditPostData `json:"data"`
}

type redditPostData struct {
	Title       string  `json:"title"`
	Selftext    string  `json:"selftext"`
	Subreddit   string  `json:"subreddit"`
	Upvotes     int     `json:"ups"`
	Downvotes   int     `json:"downs"`
	Score       int     `json:"score"`
	UpvoteRatio float64 `json:"upvote_ratio"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	LinkFlair   string  `json:"link_flair_text"`
}

// refreshAccessToken obtains a new Reddit OAuth token
func (rc *RedditCollector) refreshAccessToken(ctx context.Context) error {
	rc.Log().Debug("Refreshing Reddit OAuth token")

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://www.reddit.com/api/v1/access_token",
		strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return errors.Wrap(err, "create OAuth request")
	}

	req.SetBasicAuth(rc.clientID, rc.clientSecret)
	req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Reddit OAuth request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Reddit OAuth failed with status %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp redditOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return errors.Wrap(err, "decode OAuth response")
	}

	rc.accessToken = oauthResp.AccessToken
	rc.tokenExpiry = time.Now().Add(time.Duration(oauthResp.ExpiresIn) * time.Second)

	rc.Log().Debug("Reddit OAuth token refreshed", "expires_in", oauthResp.ExpiresIn)
	return nil
}

// collectPosts fetches recent Reddit posts mentioning a symbol
func (rc *RedditCollector) collectPosts(ctx context.Context, symbol string) ([]redditPostData, error) {
	var allPosts []redditPostData

	// Search across all tracked subreddits
	for _, subreddit := range rc.subreddits {
		// Get hot posts from subreddit (no search, just hot posts)
		// We'll filter by symbol in aggregation
		url := fmt.Sprintf("https://oauth.reddit.com/r/%s/hot?limit=50", subreddit)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, errors.Wrap(err, "create Reddit API request")
		}

		req.Header.Set("Authorization", "Bearer "+rc.accessToken)
		req.Header.Set("User-Agent", "Prometheus Trading Bot/1.0")

		resp, err := rc.httpClient.Do(req)
		if err != nil {
			return nil, errors.Wrap(err, "Reddit API request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			rc.Log().Warn("Reddit API rate limit reached")
			return allPosts, nil
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("Reddit API returned status %d: %s", resp.StatusCode, string(body))
		}

		var listing redditListingResponse
		if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
			return nil, errors.Wrap(err, "decode Reddit API response")
		}

		// Filter posts mentioning this symbol
		for _, post := range listing.Data.Children {
			if rc.postMentionsSymbol(post.Data, symbol) {
				allPosts = append(allPosts, post.Data)
			}
		}
	}

	rc.Log().Debug("Fetched Reddit posts",
		"symbol", symbol,
		"subreddits", len(rc.subreddits),
		"matching_posts", len(allPosts),
	)

	return allPosts, nil
}

// postMentionsSymbol checks if a post mentions the given symbol
func (rc *RedditCollector) postMentionsSymbol(post redditPostData, symbol string) bool {
	baseSymbol := strings.Split(symbol, "/")[0]
	searchText := strings.ToLower(post.Title + " " + post.Selftext)

	// Look for symbol variations
	variations := []string{
		strings.ToLower(baseSymbol),
		"$" + strings.ToLower(baseSymbol),
	}

	// Add full names
	switch baseSymbol {
	case "BTC":
		variations = append(variations, "bitcoin")
	case "ETH":
		variations = append(variations, "ethereum")
	case "SOL":
		variations = append(variations, "solana")
	case "BNB":
		variations = append(variations, "binance")
	}

	for _, variation := range variations {
		if strings.Contains(searchText, variation) {
			return true
		}
	}

	return false
}

// aggregateSentiment analyzes Reddit posts and creates aggregated sentiment
func (rc *RedditCollector) aggregateSentiment(symbol string, posts []redditPostData) *sentiment.SocialSentiment {
	if len(posts) == 0 {
		return nil
	}

	var positiveCount, negativeCount, neutralCount uint32
	var totalScore int

	for _, post := range posts {
		// Use upvote ratio as sentiment indicator
		// >0.7 = positive, <0.5 = negative, else neutral
		if post.UpvoteRatio >= 0.7 {
			positiveCount++
		} else if post.UpvoteRatio < 0.5 {
			negativeCount++
		} else {
			neutralCount++
		}

		totalScore += post.Score
	}

	totalMentions := uint32(len(posts))

	// Calculate sentiment score from upvote ratios
	sentimentScore := 0.0
	if totalMentions > 0 {
		sentimentScore = float64(int32(positiveCount)-int32(negativeCount)) / float64(totalMentions)
	}

	// Influencer sentiment (weight by score/engagement)
	influencerSentiment := sentimentScore
	if totalScore > 0 {
		weightedPositive := 0.0
		weightedTotal := 0.0
		for _, post := range posts {
			weight := float64(post.Score)
			weightedTotal += weight
			if post.UpvoteRatio >= 0.7 {
				weightedPositive += weight
			} else if post.UpvoteRatio < 0.5 {
				weightedPositive -= weight
			}
		}
		if weightedTotal > 0 {
			influencerSentiment = weightedPositive / weightedTotal
		}
	}

	return &sentiment.SocialSentiment{
		Platform:            "reddit",
		Symbol:              symbol,
		Timestamp:           time.Now(),
		Mentions:            totalMentions,
		SentimentScore:      sentimentScore,
		PositiveCount:       positiveCount,
		NegativeCount:       negativeCount,
		NeutralCount:        neutralCount,
		InfluencerSentiment: influencerSentiment,
		TrendingRank:        0, // Not tracked
	}
}
