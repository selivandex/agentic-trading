# Step 8: Advanced Features & Optimization

## Overview
Финальный этап: оптимизация производительности, кэширование, батчинг, тестирование и подготовка к production.

**Цель**: Production-ready система с высокой производительностью.

**Время**: 7-10 дней

---

## 8.1 Performance Optimization

### ClickHouse Batch Writer
**File**: `pkg/clickhouse/batcher.go`

```go
package clickhouse

import (
    "context"
    "sync"
    "time"
    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
    "prometheus/pkg/logger"
)

type Batcher struct {
    conn          driver.Conn
    table         string
    buffer        []interface{}
    bufferMu      sync.Mutex
    batchSize     int
    flushInterval time.Duration
    log           *logger.Logger
    shutdown      chan struct{}
    wg            sync.WaitGroup
}

type BatcherConfig struct {
    Conn          driver.Conn
    Table         string
    BatchSize     int
    FlushInterval time.Duration
}

func NewBatcher(cfg BatcherConfig) *Batcher {
    b := &Batcher{
        conn:          cfg.Conn,
        table:         cfg.Table,
        buffer:        make([]interface{}, 0, cfg.BatchSize),
        batchSize:     cfg.BatchSize,
        flushInterval: cfg.FlushInterval,
        log:           logger.Get().With("component", "clickhouse_batcher", "table", cfg.Table),
        shutdown:      make(chan struct{}),
    }

    b.wg.Add(1)
    go b.flushLoop()

    return b
}

func (b *Batcher) Write(ctx context.Context, data interface{}) error {
    b.bufferMu.Lock()
    defer b.bufferMu.Unlock()

    b.buffer = append(b.buffer, data)

    if len(b.buffer) >= b.batchSize {
        return b.flushLocked(ctx)
    }

    return nil
}

func (b *Batcher) flushLoop() {
    defer b.wg.Done()

    ticker := time.NewTicker(b.flushInterval)
    defer ticker.Stop()

    for {
        select {
        case <-b.shutdown:
            b.bufferMu.Lock()
            if len(b.buffer) > 0 {
                b.flushLocked(context.Background())
            }
            b.bufferMu.Unlock()
            return

        case <-ticker.C:
            b.bufferMu.Lock()
            if len(b.buffer) > 0 {
                b.flushLocked(context.Background())
            }
            b.bufferMu.Unlock()
        }
    }
}

func (b *Batcher) flushLocked(ctx context.Context) error {
    if len(b.buffer) == 0 {
        return nil
    }

    batch := b.buffer
    b.buffer = b.buffer[:0]

    batchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    err := b.conn.AsyncInsert(batchCtx, b.table, batch)
    if err != nil {
        b.log.Errorf("Failed to flush batch: %v", err)
        b.buffer = append(b.buffer, batch...)
        return err
    }

    b.log.Debugf("Flushed %d records", len(batch))
    return nil
}

func (b *Batcher) Shutdown(ctx context.Context) error {
    close(b.shutdown)
    
    done := make(chan struct{})
    go func() {
        b.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

---

## 8.2 Redis Caching

### Cache Layer
**File**: `internal/adapters/redis/cache.go`

```go
package redis

import (
    "context"
    "encoding/json"
    "time"
    "github.com/redis/go-redis/v9"
)

type Cache struct {
    rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
    return &Cache{rdb: rdb}
}

func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
    data, err := c.rdb.Get(ctx, key).Bytes()
    if err != nil {
        return err
    }
    return json.Unmarshal(data, dest)
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
    return c.rdb.Del(ctx, key).Err()
}

// Cache patterns
func (c *Cache) CacheTicker(ctx context.Context, exchange, symbol string, ticker *exchanges.Ticker) error {
    key := fmt.Sprintf("cache:ticker:%s:%s", exchange, symbol)
    return c.Set(ctx, key, ticker, 5*time.Second)
}

func (c *Cache) GetCachedTicker(ctx context.Context, exchange, symbol string) (*exchanges.Ticker, error) {
    key := fmt.Sprintf("cache:ticker:%s:%s", exchange, symbol)
    var ticker exchanges.Ticker
    err := c.Get(ctx, key, &ticker)
    return &ticker, err
}
```

---

## 8.3 Kafka Integration

### Producer
**File**: `internal/adapters/kafka/producer.go`

```go
package kafka

import (
    "context"
    "encoding/json"
    "github.com/segmentio/kafka-go"
    "prometheus/pkg/logger"
)

type Producer struct {
    writers map[string]*kafka.Writer
    brokers []string
    log     *logger.Logger
}

func NewProducer(brokers []string) *Producer {
    return &Producer{
        writers: make(map[string]*kafka.Writer),
        brokers: brokers,
        log:     logger.Get().With("component", "kafka_producer"),
    }
}

func (p *Producer) getWriter(topic string) *kafka.Writer {
    if w, ok := p.writers[topic]; ok {
        return w
    }

    w := &kafka.Writer{
        Addr:     kafka.TCP(p.brokers...),
        Topic:    topic,
        Balancer: &kafka.LeastBytes{},
        Async:    true,
    }

    p.writers[topic] = w
    return w
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, event interface{}) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    msg := kafka.Message{
        Key:   []byte(key),
        Value: data,
    }

    if err := p.getWriter(topic).WriteMessages(ctx, msg); err != nil {
        p.log.Errorf("Failed to publish to %s: %v", topic, err)
        return err
    }

    return nil
}

func (p *Producer) Close() error {
    for _, w := range p.writers {
        if err := w.Close(); err != nil {
            return err
        }
    }
    return nil
}
```

### Topics
**File**: `internal/adapters/kafka/topics.go`

```go
package kafka

const (
    TopicTradeOpened    = "trades.opened"
    TopicTradeClosed    = "trades.closed"
    TopicOrderPlaced    = "orders.placed"
    TopicRiskAlert      = "risk.alerts"
    TopicCircuitBreaker = "risk.circuit_breaker"
    TopicKillSwitch     = "risk.kill_switch"
    TopicNotifications  = "notifications.telegram"
)
```

---

## 8.4 Batch Executor

### Batch Order Execution
**File**: `internal/execution/batch_executor.go`

```go
package execution

import (
    "context"
    "sync"
    "time"
    "prometheus/internal/adapters/exchanges"
    "prometheus/internal/domain/order"
    "prometheus/pkg/logger"
)

type BatchExecutor struct {
    exchangeFactory exchanges.Factory
    orderRepo       order.Repository
    log             *logger.Logger
    batchSize       int
    batchInterval   time.Duration
    queue           chan *order.Order
}

func NewBatchExecutor(
    exchangeFactory exchanges.Factory,
    orderRepo order.Repository,
    batchSize int,
    batchInterval time.Duration,
) *BatchExecutor {
    ex := &BatchExecutor{
        exchangeFactory: exchangeFactory,
        orderRepo:       orderRepo,
        log:             logger.Get().With("component", "batch_executor"),
        batchSize:       batchSize,
        batchInterval:   batchInterval,
        queue:           make(chan *order.Order, 1000),
    }

    go ex.processBatches(context.Background())
    return ex
}

func (e *BatchExecutor) Submit(ctx context.Context, o *order.Order) error {
    select {
    case e.queue <- o:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (e *BatchExecutor) processBatches(ctx context.Context) {
    ticker := time.NewTicker(e.batchInterval)
    defer ticker.Stop()

    batch := make([]*order.Order, 0, e.batchSize)

    for {
        select {
        case <-ctx.Done():
            if len(batch) > 0 {
                e.executeBatch(ctx, batch)
            }
            return

        case order := <-e.queue:
            batch = append(batch, order)
            if len(batch) >= e.batchSize {
                e.executeBatch(ctx, batch)
                batch = batch[:0]
            }

        case <-ticker.C:
            if len(batch) > 0 {
                e.executeBatch(ctx, batch)
                batch = batch[:0]
            }
        }
    }
}

func (e *BatchExecutor) executeBatch(ctx context.Context, orders []*order.Order) {
    var wg sync.WaitGroup
    for _, o := range orders {
        wg.Add(1)
        go func(order *order.Order) {
            defer wg.Done()
            e.executeOrder(ctx, order)
        }(o)
    }
    wg.Wait()
}

func (e *BatchExecutor) executeOrder(ctx context.Context, o *order.Order) error {
    account, _ := e.orderRepo.GetExchangeAccount(ctx, o.ExchangeAccountID)
    client, _ := e.exchangeFactory.CreateClient(account)

    result, err := client.PlaceOrder(ctx, &exchanges.OrderRequest{
        Symbol: o.Symbol,
        Side:   string(o.Side),
        Type:   string(o.Type),
        Amount: o.Amount,
        Price:  o.Price,
    })

    if err != nil {
        o.Status = order.OrderStatusRejected
    } else {
        o.ExchangeOrderID = result.ID
        o.Status = order.OrderStatusOpen
    }

    return e.orderRepo.Update(ctx, o)
}
```

---

## 8.5 Testing

### Unit Tests Example
**File**: `internal/tools/indicators/rsi_test.go`

```go
package indicators

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCalculateRSI(t *testing.T) {
    closes := []float64{
        44.0, 44.5, 45.0, 44.8, 45.2,
        45.5, 45.3, 45.8, 46.0, 45.7,
        46.2, 46.5, 46.3, 46.8, 47.0,
    }

    rsi := calculateRSI(closes, 14)

    assert.True(t, rsi > 0 && rsi < 100, "RSI should be between 0 and 100")
    assert.True(t, rsi > 50, "Uptrend should have RSI > 50")
}

func TestRSI_Overbought(t *testing.T) {
    // Strong uptrend
    closes := make([]float64, 20)
    for i := range closes {
        closes[i] = 100.0 + float64(i)*5.0
    }

    rsi := calculateRSI(closes, 14)
    assert.True(t, rsi > 70, "Strong uptrend should have RSI > 70")
}

func TestRSI_Oversold(t *testing.T) {
    // Strong downtrend
    closes := make([]float64, 20)
    for i := range closes {
        closes[i] = 100.0 - float64(i)*5.0
    }

    rsi := calculateRSI(closes, 14)
    assert.True(t, rsi < 30, "Strong downtrend should have RSI < 30")
}
```

### Integration Tests
**File**: `tests/integration/exchange_test.go`

```go
package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "prometheus/internal/adapters/exchanges/binance"
)

func TestBinanceConnection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    client, err := binance.NewClient(binance.Config{
        APIKey:    "", // Public data doesn't need keys
        SecretKey: "",
        Testnet:   true,
    })

    assert.NoError(t, err)

    ticker, err := client.GetTicker(context.Background(), "BTCUSDT")
    assert.NoError(t, err)
    assert.NotNil(t, ticker)
    assert.Equal(t, "BTCUSDT", ticker.Symbol)
    assert.True(t, ticker.Price.GreaterThan(decimal.Zero))
}
```

---

## 8.6 Monitoring

### Prometheus Metrics
**File**: `internal/metrics/metrics.go`

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    OrdersTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "trading_orders_total",
            Help: "Total number of orders placed",
        },
        []string{"exchange", "symbol", "side", "status"},
    )

    PnLTotal = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "trading_pnl_total",
            Help: "Total PnL per user and symbol",
        },
        []string{"user_id", "symbol"},
    )

    AgentExecutionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "agent_execution_duration_seconds",
            Help:    "Agent execution time",
            Buckets: prometheus.DefBuckets,
        },
        []string{"agent"},
    )

    ToolCallsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agent_tool_calls_total",
            Help: "Total tool calls per agent",
        },
        []string{"agent", "tool"},
    )
)

func RecordOrder(exchange, symbol, side, status string) {
    OrdersTotal.WithLabelValues(exchange, symbol, side, status).Inc()
}

func SetPnL(userID, symbol string, pnl float64) {
    PnLTotal.WithLabelValues(userID, symbol).Set(pnl)
}
```

---

## 8.7 Graceful Shutdown

### Shutdown Manager
**File**: `internal/shutdown/manager.go`

```go
package shutdown

import (
    "context"
    "sync"
    "time"
    "prometheus/pkg/logger"
)

type Manager struct {
    components []Component
    log        *logger.Logger
}

type Component interface {
    Name() string
    Shutdown(ctx context.Context) error
}

func NewManager() *Manager {
    return &Manager{
        components: make([]Component, 0),
        log:        logger.Get().With("component", "shutdown_manager"),
    }
}

func (m *Manager) Register(component Component) {
    m.components = append(m.components, component)
}

func (m *Manager) Shutdown(timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    m.log.Info("Starting graceful shutdown...")

    var wg sync.WaitGroup
    errors := make(chan error, len(m.components))

    for _, component := range m.components {
        wg.Add(1)
        go func(c Component) {
            defer wg.Done()
            m.log.Infof("Shutting down %s...", c.Name())
            if err := c.Shutdown(ctx); err != nil {
                errors <- fmt.Errorf("%s: %w", c.Name(), err)
            } else {
                m.log.Infof("✓ %s shut down", c.Name())
            }
        }(component)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        m.log.Errorf("Shutdown error: %v", err)
    }

    m.log.Info("Shutdown complete")
    return nil
}
```

---

## 8.8 Configuration Updates

### Market Data Config
**File**: `internal/adapters/config/config.go` (additions)

```go
type MarketDataConfig struct {
    BinanceAPIKey    string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
    BinanceSecret    string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
    BybitAPIKey      string `envconfig:"BYBIT_MARKET_DATA_API_KEY"`
    BybitSecret      string `envconfig:"BYBIT_MARKET_DATA_SECRET"`
    OKXAPIKey        string `envconfig:"OKX_MARKET_DATA_API_KEY"`
    OKXSecret        string `envconfig:"OKX_MARKET_DATA_SECRET"`
    OKXPassphrase    string `envconfig:"OKX_MARKET_DATA_PASSPHRASE"`
}

type KafkaConfig struct {
    Brokers []string `envconfig:"KAFKA_BROKERS" required:"true"`
    GroupID string   `envconfig:"KAFKA_GROUP_ID" default:"prometheus"`
}
```

### Update .env.example
```bash
# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=prometheus

# Central Market Data API Keys
BINANCE_MARKET_DATA_API_KEY=
BINANCE_MARKET_DATA_SECRET=
BYBIT_MARKET_DATA_API_KEY=
BYBIT_MARKET_DATA_SECRET=
```

---

## 8.9 Docker Compose Updates

### Add Kafka
```yaml
# docker-compose.yml (add to services)
  kafka:
    image: bitnami/kafka:3.6
    container_name: prometheus-kafka
    ports:
      - "9092:9092"
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
    volumes:
      - kafka_data:/bitnami/kafka
    healthcheck:
      test: ["CMD-SHELL", "kafka-topics.sh --list --bootstrap-server localhost:9092"]
      interval: 10s
      timeout: 5s
      retries: 5

  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    container_name: prometheus-kafka-ui
    ports:
      - "8080:8080"
    environment:
      - KAFKA_CLUSTERS_0_NAME=local
      - KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092
    depends_on:
      - kafka

volumes:
  kafka_data:
```

---

## 8.10 Production Dockerfile

### Multi-stage Build
**File**: `Dockerfile`

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always)" \
    -o /prometheus ./cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /prometheus .

# Copy templates
COPY templates ./templates

# Copy migrations
COPY migrations ./migrations

# Create non-root user
RUN addgroup -g 1000 prometheus && \
    adduser -D -u 1000 -G prometheus prometheus && \
    chown -R prometheus:prometheus /app

USER prometheus

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/prometheus", "health"]

ENTRYPOINT ["./prometheus"]
```

---

## 8.11 Documentation

### README.md
**File**: `README.md`

```markdown
# Prometheus Trading System

Autonomous multi-agent cryptocurrency trading system with Telegram interface.

## Features

- 🤖 14 specialized AI agents with Chain-of-Thought reasoning
- 📊 80+ technical and fundamental analysis tools
- 🛡️ Advanced risk management with circuit breaker
- 📱 Full Telegram bot interface
- 💾 Multi-database architecture (PostgreSQL + ClickHouse + Redis)
- 📈 Real-time market data from multiple exchanges
- 🧠 Semantic memory with pgvector
- 📉 Self-evaluation and strategy optimization

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.22+
- Telegram Bot Token
- Claude API Key (or other AI provider)

### Setup

1. Clone repository:
```bash
git clone <repo>
cd prometheus
```

2. Configure environment:
```bash
cp .env.example .env
# Edit .env with your credentials
```

3. Start infrastructure:
```bash
make docker-up
```

4. Run application:
```bash
make dev
```

### Telegram Commands

- `/start` - Initialize bot
- `/connect` - Connect exchange account
- `/open_position BTC 100` - Open $100 BTC position
- `/positions` - View open positions
- `/stats` - Trading statistics
- `/settings` - Adjust settings

## Architecture

See [docs/specs.md](docs/specs.md) for complete technical specification.

## Development

Follow step-by-step guides in `docs/`:
- [Step 1: Foundation](docs/step-1-foundation.md)
- [Step 2: Domain & Database](docs/step-2-domain-database.md)
- [Step 3: Exchange Integration](docs/step-3-exchange-integration.md)
- [Step 4: Basic Agents](docs/step-4-basic-agents.md)
- [Step 5: Advanced Agents](docs/step-5-advanced-agents.md)
- [Step 6: Risk Management](docs/step-6-risk-management.md)
- [Step 7: Telegram Bot](docs/step-7-telegram-bot.md)
- [Step 8: Optimization](docs/step-8-advanced-features.md)

## License

MIT
```

---

## 8.12 Production Deployment

### Deployment Checklist

**Infrastructure:**
- [ ] PostgreSQL 16 with pgvector
- [ ] ClickHouse 24.1+
- [ ] Redis 7+
- [ ] Kafka 3.6+

**Security:**
- [ ] Generate 32-byte encryption key
- [ ] Secure API keys storage
- [ ] Enable SSL/TLS for databases
- [ ] Set up firewall rules

**Monitoring:**
- [ ] Prometheus metrics exposed
- [ ] Grafana dashboards
- [ ] Sentry error tracking
- [ ] Alert rules configured

**Backup:**
- [ ] PostgreSQL daily backups
- [ ] ClickHouse backups
- [ ] Backup retention policy

---

## Checklist

- [ ] ClickHouse batch writer implemented
- [ ] Redis caching layer working
- [ ] Kafka producer/consumer working
- [ ] Batch executor for orders
- [ ] Performance optimized
- [ ] Unit tests written
- [ ] Integration tests passing
- [ ] Graceful shutdown working
- [ ] Monitoring metrics exposed
- [ ] README.md complete
- [ ] Dockerfile working
- [ ] Production deployment guide

---

## Next Steps After MVP

### Phase 2: ML Integration
- Market regime detection model
- Anomaly detection
- Price prediction signals
- Training pipeline

### Phase 3: Advanced Features
- Web dashboard (React)
- Backtesting engine
- Strategy marketplace
- Copy trading
- Mobile app

### Phase 4: Scale
- Multi-region deployment
- High-frequency trading mode
- Advanced execution algorithms
- Institutional features

---

## Appendix: Key Metrics

### System Performance Targets

| Metric                 | Target    |
| ---------------------- | --------- |
| Order execution time   | < 500ms   |
| Agent decision time    | < 30s     |
| Market data latency    | < 100ms   |
| Database query time    | < 50ms    |
| Memory search time     | < 200ms   |
| Circuit breaker check  | < 10ms    |

### Scalability Targets

| Resource         | Capacity           |
| ---------------- | ------------------ |
| Concurrent users | 1,000+             |
| Positions/user   | 20                 |
| Tools/agent      | 100                |
| Memories/user    | 10,000+            |
| OHLCV records    | 100M+              |
| Events/second    | 10,000+            |

---

## Production Launch

1. **Week 1-2**: Deploy infrastructure, migrate databases
2. **Week 3**: Deploy application, monitor logs
3. **Week 4**: Invite beta testers (10-20 users)
4. **Week 5-6**: Fix bugs, optimize performance
5. **Week 7-8**: Public launch

**Monitoring during launch:**
- Error rate < 1%
- Uptime > 99.9%
- Average response time < 1s
- Memory usage stable
- No data loss

---

## Success Metrics

After 30 days of operation:

- [ ] 100+ active users
- [ ] 1000+ trades executed
- [ ] Win rate > 50%
- [ ] Zero critical bugs
- [ ] Uptime > 99.5%
- [ ] User satisfaction > 4.5/5

---

## Conclusion

Система готова к production при выполнении всех 8 этапов.

**Важно:**
- Начинайте с малых бюджетов ($10-50)
- Используйте testnet для тестирования
- Мониторьте логи в первые недели
- Собирайте feedback от пользователей
- Постепенно увеличивайте нагрузку

**Support:**
- GitHub Issues для багов
- Telegram community для вопросов
- Email support для критичных проблем

Good luck with your trading system! 🚀

