package bootstrap

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/adapters/websocket"
	"prometheus/internal/adapters/websocket/binance"
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

	// Create unified WebSocket publisher (handles both market data + user data)
	wsPublisher := events.NewWebSocketPublisher(c.Adapters.KafkaProducer)

	// Create Kafka handler for User Data events (WebSocket → Kafka)
	kafkaHandler := websocket.NewKafkaUserDataHandler(wsPublisher, c.Log)

	// Create inline factory (to avoid import cycle)
	factory := &inlineUserDataFactory{log: c.Log}

	// Create User Data Manager
	config := websocket.UserDataManagerConfig{
		HealthCheckInterval:    60 * time.Second, // Check connection health every 60 seconds
		ListenKeyRenewal:       30 * time.Minute, // Renew listenKeys every 30 minutes
		ReconciliationInterval: 5 * time.Minute,  // Sync with DB every 5 minutes (hot reload)
	}

	manager := websocket.NewUserDataManager(
		c.Repos.ExchangeAccount,
		c.Services.Exchange, // Exchange service for deactivation + notifications
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

// NOTE: User Data events are now consumed by the unified WebSocketConsumer
// which handles both market data (TopicWebSocketEvents) and user data (TopicUserDataEvents)
// See websocket.go for WebSocket consumer initialization
