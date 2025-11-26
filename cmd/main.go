package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
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
	"prometheus/internal/api/health"
	"prometheus/internal/consumers"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/regime"
	domainRisk "prometheus/internal/domain/risk"
	"prometheus/internal/domain/sentiment"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/metrics"
	chrepo "prometheus/internal/repository/clickhouse"
	pgrepo "prometheus/internal/repository/postgres"
	riskengine "prometheus/internal/risk"
	"prometheus/internal/tools"
	"prometheus/internal/tools/shared"
	"prometheus/internal/workers"
	"prometheus/internal/workers/analysis"
	"prometheus/internal/workers/evaluation"
	"prometheus/internal/workers/marketdata"
	sentimentworkers "prometheus/internal/workers/sentiment"
	"prometheus/internal/workers/trading"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// Provider functions for dependency injection
// Each component is initialized independently for better testability and clarity

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
	errorTracker := provideErrorTracker(cfg, log)
	logger.SetErrorTracker(errorTracker)

	// ========================================
	// Infrastructure Layer (Data stores)
	// ========================================
	pgClient, err := providePostgres(cfg, log)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}

	chClient, err := provideClickHouse(cfg, log)
	if err != nil {
		log.Fatalf("failed to connect clickhouse: %v", err)
	}

	redisClient, err := provideRedis(cfg, log)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}

	// ========================================
	// Domain Layer (Repositories)
	// ========================================
	userRepo := pgrepo.NewUserRepository(pgClient.DB())
	exchangeAccountRepo := pgrepo.NewExchangeAccountRepository(pgClient.DB())
	tradingPairRepo := pgrepo.NewTradingPairRepository(pgClient.DB())
	orderRepo := pgrepo.NewOrderRepository(pgClient.DB())
	positionRepo := pgrepo.NewPositionRepository(pgClient.DB())
	memoryRepo := pgrepo.NewMemoryRepository(pgClient.DB())
	journalRepo := pgrepo.NewJournalRepository(pgClient.DB())
	riskRepo := pgrepo.NewRiskRepository(pgClient.DB())
	marketDataRepo := chrepo.NewMarketDataRepository(chClient.Conn())
	regimeRepo := chrepo.NewRegimeRepository(chClient.Conn())
	sentimentRepo := chrepo.NewSentimentRepository(chClient.Conn())

	log.Info("Repositories initialized")

	// ========================================
	// Domain Layer (Services)
	// ========================================
	userService := user.NewService(userRepo)
	exchangeAccountService := exchange_account.NewService(exchangeAccountRepo)
	tradingPairService := trading_pair.NewService(tradingPairRepo)
	orderService := order.NewService(orderRepo)
	positionService := position.NewService(positionRepo)
	memoryService := memory.NewService(memoryRepo)
	journalService := journal.NewService(journalRepo)

	log.Info("Services initialized")

	// ========================================
	// External Adapters (Kafka, Exchange, Crypto)
	// ========================================
	kafkaProducer := provideKafkaProducer(cfg, log)

	// Create multiple consumers for different consumer groups
	notificationConsumer := provideKafkaConsumer(cfg, "notifications", log)
	riskConsumer := provideKafkaConsumer(cfg, "risk_events", log)
	analyticsConsumer := provideKafkaConsumer(cfg, "analytics", log)
	opportunityConsumer := provideKafkaConsumer(cfg, events.TopicOpportunityFound, log)

	encryptor, err := crypto.NewEncryptor(cfg.Crypto.EncryptionKey)
	if err != nil {
		log.Fatalf("failed to initialize encryptor: %v", err)
	}

	exchFactory := exchangefactory.NewFactory()
	marketDataFactory := exchangefactory.NewMarketDataFactory(cfg.MarketData)
	log.Info("Exchange factory initialized")

	// ========================================
	// Business Logic (Risk, Tools, Agents)
	// ========================================
	riskEngine := riskengine.NewRiskEngine(riskRepo, positionRepo, redisClient, log)

	toolRegistry := provideToolRegistry(
		marketDataRepo,
		orderRepo,
		positionRepo,
		exchangeAccountRepo,
		memoryRepo,
		riskRepo,
		riskEngine,
		redisClient,
		log,
	)

	agentFactory, agentRegistry, err := provideAgents(cfg, toolRegistry, log)
	if err != nil {
		log.Fatalf("failed to initialize agents: %v", err)
	}

	// ========================================
	// Application Lifecycle Context
	// ========================================
	// Create application context - will be cancelled on shutdown signal
	ctx, cancel := context.WithCancel(context.Background())

	// WaitGroup for tracking all goroutines
	var wg sync.WaitGroup

	// ========================================
	// Background Workers
	// ========================================
	workerScheduler := provideWorkers(
		userRepo,
		tradingPairRepo,
		orderRepo,
		positionRepo,
		journalRepo,
		marketDataRepo,
		regimeRepo,
		sentimentRepo,
		exchangeAccountRepo,
		riskEngine,
		exchFactory,
		marketDataFactory,
		encryptor,
		kafkaProducer,
		agentFactory,
		cfg,
		log,
	)

	// ========================================
	// Event Consumers (Background Processing)
	// ========================================
	notificationSvc := consumers.NewNotificationConsumer(notificationConsumer, log)
	riskSvc := consumers.NewRiskConsumer(riskConsumer, riskEngine, log)
	analyticsSvc := consumers.NewAnalyticsConsumer(analyticsConsumer, log)
	opportunitySvc := consumers.NewOpportunityConsumer(
		opportunityConsumer,
		userRepo,
		tradingPairRepo,
		agentFactory,
		cfg.Workers.MarketScannerMaxConcurrency,
		log,
	)

	// Start consumers in background (all tracked by WaitGroup)
	wg.Add(4)
	go func() {
		defer wg.Done()
		if err := notificationSvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Notification consumer failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := riskSvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Risk consumer failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := analyticsSvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Analytics consumer failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := opportunitySvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Opportunity consumer failed", "error", err)
		}
	}()

	log.Info("✓ Event consumers started (notification, risk, analytics, opportunity)")

	log.With("tools", len(toolRegistry.List())).Info("System initialized successfully")

	// ========================================
	// Observability (Metrics, Health Checks)
	// ========================================
	metrics.Init()
	customCollector := metrics.NewCustomCollector(log, pgClient.DB(), chClient.Conn(), redisClient.Client())
	metrics.RegisterCustomCollector(customCollector)
	log.Info("Metrics initialized")

	healthHandler := health.New(
		log,
		pgClient.DB(),
		chClient.Conn(),
		redisClient.Client(),
		cfg.App.Name,
		cfg.App.Version,
	)

	// ========================================
	// HTTP Server (Health + Metrics API)
	// ========================================
	httpServer := provideHTTPServer(cfg, healthHandler, log)

	// Start HTTP server in background (tracked by wg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
		log.Infof("Starting HTTP server on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("HTTP server failed: %v", err)
			cancel() // Trigger shutdown on fatal HTTP error
		}
	}()

	// Start workers (they use ctx for cancellation)
	if err := workerScheduler.Start(ctx); err != nil {
		log.Fatalf("failed to start workers: %v", err)
	}

	// Suppress unused warnings (will be used in future phases)
	_ = userService
	_ = exchangeAccountService
	_ = tradingPairService
	_ = orderService
	_ = positionService
	_ = memoryService
	_ = journalService
	_ = agentRegistry

	log.Info("All systems operational")

	// ========================================
	// Shutdown Signal Handling
	// ========================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a shutdown signal
	sig := <-quit
	log.Infof("Received signal: %v, initiating shutdown...", sig)

	// Cancel application context IMMEDIATELY
	// This signals all components to stop (workers, scanners, etc.)
	cancel()

	// Now perform coordinated cleanup with timeout
	gracefulShutdown(
		&wg,
		httpServer,
		workerScheduler,
		marketDataFactory,
		kafkaProducer,
		notificationConsumer,
		riskConsumer,
		analyticsConsumer,
		opportunityConsumer,
		pgClient,
		chClient,
		redisClient,
		errorTracker,
		log,
	)
}

// ========================================
// Configuration & Logging
// ========================================

func loadConfig() (*config.Config, error) {
	return config.Load()
}

func initLogger(cfg *config.Config) error {
	return logger.Init(cfg.App.LogLevel, cfg.App.Env)
}

// ========================================
// Provider Functions (Dependency Injection)
// ========================================

func provideErrorTracker(cfg *config.Config, log *logger.Logger) errors.Tracker {
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

func providePostgres(cfg *config.Config, log *logger.Logger) (*pgclient.Client, error) {
	log.Info("Connecting to PostgreSQL...")
	client, err := pgclient.NewClient(cfg.Postgres)
	if err != nil {
		return nil, errors.Wrap(err, "connect postgres")
	}
	log.Info("✓ PostgreSQL connected")
	return client, nil
}

func provideClickHouse(cfg *config.Config, log *logger.Logger) (*chclient.Client, error) {
	log.Info("Connecting to ClickHouse...")
	client, err := chclient.NewClient(cfg.ClickHouse)
	if err != nil {
		return nil, errors.Wrap(err, "connect clickhouse")
	}
	log.Info("✓ ClickHouse connected")
	return client, nil
}

func provideRedis(cfg *config.Config, log *logger.Logger) (*redisclient.Client, error) {
	log.Info("Connecting to Redis...")
	client, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		return nil, errors.Wrap(err, "connect redis")
	}
	log.Info("✓ Redis connected")
	return client, nil
}

func provideKafkaProducer(cfg *config.Config, log *logger.Logger) *kafka.Producer {
	log.Info("Initializing Kafka producer...")
	if len(cfg.Kafka.Brokers) == 0 {
		log.Warn("Kafka brokers not configured, using default localhost:9092")
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	producer := kafka.NewProducer(kafka.ProducerConfig{
		Brokers: cfg.Kafka.Brokers,
		Async:   false,
	})
	log.Info("✓ Kafka producer initialized")
	return producer
}

func provideKafkaConsumer(cfg *config.Config, topic string, log *logger.Logger) *kafka.Consumer {
	log.Info("Initializing Kafka consumer", "topic", topic)
	if len(cfg.Kafka.Brokers) == 0 {
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Kafka.Brokers,
		GroupID: cfg.Kafka.GroupID,
		Topic:   topic,
	})
	log.Info("✓ Kafka consumer initialized", "topic", topic)
	return consumer
}

func provideToolRegistry(
	marketDataRepo market_data.Repository,
	orderRepo order.Repository,
	positionRepo position.Repository,
	exchangeAccountRepo exchange_account.Repository,
	memoryRepo memory.Repository,
	riskRepo domainRisk.Repository,
	riskEngine *riskengine.RiskEngine,
	redisClient *redisclient.Client,
	log *logger.Logger,
) *tools.Registry {
	log.Info("Registering tools...")
	registry := tools.NewRegistry()

	deps := shared.Deps{
		MarketDataRepo:      marketDataRepo,
		OrderRepo:           orderRepo,
		PositionRepo:        positionRepo,
		ExchangeAccountRepo: exchangeAccountRepo,
		MemoryRepo:          memoryRepo,
		RiskRepo:            riskRepo,
		RiskEngine:          riskEngine,
		Redis:               redisClient,
		Log:                 log,
	}

	tools.RegisterAllTools(registry, deps)
	log.Infof("✓ Registered %d tools", len(registry.List()))
	return registry
}

func provideAgents(cfg *config.Config, toolRegistry *tools.Registry, log *logger.Logger) (*agents.Factory, *agents.Registry, error) {
	log.Info("Initializing agents...")

	aiRegistry, err := ai.BuildRegistry(cfg.AI)
	if err != nil {
		return nil, nil, errors.Wrap(err, "build AI registry")
	}

	defaultProvider := resolveProvider(aiRegistry, cfg.AI.DefaultProvider)
	if defaultProvider == "" {
		return nil, nil, errors.ErrUnavailable
	}

	factory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "create agent factory")
	}

	selector := ai.NewModelSelector(aiRegistry, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	modelCfg, modelInfo, err := selector.Get(ctx, string(agents.AgentMarketAnalyst), defaultProvider)
	if err != nil {
		return nil, nil, errors.Wrap(err, "resolve default model")
	}

	registry, err := factory.CreateDefaultRegistry(modelCfg.Provider, modelCfg.Model)
	if err != nil {
		return nil, nil, errors.Wrap(err, "create default agent registry")
	}

	log.Infof("✓ Agents initialized with provider=%s model=%s", modelCfg.Provider, modelInfo.Name)
	return factory, registry, nil
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

func provideWorkers(
	userRepo user.Repository,
	tradingPairRepo trading_pair.Repository,
	orderRepo order.Repository,
	positionRepo position.Repository,
	journalRepo journal.Repository,
	marketDataRepo market_data.Repository,
	regimeRepo regime.Repository,
	sentimentRepo sentiment.Repository,
	exchangeAccountRepo exchange_account.Repository,
	riskEngine *riskengine.RiskEngine,
	exchFactory exchanges.Factory,
	marketDataFactory exchanges.CentralFactory,
	encryptor *crypto.Encryptor,
	kafkaProducer *kafka.Producer,
	agentFactory *agents.Factory,
	cfg *config.Config,
	log *logger.Logger,
) *workers.Scheduler {
	log.Info("Initializing workers...")

	scheduler := workers.NewScheduler()

	// Default monitored symbols
	defaultSymbols := []string{"BTC/USDT", "ETH/USDT", "SOL/USDT", "BNB/USDT", "XRP/USDT"}

	// ========================================
	// Trading Workers (high frequency)
	// ========================================

	// Position monitor: Updates PnL and checks SL/TP
	scheduler.RegisterWorker(trading.NewPositionMonitor(
		userRepo,
		positionRepo,
		exchangeAccountRepo,
		exchFactory,
		*encryptor,
		kafkaProducer,
		cfg.Workers.PositionMonitorInterval,
		true, // enabled
	))

	// Order sync: Syncs order status with exchanges
	scheduler.RegisterWorker(trading.NewOrderSync(
		userRepo,
		orderRepo,
		positionRepo,
		exchangeAccountRepo,
		exchFactory,
		*encryptor,
		kafkaProducer,
		cfg.Workers.OrderSyncInterval,
		true, // enabled
	))

	// Risk monitor: Checks circuit breakers and risk limits
	scheduler.RegisterWorker(trading.NewRiskMonitor(
		userRepo,
		riskEngine,
		positionRepo,
		kafkaProducer,
		cfg.Workers.RiskMonitorInterval,
		true, // enabled
	))

	// PnL calculator: Calculates daily PnL and updates risk state
	scheduler.RegisterWorker(trading.NewPnLCalculator(
		userRepo,
		positionRepo,
		riskEngine,
		kafkaProducer,
		cfg.Workers.PnLCalculatorInterval,
		true, // enabled
	))

	// ========================================
	// Market Data Workers (medium frequency)
	// ========================================

	// OHLCV collector: Collects candles
	scheduler.RegisterWorker(marketdata.NewOHLCVCollector(
		marketDataRepo,
		exchFactory,
		defaultSymbols,
		[]string{"1m", "5m", "15m", "1h", "4h", "1d"}, // Timeframes
		cfg.Workers.OHLCVCollectorInterval,
		true, // enabled
	))

	// Ticker collector: Collects real-time prices (5-10 sec intervals)
	scheduler.RegisterWorker(marketdata.NewTickerCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		[]string{"binance", "bybit", "okx"},
		cfg.Workers.TickerCollectorInterval,
		true, // enabled
	))

	// OrderBook collector: Collects order book snapshots every 10s
	scheduler.RegisterWorker(marketdata.NewOrderBookCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		[]string{"binance", "bybit", "okx"},
		20, // depth
		cfg.Workers.OrderBookCollectorInterval,
		true, // enabled
	))

	// Trades collector: Collects real-time trades (tape)
	scheduler.RegisterWorker(marketdata.NewTradesCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		[]string{"binance", "bybit", "okx"},
		100, // limit per request
		cfg.Workers.TradesCollectorInterval,
		true, // enabled
	))

	// Funding collector: Collects funding rates from futures
	scheduler.RegisterWorker(marketdata.NewFundingCollector(
		marketDataRepo,
		marketDataFactory,
		defaultSymbols,
		[]string{"binance", "bybit", "okx"},
		cfg.Workers.FundingCollectorInterval,
		true, // enabled
	))

	// ========================================
	// Sentiment Workers (low frequency)
	// ========================================

	// News collector: Collects crypto news from external sources
	// Default currencies: BTC, ETH, SOL for news filtering
	defaultCurrencies := []string{"BTC", "ETH", "SOL"}
	scheduler.RegisterWorker(sentimentworkers.NewNewsCollector(
		sentimentRepo,
		cfg.MarketData.NewsAPIKey, // CryptoPanic API key (optional)
		defaultCurrencies,
		cfg.Workers.NewsCollectorInterval,
		true, // enabled
	))

	log.Info("Worker intervals configured",
		"position_monitor", cfg.Workers.PositionMonitorInterval,
		"order_sync", cfg.Workers.OrderSyncInterval,
		"risk_monitor", cfg.Workers.RiskMonitorInterval,
		"pnl_calculator", cfg.Workers.PnLCalculatorInterval,
		"market_scanner", cfg.Workers.MarketScannerInterval,
		"opportunity_finder", cfg.Workers.OpportunityFinderInterval,
	)

	// ========================================
	// Analysis Workers (core agentic system)
	// ========================================

	// Market scanner: Runs SCHEDULED agent analysis for ALL active users
	// For event-driven analysis, see OpportunityConsumer in consumers
	scheduler.RegisterWorker(analysis.NewMarketScanner(
		userRepo,
		tradingPairRepo,
		agentFactory,
		kafkaProducer,
		cfg.Workers.MarketScannerInterval,
		cfg.Workers.MarketScannerMaxConcurrency,
		true, // enabled
	))

	// Opportunity finder: Finds trading setups
	scheduler.RegisterWorker(analysis.NewOpportunityFinder(
		marketDataRepo,
		tradingPairRepo,
		exchFactory,
		kafkaProducer,
		defaultSymbols,
		cfg.Workers.OpportunityFinderInterval,
		true, // enabled
	))

	// Regime detector: Detects market regime
	scheduler.RegisterWorker(analysis.NewRegimeDetector(
		marketDataRepo,
		regimeRepo,
		kafkaProducer,
		defaultSymbols,
		cfg.Workers.RegimeDetectorInterval,
		true, // enabled
	))

	// SMC scanner: Scans for Smart Money Concepts patterns
	scheduler.RegisterWorker(analysis.NewSMCScanner(
		marketDataRepo,
		kafkaProducer,
		defaultSymbols,
		cfg.Workers.SMCScannerInterval,
		true, // enabled
	))

	// ========================================
	// Evaluation Workers (low frequency)
	// ========================================

	// Strategy evaluator: Evaluates and disables underperforming strategies
	scheduler.RegisterWorker(evaluation.NewStrategyEvaluator(
		userRepo,
		tradingPairRepo,
		journalRepo,
		kafkaProducer,
		cfg.Workers.StrategyEvaluatorInterval,
		true, // enabled
	))

	// Journal compiler: Creates journal entries from closed positions
	scheduler.RegisterWorker(evaluation.NewJournalCompiler(
		userRepo,
		positionRepo,
		journalRepo,
		kafkaProducer,
		cfg.Workers.JournalCompilerInterval,
		true, // enabled
	))

	// Daily report: Generates daily performance reports
	scheduler.RegisterWorker(evaluation.NewDailyReport(
		userRepo,
		positionRepo,
		journalRepo,
		kafkaProducer,
		cfg.Workers.DailyReportInterval,
		true, // enabled
	))

	log.Infof("✓ Workers initialized: %d registered", len(scheduler.GetWorkers()))
	return scheduler
}

func provideHTTPServer(cfg *config.Config, healthHandler *health.Handler, log *logger.Logger) *http.Server {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", healthHandler.HandleHealth)
	mux.HandleFunc("/ready", healthHandler.HandleReadiness)
	mux.HandleFunc("/live", healthHandler.HandleLiveness)

	// Prometheus metrics endpoint
	mux.Handle("/metrics", metrics.Handler())

	// Root endpoint (service info)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"%s","version":"%s","status":"running"}`, cfg.App.Name, cfg.App.Version)
	})

	port := 8080
	if cfg.HTTP.Port > 0 {
		port = cfg.HTTP.Port
	}

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// gracefulShutdown handles coordinated cleanup of all components
// Note: Application context is already cancelled by caller before this function
func gracefulShutdown(
	wg *sync.WaitGroup,
	httpServer *http.Server,
	workerScheduler *workers.Scheduler,
	marketDataFactory exchanges.CentralFactory,
	kafkaProducer *kafka.Producer,
	notificationConsumer *kafka.Consumer,
	riskConsumer *kafka.Consumer,
	analyticsConsumer *kafka.Consumer,
	opportunityConsumer *kafka.Consumer,
	pgClient *pgclient.Client,
	chClient *chclient.Client,
	redisClient *redisclient.Client,
	errorTracker errors.Tracker,
	log *logger.Logger,
) {
	// Create shutdown context with timeout (30 seconds for complete cleanup)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cleanup sequence (order matters!)
	// Note: Application context is already cancelled
	// Workers and goroutines are finishing their current work
	// We wait for graceful completion with timeouts

	// Step 1: Stop accepting new HTTP requests (5s timeout)
	log.Info("[1/7] Stopping HTTP server...")
	httpCtx, httpCancel := context.WithTimeout(shutdownCtx, 5*time.Second)
	defer httpCancel()
	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Error("HTTP server shutdown failed", "error", err)
	} else {
		log.Info("✓ HTTP server stopped")
	}

	// Step 2: Stop background workers (they check ctx.Done() in their loops)
	// Scheduler.Stop() calls wg.Wait() internally with 30s timeout
	log.Info("[2/7] Stopping background workers...")
	if err := workerScheduler.Stop(); err != nil {
		log.Error("Workers shutdown failed", "error", err)
	} else {
		log.Info("✓ Workers stopped")
	}

	// Step 3: Wait for auxiliary goroutines (market scanner listener, etc.)
	log.Info("[3/7] Waiting for auxiliary goroutines...")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("✓ All goroutines finished")
	case <-time.After(5 * time.Second):
		log.Warn("⚠ Some goroutines did not finish within 5s timeout")
	}

	// Step 4: Close Kafka clients
	log.Info("[4/7] Closing Kafka clients...")
	if kafkaProducer != nil {
		if err := kafkaProducer.Close(); err != nil {
			log.Error("Kafka producer close failed", "error", err)
		}
	}
	// Close all consumers
	for name, consumer := range map[string]*kafka.Consumer{
		"notification": notificationConsumer,
		"risk":         riskConsumer,
		"analytics":    analyticsConsumer,
		"opportunity":  opportunityConsumer,
	} {
		if consumer != nil {
			if err := consumer.Close(); err != nil {
				log.Error("Kafka consumer close failed", "consumer", name, "error", err)
			}
		}
	}
	log.Info("✓ Kafka clients closed")

	// Step 5: Flush error tracker
	log.Info("[5/7] Flushing error tracker...")
	if errorTracker != nil {
		flushCtx, flushCancel := context.WithTimeout(shutdownCtx, 3*time.Second)
		defer flushCancel()
		if err := errorTracker.Flush(flushCtx); err != nil {
			log.Error("Error tracker flush failed", "error", err)
		} else {
			log.Info("✓ Error tracker flushed")
		}
	}

	// Step 6: Sync logs
	log.Info("[6/7] Syncing logs...")
	if err := logger.Sync(); err != nil {
		// Ignore "sync /dev/stderr: inappropriate ioctl for device" on Linux
		log.Warn("Log sync completed with warnings")
	} else {
		log.Info("✓ Logs synced")
	}

	// Step 7: Close database connections (LAST - other components may need them)
	log.Info("[7/7] Closing database connections...")
	var dbErrors []error
	if pgClient != nil {
		if err := pgClient.Close(); err != nil {
			dbErrors = append(dbErrors, errors.Wrap(err, "postgres"))
		}
	}
	if chClient != nil {
		if err := chClient.Close(); err != nil {
			dbErrors = append(dbErrors, errors.Wrap(err, "clickhouse"))
		}
	}
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			dbErrors = append(dbErrors, errors.Wrap(err, "redis"))
		}
	}
	if len(dbErrors) > 0 {
		log.Error("Database close errors", "errors", dbErrors)
	} else {
		log.Info("✓ Database connections closed")
	}

	log.Info("✅ Graceful shutdown complete")
}
