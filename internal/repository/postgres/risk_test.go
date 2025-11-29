package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/risk"
	"prometheus/internal/testsupport"
)

func TestRiskRepository_SaveState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()

	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		TriggeredAt:        nil,
		TriggerReason:      nil,
		DailyPnL:           decimal.NewFromFloat(-500.0),
		DailyPnLPercent:    decimal.NewFromFloat(-2.5),
		DailyTradeCount:    5,
		DailyWins:          2,
		DailyLosses:        3,
		ConsecutiveLosses:  2,
		MaxDailyDrawdown:   decimal.NewFromFloat(-5.0),
		MaxConsecutiveLoss: 3,
		ResetAt:            now.Add(24 * time.Hour),
		UpdatedAt:          now,
	}

	// Test SaveState
	err := repo.SaveState(ctx, state)
	require.NoError(t, err, "SaveState should not return error")

	// Verify state can be retrieved
	retrieved, err := repo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, retrieved.UserID)
	assert.False(t, retrieved.IsTriggered)
	assert.True(t, state.DailyPnL.Equal(retrieved.DailyPnL))
	assert.Equal(t, 2, retrieved.ConsecutiveLosses)
}

func TestRiskRepository_GetState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()

	// Create initial state
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.Zero,
		DailyPnLPercent:    decimal.Zero,
		DailyTradeCount:    0,
		DailyWins:          0,
		DailyLosses:        0,
		ConsecutiveLosses:  0,
		MaxDailyDrawdown:   decimal.NewFromFloat(-5.0),
		MaxConsecutiveLoss: 3,
		ResetAt:            now.Add(24 * time.Hour),
		UpdatedAt:          now,
	}

	err := repo.SaveState(ctx, state)
	require.NoError(t, err)

	// Test GetState
	retrieved, err := repo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, retrieved.UserID)
	assert.Equal(t, 0, retrieved.DailyTradeCount)
	assert.Equal(t, 0, retrieved.ConsecutiveLosses)

	// Test non-existent user
	_, err = repo.GetState(ctx, uuid.New())
	assert.Error(t, err, "Should return error for non-existent user")
}

func TestRiskRepository_CircuitBreakerTriggered(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()
	triggeredAt := now

	// Create triggered circuit breaker state
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        true,
		TriggeredAt:        &triggeredAt,
		TriggerReason:      func() *string { s := "Max daily drawdown exceeded"; return &s }(),
		DailyPnL:           decimal.NewFromFloat(-1200.0),
		DailyPnLPercent:    decimal.NewFromFloat(-6.0),
		DailyTradeCount:    8,
		DailyWins:          2,
		DailyLosses:        6,
		ConsecutiveLosses:  4,
		MaxDailyDrawdown:   decimal.NewFromFloat(-5.0),
		MaxConsecutiveLoss: 3,
		ResetAt:            now.Add(24 * time.Hour),
		UpdatedAt:          now,
	}

	err := repo.SaveState(ctx, state)
	require.NoError(t, err)

	// Verify triggered state
	retrieved, err := repo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.True(t, retrieved.IsTriggered)
	assert.NotNil(t, retrieved.TriggeredAt)
	require.NotNil(t, retrieved.TriggerReason)
	assert.Equal(t, "Max daily drawdown exceeded", *retrieved.TriggerReason)
	assert.Equal(t, 4, retrieved.ConsecutiveLosses)
}

func TestRiskRepository_CreateEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()

	event := &risk.RiskEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Timestamp:    time.Now(),
		EventType:    risk.RiskEventDrawdown,
		Severity:     "warning",
		Message:      "Daily drawdown approaching limit",
		Data:         `{"current_drawdown": -4.5, "limit": -5.0}`,
		Acknowledged: false,
	}

	// Test CreateEvent
	err := repo.CreateEvent(ctx, event)
	require.NoError(t, err, "CreateEvent should not return error")
}

func TestRiskRepository_GetEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()

	// Create multiple risk events
	eventTypes := []risk.RiskEventType{
		risk.RiskEventDrawdown,
		risk.RiskEventConsecutiveLoss,
		risk.RiskEventCircuitBreaker,
	}

	for i, eventType := range eventTypes {
		event := &risk.RiskEvent{
			ID:           uuid.New(),
			UserID:       userID,
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour),
			EventType:    eventType,
			Severity:     "warning",
			Message:      "Test event " + string(rune(i+'0')),
			Data:         "{}",
			Acknowledged: false,
		}
		err := repo.CreateEvent(ctx, event)
		require.NoError(t, err)
	}

	// Test GetEvents with limit
	events, err := repo.GetEvents(ctx, userID, 2)
	require.NoError(t, err)
	assert.Len(t, events, 2, "Should respect limit")

	// Verify all events belong to user
	for _, event := range events {
		assert.Equal(t, userID, event.UserID)
	}

	// Get all events
	allEvents, err := repo.GetEvents(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, allEvents, 3, "Should return all 3 events")
}

func TestRiskRepository_AcknowledgeEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()

	// Create unacknowledged event
	event := &risk.RiskEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Timestamp:    time.Now(),
		EventType:    risk.RiskEventCircuitBreaker,
		Severity:     "critical",
		Message:      "Circuit breaker triggered",
		Data:         "{}",
		Acknowledged: false,
	}

	err := repo.CreateEvent(ctx, event)
	require.NoError(t, err)

	// Test AcknowledgeEvent
	err = repo.AcknowledgeEvent(ctx, event.ID)
	require.NoError(t, err)

	// Verify acknowledged
	events, err := repo.GetEvents(ctx, userID, 10)
	require.NoError(t, err)

	var found bool
	for _, e := range events {
		if e.ID == event.ID {
			found = true
			assert.True(t, e.Acknowledged, "Event should be acknowledged")
		}
	}
	assert.True(t, found, "Event should be found")
}

func TestRiskRepository_ResetDaily(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()

	// Create states for multiple users
	userIDs := []uuid.UUID{
		fixtures.CreateUser(),
		fixtures.CreateUser(),
		fixtures.CreateUser(),
	}
	for _, userID := range userIDs {
		state := &risk.CircuitBreakerState{
			UserID:             userID,
			IsTriggered:        false,
			DailyPnL:           decimal.NewFromFloat(-1000.0),
			DailyPnLPercent:    decimal.NewFromFloat(-5.0),
			DailyTradeCount:    10,
			DailyWins:          4,
			DailyLosses:        6,
			ConsecutiveLosses:  3,
			MaxDailyDrawdown:   decimal.NewFromFloat(-5.0),
			MaxConsecutiveLoss: 3,
			ResetAt:            time.Now().Add(-1 * time.Hour), // Past reset time
			UpdatedAt:          time.Now(),
		}
		err := repo.SaveState(ctx, state)
		require.NoError(t, err)
	}

	// Test ResetDaily
	err := repo.ResetDaily(ctx)
	require.NoError(t, err)

	// Verify all states are reset
	for _, userID := range userIDs {
		state, err := repo.GetState(ctx, userID)
		require.NoError(t, err)
		assert.True(t, state.DailyPnL.Equal(decimal.Zero), "DailyPnL should be reset")
		assert.Equal(t, 0, state.DailyTradeCount, "DailyTradeCount should be reset")
		assert.Equal(t, 0, state.DailyWins, "DailyWins should be reset")
		assert.Equal(t, 0, state.DailyLosses, "DailyLosses should be reset")
		assert.Equal(t, 0, state.ConsecutiveLosses, "ConsecutiveLosses should be reset")
	}
}

func TestRiskRepository_UpdateState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()

	// Create initial state
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.Zero,
		DailyPnLPercent:    decimal.Zero,
		DailyTradeCount:    0,
		DailyWins:          0,
		DailyLosses:        0,
		ConsecutiveLosses:  0,
		MaxDailyDrawdown:   decimal.NewFromFloat(-5.0),
		MaxConsecutiveLoss: 3,
		ResetAt:            now.Add(24 * time.Hour),
		UpdatedAt:          now,
	}

	err := repo.SaveState(ctx, state)
	require.NoError(t, err)

	// Update state with new trade results
	state.DailyPnL = decimal.NewFromFloat(-800.0)
	state.DailyPnLPercent = decimal.NewFromFloat(-4.0)
	state.DailyTradeCount = 3
	state.DailyWins = 1
	state.DailyLosses = 2
	state.ConsecutiveLosses = 2
	state.UpdatedAt = time.Now()

	err = repo.SaveState(ctx, state)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := repo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(-800.0).Equal(retrieved.DailyPnL))
	assert.Equal(t, 3, retrieved.DailyTradeCount)
	assert.Equal(t, 2, retrieved.ConsecutiveLosses)
}

func TestRiskRepository_MultipleEventTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()

	// Create events of all types
	eventTypes := []risk.RiskEventType{
		risk.RiskEventDrawdown,
		risk.RiskEventConsecutiveLoss,
		risk.RiskEventCircuitBreaker,
		risk.RiskEventMaxExposure,
		risk.RiskEventKillSwitch,
		risk.RiskEventAnomalyDetected,
	}

	for i, eventType := range eventTypes {
		event := &risk.RiskEvent{
			ID:           uuid.New(),
			UserID:       userID,
			Timestamp:    time.Now(),
			EventType:    eventType,
			Severity:     "warning",
			Message:      "Event type: " + eventType.String(),
			Data:         "{}",
			Acknowledged: i%2 == 0, // Alternate acknowledged status
		}
		err := repo.CreateEvent(ctx, event)
		require.NoError(t, err, "Should create event of type: "+eventType.String())
	}

	// Retrieve and verify
	events, err := repo.GetEvents(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, events, 6, "Should return all 6 event types")

	// Verify event types are preserved
	eventTypeMap := make(map[risk.RiskEventType]bool)
	for _, event := range events {
		eventTypeMap[event.EventType] = true
	}

	for _, eventType := range eventTypes {
		assert.True(t, eventTypeMap[eventType], "Event type "+eventType.String()+" should be present")
	}
}

func TestRiskRepository_EventSeverity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()

	// Create warning event
	warningEvent := &risk.RiskEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Timestamp:    time.Now(),
		EventType:    risk.RiskEventDrawdown,
		Severity:     "warning",
		Message:      "Approaching drawdown limit",
		Data:         `{"current": -4.5, "limit": -5.0}`,
		Acknowledged: false,
	}
	err := repo.CreateEvent(ctx, warningEvent)
	require.NoError(t, err)

	// Create critical event
	criticalEvent := &risk.RiskEvent{
		ID:           uuid.New(),
		UserID:       userID,
		Timestamp:    time.Now(),
		EventType:    risk.RiskEventCircuitBreaker,
		Severity:     "critical",
		Message:      "Circuit breaker activated",
		Data:         `{"reason": "max_drawdown_exceeded"}`,
		Acknowledged: false,
	}
	err = repo.CreateEvent(ctx, criticalEvent)
	require.NoError(t, err)

	// Retrieve events
	events, err := repo.GetEvents(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	// Verify severities
	var hasWarning, hasCritical bool
	for _, event := range events {
		if event.Severity == "warning" {
			hasWarning = true
		}
		if event.Severity == "critical" {
			hasCritical = true
		}
	}
	assert.True(t, hasWarning, "Should have warning event")
	assert.True(t, hasCritical, "Should have critical event")
}

func TestRiskRepository_ConsecutiveLossesTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewRiskRepository(testDB.DB())
	ctx := context.Background()
	now := time.Now()

	// Initial state
	state := &risk.CircuitBreakerState{
		UserID:             userID,
		IsTriggered:        false,
		DailyPnL:           decimal.Zero,
		DailyPnLPercent:    decimal.Zero,
		DailyTradeCount:    0,
		DailyWins:          0,
		DailyLosses:        0,
		ConsecutiveLosses:  0,
		MaxDailyDrawdown:   decimal.NewFromFloat(-5.0),
		MaxConsecutiveLoss: 3,
		ResetAt:            now.Add(24 * time.Hour),
		UpdatedAt:          now,
	}

	err := repo.SaveState(ctx, state)
	require.NoError(t, err)

	// Simulate consecutive losses
	for i := 1; i <= 3; i++ {
		state.DailyLosses = i
		state.ConsecutiveLosses = i
		state.DailyTradeCount = i
		state.DailyPnL = state.DailyPnL.Sub(decimal.NewFromFloat(100.0))

		err = repo.SaveState(ctx, state)
		require.NoError(t, err)

		retrieved, err := repo.GetState(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, i, retrieved.ConsecutiveLosses)
	}

	// Verify final state
	finalState, err := repo.GetState(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 3, finalState.ConsecutiveLosses)
	assert.Equal(t, 3, finalState.DailyLosses)
	assert.Equal(t, 0, finalState.DailyWins)
}
