# WebSocket Authentication Guide

## Overview

Binance WebSocket API supports two types of streams:

### 1. Public Market Data Streams (No Auth Required)

These streams provide real-time market data and **do not require** API keys:

- **Kline/Candlestick** (`@kline_<interval>`) - OHLCV data
- **24hr Ticker** (`@ticker`) - Price statistics
- **Order Book** (`@depth`) - Bid/ask levels
- **Trades** (`@trade`) - Recent trades
- **Mark Price** (`@markPrice`) - Futures mark price
- **Funding Rate** (`@fundingRate`) - Perpetual funding rates

**Current Implementation**: ✅ We use these streams (no auth needed)

### 2. Private User Data Streams (Auth Required)

These streams provide account-specific data and **require authentication**:

- **Account Updates** - Balance changes
- **Order Updates** - Order fills, cancellations
- **Position Updates** - Position changes
- **Margin Calls** - Liquidation warnings

**Current Implementation**: ❌ Not yet implemented

---

## Configuration

### Public Streams (Current)

No API keys needed. Just enable WebSocket in `.env`:

```bash
# WebSocket Settings (Public Streams)
WEBSOCKET_ENABLED=true
WEBSOCKET_USE_TESTNET=false
WEBSOCKET_SYMBOLS=BTCUSDT,ETHUSDT,SOLUSDT
WEBSOCKET_INTERVALS=1m,5m,15m,1h,4h
```

### Private Streams (Future)

For user data streams, add Binance credentials:

```bash
# Binance Market Data API (for WebSocket authentication)
BINANCE_MARKET_DATA_API_KEY=your_api_key_here
BINANCE_MARKET_DATA_SECRET=your_secret_key_here
```

**Important**: 
- These credentials are for **system-wide market data collection**
- They are separate from user trading accounts
- Use a **read-only** API key with only market data permissions
- **Never** use keys with withdrawal permissions

---

## Authentication Methods

### Method 1: Public Streams (Current)

```go
// No authentication needed
client := binancews.NewClient(
    "binance",
    handler,
    "",  // apiKey - empty for public streams
    "",  // secretKey - empty for public streams
    false, // useTestnet
    log,
)
```

### Method 2: WebSocket API with Ed25519 Signature (Future)

For WebSocket API endpoints that require authentication:

```go
// With authentication
client := binancews.NewClient(
    "binance",
    handler,
    apiKey,
    secretKey,
    false,
    log,
)

// The client will:
// 1. Create signature using Ed25519 or HMAC-SHA256
// 2. Send session.logon request
// 3. Use authenticated session for private requests
```

**Authentication Flow**:

1. Connect to `wss://ws-fapi.binance.com/ws-fapi/v1`
2. Send `session.logon` request with signature
3. Receive authenticated session
4. Send private data requests (orders, positions, etc.)

### Method 3: User Data Stream with Listen Key (Future)

For user data streams (account updates, order updates):

```go
// 1. Create listen key via REST API
listenKey := createListenKey(apiKey, secretKey)

// 2. Connect to user data stream
url := fmt.Sprintf("wss://fstream.binance.com/ws/%s", listenKey)

// 3. Renew listen key every 30 minutes
ticker := time.NewTicker(30 * time.Minute)
go func() {
    for range ticker.C {
        renewListenKey(apiKey, secretKey, listenKey)
    }
}()
```

---

## Security Best Practices

### 1. API Key Permissions

For market data WebSocket, create API key with **minimum permissions**:

✅ **Enable**:
- Read market data
- Read account info (if needed)

❌ **Disable**:
- Trading
- Withdrawals
- Margin trading
- Futures trading

### 2. IP Whitelist

Always restrict API keys to specific IP addresses:

```
Your Server IP → Binance API
```

### 3. Key Storage

**Never** commit API keys to git:

```bash
# .env (gitignored)
BINANCE_MARKET_DATA_API_KEY=xxx
BINANCE_MARKET_DATA_SECRET=yyy
```

Use environment variables or secrets management:
- AWS Secrets Manager
- HashiCorp Vault
- Kubernetes Secrets

### 4. Key Rotation

Rotate API keys periodically:
- Every 90 days minimum
- After any suspected compromise
- When team members leave

---

## Implementation Roadmap

### Phase 1: Public Streams ✅ (Current)

- [x] Kline/candlestick streams
- [x] Combined multi-symbol, multi-interval streams
- [x] Protobuf event publishing to Kafka
- [x] ClickHouse storage for historical data
- [x] Graceful shutdown

### Phase 2: Private Streams (TODO)

- [ ] User data stream with listen key
- [ ] Account balance updates
- [ ] Order execution updates
- [ ] Position updates
- [ ] Margin call warnings

### Phase 3: WebSocket API (TODO)

- [ ] Ed25519 signature authentication
- [ ] Session management
- [ ] Place/cancel orders via WebSocket
- [ ] Query account info via WebSocket
- [ ] Query positions via WebSocket

---

## Code Examples

### Current: Public Kline Stream

```go
// Connect to public kline stream (no auth)
streams := []websocket.StreamConfig{
    {
        Type:     websocket.StreamTypeKline,
        Symbol:   "BTCUSDT",
        Interval: websocket.Interval1m,
    },
}

client.Connect(ctx, websocket.ConnectionConfig{
    Streams: streams,
})
```

### Future: User Data Stream

```go
// TODO: Connect to user data stream (with auth)
// 1. Get listen key
listenKey, err := binanceClient.NewListenKeyService().Do(ctx)

// 2. Subscribe to user data
userDataHandler := func(event *binance.WsUserDataEvent) {
    switch event.Event {
    case "ACCOUNT_UPDATE":
        // Handle balance update
    case "ORDER_TRADE_UPDATE":
        // Handle order update
    }
}

doneC, stopC, err := binance.WsUserDataServe(
    listenKey.ListenKey,
    userDataHandler,
    errHandler,
)

// 3. Renew listen key every 30 min
go renewListenKeyPeriodically(listenKey.ListenKey)
```

---

## Troubleshooting

### Error: "Invalid API key"

**Solution**: Check that API key is correct and has proper permissions.

### Error: "Signature verification failed"

**Solution**: 
- Check that secret key is correct
- Ensure timestamp is synchronized (NTP)
- Verify signature algorithm (Ed25519 vs HMAC-SHA256)

### Error: "Listen key expired"

**Solution**: Renew listen key every 30 minutes using REST API.

### Connection drops frequently

**Solution**:
- Implement automatic reconnection with exponential backoff
- Send ping frames every 3 minutes
- Monitor connection health

---

## References

- [Binance Futures WebSocket Market Streams](https://binance-docs.github.io/apidocs/futures/en/#websocket-market-streams)
- [Binance WebSocket API](https://developers.binance.com/docs/derivatives/usds-margined-futures/websocket-api-general-info)
- [User Data Streams](https://binance-docs.github.io/apidocs/futures/en/#user-data-streams)

---

## Summary

**Current Status**: ✅ Public market data streams working (no auth needed)

**For Production**: 
- Add `BINANCE_MARKET_DATA_API_KEY` and `BINANCE_MARKET_DATA_SECRET` to enable private streams in the future
- Current implementation works fine without credentials for market data collection
- User trading accounts use separate credentials (per-user, encrypted in database)



