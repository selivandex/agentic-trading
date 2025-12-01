package telegram

import (
	"context"

	"prometheus/internal/domain/user"
	userservice "prometheus/internal/services/user"
	"prometheus/pkg/telegram"
)

// UserServiceAdapter adapts application user.Service to telegram handler requirements
// Uses application service (not domain) to get side effects like notifications and events
type UserServiceAdapter struct {
	service *userservice.Service
}

// NewUserServiceAdapter creates adapter for application user service
func NewUserServiceAdapter(service *userservice.Service) *UserServiceAdapter {
	return &UserServiceAdapter{
		service: service,
	}
}

// GetByTelegramID retrieves user by telegram ID
func (a *UserServiceAdapter) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	return a.service.GetByTelegramID(ctx, telegramID)
}

// CreateFromTelegram creates user from telegram user data
// Uses application service which:
// - Delegates to domain service for business logic (GetOrCreateByTelegramID)
// - Adds side effects: welcome notifications, events to Kafka
func (a *UserServiceAdapter) CreateFromTelegram(ctx context.Context, tgUser *telegram.User) (*user.User, error) {
	// Application service → Domain service → Repository
	// Domain handles: existence check, defaults, limit profile, race conditions
	// Application adds: notifications, events
	return a.service.GetOrCreateByTelegramID(
		ctx,
		tgUser.ID,
		tgUser.FirstName,
		tgUser.LastName,
		tgUser.Username,
		"", // language_code - can add to telegram.User if needed
	)
}
