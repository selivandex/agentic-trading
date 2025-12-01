package telegram

import (
	"context"
	"time"

	"prometheus/internal/domain/menu_session"
	svc "prometheus/internal/services/menu_session"
	"prometheus/pkg/telegram"
)

// SessionServiceAdapter adapts menu_session.Service to telegram.SessionService interface
type SessionServiceAdapter struct {
	service *svc.Service
}

// NewSessionServiceAdapter creates adapter wrapping menu_session.Service
func NewSessionServiceAdapter(service *svc.Service) *SessionServiceAdapter {
	return &SessionServiceAdapter{
		service: service,
	}
}

// GetSession retrieves session for telegram user
func (a *SessionServiceAdapter) GetSession(ctx context.Context, telegramID int64) (telegram.Session, error) {
	return a.service.GetSession(ctx, telegramID)
}

// SaveSession saves session with TTL
func (a *SessionServiceAdapter) SaveSession(ctx context.Context, session telegram.Session, ttl time.Duration) error {
	domainSession, ok := session.(*menu_session.Session)
	if !ok {
		return a.service.SaveSession(ctx, &menu_session.Session{
			TelegramID:      session.GetTelegramID(),
			MessageID:       session.GetMessageID(),
			CurrentScreen:   session.GetCurrentScreen(),
			NavigationStack: make([]string, 0),
			Data:            make(map[string]interface{}),
			CreatedAt:       session.GetCreatedAt(),
			UpdatedAt:       session.GetUpdatedAt(),
		}, ttl)
	}

	return a.service.SaveSession(ctx, domainSession, ttl)
}

// DeleteSession removes session
func (a *SessionServiceAdapter) DeleteSession(ctx context.Context, telegramID int64) error {
	return a.service.DeleteSession(ctx, telegramID)
}

// SessionExists checks if session exists
func (a *SessionServiceAdapter) SessionExists(ctx context.Context, telegramID int64) (bool, error) {
	return a.service.SessionExists(ctx, telegramID)
}

// CreateSession creates new session
func (a *SessionServiceAdapter) CreateSession(ctx context.Context, telegramID int64, initialScreen string, initialData map[string]interface{}, ttl time.Duration) (telegram.Session, error) {
	return a.service.CreateSession(ctx, telegramID, initialScreen, initialData, ttl)
}

// Verify adapter implements interface at compile time
var _ telegram.SessionService = (*SessionServiceAdapter)(nil)
