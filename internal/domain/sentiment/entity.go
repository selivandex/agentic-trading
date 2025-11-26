package sentiment

import "time"

// News represents a news article with sentiment
type News struct {
	ID          string    `ch:"id"`
	Source      string    `ch:"source"`
	Title       string    `ch:"title"`
	Content     string    `ch:"content"`
	URL         string    `ch:"url"`
	Sentiment   float64   `ch:"sentiment"` // -1 to 1
	Symbols     []string  `ch:"symbols"`
	PublishedAt time.Time `ch:"published_at"`
	CollectedAt time.Time `ch:"collected_at"`
}

// SocialSentiment represents aggregated social media sentiment
type SocialSentiment struct {
	Platform            string    `ch:"platform"` // twitter, reddit
	Symbol              string    `ch:"symbol"`
	Timestamp           time.Time `ch:"timestamp"`
	Mentions            uint32    `ch:"mentions"`
	SentimentScore      float64   `ch:"sentiment_score"`
	PositiveCount       uint32    `ch:"positive_count"`
	NegativeCount       uint32    `ch:"negative_count"`
	NeutralCount        uint32    `ch:"neutral_count"`
	InfluencerSentiment float64   `ch:"influencer_sentiment"`
	TrendingRank        uint16    `ch:"trending_rank"`
}

// FearGreedIndex represents Fear & Greed index value
type FearGreedIndex struct {
	Timestamp time.Time `db:"timestamp"`
	Value     int       `db:"value"`  // 0-100
	Rating    string    `db:"rating"` // extreme_fear, fear, neutral, greed, extreme_greed
}
