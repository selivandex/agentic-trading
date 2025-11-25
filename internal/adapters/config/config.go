package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	App           AppConfig
	Postgres      PostgresConfig
	ClickHouse    ClickHouseConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Telegram      TelegramConfig
	AI            AIConfig
	Crypto        CryptoConfig
	MarketData    MarketDataConfig
	ErrorTracking ErrorTrackingConfig
}

type AppConfig struct {
	Name     string `envconfig:"APP_NAME" default:"prometheus"`
	Env      string `envconfig:"APP_ENV" default:"development"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	Debug    bool   `envconfig:"DEBUG" default:"false"`
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
	ClaudeKey       string `envconfig:"CLAUDE_API_KEY"`
	OpenAIKey       string `envconfig:"OPENAI_API_KEY"`
	DeepSeekKey     string `envconfig:"DEEPSEEK_API_KEY"`
	GeminiKey       string `envconfig:"GEMINI_API_KEY"`
	DefaultProvider string `envconfig:"DEFAULT_AI_PROVIDER" default:"claude"`
}

type CryptoConfig struct {
	EncryptionKey string `envconfig:"ENCRYPTION_KEY" required:"true"` // 32 bytes for AES-256
}

type MarketDataConfig struct {
	// Central API keys for market data collection (not user-specific)
	BinanceAPIKey    string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
	BinanceSecret    string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
	BybitAPIKey      string `envconfig:"BYBIT_MARKET_DATA_API_KEY"`
	BybitSecret      string `envconfig:"BYBIT_MARKET_DATA_SECRET"`
	OKXAPIKey        string `envconfig:"OKX_MARKET_DATA_API_KEY"`
	OKXSecret        string `envconfig:"OKX_MARKET_DATA_SECRET"`
	OKXPassphrase    string `envconfig:"OKX_MARKET_DATA_PASSPHRASE"`
}

type ErrorTrackingConfig struct {
	Enabled     bool   `envconfig:"ERROR_TRACKING_ENABLED" default:"true"`
	Provider    string `envconfig:"ERROR_TRACKING_PROVIDER" default:"sentry"`
	SentryDSN   string `envconfig:"SENTRY_DSN"`
	Environment string `envconfig:"SENTRY_ENVIRONMENT" default:"production"`
}

// Load reads configuration from environment variables
// It first tries to load .env file (useful for local development)
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if not exists)
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	return &cfg, nil
}


import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	App           AppConfig
	Postgres      PostgresConfig
	ClickHouse    ClickHouseConfig
	Redis         RedisConfig
	Kafka         KafkaConfig
	Telegram      TelegramConfig
	AI            AIConfig
	Crypto        CryptoConfig
	MarketData    MarketDataConfig
	ErrorTracking ErrorTrackingConfig
}

type AppConfig struct {
	Name     string `envconfig:"APP_NAME" default:"prometheus"`
	Env      string `envconfig:"APP_ENV" default:"development"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	Debug    bool   `envconfig:"DEBUG" default:"false"`
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
	ClaudeKey       string `envconfig:"CLAUDE_API_KEY"`
	OpenAIKey       string `envconfig:"OPENAI_API_KEY"`
	DeepSeekKey     string `envconfig:"DEEPSEEK_API_KEY"`
	GeminiKey       string `envconfig:"GEMINI_API_KEY"`
	DefaultProvider string `envconfig:"DEFAULT_AI_PROVIDER" default:"claude"`
}

type CryptoConfig struct {
	EncryptionKey string `envconfig:"ENCRYPTION_KEY" required:"true"` // 32 bytes for AES-256
}

type MarketDataConfig struct {
	// Central API keys for market data collection (not user-specific)
	BinanceAPIKey    string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
	BinanceSecret    string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
	BybitAPIKey      string `envconfig:"BYBIT_MARKET_DATA_API_KEY"`
	BybitSecret      string `envconfig:"BYBIT_MARKET_DATA_SECRET"`
	OKXAPIKey        string `envconfig:"OKX_MARKET_DATA_API_KEY"`
	OKXSecret        string `envconfig:"OKX_MARKET_DATA_SECRET"`
	OKXPassphrase    string `envconfig:"OKX_MARKET_DATA_PASSPHRASE"`
}

type ErrorTrackingConfig struct {
	Enabled     bool   `envconfig:"ERROR_TRACKING_ENABLED" default:"true"`
	Provider    string `envconfig:"ERROR_TRACKING_PROVIDER" default:"sentry"`
	SentryDSN   string `envconfig:"SENTRY_DSN"`
	Environment string `envconfig:"SENTRY_ENVIRONMENT" default:"production"`
}

// Load reads configuration from environment variables
// It first tries to load .env file (useful for local development)
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if not exists)
	_ = godotenv.Load()

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	return &cfg, nil
}

