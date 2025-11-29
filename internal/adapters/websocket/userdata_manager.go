package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserDataFactory is an interface for creating exchange-specific User Data clients
// Implementation is in bootstrap to avoid import cycles
type UserDataFactory interface {
	Create(
		exchange exchange_account.ExchangeType,
		accountID uuid.UUID,
		userID uuid.UUID,
		handler UserDataEventHandler,
		useTestnet bool,
	) (UserDataStreamer, error)
}

// UserDataManager manages User Data WebSocket connections for all active exchange accounts
// - Loads active accounts from DB on startup
// - Creates/renews listenKeys for each account
// - Maintains WebSocket connections per account
// - Handles reconnections and health monitoring
// - Persists listenKeys to DB for crash recovery
type UserDataManager struct {
	accountRepo exchange_account.Repository
	factory     UserDataFactory
	handler     UserDataEventHandler
	encryptor   *crypto.Encryptor
	logger      *logger.Logger

	// Connection pool: accountID -> client
	clients   map[uuid.UUID]UserDataStreamer
	clientsMu sync.RWMutex

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// Health monitoring
	healthCheckInterval time.Duration
	listenKeyRenewal    time.Duration

	// Stats
	activeConnections int
	totalReconnects   int
	statsMu           sync.RWMutex
}

// UserDataManagerConfig configures the UserDataManager
type UserDataManagerConfig struct {
	HealthCheckInterval time.Duration
	ListenKeyRenewal    time.Duration
}

// NewUserDataManager creates a new User Data WebSocket manager
func NewUserDataManager(
	accountRepo exchange_account.Repository,
	factory UserDataFactory,
	handler UserDataEventHandler,
	encryptor *crypto.Encryptor,
	config UserDataManagerConfig,
	log *logger.Logger,
) *UserDataManager {
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 60 * time.Second
	}
	if config.ListenKeyRenewal == 0 {
		config.ListenKeyRenewal = 30 * time.Minute
	}

	return &UserDataManager{
		accountRepo:         accountRepo,
		factory:             factory,
		handler:             handler,
		encryptor:           encryptor,
		logger:              log,
		clients:             make(map[uuid.UUID]UserDataStreamer),
		stopChan:            make(chan struct{}),
		doneChan:            make(chan struct{}),
		healthCheckInterval: config.HealthCheckInterval,
		listenKeyRenewal:    config.ListenKeyRenewal,
	}
}

// Start initializes all User Data WebSocket connections for active accounts
func (m *UserDataManager) Start(ctx context.Context) error {
	m.logger.Info("User Data Manager initialized (connections will be added dynamically)")

	// TODO: Implement loading all active accounts from DB
	// For now, connections are added dynamically via AddAccount() method
	// when users connect their exchange accounts

	// Start background monitoring
	go m.healthCheckLoop(ctx)
	go m.listenKeyRenewalLoop(ctx)

	m.logger.Info("✓ User Data Manager started",
		"active_connections", m.GetActiveConnectionCount(),
	)

	return nil
}

// connectAccount establishes WebSocket connection for a single account
func (m *UserDataManager) connectAccount(ctx context.Context, account *exchange_account.ExchangeAccount) error {
	// Decrypt credentials
	apiKey, err := account.GetAPIKey(m.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt API key")
	}

	secret, err := account.GetSecret(m.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt secret")
	}

	// Create client
	client, err := m.factory.Create(
		account.Exchange,
		account.ID,
		account.UserID,
		m.handler,
		account.IsTestnet,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create client")
	}

	// Get or create listenKey
	listenKey, expiresAt, err := m.ensureListenKey(ctx, account, client, apiKey, secret)
	if err != nil {
		return errors.Wrap(err, "failed to ensure listenKey")
	}

	// Connect WebSocket
	if err := client.Connect(ctx, listenKey, apiKey, secret); err != nil {
		return errors.Wrap(err, "failed to connect WebSocket")
	}

	// Start receiving events
	if err := client.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start WebSocket")
	}

	// Store client in pool
	m.clientsMu.Lock()
	m.clients[account.ID] = client
	m.activeConnections++
	m.clientsMu.Unlock()

	m.logger.Info("✓ Connected User Data WebSocket",
		"account_id", account.ID,
		"user_id", account.UserID,
		"exchange", account.Exchange,
		"listen_key_expires_at", expiresAt,
	)

	return nil
}

// ensureListenKey gets existing valid listenKey or creates a new one
func (m *UserDataManager) ensureListenKey(
	ctx context.Context,
	account *exchange_account.ExchangeAccount,
	client UserDataStreamer,
	apiKey, secret string,
) (string, time.Time, error) {
	// Check if we have a valid listenKey
	if !account.IsListenKeyExpired() {
		listenKey, err := account.GetListenKey(m.encryptor)
		if err == nil && listenKey != "" {
			m.logger.Debug("Using existing listenKey",
				"account_id", account.ID,
				"expires_at", account.ListenKeyExpiresAt,
			)
			return listenKey, *account.ListenKeyExpiresAt, nil
		}
	}

	// Create new listenKey
	listenKey, expiresAt, err := client.CreateListenKey(ctx, apiKey, secret)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "failed to create listenKey")
	}

	// Encrypt and store listenKey in account
	if err := account.SetListenKey(listenKey, expiresAt, m.encryptor); err != nil {
		return "", time.Time{}, errors.Wrap(err, "failed to encrypt listenKey")
	}

	// Persist to DB
	if err := m.accountRepo.Update(ctx, account); err != nil {
		return "", time.Time{}, errors.Wrap(err, "failed to persist listenKey")
	}

	m.logger.Info("Created and persisted new listenKey",
		"account_id", account.ID,
		"expires_at", expiresAt,
	)

	return listenKey, expiresAt, nil
}

// Stop gracefully shuts down all WebSocket connections
func (m *UserDataManager) Stop(ctx context.Context) error {
	m.logger.Info("Stopping User Data Manager...")

	close(m.stopChan)

	// Stop all clients
	m.clientsMu.Lock()
	clients := make([]UserDataStreamer, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.clientsMu.Unlock()

	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c UserDataStreamer) {
			defer wg.Done()
			if err := c.Stop(ctx); err != nil {
				m.logger.Error("Failed to stop client",
					"error", err,
				)
			}
		}(client)
	}

	wg.Wait()

	m.clientsMu.Lock()
	m.clients = make(map[uuid.UUID]UserDataStreamer)
	m.activeConnections = 0
	m.clientsMu.Unlock()

	close(m.doneChan)

	m.logger.Info("✓ User Data Manager stopped")

	return nil
}

// healthCheckLoop periodically checks connection health and reconnects if needed
func (m *UserDataManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performHealthCheck(ctx)
		case <-m.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performHealthCheck checks all connections and reconnects dead ones
func (m *UserDataManager) performHealthCheck(ctx context.Context) {
	m.clientsMu.RLock()
	accountIDs := make([]uuid.UUID, 0, len(m.clients))
	for accountID := range m.clients {
		accountIDs = append(accountIDs, accountID)
	}
	m.clientsMu.RUnlock()

	for _, accountID := range accountIDs {
		m.clientsMu.RLock()
		client, exists := m.clients[accountID]
		m.clientsMu.RUnlock()

		if !exists {
			continue
		}

		if !client.IsConnected() {
			m.logger.Warn("Detected disconnected client, attempting reconnect",
				"account_id", accountID,
			)

			// Reconnect
			if err := m.reconnectAccount(ctx, accountID); err != nil {
				m.logger.Error("Failed to reconnect account",
					"account_id", accountID,
					"error", err,
				)
			}
		}
	}
}

// listenKeyRenewalLoop periodically renews listenKeys before expiration
func (m *UserDataManager) listenKeyRenewalLoop(ctx context.Context) {
	ticker := time.NewTicker(m.listenKeyRenewal)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.renewAllListenKeys(ctx)
		case <-m.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// renewAllListenKeys renews listenKeys for all active connections
func (m *UserDataManager) renewAllListenKeys(ctx context.Context) {
	m.clientsMu.RLock()
	accountIDs := make([]uuid.UUID, 0, len(m.clients))
	for accountID := range m.clients {
		accountIDs = append(accountIDs, accountID)
	}
	m.clientsMu.RUnlock()

	for _, accountID := range accountIDs {
		// Load account from DB
		account, err := m.accountRepo.GetByID(ctx, accountID)
		if err != nil {
			m.logger.Error("Failed to load account for listenKey renewal",
				"account_id", accountID,
				"error", err,
			)
			continue
		}

		// Check if renewal is needed
		if !account.ShouldRenewListenKey() {
			continue
		}

		m.logger.Info("Renewing listenKey",
			"account_id", accountID,
			"current_expires_at", account.ListenKeyExpiresAt,
		)

		// Get client
		m.clientsMu.RLock()
		client, exists := m.clients[accountID]
		m.clientsMu.RUnlock()

		if !exists {
			continue
		}

		// Decrypt credentials
		apiKey, _ := account.GetAPIKey(m.encryptor)
		secret, _ := account.GetSecret(m.encryptor)
		listenKey, _ := account.GetListenKey(m.encryptor)

		// Renew
		if err := client.RenewListenKey(ctx, apiKey, secret, listenKey); err != nil {
			m.logger.Error("Failed to renew listenKey",
				"account_id", accountID,
				"error", err,
			)
			continue
		}

		// Update expiration in DB (client already updated it internally)
		// We don't need to persist it again since client handles it
	}
}

// reconnectAccount attempts to reconnect a disconnected account
func (m *UserDataManager) reconnectAccount(ctx context.Context, accountID uuid.UUID) error {
	// Stop existing client
	m.clientsMu.Lock()
	if client, exists := m.clients[accountID]; exists {
		_ = client.Stop(ctx)
		delete(m.clients, accountID)
		m.activeConnections--
	}
	m.clientsMu.Unlock()

	// Load account
	account, err := m.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to load account")
	}

	// Reconnect
	if err := m.connectAccount(ctx, account); err != nil {
		return errors.Wrap(err, "failed to reconnect")
	}

	m.statsMu.Lock()
	m.totalReconnects++
	m.statsMu.Unlock()

	m.logger.Info("✓ Reconnected account",
		"account_id", accountID,
	)

	return nil
}

// AddAccount dynamically adds a new account to the manager
func (m *UserDataManager) AddAccount(ctx context.Context, accountID uuid.UUID) error {
	// Check if already connected
	m.clientsMu.RLock()
	_, exists := m.clients[accountID]
	m.clientsMu.RUnlock()

	if exists {
		return errors.New("account already connected")
	}

	// Load account
	account, err := m.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to load account")
	}

	// Connect
	return m.connectAccount(ctx, account)
}

// RemoveAccount dynamically removes an account from the manager
func (m *UserDataManager) RemoveAccount(ctx context.Context, accountID uuid.UUID) error {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	client, exists := m.clients[accountID]
	if !exists {
		return errors.New("account not connected")
	}

	// Stop client
	if err := client.Stop(ctx); err != nil {
		return errors.Wrap(err, "failed to stop client")
	}

	delete(m.clients, accountID)
	m.activeConnections--

	m.logger.Info("Removed account from User Data Manager",
		"account_id", accountID,
	)

	return nil
}

// GetActiveConnectionCount returns the number of active connections
func (m *UserDataManager) GetActiveConnectionCount() int {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	return m.activeConnections
}

// GetStats returns manager statistics
func (m *UserDataManager) GetStats() map[string]interface{} {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	m.clientsMu.RLock()
	activeConns := m.activeConnections
	m.clientsMu.RUnlock()

	return map[string]interface{}{
		"active_connections": activeConns,
		"total_reconnects":   m.totalReconnects,
	}
}
