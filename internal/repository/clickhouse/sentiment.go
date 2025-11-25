package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/domain/sentiment"
)

// Compile-time check
var _ sentiment.Repository = (*SentimentRepository)(nil)

// SentimentRepository implements sentiment.Repository using ClickHouse
type SentimentRepository struct {
	conn driver.Conn
}

// NewSentimentRepository creates a new sentiment repository
func NewSentimentRepository(conn driver.Conn) *SentimentRepository {
	return &SentimentRepository{conn: conn}
}

// InsertNews inserts a news article
func (r *SentimentRepository) InsertNews(ctx context.Context, news *sentiment.News) error {
	query := `
		INSERT INTO news (
			id, source, title, content, url, sentiment, symbols, published_at, collected_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	return r.conn.Exec(ctx, query,
		news.ID, news.Source, news.Title, news.Content, news.URL,
		news.Sentiment, news.Symbols, news.PublishedAt, news.CollectedAt,
	)
}

// GetLatestNews retrieves latest news for symbols
func (r *SentimentRepository) GetLatestNews(ctx context.Context, symbols []string, limit int) ([]sentiment.News, error) {
	var news []sentiment.News

	query := `
		SELECT * FROM news
		WHERE hasAny(symbols, $1)
		ORDER BY published_at DESC
		LIMIT $2`

	err := r.conn.Select(ctx, &news, query, symbols, limit)
	return news, err
}

// GetNewsSince retrieves news since a specific time
func (r *SentimentRepository) GetNewsSince(ctx context.Context, since time.Time) ([]sentiment.News, error) {
	var news []sentiment.News

	query := `
		SELECT * FROM news
		WHERE published_at >= $1
		ORDER BY published_at DESC`

	err := r.conn.Select(ctx, &news, query, since)
	return news, err
}

// InsertSocialSentiment inserts social sentiment data
func (r *SentimentRepository) InsertSocialSentiment(ctx context.Context, s *sentiment.SocialSentiment) error {
	query := `
		INSERT INTO social_sentiment (
			platform, symbol, timestamp, mentions, sentiment_score,
			positive_count, negative_count, neutral_count,
			influencer_sentiment, trending_rank
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	return r.conn.Exec(ctx, query,
		s.Platform, s.Symbol, s.Timestamp, s.Mentions, s.SentimentScore,
		s.PositiveCount, s.NegativeCount, s.NeutralCount,
		s.InfluencerSentiment, s.TrendingRank,
	)
}

// GetAggregatedSentiment retrieves aggregated sentiment for a symbol
func (r *SentimentRepository) GetAggregatedSentiment(ctx context.Context, symbol string, since time.Time) (*sentiment.SocialSentiment, error) {
	var s sentiment.SocialSentiment

	query := `
		SELECT
			platform,
			symbol,
			max(timestamp) as timestamp,
			sum(mentions) as mentions,
			avg(sentiment_score) as sentiment_score,
			sum(positive_count) as positive_count,
			sum(negative_count) as negative_count,
			sum(neutral_count) as neutral_count,
			avg(influencer_sentiment) as influencer_sentiment,
			0 as trending_rank
		FROM social_sentiment
		WHERE symbol = $1 AND timestamp >= $2
		GROUP BY platform, symbol`

	err := r.conn.QueryRow(ctx, query, symbol, since).ScanStruct(&s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetTrendingCoins retrieves trending coins by platform
func (r *SentimentRepository) GetTrendingCoins(ctx context.Context, platform string, limit int) ([]sentiment.SocialSentiment, error) {
	var sentiments []sentiment.SocialSentiment

	query := `
		SELECT * FROM social_sentiment
		WHERE platform = $1 AND trending_rank IS NOT NULL
		ORDER BY trending_rank ASC
		LIMIT $2`

	err := r.conn.Select(ctx, &sentiments, query, platform, limit)
	return sentiments, err
}

// InsertFearGreed inserts Fear & Greed index value
func (r *SentimentRepository) InsertFearGreed(ctx context.Context, index *sentiment.FearGreedIndex) error {
	// Store in PostgreSQL instead of ClickHouse (small dataset, needs ACID)
	// This would need a separate PG repository or be added to sentiment table in PG
	return nil // TODO: implement when we add fear_greed table to PostgreSQL
}

// GetLatestFearGreed retrieves the latest Fear & Greed index
func (r *SentimentRepository) GetLatestFearGreed(ctx context.Context) (*sentiment.FearGreedIndex, error) {
	// TODO: implement when we add fear_greed table to PostgreSQL
	return nil, nil
}
