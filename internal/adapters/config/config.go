package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"prometheus/pkg/errors"
)

type Config struct {
	App           AppConfig
	HTTP          HTTPConfig
	Postgres      PostgresConfig
	ClickHouse    ClickHouseConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Telegram      TelegramConfig
	AI            AIConfig
	Agents        AgentsConfig
	Crypto        CryptoConfig
	MarketData    MarketDataConfig
	ErrorTracking ErrorTrackingConfig
	Workers       WorkerConfig
}

type AppConfig struct {
	Name     string `envconfig:"APP_NAME" default:"prometheus"`
	Env      string `envconfig:"APP_ENV" default:"development"`
	Version  string `envconfig:"APP_VERSION" default:"1.0.0"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	Debug    bool   `envconfig:"DEBUG" default:"false"`
}

type HTTPConfig struct {
	Port int `envconfig:"HTTP_PORT" default:"8080"`
}

type PostgresConfig struct {
	Host     string `envconfig:"POSTGRES_HOST" required:"true"`
	Port     int    `envconfig:"POSTGRES_PORT" default:"5432"`
	User     string `envconfig:"POSTGRES_USER" required:"true"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	Database string `envconfig:"POSTGRES_DB" required:"true"`
	SSLMode  string `envconfig:"POSTGRES_SSL_MODE" default:"disable"`
	MaxConns int    `envconfig:"POSTGRES_MAX_CONNS" default:"25"`
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

type ClickHouseConfig struct {
	Host     string `envconfig:"CLICKHOUSE_HOST" required:"true"`
	Port     int    `envconfig:"CLICKHOUSE_PORT" default:"9000"`
	User     string `envconfig:"CLICKHOUSE_USER" default:"default"`
	Password string `envconfig:"CLICKHOUSE_PASSWORD"`
	Database string `envconfig:"CLICKHOUSE_DB" default:"trading"`
}

type RedisConfig struct {
	Host     string `envconfig:"REDIS_HOST" required:"true"`
	Port     int    `envconfig:"REDIS_PORT" default:"6379"`
	Password string `envconfig:"REDIS_PASSWORD"`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type KafkaConfig struct {
	Brokers []string `envconfig:"KAFKA_BROKERS" required:"true"`
	GroupID string   `envconfig:"KAFKA_GROUP_ID" default:"prometheus"`
}

type TelegramConfig struct {
	BotToken   string  `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`
	WebhookURL string  `envconfig:"TELEGRAM_WEBHOOK_URL"`
	AdminIDs   []int64 `envconfig:"TELEGRAM_ADMIN_IDS"`
}

type AIConfig struct {
	ClaudeKey           string `envconfig:"CLAUDE_API_KEY"`
	OpenAIKey           string `envconfig:"OPENAI_API_KEY"`
	DeepSeekKey         string `envconfig:"DEEPSEEK_API_KEY"`
	GeminiKey           string `envconfig:"GEMINI_API_KEY"`
	DefaultProvider     string `envconfig:"DEFAULT_AI_PROVIDER" default:"claude"`
	MaxDailyCostPerUser string `envconfig:"AI_MAX_DAILY_COST_PER_USER" default:"10.00"` // Max daily AI spending per user (USD)
	MaxCostPerExecution string `envconfig:"AI_MAX_COST_PER_EXECUTION" default:"1.00"`   // Max cost per single agent execution (USD)
}

// AgentsConfig contains configuration for agent execution
type AgentsConfig struct {
	MaxTokens              int           `envconfig:"AGENTS_MAX_TOKENS" default:"50000"`             // Max tokens for conversation history
	ExecutionTimeout       time.Duration `envconfig:"AGENTS_EXECUTION_TIMEOUT" default:"5m"`         // Default timeout for agent execution
	MaxToolCalls           int           `envconfig:"AGENTS_MAX_TOOL_CALLS" default:"25"`            // Max tool calls per agent run
	EnableCompression      bool          `envconfig:"AGENTS_ENABLE_COMPRESSION" default:"true"`      // Enable conversation compression
	EnableMemory           bool          `envconfig:"AGENTS_ENABLE_MEMORY" default:"true"`           // Enable long-term memory
	SelfReflectionInterval time.Duration `envconfig:"AGENTS_SELF_REFLECTION_INTERVAL" default:"30s"` // Self-reflection check interval

	// ADK-specific configuration
	EnablePersistentSessions bool          `envconfig:"AGENTS_ENABLE_PERSISTENT_SESSIONS" default:"true"` // Use database-backed sessions
	EnableStreaming          bool          `envconfig:"AGENTS_ENABLE_STREAMING" default:"true"`           // Enable SSE streaming
	EnableCallbacks          bool          `envconfig:"AGENTS_ENABLE_CALLBACKS" default:"true"`           // Enable lifecycle callbacks
	EnableResponseCaching    bool          `envconfig:"AGENTS_ENABLE_RESPONSE_CACHING" default:"true"`    // Cache LLM responses in Redis
	ResponseCacheTTL         time.Duration `envconfig:"AGENTS_RESPONSE_CACHE_TTL" default:"1h"`           // Cache TTL
	EnableAgentTransfer      bool          `envconfig:"AGENTS_ENABLE_AGENT_TRANSFER" default:"true"`      // Allow agents to transfer control
	EnableSchemaValidation   bool          `envconfig:"AGENTS_ENABLE_SCHEMA_VALIDATION" default:"true"`   // Validate I/O with schemas
	EnableExpertTools        bool          `envconfig:"AGENTS_ENABLE_EXPERT_TOOLS" default:"true"`        // Enable agent-as-tool pattern
}

type CryptoConfig struct {
	EncryptionKey string `envconfig:"ENCRYPTION_KEY" required:"true"` // 32 bytes for AES-256
}

// BinanceConfig contains Binance-specific market data credentials
type BinanceConfig struct {
	APIKey string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
	Secret string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
}

// BybitMarketDataConfig contains Bybit-specific market data credentials
type BybitConfig struct {
	APIKey string `envconfig:"BYBIT_MARKET_DATA_API_KEY"`
	Secret string `envconfig:"BYBIT_MARKET_DATA_SECRET"`
}

// OKXMarketDataConfig contains OKX-specific market data credentials
type OKXConfig struct {
	APIKey     string `envconfig:"OKX_MARKET_DATA_API_KEY"`
	Secret     string `envconfig:"OKX_MARKET_DATA_SECRET"`
	Passphrase string `envconfig:"OKX_MARKET_DATA_PASSPHRASE"`
}

// MarketDataConfig contains centralized market data collection credentials
// These are used by workers to collect market data, separate from user trading accounts
type MarketDataConfig struct {
	Binance BinanceConfig
	Bybit   BybitConfig
	OKX     OKXConfig

	// News/sentiment API keys
	NewsAPIKey         string `envconfig:"NEWS_API_KEY"`         // CryptoPanic API key (optional for public endpoint)
	TwitterAPIKey      string `envconfig:"TWITTER_API_KEY"`      // Twitter API v2 bearer token
	TwitterAPISecret   string `envconfig:"TWITTER_API_SECRET"`   // Twitter API secret
	RedditClientID     string `envconfig:"REDDIT_CLIENT_ID"`     // Reddit OAuth client ID
	RedditClientSecret string `envconfig:"REDDIT_CLIENT_SECRET"` // Reddit OAuth client secret

	// On-chain API keys
	WhaleAlertAPIKey  string `envconfig:"WHALE_ALERT_API_KEY"` // Whale Alert API (premium)
	CryptoquantAPIKey string `envconfig:"CRYPTOQUANT_API_KEY"` // CryptoQuant API
	GlassnodeAPIKey   string `envconfig:"GLASSNODE_API_KEY"`   // Glassnode API
	EtherscanAPIKey   string `envconfig:"ETHERSCAN_API_KEY"`   // Etherscan API (free tier)
	BlockchainAPIKey  string `envconfig:"BLOCKCHAIN_API_KEY"`  // Blockchain.com API

	// Macro API keys
	TradingEconomicsKey string `envconfig:"TRADING_ECONOMICS_KEY"` // Trading Economics API
	AlphaVantageKey     string `envconfig:"ALPHA_VANTAGE_KEY"`     // Alpha Vantage API (free tier)

	// Derivatives API keys
	DeribitAPIKey    string `envconfig:"DERIBIT_API_KEY"`    // Deribit API key
	DeribitAPISecret string `envconfig:"DERIBIT_API_SECRET"` // Deribit API secret
}

type ErrorTrackingConfig struct {
	Enabled     bool   `envconfig:"ERROR_TRACKING_ENABLED" default:"true"`
	Provider    string `envconfig:"ERROR_TRACKING_PROVIDER" default:"sentry"`
	SentryDSN   string `envconfig:"SENTRY_DSN"`
	Environment string `envconfig:"SENTRY_ENVIRONMENT" default:"production"`
}

// WorkerConfig contains intervals for all background workers
// Intervals are optimized for production with reasonable defaults
// that balance responsiveness with resource usage and API rate limits
type WorkerConfig struct {
	// Trading workers (high frequency - critical for execution)
	PositionMonitorInterval time.Duration `envconfig:"WORKER_POSITION_MONITOR_INTERVAL" default:"1m"` // Check positions every minute
	OrderSyncInterval       time.Duration `envconfig:"WORKER_ORDER_SYNC_INTERVAL" default:"30s"`      // Sync orders every 30s
	RiskMonitorInterval     time.Duration `envconfig:"WORKER_RISK_MONITOR_INTERVAL" default:"30s"`    // Check risk every 30s
	PnLCalculatorInterval   time.Duration `envconfig:"WORKER_PNL_CALCULATOR_INTERVAL" default:"5m"`   // Calculate PnL every 5 minutes

	// Market data workers (medium frequency)
	OHLCVCollectorInterval     time.Duration `envconfig:"WORKER_OHLCV_COLLECTOR_INTERVAL" default:"1m"`      // Collect candles every minute
	TickerCollectorInterval    time.Duration `envconfig:"WORKER_TICKER_COLLECTOR_INTERVAL" default:"10s"`    // Collect tickers every 10 seconds
	OrderBookCollectorInterval time.Duration `envconfig:"WORKER_ORDERBOOK_COLLECTOR_INTERVAL" default:"10s"` // Collect order books every 10 seconds
	TradesCollectorInterval    time.Duration `envconfig:"WORKER_TRADES_COLLECTOR_INTERVAL" default:"1s"`     // Collect trades every second
	FundingCollectorInterval   time.Duration `envconfig:"WORKER_FUNDING_COLLECTOR_INTERVAL" default:"1m"`    // Collect funding rates every minute

	// Sentiment workers (low frequency)
	NewsCollectorInterval      time.Duration `envconfig:"WORKER_NEWS_COLLECTOR_INTERVAL" default:"10m"`     // Collect news every 10 minutes
	TwitterCollectorInterval   time.Duration `envconfig:"WORKER_TWITTER_COLLECTOR_INTERVAL" default:"5m"`   // Collect tweets every 5 minutes
	RedditCollectorInterval    time.Duration `envconfig:"WORKER_REDDIT_COLLECTOR_INTERVAL" default:"10m"`   // Collect Reddit posts every 10 minutes
	FearGreedCollectorInterval time.Duration `envconfig:"WORKER_FEARGREED_COLLECTOR_INTERVAL" default:"1h"` // Collect Fear & Greed index every hour

	// On-chain workers (medium frequency)
	WhaleMovementCollectorInterval  time.Duration `envconfig:"WORKER_WHALE_MOVEMENT_COLLECTOR_INTERVAL" default:"2m"`   // Track whale movements every 2 minutes
	ExchangeFlowCollectorInterval   time.Duration `envconfig:"WORKER_EXCHANGE_FLOW_COLLECTOR_INTERVAL" default:"5m"`    // Track exchange flows every 5 minutes
	NetworkMetricsCollectorInterval time.Duration `envconfig:"WORKER_NETWORK_METRICS_COLLECTOR_INTERVAL" default:"10m"` // Collect network metrics every 10 minutes
	MinerMetricsCollectorInterval   time.Duration `envconfig:"WORKER_MINER_METRICS_COLLECTOR_INTERVAL" default:"15m"`   // Collect miner metrics every 15 minutes

	// Macro workers (low frequency)
	EconomicCalendarCollectorInterval  time.Duration `envconfig:"WORKER_ECONOMIC_CALENDAR_COLLECTOR_INTERVAL" default:"1h"`   // Check economic calendar every hour
	MarketCorrelationCollectorInterval time.Duration `envconfig:"WORKER_MARKET_CORRELATION_COLLECTOR_INTERVAL" default:"30m"` // Calculate correlations every 30 minutes

	// Derivatives workers (medium frequency)
	OptionsFlowCollectorInterval   time.Duration `envconfig:"WORKER_OPTIONS_FLOW_COLLECTOR_INTERVAL" default:"5m"`    // Track large options trades every 5 minutes
	GammaExposureCollectorInterval time.Duration `envconfig:"WORKER_GAMMA_EXPOSURE_COLLECTOR_INTERVAL" default:"10m"` // Calculate gamma exposure every 10 minutes
	FundingAggregatorInterval      time.Duration `envconfig:"WORKER_FUNDING_AGGREGATOR_INTERVAL" default:"5m"`        // Aggregate funding rates every 5 minutes

	// Analysis workers (core agentic system)
	MarketScannerInterval     time.Duration `envconfig:"WORKER_MARKET_SCANNER_INTERVAL" default:"2m"`      // Full agent analysis every 2 minutes
	OpportunityFinderInterval time.Duration `envconfig:"WORKER_OPPORTUNITY_FINDER_INTERVAL" default:"30s"` // Quick opportunity scan every 30s
	RegimeDetectorInterval    time.Duration `envconfig:"WORKER_REGIME_DETECTOR_INTERVAL" default:"5m"`     // Regime detection every 5 minutes
	SMCScannerInterval        time.Duration `envconfig:"WORKER_SMC_SCANNER_INTERVAL" default:"1m"`         // SMC patterns every minute

	// Evaluation workers (low frequency)
	StrategyEvaluatorInterval time.Duration `envconfig:"WORKER_STRATEGY_EVALUATOR_INTERVAL" default:"6h"` // Evaluate strategies every 6 hours
	JournalCompilerInterval   time.Duration `envconfig:"WORKER_JOURNAL_COMPILER_INTERVAL" default:"1h"`   // Compile journal every hour
	DailyReportInterval       time.Duration `envconfig:"WORKER_DAILY_REPORT_INTERVAL" default:"24h"`      // Daily report at midnight

	// Worker concurrency settings
	MarketScannerMaxConcurrency int  `envconfig:"WORKER_MARKET_SCANNER_MAX_CONCURRENCY" default:"5"` // Max users processed concurrently
	MarketScannerEventDriven    bool `envconfig:"WORKER_MARKET_SCANNER_EVENT_DRIVEN" default:"true"` // Enable event-driven mode for opportunities
}

// Load reads configuration from environment variables
// It first tries to load .env file (useful for local development)
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if not exists)
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, errors.Wrap(err, "failed to process env config")
	}

	return &cfg, nil
}
