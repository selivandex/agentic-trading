package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/errors/noop"
	"prometheus/internal/adapters/errors/sentry"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

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

	// TODO: Initialize database connections
	// db := initDatabases(cfg, log)
	// defer db.Close()

	// TODO: Initialize repositories
	// repos := initRepositories(db, log)

	// TODO: Initialize services
	// services := initServices(repos, log)

	// TODO: Initialize agents
	// agents := initAgents(cfg, services, log)

	// TODO: Initialize workers
	// workers := initWorkers(cfg, services, agents, log)

	// TODO: Initialize Telegram bot
	// bot := initTelegramBot(cfg, services, log)

	log.Info("System initialized successfully")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start all components
	// startWorkers(ctx, workers, log)
	// startBot(ctx, bot, log)

	// Wait for shutdown signal
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
		return noop.New()
	}

	tracker, err := sentry.New(cfg.ErrorTracking.SentryDSN, cfg.ErrorTracking.Environment)
	if err != nil {
		log.Warnf("Failed to initialize Sentry: %v", err)
		return noop.New()
	}

	log.Info("Error tracking initialized (Sentry)")
	return tracker
}

// initDatabases initializes database connections (PostgreSQL, ClickHouse, Redis)
// func initDatabases(cfg *config.Config, log *logger.Logger) *Database {
// 	log.Info("Initializing databases...")
//
// 	pgClient, err := postgres.NewClient(cfg.Postgres)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
// 	}
//
// 	chClient, err := clickhouse.NewClient(cfg.ClickHouse)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to ClickHouse: %v", err)
// 	}
//
// 	redisClient, err := redis.NewClient(cfg.Redis)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to Redis: %v", err)
// 	}
//
// 	log.Info("Databases initialized")
// 	return &Database{
// 		Postgres:   pgClient,
// 		ClickHouse: chClient,
// 		Redis:      redisClient,
// 	}
// }

// initRepositories initializes data repositories
// func initRepositories(db *Database, log *logger.Logger) *Repositories {
// 	log.Info("Initializing repositories...")
//
// 	return &Repositories{
// 		User:            repository.NewUserRepository(db.Postgres.DB()),
// 		ExchangeAccount: repository.NewExchangeAccountRepository(db.Postgres.DB()),
// 		TradingPair:     repository.NewTradingPairRepository(db.Postgres.DB()),
// 		Order:           repository.NewOrderRepository(db.Postgres.DB()),
// 		Position:        repository.NewPositionRepository(db.Postgres.DB()),
// 		Memory:          repository.NewMemoryRepository(db.Postgres.DB()),
// 		MarketData:      repository.NewMarketDataRepository(db.ClickHouse),
// 	}
// }

// initServices initializes business logic services
// func initServices(repos *Repositories, log *logger.Logger) *Services {
// 	log.Info("Initializing services...")
//
// 	return &Services{
// 		User: user.NewService(repos.User),
// 		// ... other services
// 	}
// }

// initAgents initializes AI agents
// func initAgents(cfg *config.Config, services *Services, log *logger.Logger) *Agents {
// 	log.Info("Initializing agents...")
//
// 	aiRegistry := ai.SetupProviders(cfg.AI)
// 	toolRegistry := tools.NewRegistry()
// 	agentFactory := agents.NewFactory(agents.FactoryDeps{
// 		AIRegistry:   aiRegistry,
// 		ToolRegistry: toolRegistry,
// 		Templates:    templates.Get(),
// 	})
//
// 	log.Info("Agents initialized")
// 	return &Agents{
// 		Factory: agentFactory,
// 	}
// }

// initWorkers initializes background workers
// func initWorkers(cfg *config.Config, services *Services, agents *Agents, log *logger.Logger) *Workers {
// 	log.Info("Initializing workers...")
//
// 	workerList := []workers.Worker{
// 		// Market data workers
// 		workers.NewOHLCVCollector(...),
// 		workers.NewTickerCollector(...),
// 		// ... other workers
// 	}
//
// 	scheduler := workers.NewScheduler(workerList)
//
// 	log.Info("Workers initialized")
// 	return &Workers{
// 		Scheduler: scheduler,
// 	}
// }

// initTelegramBot initializes Telegram bot
// func initTelegramBot(cfg *config.Config, services *Services, log *logger.Logger) *telegram.Bot {
// 	log.Info("Initializing Telegram bot...")
//
// 	bot, err := telegram.NewBot(telegram.BotDeps{
// 		Token:       cfg.Telegram.BotToken,
// 		UserService: services.User,
// 		Handlers:    handlers.NewRegistry(),
// 	})
// 	if err != nil {
// 		log.Fatalf("Failed to create Telegram bot: %v", err)
// 	}
//
// 	log.Info("Telegram bot initialized")
// 	return bot
// }

// startWorkers starts all background workers
// func startWorkers(ctx context.Context, workers *Workers, log *logger.Logger) {
// 	log.Info("Starting workers...")
// 	workers.Scheduler.Start(ctx)
// 	log.Info("Workers started")
// }

// startBot starts Telegram bot
// func startBot(ctx context.Context, bot *telegram.Bot, log *logger.Logger) {
// 	log.Info("Starting Telegram bot...")
// 	go func() {
// 		if err := bot.Start(ctx); err != nil {
// 			log.Errorf("Telegram bot error: %v", err)
// 		}
// 	}()
// 	log.Info("Telegram bot started")
// }

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
