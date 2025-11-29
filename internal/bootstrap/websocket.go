package bootstrap

import (
	"context"
	"fmt"

	"prometheus/internal/adapters/websocket"
	binancews "prometheus/internal/adapters/websocket/binance"
	"prometheus/internal/consumers"
	"prometheus/internal/events"
	"prometheus/pkg/errors"
)

// WebSocketClients groups WebSocket clients for different exchanges
type WebSocketClients struct {
	Binance websocket.Client
	// TODO: Add other exchanges (Bybit, OKX, etc.)
}

// MustInitWebSocketClients initializes WebSocket clients for all configured exchanges
func (c *Container) MustInitWebSocketClients() {
	if !c.Config.WebSocket.Enabled {
		c.Log.Info("WebSocket connections disabled by config")
		return
	}

	c.Log.Info("Initializing WebSocket clients",
		"exchanges", c.Config.WebSocket.GetExchanges(),
		"symbols", c.Config.WebSocket.GetSymbols(),
		"intervals", c.Config.WebSocket.GetIntervals(),
	)

	// Create WebSocket publisher for all WebSocket events
	wsPublisher := events.NewWebSocketPublisher(c.Adapters.KafkaProducer)
	wsHandler := websocket.NewKafkaEventHandler(wsPublisher, c.Log)

	// Initialize clients for each exchange
	clients := &WebSocketClients{}

	for _, exchange := range c.Config.WebSocket.GetExchanges() {
		switch exchange {
		case "binance":
			// Get API credentials for private streams (optional)
			// For public market data streams (klines, ticker, depth) credentials are not needed
			apiKey := c.Config.MarketData.Binance.APIKey
			secretKey := c.Config.MarketData.Binance.Secret

			if apiKey == "" {
				c.Log.Warn("Binance WebSocket API key not configured - only public streams will be available")
			}

			client := binancews.NewClient(
				"binance",
				wsHandler,
				apiKey,
				secretKey,
				c.Config.WebSocket.UseTestnet,
				c.Log,
			)
			clients.Binance = client
			c.Log.Info("✓ Binance WebSocket client created",
				"has_credentials", apiKey != "",
			)

		default:
			c.Log.Warn("Unsupported exchange for WebSocket", "exchange", exchange)
		}
	}

	// Store clients in container
	if c.Adapters.WebSocketClients == nil {
		c.Adapters.WebSocketClients = clients
	}

	// Create WebSocket consumer for kline events
	c.Log.Info("Creating WebSocket kline consumer",
		"topic", events.TopicWebSocketKline,
		"group_id", c.Config.Kafka.GroupID,
	)
	
	klineConsumer := consumers.NewWebSocketKlineConsumer(
		c.Adapters.WebSocketKlineConsumer,
		c.Repos.MarketData,
		c.Log,
	)
	c.Background.WebSocketKlineSvc = klineConsumer

	c.Log.Info("✓ WebSocket infrastructure initialized",
		"consumer_created", c.Background.WebSocketKlineSvc != nil,
	)
}

// connectWebSocketClients establishes connections for all WebSocket clients
func (c *Container) connectWebSocketClients(ctx context.Context) error {
	if !c.Config.WebSocket.Enabled || c.Adapters.WebSocketClients == nil {
		return nil
	}

	c.Log.Info("Connecting WebSocket clients...")

	// Build stream configuration
	streamConfig := c.buildStreamConfig()

	// Connect Binance client
	if c.Adapters.WebSocketClients.Binance != nil {
		if err := c.Adapters.WebSocketClients.Binance.Connect(ctx, streamConfig); err != nil {
			return errors.Wrap(err, "connect binance websocket")
		}

		if err := c.Adapters.WebSocketClients.Binance.Start(ctx); err != nil {
			return errors.Wrap(err, "start binance websocket")
		}

		c.Log.Info("✓ Binance WebSocket connected")
	}

	// TODO: Connect other exchange clients

	return nil
}

// buildStreamConfig builds WebSocket stream configuration from config
func (c *Container) buildStreamConfig() websocket.ConnectionConfig {
	symbols := c.Config.WebSocket.GetSymbols()
	intervals := c.Config.WebSocket.GetIntervals()

	var streams []websocket.StreamConfig

	// Create kline streams for all symbol-interval combinations
	for _, symbol := range symbols {
		for _, interval := range intervals {
			streams = append(streams, websocket.StreamConfig{
				Type:     websocket.StreamTypeKline,
				Symbol:   symbol,
				Interval: websocket.Interval(interval), // Convert string to Interval type
			})
		}
	}

	return websocket.ConnectionConfig{
		Streams:          streams,
		ReconnectBackoff: c.Config.WebSocket.ReconnectBackoff,
		MaxReconnects:    c.Config.WebSocket.MaxReconnects,
		PingInterval:     c.Config.WebSocket.PingInterval,
		ReadBufferSize:   c.Config.WebSocket.ReadBufferSize,
		WriteBufferSize:  c.Config.WebSocket.WriteBufferSize,
	}
}

// stopWebSocketClients gracefully stops all WebSocket clients
func (c *Container) stopWebSocketClients(ctx context.Context) error {
	if !c.Config.WebSocket.Enabled || c.Adapters.WebSocketClients == nil {
		return nil
	}

	c.Log.Info("Stopping WebSocket clients...")

	var lastErr error

	// Stop Binance client
	if c.Adapters.WebSocketClients.Binance != nil {
		if err := c.Adapters.WebSocketClients.Binance.Stop(ctx); err != nil {
			c.Log.Error("Failed to stop Binance WebSocket", "error", err)
			lastErr = err
		}
	}

	// TODO: Stop other exchange clients

	if lastErr != nil {
		return fmt.Errorf("errors occurred during WebSocket shutdown: %w", lastErr)
	}

	c.Log.Info("✓ All WebSocket clients stopped")
	return nil
}
