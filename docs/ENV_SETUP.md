<!-- @format -->

# Environment Variables Setup

## Required Configuration

Copy the following to your `.env` file in the project root:

```bash
# App Configuration
APP_NAME=prometheus
APP_ENV=development
LOG_LEVEL=debug
DEBUG=true

# PostgreSQL Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=trading
POSTGRES_PASSWORD=secret
POSTGRES_DB=prometheus
POSTGRES_SSL_MODE=disable
POSTGRES_MAX_CONNS=25

# ClickHouse Configuration
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=
CLICKHOUSE_DB=trading

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=prometheus

# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_WEBHOOK_URL=
TELEGRAM_ADMIN_IDS=
TELEGRAM_DEBUG=false

# AI Provider API Keys
CLAUDE_API_KEY=sk-ant-your-claude-key-here
OPENAI_API_KEY=sk-your-openai-key-here
DEEPSEEK_API_KEY=sk-your-deepseek-key-here
GEMINI_API_KEY=your-gemini-key-here
DEFAULT_AI_PROVIDER=claude

# Encryption Key (32 bytes base64 encoded for AES-256)
# Generate with: openssl rand -base64 32
ENCRYPTION_KEY=your-32-byte-encryption-key-base64-encoded

# Central Market Data API Keys (for data collection, not user trading)
BINANCE_MARKET_DATA_API_KEY=your_binance_api_key
BINANCE_MARKET_DATA_SECRET=your_binance_secret
BYBIT_MARKET_DATA_API_KEY=your_bybit_api_key
BYBIT_MARKET_DATA_SECRET=your_bybit_secret
OKX_MARKET_DATA_API_KEY=your_okx_api_key
OKX_MARKET_DATA_SECRET=your_okx_secret
OKX_MARKET_DATA_PASSPHRASE=your_okx_passphrase

# Sentiment API Keys (Optional)
NEWS_API_KEY=                          # CryptoPanic API key (optional, has free tier)
TWITTER_API_KEY=                       # Twitter API v2 bearer token (optional, required for Twitter collector)
TWITTER_API_SECRET=                    # Twitter API secret
REDDIT_CLIENT_ID=                      # Reddit OAuth client ID (optional, free tier available)
REDDIT_CLIENT_SECRET=                  # Reddit OAuth client secret

# On-Chain Data API Keys (Optional)
WHALE_ALERT_API_KEY=                   # Whale Alert API (premium, ~$50/month)
CRYPTOQUANT_API_KEY=                   # CryptoQuant API (premium, plans from $49/month)
GLASSNODE_API_KEY=                     # Glassnode API (premium, plans from $29/month)
ETHERSCAN_API_KEY=                     # Etherscan API (free tier: 5 calls/sec, 100k calls/day)
BLOCKCHAIN_API_KEY=                    # Blockchain.com API (free, no auth required for stats)

# Macro/Traditional Markets API Keys (Optional)
TRADING_ECONOMICS_KEY=                 # Trading Economics API (premium, ~$500/month)
ALPHA_VANTAGE_KEY=                     # Alpha Vantage API (free tier: 25 calls/day)

# Derivatives Data API Keys (Optional)
DERIBIT_API_KEY=                       # Deribit API key (free, read-only for public endpoints)
DERIBIT_API_SECRET=                    # Deribit API secret

# Error Tracking Configuration (Sentry)
ERROR_TRACKING_ENABLED=true
ERROR_TRACKING_PROVIDER=sentry
SENTRY_DSN=https://your-sentry-dsn@sentry.io/project-id
SENTRY_ENVIRONMENT=production

# Market Data Collection Settings
# Base assets to track across all exchanges (comma-separated)
# Each exchange will format symbols according to their requirements (BTC/USDT or BTCUSDT)
MARKET_DATA_BASE_ASSETS=BTC,ETH,SOL,BNB,XRP
MARKET_DATA_QUOTE_CURRENCY=USDT
MARKET_DATA_EXCHANGES=binance,bybit,okx
MARKET_DATA_PRIMARY_EXCHANGE=binance
MARKET_DATA_TIMEFRAMES=1m,5m,15m,1h,4h,1d
MARKET_DATA_ORDERBOOK_DEPTH=20

# Sentiment Data Sources
SENTIMENT_TWITTER_ACCOUNTS=APompliano,100trillionUSD,saylor,elonmusk
SENTIMENT_REDDIT_SUBS=CryptoCurrency,Bitcoin,ethereum,ethtrader

# On-Chain Data Sources
ONCHAIN_BLOCKCHAINS=bitcoin,ethereum
ONCHAIN_MINING_POOLS=AntPool,Foundry USA,F2Pool,ViaBTC,Binance Pool
ONCHAIN_MIN_WHALE_AMOUNT=1000000  # Min whale transfer amount in USD

# Macro Data Sources
MACRO_COUNTRIES=United States,Euro Area,China
MACRO_EVENT_TYPES=CPI,NFP,FOMC,GDP
MACRO_TRADITIONAL_ASSETS=SPY,GLD,DXY,TLT
MACRO_CORRELATION_SYMBOLS=BTC/USDT,ETH/USDT,SOL/USDT

# Derivatives Data Sources
DERIVATIVES_OPTIONS_SYMBOLS=BTC,ETH
DERIVATIVES_MIN_OPTIONS_PREMIUM=100000  # Min options premium in USD

# Worker Intervals Configuration (Optional - defaults shown)
# Trading workers (high frequency - critical for execution)
WORKER_POSITION_MONITOR_INTERVAL=1m   # Check positions every minute
WORKER_ORDER_SYNC_INTERVAL=30s         # Sync orders every 30s
WORKER_RISK_MONITOR_INTERVAL=30s       # Check risk every 30s

# Market data workers (medium frequency)
WORKER_OHLCV_COLLECTOR_INTERVAL=1m     # Collect candles every minute

# Sentiment workers
WORKER_TWITTER_COLLECTOR_INTERVAL=5m   # Collect tweets every 5 minutes
WORKER_REDDIT_COLLECTOR_INTERVAL=10m   # Collect Reddit posts every 10 minutes
WORKER_FEARGREED_COLLECTOR_INTERVAL=1h # Collect Fear & Greed index every hour

# On-chain workers
WORKER_WHALE_MOVEMENT_COLLECTOR_INTERVAL=2m   # Track whale movements every 2 minutes
WORKER_EXCHANGE_FLOW_COLLECTOR_INTERVAL=5m    # Track exchange flows every 5 minutes
WORKER_NETWORK_METRICS_COLLECTOR_INTERVAL=10m # Collect network metrics every 10 minutes
WORKER_MINER_METRICS_COLLECTOR_INTERVAL=15m   # Collect miner metrics every 15 minutes

# Macro workers
WORKER_ECONOMIC_CALENDAR_COLLECTOR_INTERVAL=1h  # Check economic calendar every hour
WORKER_MARKET_CORRELATION_COLLECTOR_INTERVAL=30m # Calculate correlations every 30 minutes

# Derivatives workers
WORKER_OPTIONS_FLOW_COLLECTOR_INTERVAL=5m   # Track options trades every 5 minutes
WORKER_GAMMA_EXPOSURE_COLLECTOR_INTERVAL=10m # Calculate gamma exposure every 10 minutes
WORKER_FUNDING_AGGREGATOR_INTERVAL=5m        # Aggregate funding rates every 5 minutes

# Analysis workers (core agentic system)
WORKER_MARKET_SCANNER_INTERVAL=2m      # Full agent analysis every 2 minutes
WORKER_OPPORTUNITY_FINDER_INTERVAL=30s # Quick opportunity scan every 30s
WORKER_REGIME_DETECTOR_INTERVAL=5m     # Regime detection every 5 minutes
WORKER_SMC_SCANNER_INTERVAL=1m         # SMC patterns every minute

# Evaluation workers (low frequency)
WORKER_STRATEGY_EVALUATOR_INTERVAL=6h  # Evaluate strategies every 6 hours
WORKER_JOURNAL_COMPILER_INTERVAL=1h    # Compile journal every hour
WORKER_DAILY_REPORT_INTERVAL=24h       # Daily report at midnight

# Worker settings
WORKER_MARKET_SCANNER_MAX_CONCURRENCY=5    # Max users processed concurrently
WORKER_MARKET_SCANNER_EVENT_DRIVEN=true    # Enable event-driven mode for opportunities
```

## Environment Variable Details

### App Configuration

- `APP_NAME`: Application name (default: prometheus)
- `APP_ENV`: Environment (development/production)
- `LOG_LEVEL`: Logging level (debug/info/warn/error)
- `DEBUG`: Enable debug mode (true/false)

### Database Configuration

All database credentials are required for proper operation.

### API Keys

At least one AI provider API key is required. The system supports:

- Claude (Anthropic)
- OpenAI (GPT-4)
- DeepSeek
- Google Gemini

### Encryption

Generate a secure encryption key:

```bash
openssl rand -base64 32
```

This key is used to encrypt/decrypt exchange API keys in the database.

### Market Data API Keys

These are optional but recommended for better data collection performance.
If not provided, some data sources may not be available.

### Sentiment API Keys (Optional)

- **CryptoPanic** (NEWS_API_KEY): Free tier available. Sign up at https://cryptopanic.com/developers/api/
  - Free: 25 requests/day
  - Pro: $10/month for 10k requests/day
- **Twitter API** (TWITTER_API_KEY): Required for Twitter sentiment collector. Sign up at https://developer.twitter.com/
  - Essential: $100/month (10k tweets/month)
  - Elevated: Free tier available (500k tweets/month)
- **Reddit API** (REDDIT_CLIENT_ID): Free tier available. Create app at https://www.reddit.com/prefs/apps
  - Free: 60 requests/minute
  - No paid plans needed for basic usage

### On-Chain API Keys (Optional)

- **Whale Alert** (WHALE_ALERT_API_KEY): Premium service for large transaction tracking
  - Starter: $49/month (10k transactions/month)
  - Sign up: https://whale-alert.io/
- **CryptoQuant** (CRYPTOQUANT_API_KEY): Premium on-chain analytics
  - Starter: $49/month
  - Sign up: https://cryptoquant.com/
- **Glassnode** (GLASSNODE_API_KEY): Advanced on-chain metrics
  - Advanced: $29/month (limited metrics)
  - Professional: $799/month (full access)
  - Sign up: https://glassnode.com/
- **Etherscan** (ETHERSCAN_API_KEY): Free Ethereum blockchain data
  - Free: 5 calls/sec, 100k calls/day
  - Sign up: https://etherscan.io/apis

### Macro/Traditional Markets API Keys (Optional)

- **Trading Economics** (TRADING_ECONOMICS_KEY): Economic calendar and indicators
  - Individual: $500/month
  - Sign up: https://tradingeconomics.com/api
- **Alpha Vantage** (ALPHA_VANTAGE_KEY): Stock market and forex data
  - Free: 25 API calls/day
  - Premium: $49.99/month (75 calls/minute)
  - Sign up: https://www.alphavantage.co/support/#api-key

### Derivatives API Keys (Optional)

- **Deribit** (DERIBIT_API_KEY): Crypto options and futures data
  - Free for public market data endpoints
  - Authentication required for historical data
  - Sign up: https://www.deribit.com/

### Worker Intervals

Worker intervals control how frequently background workers execute. All intervals accept duration strings like:

- `30s` - 30 seconds
- `1m` - 1 minute
- `5m` - 5 minutes
- `1h` - 1 hour
- `24h` - 24 hours

#### Trading Workers (High Frequency)

- `WORKER_POSITION_MONITOR_INTERVAL`: How often to check open positions and update PnL (default: 1m)
- `WORKER_ORDER_SYNC_INTERVAL`: How often to sync order status with exchanges (default: 30s)
- `WORKER_RISK_MONITOR_INTERVAL`: How often to check risk limits and circuit breakers (default: 30s)

#### Market Data Workers

- `WORKER_OHLCV_COLLECTOR_INTERVAL`: How often to collect OHLCV candles from exchanges (default: 1m)

#### Analysis Workers (Agentic System)

- `WORKER_MARKET_SCANNER_INTERVAL`: How often to run full agent analysis for all users (default: 2m)
- `WORKER_OPPORTUNITY_FINDER_INTERVAL`: How often to scan for quick trading opportunities (default: 30s)
- `WORKER_REGIME_DETECTOR_INTERVAL`: How often to detect market regime changes (default: 5m)
- `WORKER_SMC_SCANNER_INTERVAL`: How often to scan for Smart Money Concepts patterns (default: 1m)

#### Evaluation Workers (Low Frequency)

- `WORKER_STRATEGY_EVALUATOR_INTERVAL`: How often to evaluate strategy performance (default: 6h)
- `WORKER_JOURNAL_COMPILER_INTERVAL`: How often to compile journal entries (default: 1h)
- `WORKER_DAILY_REPORT_INTERVAL`: How often to generate daily reports (default: 24h)

#### Worker Settings

- `WORKER_MARKET_SCANNER_MAX_CONCURRENCY`: Maximum number of users to process concurrently (default: 5)
- `WORKER_MARKET_SCANNER_EVENT_DRIVEN`: Enable event-driven mode for immediate response to opportunities (default: true)

**Event-Driven Mode**: When enabled, MarketScanner operates in hybrid mode:

- **Scheduled**: Runs full analysis periodically (every 2 minutes)
- **Event-driven**: Responds immediately to opportunity events from OpportunityFinder (<30s response time)

This allows the system to balance comprehensive analysis with quick reactions to market opportunities.

### WebSocket Configuration (Real-time Market Data)

WebSocket connections provide real-time market data streams for low-latency analysis:

```bash
# WebSocket Settings
WEBSOCKET_ENABLED=true                                  # Enable/disable WebSocket connections
WEBSOCKET_USE_TESTNET=false                            # Use testnet endpoints for testing
WEBSOCKET_SYMBOLS=BTCUSDT,ETHUSDT,SOLUSDT              # Symbols to stream (comma-separated)
WEBSOCKET_INTERVALS=1m,5m,15m,1h,4h                    # Timeframes to collect (comma-separated)
WEBSOCKET_STREAM_TYPES=kline,ticker,trade,markPrice    # Stream types (comma-separated)
WEBSOCKET_MARKET_TYPES=spot,futures                    # Market types to collect (comma-separated)
WEBSOCKET_EXCHANGES=binance                             # Exchanges to connect (comma-separated)
WEBSOCKET_RECONNECT_BACKOFF=5s                         # Backoff between reconnection attempts
WEBSOCKET_MAX_RECONNECTS=10                            # Maximum reconnection attempts before giving up
WEBSOCKET_PING_INTERVAL=60s                            # Ping interval to keep connection alive
WEBSOCKET_READ_BUFFER_SIZE=4096                        # Read buffer size in bytes
WEBSOCKET_WRITE_BUFFER_SIZE=4096                       # Write buffer size in bytes
```

**Supported Timeframes**: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`

**Supported Stream Types**: `kline`, `ticker`, `trade`, `markPrice`, `depth`

**Supported Market Types**:

- `spot` - Spot markets (no leverage, direct asset ownership)
- `futures` - Futures/perpetual markets (leverage, funding rates)

**Notes**:

- **Public streams** (klines, ticker, depth, trades) don't require API keys
- **Private streams** (orders, positions) require `BINANCE_MARKET_DATA_API_KEY` and `BINANCE_MARKET_DATA_SECRET`
- **markPrice stream** is only available for `futures` market type (includes mark price, index price, funding rate)
- Spot and futures use different WebSocket endpoints and have different data structures
- Events are published to Kafka and consumed by batch writer to ClickHouse
- Data is automatically tagged with `market_type` field for proper differentiation in ClickHouse
- Deduplication happens at batch level to avoid redundant writes
- See `docs/WEBSOCKET_AUTH.md` for authentication details
