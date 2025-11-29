package bootstrap

import (
	"fmt"

	"github.com/google/uuid"

	"prometheus/internal/adapters/websocket"
	"prometheus/internal/adapters/websocket/binance"
	"prometheus/internal/consumers"
	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/events"
	"prometheus/pkg/logger"
)

// UserDataManager wraps the websocket user data manager
type UserDataManager struct {
	Manager *websocket.UserDataManager
}

// MustInitUserDataManager initializes User Data WebSocket manager
func (c *Container) MustInitUserDataManager() {
	if !c.Config.WebSocket.Enabled {
		c.Log.Info("User Data WebSocket manager disabled (WebSocket connections disabled)")
		return
	}

	c.Log.Info("Initializing User Data WebSocket Manager...")

	// Create User Data publisher
	userDataPublisher := events.NewUserDataPublisher(c.Adapters.KafkaProducer, c.Log)

	// Create Kafka handler for User Data events (WebSocket → Kafka)
	kafkaHandler := websocket.NewKafkaUserDataHandler(userDataPublisher, c.Log)

	// Create inline factory (to avoid import cycle)
	factory := &inlineUserDataFactory{log: c.Log}

	// Create User Data Manager
	config := websocket.UserDataManagerConfig{
		HealthCheckInterval: 60, // 60 seconds
		ListenKeyRenewal:    30, // 30 minutes
	}

	manager := websocket.NewUserDataManager(
		c.Repos.ExchangeAccount,
		factory,
		kafkaHandler,
		c.Adapters.Encryptor,
		config,
		c.Log,
	)

	c.Adapters.UserDataManager = &UserDataManager{
		Manager: manager,
	}

	c.Log.Info("✓ User Data WebSocket Manager initialized")
}

// inlineUserDataFactory creates exchange-specific User Data clients
type inlineUserDataFactory struct {
	log *logger.Logger
}

func (f *inlineUserDataFactory) Create(
	exchange exchange_account.ExchangeType,
	accountID uuid.UUID,
	userID uuid.UUID,
	handler websocket.UserDataEventHandler,
	useTestnet bool,
) (websocket.UserDataStreamer, error) {
	switch exchange {
	case exchange_account.ExchangeBinance:
		return binance.NewUserDataClient(
			accountID,
			userID,
			string(exchange),
			handler,
			useTestnet,
			f.log,
		), nil

	case exchange_account.ExchangeBybit:
		return nil, fmt.Errorf("bybit user data websocket not yet implemented")

	case exchange_account.ExchangeOKX:
		return nil, fmt.Errorf("okx user data websocket not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported exchange for user data websocket: %s", exchange)
	}
}

// provideUserDataConsumer creates a unified User Data consumer service
func provideUserDataConsumer(c *Container) *consumers.UserDataConsumer {
	// Create unified Kafka consumer for all User Data topics
	// The consumer will automatically handle multiple topics internally
	consumer := provideKafkaConsumer(c.Config, "user-data-all-topics", c.Log)

	return consumers.NewUserDataConsumer(
		consumer,
		c.Repos.Order,
		c.Repos.Position,
		c.Log,
	)
}

// MustInitUserDataConsumer initializes the User Data event consumer
func (c *Container) MustInitUserDataConsumer() {
	if !c.Config.WebSocket.Enabled {
		c.Log.Info("User Data consumer disabled (WebSocket connections disabled)")
		return
	}

	c.Log.Info("Initializing User Data Consumer...")

	c.Background.UserDataSvc = provideUserDataConsumer(c)

	c.Log.Info("✓ User Data Consumer initialized")
}
