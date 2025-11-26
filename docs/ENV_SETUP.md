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

# Error Tracking Configuration (Sentry)
ERROR_TRACKING_ENABLED=true
ERROR_TRACKING_PROVIDER=sentry
SENTRY_DSN=https://your-sentry-dsn@sentry.io/project-id
SENTRY_ENVIRONMENT=production

# Worker Intervals Configuration (Optional - defaults shown)
# Trading workers (high frequency - critical for execution)
WORKER_POSITION_MONITOR_INTERVAL=1m   # Check positions every minute
WORKER_ORDER_SYNC_INTERVAL=30s         # Sync orders every 30s
WORKER_RISK_MONITOR_INTERVAL=30s       # Check risk every 30s

# Market data workers (medium frequency)
WORKER_OHLCV_COLLECTOR_INTERVAL=1m     # Collect candles every minute

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

