package websocket

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/internal/metrics"
	exchangeservice "prometheus/internal/services/exchange"
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
// ExchangeService defines interface for exchange account management
type ExchangeService interface {
	DeactivateAccount(ctx context.Context, input exchangeservice.DeactivateAccountInput) error
	SaveListenKey(ctx context.Context, accountID uuid.UUID, listenKey string, expiresAt time.Time) error
	UpdateListenKeyExpiration(ctx context.Context, accountID uuid.UUID, expiresAt time.Time) error
}

type UserDataManager struct {
	accountRepo     exchange_account.Repository
	exchangeService ExchangeService
	factory         UserDataFactory
	handler         UserDataEventHandler
	encryptor       *crypto.Encryptor
	logger          *logger.Logger

	// Connection pool: accountID -> client
	clients   map[uuid.UUID]UserDataStreamer
	clientsMu sync.RWMutex

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// Health monitoring
	healthCheckInterval    time.Duration
	listenKeyRenewal       time.Duration
	reconciliationInterval time.Duration

	// Stats
	activeConnections  int
	totalReconnects    int
	totalReconciled    int
	lastReconciliation time.Time
	statsMu            sync.RWMutex
}

// UserDataManagerConfig configures the UserDataManager
type UserDataManagerConfig struct {
	HealthCheckInterval    time.Duration // How often to check connection health
	ListenKeyRenewal       time.Duration // How often to renew listenKeys
	ReconciliationInterval time.Duration // How often to sync with DB (hot reload)
}

// NewUserDataManager creates a new User Data WebSocket manager
func NewUserDataManager(
	accountRepo exchange_account.Repository,
	exchangeService ExchangeService,
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
	if config.ReconciliationInterval == 0 {
		config.ReconciliationInterval = 20 * time.Second // Default: check DB every 20 seconds
	}

	return &UserDataManager{
		accountRepo:            accountRepo,
		exchangeService:        exchangeService,
		factory:                factory,
		handler:                handler,
		encryptor:              encryptor,
		logger:                 log,
		clients:                make(map[uuid.UUID]UserDataStreamer),
		stopChan:               make(chan struct{}),
		doneChan:               make(chan struct{}),
		healthCheckInterval:    config.HealthCheckInterval,
		listenKeyRenewal:       config.ListenKeyRenewal,
		reconciliationInterval: config.ReconciliationInterval,
	}
}

// Start initializes all User Data WebSocket connections for active accounts
func (m *UserDataManager) Start(ctx context.Context) error {
	m.logger.Infow("Starting User Data Manager...")

	// Load and connect all active accounts from DB
	if err := m.loadAllActiveAccounts(ctx); err != nil {
		m.logger.Errorw("Failed to load active accounts on startup", "error", err)
		// Don't fail startup, continue with empty state
	}

	// Start background monitoring loops
	go m.healthCheckLoop(ctx)
	go m.listenKeyRenewalLoop(ctx)
	go m.reconciliationLoop(ctx) // Hot reload: sync with DB periodically

	m.logger.Infow("✓ User Data Manager started",
		"active_connections", m.GetActiveConnectionCount(),
	)

	return nil
}

// loadAllActiveAccounts loads all active exchange accounts from DB and connects them
func (m *UserDataManager) loadAllActiveAccounts(ctx context.Context) error {
	m.logger.Infow("Loading all active exchange accounts from DB...")

	accounts, err := m.accountRepo.GetAllActive(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get all active accounts")
	}

	m.logger.Infow("Found active accounts to connect",
		"count", len(accounts),
	)

	var successCount, failureCount int

	for _, account := range accounts {
		// Check if shutdown requested during startup
		select {
		case <-ctx.Done():
			m.logger.Warnw("Loading accounts interrupted by shutdown",
				"loaded", successCount,
				"failed", failureCount,
				"remaining", len(accounts)-successCount-failureCount,
			)
			return errors.Wrap(ctx.Err(), "startup interrupted")
		default:
		}

		if err := m.connectAccount(ctx, account); err != nil {
			m.logger.Errorw("Failed to connect account on startup",
				"account_id", account.ID,
				"user_id", account.UserID,
				"exchange", account.Exchange,
				"error", err,
			)
			failureCount++
			continue
		}
		successCount++
	}

	m.logger.Infow("✓ Loaded active accounts",
		"total", len(accounts),
		"success", successCount,
		"failures", failureCount,
	)

	return nil
}

// connectAccount establishes WebSocket connection for a single account
func (m *UserDataManager) connectAccount(ctx context.Context, account *exchange_account.ExchangeAccount) error {
	m.logger.Infow("🔌 Connecting User Data WebSocket for account",
		"account_id", account.ID,
		"user_id", account.UserID,
		"exchange", account.Exchange,
		"label", account.Label,
		"testnet", account.IsTestnet,
	)

	// Decrypt credentials
	m.logger.Debugw("Decrypting API credentials",
		"account_id", account.ID,
		"encrypted_api_key_bytes", len(account.APIKeyEncrypted),
		"encrypted_secret_bytes", len(account.SecretEncrypted),
	)

	apiKey, err := account.GetAPIKey(m.encryptor)
	if err != nil {
		m.logger.Errorw("Failed to decrypt API key",
			"account_id", account.ID,
			"encrypted_bytes", len(account.APIKeyEncrypted),
			"error", err,
		)
		return errors.Wrap(err, "failed to decrypt API key")
	}

	secret, err := account.GetSecret(m.encryptor)
	if err != nil {
		m.logger.Errorw("Failed to decrypt secret",
			"account_id", account.ID,
			"encrypted_bytes", len(account.SecretEncrypted),
			"error", err,
		)
		return errors.Wrap(err, "failed to decrypt secret")
	}

	// Log safely (length and first/last chars only for debugging)
	apiKeyPreview := "empty"
	if len(apiKey) > 0 {
		if len(apiKey) >= 8 {
			apiKeyPreview = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		} else {
			apiKeyPreview = apiKey[:1] + "..." + apiKey[len(apiKey)-1:]
		}
	}

	// Get char codes for debugging
	var firstCharCode, lastCharCode int
	if len(apiKey) > 0 {
		firstCharCode = int(apiKey[0])
		lastCharCode = int(apiKey[len(apiKey)-1])
	}

	m.logger.Debugw("Credentials decrypted successfully",
		"account_id", account.ID,
		"api_key_length", len(apiKey),
		"api_key_preview", apiKeyPreview,
		"secret_length", len(secret),
		"has_whitespace_start", len(apiKey) > 0 && (apiKey[0] == ' ' || apiKey[0] == '\t' || apiKey[0] == '\n'),
		"has_whitespace_end", len(apiKey) > 0 && (apiKey[len(apiKey)-1] == ' ' || apiKey[len(apiKey)-1] == '\t' || apiKey[len(apiKey)-1] == '\n'),
		"api_key_first_char_code", firstCharCode,
		"api_key_last_char_code", lastCharCode,
	)

	// Create client
	m.logger.Debugw("Creating WebSocket client",
		"account_id", account.ID,
		"exchange", account.Exchange,
	)

	client, err := m.factory.Create(
		account.Exchange,
		account.ID,
		account.UserID,
		m.handler,
		account.IsTestnet,
	)
	if err != nil {
		m.logger.Errorw("Failed to create WebSocket client",
			"account_id", account.ID,
			"exchange", account.Exchange,
			"error", err,
		)
		return errors.Wrap(err, "failed to create client")
	}

	m.logger.Debugw("WebSocket client created",
		"account_id", account.ID,
	)

	// Get or create listenKey
	m.logger.Debugw("Ensuring listenKey",
		"account_id", account.ID,
	)

	listenKey, expiresAt, err := m.ensureListenKey(ctx, account, client, apiKey, secret)
	if err != nil {
		m.logger.Errorw("Failed to ensure listenKey",
			"account_id", account.ID,
			"error", err,
		)

		// Check if it's a credentials error
		if m.isCredentialsError(err) {
			m.logger.Warnw("Detected credentials error, deactivating account and notifying user",
				"account_id", account.ID,
				"user_id", account.UserID,
				"exchange", account.Exchange,
			)
			m.handleCredentialsError(ctx, account, err)
		}

		return errors.Wrap(err, "failed to ensure listenKey")
	}

	m.logger.Debugw("ListenKey ready",
		"account_id", account.ID,
		"expires_at", expiresAt,
	)

	// Connect WebSocket
	m.logger.Debugw("Connecting WebSocket",
		"account_id", account.ID,
	)

	if err := client.Connect(ctx, listenKey, apiKey, secret); err != nil {
		m.logger.Errorw("Failed to connect WebSocket",
			"account_id", account.ID,
			"error", err,
		)
		return errors.Wrap(err, "failed to connect WebSocket")
	}

	m.logger.Debugw("WebSocket connected, starting event stream",
		"account_id", account.ID,
	)

	// Start receiving events
	if err := client.Start(ctx); err != nil {
		m.logger.Errorw("Failed to start WebSocket event stream",
			"account_id", account.ID,
			"error", err,
		)
		return errors.Wrap(err, "failed to start WebSocket")
	}

	m.logger.Debugw("WebSocket event stream started",
		"account_id", account.ID,
	)

	// Store client in pool
	m.clientsMu.Lock()
	m.clients[account.ID] = client
	m.activeConnections++
	currentConnections := m.activeConnections
	m.clientsMu.Unlock()

	// Update metrics
	metrics.UserDataConnections.WithLabelValues(string(account.Exchange)).Inc()

	m.logger.Infow("✅ User Data WebSocket connected and streaming",
		"account_id", account.ID,
		"user_id", account.UserID,
		"exchange", account.Exchange,
		"label", account.Label,
		"listen_key_expires_at", expiresAt,
		"total_active_connections", currentConnections,
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
	m.logger.Debugw("Checking listenKey status",
		"account_id", account.ID,
		"has_listen_key", account.ListenKeyExpiresAt != nil,
	)

	// Check if we have a valid listenKey
	if !account.IsListenKeyExpired() {
		listenKey, err := account.GetListenKey(m.encryptor)
		if err == nil && listenKey != "" {
			m.logger.Debugw("✓ Using existing valid listenKey",
				"account_id", account.ID,
				"expires_at", account.ListenKeyExpiresAt,
			)
			return listenKey, *account.ListenKeyExpiresAt, nil
		}
		m.logger.Debugw("Existing listenKey found but failed to decrypt",
			"account_id", account.ID,
			"error", err,
		)
	} else {
		m.logger.Debugw("ListenKey expired or not found, creating new one",
			"account_id", account.ID,
		)
	}

	// Create new listenKey
	m.logger.Debugw("Requesting new listenKey from exchange",
		"account_id", account.ID,
		"exchange", account.Exchange,
	)

	listenKey, expiresAt, err := client.CreateListenKey(ctx, apiKey, secret)
	if err != nil {
		m.logger.Errorw("Failed to create listenKey from exchange",
			"account_id", account.ID,
			"exchange", account.Exchange,
			"error", err,
		)
		return "", time.Time{}, errors.Wrap(err, "failed to create listenKey")
	}

	m.logger.Debugw("ListenKey received from exchange",
		"account_id", account.ID,
		"expires_at", expiresAt,
	)

	// Persist listenKey via exchange service (Clean Architecture)
	m.logger.Debugw("Persisting listenKey via exchange service",
		"account_id", account.ID,
	)

	if err := m.exchangeService.SaveListenKey(ctx, account.ID, listenKey, expiresAt); err != nil {
		m.logger.Errorw("Failed to persist listenKey via service",
			"account_id", account.ID,
			"error", err,
		)
		return "", time.Time{}, errors.Wrap(err, "failed to persist listenKey")
	}

	m.logger.Infow("✅ Created and persisted new listenKey",
		"account_id", account.ID,
		"exchange", account.Exchange,
		"expires_at", expiresAt,
	)

	return listenKey, expiresAt, nil
}

// Stop gracefully shuts down all WebSocket connections
func (m *UserDataManager) Stop(ctx context.Context) error {
	m.logger.Infow("Stopping User Data Manager...")

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
				m.logger.Errorw("Failed to stop client",
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

	m.logger.Infow("✓ User Data Manager stopped")

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
		// Check if shutdown requested
		select {
		case <-ctx.Done():
			m.logger.Debugw("Health check interrupted by shutdown")
			return
		default:
		}

		m.clientsMu.RLock()
		client, exists := m.clients[accountID]
		m.clientsMu.RUnlock()

		if !exists {
			continue
		}

		if !client.IsConnected() {
			m.logger.Warnw("Detected disconnected client, attempting reconnect",
				"account_id", accountID,
			)

			// Reconnect
			if err := m.reconnectAccount(ctx, accountID); err != nil {
				m.logger.Errorw("Failed to reconnect account",
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

// reconciliationLoop periodically syncs manager state with DB (hot reload)
// - Connects new active accounts
// - Disconnects deactivated/deleted accounts
func (m *UserDataManager) reconciliationLoop(ctx context.Context) {
	ticker := time.NewTicker(m.reconciliationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performReconciliation(ctx)
		case <-m.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performReconciliation syncs manager state with database
func (m *UserDataManager) performReconciliation(ctx context.Context) {
	m.logger.Debugw("🔄 Starting reconciliation with database (hot reload check)...")

	// Get all active accounts from DB
	activeAccounts, err := m.accountRepo.GetAllActive(ctx)
	if err != nil {
		m.logger.Errorw("Failed to get active accounts for reconciliation", "error", err)
		metrics.UserDataReconciliations.WithLabelValues("error").Inc()
		return
	}

	// Build set of active account IDs from DB
	activeAccountIDs := make(map[uuid.UUID]bool)
	for _, acc := range activeAccounts {
		activeAccountIDs[acc.ID] = true
	}

	// Get currently connected account IDs
	m.clientsMu.RLock()
	connectedAccountIDs := make(map[uuid.UUID]bool)
	for accountID := range m.clients {
		connectedAccountIDs[accountID] = true
	}
	currentlyConnected := len(connectedAccountIDs)
	m.clientsMu.RUnlock()

	m.logger.Debugw("Reconciliation state check",
		"active_in_db", len(activeAccounts),
		"currently_connected", currentlyConnected,
	)

	var addedCount, removedCount int

	// Step 1: Connect accounts that are active in DB but not connected
	for _, account := range activeAccounts {
		// Check if shutdown requested
		select {
		case <-ctx.Done():
			m.logger.Infow("Reconciliation interrupted by shutdown",
				"added_so_far", addedCount,
				"remaining", len(activeAccounts)-addedCount,
			)
			return
		default:
		}

		if !connectedAccountIDs[account.ID] {
			m.logger.Infow("🆕 HOT RELOAD: Detected new active account in database",
				"account_id", account.ID,
				"user_id", account.UserID,
				"exchange", account.Exchange,
				"label", account.Label,
				"created_at", account.CreatedAt,
			)

			if err := m.connectAccount(ctx, account); err != nil {
				m.logger.Errorw("❌ Failed to connect account during hot reload",
					"account_id", account.ID,
					"user_id", account.UserID,
					"exchange", account.Exchange,
					"error", err,
				)
				continue
			}

			// Record hot reload metric
			metrics.UserDataHotReload.WithLabelValues("add", string(account.Exchange)).Inc()
			addedCount++

			m.logger.Infow("✅ HOT RELOAD: Successfully connected new account",
				"account_id", account.ID,
				"user_id", account.UserID,
				"exchange", account.Exchange,
				"total_connections_now", m.GetActiveConnectionCount(),
			)
		}
	}

	// Step 2: Disconnect accounts that are connected but no longer active in DB
	for accountID := range connectedAccountIDs {
		// Check if shutdown requested
		select {
		case <-ctx.Done():
			m.logger.Infow("Reconciliation interrupted by shutdown during cleanup",
				"removed_so_far", removedCount,
			)
			return
		default:
		}

		if !activeAccountIDs[accountID] {
			// Get exchange info before removal
			account, _ := m.accountRepo.GetByID(ctx, accountID)
			exchange := "unknown"
			if account != nil {
				exchange = string(account.Exchange)
			}

			m.logger.Infow("Hot reload: disconnecting deactivated account",
				"account_id", accountID,
				"exchange", exchange,
			)

			if err := m.RemoveAccount(ctx, accountID); err != nil {
				m.logger.Errorw("Failed to disconnect account during reconciliation",
					"account_id", accountID,
					"error", err,
				)
				continue
			}

			// Record hot reload metric
			metrics.UserDataHotReload.WithLabelValues("remove", exchange).Inc()
			removedCount++
		}
	}

	// Update stats
	m.statsMu.Lock()
	m.totalReconciled++
	m.lastReconciliation = time.Now()
	m.statsMu.Unlock()

	// Update metrics
	metrics.UserDataReconciliations.WithLabelValues("success").Inc()

	if addedCount > 0 || removedCount > 0 {
		m.logger.Infow("✓ Reconciliation completed",
			"added", addedCount,
			"removed", removedCount,
			"total_active", len(activeAccountIDs),
			"total_connected", m.GetActiveConnectionCount(),
		)
	} else {
		m.logger.Debugw("Reconciliation completed, no changes",
			"total_active", len(activeAccountIDs),
		)
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
		// Check if shutdown requested
		select {
		case <-ctx.Done():
			m.logger.Debugw("ListenKey renewal interrupted by shutdown")
			return
		default:
		}

		if err := m.renewListenKeyForAccount(ctx, accountID); err != nil {
			m.logger.Errorw("Failed to renew listenKey for account",
				"account_id", accountID,
				"error", err,
			)
			continue
		}
	}
}

// renewListenKeyForAccount renews listenKey for a specific account
// Business logic (check rules, handle errors) delegated to exchange service
func (m *UserDataManager) renewListenKeyForAccount(ctx context.Context, accountID uuid.UUID) error {
	// Load account from DB (via service to apply business rules)
	account, err := m.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to load account")
	}

	// Business rule: check if renewal needed (domain logic)
	if !account.ShouldRenewListenKey() {
		return nil // Skip renewal
	}

	// Get client
	m.clientsMu.RLock()
	client, exists := m.clients[accountID]
	m.clientsMu.RUnlock()

	if !exists {
		return errors.New("client not found")
	}

	// Decrypt credentials
	apiKey, err := account.GetAPIKey(m.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt API key")
	}

	secret, err := account.GetSecret(m.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt secret")
	}

	listenKey, err := account.GetListenKey(m.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt listenKey")
	}

	// Infrastructure call: actual API renewal
	if err := client.RenewListenKey(ctx, apiKey, secret, listenKey); err != nil {
		m.logger.Errorw("Failed to renew listenKey via API",
			"account_id", accountID,
			"error", err,
		)

		// Delegate error handling to exchange service (business logic)
		if m.isCredentialsError(err) {
			m.logger.Warnw("Credentials error during renewal, deactivating via service",
				"account_id", accountID,
			)
			deactivateErr := m.exchangeService.DeactivateAccount(ctx, exchangeservice.DeactivateAccountInput{
				AccountID: accountID,
				Reason:    "listenkey_renewal_failed",
				ErrorMsg:  err.Error(),
			})
			if deactivateErr != nil {
				m.logger.Errorw("Failed to deactivate account after renewal error",
					"account_id", accountID,
					"error", deactivateErr,
				)
			}
		}

		metrics.UserDataListenKeyRenewals.WithLabelValues(string(account.Exchange), "error").Inc()
		return errors.Wrap(err, "listenKey renewal failed")
	}

	// Update expiration in DB via exchange service
	expiresAt := time.Now().Add(60 * time.Minute)
	if err := m.exchangeService.UpdateListenKeyExpiration(ctx, accountID, expiresAt); err != nil {
		m.logger.Errorw("Failed to update listenKey expiration in DB",
			"account_id", accountID,
			"error", err,
		)
		// Don't fail renewal if DB update fails - renewal succeeded on Binance
	}

	// Update metrics
	metrics.UserDataListenKeyRenewals.WithLabelValues(string(account.Exchange), "success").Inc()

	m.logger.Debugw("✅ ListenKey renewed successfully",
		"account_id", accountID,
		"new_expires_at", expiresAt,
	)

	return nil
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

	// Update metrics
	exchange := string(account.Exchange)
	metrics.UserDataReconnects.WithLabelValues(exchange, "health_check").Inc()

	m.logger.Infow("✓ Reconnected account",
		"account_id", accountID,
		"exchange", exchange,
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

	// Get account info for metrics
	account, _ := m.accountRepo.GetByID(ctx, accountID)
	exchange := "unknown"
	if account != nil {
		exchange = string(account.Exchange)
	}

	// Stop client
	if err := client.Stop(ctx); err != nil {
		return errors.Wrap(err, "failed to stop client")
	}

	delete(m.clients, accountID)
	m.activeConnections--

	// Update metrics
	metrics.UserDataConnections.WithLabelValues(exchange).Dec()

	m.logger.Infow("Removed account from User Data Manager",
		"account_id", accountID,
		"exchange", exchange,
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
	totalReconnects := m.totalReconnects
	totalReconciled := m.totalReconciled
	lastReconciliation := m.lastReconciliation
	m.statsMu.RUnlock()

	m.clientsMu.RLock()
	activeConns := m.activeConnections
	m.clientsMu.RUnlock()

	return map[string]interface{}{
		"active_connections":  activeConns,
		"total_reconnects":    totalReconnects,
		"total_reconciled":    totalReconciled,
		"last_reconciliation": lastReconciliation,
	}
}

// isCredentialsError checks if error is related to invalid credentials
func (m *UserDataManager) isCredentialsError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Binance API error codes and messages
	credentialsErrors := []string{
		"code=-2015", // Invalid API-key, IP, or permissions for action
		"invalid api-key",
		"invalid signature",
		"api-key not found",
		"unauthorized",
		"permission denied",
		"insufficient permissions",
		"ip not whitelisted",
	}

	for _, errPattern := range credentialsErrors {
		if strings.Contains(errStr, errPattern) {
			return true
		}
	}

	return false
}

// handleCredentialsError handles credentials-related errors:
// - Deactivates the exchange account
// - Publishes event for notification service to alert user
func (m *UserDataManager) handleCredentialsError(ctx context.Context, account *exchange_account.ExchangeAccount, err error) {
	m.logger.Infow("Handling credentials error via exchange service",
		"account_id", account.ID,
		"user_id", account.UserID,
		"exchange", account.Exchange,
	)

	// Use exchange service to deactivate account and publish notification
	deactivateErr := m.exchangeService.DeactivateAccount(ctx, exchangeservice.DeactivateAccountInput{
		AccountID: account.ID,
		Reason:    "invalid_credentials",
		ErrorMsg:  err.Error(),
	})

	if deactivateErr != nil {
		m.logger.Errorw("Failed to deactivate account via exchange service",
			"account_id", account.ID,
			"error", deactivateErr,
		)
	} else {
		m.logger.Infow("✅ Account deactivated and notification sent",
			"account_id", account.ID,
			"user_id", account.UserID,
			"exchange", account.Exchange,
		)
	}
}
