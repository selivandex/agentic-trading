package bootstrap

import (
	"context"
	"fmt"
	"time"

	"prometheus/internal/adapters/websocket"
	binancews "prometheus/internal/adapters/websocket/binance"
	"prometheus/internal/consumers"
	"prometheus/internal/events"
)

// WebSocketClients groups WebSocket clients for different exchanges
type WebSocketClients struct {
	Binance websocket.Client
	Bybit   websocket.Client
	OKX     websocket.Client
}

// MarketDataManager wraps the market data websocket manager
type MarketDataManager struct {
	Manager *websocket.MarketDataManager
}

// MustInitWebSocketClients initializes WebSocket clients for all configured exchanges
func (c *Container) MustInitWebSocketClients() {
	if !c.Config.WebSocket.Enabled {
		c.Log.Info("WebSocket connections disabled by config")
		return
	}

	c.Log.Infow("Initializing WebSocket clients",
		"exchanges", c.Config.WebSocket.GetExchanges(),
		"symbols", c.Config.WebSocket.GetSymbols(),
		"intervals", c.Config.WebSocket.GetIntervals(),
	)

	// Create WebSocket publisher for all WebSocket events
	wsPublisher := events.NewWebSocketPublisher(c.Adapters.KafkaProducer)
	wsHandler := websocket.NewKafkaEventHandler(wsPublisher, c.Log)

	// Store publisher in container for graceful shutdown
	c.Adapters.WebSocketPublisher = wsPublisher

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
			c.Log.Infow("✓ Binance WebSocket client created",
				"has_credentials", apiKey != "",
			)

		// case "bybit":
		// 	// Get API credentials for Bybit (optional for public streams)
		// 	apiKey := c.Config.MarketData.Bybit.APIKey
		// 	secretKey := c.Config.MarketData.Bybit.Secret

		// 	if apiKey == "" {
		// 		c.Log.Warn("Bybit WebSocket API key not configured - only public streams will be available")
		// 	}

		// 	client := bybitws.NewClient(
		// 		"bybit",
		// 		wsHandler,
		// 		apiKey,
		// 		secretKey,
		// 		c.Config.WebSocket.UseTestnet,
		// 		c.Log,
		// 	)
		// 	clients.Bybit = client
		// 	c.Log.Info("✓ Bybit WebSocket client created",
		// 		"has_credentials", apiKey != "",
		// 	)

		// case "okx":
		// 	// Get API credentials for OKX (optional for public streams)
		// 	apiKey := c.Config.MarketData.OKX.APIKey
		// 	secretKey := c.Config.MarketData.OKX.Secret

		// 	if apiKey == "" {
		// 		c.Log.Warn("OKX WebSocket API key not configured - only public streams will be available")
		// 	}

		// 	client := okxws.NewClient(
		// 		"okx",
		// 		wsHandler,
		// 		apiKey,
		// 		secretKey,
		// 		c.Config.WebSocket.UseTestnet,
		// 		c.Log,
		// 	)
		// 	clients.OKX = client
		// 	c.Log.Info("✓ OKX WebSocket client created",
		// 		"has_credentials", apiKey != "",
		// 	)

		default:
			c.Log.Warn("Unsupported exchange for WebSocket", "exchange", exchange)
		}
	}

	// Store clients in container
	if c.Adapters.WebSocketClients == nil {
		c.Adapters.WebSocketClients = clients
	}

	// Create unified WebSocket consumer (handles market data + user data events)
	c.Log.Infow("Creating unified WebSocket consumer",
		"topic", events.TopicWebSocketEvents,
		"market_data_types", []string{"kline", "ticker", "depth", "trade", "mark_price", "liquidation"},
		"user_data_types", []string{"order", "position", "balance", "margin_call", "account_config"},
		"group_id", c.Config.Kafka.GroupID,
	)
	c.Background.WebSocketSvc = consumers.NewWebSocketConsumer(
		c.Adapters.WebSocketConsumer,
		c.Services.MarketData,
		c.Services.PositionManagement,
		c.Log,
	)

	c.Log.Infow("✓ WebSocket infrastructure initialized",
		"consumer", "unified",
		"single_topic", events.TopicWebSocketEvents,
		"handles_market_and_user_data", true,
	)
}

// MustInitMarketDataManager initializes Market Data WebSocket manager with health monitoring
func (c *Container) MustInitMarketDataManager() {
	if !c.Config.WebSocket.Enabled {
		c.Log.Info("Market Data WebSocket manager disabled (WebSocket connections disabled)")
		return
	}

	if c.Adapters.WebSocketClients == nil || c.Adapters.WebSocketClients.Binance == nil {
		c.Log.Warn("Market Data WebSocket manager disabled (no clients initialized)")
		return
	}

	c.Log.Info("Initializing Market Data WebSocket Manager...")

	// Build stream configuration
	streamConfig := c.buildStreamConfig()

	// Create Market Data Manager for Binance client
	managerConfig := websocket.MarketDataManagerConfig{
		HealthCheckInterval:    60 * time.Second, // Check connection health every 60 seconds
		MaxConsecutiveFailures: 3,                // Log warning after 3 consecutive failures
	}

	manager := websocket.NewMarketDataManager(
		c.Adapters.WebSocketClients.Binance,
		streamConfig,
		managerConfig,
		c.Log,
	)

	c.Adapters.MarketDataManager = &MarketDataManager{
		Manager: manager,
	}

	c.Log.Info("✓ Market Data WebSocket Manager initialized")
}

// connectWebSocketClients is deprecated - Market Data connections are now managed by MarketDataManager
// This function is kept for backward compatibility but does nothing
func (c *Container) connectWebSocketClients(ctx context.Context) error {
	// NOTE: WebSocket connections are now managed by MarketDataManager with auto-reconnect
	// See MustInitMarketDataManager() and Start() in container.go
	c.Log.Debug("WebSocket connections now managed by MarketDataManager")
	return nil
}

// buildStreamConfig builds WebSocket stream configuration from config
func (c *Container) buildStreamConfig() websocket.ConnectionConfig {
	symbols := c.Config.WebSocket.GetSymbols()
	intervals := c.Config.WebSocket.GetIntervals()
	streamTypes := c.Config.WebSocket.GetStreamTypes()
	marketTypes := c.Config.WebSocket.GetMarketTypes()

	var streams []websocket.StreamConfig

	// Build streams based on configured types and market types
	for _, streamType := range streamTypes {
		for _, marketTypeStr := range marketTypes {
			marketType := websocket.MarketType(marketTypeStr)

			switch streamType {
			case "kline":
				// Create kline streams for all symbol-interval combinations
				for _, symbol := range symbols {
					for _, interval := range intervals {
						streams = append(streams, websocket.StreamConfig{
							Type:       websocket.StreamTypeKline,
							Symbol:     symbol,
							Interval:   websocket.Interval(interval),
							MarketType: marketType,
						})
					}
				}

			case "markPrice":
				// Create mark price streams for all symbols (futures only!)
				if marketType == websocket.MarketTypeFutures {
					for _, symbol := range symbols {
						streams = append(streams, websocket.StreamConfig{
							Type:       websocket.StreamTypeMarkPrice,
							Symbol:     symbol,
							MarketType: marketType,
						})
					}
				} else {
					c.Log.Debug("Skipping mark price for spot (not supported)",
						"market_type", marketType,
					)
				}

			case "ticker":
				// Create ticker streams for all symbols
				for _, symbol := range symbols {
					streams = append(streams, websocket.StreamConfig{
						Type:       websocket.StreamTypeTicker,
						Symbol:     symbol,
						MarketType: marketType,
					})
				}

			case "trade":
				// Create trade streams for all symbols
				for _, symbol := range symbols {
					streams = append(streams, websocket.StreamConfig{
						Type:       websocket.StreamTypeTrade,
						Symbol:     symbol,
						MarketType: marketType,
					})
				}

			case "depth":
				// Create depth streams for all symbols
				for _, symbol := range symbols {
					streams = append(streams, websocket.StreamConfig{
						Type:       websocket.StreamTypeDepth,
						Symbol:     symbol,
						MarketType: marketType,
					})
				}

			case "liquidation":
				// Create liquidation streams for all symbols (futures only!)
				if marketType == websocket.MarketTypeFutures {
					for _, symbol := range symbols {
						streams = append(streams, websocket.StreamConfig{
							Type:       websocket.StreamTypeLiquidation,
							Symbol:     symbol,
							MarketType: marketType,
						})
					}
				} else {
					c.Log.Debug("Skipping liquidation for spot (not supported)",
						"market_type", marketType,
					)
				}

			default:
				c.Log.Warn("Unknown stream type in config", "type", streamType)
			}
		}
	}

	c.Log.Infow("Built stream configuration",
		"total_streams", len(streams),
		"stream_types", streamTypes,
		"market_types", marketTypes,
		"symbols", len(symbols),
		"intervals", len(intervals),
	)

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

	// // Stop Bybit client
	// if c.Adapters.WebSocketClients.Bybit != nil {
	// 	if err := c.Adapters.WebSocketClients.Bybit.Stop(ctx); err != nil {
	// 		c.Log.Error("Failed to stop Bybit WebSocket", "error", err)
	// 		lastErr = err
	// 	}
	// }

	// // Stop OKX client
	// if c.Adapters.WebSocketClients.OKX != nil {
	// 	if err := c.Adapters.WebSocketClients.OKX.Stop(ctx); err != nil {
	// 		c.Log.Error("Failed to stop OKX WebSocket", "error", err)
	// 		lastErr = err
	// 	}
	// }

	if lastErr != nil {
		return fmt.Errorf("errors occurred during WebSocket shutdown: %w", lastErr)
	}

	c.Log.Info("✓ All WebSocket clients stopped")
	return nil
}
