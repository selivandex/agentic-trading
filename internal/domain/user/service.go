package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

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
		return fmt.Errorf("create user: user is nil")
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.TelegramID == 0 {
		return fmt.Errorf("create user: telegram id required")
	}
	if user.Settings == (Settings{}) {
		user.Settings = DefaultSettings()
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetByID fetches a user by UUID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("get user: id is required")
	}
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// GetByTelegramID fetches a user using the Telegram identifier.
func (s *Service) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	if telegramID == 0 {
		return nil, fmt.Errorf("get user by telegram: id is required")
	}
	user, err := s.repo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("get user by telegram: %w", err)
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
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// Update persists user changes.
func (s *Service) Update(ctx context.Context, user *User) error {
	if user == nil {
		return fmt.Errorf("update user: user is nil")
	}
	if user.ID == uuid.Nil {
		return fmt.Errorf("update user: id is required")
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// Delete removes a user by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("delete user: id is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
