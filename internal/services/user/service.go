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
func (s *Service) SetActive(ctx context.Context, userID uuid.UUID, active bool) error {
	if err := s.domainService.SetActive(ctx, userID, active); err != nil {
		return errors.Wrap(err, "failed to set user active status")
	}

	// TODO: Send notification about account status change
	// TODO: Publish user.activated or user.deactivated event

	return nil
}

// UpdateSettings updates user settings (with events)
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, settings user.Settings) error {
	if err := s.domainService.UpdateSettings(ctx, userID, settings); err != nil {
		return errors.Wrap(err, "failed to update settings")
	}

	// TODO: Publish settings.updated event to Kafka

	return nil
}
