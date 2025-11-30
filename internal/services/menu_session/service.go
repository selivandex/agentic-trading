package menu_session

import (
	"context"
	"time"

	"prometheus/internal/domain/menu_session"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service provides business logic for menu sessions
type Service struct {
	repo menu_session.Repository
	log  *logger.Logger
}

// NewService creates a new menu session service
func NewService(repo menu_session.Repository, log *logger.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log.With("service", "menu_session"),
	}
}

// GetSession retrieves a session by telegram ID
func (s *Service) GetSession(ctx context.Context, telegramID int64) (*menu_session.Session, error) {
	session, err := s.repo.Get(ctx, telegramID)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			s.log.Debugw("Menu session not found",
				"telegram_id", telegramID,
			)
		} else {
			s.log.Errorw("Failed to get menu session",
				"telegram_id", telegramID,
				"error", err,
			)
		}
		return nil, err
	}

	return session, nil
}

// SaveSession stores a session with TTL
func (s *Service) SaveSession(ctx context.Context, session *menu_session.Session, ttl time.Duration) error {
	if err := s.repo.Save(ctx, session, ttl); err != nil {
		s.log.Errorw("Failed to save menu session",
			"telegram_id", session.TelegramID,
			"error", err,
		)
		return err
	}

	s.log.Debugw("Menu session saved",
		"telegram_id", session.TelegramID,
		"screen", session.CurrentScreen,
	)

	return nil
}

// DeleteSession removes a session
func (s *Service) DeleteSession(ctx context.Context, telegramID int64) error {
	if err := s.repo.Delete(ctx, telegramID); err != nil {
		s.log.Errorw("Failed to delete menu session",
			"telegram_id", telegramID,
			"error", err,
		)
		return err
	}

	s.log.Debugw("Menu session deleted",
		"telegram_id", telegramID,
	)

	return nil
}

// SessionExists checks if session exists
func (s *Service) SessionExists(ctx context.Context, telegramID int64) (bool, error) {
	exists, err := s.repo.Exists(ctx, telegramID)
	if err != nil {
		s.log.Errorw("Failed to check menu session existence",
			"telegram_id", telegramID,
			"error", err,
		)
		return false, err
	}

	return exists, nil
}

// CreateSession creates a new session
func (s *Service) CreateSession(ctx context.Context, telegramID int64, initialScreen string, initialData map[string]interface{}, ttl time.Duration) (*menu_session.Session, error) {
	session := menu_session.NewSession(telegramID, initialScreen, initialData)

	if err := s.SaveSession(ctx, session, ttl); err != nil {
		return nil, errors.Wrap(err, "failed to create menu session")
	}

	s.log.Infow("Menu session created",
		"telegram_id", telegramID,
		"screen", initialScreen,
	)

	return session, nil
}

// UpdateSessionData updates session data fields
func (s *Service) UpdateSessionData(ctx context.Context, telegramID int64, key string, value interface{}, ttl time.Duration) error {
	session, err := s.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "failed to get session for update")
	}

	session.SetData(key, value)

	return s.SaveSession(ctx, session, ttl)
}

// NavigateTo pushes current screen to stack and sets new screen
func (s *Service) NavigateTo(ctx context.Context, telegramID int64, screenID string, ttl time.Duration) error {
	session, err := s.GetSession(ctx, telegramID)
	if err != nil {
		return errors.Wrap(err, "failed to get session for navigation")
	}

	session.PushScreen(screenID)

	return s.SaveSession(ctx, session, ttl)
}

// NavigateBack returns to previous screen
func (s *Service) NavigateBack(ctx context.Context, telegramID int64, ttl time.Duration) (previousScreen string, err error) {
	session, err := s.GetSession(ctx, telegramID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get session for back navigation")
	}

	prevScreen, ok := session.PopScreen()
	if !ok {
		return "", errors.New("navigation stack is empty")
	}

	if err := s.SaveSession(ctx, session, ttl); err != nil {
		return "", errors.Wrap(err, "failed to save session after back navigation")
	}

	return prevScreen, nil
}

