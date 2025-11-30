package binance

import (
	"context"
	"fmt"
	"time"

	"github.com/adshao/go-binance/v2/futures"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// ListenKeyService manages Binance Futures User Data Stream listenKey lifecycle
// ListenKey is required for authenticated WebSocket connections to receive:
// - Order updates (ORDER_TRADE_UPDATE)
// - Position updates (ACCOUNT_UPDATE)
// - Balance updates (ACCOUNT_UPDATE)
// - Margin calls (MARGIN_CALL)
type ListenKeyService struct {
	useTestnet bool
	logger     *logger.Logger
}

// NewListenKeyService creates a new listenKey manager
func NewListenKeyService(useTestnet bool, log *logger.Logger) *ListenKeyService {
	return &ListenKeyService{
		useTestnet: useTestnet,
		logger:     log,
	}
}

// Create generates a new listenKey via REST API
// Binance Futures: POST /fapi/v1/listenKey
// ListenKey is valid for 24 hours and must be renewed every 30 minutes
func (s *ListenKeyService) Create(ctx context.Context, apiKey, secret string) (string, time.Time, error) {
	// Log safely (preview only for debugging)
	apiKeyPreview := "empty"
	if len(apiKey) > 0 {
		if len(apiKey) >= 8 {
			apiKeyPreview = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		} else {
			apiKeyPreview = "***"
		}
	}

	s.logger.Debugw("Creating Binance listenKey",
		"testnet", s.useTestnet,
		"api_key_length", len(apiKey),
		"api_key_preview", apiKeyPreview,
		"secret_length", len(secret),
	)

	futures.UseTestnet = s.useTestnet

	client := futures.NewClient(apiKey, secret)

	s.logger.Debugw("Calling Binance API to create listenKey",
		"testnet", s.useTestnet,
	)

	listenKey, err := client.NewStartUserStreamService().Do(ctx)
	if err != nil {
		s.logger.Errorw("Binance API returned error when creating listenKey",
			"testnet", s.useTestnet,
			"api_key_length", len(apiKey),
			"api_key_preview", apiKeyPreview,
			"error", err,
		)
		return "", time.Time{}, errors.Wrap(err, "failed to create listenKey")
	}

	if listenKey == "" {
		return "", time.Time{}, errors.New("received empty listenKey from binance")
	}

	// Binance listenKeys expire after 24 hours if not renewed
	// We set expiration to 60 minutes and renew every 30 minutes for safety
	expiresAt := time.Now().Add(60 * time.Minute)

	s.logger.Infow("Created new Binance listenKey",
		"testnet", s.useTestnet,
		"expires_at", expiresAt,
	)

	return listenKey, expiresAt, nil
}

// Renew extends the validity period of an existing listenKey
// Binance Futures: PUT /fapi/v1/listenKey
// Must be called at least once every 60 minutes (we do it every 30 min)
func (s *ListenKeyService) Renew(ctx context.Context, apiKey, secret, listenKey string) (time.Time, error) {
	if listenKey == "" {
		return time.Time{}, errors.New("listenKey is empty")
	}

	futures.UseTestnet = s.useTestnet

	client := futures.NewClient(apiKey, secret)

	err := client.NewKeepaliveUserStreamService().
		ListenKey(listenKey).
		Do(ctx)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to renew listenKey")
	}

	// Reset expiration to 60 minutes from now
	expiresAt := time.Now().Add(60 * time.Minute)

	s.logger.Debugw("Renewed Binance listenKey",
		"testnet", s.useTestnet,
		"new_expires_at", expiresAt,
	)

	return expiresAt, nil
}

// Delete closes the User Data Stream and invalidates the listenKey
// Binance Futures: DELETE /fapi/v1/listenKey
// Should be called when user disconnects or account is deactivated
func (s *ListenKeyService) Delete(ctx context.Context, apiKey, secret, listenKey string) error {
	if listenKey == "" {
		return nil // Nothing to delete
	}

	futures.UseTestnet = s.useTestnet

	client := futures.NewClient(apiKey, secret)

	err := client.NewCloseUserStreamService().
		ListenKey(listenKey).
		Do(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to delete listenKey")
	}

	s.logger.Infow("Deleted Binance listenKey",
		"testnet", s.useTestnet,
	)

	return nil
}

// ValidateCredentials checks if API credentials are valid by creating a temporary listenKey
// Returns an error if authentication fails
func (s *ListenKeyService) ValidateCredentials(ctx context.Context, apiKey, secret string) error {
	if apiKey == "" || secret == "" {
		return fmt.Errorf("api key or secret is empty")
	}

	// Try to create a listenKey
	listenKey, _, err := s.Create(ctx, apiKey, secret)
	if err != nil {
		return errors.Wrap(err, "invalid credentials")
	}

	// Clean up immediately
	if err := s.Delete(ctx, apiKey, secret, listenKey); err != nil {
		s.logger.Warnw("Failed to delete validation listenKey",
			"error", err,
		)
	}

	return nil
}
