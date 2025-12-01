# OrderBook WebSocket Implementation

## Overview

OrderBook (depth) data is now available through WebSocket streams. The implementation follows the same architecture as other market data streams (kline, ticker, trade, markPrice).

## Architecture

```
Binance WebSocket → Client → KafkaEventHandler → WebSocketPublisher → Kafka → DepthConsumer → ClickHouse
```

### Flow

1. **Binance WebSocket** sends depth updates (100ms frequency, 10 levels)
2. **Client** (`internal/adapters/websocket/binance/client.go`) converts to generic `DepthEvent`
3. **KafkaEventHandler** publishes to Kafka topic `websocket.depth`
4. **WebSocketDepthConsumer** reads from Kafka and batches data
5. **ClickHouse** stores snapshots in `orderbook_snapshots` table

## Features

- ✅ **Spot & Futures support** - Works for both market types
- ✅ **Real-time updates** - 100ms update frequency
- ✅ **10 depth levels** - Top 10 bids and asks
- ✅ **Batch processing** - Efficient ClickHouse writes
- ✅ **Deduplication** - Removes duplicate snapshots
- ✅ **Market type tracking** - Distinguishes spot/futures

## Configuration

### Enable Depth Streams

Add `depth` to stream types in your `.env`:

```env
WEBSOCKET_ENABLED=true
WEBSOCKET_STREAM_TYPES=kline,ticker,trade,markPrice,depth
WEBSOCKET_MARKET_TYPES=futures,spot
WEBSOCKET_SYMBOLS=BTCUSDT,ETHUSDT,BNBUSDT
```

### ClickHouse Schema

The `orderbook_snapshots` table stores depth data:

```sql
CREATE TABLE orderbook_snapshots (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    market_type LowCardinality(String),
    timestamp DateTime,
    bids String,  -- JSON array of {p: price, q: quantity}
    asks String,  -- JSON array of {p: price, q: quantity}
    bid_depth Float64,
    ask_depth Float64,
    event_time DateTime64(3),
    collected_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(event_time)
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (exchange, market_type, symbol, timestamp)
TTL timestamp + INTERVAL 7 DAY;
```

## Usage

### Query Recent OrderBook

```sql
SELECT 
    symbol,
    timestamp,
    bids,
    asks,
    bid_depth,
    ask_depth
FROM orderbook_snapshots
WHERE exchange = 'binance'
    AND symbol = 'BTCUSDT'
    AND market_type = 'futures'
ORDER BY timestamp DESC
LIMIT 10;
```

### Calculate Spread

```sql
SELECT 
    symbol,
    timestamp,
    JSONExtractFloat(asks, 1, 'p') - JSONExtractFloat(bids, 1, 'p') AS spread,
    (JSONExtractFloat(asks, 1, 'p') - JSONExtractFloat(bids, 1, 'p')) / JSONExtractFloat(bids, 1, 'p') * 100 AS spread_pct
FROM orderbook_snapshots
WHERE exchange = 'binance'
    AND symbol = 'BTCUSDT'
    AND market_type = 'futures'
ORDER BY timestamp DESC
LIMIT 100;
```

### Aggregate Depth Stats

```sql
SELECT 
    symbol,
    toStartOfMinute(timestamp) AS minute,
    avg(bid_depth) AS avg_bid_depth,
    avg(ask_depth) AS avg_ask_depth,
    avg(bid_depth + ask_depth) AS avg_total_depth
FROM orderbook_snapshots
WHERE exchange = 'binance'
    AND market_type = 'futures'
    AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY symbol, minute
ORDER BY minute DESC;
```

## Implementation Details

### Key Files

| File | Purpose |
|------|---------|
| `internal/adapters/websocket/binance/client.go` | WebSocket client with depth stream support |
| `internal/adapters/websocket/types.go` | Generic `DepthEvent` type |
| `internal/adapters/websocket/kafka_handler.go` | Publishes depth events to Kafka |
| `internal/events/websocket_publisher.go` | Kafka publisher for depth events |
| `internal/events/proto/events.proto` | Protobuf definition for `WebSocketDepthEvent` |
| `internal/consumers/websocket_depth_consumer.go` | Kafka consumer for depth events |
| `internal/repository/clickhouse/market_data.go` | `InsertOrderBook()` method |
| `internal/domain/market_data/entity.go` | `OrderBookSnapshot` entity |

### Data Flow

1. **Subscribe to Stream**
   - Spot: `<symbol>@depth10@100ms` (e.g., `btcusdt@depth10@100ms`)
   - Futures: `<symbol>@depth10@100ms` (e.g., `btcusdt@depth10@100ms`)

2. **Event Conversion**
   ```go
   DepthEvent {
       Exchange:     "binance",
       Symbol:       "BTCUSDT",
       MarketType:   "futures",
       Bids:         []PriceLevel{{Price: "45000.00", Quantity: "1.5"}},
       Asks:         []PriceLevel{{Price: "45001.00", Quantity: "2.0"}},
       LastUpdateID: 123456789,
       EventTime:    time.Now(),
   }
   ```

3. **Protobuf Serialization**
   ```protobuf
   message WebSocketDepthEvent {
       BaseEvent base = 1;
       string exchange = 2;
       string symbol = 3;
       string market_type = 4;
       repeated PriceLevel bids = 5;
       repeated PriceLevel asks = 6;
       int64 last_update_id = 7;
       google.protobuf.Timestamp event_time = 8;
   }
   ```

4. **JSON Storage**
   ```json
   {
     "bids": [{"p": "45000.00", "q": "1.5"}, {"p": "44999.50", "q": "0.8"}],
     "asks": [{"p": "45001.00", "q": "2.0"}, {"p": "45001.50", "q": "1.2"}]
   }
   ```

### Consumer Configuration

- **Batch Size**: 50 snapshots
- **Flush Interval**: 3 seconds
- **Stats Interval**: 1 minute
- **Deduplication**: By exchange + symbol + timestamp

### Monitoring

Consumer logs stats every minute:

```
📊 Depth consumer stats 
  total_received=15234 
  total_processed=14998 
  total_deduplicated=236 
  pending_batch=12 
  received_per_sec=254 
  processed_per_sec=250
```

## Binance API Details

### Partial Book Depth Streams

- **Update Speed**: 100ms or 1000ms
- **Levels**: 5, 10, or 20
- **Stream Name**: `<symbol>@depth<levels>[@100ms]`

Examples:
- `btcusdt@depth10@100ms` - 10 levels, 100ms updates
- `ethusdt@depth20` - 20 levels, 1000ms updates (default)

### Combined Streams

Multiple symbols can be subscribed in a single connection:

```
wss://fstream.binance.com/stream?streams=
  btcusdt@depth10@100ms/
  ethusdt@depth10@100ms/
  bnbusdt@depth10@100ms
```

## Use Cases

### 1. Liquidity Analysis

Monitor available liquidity at different price levels:

```sql
SELECT 
    symbol,
    timestamp,
    bid_depth,
    ask_depth,
    bid_depth + ask_depth AS total_liquidity
FROM orderbook_snapshots
WHERE timestamp >= now() - INTERVAL 5 MINUTE
ORDER BY timestamp DESC;
```

### 2. Order Book Imbalance

Detect buy/sell pressure:

```sql
SELECT 
    symbol,
    timestamp,
    bid_depth / (bid_depth + ask_depth) AS bid_ratio,
    CASE 
        WHEN bid_depth > ask_depth * 1.5 THEN 'bullish'
        WHEN ask_depth > bid_depth * 1.5 THEN 'bearish'
        ELSE 'neutral'
    END AS sentiment
FROM orderbook_snapshots
WHERE timestamp >= now() - INTERVAL 1 HOUR;
```

### 3. Spread Monitoring

Track bid-ask spread changes:

```sql
SELECT 
    symbol,
    toStartOfMinute(timestamp) AS minute,
    avg(JSONExtractFloat(asks, 1, 'p') - JSONExtractFloat(bids, 1, 'p')) AS avg_spread,
    max(JSONExtractFloat(asks, 1, 'p') - JSONExtractFloat(bids, 1, 'p')) AS max_spread,
    min(JSONExtractFloat(asks, 1, 'p') - JSONExtractFloat(bids, 1, 'p')) AS min_spread
FROM orderbook_snapshots
WHERE timestamp >= now() - INTERVAL 1 HOUR
GROUP BY symbol, minute
ORDER BY minute DESC;
```

## Performance

- **Throughput**: ~250 snapshots/sec per symbol
- **Latency**: < 200ms from exchange to ClickHouse
- **Storage**: ~7 days retention (TTL)
- **Batch Write**: 50 snapshots every 3s

## Troubleshooting

### No Data in ClickHouse

1. Check WebSocket connection:
   ```bash
   grep "Connected to.*depth" logs/app.log
   ```

2. Check Kafka messages:
   ```bash
   kafka-console-consumer --bootstrap-server localhost:9092 \
     --topic websocket.depth --from-beginning
   ```

3. Check consumer logs:
   ```bash
   grep "Depth consumer" logs/app.log
   ```

### High Memory Usage

Reduce batch size in `internal/consumers/websocket_depth_consumer.go`:

```go
const (
    depthBatchSize = 25  // Reduced from 50
)
```

### Too Many Updates

Change to 1000ms (1s) updates by modifying stream configuration in client.

## Future Enhancements

- [ ] Support for different depth levels (5, 20)
- [ ] Configurable update frequency (100ms, 1000ms)
- [ ] Order book reconstruction from diff updates
- [ ] Real-time order book visualization
- [ ] Depth chart generation
- [ ] Liquidity heatmaps

## References

- [Binance Futures WebSocket Depth Streams](https://binance-docs.github.io/apidocs/futures/en/#partial-book-depth-streams)
- [Binance Spot WebSocket Depth Streams](https://binance-docs.github.io/apidocs/spot/en/#partial-book-depth-streams)


