package sentiment

import (
	"context"
	"time"
)

// Repository defines the interface for sentiment data access (ClickHouse)
type Repository interface {
	// News operations
	InsertNews(ctx context.Context, news *News) error
	GetLatestNews(ctx context.Context, symbols []string, limit int) ([]News, error)
	GetNewsSince(ctx context.Context, since time.Time) ([]News, error)

	// Social sentiment operations
	InsertSocialSentiment(ctx context.Context, sentiment *SocialSentiment) error
	GetAggregatedSentiment(ctx context.Context, symbol string, since time.Time) (*SocialSentiment, error)
	GetTrendingCoins(ctx context.Context, platform string, limit int) ([]SocialSentiment, error)

	// Fear & Greed operations
	InsertFearGreed(ctx context.Context, index *FearGreedIndex) error
	GetLatestFearGreed(ctx context.Context) (*FearGreedIndex, error)
}
