package sentiment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"prometheus/internal/domain/sentiment"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// SentimentAggregationTool aggregates sentiment from multiple sources
// Combines Fear & Greed index with recent news headlines
type SentimentAggregationTool struct {
	sentimentRepo sentiment.Repository
	log           *logger.Logger
}

// NewSentimentAggregationTool creates a new sentiment aggregation tool
func NewSentimentAggregationTool(sentimentRepo sentiment.Repository) *SentimentAggregationTool {
	return &SentimentAggregationTool{
		sentimentRepo: sentimentRepo,
		log:           logger.Get().With("component", "sentiment_tool"),
	}
}

// Execute aggregates sentiment data and returns human-readable analysis
func (t *SentimentAggregationTool) Execute(ctx context.Context, symbol string) (string, error) {
	t.log.Debug("Executing sentiment aggregation", "symbol", symbol)

	// Get Fear & Greed Index
	fearGreed, err := t.sentimentRepo.GetLatestFearGreed(ctx)
	if err != nil {
		t.log.Warn("Failed to get Fear & Greed index", "error", err)
		fearGreed = nil // Continue without it
	}

	// Get recent news (last 24 hours)
	since := time.Now().Add(-24 * time.Hour)
	news, err := t.sentimentRepo.GetLatestNews(ctx, []string{symbol, "BTC/USDT", "crypto"}, 10)
	if err != nil {
		t.log.Warn("Failed to get recent news", "error", err)
		news = nil
	}

	// Get social sentiment if available
	socialSentiment, err := t.sentimentRepo.GetAggregatedSentiment(ctx, symbol, since)
	if err != nil {
		t.log.Debug("Social sentiment not available", "symbol", symbol)
		socialSentiment = nil
	}

	// Format the aggregated analysis
	return t.formatAnalysis(symbol, fearGreed, news, socialSentiment), nil
}

// formatAnalysis creates human-readable sentiment analysis
func (t *SentimentAggregationTool) formatAnalysis(
	symbol string,
	fearGreed *sentiment.FearGreedIndex,
	news []sentiment.News,
	social *sentiment.SocialSentiment,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("SENTIMENT ANALYSIS - %s\n\n", symbol))

	// Fear & Greed Index (market-wide)
	if fearGreed != nil {
		sb.WriteString(fmt.Sprintf("FEAR & GREED INDEX: %d/100 (%s)\n", fearGreed.Value, fearGreed.Rating))
		sb.WriteString(fmt.Sprintf("Interpretation: %s\n\n", t.interpretFearGreed(fearGreed.Value)))
	} else {
		sb.WriteString("FEAR & GREED INDEX: Data not available\n\n")
	}

	// Calculate composite sentiment score
	compositeScore, dataQuality := t.calculateCompositeScore(fearGreed, news, social)
	sb.WriteString(fmt.Sprintf("COMPOSITE SENTIMENT: %.2f (-1 to +1)\n", compositeScore))
	sb.WriteString(fmt.Sprintf("Overall: %s\n\n", t.classifySentiment(compositeScore)))

	// News sentiment
	if len(news) > 0 {
		avgNewsSentiment := t.calculateAverageNewsSentiment(news)
		sb.WriteString(fmt.Sprintf("NEWS SENTIMENT (24h): %.2f (%s)\n", avgNewsSentiment, t.classifySentiment(avgNewsSentiment)))
		sb.WriteString(fmt.Sprintf("Headlines analyzed: %d\n\n", len(news)))
		
		// Show top 3 recent headlines
		sb.WriteString("RECENT HEADLINES:\n")
		for i, article := range news {
			if i >= 3 {
				break
			}
			sentiment := t.classifySentiment(article.Sentiment)
			sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, sentiment, article.Title))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("NEWS SENTIMENT: No recent news available\n\n")
	}

	// Social sentiment
	if social != nil {
		sb.WriteString(fmt.Sprintf("SOCIAL SENTIMENT:\n"))
		sb.WriteString(fmt.Sprintf("- Platform: %s\n", social.Platform))
		sb.WriteString(fmt.Sprintf("- Mentions (24h): %d\n", social.Mentions))
		sb.WriteString(fmt.Sprintf("- Sentiment Score: %.2f\n", social.SentimentScore))
		if social.TrendingRank > 0 {
			sb.WriteString(fmt.Sprintf("- Trending Rank: #%d\n", social.TrendingRank))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("SOCIAL SENTIMENT: Data not available\n\n")
	}

	// Trading implications
	sb.WriteString("TRADING IMPLICATIONS:\n")
	implications := t.getTradingImplications(compositeScore, fearGreed)
	for _, impl := range implications {
		sb.WriteString(fmt.Sprintf("- %s\n", impl))
	}

	// Data quality note
	sb.WriteString(fmt.Sprintf("\nDATA QUALITY: %s\n", dataQuality))
	
	// Contrarian signal
	if fearGreed != nil {
		sb.WriteString(fmt.Sprintf("\nCONTRARIAN SIGNAL: %s\n", t.getContrarianSignal(fearGreed.Value)))
	}

	return sb.String()
}

// interpretFearGreed interprets Fear & Greed index value
func (t *SentimentAggregationTool) interpretFearGreed(value int) string {
	switch {
	case value <= 25:
		return "EXTREME FEAR - Market participants are very worried (potential buying opportunity)"
	case value <= 45:
		return "FEAR - Market is cautious but not panicking"
	case value <= 55:
		return "NEUTRAL - Market sentiment balanced"
	case value <= 75:
		return "GREED - Market is optimistic and buying (caution warranted)"
	default:
		return "EXTREME GREED - Market euphoria, high risk of correction (contrarian sell signal)"
	}
}

// classifySentiment classifies sentiment score
func (t *SentimentAggregationTool) classifySentiment(score float64) string {
	switch {
	case score >= 0.6:
		return "very bullish"
	case score >= 0.3:
		return "bullish"
	case score >= 0.1:
		return "slightly bullish"
	case score >= -0.1:
		return "neutral"
	case score >= -0.3:
		return "slightly bearish"
	case score >= -0.6:
		return "bearish"
	default:
		return "very bearish"
	}
}

// calculateAverageNewsSentiment calculates average sentiment from news
func (t *SentimentAggregationTool) calculateAverageNewsSentiment(news []sentiment.News) float64 {
	if len(news) == 0 {
		return 0
	}

	sum := 0.0
	for _, article := range news {
		sum += article.Sentiment
	}
	return sum / float64(len(news))
}

// calculateCompositeScore calculates weighted composite sentiment
func (t *SentimentAggregationTool) calculateCompositeScore(
	fearGreed *sentiment.FearGreedIndex,
	news []sentiment.News,
	social *sentiment.SocialSentiment,
) (float64, string) {
	score := 0.0
	weight := 0.0

	// Fear & Greed (weight: 0.4)
	if fearGreed != nil {
		// Convert 0-100 to -1 to +1
		fgScore := (float64(fearGreed.Value) - 50) / 50
		score += fgScore * 0.4
		weight += 0.4
	}

	// News sentiment (weight: 0.4)
	if len(news) > 0 {
		newsScore := t.calculateAverageNewsSentiment(news)
		score += newsScore * 0.4
		weight += 0.4
	}

	// Social sentiment (weight: 0.2)
	if social != nil {
		score += social.SentimentScore * 0.2
		weight += 0.2
	}

	if weight == 0 {
		return 0, "NO DATA"
	}

	// Normalize by actual weight
	finalScore := score / weight

	// Determine data quality
	quality := ""
	switch {
	case weight >= 0.9:
		quality = "EXCELLENT (all sources available)"
	case weight >= 0.7:
		quality = "GOOD (major sources available)"
	case weight >= 0.4:
		quality = "MODERATE (limited sources)"
	default:
		quality = "LOW (insufficient data)"
	}

	return finalScore, quality
}

// getTradingImplications provides trading implications based on sentiment
func (t *SentimentAggregationTool) getTradingImplications(
	compositeScore float64,
	fearGreed *sentiment.FearGreedIndex,
) []string {
	implications := []string{}

	// Composite sentiment implications
	if compositeScore > 0.5 {
		implications = append(implications, "Very bullish sentiment - confirms uptrend but watch for exhaustion")
		implications = append(implications, "Positive momentum favors long entries")
	} else if compositeScore > 0.2 {
		implications = append(implications, "Moderately bullish - supportive of long positions")
	} else if compositeScore < -0.5 {
		implications = append(implications, "Very bearish sentiment - confirms downtrend, caution on longs")
		implications = append(implications, "Consider waiting for sentiment improvement")
	} else if compositeScore < -0.2 {
		implications = append(implications, "Moderately bearish - headwinds for long positions")
	} else {
		implications = append(implications, "Neutral sentiment - price action and technicals more important")
	}

	// Fear & Greed specific implications
	if fearGreed != nil {
		if fearGreed.Value <= 20 {
			implications = append(implications, "Extreme fear = potential contrarian buy opportunity")
		} else if fearGreed.Value >= 80 {
			implications = append(implications, "Extreme greed = risk of top, consider taking profits")
		}
	}

	return implications
}

// getContrarianSignal provides contrarian trading signal
func (t *SentimentAggregationTool) getContrarianSignal(fearGreed int) string {
	switch {
	case fearGreed <= 20:
		return "STRONG BUY (extreme fear, crowds panicking)"
	case fearGreed <= 35:
		return "BUY (fear, good entry point)"
	case fearGreed >= 80:
		return "STRONG SELL (extreme greed, euphoria)"
	case fearGreed >= 65:
		return "SELL (greed, take profits)"
	default:
		return "NEUTRAL (no extreme emotion)"
	}
}

// GetDefinition returns the tool definition for ADK registration
func (t *SentimentAggregationTool) GetDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "get_sentiment_analysis",
		"description": "Aggregates sentiment data from multiple sources including Fear & Greed index, news headlines, and social media. Returns composite sentiment score, recent headlines, and trading implications.",
		"category":    "sentiment",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"symbol": map[string]interface{}{
					"type":        "string",
					"description": "Trading pair symbol (e.g., 'BTC/USDT', 'ETH/USDT')",
				},
			},
			"required": []string{"symbol"},
		},
	}
}

