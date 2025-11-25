<!-- @format -->

# Step 3.5: External Data Sources Integration

## Overview
Интеграция внешних источников данных для sentiment, on-chain, derivatives, macro analysis.

**Цель**: Агенты получают доступ к новостям, соцсетям, blockchain метрикам, опционам, макро данным.

**Время**: 7-10 дней

---

## 3.5.1 Data Sources Overview

### Available Data Sources

| Category      | Providers                                   | Data Type                                |
| ------------- | ------------------------------------------- | ---------------------------------------- |
| **News**      | CoinDesk, CoinTelegraph, The Block          | Crypto news, articles                    |
| **Sentiment** | LunarCrush, Santiment, Twitter, Reddit      | Social sentiment, mentions, trending     |
| **On-Chain**  | Glassnode, Santiment                        | Whale moves, MVRV, SOPR, exchange flows  |
| **Derivatives** | Deribit, Laevitas, Greeks.live            | Options OI, max pain, gamma exposure     |
| **Liquidations** | Coinglass, Hyblock                       | Real-time liquidations, heatmaps         |
| **Macro**     | Investing.com, FRED, CME FedWatch           | Economic calendar, Fed rates, CPI        |
| **Correlations** | Yahoo Finance, TradingView              | BTC vs SPX/DXY/Gold                      |

---

## 3.5.2 News Sources

### News Interface
**File**: `internal/adapters/datasources/news/interface.go`

```go
package news

import (
    "context"
    "time"
)

type Article struct {
    ID          string
    Title       string
    Content     string
    URL         string
    Source      string
    Author      string
    PublishedAt time.Time
    Symbols     []string  // Extracted symbols (BTC, ETH, etc.)
    Tags        []string
    ImageURL    string
}

type NewsSource interface {
    Name() string
    FetchLatest(ctx context.Context, limit int) ([]Article, error)
    FetchSince(ctx context.Context, since time.Time) ([]Article, error)
    Search(ctx context.Context, query string, limit int) ([]Article, error)
}
```

### CoinDesk Provider
**File**: `internal/adapters/datasources/news/coindesk/provider.go`

```go
package coindesk

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "prometheus/internal/adapters/datasources/news"
)

type Provider struct {
    httpClient *http.Client
    baseURL    string
}

func New() news.NewsSource {
    return &Provider{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    "https://www.coindesk.com/arc/outboundfeeds/rss/",
    }
}

func (p *Provider) Name() string {
    return "coindesk"
}

func (p *Provider) FetchLatest(ctx context.Context, limit int) ([]news.Article, error) {
    // Parse RSS feed
    resp, err := p.httpClient.Get(p.baseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch RSS: %w", err)
    }
    defer resp.Body.Close()

    // Parse XML/RSS to articles
    articles := parseRSSFeed(resp.Body, limit)
    
    // Extract symbols from content
    for i := range articles {
        articles[i].Symbols = extractSymbols(articles[i].Content)
        articles[i].Source = p.Name()
    }

    return articles, nil
}

func (p *Provider) FetchSince(ctx context.Context, since time.Time) ([]news.Article, error) {
    articles, err := p.FetchLatest(ctx, 100)
    if err != nil {
        return nil, err
    }

    // Filter by date
    var filtered []news.Article
    for _, article := range articles {
        if article.PublishedAt.After(since) {
            filtered = append(filtered, article)
        }
    }

    return filtered, nil
}

func (p *Provider) Search(ctx context.Context, query string, limit int) ([]news.Article, error) {
    // CoinDesk search API (if available)
    return nil, fmt.Errorf("search not implemented")
}

// Helper: Extract crypto symbols from text
func extractSymbols(text string) []string {
    symbols := []string{}
    cryptoKeywords := map[string]bool{
        "Bitcoin": true, "BTC": true,
        "Ethereum": true, "ETH": true,
        "Solana": true, "SOL": true,
        // ... add more
    }

    for keyword := range cryptoKeywords {
        if strings.Contains(text, keyword) {
            symbols = append(symbols, keyword)
        }
    }

    return symbols
}
```

---

## 3.5.3 Sentiment Sources

### Sentiment Interface
**File**: `internal/adapters/datasources/sentiment/interface.go`

```go
package sentiment

import (
    "context"
    "time"
)

type SentimentData struct {
    Symbol           string
    Platform         string
    Timestamp        time.Time
    Score            float64  // -1 to 1
    Mentions         int
    PositiveCount    int
    NegativeCount    int
    NeutralCount     int
    TrendingRank     int
    InfluencerScore  float64
}

type SentimentSource interface {
    Name() string
    GetSentiment(ctx context.Context, symbol string) (*SentimentData, error)
    GetTrending(ctx context.Context, limit int) ([]TrendingAsset, error)
}

type TrendingAsset struct {
    Symbol     string
    Rank       int
    Score      float64
    Change24h  float64
}
```

### LunarCrush Provider
**File**: `internal/adapters/datasources/sentiment/lunarcrush/provider.go`

```go
package lunarcrush

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "prometheus/internal/adapters/datasources/sentiment"
)

type Provider struct {
    apiKey     string
    httpClient *http.Client
    baseURL    string
}

func New(apiKey string) sentiment.SentimentSource {
    return &Provider{
        apiKey:     apiKey,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    "https://lunarcrush.com/api4/public",
    }
}

func (p *Provider) Name() string {
    return "lunarcrush"
}

func (p *Provider) GetSentiment(ctx context.Context, symbol string) (*sentiment.SentimentData, error) {
    url := fmt.Sprintf("%s/coins/%s/v1", p.baseURL, symbol)
    
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data struct {
            Symbol            string  `json:"symbol"`
            SocialScore       float64 `json:"social_score"`
            SocialMentions    int     `json:"social_mentions"`
            SentimentPositive int     `json:"sentiment_positive"`
            SentimentNegative int     `json:"sentiment_negative"`
            SentimentNeutral  int     `json:"sentiment_neutral"`
        } `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    // Normalize score to -1..1
    normalizedScore := (result.Data.SocialScore - 50) / 50

    return &sentiment.SentimentData{
        Symbol:        symbol,
        Platform:      "lunarcrush",
        Timestamp:     time.Now(),
        Score:         normalizedScore,
        Mentions:      result.Data.SocialMentions,
        PositiveCount: result.Data.SentimentPositive,
        NegativeCount: result.Data.SentimentNegative,
        NeutralCount:  result.Data.SentimentNeutral,
    }, nil
}

func (p *Provider) GetTrending(ctx context.Context, limit int) ([]sentiment.TrendingAsset, error) {
    url := fmt.Sprintf("%s/coins/list/v2?sort=social_volume_24h&limit=%d", p.baseURL, limit)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data []struct {
            Symbol string  `json:"symbol"`
            Rank   int     `json:"rank"`
            Score  float64 `json:"galaxy_score"`
        } `json:"data"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    trending := make([]sentiment.TrendingAsset, len(result.Data))
    for i, item := range result.Data {
        trending[i] = sentiment.TrendingAsset{
            Symbol: item.Symbol,
            Rank:   item.Rank,
            Score:  item.Score,
        }
    }

    return trending, nil
}
```

---

## 3.5.4 On-Chain Sources

### On-Chain Interface
**File**: `internal/adapters/datasources/onchain/interface.go`

```go
package onchain

import (
    "context"
    "time"
)

type WhaleMovement struct {
    TxHash      string
    From        string
    To          string
    Amount      float64
    AmountUSD   float64
    Symbol      string
    Timestamp   time.Time
    IsExchange  bool
    Direction   string  // in/out
}

type ExchangeFlow struct {
    Exchange    string
    Symbol      string
    Timestamp   time.Time
    Inflow      float64
    Outflow     float64
    NetFlow     float64
    NetFlowUSD  float64
}

type Metric struct {
    Symbol    string
    Metric    string  // mvrv, sopr, nvt
    Value     float64
    Timestamp time.Time
}

type OnChainSource interface {
    Name() string
    GetWhaleMovements(ctx context.Context, symbol string, minUSD float64, limit int) ([]WhaleMovement, error)
    GetExchangeFlows(ctx context.Context, symbol string, period string) ([]ExchangeFlow, error)
    GetMetric(ctx context.Context, symbol string, metric string) (*Metric, error)
}
```

### Glassnode Provider
**File**: `internal/adapters/datasources/onchain/glassnode/provider.go`

```go
package glassnode

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "prometheus/internal/adapters/datasources/onchain"
)

type Provider struct {
    apiKey     string
    httpClient *http.Client
    baseURL    string
}

func New(apiKey string) onchain.OnChainSource {
    return &Provider{
        apiKey:     apiKey,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    "https://api.glassnode.com/v1/metrics",
    }
}

func (p *Provider) Name() string {
    return "glassnode"
}

func (p *Provider) GetWhaleMovements(ctx context.Context, symbol string, minUSD float64, limit int) ([]onchain.WhaleMovement, error) {
    // Glassnode whale tracking endpoint
    url := fmt.Sprintf("%s/transactions/transfers_volume_whales_min_%d?a=%s&api_key=%s",
        p.baseURL, int(minUSD/1000000), symbol, p.apiKey)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var data []struct {
        T int64   `json:"t"`  // Timestamp
        V float64 `json:"v"`  // Value
    }

    json.NewDecoder(resp.Body).Decode(&data)

    movements := make([]onchain.WhaleMovement, 0)
    // Transform to WhaleMovement struct
    // ... implementation

    return movements, nil
}

func (p *Provider) GetExchangeFlows(ctx context.Context, symbol string, period string) ([]onchain.ExchangeFlow, error) {
    // Glassnode exchange flows
    url := fmt.Sprintf("%s/transactions/transfers_volume_exchanges_net?a=%s&api_key=%s&i=%s",
        p.baseURL, symbol, p.apiKey, period)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var data []struct {
        T int64   `json:"t"`
        V float64 `json:"v"`
    }

    json.NewDecoder(resp.Body).Decode(&data)

    flows := make([]onchain.ExchangeFlow, len(data))
    for i, item := range data {
        flows[i] = onchain.ExchangeFlow{
            Symbol:    symbol,
            Timestamp: time.Unix(item.T, 0),
            NetFlow:   item.V,
        }
    }

    return flows, nil
}

func (p *Provider) GetMetric(ctx context.Context, symbol string, metric string) (*onchain.Metric, error) {
    // Get MVRV, SOPR, NVT, etc.
    url := fmt.Sprintf("%s/indicators/%s?a=%s&api_key=%s",
        p.baseURL, metric, symbol, p.apiKey)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var data []struct {
        T int64   `json:"t"`
        V float64 `json:"v"`
    }

    json.NewDecoder(resp.Body).Decode(&data)

    if len(data) == 0 {
        return nil, fmt.Errorf("no data")
    }

    latest := data[len(data)-1]

    return &onchain.Metric{
        Symbol:    symbol,
        Metric:    metric,
        Value:     latest.V,
        Timestamp: time.Unix(latest.T, 0),
    }, nil
}
```

---

## 3.5.5 Derivatives Sources

### Derivatives Interface
**File**: `internal/adapters/datasources/derivatives/interface.go`

```go
package derivatives

import (
    "context"
    "time"
)

type OptionsSnapshot struct {
    Symbol        string
    Timestamp     time.Time
    CallOI        float64
    PutOI         float64
    TotalOI       float64
    PutCallRatio  float64
    MaxPainPrice  float64
    GammaExposure float64
    GammaFlip     float64
    IVIndex       float64
}

type OptionsFlow struct {
    ID        string
    Timestamp time.Time
    Symbol    string
    Side      string  // call/put
    Strike    float64
    Expiry    time.Time
    Premium   float64
    Size      int
    Spot      float64
    Sentiment string  // bullish/bearish
}

type DerivativesSource interface {
    Name() string
    GetOptionsSnapshot(ctx context.Context, symbol string) (*OptionsSnapshot, error)
    GetOptionsFlow(ctx context.Context, symbol string, limit int) ([]OptionsFlow, error)
    GetMaxPain(ctx context.Context, symbol string, expiry time.Time) (float64, error)
}
```

### Deribit Provider
**File**: `internal/adapters/datasources/derivatives/deribit/provider.go`

```go
package deribit

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "prometheus/internal/adapters/datasources/derivatives"
)

type Provider struct {
    httpClient *http.Client
    baseURL    string
}

func New() derivatives.DerivativesSource {
    return &Provider{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    "https://www.deribit.com/api/v2/public",
    }
}

func (p *Provider) Name() string {
    return "deribit"
}

func (p *Provider) GetOptionsSnapshot(ctx context.Context, symbol string) (*derivatives.OptionsSnapshot, error) {
    // Deribit uses BTC-USD format
    deribitSymbol := convertSymbol(symbol)

    url := fmt.Sprintf("%s/get_book_summary_by_currency?currency=%s&kind=option", p.baseURL, deribitSymbol)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Result []struct {
            InstrumentName string  `json:"instrument_name"`
            OpenInterest   float64 `json:"open_interest"`
            Volume         float64 `json:"volume"`
            MarkIV         float64 `json:"mark_iv"`
        } `json:"result"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    // Aggregate call/put OI
    var callOI, putOI float64
    for _, instrument := range result.Result {
        if strings.Contains(instrument.InstrumentName, "-C") {
            callOI += instrument.OpenInterest
        } else if strings.Contains(instrument.InstrumentName, "-P") {
            putOI += instrument.OpenInterest
        }
    }

    snapshot := &derivatives.OptionsSnapshot{
        Symbol:       symbol,
        Timestamp:    time.Now(),
        CallOI:       callOI,
        PutOI:        putOI,
        TotalOI:      callOI + putOI,
        PutCallRatio: putOI / callOI,
    }

    return snapshot, nil
}

func (p *Provider) GetMaxPain(ctx context.Context, symbol string, expiry time.Time) (float64, error) {
    // Calculate max pain from open interest distribution
    // Implementation depends on Deribit API
    return 0, fmt.Errorf("not implemented")
}

func convertSymbol(symbol string) string {
    // BTC/USDT -> BTC
    parts := strings.Split(symbol, "/")
    return parts[0]
}
```

---

## 3.5.6 Liquidation Sources

### Liquidation Interface
**File**: `internal/adapters/datasources/liquidations/interface.go`

```go
package liquidations

import (
    "context"
    "time"
)

type Liquidation struct {
    Exchange  string
    Symbol    string
    Timestamp time.Time
    Side      string  // long/short
    Price     float64
    Quantity  float64
    Value     float64  // USD
}

type LiquidationHeatmap struct {
    Symbol    string
    Timestamp time.Time
    Levels    []LiquidationLevel
}

type LiquidationLevel struct {
    Price         float64
    LongLiqValue  float64
    ShortLiqValue float64
    Leverage      int
}

type LiquidationSource interface {
    Name() string
    GetRecentLiquidations(ctx context.Context, symbol string, limit int) ([]Liquidation, error)
    GetLiquidationHeatmap(ctx context.Context, symbol string) (*LiquidationHeatmap, error)
    StreamLiquidations(ctx context.Context, symbol string) (<-chan Liquidation, error)
}
```

### Coinglass Provider
**File**: `internal/adapters/datasources/liquidations/coinglass/provider.go`

```go
package coinglass

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "prometheus/internal/adapters/datasources/liquidations"
)

type Provider struct {
    apiKey     string
    httpClient *http.Client
    baseURL    string
}

func New(apiKey string) liquidations.LiquidationSource {
    return &Provider{
        apiKey:     apiKey,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    "https://open-api.coinglass.com/public/v2",
    }
}

func (p *Provider) Name() string {
    return "coinglass"
}

func (p *Provider) GetRecentLiquidations(ctx context.Context, symbol string, limit int) ([]liquidations.Liquidation, error) {
    url := fmt.Sprintf("%s/liquidation?symbol=%s&limit=%d", p.baseURL, symbol, limit)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("CG-API-KEY", p.apiKey)

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data []struct {
            Exchange  string  `json:"exchange"`
            Symbol    string  `json:"symbol"`
            Side      string  `json:"side"`
            Price     float64 `json:"price"`
            Quantity  float64 `json:"quantity"`
            Value     float64 `json:"value"`
            Timestamp int64   `json:"timestamp"`
        } `json:"data"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    liqs := make([]liquidations.Liquidation, len(result.Data))
    for i, item := range result.Data {
        liqs[i] = liquidations.Liquidation{
            Exchange:  item.Exchange,
            Symbol:    item.Symbol,
            Timestamp: time.Unix(item.Timestamp/1000, 0),
            Side:      item.Side,
            Price:     item.Price,
            Quantity:  item.Quantity,
            Value:     item.Value,
        }
    }

    return liqs, nil
}

func (p *Provider) GetLiquidationHeatmap(ctx context.Context, symbol string) (*liquidations.LiquidationHeatmap, error) {
    url := fmt.Sprintf("%s/liquidation_heatmap?symbol=%s", p.baseURL, symbol)

    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("CG-API-KEY", p.apiKey)

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse heatmap data
    // ... implementation

    return &liquidations.LiquidationHeatmap{
        Symbol:    symbol,
        Timestamp: time.Now(),
        Levels:    []liquidations.LiquidationLevel{},
    }, nil
}
```

---

## 3.5.7 Macro Sources

### Macro Interface
**File**: `internal/adapters/datasources/macro/interface.go`

```go
package macro

import (
    "context"
    "time"
)

type EconomicEvent struct {
    ID          string
    EventType   string  // cpi, fomc, nfp, gdp
    Title       string
    Country     string
    Currency    string
    EventTime   time.Time
    Previous    string
    Forecast    string
    Actual      string
    Impact      string  // low/medium/high
}

type FedRate struct {
    Rate      float64
    Date      time.Time
    NextMeet  time.Time
}

type MacroSource interface {
    Name() string
    GetEconomicCalendar(ctx context.Context, days int) ([]EconomicEvent, error)
    GetFedRate(ctx context.Context) (*FedRate, error)
    GetCPI(ctx context.Context) (float64, error)
}
```

---

## 3.5.8 Workers for Data Sources

### News Collector Worker
**File**: `internal/workers/sentiment/news_collector.go`

```go
package sentiment

import (
    "context"
    "time"
    "prometheus/internal/adapters/datasources/news"
    "prometheus/internal/workers"
    "prometheus/pkg/logger"
)

type NewsCollector struct {
    workers.BaseWorker
    sources []news.NewsSource
    repo    SentimentRepository
    log     *logger.Logger
}

func NewNewsCollector(
    sources []news.NewsSource,
    repo SentimentRepository,
    interval time.Duration,
) *NewsCollector {
    return &NewsCollector{
        BaseWorker: workers.BaseWorker{
            Name:     "news_collector",
            Interval: interval,
            Enabled:  true,
        },
        sources: sources,
        repo:    repo,
        log:     logger.Get().With("worker", "news_collector"),
    }
}

func (w *NewsCollector) Run(ctx context.Context) error {
    for _, source := range w.sources {
        articles, err := source.FetchLatest(ctx, 50)
        if err != nil {
            w.log.Warnf("Failed to fetch from %s: %v", source.Name(), err)
            continue
        }

        w.log.Infof("Fetched %d articles from %s", len(articles), source.Name())

        for _, article := range articles {
            // Analyze sentiment (simple keyword-based or use AI)
            sentimentScore := analyzeSentiment(article.Content)

            // Save to ClickHouse
            if err := w.repo.InsertNews(ctx, &NewsRecord{
                Source:      source.Name(),
                Title:       article.Title,
                Content:     article.Content,
                URL:         article.URL,
                Sentiment:   sentimentScore,
                Symbols:     article.Symbols,
                PublishedAt: article.PublishedAt,
            }); err != nil {
                w.log.Errorf("Failed to insert news: %v", err)
            }
        }
    }

    return nil
}

func analyzeSentiment(text string) float64 {
    // Simple keyword-based sentiment analysis
    positiveWords := []string{"bullish", "surge", "rally", "breakout", "growth"}
    negativeWords := []string{"bearish", "crash", "dump", "decline", "fear"}

    score := 0.0
    textLower := strings.ToLower(text)

    for _, word := range positiveWords {
        if strings.Contains(textLower, word) {
            score += 0.1
        }
    }

    for _, word := range negativeWords {
        if strings.Contains(textLower, word) {
            score -= 0.1
        }
    }

    // Clamp to -1..1
    if score > 1 {
        score = 1
    }
    if score < -1 {
        score = -1
    }

    return score
}
```

### Sentiment Collector Worker
**File**: `internal/workers/sentiment/sentiment_collector.go`

```go
package sentiment

import (
    "context"
    "time"
    "prometheus/internal/adapters/datasources/sentiment"
    "prometheus/internal/workers"
    "prometheus/pkg/logger"
)

type SentimentCollector struct {
    workers.BaseWorker
    sources []sentiment.SentimentSource
    repo    SentimentRepository
    symbols []string
    log     *logger.Logger
}

func NewSentimentCollector(
    sources []sentiment.SentimentSource,
    repo SentimentRepository,
    symbols []string,
    interval time.Duration,
) *SentimentCollector {
    return &SentimentCollector{
        BaseWorker: workers.BaseWorker{
            Name:     "sentiment_collector",
            Interval: interval,
            Enabled:  true,
        },
        sources: sources,
        repo:    repo,
        symbols: symbols,
        log:     logger.Get().With("worker", "sentiment_collector"),
    }
}

func (w *SentimentCollector) Run(ctx context.Context) error {
    for _, source := range w.sources {
        for _, symbol := range w.symbols {
            data, err := source.GetSentiment(ctx, symbol)
            if err != nil {
                w.log.Warnf("Failed to get sentiment from %s for %s: %v", source.Name(), symbol, err)
                continue
            }

            // Save to ClickHouse
            if err := w.repo.InsertSentiment(ctx, data); err != nil {
                w.log.Errorf("Failed to insert sentiment: %v", err)
            } else {
                w.log.Debugf("✓ Sentiment for %s from %s: score=%.2f, mentions=%d",
                    symbol, source.Name(), data.Score, data.Mentions)
            }
        }
    }

    return nil
}
```

---

## 3.5.9 ClickHouse Schemas for Data Sources

### Sentiment & News Schema
**File**: `migrations/clickhouse/002_sentiment.up.sql`

```sql
-- News
CREATE TABLE IF NOT EXISTS news (
    id UUID DEFAULT generateUUIDv4(),
    source LowCardinality(String),
    title String,
    content String,
    url String,
    sentiment Float32,
    symbols Array(LowCardinality(String)),
    published_at DateTime64(3),
    collected_at DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(published_at)
ORDER BY (published_at, source)
TTL published_at + INTERVAL 1 YEAR;

-- Social Sentiment
CREATE TABLE IF NOT EXISTS social_sentiment (
    platform LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    score Float32,
    mentions UInt32,
    positive_count UInt32,
    negative_count UInt32,
    neutral_count UInt32,
    influencer_score Float32,
    trending_rank Nullable(UInt16)
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (platform, symbol, timestamp)
TTL timestamp + INTERVAL 90 DAY;

-- Sentiment aggregation view
CREATE MATERIALIZED VIEW IF NOT EXISTS sentiment_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (symbol, hour)
AS SELECT
    symbol,
    toStartOfHour(timestamp) as hour,
    avg(score) as avg_sentiment,
    sum(mentions) as total_mentions,
    sum(positive_count) as total_positive,
    sum(negative_count) as total_negative
FROM social_sentiment
GROUP BY symbol, hour;
```

### On-Chain & Derivatives Schema
**File**: `migrations/clickhouse/003_onchain_derivatives.up.sql`

```sql
-- Whale Movements
CREATE TABLE IF NOT EXISTS whale_movements (
    tx_hash String,
    symbol LowCardinality(String),
    from_address String,
    to_address String,
    amount Float64,
    amount_usd Float64,
    timestamp DateTime64(3),
    is_exchange Boolean,
    direction LowCardinality(String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (symbol, timestamp)
TTL timestamp + INTERVAL 180 DAY;

-- Exchange Flows
CREATE TABLE IF NOT EXISTS exchange_flows (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    inflow Float64,
    outflow Float64,
    net_flow Float64,
    net_flow_usd Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (exchange, symbol, timestamp)
TTL timestamp + INTERVAL 1 YEAR;

-- On-Chain Metrics
CREATE TABLE IF NOT EXISTS onchain_metrics (
    symbol LowCardinality(String),
    metric LowCardinality(String),
    value Float64,
    timestamp DateTime64(3)
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (symbol, metric, timestamp)
TTL timestamp + INTERVAL 2 YEAR;

-- Options Snapshots
CREATE TABLE IF NOT EXISTS options_snapshots (
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    call_oi Float64,
    put_oi Float64,
    total_oi Float64,
    put_call_ratio Float64,
    max_pain_price Float64,
    gamma_exposure Float64,
    iv_index Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (symbol, timestamp)
TTL timestamp + INTERVAL 6 MONTH;

-- Liquidations
CREATE TABLE IF NOT EXISTS liquidations (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    side LowCardinality(String),
    price Float64,
    quantity Float64,
    value Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (exchange, symbol, timestamp)
TTL timestamp + INTERVAL 30 DAY;

-- Macro Events
CREATE TABLE IF NOT EXISTS macro_events (
    id String,
    event_type LowCardinality(String),
    title String,
    country LowCardinality(String),
    event_time DateTime64(3),
    previous String,
    forecast String,
    actual String,
    impact LowCardinality(String),
    collected_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (event_type, event_time)
TTL event_time + INTERVAL 2 YEAR;
```

---

## 3.5.10 Tools Using Data Sources

### Get News Tool
**File**: `internal/tools/sentiment/get_news.go`

```go
package sentiment

import (
    "context"
    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/functiontool"
    "prometheus/internal/adapters/datasources/news"
)

type GetNewsArgs struct {
    Symbol string `json:"symbol" jsonschema:"description=Crypto symbol (BTC/ETH/etc)"`
    Limit  int    `json:"limit" jsonschema:"description=Number of articles (default 10)"`
}

type GetNewsResult struct {
    Articles []NewsArticle `json:"articles"`
}

type NewsArticle struct {
    Title       string  `json:"title"`
    Source      string  `json:"source"`
    Sentiment   float64 `json:"sentiment"`
    PublishedAt string  `json:"published_at"`
    URL         string  `json:"url"`
}

func NewGetNewsTool(sources []news.NewsSource) tool.Tool {
    return functiontool.New(
        "get_news",
        "Get latest cryptocurrency news with sentiment analysis",
        func(ctx tool.Context, args GetNewsArgs) (GetNewsResult, error) {
            limit := args.Limit
            if limit == 0 {
                limit = 10
            }

            allArticles := []NewsArticle{}

            for _, source := range sources {
                articles, err := source.FetchLatest(context.Background(), limit)
                if err != nil {
                    continue
                }

                for _, article := range articles {
                    // Filter by symbol if specified
                    if args.Symbol != "" {
                        hasSymbol := false
                        for _, sym := range article.Symbols {
                            if sym == args.Symbol {
                                hasSymbol = true
                                break
                            }
                        }
                        if !hasSymbol {
                            continue
                        }
                    }

                    allArticles = append(allArticles, NewsArticle{
                        Title:       article.Title,
                        Source:      article.Source,
                        Sentiment:   analyzeSentiment(article.Content),
                        PublishedAt: article.PublishedAt.Format("2006-01-02 15:04"),
                        URL:         article.URL,
                    })

                    if len(allArticles) >= limit {
                        break
                    }
                }
            }

            return GetNewsResult{Articles: allArticles}, nil
        },
    )
}
```

### Get Whale Movements Tool
**File**: `internal/tools/onchain/get_whale_movements.go`

```go
package onchain

import (
    "context"
    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/functiontool"
    "prometheus/internal/adapters/datasources/onchain"
)

type GetWhaleMovementsArgs struct {
    Symbol string  `json:"symbol" jsonschema:"required"`
    MinUSD float64 `json:"min_usd" jsonschema:"description=Minimum transaction size in USD (default 1000000)"`
    Limit  int     `json:"limit" jsonschema:"description=Max results (default 20)"`
}

type GetWhaleMovementsResult struct {
    Movements []WhaleMovement `json:"movements"`
}

type WhaleMovement struct {
    Amount    float64 `json:"amount"`
    AmountUSD float64 `json:"amount_usd"`
    Direction string  `json:"direction"`  // to_exchange/from_exchange
    Timestamp string  `json:"timestamp"`
}

func NewGetWhaleMovementsTool(sources []onchain.OnChainSource) tool.Tool {
    return functiontool.New(
        "get_whale_movements",
        "Get large wallet transfers (whale movements) that may indicate market sentiment",
        func(ctx tool.Context, args GetWhaleMovementsArgs) (GetWhaleMovementsResult, error) {
            minUSD := args.MinUSD
            if minUSD == 0 {
                minUSD = 1000000  // $1M default
            }

            limit := args.Limit
            if limit == 0 {
                limit = 20
            }

            allMovements := []WhaleMovement{}

            for _, source := range sources {
                movements, err := source.GetWhaleMovements(context.Background(), args.Symbol, minUSD, limit)
                if err != nil {
                    continue
                }

                for _, m := range movements {
                    allMovements = append(allMovements, WhaleMovement{
                        Amount:    m.Amount,
                        AmountUSD: m.AmountUSD,
                        Direction: m.Direction,
                        Timestamp: m.Timestamp.Format("2006-01-02 15:04"),
                    })
                }
            }

            return GetWhaleMovementsResult{Movements: allMovements}, nil
        },
    )
}
```

---

## 3.5.11 Update Configuration

### Add API Keys for Data Sources
**File**: `internal/adapters/config/config.go` (additions)

```go
type DataSourcesConfig struct {
    // News
    CoinDeskAPIKey      string `envconfig:"COINDESK_API_KEY"`
    CoinTelegraphAPIKey string `envconfig:"COINTELEGRAPH_API_KEY"`
    TheBlockAPIKey      string `envconfig:"THEBLOCK_API_KEY"`

    // Sentiment
    LunarCrushAPIKey string `envconfig:"LUNARCRUSH_API_KEY"`
    SantimentAPIKey  string `envconfig:"SANTIMENT_API_KEY"`
    TwitterAPIKey    string `envconfig:"TWITTER_API_KEY"`
    RedditAPIKey     string `envconfig:"REDDIT_API_KEY"`

    // On-Chain
    GlassnodeAPIKey string `envconfig:"GLASSNODE_API_KEY"`

    // Derivatives
    DeribitAPIKey   string `envconfig:"DERIBIT_API_KEY"`
    LaevitasAPIKey  string `envconfig:"LAEVITAS_API_KEY"`

    // Liquidations
    CoinglassAPIKey string `envconfig:"COINGLASS_API_KEY"`
    HyblockAPIKey   string `envconfig:"HYBLOCK_API_KEY"`

    // Macro
    InvestingAPIKey string `envconfig:"INVESTING_API_KEY"`
    FREDAPIKey      string `envconfig:"FRED_API_KEY"`
}

type Config struct {
    App           AppConfig
    Postgres      PostgresConfig
    ClickHouse    ClickHouseConfig
    Redis         RedisConfig
    Kafka         KafkaConfig
    Telegram      TelegramConfig
    AI            AIConfig
    Crypto        CryptoConfig
    MarketData    MarketDataConfig
    DataSources   DataSourcesConfig  // NEW!
    ErrorTracking ErrorTrackingConfig
}
```

### Update .env.example
```bash
# Data Sources - News
COINDESK_API_KEY=
COINTELEGRAPH_API_KEY=
THEBLOCK_API_KEY=

# Data Sources - Sentiment
LUNARCRUSH_API_KEY=your_lunarcrush_key
SANTIMENT_API_KEY=your_santiment_key
TWITTER_API_KEY=your_twitter_bearer_token
REDDIT_API_KEY=your_reddit_key

# Data Sources - On-Chain
GLASSNODE_API_KEY=your_glassnode_key

# Data Sources - Derivatives
DERIBIT_API_KEY=
LAEVITAS_API_KEY=

# Data Sources - Liquidations
COINGLASS_API_KEY=your_coinglass_key
HYBLOCK_API_KEY=

# Data Sources - Macro
INVESTING_API_KEY=
FRED_API_KEY=your_fred_key
```

---

## 3.5.12 Initialize Data Sources in Main

### Update cmd/main.go
```go
package main

import (
    "prometheus/internal/adapters/datasources/news/coindesk"
    "prometheus/internal/adapters/datasources/news/cointelegraph"
    "prometheus/internal/adapters/datasources/sentiment/lunarcrush"
    "prometheus/internal/adapters/datasources/sentiment/santiment"
    "prometheus/internal/adapters/datasources/onchain/glassnode"
    "prometheus/internal/adapters/datasources/liquidations/coinglass"
)

func main() {
    // ... previous initialization

    // Initialize data sources
    newsSources := []news.NewsSource{
        coindesk.New(),
        cointelegraph.New(),
        theblock.New(),
    }
    log.Infof("✓ Registered %d news sources", len(newsSources))

    sentimentSources := []sentiment.SentimentSource{
        lunarcrush.New(cfg.DataSources.LunarCrushAPIKey),
        santiment.New(cfg.DataSources.SantimentAPIKey),
    }
    log.Infof("✓ Registered %d sentiment sources", len(sentimentSources))

    onchainSources := []onchain.OnChainSource{
        glassnode.New(cfg.DataSources.GlassnodeAPIKey),
    }
    log.Infof("✓ Registered %d on-chain sources", len(onchainSources))

    liquidationSources := []liquidations.LiquidationSource{
        coinglass.New(cfg.DataSources.CoinglassAPIKey),
    }

    // Initialize workers with data sources
    workerList := []workers.Worker{
        // Market data (from Step 3)
        marketWorkers.NewOHLCVCollector(...),

        // NEW: Data source workers
        sentimentWorkers.NewNewsCollector(newsSources, sentimentRepo, 5*time.Minute),
        sentimentWorkers.NewSentimentCollector(sentimentSources, sentimentRepo, []string{"BTC", "ETH"}, 2*time.Minute),
        onchainWorkers.NewWhaleCollector(onchainSources, onchainRepo, 5*time.Minute),
        onchainWorkers.NewExchangeFlowCollector(onchainSources, onchainRepo, 10*time.Minute),
        derivativesWorkers.NewOptionsCollector(derivativesSources, derivativesRepo, 5*time.Minute),
        liquidationsWorkers.NewLiquidationCollector(liquidationSources, liquidationsRepo, 10*time.Second),
    }

    log.Infof("✓ Initialized %d workers", len(workerList))

    // ... rest of initialization
}
```

---

## Checklist

- [ ] News sources implemented (CoinDesk, CoinTelegraph, The Block)
- [ ] Sentiment sources implemented (LunarCrush, Santiment, Twitter, Reddit)
- [ ] On-chain sources implemented (Glassnode, Santiment)
- [ ] Derivatives sources implemented (Deribit, Laevitas)
- [ ] Liquidation sources implemented (Coinglass, Hyblock)
- [ ] Macro sources implemented (Investing.com, FRED)
- [ ] ClickHouse schemas for all data types
- [ ] Workers collecting data from all sources
- [ ] Tools using data sources (get_news, get_whale_movements)
- [ ] Data flowing into ClickHouse
- [ ] Agents can access data through tools

---

## Data Source Priority by Phase

### Phase 1 (MVP)
**Must have:**
- ✅ News (CoinDesk) - for Sentiment Analyst
- ✅ Fear & Greed Index - for Sentiment Analyst
- ✅ Liquidations (Coinglass) - for Order Flow Analyst

### Phase 2
**Nice to have:**
- On-Chain (Glassnode) - whale movements, exchange flows
- Social Sentiment (LunarCrush) - Twitter/Reddit sentiment
- Derivatives (Deribit) - options data

### Phase 3
**Advanced:**
- Macro data (FRED, CME FedWatch)
- Advanced on-chain metrics (MVRV, SOPR, NVT)
- Correlation data (SPX, DXY, Gold)

---

## Integration with Agents

### Sentiment Analyst Prompt
**File**: `templates/agents/sentiment_analyst/system.tmpl`

```
You are an expert in cryptocurrency market sentiment analysis.

## Available Data Sources
You have access to:
- **News**: CoinDesk, CoinTelegraph, The Block
- **Social**: Twitter, Reddit sentiment from LunarCrush
- **Fear & Greed Index**: Market emotion indicator

## Your Tools
{{range .Tools}}
- **{{.Name}}**: {{.Description}}
{{end}}

## Analysis Process

**STEP 1 - News Analysis:**
<thinking>
Check latest news for {{.Symbol}}. Look for:
- Breaking news
- Regulatory updates
- Major partnerships
- Market-moving events
</thinking>

[Call get_news tool]

**STEP 2 - Social Sentiment:**
<thinking>
Analyze social media sentiment.
- What is the crowd saying?
- Is there unusual hype or FUD?
</thinking>

[Call get_social_sentiment tool]

**STEP 3 - Fear & Greed:**
<thinking>
Check market-wide sentiment.
- Extreme fear = potential buy
- Extreme greed = potential sell
</thinking>

[Call get_fear_greed tool]

## Output
{
  "sentiment": "bullish|bearish|neutral",
  "confidence": 75,
  "news_sentiment": 0.6,
  "social_sentiment": 0.4,
  "fear_greed": 45,
  "reasoning": "...",
  "key_events": ["event1", "event2"]
}
```

---

## Next Step

**→ Continue with [Step 4: Basic Agent System](step-4-basic-agents.md)**

Теперь агенты смогут использовать богатые источники данных для принятия решений!

