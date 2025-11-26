package session

import (
	"context"
	"time"

	"github.com/google/uuid"

	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// Service provides business logic for session management
type Service struct {
	repo Repository
	log  *logger.Logger
}

// NewService creates a new session service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		log:  logger.Get().With("component", "session_service"),
	}
}

// CreateSession creates a new session with initial state
func (s *Service) CreateSession(ctx context.Context, appName, userID, sessionID string, initialState map[string]interface{}) (*Session, error) {
	if appName == "" || userID == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "app_name and user_id are required")
	}

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Process initial state to extract app/user/session level state
	appDelta, userDelta, sessionState := s.extractStateDeltas(initialState)

	// Update app state if needed
	if len(appDelta) > 0 {
		if err := s.updateAppState(ctx, appName, appDelta); err != nil {
			return nil, errors.Wrap(err, "failed to update app state")
		}
	}

	// Update user state if needed
	if len(userDelta) > 0 {
		if err := s.updateUserState(ctx, appName, userID, userDelta); err != nil {
			return nil, errors.Wrap(err, "failed to update user state")
		}
	}

	session := &Session{
		ID:        uuid.New(),
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
		State:     sessionState,
		Events:    []Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, errors.Wrap(err, "failed to create session")
	}

	s.log.Infof("Created session: app=%s user=%s session=%s", appName, userID, sessionID)
	return session, nil
}

// GetSession retrieves a session with its events
func (s *Service) GetSession(ctx context.Context, appName, userID, sessionID string, opts *GetOptions) (*Session, error) {
	if appName == "" || userID == "" || sessionID == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "app_name, user_id, and session_id are required")
	}

	session, err := s.repo.Get(ctx, appName, userID, sessionID, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get session")
	}

	// Merge app and user state into session state
	if err := s.mergeStates(ctx, session); err != nil {
		return nil, errors.Wrap(err, "failed to merge states")
	}

	return session, nil
}

// ListSessions lists all sessions for a user
func (s *Service) ListSessions(ctx context.Context, appName, userID string) ([]*Session, error) {
	if appName == "" {
		return nil, errors.Wrap(errors.ErrInvalidInput, "app_name is required")
	}

	sessions, err := s.repo.List(ctx, appName, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list sessions")
	}

	// Merge states for all sessions
	for _, session := range sessions {
		if err := s.mergeStates(ctx, session); err != nil {
			s.log.Warnf("Failed to merge states for session %s: %v", session.SessionID, err)
		}
	}

	return sessions, nil
}

// DeleteSession deletes a session
func (s *Service) DeleteSession(ctx context.Context, appName, userID, sessionID string) error {
	if appName == "" || userID == "" || sessionID == "" {
		return errors.Wrap(errors.ErrInvalidInput, "app_name, user_id, and session_id are required")
	}

	if err := s.repo.Delete(ctx, appName, userID, sessionID); err != nil {
		return errors.Wrap(err, "failed to delete session")
	}

	s.log.Infof("Deleted session: app=%s user=%s session=%s", appName, userID, sessionID)
	return nil
}

// AppendEvent appends an event to a session
func (s *Service) AppendEvent(ctx context.Context, session *Session, event *Event) error {
	if session == nil || event == nil {
		return errors.Wrap(errors.ErrInvalidInput, "session and event are required")
	}

	// Skip partial events for persistence
	if event.Partial {
		return nil
	}

	// Process state delta from event actions
	if len(event.Actions.StateDelta) > 0 {
		appDelta, userDelta, sessionDelta := s.extractStateDeltas(event.Actions.StateDelta)

		// Update app state
		if len(appDelta) > 0 {
			if err := s.updateAppState(ctx, session.AppName, appDelta); err != nil {
				return errors.Wrap(err, "failed to update app state")
			}
		}

		// Update user state
		if len(userDelta) > 0 {
			if err := s.updateUserState(ctx, session.AppName, session.UserID, userDelta); err != nil {
				return errors.Wrap(err, "failed to update user state")
			}
		}

		// Update session state
		if len(sessionDelta) > 0 {
			for k, v := range sessionDelta {
				session.State[k] = v
			}
			if err := s.repo.UpdateState(ctx, session.AppName, session.UserID, session.SessionID, session.State); err != nil {
				return errors.Wrap(err, "failed to update session state")
			}
		}
	}

	// Append event to repository
	if err := s.repo.AppendEvent(ctx, session.ID, event); err != nil {
		return errors.Wrap(err, "failed to append event")
	}

	session.Events = append(session.Events, *event)
	session.UpdatedAt = time.Now()

	return nil
}

// extractStateDeltas splits state map into app/user/session level deltas
func (s *Service) extractStateDeltas(state map[string]interface{}) (app, user, session map[string]interface{}) {
	app = make(map[string]interface{})
	user = make(map[string]interface{})
	session = make(map[string]interface{})

	for key, value := range state {
		if len(key) > len(KeyPrefixApp) && key[:len(KeyPrefixApp)] == KeyPrefixApp {
			// App-level state
			cleanKey := key[len(KeyPrefixApp):]
			app[cleanKey] = value
		} else if len(key) > len(KeyPrefixUser) && key[:len(KeyPrefixUser)] == KeyPrefixUser {
			// User-level state
			cleanKey := key[len(KeyPrefixUser):]
			user[cleanKey] = value
		} else if len(key) > len(KeyPrefixTemp) && key[:len(KeyPrefixTemp)] == KeyPrefixTemp {
			// Temporary state - skip persistence
			continue
		} else {
			// Session-level state
			session[key] = value
		}
	}

	return app, user, session
}

// mergeStates merges app and user state into session state with proper prefixes
func (s *Service) mergeStates(ctx context.Context, session *Session) error {
	// Get app state
	appState, err := s.repo.GetAppState(ctx, session.AppName)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return errors.Wrap(err, "failed to get app state")
	}

	// Get user state
	userState, err := s.repo.GetUserState(ctx, session.AppName, session.UserID)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return errors.Wrap(err, "failed to get user state")
	}

	// Create merged state starting with session state
	merged := make(map[string]interface{})
	for k, v := range session.State {
		merged[k] = v
	}

	// Add app state with prefix
	if appState != nil {
		for k, v := range appState.State {
			merged[KeyPrefixApp+k] = v
		}
	}

	// Add user state with prefix
	if userState != nil {
		for k, v := range userState.State {
			merged[KeyPrefixUser+k] = v
		}
	}

	session.State = merged
	return nil
}

// updateAppState updates application-level state
func (s *Service) updateAppState(ctx context.Context, appName string, delta map[string]interface{}) error {
	appState, err := s.repo.GetAppState(ctx, appName)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return err
	}

	if appState == nil {
		appState = &AppState{
			AppName: appName,
			State:   make(map[string]interface{}),
		}
	}

	// Merge delta into existing state
	for k, v := range delta {
		appState.State[k] = v
	}

	return s.repo.SetAppState(ctx, appName, appState.State)
}

// updateUserState updates user-level state
func (s *Service) updateUserState(ctx context.Context, appName, userID string, delta map[string]interface{}) error {
	userState, err := s.repo.GetUserState(ctx, appName, userID)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return err
	}

	if userState == nil {
		userState = &UserState{
			AppName: appName,
			UserID:  userID,
			State:   make(map[string]interface{}),
		}
	}

	// Merge delta into existing state
	for k, v := range delta {
		userState.State[k] = v
	}

	return s.repo.SetUserState(ctx, appName, userID, userState.State)
}
