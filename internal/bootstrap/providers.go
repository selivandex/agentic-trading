package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
	graphql "prometheus/internal/api/graphql"
	"prometheus/internal/api/health"
	tgapi "prometheus/internal/api/telegram"
	"prometheus/internal/consumers"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/domain/limit_profile"
	"prometheus/internal/domain/market_data"
	"prometheus/internal/domain/memory"
	"prometheus/internal/domain/order"
	"prometheus/internal/domain/position"
	domainRisk "prometheus/internal/domain/risk"
	domainsession "prometheus/internal/domain/session"
	strategyDomain "prometheus/internal/domain/strategy"
	"prometheus/internal/domain/user"
	"prometheus/internal/events"
	"prometheus/internal/metrics"
	chrepo "prometheus/internal/repository/clickhouse"
	pgrepo "prometheus/internal/repository/postgres"
	"prometheus/internal/repository/redis"
	aiusagesvc "prometheus/internal/services/ai_usage"
	"prometheus/internal/services/exchange"
	fundwatchlistsvc "prometheus/internal/services/fund_watchlist"
	marketdatasvc "prometheus/internal/services/market_data"
	"prometheus/internal/services/menu_session"
	onboardingservice "prometheus/internal/services/onboarding"
	positionservice "prometheus/internal/services/position"
	riskservice "prometheus/internal/services/risk"
	strategyservice "prometheus/internal/services/strategy"
	userservice "prometheus/internal/services/user"
	"prometheus/internal/tools"
	"prometheus/internal/tools/shared"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	tg "prometheus/pkg/telegram"
	"prometheus/pkg/telegram/adapters/tgbotapi"
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
	c.Repos.FundWatchlist = pgrepo.NewFundWatchlistRepository(c.PG.DB())
	c.Repos.Order = pgrepo.NewOrderRepository(c.PG.DB())
	c.Repos.Position = pgrepo.NewPositionRepository(c.PG.DB())
	c.Repos.Memory = pgrepo.NewMemoryRepository(c.PG.DB())
	c.Repos.Risk = pgrepo.NewRiskRepository(c.PG.DB())
	c.Repos.Session = pgrepo.NewSessionRepository(c.PG.DB())
	c.Repos.LimitProfile = pgrepo.NewLimitProfileRepository(c.PG.DB())
	c.Repos.Strategy = pgrepo.NewStrategyRepository(c.PG.DB())
	c.Repos.StrategyTransaction = pgrepo.NewStrategyTransactionRepository(c.PG.DB())
	c.Repos.MarketData = chrepo.NewMarketDataRepository(c.CH.Conn())
	c.Repos.Stats = chrepo.NewStatsRepository(c.CH.Conn())
	c.Repos.Regime = chrepo.NewRegimeRepository(c.CH.Conn())
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
	c.Adapters.OpportunityConsumer = provideKafkaConsumer(c.Config, events.TopicMarketEvents, c.Log)
	c.Adapters.TelegramNotificationConsumer = provideKafkaConsumer(c.Config, events.TopicNotifications, c.Log)
	c.Adapters.SystemEventsConsumer = provideKafkaConsumer(c.Config, events.TopicSystemEvents, c.Log)

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
	// Create limit profile adapter for domain service
	limitProfileAdapter := user.NewLimitProfileAdapter(func(ctx context.Context, name string) (uuid.UUID, error) {
		profile, err := c.Repos.LimitProfile.GetByName(ctx, name)
		if err != nil {
			return uuid.Nil, err
		}
		return profile.ID, nil
	})

	// Domain user service (pure business logic - for consumers)
	c.Services.DomainUser = user.NewService(c.Repos.User, limitProfileAdapter)

	// Application user service (wraps domain service + adds side effects)
	c.Services.User = userservice.NewService(c.Services.DomainUser, c.Log)
	c.Services.ExchangeAccount = exchange_account.NewService(c.Repos.ExchangeAccount)
	c.Services.Order = order.NewService(c.Repos.Order)
	c.Services.Position = position.NewService(c.Repos.Position)
	c.Services.Memory = memory.NewService(c.Repos.Memory, c.Adapters.EmbeddingProvider)

	// Strategy Service (Clean Architecture: Domain + Application layers)
	strategyDomainService := strategyDomain.NewService(c.Repos.Strategy)
	dbAdapter := strategyservice.NewDBAdapter(c.PG.DB())
	c.Services.Strategy = strategyservice.NewService(
		strategyDomainService, // Domain service for CRUD
		c.Repos.Strategy,      // Repository for transactions
		c.Repos.StrategyTransaction,
		dbAdapter,
		c.Log,
	)

	// Fund Watchlist Service (globally monitored symbols)
	// Note: Using fundwatchlist repo (not fund_watchlist) - see domain/fundwatchlist
	c.Services.FundWatchlist = fundwatchlistsvc.NewService(c.Repos.FundWatchlist, c.Log)

	c.Services.Session = domainsession.NewService(c.Repos.Session)
	c.Services.ADKSession = adk.NewSessionService(c.Services.Session)

	// Position management service for WebSocket updates
	c.Services.PositionManagement = positionservice.NewService(c.Repos.Position, c.Log)

	// Exchange service with notification publisher for Telegram
	notificationPublisher := events.NewNotificationPublisher(c.Adapters.KafkaProducer)
	c.Services.Exchange = exchange.NewService(
		c.Repos.ExchangeAccount,
		c.Adapters.Encryptor,
		nil, // limitsChecker - can be nil, validation moved to strategy service
		notificationPublisher,
		c.Log,
	)

	// Consumer-facing services (Clean Architecture: abstraction over repositories)
	c.Services.MarketData = marketdatasvc.NewService(c.Repos.MarketData, c.Log)
	c.Services.AIUsage = aiusagesvc.NewService(c.Repos.AIUsage, c.Log)

	// NOTE: Onboarding orchestrator moved to Phase 6 (depends on WorkflowFactory)

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
		c.Services.Order, // Inject order service for DI
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
		c.Repos.Stats.(*chrepo.StatsRepository), // Type assertion
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

	// Onboarding orchestrator (depends on WorkflowFactory, so initialized here)
	onboardingNotificationPublisher := events.NewNotificationPublisher(c.Adapters.KafkaProducer)
	c.Services.Onboarding = onboardingservice.NewService(
		c.Business.WorkflowFactory,
		c.Services.ADKSession,
		templates.Get(),
		c.Repos.User,
		c.Repos.ExchangeAccount,
		c.Services.Strategy,
		onboardingNotificationPublisher,
		c.Log,
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

	// Telegram bot and handlers (uses new framework)
	bot, telegramHandler := provideTelegramBot(
		c.Config,
		c.Services.User,
		c.Repos.User,
		c.Repos.Position,
		c.Repos.ExchangeAccount,
		c.Redis,
		c.Adapters.Encryptor,
		c.Adapters.ExchangeFactory,
		c.Adapters.KafkaProducer,
		c.Adapters.TelegramNotificationConsumer,
		c.Services.Exchange,
		c.Services.Onboarding,
		c.Repos.LimitProfile,
		c.Repos.Strategy,
		c.Log,
	)
	c.Application.TelegramBot = bot
	c.Application.TelegramHandler = telegramHandler

	// Telegram notification service (uses telegram embedded templates for notifications)
	// Note: Notification templates are in pkg/templates/notifications/, not pkg/telegram/templates/
	// So we use main templates registry here, not telegram's embedded one
	templateAdapter := telegram.NewTemplateRendererAdapter(templates.Get())
	notificationService := provideTelegramNotificationService(bot, templateAdapter, c.Log)
	c.Application.TelegramNotificationService = notificationService

	// HTTP server (with GraphQL)
	c.Application.HTTPServer = provideHTTPServer(
		c.Config,
		c.Application.HealthHandler,
		c.Application.TelegramBot,
		c.Application.TelegramHandler,
		c.Services.User,
		c.Services.Strategy,
		c.Services.FundWatchlist,
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
		c.Repos.FundWatchlist,
		c.Repos.Strategy,
		c.Repos.Order,
		c.Repos.Position,
		c.Repos.MarketData,
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

	// Telegram Notification Consumer (uses new framework)
	c.Background.TelegramNotificationSvc = consumers.NewTelegramNotificationConsumer(
		c.Adapters.TelegramNotificationConsumer,
		c.Application.TelegramNotificationService,
		c.Services.DomainUser, // Domain service for getting user chat IDs
		c.Log,
	)

	c.Background.RiskSvc = consumers.NewRiskConsumer(
		c.Adapters.RiskConsumer,
		c.Business.RiskEngine,
		c.Log,
	)

	// Opportunity Consumer - uses Strategy repo for target_allocations filtering
	c.Background.OpportunitySvc = consumers.NewOpportunityConsumer(
		c.Adapters.OpportunityConsumer,
		c.Repos.Strategy, // Will check target_allocations
		c.Services.DomainUser,
		c.Business.WorkflowFactory,
		c.Services.ADKSession,
		c.Config.Workers.MarketScannerMaxConcurrency,
		c.Log,
	)

	// System events consumer (portfolio initialization, worker failures)
	c.Background.SystemEventsSvc = consumers.NewSystemEventsConsumer(
		c.Adapters.SystemEventsConsumer,
		c.Services.Onboarding, // Use from container
		c.Application.TelegramBot,
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
	orderService *order.Service,
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
		OrderService:        orderService,
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
	statsRepo *chrepo.StatsRepository,
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
			StatsRepo:      statsRepo, // stats.Repository interface
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
	telegramBot tg.Bot,
	telegramHandler *telegram.Handler,
	userService *userservice.Service,
	strategyService *strategyservice.Service,
	fundWatchlistService *fundwatchlistsvc.Service,
	log *logger.Logger,
) *api.Server {
	var webhookHandler *tg.WebhookHandler
	if cfg.Telegram.WebhookURL != "" {
		// Create webhook handler using pkg/telegram framework
		// Pass the handler's HandleUpdate method as the update processor
		webhookHandler = tgapi.NewWebhookHandler(telegramHandler.HandleUpdate, log)
		log.Infow("✓ Telegram webhook mode enabled", "url", cfg.Telegram.WebhookURL)
	} else {
		log.Info("✓ Telegram polling mode enabled")
	}

	// GraphQL handlers (only in development for now)
	var graphqlHandler, playgroundHandler http.Handler
	if cfg.App.Env != "production" {
		graphqlHandler = graphql.Handler(userService, strategyService, fundWatchlistService)
		playgroundHandler = graphql.PlaygroundHandler()
		log.Info("✓ GraphQL API enabled (development mode)")
	}

	return api.NewServer(api.ServerConfig{
		Port:              cfg.HTTP.Port,
		ServiceName:       cfg.App.Name,
		Version:           cfg.App.Version,
		TelegramWebhook:   webhookHandler,
		GraphQLHandler:    graphqlHandler,
		PlaygroundHandler: playgroundHandler,
	}, healthHandler, log)
}

func provideTelegramBot(
	cfg *config.Config,
	userService *userservice.Service,
	userRepo user.Repository,
	positionRepo position.Repository,
	exchAcctRepo exchange_account.Repository,
	redisClient *redisclient.Client,
	encryptor *crypto.Encryptor,
	exchFactory exchanges.Factory,
	kafkaProducer *kafka.Producer,
	notificationConsumer *kafka.Consumer,
	exchangeService *exchange.Service,
	onboardingSvc *onboardingservice.Service,
	limitProfileRepo limit_profile.Repository,
	strategyRepo strategyDomain.Repository,
	log *logger.Logger,
) (tg.Bot, *telegram.Handler) {
	log.Info("Initializing Telegram bot...")

	// Create bot using tgbotapi adapter (only place that knows about tgbotapi!)
	webhookMode := cfg.Telegram.WebhookURL != ""
	bot, err := tgbotapi.NewBot(tgbotapi.Config{
		Token:       cfg.Telegram.BotToken,
		Debug:       cfg.Telegram.Debug,
		Timeout:     60,
		BufferSize:  100,
		WebhookMode: webhookMode,
	}, log)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	// TODO: Migrate notification service to new framework
	// notificationService := telegram.NewNotificationService(bot, templates.Get(), log)

	// TODO: Migrate onboarding to new framework
	// TODO: Migrate exchange setup to new framework

	// Menu session repository and service (Clean Architecture)
	menuSessionRepo := redis.NewMenuSessionRepository(redisClient.Client())
	menuSessionService := menu_session.NewService(menuSessionRepo, log)

	// Create adapters for pkg/telegram framework
	sessionAdapter := telegram.NewSessionServiceAdapter(menuSessionService)

	// Use telegram's embedded template registry for menus (has invest/, common/, etc.)
	telegramTemplates, err := tg.NewDefaultTemplateRegistry()
	if err != nil {
		log.Fatalf("Failed to create telegram template registry: %v", err)
	}

	// Menu navigator for interactive menus (uses telegram templates from pkg/telegram/templates/)
	menuNavigator := tg.NewMenuNavigator(
		sessionAdapter,
		bot,
		telegramTemplates, // Direct use of telegram.TemplateRegistry (implements TemplateRenderer)
		log,
		30*time.Minute, // Session TTL
	)

	// Job publisher for async workflows
	jobPublisher := events.NewWorkerPublisher(kafkaProducer)

	// User service adapter for telegram (uses application service for side effects)
	userServiceAdapter := telegram.NewUserServiceAdapter(userService)

	// Investment validator (checks tier limits + user risk settings)
	investmentValidator := strategyservice.NewInvestmentValidator(
		limitProfileRepo,
		strategyRepo,
		log,
	)

	// Invest menu service (async portfolio creation via Kafka)
	investMenuService := telegram.NewInvestMenuService(
		menuNavigator,
		exchangeService,     // Exchange service
		userServiceAdapter,  // User service adapter
		jobPublisher,        // Job publisher for async portfolio creation
		investmentValidator, // Validates against tier limits + user settings
		log,
	)

	// Exchanges menu service (manage exchange accounts)
	exchangesMenuService := telegram.NewExchangesMenuService(
		menuNavigator,
		exchangeService,    // Exchange service
		userServiceAdapter, // User service adapter
		log,
	)

	// Menu registry for auto-routing (uses framework)
	menuRegistry := tg.NewMenuRegistry(log, sessionAdapter)
	menuRegistry.Register(investMenuService)
	menuRegistry.Register(exchangesMenuService)
	// TODO: Register other menu handlers (settings, etc.)

	// Command registry (uses framework)
	commandRegistry := tg.NewCommandRegistry(bot, log)
	commandRegistry.Use(tg.LoggingMiddleware(log))
	commandRegistry.Use(tg.RecoveryMiddleware(log))

	// Register commands
	registerTelegramCommands(commandRegistry, investMenuService, exchangesMenuService, userServiceAdapter, log)

	// Create handler (uses framework)
	handler := telegram.NewHandler(
		bot,
		commandRegistry,
		menuRegistry,
		userServiceAdapter,
		log,
	)

	// Configure webhook if needed
	if cfg.Telegram.WebhookURL != "" {
		log.Infow("Configuring Telegram webhook...", "url", cfg.Telegram.WebhookURL)

		// Cast to concrete type to call SetWebhook (bot is already *tgbotapi.Bot)
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

	// Set handler for updates
	bot.SetHandler(handler.HandleUpdate)

	log.Info("✓ Telegram bot initialized")
	return bot, handler
}

// provideTelegramNotificationService creates notification service for Telegram
func provideTelegramNotificationService(
	bot tg.Bot,
	templates tg.TemplateRenderer,
	log *logger.Logger,
) *telegram.NotificationService {
	return telegram.NewNotificationService(bot, templates, log)
}

// registerTelegramCommands registers all bot commands with the command registry
func registerTelegramCommands(
	registry *tg.CommandRegistry,
	investMenu *telegram.InvestMenuService,
	exchangesMenu *telegram.ExchangesMenuService,
	userService telegram.UserService,
	log *logger.Logger,
) {
	// /start - Welcome message
	registry.Register(tg.CommandConfig{
		Name:        "start",
		Description: "Start the bot and see welcome message",
		Category:    "General",
		Handler: func(ctx *tg.CommandContext) error {
			welcomeText := "👋 Welcome to Prometheus Trading!\n\n" +
				"Your AI-powered hedge fund running 24/7.\n\n" +
				"🚀 Quick Start:\n" +
				"/invest - Start investing\n" +
				"/status - Check portfolio status\n" +
				"/help - Show all commands\n\n" +
				"Let's make some money! 💰"
			return ctx.Bot.SendMessage(ctx.ChatID, welcomeText)
		},
	})

	// /help - Show available commands
	registry.Register(tg.CommandConfig{
		Name:        "help",
		Aliases:     []string{"h"},
		Description: "Show available commands",
		Category:    "General",
		Handler: func(ctx *tg.CommandContext) error {
			commandsByCategory := registry.GetCommandsByCategory(false)

			helpText := "📋 *Available Commands*\n\n"
			for category, commands := range commandsByCategory {
				helpText += fmt.Sprintf("*%s*\n", category)
				for _, cmd := range commands {
					helpText += fmt.Sprintf("/%s", cmd.Name)
					if len(cmd.Aliases) > 0 {
						helpText += fmt.Sprintf(" (/%s)", strings.Join(cmd.Aliases, ", /"))
					}
					helpText += fmt.Sprintf(" - %s\n", cmd.Description)
				}
				helpText += "\n"
			}

			_, err := ctx.Bot.SendMessageWithOptions(ctx.ChatID, helpText, tg.MessageOptions{
				ParseMode: "Markdown",
			})
			return err
		},
	})

	// /invest - Start investment flow
	registry.Register(tg.CommandConfig{
		Name:        "invest",
		Aliases:     []string{"i"},
		Description: "Start new investment",
		Usage:       "/invest",
		Category:    "Trading",
		Handler: func(ctx *tg.CommandContext) error {
			// Invest command doesn't accept arguments
			if ctx.Args != "" {
				return ctx.Bot.SendMessage(ctx.ChatID, "❌ `/invest` command doesn't take arguments.\n\nJust use: `/invest`")
			}

			usr := ctx.User.(*user.User)
			log.Infow("Starting invest flow",
				"user_id", usr.ID,
				"telegram_id", ctx.TelegramID,
			)

			if err := investMenu.StartInvest(ctx.Ctx, usr.ID, ctx.TelegramID); err != nil {
				log.Errorw("Failed to start invest menu", "error", err)
				return ctx.Bot.SendMessage(ctx.ChatID, "❌ Failed to start investment flow. Please try again.")
			}

			return nil
		},
	})

	// /exchanges - Manage exchange accounts
	registry.Register(tg.CommandConfig{
		Name:        "exchanges",
		Aliases:     []string{"ex"},
		Description: "Manage exchange accounts",
		Usage:       "/exchanges",
		Category:    "Settings",
		Handler: func(ctx *tg.CommandContext) error {
			// Exchanges command doesn't accept arguments
			if ctx.Args != "" {
				return ctx.Bot.SendMessage(ctx.ChatID, "❌ `/exchanges` command doesn't take arguments.\n\nJust use: `/exchanges`")
			}

			usr := ctx.User.(*user.User)
			log.Infow("Starting exchanges management flow",
				"user_id", usr.ID,
				"telegram_id", ctx.TelegramID,
			)

			if err := exchangesMenu.StartExchanges(ctx.Ctx, usr.ID, ctx.TelegramID); err != nil {
				log.Errorw("Failed to start exchanges menu", "error", err)
				return ctx.Bot.SendMessage(ctx.ChatID, "❌ Failed to start exchanges management. Please try again.")
			}

			return nil
		},
	})

	// /status - Show portfolio status
	registry.Register(tg.CommandConfig{
		Name:        "status",
		Aliases:     []string{"s", "portfolio"},
		Description: "View your portfolio status",
		Category:    "Portfolio",
		Handler: func(ctx *tg.CommandContext) error {
			usr := ctx.User.(*user.User)
			// TODO: Implement portfolio status
			statusText := fmt.Sprintf("📊 *Portfolio Status*\n\n"+
				"User: %s\n"+
				"Status: Active\n\n"+
				"_Full portfolio view coming soon..._",
				usr.FirstName,
			)
			_, err := ctx.Bot.SendMessageWithOptions(ctx.ChatID, statusText, tg.MessageOptions{
				ParseMode: "Markdown",
			})
			return err
		},
	})

	// /cancel - Cancel current operation
	registry.Register(tg.CommandConfig{
		Name:        "cancel",
		Description: "Cancel current operation",
		Category:    "General",
		Handler: func(ctx *tg.CommandContext) error {
			// End any active menu session
			if err := investMenu.EndMenu(ctx.Ctx, ctx.TelegramID); err != nil {
				log.Debugw("No active menu to cancel", "telegram_id", ctx.TelegramID)
			}

			return ctx.Bot.SendMessage(ctx.ChatID, "✅ Operation cancelled. Use /help to see available commands.")
		},
	})

	log.Info("✓ Telegram commands registered")
}

// provideWorkers is defined in workers.go due to its size
