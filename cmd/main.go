package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"prometheus/internal/adapters/ai"
	chclient "prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/config"
	errnoop "prometheus/internal/adapters/errors/noop"
	"prometheus/internal/adapters/errors/sentry"
	"prometheus/internal/adapters/exchangefactory"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	pgclient "prometheus/internal/adapters/postgres"
	redisclient "prometheus/internal/adapters/redis"
	"prometheus/internal/agents"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	domainRisk "prometheus/internal/domain/risk"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	chrepo "prometheus/internal/repository/clickhouse"
	pgrepo "prometheus/internal/repository/postgres"
	riskengine "prometheus/internal/risk"
	"prometheus/internal/tools"
	"prometheus/internal/tools/shared"
	"prometheus/internal/workers"
	"prometheus/internal/workers/marketdata"
	"prometheus/internal/workers/trading"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

type Database struct {
	Postgres   *pgclient.Client
	ClickHouse *chclient.Client
	Redis      *redisclient.Client
}

func (d *Database) Close(log *logger.Logger) {
	if d == nil {
		return
	}
	if d.Postgres != nil {
		if err := d.Postgres.Close(); err != nil {
			log.Warnf("failed to close postgres: %v", err)
		}
	}
	if d.ClickHouse != nil {
		if err := d.ClickHouse.Close(); err != nil {
			log.Warnf("failed to close clickhouse: %v", err)
		}
	}
	if d.Redis != nil {
		if err := d.Redis.Close(); err != nil {
			log.Warnf("failed to close redis: %v", err)
		}
	}
}

type Repositories struct {
	User            user.Repository
	ExchangeAccount exchange_account.Repository
	TradingPair     trading_pair.Repository
	Order           order.Repository
	Position        position.Repository
	Memory          memory.Repository
	Journal         journal.Repository
	MarketData      market_data.Repository
	Risk            domainRisk.Repository
}

type Services struct {
	User            *user.Service
	ExchangeAccount *exchange_account.Service
	TradingPair     *trading_pair.Service
	Order           *order.Service
	Position        *position.Service
	Memory          *memory.Service
	Journal         *journal.Service
}

type AgentSystem struct {
	Factory         *agents.Factory
	Registry        *agents.Registry
	ModelSelector   *ai.ModelSelector
	DefaultProvider string
	DefaultModel    string
}

func main() {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Initialize logger
	if err := initLogger(cfg); err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer logger.Sync()

	log := logger.Get()
	log.Infof("Starting %s in %s mode", cfg.App.Name, cfg.App.Env)

	// Initialize error tracker
	errorTracker := initErrorTracker(cfg, log)
	logger.SetErrorTracker(errorTracker)

	// Initialize database connections
	db, err := initDatabases(cfg, log)
	if err != nil {
		log.Fatalf("failed to initialize databases: %v", err)
	}
	defer db.Close(log)

	// Initialize repositories
	repos, err := initRepositories(db, log)
	if err != nil {
		log.Fatalf("failed to initialize repositories: %v", err)
	}

	// Initialize services
	services := initServices(repos, log)

	// Initialize Kafka
	kafkaProducer := initKafka(cfg, log)

	// Initialize exchange factory
	exchFactory := exchangefactory.NewFactory()

	// Initialize risk engine
	riskEngine := riskengine.NewRiskEngine(
		repos.Risk,
		repos.Position,
		db.Redis,
		log,
	)

	// Initialize tools registry
	toolRegistry := initTools(repos, db.Redis, riskEngine, log)

	// Initialize agents
	agentSystem, err := initAgents(cfg, toolRegistry, log)
	if err != nil {
		log.Fatalf("failed to initialize agents: %v", err)
	}

	// Initialize workers
	workerScheduler := initWorkers(
		repos,
		riskEngine,
		exchFactory,
		kafkaProducer,
		log,
	)

	log.With("tools", len(toolRegistry.List())).Info("System initialized successfully")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers
	if err := workerScheduler.Start(ctx); err != nil {
		log.Fatalf("failed to start workers: %v", err)
	}
	defer func() {
		if err := workerScheduler.Stop(); err != nil {
			log.Warnf("failed to stop workers gracefully: %v", err)
		}
	}()

	// Wait for shutdown signal
	_ = services
	_ = agentSystem
	waitForShutdown(ctx, cancel, errorTracker, log)
}

// loadConfig loads application configuration from environment
func loadConfig() (*config.Config, error) {
	return config.Load()
}

// initLogger initializes structured logging
func initLogger(cfg *config.Config) error {
	return logger.Init(cfg.App.LogLevel, cfg.App.Env)
}

// initErrorTracker initializes error tracking (Sentry or no-op)
func initErrorTracker(cfg *config.Config, log *logger.Logger) errors.Tracker {
	if !cfg.ErrorTracking.Enabled || cfg.ErrorTracking.SentryDSN == "" {
		log.Info("Error tracking disabled")
		return errnoop.New()
	}

	tracker, err := sentry.New(cfg.ErrorTracking.SentryDSN, cfg.ErrorTracking.Environment)
	if err != nil {
		log.Warnf("Failed to initialize Sentry: %v", err)
		return errnoop.New()
	}

	log.Info("Error tracking initialized (Sentry)")
	return tracker
}

// initDatabases initializes database connections (PostgreSQL, ClickHouse, Redis)
func initDatabases(cfg *config.Config, log *logger.Logger) (*Database, error) {
	log.Info("Initializing databases...")

	pgClient, err := pgclient.NewClient(cfg.Postgres)
	if err != nil {
		return nil, errors.Wrap(err, "connect postgres")
	}

	chClient, err := chclient.NewClient(cfg.ClickHouse)
	if err != nil {
		pgClient.Close()
		return nil, errors.Wrap(err, "connect clickhouse")
	}

	redisClient, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		chClient.Close()
		pgClient.Close()
		return nil, errors.Wrap(err, "connect redis")
	}

	log.Info("Databases initialized")
	return &Database{Postgres: pgClient, ClickHouse: chClient, Redis: redisClient}, nil
}

// initRepositories initializes data repositories
func initRepositories(db *Database, log *logger.Logger) (*Repositories, error) {
	if db == nil || db.Postgres == nil || db.ClickHouse == nil {
		return nil, errors.ErrInternal
	}

	log.Info("Initializing repositories...")

	repos := &Repositories{
		User:            pgrepo.NewUserRepository(db.Postgres.DB()),
		ExchangeAccount: pgrepo.NewExchangeAccountRepository(db.Postgres.DB()),
		TradingPair:     pgrepo.NewTradingPairRepository(db.Postgres.DB()),
		Order:           pgrepo.NewOrderRepository(db.Postgres.DB()),
		Position:        pgrepo.NewPositionRepository(db.Postgres.DB()),
		Memory:          pgrepo.NewMemoryRepository(db.Postgres.DB()),
		Journal:         pgrepo.NewJournalRepository(db.Postgres.DB()),
		MarketData:      chrepo.NewMarketDataRepository(db.ClickHouse.Conn()),
		Risk:            pgrepo.NewRiskRepository(db.Postgres.DB()),
	}

	return repos, nil
}

// initServices initializes business logic services
func initServices(repos *Repositories, log *logger.Logger) *Services {
	log.Info("Initializing services...")

	return &Services{
		User:            user.NewService(repos.User),
		ExchangeAccount: exchange_account.NewService(repos.ExchangeAccount),
		TradingPair:     trading_pair.NewService(repos.TradingPair),
		Order:           order.NewService(repos.Order),
		Position:        position.NewService(repos.Position),
		Memory:          memory.NewService(repos.Memory),
		Journal:         journal.NewService(repos.Journal),
	}
}

// initKafka initializes Kafka producer
func initKafka(cfg *config.Config, log *logger.Logger) *kafka.Producer {
	log.Info("Initializing Kafka producer...")

	if len(cfg.Kafka.Brokers) == 0 {
		log.Warn("Kafka brokers not configured, using default localhost:9092")
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	producer := kafka.NewProducer(kafka.ProducerConfig{
		Brokers: cfg.Kafka.Brokers,
		Async:   false,
	})

	log.Info("Kafka producer initialized")
	return producer
}

func initTools(repos *Repositories, redis *redisclient.Client, riskEngine *riskengine.RiskEngine, log *logger.Logger) *tools.Registry {
	log.Info("Registering tools...")
	registry := tools.NewRegistry()
	deps := shared.Deps{
		MarketDataRepo:      repos.MarketData,
		OrderRepo:           repos.Order,
		PositionRepo:        repos.Position,
		ExchangeAccountRepo: repos.ExchangeAccount,
		MemoryRepo:          repos.Memory,
		RiskRepo:            repos.Risk,
		RiskEngine:          riskEngine,
		Redis:               redis,
		Log:                 log,
	}
	tools.RegisterAllTools(registry, deps)
	log.Infof("Registered %d tools", len(registry.List()))
	return registry
}

// initWorkers initializes background workers
func initWorkers(
	repos *Repositories,
	riskEngine *riskengine.RiskEngine,
	exchFactory exchanges.Factory,
	kafkaProducer *kafka.Producer,
	log *logger.Logger,
) *workers.Scheduler {
	log.Info("Initializing workers...")

	scheduler := workers.NewScheduler()

	// Trading workers
	positionMonitor := trading.NewPositionMonitor(
		repos.Position,
		repos.ExchangeAccount,
		exchFactory,
		kafkaProducer,
		true, // enabled
	)
	scheduler.RegisterWorker(positionMonitor)

	orderSync := trading.NewOrderSync(
		repos.Order,
		repos.Position,
		repos.ExchangeAccount,
		exchFactory,
		kafkaProducer,
		true, // enabled
	)
	scheduler.RegisterWorker(orderSync)

	riskMonitor := trading.NewRiskMonitor(
		riskEngine,
		repos.Position,
		kafkaProducer,
		true, // enabled
	)
	scheduler.RegisterWorker(riskMonitor)

	// Market data workers
	ohlcvCollector := marketdata.NewOHLCVCollector(
		repos.MarketData,
		exchFactory,
		[]string{"BTC/USDT", "ETH/USDT", "SOL/USDT"},  // Default symbols
		[]string{"1m", "5m", "15m", "1h", "4h", "1d"}, // Default timeframes
		true, // enabled
	)
	scheduler.RegisterWorker(ohlcvCollector)

	log.Info("Workers initialized", "count", len(scheduler.GetWorkers()))
	return scheduler
}

// initAgents initializes AI agents
func initAgents(cfg *config.Config, toolRegistry *tools.Registry, log *logger.Logger) (*AgentSystem, error) {
	log.Info("Initializing agents...")

	aiRegistry, err := ai.BuildRegistry(cfg.AI)
	if err != nil {
		return nil, errors.Wrap(err, "build AI registry")
	}

	defaultProvider := resolveProvider(aiRegistry, cfg.AI.DefaultProvider)
	if defaultProvider == "" {
		return nil, errors.ErrUnavailable
	}

	factory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "create agent factory")
	}

	selector := ai.NewModelSelector(aiRegistry, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	modelCfg, modelInfo, err := selector.Get(ctx, string(agents.AgentMarketAnalyst), defaultProvider)
	if err != nil {
		return nil, errors.Wrap(err, "resolve default model")
	}

	registry, err := factory.CreateDefaultRegistry(modelCfg.Provider, modelCfg.Model)
	if err != nil {
		return nil, errors.Wrap(err, "create default agent registry")
	}

	log.Infof("Agents initialized with provider=%s model=%s", modelCfg.Provider, modelInfo.Name)
	return &AgentSystem{
		Factory:         factory,
		Registry:        registry,
		ModelSelector:   selector,
		DefaultProvider: modelCfg.Provider,
		DefaultModel:    modelInfo.Name,
	}, nil
}

func resolveProvider(registry *ai.ProviderRegistry, desired string) string {
	desired = ai.NormalizeProviderName(desired)
	if desired != "" {
		if _, err := registry.Get(desired); err == nil {
			return desired
		}
	}

	providers := registry.List()
	if len(providers) == 0 {
		return ""
	}

	return ai.NormalizeProviderName(providers[0].Name())
}

// waitForShutdown waits for shutdown signal and performs graceful shutdown
func waitForShutdown(ctx context.Context, cancel context.CancelFunc, errorTracker errors.Tracker, log *logger.Logger) {
	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down...")

	// Graceful shutdown
	cancel()

	// Flush error tracker
	if errorTracker != nil {
		if err := errorTracker.Flush(ctx); err != nil {
			log.Warnf("Failed to flush error tracker: %v", err)
		}
	}

	log.Info("Shutdown complete")
}
