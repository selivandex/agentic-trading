<!-- @format -->

# WebSocket Adapter

Real-time market data streaming from crypto exchanges via WebSocket connections.

## Architecture

```
Exchange WebSocket → Client → KafkaEventHandler → WebSocketPublisher → Kafka → Consumer → ClickHouse
                                                                                      ↓
                                                                               Analysis/Trading
```

## Components

### 1. Abstraction Layer (`types.go`)

Generic types for all exchanges:

- `Client` interface
- `StreamConfig` - stream configuration
- `KlineEvent`, `TickerEvent`, `DepthEvent`, etc. - generic event types
- `EventHandler` - callback interface for events

### 2. Exchange Implementations

#### Binance (`binance/client.go`)

- Connects to Binance Futures WebSocket API
- Supports combined multi-interval kline streams
- Automatic reconnection (TODO)
- Graceful shutdown with context cancellation

**Features:**

- Multiple symbols simultaneously
- Multiple timeframes per symbol
- Efficient combined stream (single connection)
- Statistics tracking (messages, errors, reconnects)

### 3. Event Publishing (`kafka_handler.go`)

Implements `EventHandler` interface and publishes to Kafka via protobuf:

- Converts exchange-specific events to protobuf
- Publishes to topic-specific Kafka topics
- Structured logging

### 4. Kafka Topics

All WebSocket events are published to Kafka using protobuf:

- `websocket.kline` - Candlestick data
- `websocket.ticker` - 24hr ticker statistics
- `websocket.depth` - Order book snapshots
- `websocket.trade` - Individual trades
- `websocket.funding_rate` - Futures funding rates
- `websocket.mark_price` - Futures mark prices

### 5. Consumers

#### WebSocket Kline Consumer (`consumers/websocket_kline_consumer.go`)

- Consumes kline events from Kafka
- Writes to ClickHouse for historical analysis
- Only processes final (closed) candles
- Graceful shutdown with message completion

## Configuration

Environment variables (`.env`):

```bash
# WebSocket Settings
WEBSOCKET_ENABLED=true
WEBSOCKET_USE_TESTNET=false
WEBSOCKET_SYMBOLS=BTCUSDT,ETHUSDT,SOLUSDT
WEBSOCKET_INTERVALS=1m,5m,15m,1h,4h
WEBSOCKET_EXCHANGES=binance
WEBSOCKET_RECONNECT_BACKOFF=5s
WEBSOCKET_MAX_RECONNECTS=10
WEBSOCKET_PING_INTERVAL=60s
WEBSOCKET_READ_BUFFER_SIZE=4096
WEBSOCKET_WRITE_BUFFER_SIZE=4096
```

## Usage

### Adding a New Exchange

1. Create `internal/adapters/websocket/<exchange>/client.go`
2. Implement `websocket.Client` interface
3. Convert exchange events to generic `websocket.*Event` types
4. Register in `bootstrap/websocket.go`

Example:

```go
type BybitClient struct {
    // ...
}

func (c *BybitClient) Connect(ctx context.Context, config websocket.ConnectionConfig) error {
    // Connect to Bybit WebSocket
}

func (c *BybitClient) Start(ctx context.Context) error {
    // Start receiving events
}
```

### Adding a New Stream Type

1. Add stream type constant in `types.go`:

```go
const StreamTypeLiquidations StreamType = "liquidations"
```

2. Add event type:

```go
type LiquidationEvent struct {
    Exchange string
    Symbol   string
    // ...
}
```

3. Add protobuf message in `events/proto/events.proto`
4. Add publisher method in `events/websocket_publisher.go`
5. Add handler method in `EventHandler` interface
6. Implement in `kafka_handler.go`

## Supported Timeframes

All Binance futures timeframes:

- `1m`, `3m`, `5m`, `15m`, `30m`
- `1h`, `2h`, `4h`, `6h`, `8h`, `12h`
- `1d`, `3d`, `1w`, `1M`

## Data Flow

1. **Exchange** → WebSocket connection with market data
2. **Client** → Receives raw events, converts to generic format
3. **KafkaEventHandler** → Publishes protobuf events to Kafka
4. **Kafka** → Message broker (reliable delivery, replay capability)
5. **Consumer** → Consumes events, writes to ClickHouse
6. **ClickHouse** → Time-series database for historical analysis
7. **Agents** → Use historical data for trading decisions

## Benefits

- **Decoupling**: Producers and consumers are independent
- **Reliability**: Kafka ensures no data loss
- **Scalability**: Add more consumers to handle load
- **Replay**: Reprocess historical events if needed
- **Type Safety**: Protobuf ensures schema consistency
- **Performance**: Binary protobuf format

## Bootstrap Integration

WebSocket clients are initialized and started in application bootstrap:

```go
// In bootstrap/websocket.go
c.MustInitWebSocketClients() // Initialize clients
c.connectWebSocketClients()  // Connect and start
c.stopWebSocketClients()     // Graceful shutdown
```

All components support graceful shutdown:

- Context cancellation propagates to all goroutines
- Current messages are processed before stopping
- Connections closed cleanly
- No data loss

## Monitoring

WebSocket clients expose statistics:

```go
stats := client.GetStats()
// stats.ConnectedSince
// stats.ReconnectCount
// stats.MessagesReceived
// stats.ErrorCount
// stats.ActiveStreams
```

TODO: Export to Prometheus metrics.

## Error Handling

- Transient errors → logged, continue operation
- Connection errors → automatic reconnection (TODO)
- Context cancellation → graceful shutdown
- Parse errors → logged, skip message

## Future Improvements

- [ ] Automatic reconnection with exponential backoff
- [ ] Connection health monitoring
- [ ] Rate limiting awareness
- [ ] Message compression
- [ ] More stream types (liquidations, funding rates, etc.)
- [ ] More exchanges (Bybit, OKX, etc.)
- [ ] Metrics export (Prometheus)
- [ ] Connection pooling for multiple streams
