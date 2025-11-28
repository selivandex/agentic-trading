package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/adk/session"

	"prometheus/internal/adapters/adk"
	"prometheus/internal/adapters/ai"
	chclient "prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/embeddings"
	errnoop "prometheus/internal/adapters/errors/noop"
	"prometheus/internal/adapters/errors/sentry"
	"prometheus/internal/adapters/exchangefactory"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	pgclient "prometheus/internal/adapters/postgres"
	redisclient "prometheus/internal/adapters/redis"
	telegram "prometheus/internal/adapters/telegram"
	"prometheus/internal/agents"
	"prometheus/internal/agents/workflows"
	"prometheus/internal/api"
	"prometheus/internal/api/health"
	telegramapi "prometheus/internal/api/telegram"
	"prometheus/internal/consumers"
	"prometheus/internal/domain/derivatives"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/macro"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/onchain"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	"prometheus/internal/domain/regime"
	domainRisk "prometheus/internal/domain/risk"
	"prometheus/internal/domain/sentiment"
	domainsession "prometheus/internal/domain/session"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/metrics"
	regimeml "prometheus/internal/ml/regime"
	chrepo "prometheus/internal/repository/clickhouse"
	pgrepo "prometheus/internal/repository/postgres"
	analysisservice "prometheus/internal/services/analysis"
	onboardingservice "prometheus/internal/services/onboarding"
	positionservice "prometheus/internal/services/position"
	riskservice "prometheus/internal/services/risk"
	"prometheus/internal/tools"
	"prometheus/internal/tools/shared"
	"prometheus/internal/workers"
	"prometheus/internal/workers/analysis"
	derivworkers "prometheus/internal/workers/derivatives"
	"prometheus/internal/workers/evaluation"
	macroworkers "prometheus/internal/workers/macro"
	"prometheus/internal/workers/marketdata"
	onchainworkers "prometheus/internal/workers/onchain"
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
	defer func() {
		_ = logger.Sync() // Ignore error on shutdown
	}()

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
	sessionRepo := pgrepo.NewSessionRepository(pgClient.DB())
	reasoningRepo := pgrepo.NewReasoningRepository(pgClient.DB())
	marketDataRepo := chrepo.NewMarketDataRepository(chClient.Conn())
	regimeRepo := chrepo.NewRegimeRepository(chClient.Conn())
	sentimentRepo := chrepo.NewSentimentRepository(chClient.Conn())
	onchainRepo := chrepo.NewOnChainRepository(chClient.Conn())
	derivRepo := chrepo.NewDerivativesRepository(chClient.Conn())
	macroRepo := chrepo.NewMacroRepository(chClient.Conn())
	aiUsageRepo := chrepo.NewAIUsageRepository(chClient.Conn())

	log.Info("Repositories initialized")

	// ========================================
	// Domain Layer (Services)
	// ========================================
	// ========================================
	// External Adapters (Kafka, Exchange, Crypto, Embeddings)
	// ========================================
	kafkaProducer := provideKafkaProducer(cfg, log)

	// Create multiple consumers for different consumer groups
	notificationConsumer := provideKafkaConsumer(cfg, "notifications", log)
	riskConsumer := provideKafkaConsumer(cfg, "risk_events", log)
	analyticsConsumer := provideKafkaConsumer(cfg, "analytics", log)
	opportunityKafkaConsumer := provideKafkaConsumer(cfg, events.TopicOpportunityFound, log)
	aiUsageKafkaConsumer := provideKafkaConsumer(cfg, events.TopicAIUsage, log)
	positionGuardianConsumer := provideKafkaConsumer(cfg, "position_guardian", log)
	telegramNotificationConsumer := provideKafkaConsumer(cfg, "telegram_notifications", log)

	encryptor, err := crypto.NewEncryptor(cfg.Crypto.EncryptionKey)
	if err != nil {
		log.Fatalf("failed to initialize encryptor: %v", err)
	}

	exchFactory := exchangefactory.NewFactory()
	marketDataFactory := exchangefactory.NewMarketDataFactory(cfg.MarketData)
	log.Info("Exchange factory initialized")

	// Create embedding provider for semantic memory search (needed for memory service)
	embeddingProvider, err := embeddings.NewProvider(embeddings.Config{
		Provider: embeddings.ProviderOpenAI,
		APIKey:   cfg.AI.OpenAIKey,
		Model:    "text-embedding-3-small", // 1536 dimensions, $0.02/1M tokens
		Timeout:  30 * time.Second,
	})
	if err != nil {
		log.Fatalf("failed to create embedding provider: %v", err)
	}
	log.Infof("Embedding provider initialized: %s (%d dimensions)", embeddingProvider.Name(), embeddingProvider.Dimensions())

	// Create domain services
	userService := user.NewService(userRepo)
	exchangeAccountService := exchange_account.NewService(exchangeAccountRepo)
	tradingPairService := trading_pair.NewService(tradingPairRepo)
	orderService := order.NewService(orderRepo)
	positionService := position.NewService(positionRepo)
	memoryService := memory.NewService(memoryRepo, embeddingProvider)
	journalService := journal.NewService(journalRepo)
	sessionDomainService := domainsession.NewService(sessionRepo)

	log.Info("Services initialized")

	// Create ADK session service adapter
	adkSessionService := adk.NewSessionService(sessionDomainService)

	// ========================================
	// Business Logic (Risk, Tools, Agents)
	// ========================================
	riskEngine := riskservice.NewRiskEngine(riskRepo, positionRepo, userRepo, redisClient, log)

	toolRegistry := provideToolRegistry(
		marketDataRepo,
		orderRepo,
		positionRepo,
		exchangeAccountRepo,
		memoryRepo,
		riskRepo,
		userRepo,
		riskEngine,
		redisClient,
		embeddingProvider,
		kafkaProducer,
		log,
	)

	agentFactory, agentRegistry, defaultProvider, defaultModel, err := provideAgents(cfg, toolRegistry, adkSessionService, redisClient, riskEngine, kafkaProducer, aiUsageRepo, reasoningRepo, log)
	if err != nil {
		log.Fatalf("failed to initialize agents: %v", err)
	}

	// Register expert agent tools (agent-as-tool pattern)
	if err := registerExpertTools(agentFactory, toolRegistry, defaultProvider, defaultModel, log); err != nil {
		log.Warnf("Failed to register expert tools: %v", err)
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
		onchainRepo,
		derivRepo,
		macroRepo,
		exchangeAccountRepo,
		riskEngine,
		exchFactory,
		marketDataFactory,
		encryptor,
		kafkaProducer,
		agentFactory,
		agentRegistry, // Add agent registry
		adkSessionService,
		defaultProvider,
		defaultModel,
		cfg,
		log,
	)

	// ========================================
	// Event Consumers (Background Processing)
	// ========================================
	notificationSvc := consumers.NewNotificationConsumer(notificationConsumer, log)
	riskSvc := consumers.NewRiskConsumer(riskConsumer, riskEngine, log)
	analyticsSvc := consumers.NewAnalyticsConsumer(analyticsConsumer, log)

	// AI usage consumer: Reads AI usage events from Kafka and writes to ClickHouse in batches
	aiUsageSvc := consumers.NewAIUsageConsumer(aiUsageKafkaConsumer, aiUsageRepo, log)

	// Create workflow factory for personal trading workflows
	workflowFactory := workflows.NewFactory(agentFactory, defaultProvider, defaultModel)

	opportunitySvc := consumers.NewOpportunityConsumer(
		opportunityKafkaConsumer,
		userRepo,
		tradingPairRepo,
		workflowFactory,
		adkSessionService,
		cfg.Workers.MarketScannerMaxConcurrency,
		log,
	)

	// ========================================
	// Position Guardian (Event-Driven Monitoring) - Phase 2
	// ========================================

	// Critical event handler (algorithmic, <1s response)
	criticalHandler := positionservice.NewCriticalEventHandler(
		positionRepo,
		log,
	)

	// Agent event handler (LLM-based decisions)
	// Use PositionManager agent for position monitoring decisions
	var agentHandler *positionservice.AgentEventHandler

	// Create PositionManager agent using resolved provider and model
	positionManagerAgent, agentErr := agentFactory.CreateAgentForUser(
		agents.AgentPositionManager,
		defaultProvider,
		defaultModel,
	)
	if agentErr != nil {
		log.Warnf("PositionManager agent creation failed, agent handler disabled: %v", agentErr)
	} else {
		agentHandler = positionservice.NewAgentEventHandler(
			positionRepo,
			positionManagerAgent,
			adkSessionService,
			log,
			30*time.Second, // timeout
		)
		log.Info("✓ Agent event handler initialized with PositionManager agent")
	}

	// Position guardian consumer
	positionGuardianSvc := consumers.NewPositionGuardianConsumer(
		positionGuardianConsumer,
		criticalHandler,
		agentHandler,
		log,
	)

	// ========================================
	// Telegram Bot & Notifications
	// ========================================
	telegramBot, telegramHandlers, telegramNotificationSvc := provideTelegramBot(
		cfg,
		userRepo,
		positionRepo,
		exchangeAccountRepo,
		workflowFactory,
		adkSessionService,
		redisClient,
		encryptor,
		exchFactory,
		telegramNotificationConsumer,
		log,
	)

	// Start consumers in background (all tracked by WaitGroup)
	wg.Add(8) // Increased from 6 to 8 (added Telegram bot + notification consumer)
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
	go func() {
		defer wg.Done()
		if err := aiUsageSvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("AI usage consumer failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := positionGuardianSvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Position guardian consumer failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := telegramBot.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Telegram bot failed", "error", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := telegramNotificationSvc.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("Telegram notification consumer failed", "error", err)
		}
	}()

	log.Info("✓ Event consumers started: notification, risk, analytics, opportunity, ai_usage, position_guardian, telegram_bot, telegram_notifications")

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
	// HTTP Server (Health + Metrics API + Telegram Webhook)
	// ========================================
	httpServer := provideHTTPServer(cfg, healthHandler, telegramBot, telegramHandlers, log)

	// Start HTTP server in background (tracked by wg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(); err != nil {
			log.Errorf("HTTP server failed: %v", err)
			cancel() // Trigger shutdown on fatal HTTP error
		}
	}()

	// Start workers (they use ctx for cancellation)
	// if err := workerScheduler.Start(ctx); err != nil {
	// 	log.Fatalf("failed to start workers: %v", err)
	// }

	// Suppress unused warnings (will be used in future phases)
	_ = userService
	_ = exchangeAccountService
	_ = tradingPairService
	_ = orderService
	_ = positionService
	_ = memoryService
	_ = journalService
	_ = agentRegistry
	_ = telegramHandlers // Handler registered with bot, used indirectly

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
		opportunityKafkaConsumer,
		aiUsageKafkaConsumer,
		positionGuardianConsumer,
		telegramNotificationConsumer,
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
	userRepo user.Repository,
	riskEngine *riskservice.RiskEngine,
	redisClient *redisclient.Client,
	embeddingProvider shared.EmbeddingProvider,
	kafkaProducer *kafka.Producer,
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
		UserRepo:            userRepo,
		RiskEngine:          riskEngine,
		EmbeddingProvider:   embeddingProvider,
		Redis:               redisClient,
		Log:                 log,
	}

	// Create WorkerPublisher for event publishing
	workerPublisher := events.NewWorkerPublisher(kafkaProducer)

	tools.RegisterAllTools(registry, deps, workerPublisher)
	log.Infof("✓ Registered %d tools", len(registry.List()))
	return registry
}

func provideAgents(
	cfg *config.Config,
	toolRegistry *tools.Registry,
	sessionService session.Service,
	redisClient *redisclient.Client,
	riskEngine *riskservice.RiskEngine,
	kafkaProducer *kafka.Producer,
	aiUsageRepo *chrepo.AIUsageRepository,
	reasoningRepo *pgrepo.ReasoningRepository,
	log *logger.Logger,
) (*agents.Factory, *agents.Registry, string, string, error) {
	log.Info("Initializing agents...")

	aiRegistry, err := ai.BuildRegistry(cfg.AI)
	if err != nil {
		return nil, nil, "", "", errors.Wrap(err, "build AI registry")
	}

	defaultProvider := resolveProvider(aiRegistry, cfg.AI.DefaultProvider)
	if defaultProvider == "" {
		return nil, nil, "", "", errors.ErrUnavailable
	}

	// Resolve model from config or auto-select
	selector := ai.NewModelSelector(aiRegistry, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var selectedModel string
	if cfg.AI.DefaultModel != "" {
		// Use model from ENV config
		selectedModel = cfg.AI.DefaultModel
		log.Infof("Using configured model from ENV: provider=%s model=%s", defaultProvider, selectedModel)
	} else {
		// Auto-select best model for provider
		modelCfg, modelInfo, err := selector.Get(ctx, string(agents.AgentOpportunitySynthesizer), defaultProvider)
		if err != nil {
			return nil, nil, "", "", errors.Wrap(err, "auto-select model")
		}
		selectedModel = modelCfg.Model
		log.Infof("Auto-selected model: provider=%s model=%s (%s)", defaultProvider, selectedModel, modelInfo.Name)
	}

	// Create cost check function using ClickHouse
	costCheckFunc := func(userID string) (bool, error) {
		cost, err := aiUsageRepo.GetUserDailyCost(context.Background(), userID, time.Now())
		if err != nil {
			return false, err
		}
		const dailyLimit = 10.0 // $10 daily limit per user
		return cost > dailyLimit, nil
	}

	// Create event publisher for callbacks
	eventPublisher := events.NewWorkerPublisher(kafkaProducer)

	factory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:      aiRegistry,
		ToolRegistry:    toolRegistry,
		Templates:       templates.Get(),
		SessionService:  sessionService,
		DefaultProvider: defaultProvider,
		DefaultModel:    selectedModel,
		CallbackDeps: &agents.CallbackDeps{
			Redis:          redisClient.Client(),
			EventPublisher: eventPublisher,
			CostCheckFunc:  costCheckFunc,
			RiskEngine:     riskEngine,
			ReasoningRepo:  reasoningRepo,
		},
	})
	if err != nil {
		return nil, nil, "", "", errors.Wrap(err, "create agent factory")
	}

	registry, err := factory.CreateDefaultRegistry(defaultProvider, selectedModel)
	if err != nil {
		return nil, nil, "", "", errors.Wrap(err, "create default agent registry")
	}

	log.Infof("✓ Agents initialized with provider=%s model=%s", defaultProvider, selectedModel)
	return factory, registry, defaultProvider, selectedModel, nil
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

func registerExpertTools(agentFactory *agents.Factory, toolRegistry *tools.Registry, provider, model string, log *logger.Logger) error {
	log.Info("Registering expert agent tools...")

	// Create expert tools using agent-as-tool pattern
	expertTools, err := agentFactory.CreateExpertTools(provider, model)
	if err != nil {
		return errors.Wrap(err, "failed to create expert tools")
	}

	// Register them in tool registry
	tools.RegisterExpertTools(toolRegistry, expertTools, log)

	return nil
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
	onchainRepo onchain.Repository,
	derivRepo derivatives.Repository,
	macroRepo macro.Repository,
	exchangeAccountRepo exchange_account.Repository,
	riskEngine *riskservice.RiskEngine,
	exchFactory exchanges.Factory,
	marketDataFactory exchanges.CentralFactory,
	encryptor *crypto.Encryptor,
	kafkaProducer *kafka.Producer,
	agentFactory *agents.Factory,
	agentRegistry *agents.Registry,
	adkSessionService session.Service,
	defaultProvider string,
	defaultModel string,
	cfg *config.Config,
	log *logger.Logger,
) *workers.Scheduler {
	log.Info("Initializing workers...")

	scheduler := workers.NewScheduler()

	// Create workflow factory for market research and personal trading workflows (passed from provideAgents)
	workflowFactory := workflows.NewFactory(agentFactory, defaultProvider, defaultModel)

	// Phase 3: Create PathSelector for intelligent routing between fast-path and committee
	// PathSelector routes analysis requests based on:
	// - Priority level (high/critical → committee)
	// - Position size (>10% portfolio → committee)
	// - Market volatility (>5% → committee)
	// - Conflicting signals (escalation from fast-path → committee)
	// - Cost control (target 20% of requests to committee)
	pathSelector, err := workflowFactory.CreatePathSelector()
	if err != nil {
		log.Fatal("Failed to create PathSelector", "error", err)
	}
	log.Info("PathSelector created successfully",
		"high_stakes_threshold", pathSelector.GetMetrics()["high_stakes_threshold"],
		"volatility_threshold", pathSelector.GetMetrics()["volatility_threshold"],
		"target_committee_pct", pathSelector.GetMetrics()["target_committee_pct"],
	)

	// TODO (Phase 3 integration): Update OpportunityFinder to use PathSelector
	// Instead of single workflow, use pathSelector.SelectPath() to route requests
	// Example:
	//   analysisAgent := pathSelector.SelectPath(ctx, workflows.AnalysisRequest{
	//       Symbol: symbol,
	//       Exchange: exchange,
	//       Priority: workflows.PriorityMedium,
	//   })
	//   result, err := analysisAgent.Run(ctx, input)

	// Default monitored symbols
	defaultSymbols := []string{"BTC/USDT", "ETH/USDT", "SOL/USDT", "BNB/USDT", "XRP/USDT"}

	// ========================================
	// Trading Workers (high frequency)
	// ========================================

	// Event publisher for position events
	eventPublisher := events.NewPublisher(kafkaProducer, log)

	// Position event generator for trigger checking
	positionEventGenerator := trading.NewPositionEventGenerator(eventPublisher, log)

	// Position monitor: Updates PnL and checks SL/TP
	scheduler.RegisterWorker(trading.NewPositionMonitor(
		userRepo,
		positionRepo,
		exchangeAccountRepo,
		exchFactory,
		*encryptor,
		kafkaProducer,
		positionEventGenerator,
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

	// Twitter collector: Collects tweets from crypto influencers
	trackedAccounts := []string{"APompliano", "100trillionUSD", "saylor", "elonmusk"}
	scheduler.RegisterWorker(sentimentworkers.NewTwitterCollector(
		sentimentRepo,
		cfg.MarketData.TwitterAPIKey,
		cfg.MarketData.TwitterAPISecret,
		trackedAccounts,
		defaultSymbols,
		cfg.Workers.TwitterCollectorInterval,
		cfg.MarketData.TwitterAPIKey != "", // Enable only if API key provided
	))

	// Reddit collector: Collects sentiment from crypto subreddits
	subreddits := []string{"CryptoCurrency", "Bitcoin", "ethereum", "ethtrader"}
	scheduler.RegisterWorker(sentimentworkers.NewRedditCollector(
		sentimentRepo,
		cfg.MarketData.RedditClientID,
		cfg.MarketData.RedditClientSecret,
		subreddits,
		defaultSymbols,
		cfg.Workers.RedditCollectorInterval,
		cfg.MarketData.RedditClientID != "", // Enable only if credentials provided
	))

	// Fear & Greed collector: Collects crypto Fear & Greed index (no auth required)
	scheduler.RegisterWorker(sentimentworkers.NewFearGreedCollector(
		sentimentRepo,
		cfg.Workers.FearGreedCollectorInterval,
		true, // Always enabled (free API)
	))

	// ========================================
	// On-Chain Workers (medium frequency)
	// ========================================

	// Whale movement collector: Tracks large on-chain transfers
	blockchains := []string{"bitcoin", "ethereum"}
	scheduler.RegisterWorker(onchainworkers.NewWhaleMovementCollector(
		onchainRepo,
		cfg.MarketData.WhaleAlertAPIKey,
		blockchains,
		1_000_000, // Min $1M transfers
		cfg.Workers.WhaleMovementCollectorInterval,
		cfg.MarketData.WhaleAlertAPIKey != "",
	))

	// Exchange flow collector: Tracks BTC/ETH flows to/from exchanges
	exchanges := []string{"binance", "coinbase", "kraken"}
	tokens := []string{"BTC", "ETH"}
	scheduler.RegisterWorker(onchainworkers.NewExchangeFlowCollector(
		onchainRepo,
		cfg.MarketData.CryptoquantAPIKey,
		exchanges,
		tokens,
		cfg.Workers.ExchangeFlowCollectorInterval,
		cfg.MarketData.CryptoquantAPIKey != "",
	))

	// Network metrics collector: Collects blockchain health metrics
	scheduler.RegisterWorker(onchainworkers.NewNetworkMetricsCollector(
		onchainRepo,
		cfg.MarketData.BlockchainAPIKey,
		cfg.MarketData.EtherscanAPIKey,
		blockchains,
		cfg.Workers.NetworkMetricsCollectorInterval,
		true, // Always enabled (free APIs available)
	))

	// Miner metrics collector: Tracks Bitcoin mining activity
	miningPools := []string{"AntPool", "Foundry USA", "F2Pool", "ViaBTC", "Binance Pool"}
	scheduler.RegisterWorker(onchainworkers.NewMinerMetricsCollector(
		onchainRepo,
		cfg.MarketData.GlassnodeAPIKey,
		miningPools,
		cfg.Workers.MinerMetricsCollectorInterval,
		cfg.MarketData.GlassnodeAPIKey != "",
	))

	// ========================================
	// Macro Workers (low frequency)
	// ========================================

	// Economic calendar collector: Tracks CPI, NFP, FOMC, GDP releases
	countries := []string{"United States", "Euro Area", "China"}
	eventTypes := []macro.EventType{macro.EventCPI, macro.EventNFP, macro.EventFOMC, macro.EventGDP}
	scheduler.RegisterWorker(macroworkers.NewEconomicCalendarCollector(
		macroRepo,
		cfg.MarketData.TradingEconomicsKey,
		countries,
		eventTypes,
		cfg.Workers.EconomicCalendarCollectorInterval,
		cfg.MarketData.TradingEconomicsKey != "",
	))

	// Market correlation collector: Tracks crypto correlation with SPX, Gold, DXY, Bonds
	// Analyzes multiple crypto assets to understand risk-on/risk-off dynamics
	cryptoSymbolsForCorr := []string{"BTC/USDT", "ETH/USDT", "SOL/USDT"}
	traditionalAssets := []string{"SPY", "GLD", "DXY", "TLT"}
	scheduler.RegisterWorker(macroworkers.NewMarketCorrelationCollector(
		marketDataRepo,
		macroRepo,
		cfg.MarketData.AlphaVantageKey,
		cryptoSymbolsForCorr,
		traditionalAssets,
		cfg.Workers.MarketCorrelationCollectorInterval,
		cfg.MarketData.AlphaVantageKey != "",
	))

	// ========================================
	// Derivatives Workers (medium frequency)
	// ========================================

	// Options flow collector: Tracks large options trades from Deribit
	optionsSymbols := []string{"BTC", "ETH"}
	scheduler.RegisterWorker(derivworkers.NewOptionsFlowCollector(
		derivRepo,
		cfg.MarketData.DeribitAPIKey,
		cfg.MarketData.DeribitAPISecret,
		optionsSymbols,
		100_000, // Min $100k premium
		cfg.Workers.OptionsFlowCollectorInterval,
		cfg.MarketData.DeribitAPIKey != "",
	))

	// Gamma exposure collector: Calculates gamma exposure and max pain
	scheduler.RegisterWorker(derivworkers.NewGammaExposureCollector(
		derivRepo,
		cfg.MarketData.DeribitAPIKey,
		optionsSymbols,
		cfg.Workers.GammaExposureCollectorInterval,
		cfg.MarketData.DeribitAPIKey != "",
	))

	// Funding aggregator: Aggregates funding rates from multiple exchanges
	scheduler.RegisterWorker(derivworkers.NewFundingAggregator(
		marketDataRepo,
		exchanges,
		defaultSymbols,
		cfg.Workers.FundingAggregatorInterval,
		true, // Always enabled (uses existing exchange connections)
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

	// Phase 5: Create pre-screener service for cost optimization
	preScreener := analysisservice.NewPreScreener(
		analysisservice.PreScreenConfig{
			Enabled:            true,
			MinPriceChangePct:  0.002, // 0.2%
			MinVolumePct:       0.50,  // 50%
			MinATRPct:          0.01,  // 1%
			CooldownDuration:   2 * time.Hour,
			CheckPriceMovement: true,
			CheckVolume:        true,
			CheckVolatility:    true,
			CheckOrderBook:     false,
			CheckCooldown:      true,
		},
		marketDataRepo,
	)

	// Opportunity finder: Runs market research workflow (8 analysts + synthesizer)
	// This replaces the old MarketScanner - now we do global analysis once per symbol
	// instead of per-user analysis (much more efficient)
	opportunityFinder, err := analysis.NewOpportunityFinder(
		workflowFactory,
		adkSessionService,
		templates.Get(),
		preScreener, // Phase 5 optimization
		defaultSymbols,
		"binance", // Primary exchange
		cfg.Workers.OpportunityFinderInterval,
		true, // enabled
	)
	if err != nil {
		log.Fatalf("Failed to create opportunity finder: %v", err)
	}
	scheduler.RegisterWorker(opportunityFinder)

	// Phase 4: ML-Based Regime Detection
	// Feature extractor: Extracts ML features from market data every hour
	scheduler.RegisterWorker(analysis.NewFeatureExtractor(
		marketDataRepo,
		regimeRepo,
		defaultSymbols,
		1*time.Hour, // Run every hour
		true,        // enabled
	))

	// Load ONNX regime classifier (optional - graceful degradation if model not found)
	regimeClassifier, classifierErr := regimeml.NewClassifier("models/regime_detector.onnx")
	if classifierErr != nil {
		log.Warn("Regime ML classifier not available, will use algorithmic fallback", "error", classifierErr)
		// Fall back to algorithmic regime detector
		scheduler.RegisterWorker(analysis.NewRegimeDetector(
			marketDataRepo,
			regimeRepo,
			kafkaProducer,
			defaultSymbols,
			cfg.Workers.RegimeDetectorInterval,
			true, // enabled
		))
	} else {
		log.Info("Regime ML classifier loaded successfully")
		defer regimeClassifier.Close()

		// RegimeInterpreter agent is NOT created here - it will be created on-demand
		// in regime_detector_ml.go when there is actual data to interpret
		// This avoids template rendering errors at startup (template requires runtime data)

		// Register ML-based regime detector with LLM interpretation (agent=nil for now)
		mlRegimeDetector, err := analysis.NewRegimeDetectorML(
			marketDataRepo,
			regimeRepo,
			regimeClassifier,
			nil, // interpreterAgent - not needed yet, will be used in future
			adkSessionService,
			kafkaProducer,
			defaultSymbols,
			cfg.Workers.RegimeDetectorInterval,
			true, // enabled
		)
		if err != nil {
			log.Fatal("Failed to create ML regime detector", "error", err)
		}
		scheduler.RegisterWorker(mlRegimeDetector)
	}

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

func provideHTTPServer(cfg *config.Config, healthHandler *health.Handler, telegramBot *telegram.Bot, telegramHandler *telegram.Handler, log *logger.Logger) *api.Server {
	// Create Telegram webhook handler if webhook URL configured
	var webhookHandler *telegramapi.WebhookHandler
	if cfg.Telegram.WebhookURL != "" {
		webhookHandler = telegramapi.NewWebhookHandler(telegramBot, telegramHandler, log)
		log.Info("Telegram webhook mode enabled", "url", cfg.Telegram.WebhookURL)
	} else {
		log.Info("Telegram polling mode enabled (no webhook URL configured)")
	}

	// HTTP server setup moved to internal/api (Clean Architecture)
	return api.NewServer(api.ServerConfig{
		Port:            cfg.HTTP.Port,
		ServiceName:     cfg.App.Name,
		Version:         cfg.App.Version,
		TelegramWebhook: webhookHandler,
	}, healthHandler, log)
}

func provideTelegramBot(
	cfg *config.Config,
	userRepo user.Repository,
	positionRepo position.Repository,
	exchAcctRepo exchange_account.Repository,
	workflowFactory *workflows.Factory,
	sessionService session.Service,
	redisClient *redisclient.Client,
	encryptor *crypto.Encryptor,
	exchFactory exchanges.Factory,
	notificationConsumer *kafka.Consumer,
	log *logger.Logger,
) (*telegram.Bot, *telegram.Handler, *consumers.TelegramNotificationConsumer) {
	log.Info("Initializing Telegram bot...")

	// Create bot client
	webhookMode := cfg.Telegram.WebhookURL != ""
	bot, err := telegram.NewBot(telegram.Config{
		Token:       cfg.Telegram.BotToken,
		Debug:       cfg.Telegram.Debug,
		Timeout:     60,
		BufferSize:  100,
		WebhookMode: webhookMode,
	}, log)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	// Create notification service
	notificationService := telegram.NewNotificationService(bot, templates.Get(), log)

	// Create onboarding orchestrator
	onboardingOrchestrator := onboardingservice.NewService(
		workflowFactory,
		sessionService,
		templates.Get(),
		userRepo,
		exchAcctRepo,
		log,
	)

	// Create onboarding service
	onboardingService := telegram.NewOnboardingService(
		redisClient.Client(),
		bot,
		userRepo,
		exchAcctRepo,
		onboardingOrchestrator,
		templates.Get(),
		log,
	)

	// Create exchange setup service
	exchangeSetupService := telegram.NewExchangeSetupService(
		redisClient.Client(),
		bot,
		exchAcctRepo,
		exchFactory,
		encryptor,
		templates.Get(),
		log,
	)

	// Create query command handler (status, portfolio)
	queryHandler := telegram.NewQueryCommandHandler(
		positionRepo,
		userRepo,
		templates.Get(),
		bot,
		log,
	)

	// Create control command handler (stop, settings)
	controlHandler := telegram.NewControlCommandHandler(
		positionRepo,
		userRepo,
		templates.Get(),
		bot,
		log,
	)

	// Create main handler
	handler := telegram.NewHandler(telegram.HandlerDeps{
		Bot:              bot,
		UserRepo:         userRepo,
		OnboardingMgr:    onboardingService,
		StatusHandler:    queryHandler,
		PortfolioHandler: queryHandler,
		ControlHandler:   controlHandler,
		ExchangeHandler:  exchangeSetupService,
		Templates:        templates.Get(),
		Log:              log,
	})

	// Register handlers with bot
	handler.RegisterHandlers()

	// Set up webhook if configured
	if cfg.Telegram.WebhookURL != "" {
		log.Info("Configuring Telegram webhook...", "url", cfg.Telegram.WebhookURL)
		if err := bot.SetWebhook(cfg.Telegram.WebhookURL); err != nil {
			log.Fatalf("Failed to set Telegram webhook: %v", err)
		}

		// Verify webhook was set
		if webhookInfo, err := bot.GetWebhookInfo(); err == nil {
			log.Info("✓ Telegram webhook configured",
				"url", webhookInfo.URL,
				"pending_updates", webhookInfo.PendingUpdateCount,
				"max_connections", webhookInfo.MaxConnections,
			)
		}
	}

	// Create Telegram notification consumer
	telegramNotificationSvc := consumers.NewTelegramNotificationConsumer(
		notificationConsumer,
		bot,
		notificationService,
		userRepo,
		log,
	)

	log.Info("✓ Telegram bot initialized")
	return bot, handler, telegramNotificationSvc
}

// gracefulShutdown handles coordinated cleanup of all components
// Note: Application context is already cancelled by caller before this function
//
// Shutdown sequence:
// 1. Stop HTTP server (no new requests)
// 2. Stop background workers (market data collectors, etc.)
// 3. Close Kafka consumers (unblock ReadMessage calls)
// 4. Wait for consumer goroutines (should exit quickly now)
// 5. Close Kafka producer (after consumers finished)
// 6. Flush error tracker
// 7. Sync logs
// 8. Close database connections (last - other components may need them)
func gracefulShutdown(
	wg *sync.WaitGroup,
	httpServer *api.Server,
	workerScheduler *workers.Scheduler,
	marketDataFactory exchanges.CentralFactory,
	kafkaProducer *kafka.Producer,
	notificationConsumer *kafka.Consumer,
	riskConsumer *kafka.Consumer,
	analyticsConsumer *kafka.Consumer,
	opportunityKafkaConsumer *kafka.Consumer,
	aiUsageKafkaConsumer *kafka.Consumer,
	positionGuardianConsumer *kafka.Consumer,
	telegramNotificationConsumer *kafka.Consumer,
	pgClient *pgclient.Client,
	chClient *chclient.Client,
	redisClient *redisclient.Client,
	errorTracker errors.Tracker,
	log *logger.Logger,
) {
	// Create shutdown context with timeout (2.5 minutes for complete cleanup)
	// This accommodates the 2-minute worker timeout + time for other cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer shutdownCancel()

	// Cleanup sequence (order matters!)
	// Note: Application context is already cancelled
	// Workers and goroutines are finishing their current work
	// We wait for graceful completion with timeouts

	// Step 1: Stop accepting new HTTP requests (5s timeout)
	httpCtx, httpCancel := context.WithTimeout(shutdownCtx, 5*time.Second)
	defer httpCancel()
	if err := httpServer.Shutdown(httpCtx); err != nil {
		log.Error("HTTP server shutdown failed", "error", err)
	}

	// Step 2: Stop background workers (they check ctx.Done() in their loops)
	// Scheduler.Stop() calls wg.Wait() internally with 30s timeout
	log.Info("[2/7] Stopping background workers...")
	if err := workerScheduler.Stop(); err != nil {
		log.Error("Workers shutdown failed", "error", err)
	} else {
		log.Info("✓ Workers stopped")
	}

	// Step 3: Close Kafka consumers FIRST to unblock ReadMessage() calls
	// This is critical - consumer.Close() will interrupt blocking ReadMessage()
	log.Info("[3/7] Closing Kafka consumers (to unblock ReadMessage)...")
	for name, consumer := range map[string]*kafka.Consumer{
		"notification":           notificationConsumer,
		"risk":                   riskConsumer,
		"analytics":              analyticsConsumer,
		"opportunity":            opportunityKafkaConsumer,
		"ai_usage":               aiUsageKafkaConsumer,
		"position_guardian":      positionGuardianConsumer,
		"telegram_notifications": telegramNotificationConsumer,
	} {
		if consumer != nil {
			if err := consumer.Close(); err != nil {
				log.Error("Kafka consumer close failed", "consumer", name, "error", err)
			}
		}
	}
	log.Info("✓ Kafka consumers closed")

	// Step 4: Wait for consumer goroutines (now they should exit quickly)
	log.Info("[4/7] Waiting for consumer goroutines...")
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

	// Step 5: Close Kafka producer (after consumers finished)
	log.Info("[5/7] Closing Kafka producer...")
	if kafkaProducer != nil {
		if err := kafkaProducer.Close(); err != nil {
			log.Error("Kafka producer close failed", "error", err)
		}
	}
	log.Info("✓ Kafka producer closed")

	// Step 6: Flush error tracker
	log.Info("[6/7] Flushing error tracker...")
	if errorTracker != nil {
		flushCtx, flushCancel := context.WithTimeout(shutdownCtx, 3*time.Second)
		defer flushCancel()
		if err := errorTracker.Flush(flushCtx); err != nil {
			log.Error("Error tracker flush failed", "error", err)
		} else {
			log.Info("✓ Error tracker flushed")
		}
	}

	// Step 7: Sync logs
	log.Info("[7/7] Syncing logs...")
	if err := logger.Sync(); err != nil {
		// Ignore "sync /dev/stderr: inappropriate ioctl for device" on Linux
		log.Warn("Log sync completed with warnings")
	} else {
		log.Info("✓ Logs synced")
	}

	// Step 8: Close database connections (LAST - other components may need them)
	log.Info("[8/8] Closing database connections...")
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
