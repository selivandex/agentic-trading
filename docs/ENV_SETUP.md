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

