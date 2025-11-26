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
	if user.Settings == (Settings{}) {
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
