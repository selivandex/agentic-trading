package user

import (
	"context"

	"github.com/google/uuid"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service provides business logic for user operations.
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService constructs a user service instance.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, log: logger.Get()}
}

// Create registers a new user with default settings when not provided.
func (s *Service) Create(ctx context.Context, user *User) error {
	if user == nil {
		return errors.ErrInvalidInput
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.TelegramID == 0 {
		return errors.ErrInvalidInput
	}
	if user.Settings.RiskLevel == "" {
		user.Settings = DefaultSettings()
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return errors.Wrap(err, "create user")
	}
	return nil
}

// GetByID fetches a user by UUID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, errors.ErrInvalidInput
	}
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get user")
	}
	return user, nil
}

// GetByTelegramID fetches a user using the Telegram identifier.
func (s *Service) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	if telegramID == 0 {
		return nil, errors.ErrInvalidInput
	}
	user, err := s.repo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, errors.Wrap(err, "get user by telegram")
	}
	return user, nil
}

// List returns a paginated list of users.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*User, error) {
	if limit <= 0 {
		limit = 20
	}
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "list users")
	}
	return users, nil
}

// Update persists user changes.
func (s *Service) Update(ctx context.Context, user *User) error {
	if user == nil {
		return errors.ErrInvalidInput
	}
	if user.ID == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update user")
	}
	return nil
}

// Delete removes a user by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.ErrInvalidInput
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete user")
	}
	return nil
}

// GetOrCreateByTelegramID gets existing user or creates a new one
// Used by Telegram adapter to handle first-time users
func (s *Service) GetOrCreateByTelegramID(ctx context.Context, telegramID int64, firstName, lastName, username, languageCode string) (*User, error) {
	if telegramID == 0 {
		return nil, errors.ErrInvalidInput
	}

	// Try to get existing user
	user, err := s.repo.GetByTelegramID(ctx, telegramID)
	if err == nil {
		s.log.Debugw("User found",
			"user_id", user.ID,
			"telegram_id", telegramID,
		)
		return user, nil
	}

	// User doesn't exist, create new one
	s.log.Infow("Creating new user from Telegram",
		"telegram_id", telegramID,
		"username", username,
	)

	newUser := &User{
		ID:               uuid.New(),
		TelegramID:       telegramID,
		TelegramUsername: username,
		FirstName:        firstName,
		LastName:         lastName,
		LanguageCode:     languageCode,
		IsActive:         true,
		IsPremium:        false,
		Settings:         DefaultSettings(),
	}

	if err := s.repo.Create(ctx, newUser); err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	s.log.Infow("✅ Created new user",
		"user_id", newUser.ID,
		"telegram_id", newUser.TelegramID,
		"username", newUser.TelegramUsername,
	)

	return newUser, nil
}

// SetActive activates or deactivates a user (for /stop command)
func (s *Service) SetActive(ctx context.Context, userID uuid.UUID, active bool) error {
	if userID == uuid.Nil {
		return errors.ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	user.IsActive = active

	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update user active status")
	}

	s.log.Infow("User active status updated",
		"user_id", userID,
		"active", active,
	)

	return nil
}

// UpdateSettings updates user settings
func (s *Service) UpdateSettings(ctx context.Context, userID uuid.UUID, settings Settings) error {
	if userID == uuid.Nil {
		return errors.ErrInvalidInput
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	user.Settings = settings

	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update user settings")
	}

	s.log.Debugw("User settings updated",
		"user_id", userID,
	)

	return nil
}

// ToggleCircuitBreaker toggles circuit breaker setting
func (s *Service) ToggleCircuitBreaker(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	user.Settings.CircuitBreakerOn = !user.Settings.CircuitBreakerOn

	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update circuit breaker setting")
	}

	return nil
}

// ToggleNotifications toggles notifications setting
func (s *Service) ToggleNotifications(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	user.Settings.NotificationsOn = !user.Settings.NotificationsOn

	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update notifications setting")
	}

	return nil
}

// UpdateRiskLevel updates user risk level
func (s *Service) UpdateRiskLevel(ctx context.Context, userID uuid.UUID, riskLevel string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	user.Settings.RiskLevel = riskLevel

	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update risk level")
	}

	return nil
}

// UpdateMaxPositions updates max positions setting
func (s *Service) UpdateMaxPositions(ctx context.Context, userID uuid.UUID, maxPositions int) error {
	if maxPositions < 1 || maxPositions > 10 {
		return errors.Wrapf(errors.ErrInvalidInput, "maxPositions must be between 1 and 10, got %d", maxPositions)
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	user.Settings.MaxPositions = maxPositions

	if err := s.repo.Update(ctx, user); err != nil {
		return errors.Wrap(err, "update max positions")
	}

	return nil
}
