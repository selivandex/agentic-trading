# Testnet Configuration Guide

This guide explains how to configure testnet mode for user exchange accounts in the Prometheus trading system.

## Overview

The system supports testnet mode for all major exchanges (Binance, Bybit, OKX). When testnet mode is enabled:
- User exchange accounts are created with testnet credentials
- WebSocket connections (including User Data streams) connect to testnet endpoints
- ListenKeys are obtained from testnet REST API
- All trading operations execute on testnet environments

## Configuration

Testnet mode is configured per exchange using environment variables:

### Binance Testnet

```bash
# Enable Binance testnet for all user accounts
BINANCE_USE_TESTNET=true

# Binance testnet credentials (optional, for market data collection)
BINANCE_MARKET_DATA_API_KEY=your_testnet_api_key
BINANCE_MARKET_DATA_SECRET=your_testnet_secret
```

**Binance Testnet URLs:**
- REST API: `https://testnet.binancefuture.com`
- WebSocket: `wss://stream.binancefuture.com`
- Futures testnet: `https://testnet.binance.vision`

### Bybit Testnet

```bash
# Enable Bybit testnet for all user accounts
BYBIT_USE_TESTNET=true

# Bybit testnet credentials (optional, for market data collection)
BYBIT_MARKET_DATA_API_KEY=your_testnet_api_key
BYBIT_MARKET_DATA_SECRET=your_testnet_secret
```

**Bybit Testnet URLs:**
- REST API: `https://api-testnet.bybit.com`
- WebSocket: `wss://stream-testnet.bybit.com`

### OKX Testnet

```bash
# Enable OKX testnet for all user accounts
OKX_USE_TESTNET=true

# OKX testnet credentials (optional, for market data collection)
OKX_MARKET_DATA_API_KEY=your_testnet_api_key
OKX_MARKET_DATA_SECRET=your_testnet_secret
OKX_MARKET_DATA_PASSPHRASE=your_testnet_passphrase
```

**OKX Testnet URLs:**
- REST API: `https://www.okx.com` (with simulated trading flag)
- WebSocket: `wss://wspap.okx.com:8443/ws/v5/public?brokerId=9999`

## How It Works

### Architecture Flow

1. **User Account Creation (Telegram Bot)**
   - User runs `/add_exchange` command
   - Enters API credentials
   - System checks `BINANCE_USE_TESTNET` / `BYBIT_USE_TESTNET` / `OKX_USE_TESTNET` config
   - Creates `ExchangeAccount` with `is_testnet` flag set accordingly

2. **User Data WebSocket Manager**
   - On startup, loads all active accounts from database
   - For each account, reads `is_testnet` flag
   - Creates exchange-specific client with testnet mode:
     ```go
     client := binance.NewUserDataClient(
         accountID,
         userID,
         "binance",
         handler,
         account.IsTestnet, // ← from database
         log,
     )
     ```

3. **ListenKey Management**
   - `ListenKeyService` uses `futures.UseTestnet` global flag
   - When `is_testnet=true`, automatically connects to:
     - Binance: `POST https://testnet.binancefuture.com/fapi/v1/listenKey`
     - Binance: `wss://stream.binancefuture.com/ws/{listenKey}`

4. **WebSocket Connection**
   - go-binance library checks `futures.UseTestnet` flag
   - Routes WebSocket to correct endpoint automatically

### Code References

**Config Definition:**
```go
// internal/adapters/config/config.go
type BinanceConfig struct {
    APIKey     string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
    Secret     string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
    UseTestnet bool   `envconfig:"BINANCE_USE_TESTNET" default:"false"`
}
```

**Exchange Account Creation:**
```go
// internal/adapters/telegram/exchange_setup.go
// Set testnet flag based on exchange and config
switch exchangeType {
case exchange_account.ExchangeBinance:
    session.IsTestnet = es.binanceTestnet
case exchange_account.ExchangeBybit:
    session.IsTestnet = es.bybitTestnet
case exchange_account.ExchangeOKX:
    session.IsTestnet = es.okxTestnet
}
```

**ListenKey Service:**
```go
// internal/adapters/websocket/binance/listenkey_service.go
func (s *ListenKeyService) Create(ctx context.Context, apiKey, secret string) (string, time.Time, error) {
    futures.UseTestnet = s.useTestnet // ← Set global flag
    client := futures.NewClient(apiKey, secret)
    listenKey, err := client.NewStartUserStreamService().Do(ctx)
    // ...
}
```

## Testing on Binance Testnet

### 1. Get Testnet API Keys

Visit: https://testnet.binancefuture.com/

1. Login with your Binance account
2. Go to API Management
3. Create new API key
4. Enable futures trading permissions
5. Note down your API key and secret

### 2. Configure Environment

```bash
# .env file
BINANCE_USE_TESTNET=true
```

### 3. Add Exchange Account via Telegram

```
/add_exchange
→ Select Binance
→ Enter testnet API key
→ Enter testnet secret
→ Enter label (e.g., "Binance Testnet")
```

### 4. Verify Connection

Check logs for:
```
✅ User Data WebSocket connected and streaming
    account_id=xxx
    exchange=binance
    testnet=true
```

### 5. Test User Data Stream

The system will automatically:
- Create listenKey from testnet API
- Connect to testnet WebSocket
- Receive order updates, position updates, balance updates
- Renew listenKey every 30 minutes

## Troubleshooting

### Problem: ListenKey Creation Fails

**Error:**
```
Failed to create listenKey from exchange
code=-2015: Invalid API-key, IP, or permissions
```

**Solution:**
- Verify API key is from testnet (not production)
- Check API key has "Enable Futures" permission
- Ensure IP restrictions allow your server IP
- Confirm testnet flag is set: `BINANCE_USE_TESTNET=true`

### Problem: WebSocket Disconnects Immediately

**Error:**
```
User Data WebSocket error: websocket: close 1006
```

**Solution:**
- Verify listenKey was created successfully
- Check testnet WebSocket endpoint is reachable
- Ensure `futures.UseTestnet = true` is set before connection
- Check logs for "Creating Binance listenKey" with `testnet=true`

### Problem: Production Keys Used on Testnet

**Symptom:** Trades don't execute, positions not created

**Solution:**
- Production API keys don't work on testnet
- Must create separate testnet API keys from testnet website
- Check account `is_testnet` flag in database: `SELECT is_testnet FROM exchange_accounts;`

## Database Schema

```sql
-- Check if account is configured for testnet
SELECT id, user_id, exchange, label, is_testnet, is_active
FROM exchange_accounts
WHERE user_id = 'xxx';

-- Update existing account to testnet (not recommended in production)
UPDATE exchange_accounts
SET is_testnet = true
WHERE id = 'xxx';
```

## Environment Variables Reference

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `BINANCE_USE_TESTNET` | Use Binance testnet for user accounts | `false` | `true` |
| `BYBIT_USE_TESTNET` | Use Bybit testnet for user accounts | `false` | `true` |
| `OKX_USE_TESTNET` | Use OKX testnet for user accounts | `false` | `true` |
| `WEBSOCKET_USE_TESTNET` | Use testnet for public market data streams | `false` | `true` |

**Note:** `BINANCE_USE_TESTNET` is for **user accounts** (authenticated), while `WEBSOCKET_USE_TESTNET` is for **public market data** (unauthenticated).

## Production Recommendations

1. **Never enable testnet in production** - Keep `BINANCE_USE_TESTNET=false` in production
2. **Separate environments** - Use different databases for testnet and production
3. **Clear labeling** - Label accounts clearly: "Binance Mainnet" vs "Binance Testnet"
4. **Monitoring** - Set up alerts for testnet accounts appearing in production
5. **Testing workflow** - Always test new features on testnet first

## Further Reading

- [Binance Futures Testnet Documentation](https://testnet.binancefuture.com/en/futures/BTCUSDT)
- [Bybit Testnet Documentation](https://testnet.bybit.com/)
- [OKX Demo Trading](https://www.okx.com/trade-demo)
- [User Data WebSocket Architecture](./WEBSOCKET_AUTH.md)


