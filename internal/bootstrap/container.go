package bootstrap

import (
	"context"
	"sync"

	"google.golang.org/adk/session"

	chclient "prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/embeddings"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/kafka"
	pgclient "prometheus/internal/adapters/postgres"
	redisclient "prometheus/internal/adapters/redis"
	telegram "prometheus/internal/adapters/telegram"
	"prometheus/internal/agents"
	"prometheus/internal/agents/workflows"
	"prometheus/internal/api"
	"prometheus/internal/api/health"
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
	chrepo "prometheus/internal/repository/clickhouse"
	pgrepo "prometheus/internal/repository/postgres"
	exchangeservice "prometheus/internal/services/exchange"
	riskservice "prometheus/internal/services/risk"
	"prometheus/internal/tools"
	"prometheus/internal/workers"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
	"prometheus/pkg/templates"
)

// Container holds all application dependencies and their lifecycle
// Components are organized in initialization order
type Container struct {
	// Core configuration & logging
	Config       *config.Config
	Log          *logger.Logger
	ErrorTracker errors.Tracker

	// Infrastructure Layer (Data stores)
	PG    *pgclient.Client
	CH    *chclient.Client
	Redis *redisclient.Client

	// Domain Layer - Repositories
	Repos *Repositories

	// Domain Layer - Services
	Services *Services

	// External Adapters
	Adapters *Adapters

	// Business Logic
	Business *Business

	// Application Layer
	Application *Application

	// Background Processing
	Background *Background

	// Lifecycle management
	Lifecycle *Lifecycle
	WG        *sync.WaitGroup
	Context   context.Context
	Cancel    context.CancelFunc
}

// Repositories groups all domain repositories
type Repositories struct {
	User            user.Repository
	ExchangeAccount exchange_account.Repository
	TradingPair     trading_pair.Repository
	Order           order.Repository
	Position        position.Repository
	Memory          memory.Repository
	Journal         journal.Repository
	Risk            domainRisk.Repository
	Session         domainsession.Repository
	Reasoning       *pgrepo.ReasoningRepository
	MarketData      market_data.Repository
	Regime          regime.Repository
	Sentiment       sentiment.Repository
	OnChain         onchain.Repository
	Derivatives     derivatives.Repository
	Macro           macro.Repository
	AIUsage         *chrepo.AIUsageRepository
}

// Services groups all domain services
type Services struct {
	User            *user.Service
	ExchangeAccount *exchange_account.Service
	Exchange        *exchangeservice.Service // Exchange account management service
	TradingPair     *trading_pair.Service
	Order           *order.Service
	Position        *position.Service
	Memory          *memory.Service
	Journal         *journal.Service
	Session         *domainsession.Service
	ADKSession      session.Service // ADK interface
}

// Adapters groups all external adapters
type Adapters struct {
	// Kafka
	KafkaProducer                *kafka.Producer
	NotificationConsumer         *kafka.Consumer
	RiskConsumer                 *kafka.Consumer
	AnalyticsConsumer            *kafka.Consumer
	OpportunityConsumer          *kafka.Consumer
	AIUsageConsumer              *kafka.Consumer
	PositionGuardianConsumer     *kafka.Consumer
	TelegramNotificationConsumer *kafka.Consumer

	// WebSocket consumer (unified for all stream types)
	WebSocketConsumer *kafka.Consumer

	// User Data WebSocket consumers
	UserDataOrderConsumer         *kafka.Consumer
	UserDataPositionConsumer      *kafka.Consumer
	UserDataBalanceConsumer       *kafka.Consumer
	UserDataMarginCallConsumer    *kafka.Consumer
	UserDataAccountConfigConsumer *kafka.Consumer

	// Crypto & Exchanges
	Encryptor         *crypto.Encryptor
	ExchangeFactory   exchanges.Factory
	MarketDataFactory exchanges.CentralFactory

	// AI & Embeddings
	EmbeddingProvider embeddings.Provider

	// WebSocket
	WebSocketClients   *WebSocketClients
	WebSocketPublisher *events.WebSocketPublisher
	UserDataManager    *UserDataManager
	MarketDataManager  *MarketDataManager
}

// Business groups business logic components
type Business struct {
	RiskEngine      *riskservice.RiskEngine
	ToolRegistry    *tools.Registry
	AgentFactory    *agents.Factory
	AgentRegistry   *agents.Registry
	WorkflowFactory *workflows.Factory
	DefaultProvider string
	DefaultModel    string
}

// Application groups application layer components
type Application struct {
	HTTPServer      *api.Server
	HealthHandler   *health.Handler
	TelegramBot     *telegram.Bot
	TelegramHandler *telegram.Handler

	// Notification services
	TelegramNotificationSvc *consumers.TelegramNotificationConsumer
}

// Background groups all background processing components
type Background struct {
	WorkerScheduler *workers.Scheduler

	// Event consumers
	NotificationSvc     *consumers.NotificationConsumer
	RiskSvc             *consumers.RiskConsumer
	AnalyticsSvc        *consumers.AnalyticsConsumer
	OpportunitySvc      *consumers.OpportunityConsumer
	AIUsageSvc          *consumers.AIUsageConsumer
	PositionGuardianSvc *consumers.PositionGuardianConsumer

	// WebSocket consumer service (unified)
	WebSocketSvc *consumers.WebSocketConsumer

	// User Data WebSocket consumer service
	UserDataSvc *consumers.UserDataConsumer
}

// NewContainer creates a new dependency container
func NewContainer() *Container {
	ctx, cancel := context.WithCancel(context.Background())

	return &Container{
		Repos:       &Repositories{},
		Services:    &Services{},
		Adapters:    &Adapters{},
		Business:    &Business{},
		Application: &Application{},
		Background:  &Background{},
		Lifecycle:   NewLifecycle(),
		WG:          &sync.WaitGroup{},
		Context:     ctx,
		Cancel:      cancel,
	}
}

// MustInit initializes all components in the correct order
// Panics on any initialization error (fail-fast at startup)
func (c *Container) MustInit() {
	c.MustInitConfig()
	c.MustInitInfrastructure()
	c.MustInitRepositories()
	c.MustInitAdapters()
	c.MustInitServices()
	c.MustInitBusiness()
	c.MustInitApplication()
	c.MustInitBackground()
	c.MustInitWebSocketClients()
	c.MustInitMarketDataManager()
	c.MustInitUserDataManager()
	c.MustInitUserDataConsumer()
}

// Start starts all background components
func (c *Container) Start() error {
	c.Log.Info("Starting all systems...")

	// Start Market Data WebSocket Manager (with auto-reconnect)
	if c.Adapters.MarketDataManager != nil && c.Adapters.MarketDataManager.Manager != nil {
		if err := c.Adapters.MarketDataManager.Manager.Start(c.Context); err != nil {
			return errors.Wrap(err, "failed to start Market Data Manager")
		}
		c.Log.Infow("✓ Market Data Manager started",
			"connected", c.Adapters.MarketDataManager.Manager.IsConnected(),
		)
	}

	// Start User Data WebSocket Manager
	if c.Adapters.UserDataManager != nil && c.Adapters.UserDataManager.Manager != nil {
		if err := c.Adapters.UserDataManager.Manager.Start(c.Context); err != nil {
			return errors.Wrap(err, "failed to start User Data Manager")
		}
		c.Log.Infow("✓ User Data Manager started",
			"active_connections", c.Adapters.UserDataManager.Manager.GetActiveConnectionCount(),
		)
	}

	// Start background consumers
	if err := c.startConsumers(); err != nil {
		return err
	}

	// Start HTTP server
	c.WG.Add(1)
	go func() {
		defer c.WG.Done()
		if err := c.Application.HTTPServer.Start(); err != nil {
			c.Log.Errorf("HTTP server failed: %v", err)
			c.Cancel() // Trigger shutdown on fatal HTTP error
		}
	}()

	// Start workers (optional - currently commented out in original)
	// if err := c.Background.WorkerScheduler.Start(c.Context); err != nil {
	// 	return fmt.Errorf("failed to start workers: %w", err)
	// }

	c.Log.Info("✓ All systems operational")
	return nil
}

// startConsumers starts all Kafka consumers in background goroutines
func (c *Container) startConsumers() error {
	consumers := []struct {
		name string
		svc  interface{ Start(context.Context) error }
	}{
		{"notification", c.Background.NotificationSvc},
		{"risk", c.Background.RiskSvc},
		{"analytics", c.Background.AnalyticsSvc},
		{"opportunity", c.Background.OpportunitySvc},
		{"ai_usage", c.Background.AIUsageSvc},
		{"position_guardian", c.Background.PositionGuardianSvc},
		{"telegram_bot", c.Application.TelegramBot},
		{"telegram_notifications", c.Application.TelegramNotificationSvc},
	}

	// Add unified WebSocket consumer (handles all stream types)
	consumerNames := []string{"notification", "risk", "analytics", "opportunity", "ai_usage", "position_guardian", "telegram_bot", "telegram_notifications"}

	if c.Background.WebSocketSvc != nil {
		consumers = append(consumers, struct {
			name string
			svc  interface{ Start(context.Context) error }
		}{"websocket", c.Background.WebSocketSvc})
		consumerNames = append(consumerNames, "websocket")
	}

	// Add User Data consumer if enabled
	if c.Background.UserDataSvc != nil {
		consumers = append(consumers, struct {
			name string
			svc  interface{ Start(context.Context) error }
		}{"user_data", c.Background.UserDataSvc})
		consumerNames = append(consumerNames, "user_data")
	}

	c.WG.Add(len(consumers))
	for _, consumer := range consumers {
		svc := consumer.svc
		name := consumer.name
		go func() {
			defer c.WG.Done()
			if svc != nil {
				if err := svc.Start(c.Context); err != nil && c.Context.Err() == nil {
					c.Log.Error(name+" consumer failed", "error", err)
				}
			}
		}()
	}

	c.Log.Infow("✓ Event consumers started", "consumers", consumerNames)
	return nil
}

// Shutdown performs graceful shutdown in the correct order
func (c *Container) Shutdown() {
	c.Log.Info("Initiating graceful shutdown...")

	// Step 0: Stop WebSocket clients FIRST (before cancelling context)
	// This prevents new events from being published to Kafka
	c.Log.Info("[0/9] Stopping WebSocket clients...")
	shutdownCtx := context.Background()
	if err := c.stopWebSocketClients(shutdownCtx); err != nil {
		c.Log.Error("Error stopping WebSocket clients", "error", err)
	} else {
		c.Log.Info("✓ WebSocket clients stopped")
	}

	// Step 0.5: Shutdown WebSocketPublisher to prevent any remaining goroutines from publishing
	// This is critical: even if WebSocket clients timeout, no more events will reach Kafka
	if c.Adapters.WebSocketPublisher != nil {
		c.Adapters.WebSocketPublisher.Shutdown()
		c.Log.Info("✓ WebSocket publisher stopped accepting new events")
	}

	// Cancel application context to signal all other components to stop
	c.Cancel()

	// Perform coordinated cleanup with explicit order
	c.Lifecycle.Shutdown(
		c.WG,
		c.Application.HTTPServer,
		c.Background.WorkerScheduler,
		c.Adapters.MarketDataFactory,
		c.Adapters.WebSocketClients,
		c.Adapters.MarketDataManager,
		c.Adapters.UserDataManager,
		c.Adapters.KafkaProducer,
		c.Adapters.NotificationConsumer,
		c.Adapters.RiskConsumer,
		c.Adapters.AnalyticsConsumer,
		c.Adapters.OpportunityConsumer,
		c.Adapters.AIUsageConsumer,
		c.Adapters.PositionGuardianConsumer,
		c.Adapters.TelegramNotificationConsumer,
		c.Adapters.WebSocketConsumer,
		c.PG,
		c.CH,
		c.Redis,
		c.ErrorTracker,
		c.Log,
	)
}

// GetMetrics returns metrics for observability
func (c *Container) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"tools":  len(c.Business.ToolRegistry.List()),
		"agents": len(c.Business.AgentRegistry.List()),
	}
}

// TemplateRegistry returns the global template registry
func (c *Container) TemplateRegistry() *templates.Registry {
	return templates.Get()
}
