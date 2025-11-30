package exchange

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service manages exchange account operations
type Service struct {
	repo                  exchange_account.Repository
	encryptor             *crypto.Encryptor
	notificationPublisher NotificationPublisher
	log                   *logger.Logger
}

// NotificationPublisher defines interface for publishing notification events
type NotificationPublisher interface {
	PublishExchangeDeactivated(ctx context.Context, accountID, userID, exchange, label, reason, errorMsg string, isTestnet bool) error
}

// NewService creates a new exchange service
func NewService(
	repo exchange_account.Repository,
	encryptor *crypto.Encryptor,
	notificationPublisher NotificationPublisher,
	log *logger.Logger,
) *Service {
	return &Service{
		repo:                  repo,
		encryptor:             encryptor,
		notificationPublisher: notificationPublisher,
		log:                   log.With("component", "exchange_service"),
	}
}

// CreateAccountInput contains data for creating exchange account
type CreateAccountInput struct {
	UserID     uuid.UUID
	Exchange   exchange_account.ExchangeType
	APIKey     string
	Secret     string
	Label      string
	IsTestnet  bool
	Passphrase string // Optional, for OKX
}

// UpdateAccountInput contains data for updating exchange account
type UpdateAccountInput struct {
	AccountID  uuid.UUID
	Label      *string // Optional
	APIKey     *string // Optional
	Secret     *string // Optional
	Passphrase *string // Optional
}

// CreateAccount creates a new exchange account with encrypted credentials
func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (*exchange_account.ExchangeAccount, error) {
	s.log.Infow("Creating exchange account",
		"user_id", input.UserID,
		"exchange", input.Exchange,
		"testnet", input.IsTestnet,
	)

	// Validate input
	if input.APIKey == "" || input.Secret == "" {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "API key and secret are required")
	}

	if !input.Exchange.Valid() {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "invalid exchange: %s", input.Exchange)
	}

	// Create account entity
	account := &exchange_account.ExchangeAccount{
		ID:          uuid.New(),
		UserID:      input.UserID,
		Exchange:    input.Exchange,
		Label:       input.Label,
		IsTestnet:   input.IsTestnet,
		Permissions: []string{"read", "trade"}, // Default permissions
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Encrypt credentials
	if err := account.SetAPIKey(input.APIKey, s.encryptor); err != nil {
		s.log.Errorw("Failed to encrypt API key",
			"account_id", account.ID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to encrypt API key")
	}

	if err := account.SetSecret(input.Secret, s.encryptor); err != nil {
		s.log.Errorw("Failed to encrypt secret",
			"account_id", account.ID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to encrypt secret")
	}

	if input.Passphrase != "" {
		if err := account.SetPassphrase(input.Passphrase, s.encryptor); err != nil {
			s.log.Errorw("Failed to encrypt passphrase",
				"account_id", account.ID,
				"error", err,
			)
			return nil, errors.Wrap(err, "failed to encrypt passphrase")
		}
	}

	// Save to repository
	if err := s.repo.Create(ctx, account); err != nil {
		s.log.Errorw("Failed to create exchange account",
			"account_id", account.ID,
			"user_id", input.UserID,
			"exchange", input.Exchange,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to create exchange account")
	}

	s.log.Infow("✅ Exchange account created successfully",
		"account_id", account.ID,
		"user_id", input.UserID,
		"exchange", input.Exchange,
		"label", input.Label,
	)

	return account, nil
}

// UpdateAccount updates exchange account details
func (s *Service) UpdateAccount(ctx context.Context, input UpdateAccountInput) (*exchange_account.ExchangeAccount, error) {
	s.log.Infow("Updating exchange account",
		"account_id", input.AccountID,
	)

	// Load account
	account, err := s.repo.GetByID(ctx, input.AccountID)
	if err != nil {
		s.log.Errorw("Failed to load exchange account",
			"account_id", input.AccountID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to load exchange account")
	}

	// Update label
	if input.Label != nil {
		oldLabel := account.Label
		account.Label = *input.Label
		s.log.Debugw("Updating label",
			"account_id", input.AccountID,
			"old_label", oldLabel,
			"new_label", *input.Label,
		)
	}

	// Update API key
	if input.APIKey != nil {
		if err := account.SetAPIKey(*input.APIKey, s.encryptor); err != nil {
			s.log.Errorw("Failed to encrypt new API key",
				"account_id", input.AccountID,
				"error", err,
			)
			return nil, errors.Wrap(err, "failed to encrypt API key")
		}
		s.log.Debugw("API key updated",
			"account_id", input.AccountID,
		)
	}

	// Update secret
	if input.Secret != nil {
		if err := account.SetSecret(*input.Secret, s.encryptor); err != nil {
			s.log.Errorw("Failed to encrypt new secret",
				"account_id", input.AccountID,
				"error", err,
			)
			return nil, errors.Wrap(err, "failed to encrypt secret")
		}
		s.log.Debugw("Secret updated",
			"account_id", input.AccountID,
		)
	}

	// Update passphrase
	if input.Passphrase != nil {
		if err := account.SetPassphrase(*input.Passphrase, s.encryptor); err != nil {
			s.log.Errorw("Failed to encrypt new passphrase",
				"account_id", input.AccountID,
				"error", err,
			)
			return nil, errors.Wrap(err, "failed to encrypt passphrase")
		}
		s.log.Debugw("Passphrase updated",
			"account_id", input.AccountID,
		)
	}

	account.UpdatedAt = time.Now()

	// Save changes
	if err := s.repo.Update(ctx, account); err != nil {
		s.log.Errorw("Failed to update exchange account",
			"account_id", input.AccountID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to update exchange account")
	}

	s.log.Infow("✅ Exchange account updated successfully",
		"account_id", input.AccountID,
		"exchange", account.Exchange,
	)

	return account, nil
}

// DeleteAccount deletes exchange account
func (s *Service) DeleteAccount(ctx context.Context, accountID uuid.UUID) error {
	s.log.Infow("Deleting exchange account",
		"account_id", accountID,
	)

	// Check account exists
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		s.log.Errorw("Failed to load exchange account for deletion",
			"account_id", accountID,
			"error", err,
		)
		return errors.Wrap(err, "failed to load exchange account")
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, accountID); err != nil {
		s.log.Errorw("Failed to delete exchange account",
			"account_id", accountID,
			"error", err,
		)
		return errors.Wrap(err, "failed to delete exchange account")
	}

	s.log.Infow("✅ Exchange account deleted successfully",
		"account_id", accountID,
		"exchange", account.Exchange,
		"user_id", account.UserID,
	)

	return nil
}

// SetAccountActive activates or deactivates exchange account
func (s *Service) SetAccountActive(ctx context.Context, accountID uuid.UUID, active bool) error {
	s.log.Infow("Setting exchange account active state",
		"account_id", accountID,
		"active", active,
	)

	// Load account
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		s.log.Errorw("Failed to load exchange account",
			"account_id", accountID,
			"error", err,
		)
		return errors.Wrap(err, "failed to load exchange account")
	}

	// Update active state
	account.IsActive = active
	account.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, account); err != nil {
		s.log.Errorw("Failed to update account active state",
			"account_id", accountID,
			"active", active,
			"error", err,
		)
		return errors.Wrap(err, "failed to update account state")
	}

	status := "deactivated"
	if active {
		status = "activated"
	}

	s.log.Infow(fmt.Sprintf("✅ Exchange account %s", status),
		"account_id", accountID,
		"exchange", account.Exchange,
	)

	return nil
}

// SaveListenKey saves a new listenKey (encrypted) to the database
// Called by infrastructure layer after creating listenKey via API
func (s *Service) SaveListenKey(ctx context.Context, accountID uuid.UUID, listenKey string, expiresAt time.Time) error {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to load account")
	}

	// Encrypt and set listenKey
	if err := account.SetListenKey(listenKey, expiresAt, s.encryptor); err != nil {
		return errors.Wrap(err, "failed to encrypt listenKey")
	}

	// Persist
	if err := s.repo.Update(ctx, account); err != nil {
		return errors.Wrap(err, "failed to persist listenKey")
	}

	s.log.Debugw("Saved listenKey to database",
		"account_id", accountID,
		"expires_at", expiresAt,
	)

	return nil
}

// UpdateListenKeyExpiration updates listenKey expiration after successful renewal
// Called by infrastructure layer after actual API renewal (listenKey stays same, only expiration changes)
func (s *Service) UpdateListenKeyExpiration(ctx context.Context, accountID uuid.UUID, expiresAt time.Time) error {
	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		return errors.Wrap(err, "failed to load account")
	}

	// Get current listenKey
	listenKey, err := account.GetListenKey(s.encryptor)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt listenKey")
	}

	// Update expiration (re-encrypt with new expiration)
	if err := account.SetListenKey(listenKey, expiresAt, s.encryptor); err != nil {
		return errors.Wrap(err, "failed to encrypt listenKey")
	}

	// Persist
	if err := s.repo.Update(ctx, account); err != nil {
		return errors.Wrap(err, "failed to persist listenKey expiration")
	}

	s.log.Debugw("Updated listenKey expiration",
		"account_id", accountID,
		"expires_at", expiresAt,
	)

	return nil
}

// DeactivateAccountInput contains data for deactivating an account
type DeactivateAccountInput struct {
	AccountID uuid.UUID
	Reason    string // "invalid_credentials", "permission_error", "connection_error"
	ErrorMsg  string
}

// DeactivateAccount deactivates exchange account and publishes notification event
func (s *Service) DeactivateAccount(ctx context.Context, input DeactivateAccountInput) error {
	s.log.Infow("Deactivating exchange account",
		"account_id", input.AccountID,
		"reason", input.Reason,
	)

	// Load account
	account, err := s.repo.GetByID(ctx, input.AccountID)
	if err != nil {
		s.log.Errorw("Failed to load exchange account for deactivation",
			"account_id", input.AccountID,
			"error", err,
		)
		return errors.Wrap(err, "failed to load exchange account")
	}

	// Deactivate account
	account.IsActive = false
	account.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, account); err != nil {
		s.log.Errorw("Failed to deactivate account",
			"account_id", input.AccountID,
			"error", err,
		)
		return errors.Wrap(err, "failed to deactivate account")
	}

	s.log.Infow("✅ Exchange account deactivated",
		"account_id", input.AccountID,
		"user_id", account.UserID,
		"exchange", account.Exchange,
		"reason", input.Reason,
	)

	// Publish notification to Kafka for Telegram
	if s.notificationPublisher != nil {
		if err := s.notificationPublisher.PublishExchangeDeactivated(
			ctx,
			input.AccountID.String(),
			account.UserID.String(),
			string(account.Exchange),
			account.Label,
			input.Reason,
			input.ErrorMsg,
			account.IsTestnet,
		); err != nil {
			s.log.Errorw("Failed to publish exchange deactivated notification",
				"account_id", input.AccountID,
				"error", err,
			)
			// Don't fail the deactivation if notification publishing fails
		}
	}

	return nil
}

// GetUserAccounts returns all exchange accounts for a user
func (s *Service) GetUserAccounts(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	s.log.Debugw("Getting user exchange accounts",
		"user_id", userID,
	)

	accounts, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		s.log.Errorw("Failed to get user exchange accounts",
			"user_id", userID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to get user accounts")
	}

	s.log.Debugw("Retrieved user exchange accounts",
		"user_id", userID,
		"count", len(accounts),
	)

	return accounts, nil
}

// GetActiveUserAccounts returns only active exchange accounts for a user
func (s *Service) GetActiveUserAccounts(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	s.log.Debugw("Getting active user exchange accounts",
		"user_id", userID,
	)

	accounts, err := s.repo.GetActiveByUser(ctx, userID)
	if err != nil {
		s.log.Errorw("Failed to get active user exchange accounts",
			"user_id", userID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to get active user accounts")
	}

	s.log.Debugw("Retrieved active user exchange accounts",
		"user_id", userID,
		"count", len(accounts),
	)

	return accounts, nil
}

// GetAccount returns exchange account by ID
func (s *Service) GetAccount(ctx context.Context, accountID uuid.UUID) (*exchange_account.ExchangeAccount, error) {
	s.log.Debugw("Getting exchange account",
		"account_id", accountID,
	)

	account, err := s.repo.GetByID(ctx, accountID)
	if err != nil {
		s.log.Errorw("Failed to get exchange account",
			"account_id", accountID,
			"error", err,
		)
		return nil, errors.Wrap(err, "failed to get exchange account")
	}

	return account, nil
}

// FormatAccountSummary returns human-readable account summary
func FormatAccountSummary(account *exchange_account.ExchangeAccount) string {
	statusEmoji := "✅"
	if !account.IsActive {
		statusEmoji = "❌"
	}

	return fmt.Sprintf("%s %s - %s",
		statusEmoji,
		strings.Title(string(account.Exchange)),
		account.Label,
	)
}
