package adk

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/session"

	domainsession "prometheus/internal/domain/session"
	"prometheus/pkg/errors"
)

// TestSessionService_Create tests session creation with ADK adapter
func TestSessionService_Create(t *testing.T) {
	// Use mock repository
	repo := newMockSessionRepo()
	domainService := domainsession.NewService(repo)
	adkService := NewSessionService(domainService)

	ctx := context.Background()

	// Test creating a session
	req := &session.CreateRequest{
		AppName:   "test_app",
		UserID:    "user123",
		SessionID: "session456",
		State: map[string]interface{}{
			"key1":               "value1",
			"app:global_setting": true,
			"user:preference":    "dark_mode",
		},
	}

	resp, err := adkService.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Session)

	assert.Equal(t, "test_app", resp.Session.AppName())
	assert.Equal(t, "user123", resp.Session.UserID())
	assert.Equal(t, "session456", resp.Session.ID())
}

// TestSessionService_Adapter tests the adapter pattern
func TestSessionService_Adapter(t *testing.T) {
	// Test that adapter correctly implements session.Service interface
	repo := newMockSessionRepo()
	domainService := domainsession.NewService(repo)
	adkService := NewSessionService(domainService)

	// Verify it implements the interface
	var _ session.Service = adkService
}

// Mock repository for unit testing
type mockSessionRepo struct {
	sessions  map[string]*domainsession.Session
	appState  map[string]*domainsession.AppState
	userState map[string]*domainsession.UserState
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{
		sessions:  make(map[string]*domainsession.Session),
		appState:  make(map[string]*domainsession.AppState),
		userState: make(map[string]*domainsession.UserState),
	}
}

func (m *mockSessionRepo) Create(ctx context.Context, sess *domainsession.Session) error {
	key := sess.AppName + ":" + sess.UserID + ":" + sess.SessionID
	m.sessions[key] = sess
	return nil
}

func (m *mockSessionRepo) Get(ctx context.Context, appName, userID, sessionID string, opts *domainsession.GetOptions) (*domainsession.Session, error) {
	key := appName + ":" + userID + ":" + sessionID
	if sess, ok := m.sessions[key]; ok {
		return sess, nil
	}
	return nil, errors.ErrNotFound
}

func (m *mockSessionRepo) List(ctx context.Context, appName, userID string) ([]*domainsession.Session, error) {
	var sessions []*domainsession.Session
	for _, sess := range m.sessions {
		if sess.AppName == appName && (userID == "" || sess.UserID == userID) {
			sessions = append(sessions, sess)
		}
	}
	return sessions, nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, appName, userID, sessionID string) error {
	key := appName + ":" + userID + ":" + sessionID
	delete(m.sessions, key)
	return nil
}

func (m *mockSessionRepo) UpdateState(ctx context.Context, appName, userID, sessionID string, state map[string]interface{}) error {
	key := appName + ":" + userID + ":" + sessionID
	if sess, ok := m.sessions[key]; ok {
		sess.State = state
	}
	return nil
}

func (m *mockSessionRepo) AppendEvent(ctx context.Context, sessionUUID uuid.UUID, event *domainsession.Event) error {
	return nil
}

func (m *mockSessionRepo) GetEvents(ctx context.Context, sessionUUID uuid.UUID, opts *domainsession.GetEventsOptions) ([]*domainsession.Event, error) {
	return []*domainsession.Event{}, nil
}

func (m *mockSessionRepo) GetAppState(ctx context.Context, appName string) (*domainsession.AppState, error) {
	if state, ok := m.appState[appName]; ok {
		return state, nil
	}
	return nil, errors.ErrNotFound
}

func (m *mockSessionRepo) SetAppState(ctx context.Context, appName string, state map[string]interface{}) error {
	m.appState[appName] = &domainsession.AppState{
		AppName: appName,
		State:   state,
	}
	return nil
}

func (m *mockSessionRepo) GetUserState(ctx context.Context, appName, userID string) (*domainsession.UserState, error) {
	key := appName + ":" + userID
	if state, ok := m.userState[key]; ok {
		return state, nil
	}
	return nil, errors.ErrNotFound
}

func (m *mockSessionRepo) SetUserState(ctx context.Context, appName, userID string, state map[string]interface{}) error {
	key := appName + ":" + userID
	m.userState[key] = &domainsession.UserState{
		AppName: appName,
		UserID:  userID,
		State:   state,
	}
	return nil
}
