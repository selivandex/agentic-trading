# WebSocket Auto-Reconnect

## Overview

The system now has automatic reconnection logic for both **Market Data WebSocket** and **User Data WebSocket** connections.

## Market Data WebSocket Manager

**Purpose**: Manages centralized market data streams (klines, ticker, depth, trades, liquidations, mark price) with health monitoring and automatic reconnection.

### Features

- ✅ **Health Check Loop**: Checks connection status every 60 seconds
- ✅ **Auto-Reconnect**: Automatically reconnects when "broken pipe" or connection loss detected
- ✅ **Graceful Degradation**: Logs warnings after 3 consecutive failures
- ✅ **Metrics**: Tracks reconnection attempts (success/failed) via Prometheus

### Configuration

Located in `internal/bootstrap/websocket.go`:

```go
managerConfig := websocket.MarketDataManagerConfig{
    HealthCheckInterval:    60 * time.Second, // Check every 60 seconds
    MaxConsecutiveFailures: 3,                 // Warn after 3 failures
}
```

### Metrics

- `prometheus_marketdata_reconnects_total{status="success"}` - Successful reconnects
- `prometheus_marketdata_reconnects_total{status="failed"}` - Failed reconnect attempts

### Logs

Watch for these log messages:

```
✓ Market Data Manager started
⚠️ Market Data WebSocket disconnected, attempting reconnect
✅ Market Data WebSocket reconnected successfully
```

## User Data WebSocket Manager

**Purpose**: Manages per-user WebSocket connections for order updates, position updates, and account events.

### Features

- ✅ **Hot Reload**: Automatically detects new/deleted exchange accounts every 5 minutes
- ✅ **Health Check Loop**: Monitors connection health every 60 seconds
- ✅ **ListenKey Renewal**: Automatically renews Binance listenKeys every 30 minutes
- ✅ **Credentials Validation**: Auto-deactivates accounts with invalid API keys
- ✅ **Metrics**: Tracks connections, reconnects, hot reloads, listenKey renewals

### Configuration

Located in `internal/bootstrap/userdata_manager.go`:

```go
config := websocket.UserDataManagerConfig{
    HealthCheckInterval:    60 * time.Second, // Health check every 60s
    ListenKeyRenewal:       30 * time.Minute, // Renew every 30 min
    ReconciliationInterval: 5 * time.Minute,  // Hot reload every 5 min
}
```

### Metrics

- `prometheus_userdata_connections{exchange="binance"}` - Active connections
- `prometheus_userdata_reconnects_total{exchange,reason}` - Reconnect attempts
- `prometheus_userdata_hotreload_operations_total{operation,exchange}` - Hot reloads (add/remove)
- `prometheus_userdata_listenkey_renewals_total{exchange,status}` - ListenKey renewals
- `prometheus_userdata_reconciliations_total{status}` - Reconciliation cycles

### Logs

Watch for these log messages:

```
✓ User Data Manager started
🆕 HOT RELOAD: Detected new active account in database
🔌 Connecting User Data WebSocket for account
✅ User Data WebSocket connected and streaming
⚠️ Detected credentials error, deactivating account
✓ Reconnected account
```

## Lifecycle

### Startup

1. **WebSocket Clients** created (Binance, Bybit, OKX adapters)
2. **MarketDataManager** initialized and started (handles market data auto-reconnect)
3. **UserDataManager** loads all active accounts from DB and connects them

### Runtime

- **Market Data**: Health check every 60s, auto-reconnect on failure
- **User Data**: Health check every 60s, reconciliation every 5 min, listenKey renewal every 30 min

### Shutdown

1. Stop Market Data Manager (graceful close with 10s timeout)
2. Stop User Data Manager (closes all per-user connections with 30s timeout)
3. Close Kafka consumers
4. Close databases

## Error Handling

### Invalid Credentials (code=-2015)

When Binance returns `Invalid API-key, IP, or permissions for action`:

1. System automatically **deactivates** the exchange account
2. Logs error with account details
3. TODO: Sends Telegram notification to user (not yet implemented)

**Fix**: User must update credentials via `/manage_exchange` command in Telegram or reactivate with correct keys.

### Broken Pipe

When WebSocket connection breaks (`write tcp: broken pipe`):

1. Health check detects disconnection
2. Manager automatically attempts reconnection
3. Metrics track success/failure
4. System continues operating with fresh connection

## Troubleshooting

### Market Data not reconnecting?

Check logs for:
```bash
grep -i "market data" /path/to/logs
```

View metrics:
```bash
curl http://localhost:9090/metrics | grep marketdata_reconnects
```

### User Data not connecting?

1. **Check account is active**:
```sql
SELECT id, exchange, label, is_active FROM exchange_accounts;
```

2. **Check API key permissions on Binance**:
   - ✅ Enable Reading (required for User Data Stream)
   - ✅ Enable Spot & Margin Trading
   - ✅ IP Whitelist (add server IP or disable)

3. **View reconciliation logs**:
```bash
grep -i "hot reload\|reconciliation" /path/to/logs
```

### Adjust intervals

Edit `internal/bootstrap/websocket.go` or `internal/bootstrap/userdata_manager.go`:

```go
// Faster health checks (for testing)
HealthCheckInterval: 10 * time.Second,

// Faster reconciliation (for testing)
ReconciliationInterval: 30 * time.Second,
```

## Architecture

```
┌─────────────────────────────────────────┐
│         Bootstrap Container             │
├─────────────────────────────────────────┤
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   MarketDataManager               │ │
│  │  (centralized market data)        │ │
│  ├───────────────────────────────────┤ │
│  │ - Health Check Loop (60s)         │ │
│  │ - Auto-Reconnect                  │ │
│  │ - Binance/Bybit/OKX Client        │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   UserDataManager                 │ │
│  │  (per-user connections)           │ │
│  ├───────────────────────────────────┤ │
│  │ - Health Check Loop (60s)         │ │
│  │ - Hot Reload (5min)               │ │
│  │ - ListenKey Renewal (30min)       │ │
│  │ - Per-Account Connections         │ │
│  └───────────────────────────────────┘ │
│                                         │
└─────────────────────────────────────────┘
            ↓
    ┌──────────────┐
    │    Kafka     │ → Consumers → ClickHouse/Postgres
    └──────────────┘
```

## Performance

### Resource Usage

- **Market Data**: 1 connection per exchange (shared across all users)
- **User Data**: 1 connection per active exchange account
- **Memory**: ~2-5 MB per WebSocket connection
- **CPU**: Minimal (<1% per connection)

### Scalability

- Market Data: O(1) - one connection per exchange regardless of user count
- User Data: O(n) - scales linearly with number of active exchange accounts

### Metrics Overhead

- ~100 bytes per metric sample
- Exported every 10-15 seconds to Prometheus
- Minimal performance impact

## Future Improvements

1. **Backoff Strategy**: Exponential backoff for repeated failures (currently fixed 2s delay)
2. **Circuit Breaker**: Stop reconnect attempts after N consecutive failures
3. **Telegram Notifications**: Notify users when credentials are invalid
4. **Connection Pooling**: Reuse connections for multiple symbols (currently done)
5. **Compression**: Enable WebSocket compression for bandwidth savings
6. **Rate Limiting**: Respect exchange rate limits during reconnects





