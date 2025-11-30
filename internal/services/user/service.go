package user

import (
	"context"

	"github.com/google/uuid"

	"prometheus/internal/domain/user"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service handles user business logic
// Provides abstraction over user repository for consumers and other services
type Service struct {
	repository user.Repository
	log        *logger.Logger
}

// NewService creates a new user service
func NewService(
	repository user.Repository,
	log *logger.Logger,
) *Service {
	return &Service{
		repository: repository,
		log:        log,
	}
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	usr, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user")
	}
	return usr, nil
}

// GetActiveUsers returns all active users
func (s *Service) GetActiveUsers(ctx context.Context) ([]*user.User, error) {
	// TODO: Add filtering logic if needed
	users, err := s.repository.List(ctx, 1000, 0) // Get up to 1000 users
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
	// Parse user ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return 0, errors.Wrap(err, "invalid user_id format")
	}

	// Get user from repository
	usr, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return 0, errors.Wrap(err, "failed to get user")
	}

	// Check if notifications are enabled
	if !usr.Settings.NotificationsOn {
		s.log.Debugw("Notifications disabled for user", "user_id", userID)
		return 0, errors.New("notifications disabled")
	}

	return usr.TelegramID, nil
}

// IsUserActiveWithCircuitBreaker checks if user is active and has circuit breaker enabled
// Used by opportunity consumer to filter users before running workflows
func (s *Service) IsUserActiveWithCircuitBreaker(ctx context.Context, userID uuid.UUID) (bool, error) {
	usr, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return false, errors.Wrap(err, "failed to get user")
	}

	return usr.IsActive && usr.Settings.CircuitBreakerOn, nil
}
