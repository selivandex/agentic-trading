package user

import (
	"context"

	"github.com/google/uuid"

	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles user business logic (Application Service)
// Coordinates domain service + adds side effects (notifications, events)
type Service struct {
	domainService *user.Service // Domain service does all the heavy lifting
	log           *logger.Logger
	// TODO: Add event publisher for user.created, user.updated events
	// TODO: Add notification service for welcome messages
}

// NewService creates a new user application service
func NewService(
	domainService *user.Service,
	log *logger.Logger,
) *Service {
	return &Service{
		domainService: domainService,
		log:           log.With("service", "user_application"),
	}
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	return s.domainService.GetByID(ctx, userID)
}

// GetByTelegramID retrieves a user by Telegram ID
func (s *Service) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	return s.domainService.GetByTelegramID(ctx, telegramID)
}

// Create creates a new user (with side effects like notifications)
func (s *Service) Create(ctx context.Context, usr *user.User) error {
	// Domain logic handles validation, defaults, repository
	if err := s.domainService.Create(ctx, usr); err != nil {
		return errors.Wrap(err, "failed to create user")
	}

	s.log.Infow("User created",
		"user_id", usr.ID,
		"telegram_id", usr.TelegramID,
		"username", usr.TelegramUsername,
	)

	// TODO: Send welcome notification via Kafka
	// TODO: Publish user.created event to Kafka
	// TODO: Initialize default settings/data

	return nil
}

// GetOrCreateByTelegramID gets existing user or creates a new one
// This is used by Telegram adapter for onboarding
func (s *Service) GetOrCreateByTelegramID(ctx context.Context, telegramID int64, firstName, lastName, username, languageCode string) (*user.User, error) {
	// Check if this is a new user creation (before calling domain service)
	_, err := s.domainService.GetByTelegramID(ctx, telegramID)
	isNewUser := err != nil

	// Domain service handles all the heavy lifting:
	// - Check if user exists
	// - Create with defaults if not
	// - Assign limit profile (free tier)
	// - Handle race conditions
	usr, err := s.domainService.GetOrCreateByTelegramID(ctx, telegramID, firstName, lastName, username, languageCode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get or create user")
	}

	// Send welcome message only for NEW users
	if isNewUser {
		s.log.Infow("New user onboarded",
			"user_id", usr.ID,
			"telegram_id", telegramID,
			"username", username,
		)
		// TODO: Send welcome notification via Kafka
		// TODO: Publish user.onboarded event to Kafka
	}

	return usr, nil
}

// GetActiveUsers returns all active users (for bulk operations)
func (s *Service) GetActiveUsers(ctx context.Context) ([]*user.User, error) {
	users, err := s.domainService.List(ctx, 1000, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get users")
	}

	// Filter active users
	activeUsers := make([]*user.User, 0, len(users))
	for _, u := range users {
		if u.IsActive {
			activeUsers = append(activeUsers, u)
		}
	}

	return activeUsers, nil
}

// GetUserChatID retrieves Telegram chat ID for notifications
// Returns error if user not found or notifications disabled
func (s *Service) GetUserChatID(ctx context.Context, userIDStr string) (int64, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return 0, errors.Wrap(err, "invalid user_id format")
	}

	usr, err := s.domainService.GetByID(ctx, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get user")
	}

	if !usr.Settings.NotificationsOn {
		s.log.Debugw("Notifications disabled for user", "user_id", userID)
		return 0, errors.New("notifications disabled")
	}

	if usr.TelegramID == nil {
		return 0, errors.New("user registered via email, no telegram ID")
	}

	return *usr.TelegramID, nil
}

// IsUserActiveWithCircuitBreaker checks if user is active and has circuit breaker enabled
func (s *Service) IsUserActiveWithCircuitBreaker(ctx context.Context, userID uuid.UUID) (bool, error) {
	usr, err := s.domainService.GetByID(ctx, userID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get user")
	}

	return usr.IsActive && usr.Settings.CircuitBreakerOn, nil
}

// Update updates user data (with events)
func (s *Service) Update(ctx context.Context, usr *user.User) error {
	if err := s.domainService.Update(ctx, usr); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	s.log.Debugw("User updated", "user_id", usr.ID)

	// TODO: Publish user.updated event to Kafka

	return nil
}

// SetActive activates or deactivates a user (with notifications)
func (s *Service) SetActive(ctx context.Context, userID uuid.UUID, active bool) (*user.User, error) {
	s.log.Infow("Setting user active status",
		"user_id", userID,
		"active", active,
	)

	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get user via domain service
	usr, err := s.domainService.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}

	usr.IsActive = active

	// Update via domain service
	if err := s.domainService.Update(ctx, usr); err != nil {
		return nil, errors.Wrap(err, "failed to set user active status")
	}

	s.log.Infow("User active status updated",
		"user_id", userID,
		"active", active,
	)

	// TODO: Send notification about account status change
	// TODO: Publish user.activated or user.deactivated event

	return usr, nil
}

// UpdateSettingsParams contains parameters for updating user settings
type UpdateSettingsParams struct {
	DefaultAIProvider   *string
	DefaultAIModel      *string
	RiskLevel           *string
	MaxPositions        *int
	MaxPortfolioRisk    *float64
	MaxDailyDrawdown    *float64
	MaxConsecutiveLoss  *int
	NotificationsOn     *bool
	DailyReportTime     *string
	Timezone            *string
	CircuitBreakerOn    *bool
	MaxPositionSizeUSD  *float64
	MaxTotalExposureUSD *float64
	MinPositionSizeUSD  *float64
	MaxLeverageMultiple *float64
	AllowedExchanges    []string
}

// UpdateSettings updates user settings with partial updates (admin function)
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, params UpdateSettingsParams) (*user.User, error) {
	s.log.Infow("Updating user settings", "user_id", userID)

	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get existing user via domain service
	usr, err := s.domainService.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}

	// Apply partial updates to settings
	if params.DefaultAIProvider != nil {
		usr.Settings.DefaultAIProvider = *params.DefaultAIProvider
	}
	if params.DefaultAIModel != nil {
		usr.Settings.DefaultAIModel = *params.DefaultAIModel
	}
	if params.RiskLevel != nil {
		usr.Settings.RiskLevel = *params.RiskLevel
	}
	if params.MaxPositions != nil {
		usr.Settings.MaxPositions = *params.MaxPositions
	}
	if params.MaxPortfolioRisk != nil {
		usr.Settings.MaxPortfolioRisk = *params.MaxPortfolioRisk
	}
	if params.MaxDailyDrawdown != nil {
		usr.Settings.MaxDailyDrawdown = *params.MaxDailyDrawdown
	}
	if params.MaxConsecutiveLoss != nil {
		usr.Settings.MaxConsecutiveLoss = *params.MaxConsecutiveLoss
	}
	if params.NotificationsOn != nil {
		usr.Settings.NotificationsOn = *params.NotificationsOn
	}
	if params.DailyReportTime != nil {
		usr.Settings.DailyReportTime = *params.DailyReportTime
	}
	if params.Timezone != nil {
		usr.Settings.Timezone = *params.Timezone
	}
	if params.CircuitBreakerOn != nil {
		usr.Settings.CircuitBreakerOn = *params.CircuitBreakerOn
	}
	if params.MaxPositionSizeUSD != nil {
		usr.Settings.MaxPositionSizeUSD = *params.MaxPositionSizeUSD
	}
	if params.MaxTotalExposureUSD != nil {
		usr.Settings.MaxTotalExposureUSD = *params.MaxTotalExposureUSD
	}
	if params.MinPositionSizeUSD != nil {
		usr.Settings.MinPositionSizeUSD = *params.MinPositionSizeUSD
	}
	if params.MaxLeverageMultiple != nil {
		usr.Settings.MaxLeverageMultiple = *params.MaxLeverageMultiple
	}
	if params.AllowedExchanges != nil {
		usr.Settings.AllowedExchanges = params.AllowedExchanges
	}

	// Update via domain service
	if err := s.domainService.Update(ctx, usr); err != nil {
		return nil, errors.Wrap(err, "failed to update settings")
	}

	s.log.Infow("User settings updated successfully", "user_id", userID)

	// TODO: Publish settings.updated event to Kafka

	return usr, nil
}

// CreateUserParams contains parameters for creating a new user (admin)
type CreateUserParams struct {
	TelegramID       *int64
	TelegramUsername string
	Email            *string
	PasswordHash     *string
	FirstName        string
	LastName         string
	LanguageCode     string
	IsActive         bool
	IsPremium        bool
	LimitProfileID   *uuid.UUID
	Settings         *user.Settings
}

// CreateUser creates a new user (admin function)
func (s *Service) CreateUser(ctx context.Context, params CreateUserParams) (*user.User, error) {
	s.log.Infow("Creating new user (admin)",
		"telegram_id", params.TelegramID,
		"email", params.Email,
	)

	// Validate input
	if (params.TelegramID == nil || *params.TelegramID == 0) && (params.Email == nil || *params.Email == "") {
		return nil, errors.New("user must have either telegram_id or email")
	}

	// Use default settings if not provided
	settings := user.DefaultSettings()
	if params.Settings != nil {
		settings = *params.Settings
	}

	newUser := &user.User{
		ID:               uuid.New(),
		TelegramID:       params.TelegramID,
		TelegramUsername: params.TelegramUsername,
		Email:            params.Email,
		PasswordHash:     params.PasswordHash,
		FirstName:        params.FirstName,
		LastName:         params.LastName,
		LanguageCode:     params.LanguageCode,
		IsActive:         params.IsActive,
		IsPremium:        params.IsPremium,
		LimitProfileID:   params.LimitProfileID,
		Settings:         settings,
	}

	// Use domain service to create
	if err := s.domainService.Create(ctx, newUser); err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	s.log.Infow("User created successfully (admin)",
		"user_id", newUser.ID,
		"telegram_id", newUser.TelegramID,
		"email", newUser.Email,
	)

	// TODO: Publish user.created event to Kafka

	return newUser, nil
}

// UpdateUserParams contains parameters for updating a user (admin)
type UpdateUserParams struct {
	TelegramID       *int64
	TelegramUsername *string
	Email            *string
	FirstName        *string
	LastName         *string
	LanguageCode     *string
	IsPremium        *bool
	LimitProfileID   *uuid.UUID
}

// UpdateUser updates user data (admin function)
func (s *Service) UpdateUser(ctx context.Context, userID uuid.UUID, params UpdateUserParams) (*user.User, error) {
	s.log.Infow("Updating user (admin)", "user_id", userID)

	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get existing user via domain service
	usr, err := s.domainService.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}

	// Apply updates (only non-nil fields)
	if params.TelegramID != nil {
		usr.TelegramID = params.TelegramID
	}
	if params.TelegramUsername != nil {
		usr.TelegramUsername = *params.TelegramUsername
	}
	if params.Email != nil {
		usr.Email = params.Email
	}
	if params.FirstName != nil {
		usr.FirstName = *params.FirstName
	}
	if params.LastName != nil {
		usr.LastName = *params.LastName
	}
	if params.LanguageCode != nil {
		usr.LanguageCode = *params.LanguageCode
	}
	if params.IsPremium != nil {
		usr.IsPremium = *params.IsPremium
	}
	if params.LimitProfileID != nil {
		usr.LimitProfileID = params.LimitProfileID
	}

	// Use domain service to update
	if err := s.domainService.Update(ctx, usr); err != nil {
		return nil, errors.Wrap(err, "failed to update user")
	}

	s.log.Infow("User updated successfully (admin)", "user_id", userID)

	// TODO: Publish user.updated event to Kafka

	return usr, nil
}

// DeleteUser permanently deletes a user (admin function, hard delete)
func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	s.log.Warnw("Deleting user (admin, hard delete)", "user_id", userID)

	if userID == uuid.Nil {
		return errors.ErrInvalidInput
	}

	// Use domain service to delete
	if err := s.domainService.Delete(ctx, userID); err != nil {
		return errors.Wrap(err, "failed to delete user")
	}

	s.log.Infow("User deleted successfully (admin)", "user_id", userID)

	// TODO: Publish user.deleted event to Kafka

	return nil
}

// BatchDeleteUsers deletes multiple users in batch (admin function, hard delete)
// Returns the number of successfully deleted users
func (s *Service) BatchDeleteUsers(ctx context.Context, userIDs []uuid.UUID) (int, error) {
	s.log.Warnw("Batch deleting users (admin, hard delete)",
		"count", len(userIDs),
	)

	if len(userIDs) == 0 {
		return 0, errors.ErrInvalidInput
	}

	deleted := 0
	var lastErr error

	for _, userID := range userIDs {
		if err := s.DeleteUser(ctx, userID); err != nil {
			s.log.Warnw("Failed to delete user in batch",
				"user_id", userID,
				"error", err,
			)
			lastErr = err
			continue
		}
		deleted++
	}

	s.log.Infow("Batch delete completed (admin)",
		"deleted", deleted,
		"total", len(userIDs),
	)

	// Return error only if all deletions failed
	if deleted == 0 && lastErr != nil {
		return 0, lastErr
	}

	// TODO: Publish batch user.deleted events to Kafka

	return deleted, nil
}

// SetPremium sets premium status for a user (admin function)
func (s *Service) SetPremium(ctx context.Context, userID uuid.UUID, isPremium bool) (*user.User, error) {
	s.log.Infow("Setting user premium status (admin)",
		"user_id", userID,
		"is_premium", isPremium,
	)

	if userID == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}

	// Get user via domain service
	usr, err := s.domainService.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}

	usr.IsPremium = isPremium

	// Update via domain service
	if err := s.domainService.Update(ctx, usr); err != nil {
		return nil, errors.Wrap(err, "failed to update premium status")
	}

	s.log.Infow("User premium status updated (admin)",
		"user_id", userID,
		"is_premium", isPremium,
	)

	// TODO: Send notification about premium status change
	// TODO: Publish user.premium_updated event to Kafka

	return usr, nil
}

// List returns paginated list of users (admin function)
func (s *Service) List(ctx context.Context, limit, offset int) ([]*user.User, error) {
	return s.domainService.List(ctx, limit, offset)
}

// GetUsersWithScope returns users filtered by scope, search and filters
// Scope examples: "all", "active", "inactive", "premium", "free"
// Delegates to repository layer for SQL-based filtering
func (s *Service) GetUsersWithScope(ctx context.Context, scopeID *string, search *string, filters map[string]interface{}) ([]*user.User, error) {
	s.log.Infow("Getting users with scope",
		"scope", scopeID,
		"search", search,
		"has_filters", len(filters) > 0,
	)

	// Delegate to repository layer
	users, err := s.domainService.GetUsersWithScope(ctx, scopeID, search, filters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get users with scope")
	}

	s.log.Infow("Users retrieved",
		"count", len(users),
	)

	return users, nil
}

// GetUsersScopes returns counts for each scope
// Uses SQL GROUP BY for efficiency
func (s *Service) GetUsersScopes(ctx context.Context) (map[string]int, error) {
	s.log.Infow("Getting users scopes")

	// Delegate to repository layer
	scopes, err := s.domainService.GetUsersScopes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get users scopes")
	}

	s.log.Infow("Users scopes retrieved",
		"scopes", scopes,
	)

	return scopes, nil
}
