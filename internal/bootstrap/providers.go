package bootstrap

import (
	"context"
	"time"

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
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/journal"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	domainRisk "prometheus/internal/domain/risk"
	domainsession "prometheus/internal/domain/session"
	"prometheus/internal/domain/trading_pair"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/metrics"
	chrepo "prometheus/internal/repository/clickhouse"
	pgrepo "prometheus/internal/repository/postgres"
	"prometheus/internal/repository/redis"
	aiusagesvc "prometheus/internal/services/ai_usage"
	"prometheus/internal/services/exchange"
	limitsservice "prometheus/internal/services/limits"
	marketdatasvc "prometheus/internal/services/market_data"
	"prometheus/internal/services/menu_session"
	onboardingservice "prometheus/internal/services/onboarding"
	positionservice "prometheus/internal/services/position"
	riskservice "prometheus/internal/services/risk"
	"prometheus/internal/tools"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"

	"github.com/google/uuid"
	"google.golang.org/adk/session"
)

// ========================================
// Phase 1: Configuration & Logging
// ========================================

// MustInitConfig loads configuration and initializes logger
func (c *Container) MustInitConfig() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}
	c.Config = cfg

	// Initialize logger
	if err := logger.Init(cfg.App.LogLevel, cfg.App.Env); err != nil {
		panic("failed to init logger: " + err.Error())
	}

	c.Log = logger.Get()
	c.Log.Infof("Starting %s in %s mode", cfg.App.Name, cfg.App.Env)

	// Initialize error tracker
	c.ErrorTracker = provideErrorTracker(cfg, c.Log)
	logger.SetErrorTracker(c.ErrorTracker)
}

// ========================================
// Phase 2: Infrastructure Layer
// ========================================

// MustInitInfrastructure initializes data stores (Postgres, ClickHouse, Redis)
func (c *Container) MustInitInfrastructure() {
	var err error

	// PostgreSQL
	c.Log.Info("Connecting to PostgreSQL...")
	c.PG, err = pgclient.NewClient(c.Config.Postgres)
	if err != nil {
		c.Log.Fatalf("failed to connect postgres: %v", err)
	}
	c.Log.Info("✓ PostgreSQL connected")

	// ClickHouse
	c.Log.Info("Connecting to ClickHouse...")
	c.CH, err = chclient.NewClient(c.Config.ClickHouse)
	if err != nil {
		c.Log.Fatalf("failed to connect clickhouse: %v", err)
	}
	c.Log.Info("✓ ClickHouse connected")

	// Redis
	c.Log.Info("Connecting to Redis...")
	c.Redis, err = redisclient.NewClient(c.Config.Redis)
	if err != nil {
		c.Log.Fatalf("failed to connect redis: %v", err)
	}
	c.Log.Info("✓ Redis connected")
}

// ========================================
// Phase 3: Domain Layer - Repositories
// ========================================

// MustInitRepositories initializes all domain repositories
func (c *Container) MustInitRepositories() {
	c.Repos.User = pgrepo.NewUserRepository(c.PG.DB())
	c.Repos.ExchangeAccount = pgrepo.NewExchangeAccountRepository(c.PG.DB())
	c.Repos.TradingPair = pgrepo.NewTradingPairRepository(c.PG.DB())
	c.Repos.Order = pgrepo.NewOrderRepository(c.PG.DB())
	c.Repos.Position = pgrepo.NewPositionRepository(c.PG.DB())
	c.Repos.Memory = pgrepo.NewMemoryRepository(c.PG.DB())
	c.Repos.Journal = pgrepo.NewJournalRepository(c.PG.DB())
	c.Repos.Risk = pgrepo.NewRiskRepository(c.PG.DB())
	c.Repos.Session = pgrepo.NewSessionRepository(c.PG.DB())
	c.Repos.Reasoning = pgrepo.NewReasoningRepository(c.PG.DB())
	c.Repos.LimitProfile = pgrepo.NewLimitProfileRepository(c.PG.DB())
	c.Repos.MarketData = chrepo.NewMarketDataRepository(c.CH.Conn())
	c.Repos.Regime = chrepo.NewRegimeRepository(c.CH.Conn())
	c.Repos.Sentiment = chrepo.NewSentimentRepository(c.CH.Conn())
	c.Repos.OnChain = chrepo.NewOnChainRepository(c.CH.Conn())
	c.Repos.Derivatives = chrepo.NewDerivativesRepository(c.CH.Conn())
	c.Repos.Macro = chrepo.NewMacroRepository(c.CH.Conn())
	c.Repos.AIUsage = chrepo.NewAIUsageRepository(c.CH.Conn())

	c.Log.Info("✓ Repositories initialized")
}

// ========================================
// Phase 4: External Adapters
// ========================================

// MustInitAdapters initializes external adapters (Kafka, Exchange, Crypto, Embeddings)
func (c *Container) MustInitAdapters() {
	var err error

	// Kafka
	c.Adapters.KafkaProducer = provideKafkaProducer(c.Config, c.Log)
	// NOTE: Old NotificationConsumer removed - replaced with TelegramNotificationConsumer
	// to avoid two consumers competing for same messages on notifications topic
	c.Adapters.NotificationConsumer = nil // Not used anymore, kept for backwards compatibility
	c.Adapters.RiskConsumer = provideKafkaConsumer(c.Config, events.TopicRiskEvents, c.Log)
	c.Adapters.AnalyticsConsumer = provideKafkaConsumer(c.Config, events.TopicAnalytics, c.Log)
	c.Adapters.OpportunityConsumer = provideKafkaConsumer(c.Config, events.TopicMarketEvents, c.Log)
	c.Adapters.AIUsageConsumer = provideKafkaConsumer(c.Config, events.TopicAIEvents, c.Log)
	c.Adapters.PositionGuardianConsumer = provideKafkaConsumer(c.Config, events.TopicPositionEvents, c.Log)
	c.Adapters.TelegramNotificationConsumer = provideKafkaConsumer(c.Config, events.TopicNotifications, c.Log)

	// WebSocket consumer (unified consumer for ALL websocket.events)
	// Single consumer handles all WebSocket event types:
	// - Market data: kline, ticker, depth, trade, mark_price, liquidation, funding_rate
	// - User data: orders, positions, balance, margin_call, account_config
	// Routes by wrapper type (WebSocketEventWrapper vs UserDataEventWrapper) and batches inserts
	c.Adapters.WebSocketConsumer = provideKafkaConsumer(c.Config, events.TopicWebSocketEvents, c.Log)

	// Crypto
	c.Adapters.Encryptor, err = crypto.NewEncryptor(c.Config.Crypto.EncryptionKey)
	if err != nil {
		c.Log.Fatalf("failed to initialize encryptor: %v", err)
	}

	// Exchanges
	c.Adapters.ExchangeFactory = exchangefactory.NewFactory()
	c.Adapters.MarketDataFactory = exchangefactory.NewMarketDataFactory(c.Config.MarketData)
	c.Log.Info("✓ Exchange factory initialized")

	// Embeddings
	c.Adapters.EmbeddingProvider, err = embeddings.NewProvider(embeddings.Config{
		Provider: embeddings.ProviderOpenAI,
		APIKey:   c.Config.AI.OpenAIKey,
		Model:    "text-embedding-3-small",
		Timeout:  30 * time.Second,
	})
	if err != nil {
		c.Log.Fatalf("failed to create embedding provider: %v", err)
	}
	c.Log.Infof("✓ Embedding provider initialized: %s (%d dimensions)",
		c.Adapters.EmbeddingProvider.Name(),
		c.Adapters.EmbeddingProvider.Dimensions(),
	)
}

// ========================================
// Phase 5: Domain Services
// ========================================

// MustInitServices initializes domain services
func (c *Container) MustInitServices() {
	// Create limit profile adapter to avoid circular dependency
	limitProfileAdapter := user.NewLimitProfileAdapter(func(ctx context.Context, name string) (uuid.UUID, error) {
		profile, err := c.Repos.LimitProfile.GetByName(ctx, name)
		if err != nil {
			return uuid.Nil, err
		}
		return profile.ID, nil
	})

	// Core domain services
	c.Services.User = user.NewService(c.Repos.User, limitProfileAdapter)
	c.Services.ExchangeAccount = exchange_account.NewService(c.Repos.ExchangeAccount)
	c.Services.TradingPair = trading_pair.NewService(c.Repos.TradingPair)
	c.Services.Order = order.NewService(c.Repos.Order)
	c.Services.Position = position.NewService(c.Repos.Position)
	c.Services.Memory = memory.NewService(c.Repos.Memory, c.Adapters.EmbeddingProvider)
	c.Services.Journal = journal.NewService(c.Repos.Journal)
	c.Services.Session = domainsession.NewService(c.Repos.Session)
	c.Services.ADKSession = adk.NewSessionService(c.Services.Session)

	// Position management service for WebSocket updates
	c.Services.PositionManagement = positionservice.NewService(c.Repos.Position, c.Log)

	// Limits service for tier/subscription management
	c.Services.Limits = limitsservice.NewService(
		c.Repos.LimitProfile,
		c.Repos.ExchangeAccount,
		c.Repos.User,
	)

	// Exchange service with notification publisher for Telegram
	notificationPublisher := events.NewNotificationPublisher(c.Adapters.KafkaProducer)
	c.Services.Exchange = exchange.NewService(
		c.Repos.ExchangeAccount,
		c.Adapters.Encryptor,
		c.Services.Limits, // Inject limits service for validation
		notificationPublisher,
		c.Log,
	)

	// Consumer-facing services (Clean Architecture: abstraction over repositories)
	c.Services.MarketData = marketdatasvc.NewService(c.Repos.MarketData, c.Log)
	c.Services.AIUsage = aiusagesvc.NewService(c.Repos.AIUsage, c.Log)

	c.Log.Info("✓ Services initialized")
}

// ========================================
// Phase 6: Business Logic
// ========================================

// MustInitBusiness initializes business logic (Risk, Tools, Agents)
func (c *Container) MustInitBusiness() {
	// Risk engine
	c.Business.RiskEngine = riskservice.NewRiskEngine(
		c.Repos.Risk,
		c.Repos.Position,
		c.Repos.User,
		c.Redis,
		c.Log,
	)

	// Tool registry
	c.Business.ToolRegistry = provideToolRegistry(
		c.Repos.MarketData,
		c.Repos.Order,
		c.Repos.Position,
		c.Repos.ExchangeAccount,
		c.Repos.Memory,
		c.Repos.Risk,
		c.Repos.User,
		c.Business.RiskEngine,
		c.Redis,
		c.Adapters.EmbeddingProvider,
		c.Adapters.KafkaProducer,
		c.Log,
	)

	// Agent factory and registry
	var err error
	c.Business.AgentFactory,
		c.Business.AgentRegistry,
		c.Business.DefaultProvider,
		c.Business.DefaultModel,
		err = provideAgents(
		c.Config,
		c.Business.ToolRegistry,
		c.Services.ADKSession,
		c.Redis,
		c.Business.RiskEngine,
		c.Adapters.KafkaProducer,
		c.Repos.AIUsage,
		c.Repos.Reasoning,
		c.Log,
	)
	if err != nil {
		c.Log.Fatalf("failed to initialize agents: %v", err)
	}

	// Register expert agent tools
	if err := registerExpertTools(
		c.Business.AgentFactory,
		c.Business.ToolRegistry,
		c.Business.DefaultProvider,
		c.Business.DefaultModel,
		c.Log,
	); err != nil {
		c.Log.Warnf("Failed to register expert tools: %v", err)
	}

	// Workflow factory
	c.Business.WorkflowFactory = workflows.NewFactory(
		c.Business.AgentFactory,
		c.Business.DefaultProvider,
		c.Business.DefaultModel,
	)

	c.Log.With("tools", len(c.Business.ToolRegistry.List())).Info("✓ Business logic initialized")
}

// ========================================
// Phase 7: Application Layer
// ========================================

// MustInitApplication initializes application layer (HTTP, Telegram)
func (c *Container) MustInitApplication() {
	// Health handler
	c.Application.HealthHandler = health.New(
		c.Log,
		c.PG.DB(),
		c.CH.Conn(),
		c.Redis.Client(),
		c.Config.App.Name,
		c.Config.App.Version,
	)

	// Telegram bot and handlers
	var telegramHandler *telegram.Handler
	c.Application.TelegramBot,
		telegramHandler,
		c.Application.TelegramNotificationSvc = provideTelegramBot(
		c.Config,
		c.Services.User, // User service (Clean Architecture)
		c.Repos.User,    // Still needed for some components (onboarding)
		c.Repos.Position,
		c.Repos.ExchangeAccount,
		c.Business.WorkflowFactory,
		c.Services.ADKSession,
		c.Redis,
		c.Adapters.Encryptor,
		c.Adapters.ExchangeFactory,
		c.Adapters.KafkaProducer,
		c.Adapters.TelegramNotificationConsumer,
		c.Services.Exchange, // Exchange service (Clean Architecture)
		c.Log,
	)

	// HTTP server
	c.Application.HTTPServer = provideHTTPServer(
		c.Config,
		c.Application.HealthHandler,
		c.Application.TelegramBot,
		telegramHandler,
		c.Log,
	)

	// Initialize metrics
	metrics.Init()
	customCollector := metrics.NewCustomCollector(c.Log, c.PG.DB(), c.CH.Conn(), c.Redis.Client())
	metrics.RegisterCustomCollector(customCollector)
	c.Log.Info("✓ Metrics initialized")

	c.Log.Info("✓ Application layer initialized")
}

// ========================================
// Phase 8: Background Processing
// ========================================

// MustInitBackground initializes background workers and consumers
func (c *Container) MustInitBackground() {
	// Workers
	c.Background.WorkerScheduler = provideWorkers(
		c.Repos.User,
		c.Repos.TradingPair,
		c.Repos.Order,
		c.Repos.Position,
		c.Repos.Journal,
		c.Repos.MarketData,
		c.Repos.Regime,
		c.Repos.Sentiment,
		c.Repos.OnChain,
		c.Repos.Derivatives,
		c.Repos.Macro,
		c.Repos.ExchangeAccount,
		c.Business.RiskEngine,
		c.Adapters.ExchangeFactory,
		c.Adapters.MarketDataFactory,
		c.Adapters.Encryptor,
		c.Adapters.KafkaProducer,
		c.Business.AgentFactory,
		c.Business.AgentRegistry,
		c.Services.ADKSession,
		c.Business.DefaultProvider,
		c.Business.DefaultModel,
		c.Config,
		c.Log,
	)

	// Event consumers
	// NOTE: Old NotificationConsumer disabled - replaced with TelegramNotificationConsumer
	// to avoid two consumers competing for same messages on notifications topic
	c.Background.NotificationSvc = nil // Not used anymore

	c.Background.RiskSvc = consumers.NewRiskConsumer(
		c.Adapters.RiskConsumer,
		c.Business.RiskEngine,
		c.Log,
	)

	c.Background.AnalyticsSvc = consumers.NewAnalyticsConsumer(
		c.Adapters.AnalyticsConsumer,
		c.Log,
	)

	// AI Usage Consumer - uses AIUsageService (Clean Architecture)
	c.Background.AIUsageSvc = consumers.NewAIUsageConsumer(
		c.Adapters.AIUsageConsumer,
		c.Services.AIUsage,
		c.Log,
	)

	// Opportunity Consumer - uses TradingPairRepo (for GetActiveBySymbol) and UserService
	c.Background.OpportunitySvc = consumers.NewOpportunityConsumer(
		c.Adapters.OpportunityConsumer,
		c.Repos.TradingPair,
		c.Services.User,
		c.Business.WorkflowFactory,
		c.Services.ADKSession,
		c.Config.Workers.MarketScannerMaxConcurrency,
		c.Log,
	)

	// Position guardian
	criticalHandler := positionservice.NewCriticalEventHandler(c.Repos.Position, c.Log)

	var agentHandler *positionservice.AgentEventHandler
	positionManagerAgent, agentErr := c.Business.AgentFactory.CreateAgentForUser(
		agents.AgentPositionManager,
		c.Business.DefaultProvider,
		c.Business.DefaultModel,
	)
	if agentErr != nil {
		c.Log.Warnf("PositionManager agent creation failed, agent handler disabled: %v", agentErr)
	} else {
		agentHandler = positionservice.NewAgentEventHandler(
			c.Repos.Position,
			positionManagerAgent,
			c.Services.ADKSession,
			c.Log,
			30*time.Second,
		)
		c.Log.Info("✓ Agent event handler initialized with PositionManager agent")
	}

	c.Background.PositionGuardianSvc = consumers.NewPositionGuardianConsumer(
		c.Adapters.PositionGuardianConsumer,
		criticalHandler,
		agentHandler,
		c.Log,
	)

	c.Log.Info("✓ Background processing initialized")
}

// ========================================
// Helper Provider Functions
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

	log.Info("✓ Error tracking initialized (Sentry)")
	return tracker
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
	log.Infow("Initializing Kafka consumer", "topic", topic)
	if len(cfg.Kafka.Brokers) == 0 {
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	consumer := kafka.NewConsumer(kafka.ConsumerConfig{
		Brokers: cfg.Kafka.Brokers,
		GroupID: cfg.Kafka.GroupID,
		Topic:   topic,
	})
	log.Infow("✓ Kafka consumer initialized", "topic", topic)
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

	// Resolve model
	selector := ai.NewModelSelector(aiRegistry, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var selectedModel string
	if cfg.AI.DefaultModel != "" {
		selectedModel = cfg.AI.DefaultModel
		log.Infof("Using configured model from ENV: provider=%s model=%s", defaultProvider, selectedModel)
	} else {
		modelCfg, modelInfo, err := selector.Get(ctx, string(agents.AgentOpportunitySynthesizer), defaultProvider)
		if err != nil {
			return nil, nil, "", "", errors.Wrap(err, "auto-select model")
		}
		selectedModel = modelCfg.Model
		log.Infof("Auto-selected model: provider=%s model=%s (%s)", defaultProvider, selectedModel, modelInfo.Name)
	}

	// Cost check function
	costCheckFunc := func(userID string) (bool, error) {
		cost, err := aiUsageRepo.GetUserDailyCost(context.Background(), userID, time.Now())
		if err != nil {
			return false, err
		}
		const dailyLimit = 10.0
		return cost > dailyLimit, nil
	}

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

func registerExpertTools(
	agentFactory *agents.Factory,
	toolRegistry *tools.Registry,
	provider, model string,
	log *logger.Logger,
) error {
	log.Info("Registering expert agent tools...")

	expertTools, err := agentFactory.CreateExpertTools(provider, model)
	if err != nil {
		return errors.Wrap(err, "failed to create expert tools")
	}

	tools.RegisterExpertTools(toolRegistry, expertTools, log)
	return nil
}

func provideHTTPServer(
	cfg *config.Config,
	healthHandler *health.Handler,
	telegramBot *telegram.Bot,
	telegramHandler *telegram.Handler,
	log *logger.Logger,
) *api.Server {
	var webhookHandler *telegramapi.WebhookHandler
	if cfg.Telegram.WebhookURL != "" {
		webhookHandler = telegramapi.NewWebhookHandler(telegramBot, telegramHandler, log)
		log.Infow("✓ Telegram webhook mode enabled", "url", cfg.Telegram.WebhookURL)
	} else {
		log.Info("✓ Telegram polling mode enabled")
	}

	return api.NewServer(api.ServerConfig{
		Port:            cfg.HTTP.Port,
		ServiceName:     cfg.App.Name,
		Version:         cfg.App.Version,
		TelegramWebhook: webhookHandler,
	}, healthHandler, log)
}

func provideTelegramBot(
	cfg *config.Config,
	userService *user.Service,
	userRepo user.Repository,
	positionRepo position.Repository,
	exchAcctRepo exchange_account.Repository,
	workflowFactory *workflows.Factory,
	sessionService session.Service,
	redisClient *redisclient.Client,
	encryptor *crypto.Encryptor,
	exchFactory exchanges.Factory,
	kafkaProducer *kafka.Producer,
	notificationConsumer *kafka.Consumer,
	exchangeService *exchange.Service,
	log *logger.Logger,
) (*telegram.Bot, *telegram.Handler, *consumers.TelegramNotificationConsumer) {
	log.Info("Initializing Telegram bot...")

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

	notificationService := telegram.NewNotificationService(bot, templates.Get(), log)

	onboardingOrchestrator := onboardingservice.NewService(
		workflowFactory,
		sessionService,
		templates.Get(),
		userRepo,
		exchAcctRepo,
		log,
	)

	onboardingService := telegram.NewOnboardingService(
		redisClient.Client(),
		bot,
		userRepo,
		exchAcctRepo,
		onboardingOrchestrator,
		templates.Get(),
		log,
	)

	// Create Telegram UI adapter (wraps business service)
	exchangeSetupService := telegram.NewExchangeSetupService(
		redisClient.Client(),
		bot,
		exchangeService, // Injected from container
		exchFactory,
		templates.Get(),
		cfg.MarketData.Binance.UseTestnet, // Binance testnet flag
		cfg.MarketData.Bybit.UseTestnet,   // Bybit testnet flag
		cfg.MarketData.OKX.UseTestnet,     // OKX testnet flag
		log,
	)

	// Menu session repository and service (Clean Architecture)
	menuSessionRepo := redis.NewMenuSessionRepository(redisClient.Client())
	menuSessionService := menu_session.NewService(menuSessionRepo, log)

	// Menu navigator for interactive menus
	menuNavigator := telegram.NewMenuNavigator(
		menuSessionService,
		bot,
		templates.Get(),
		log,
		30*time.Minute, // Session TTL
	)

	// Invest menu service (uses MenuNavigator and services, not repos)
	investMenuService := telegram.NewInvestMenuService(
		menuNavigator,
		exchangeService, // Use service, not repo (Clean Architecture)
		userService,     // Use service, not repo (Clean Architecture)
		onboardingOrchestrator,
		log,
	)

	// Menu registry for auto-routing
	menuRegistry := telegram.NewMenuRegistry(log)
	menuRegistry.Register(investMenuService)
	// TODO: Register other menu handlers (settings, etc.)

	queryHandler := telegram.NewQueryCommandHandler(
		positionRepo,
		userService, // Use service instead of repository (Clean Architecture)
		templates.Get(),
		bot,
		log,
	)

	controlHandler := telegram.NewControlCommandHandler(
		positionRepo,
		userService, // Use service instead of repository (Clean Architecture)
		templates.Get(),
		bot,
		log,
	)

	handler := telegram.NewHandler(telegram.HandlerDeps{
		Bot:              bot,
		UserService:      userService,
		OnboardingMgr:    onboardingService,
		InvestMenu:       investMenuService, // NEW: Menu-based invest flow
		MenuRegistry:     menuRegistry,      // Auto-routing for all menus
		StatusHandler:    queryHandler,
		PortfolioHandler: queryHandler,
		ControlHandler:   controlHandler,
		ExchangeHandler:  exchangeSetupService,
		Templates:        templates.Get(),
		Log:              log,
	})

	handler.RegisterHandlers()

	if cfg.Telegram.WebhookURL != "" {
		log.Infow("Configuring Telegram webhook...", "url", cfg.Telegram.WebhookURL)
		if err := bot.SetWebhook(cfg.Telegram.WebhookURL); err != nil {
			log.Fatalf("Failed to set Telegram webhook: %v", err)
		}

		if webhookInfo, err := bot.GetWebhookInfo(); err == nil {
			log.Infow("✓ Telegram webhook configured",
				"url", webhookInfo.URL,
				"pending_updates", webhookInfo.PendingUpdateCount,
			)
		}
	}

	// Telegram Notification Consumer - uses UserService (Clean Architecture)
	telegramNotificationSvc := consumers.NewTelegramNotificationConsumer(
		notificationConsumer,
		bot,
		notificationService,
		userService,
		log,
	)

	log.Info("✓ Telegram bot initialized")
	return bot, handler, telegramNotificationSvc
}

// provideWorkers is defined in workers.go due to its size
