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

	// Create WebSocket consumers for all stream types
	streamTypes := c.Config.WebSocket.GetStreamTypes()

	for _, streamType := range streamTypes {
		switch streamType {
		case "kline":
			c.Log.Info("Creating WebSocket kline consumer",
				"topic", events.TopicWebSocketKline,
				"group_id", c.Config.Kafka.GroupID,
			)
			c.Background.WebSocketKlineSvc = consumers.NewWebSocketKlineConsumer(
				c.Adapters.WebSocketKlineConsumer,
				c.Repos.MarketData,
				c.Log,
			)

		case "markPrice":
			c.Log.Info("Creating WebSocket mark price consumer",
				"topic", events.TopicWebSocketMarkPrice,
				"group_id", c.Config.Kafka.GroupID,
			)
			c.Background.WebSocketMarkPriceSvc = consumers.NewWebSocketMarkPriceConsumer(
				c.Adapters.WebSocketMarkPriceConsumer,
				c.Repos.MarketData,
				c.Log,
			)

		case "ticker":
			c.Log.Info("Creating WebSocket ticker consumer",
				"topic", events.TopicWebSocketTicker,
				"group_id", c.Config.Kafka.GroupID,
			)
			c.Background.WebSocketTickerSvc = consumers.NewWebSocketTickerConsumer(
				c.Adapters.WebSocketTickerConsumer,
				c.Repos.MarketData,
				c.Log,
			)

		case "trade":
			c.Log.Info("Creating WebSocket trade consumer",
				"topic", events.TopicWebSocketTrade,
				"group_id", c.Config.Kafka.GroupID,
			)
			c.Background.WebSocketTradeSvc = consumers.NewWebSocketTradeConsumer(
				c.Adapters.WebSocketTradeConsumer,
				c.Repos.MarketData,
				c.Log,
			)

		case "depth":
			c.Log.Info("Creating WebSocket depth consumer",
				"topic", events.TopicWebSocketDepth,
				"group_id", c.Config.Kafka.GroupID,
			)
			c.Background.WebSocketDepthSvc = consumers.NewWebSocketDepthConsumer(
				c.Adapters.WebSocketDepthConsumer,
				c.Repos.MarketData,
				c.Log,
			)
		}
	}

	c.Log.Info("✓ WebSocket infrastructure initialized",
		"stream_types", streamTypes,
		"kline_consumer", c.Background.WebSocketKlineSvc != nil,
		"markprice_consumer", c.Background.WebSocketMarkPriceSvc != nil,
		"ticker_consumer", c.Background.WebSocketTickerSvc != nil,
		"trade_consumer", c.Background.WebSocketTradeSvc != nil,
		"depth_consumer", c.Background.WebSocketDepthSvc != nil,
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

			default:
				c.Log.Warn("Unknown stream type in config", "type", streamType)
			}
		}
	}

	c.Log.Info("Built stream configuration",
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

	// TODO: Stop other exchange clients

	if lastErr != nil {
		return fmt.Errorf("errors occurred during WebSocket shutdown: %w", lastErr)
	}

	c.Log.Info("✓ All WebSocket clients stopped")
	return nil
}
