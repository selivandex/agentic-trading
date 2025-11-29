package bootstrap

import (
	"prometheus/internal/adapters/websocket"
	"prometheus/internal/consumers"
	"prometheus/internal/events"
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

	// Create User Data Stream Factory
	factory := websocket.NewUserDataStreamFactory(c.Log)

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

