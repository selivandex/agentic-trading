package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/session"
	"prometheus/internal/testsupport"
)

func TestSessionRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State: map[string]interface{}{
			"current_mode":    "analysis",
			"last_symbol":     "BTC/USDT",
			"conversation_id": "conv789",
		},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, sess)
	require.NoError(t, err, "Create should not return error")

	// Verify session can be retrieved
	opts := &session.GetOptions{}
	retrieved, err := repo.Get(ctx, sess.AppName, sess.UserID, sess.SessionID, opts)
	require.NoError(t, err)
	assert.Equal(t, sess.AppName, retrieved.AppName)
	assert.Equal(t, sess.UserID, retrieved.UserID)
	assert.Equal(t, sess.SessionID, retrieved.SessionID)
}

func TestSessionRepository_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State: map[string]interface{}{
			"strategy":   "momentum",
			"risk_level": "moderate",
		},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Test Get
	opts := &session.GetOptions{}
	retrieved, err := repo.Get(ctx, sess.AppName, sess.UserID, sess.SessionID, opts)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, retrieved.ID)
	assert.Equal(t, "momentum", retrieved.State["strategy"])
	assert.Equal(t, "moderate", retrieved.State["risk_level"])

	// Test non-existent session
	_, err = repo.Get(ctx, "nonexistent", "user", "session", opts)
	assert.Error(t, err, "Should return error for non-existent session")
}

func TestSessionRepository_UpdateState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State: map[string]interface{}{
			"counter": 0,
			"mode":    "initial",
		},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Test UpdateState
	newState := map[string]interface{}{
		"counter": 5,
		"mode":    "active",
		"symbol":  "ETH/USDT",
	}

	err = repo.UpdateState(ctx, sess.AppName, sess.UserID, sess.SessionID, newState)
	require.NoError(t, err)

	// Verify updates
	opts := &session.GetOptions{}
	retrieved, err := repo.Get(ctx, sess.AppName, sess.UserID, sess.SessionID, opts)
	require.NoError(t, err)
	assert.Equal(t, 5.0, retrieved.State["counter"]) // JSON numbers are float64
	assert.Equal(t, "active", retrieved.State["mode"])
	assert.Equal(t, "ETH/USDT", retrieved.State["symbol"])
}

func TestSessionRepository_AppendEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	// Create session
	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State:     map[string]interface{}{},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Test AppendEvent
	eventID := testsupport.UniqueEventID()
	event := &session.Event{
		ID:        uuid.New(),
		SessionID: sess.ID,
		EventID:   eventID,
		Author:    "user",
		Content: map[string]interface{}{
			"type":    "text",
			"message": "Analyze BTC/USDT",
		},
		Timestamp:    time.Now(),
		Branch:       "main",
		Partial:      false,
		TurnComplete: false,
		Actions: session.EventActions{
			SkipSummarization: false,
			StateDelta:        map[string]interface{}{},
		},
		UsageMetadata: &session.UsageMetadata{
			PromptTokenCount:     50,
			CandidatesTokenCount: 100,
			TotalTokenCount:      150,
		},
	}

	err = repo.AppendEvent(ctx, sess.ID, event)
	require.NoError(t, err)

	// Verify event was added
	opts := &session.GetOptions{NumRecentEvents: 10}
	retrieved, err := repo.Get(ctx, sess.AppName, sess.UserID, sess.SessionID, opts)
	require.NoError(t, err)
	assert.Len(t, retrieved.Events, 1, "Should have 1 event")
	assert.Equal(t, eventID, retrieved.Events[0].EventID)
	assert.Equal(t, "user", retrieved.Events[0].Author)
}

func TestSessionRepository_GetEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	// Create session
	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State:     map[string]interface{}{},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Add multiple events
	for i := 0; i < 5; i++ {
		event := &session.Event{
			ID:        uuid.New(),
			SessionID: sess.ID,
			EventID:   testsupport.UniqueEventID(),
			Author:    "agent",
			Content: map[string]interface{}{
				"response": "Analysis result " + string(rune(i+'0')),
			},
			Timestamp:    time.Now().Add(time.Duration(i) * time.Second),
			Branch:       "main",
			Partial:      false,
			TurnComplete: i == 4,
			Actions: session.EventActions{
				SkipSummarization: false,
			},
		}
		err = repo.AppendEvent(ctx, sess.ID, event)
		require.NoError(t, err)
	}

	// Test GetEvents with limit
	opts := &session.GetEventsOptions{
		Limit: 3,
	}
	events, err := repo.GetEvents(ctx, sess.ID, opts)
	require.NoError(t, err)
	assert.Len(t, events, 3, "Should respect limit")

	// Test GetEvents without limit
	allEventsOpts := &session.GetEventsOptions{}
	allEvents, err := repo.GetEvents(ctx, sess.ID, allEventsOpts)
	require.NoError(t, err)
	assert.Len(t, allEvents, 5, "Should return all events")
}

func TestSessionRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	appName := "trading_agent"
	userID := "user123"

	// Create multiple sessions for same user
	for i := 0; i < 4; i++ {
		sess := &session.Session{
			ID:        uuid.New(),
			AppName:   appName,
			UserID:    userID,
			SessionID: testsupport.UniqueSessionID(),
			State:     map[string]interface{}{},
			Events:    []session.Event{},
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, sess)
		require.NoError(t, err)
	}

	// Test List
	sessions, err := repo.List(ctx, appName, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 4, "Should return all 4 sessions")

	// Verify all belong to same user
	for _, s := range sessions {
		assert.Equal(t, userID, s.UserID)
		assert.Equal(t, appName, s.AppName)
	}
}

func TestSessionRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    "user123",
		SessionID: testsupport.UniqueSessionID(),
		State:     map[string]interface{}{},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Verify session exists
	opts := &session.GetOptions{}
	_, err = repo.Get(ctx, sess.AppName, sess.UserID, sess.SessionID, opts)
	require.NoError(t, err)

	// Test Delete
	err = repo.Delete(ctx, sess.AppName, sess.UserID, sess.SessionID)
	require.NoError(t, err)

	// Verify session is deleted
	_, err = repo.Get(ctx, sess.AppName, sess.UserID, sess.SessionID, opts)
	assert.Error(t, err, "Should return error after deletion")
}

func TestSessionRepository_SetGetAppState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	appName := "trading_agent"
	appState := map[string]interface{}{
		"global_mode": "production",
		"max_agents":  10,
		"feature_flags": map[string]interface{}{
			"enable_options": true,
			"enable_futures": false,
		},
	}

	// Test SetAppState
	err := repo.SetAppState(ctx, appName, appState)
	require.NoError(t, err)

	// Test GetAppState
	retrieved, err := repo.GetAppState(ctx, appName)
	require.NoError(t, err)
	assert.Equal(t, appName, retrieved.AppName)
	assert.Equal(t, "production", retrieved.State["global_mode"])
	assert.Equal(t, 10.0, retrieved.State["max_agents"]) // JSON number

	// Verify nested map
	featureFlags, ok := retrieved.State["feature_flags"].(map[string]interface{})
	assert.True(t, ok)
	assert.True(t, featureFlags["enable_options"].(bool))
}

func TestSessionRepository_SetGetUserState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	appName := "trading_agent"
	userID := "user123"
	userState := map[string]interface{}{
		"preferences": map[string]interface{}{
			"language": "en",
			"timezone": "UTC",
			"theme":    "dark",
		},
		"total_trades": 42,
		"last_login":   time.Now().Format(time.RFC3339),
	}

	// Test SetUserState
	err := repo.SetUserState(ctx, appName, userID, userState)
	require.NoError(t, err)

	// Test GetUserState
	retrieved, err := repo.GetUserState(ctx, appName, userID)
	require.NoError(t, err)
	assert.Equal(t, appName, retrieved.AppName)
	assert.Equal(t, userID, retrieved.UserID)
	assert.Equal(t, 42.0, retrieved.State["total_trades"]) // JSON number

	// Verify nested preferences
	prefs, ok := retrieved.State["preferences"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "en", prefs["language"])
	assert.Equal(t, "dark", prefs["theme"])
}

func TestSessionRepository_EventWithUsageMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State:     map[string]interface{}{},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Event with detailed usage metadata
	event := &session.Event{
		ID:        uuid.New(),
		SessionID: sess.ID,
		EventID:   testsupport.UniqueEventID(),
		Author:    "opportunity_synthesizer",
		Content: map[string]interface{}{
			"analysis": "Market shows bullish momentum",
		},
		Timestamp:    time.Now(),
		Branch:       "main",
		Partial:      false,
		TurnComplete: true,
		Actions: session.EventActions{
			SkipSummarization: false,
			StateDelta: map[string]interface{}{
				"last_analysis": "completed",
			},
		},
		UsageMetadata: &session.UsageMetadata{
			PromptTokenCount:     2500,
			CandidatesTokenCount: 1500,
			TotalTokenCount:      4000,
		},
	}

	err = repo.AppendEvent(ctx, sess.ID, event)
	require.NoError(t, err)

	// Retrieve and verify usage metadata
	eventsOpts := &session.GetEventsOptions{}
	events, err := repo.GetEvents(ctx, sess.ID, eventsOpts)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	assert.NotNil(t, events[0].UsageMetadata)
	assert.Equal(t, int32(2500), events[0].UsageMetadata.PromptTokenCount)
	assert.Equal(t, int32(1500), events[0].UsageMetadata.CandidatesTokenCount)
	assert.Equal(t, int32(4000), events[0].UsageMetadata.TotalTokenCount)
}

func TestSessionRepository_MultipleApps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	userID := uuid.New().String()    // Unique user for this test
	sessionID := uuid.New().String() // Unique session ID
	apps := []string{"trading_agent", "analysis_agent", "portfolio_agent"}

	// Create sessions for different apps
	for _, appName := range apps {
		sess := &session.Session{
			ID:        uuid.New(),
			AppName:   appName,
			UserID:    userID,
			SessionID: sessionID,
			State:     map[string]interface{}{"app": appName},
			Events:    []session.Event{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Create(ctx, sess)
		require.NoError(t, err)
	}

	// Verify each app has its own session
	for _, appName := range apps {
		sessions, err := repo.List(ctx, appName, userID)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
		assert.Equal(t, appName, sessions[0].AppName)
	}
}

func TestSessionRepository_EventActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewSessionRepository(testDB.Tx())
	ctx := context.Background()

	sess := &session.Session{
		ID:        uuid.New(),
		AppName:   "trading_agent",
		UserID:    uuid.New().String(),
		SessionID: uuid.New().String(),
		State:     map[string]interface{}{},
		Events:    []session.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Create(ctx, sess)
	require.NoError(t, err)

	// Event with actions
	event := &session.Event{
		ID:           uuid.New(),
		SessionID:    sess.ID,
		EventID:      testsupport.UniqueEventID(),
		Author:       "agent",
		Content:      map[string]interface{}{"message": "Transfer to specialist"},
		Timestamp:    time.Now(),
		Branch:       "main",
		Partial:      false,
		TurnComplete: true,
		Actions: session.EventActions{
			TransferToAgent:   "portfolio_manager",
			Escalate:          true,
			SkipSummarization: true,
			StateDelta: map[string]interface{}{
				"transferred_to": "portfolio_manager",
				"escalated":      true,
			},
		},
	}

	err = repo.AppendEvent(ctx, sess.ID, event)
	require.NoError(t, err)

	// Retrieve and verify actions
	eventsOpts := &session.GetEventsOptions{}
	events, err := repo.GetEvents(ctx, sess.ID, eventsOpts)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	assert.Equal(t, "portfolio_manager", events[0].Actions.TransferToAgent)
	assert.True(t, events[0].Actions.Escalate)
	assert.True(t, events[0].Actions.SkipSummarization)
	assert.Equal(t, "portfolio_manager", events[0].Actions.StateDelta["transferred_to"])
}
