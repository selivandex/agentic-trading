package websocket

import (
	"context"
	"time"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ReconnectConfig configures auto-reconnection behavior
type ReconnectConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	ReconnectOnPing bool // Reconnect if ping fails
}

// DefaultReconnectConfig returns sensible defaults
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		MaxRetries:      10,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		ReconnectOnPing: true,
	}
}

// ReconnectHandler manages automatic reconnection for a WebSocket client
type ReconnectHandler struct {
	client  WebSocketClient
	config  ReconnectConfig
	log     *logger.Logger
	retries int
}

// NewReconnectHandler creates a new reconnect handler
func NewReconnectHandler(client WebSocketClient, config ReconnectConfig) *ReconnectHandler {
	return &ReconnectHandler{
		client:  client,
		config:  config,
		log:     logger.Get().With("component", "ws_reconnect"),
		retries: 0,
	}
}

// HandleDisconnect handles disconnection and attempts reconnection
func (rh *ReconnectHandler) HandleDisconnect(ctx context.Context) error {
	rh.log.Warn("WebSocket disconnected, starting reconnection attempts")

	delay := rh.config.InitialDelay

	for attempt := 1; attempt <= rh.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			rh.log.Infof("Reconnection attempt %d/%d", attempt, rh.config.MaxRetries)

			if err := rh.client.Connect(ctx); err != nil {
				rh.log.Errorf("Reconnection attempt %d failed: %v", attempt, err)

				// Increase delay with exponential backoff
				delay = min(time.Duration(float64(delay)*rh.config.BackoffFactor), rh.config.MaxDelay)

				continue
			}

			// Successful reconnection
			rh.log.Info("Successfully reconnected WebSocket")
			rh.retries = 0
			return nil
		}
	}

	rh.log.Error("Max reconnection attempts reached, giving up")
	return errors.ErrWSMaxReconnectAttempts
}

// MonitorConnection monitors connection health via periodic pings
func (rh *ReconnectHandler) MonitorConnection(ctx context.Context, pingInterval time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !rh.client.IsConnected() {
				rh.log.Warn("Connection lost, attempting reconnect")
				if err := rh.HandleDisconnect(ctx); err != nil {
					rh.log.Errorf("Failed to reconnect: %v", err)
				}
				continue
			}

			// Send ping if configured
			if rh.config.ReconnectOnPing {
				if err := rh.client.Ping(); err != nil {
					rh.log.Warnf("Ping failed: %v, reconnecting", err)
					if err := rh.HandleDisconnect(ctx); err != nil {
						rh.log.Errorf("Failed to reconnect after ping failure: %v", err)
					}
				}
			}
		}
	}
}

// GetRetryCount returns current retry count
func (rh *ReconnectHandler) GetRetryCount() int {
	return rh.retries
}

// ResetRetryCount resets retry counter
func (rh *ReconnectHandler) ResetRetryCount() {
	rh.retries = 0
}
