package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/reasoning"
	"prometheus/internal/testsupport"
)

func TestReasoningRepository_Create(t *testing.T) {
	// Skip if not in test environment
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewReasoningRepository(testDB.DB())
	ctx := context.Background()

	// Create test reasoning log
	reasoningSteps := []map[string]interface{}{
		{
			"step":      1,
			"action":    "thinking",
			"content":   "Analyzing market conditions",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	reasoningStepsJSON, _ := json.Marshal(reasoningSteps)

	decision := map[string]interface{}{
		"action":     "publish",
		"confidence": 0.75,
	}
	decisionJSON, _ := json.Marshal(decision)

	userID := uuid.New()
	entry := &reasoning.LogEntry{
		ID:             uuid.New(),
		UserID:         &userID,
		AgentID:        "test_agent",
		AgentType:      "market_analyst",
		SessionID:      uuid.New().String(),
		ReasoningSteps: reasoningStepsJSON,
		Decision:       decisionJSON,
		Confidence:     75.0,
		TokensUsed:     1000,
		CostUSD:        0.05,
		DurationMs:     500,
		ToolCallsCount: 3,
		CreatedAt:      time.Now(),
	}

	// Test Create
	err := repo.Create(ctx, entry)
	require.NoError(t, err, "Create should not return error")

	// Test GetByID
	retrieved, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err, "GetByID should not return error")
	assert.Equal(t, entry.AgentID, retrieved.AgentID)
	assert.Equal(t, entry.AgentType, retrieved.AgentType)
	assert.Equal(t, entry.Confidence, retrieved.Confidence)
}

func TestReasoningRepository_GetBySession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewReasoningRepository(testDB.DB())
	ctx := context.Background()

	sessionID := uuid.New().String()

	// Create multiple entries for same session
	for i := 0; i < 3; i++ {
		reasoningSteps, _ := json.Marshal([]map[string]interface{}{
			{"step": i, "action": "test"},
		})
		decision, _ := json.Marshal(map[string]interface{}{"action": "skip"})

		entry := &reasoning.LogEntry{
			ID:             uuid.New(),
			AgentID:        "test_agent",
			AgentType:      "market_analyst",
			SessionID:      sessionID,
			ReasoningSteps: reasoningSteps,
			Decision:       decision,
			CreatedAt:      time.Now(),
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Test GetBySession
	entries, err := repo.GetBySession(ctx, sessionID)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "Should retrieve all 3 entries for session")
}

func TestReasoningRepository_GetByAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewReasoningRepository(testDB.DB())
	ctx := context.Background()

	agentID := "specific_agent_" + uuid.New().String()

	// Create entries for specific agent
	for i := 0; i < 5; i++ {
		reasoningSteps, _ := json.Marshal([]map[string]interface{}{
			{"step": i},
		})
		decision, _ := json.Marshal(map[string]interface{}{"action": "publish"})

		entry := &reasoning.LogEntry{
			ID:             uuid.New(),
			AgentID:        agentID,
			AgentType:      "market_analyst",
			SessionID:      uuid.New().String(),
			ReasoningSteps: reasoningSteps,
			Decision:       decision,
			CreatedAt:      time.Now(),
		}
		err := repo.Create(ctx, entry)
		require.NoError(t, err)
	}

	// Test GetByAgent with limit
	entries, err := repo.GetByAgent(ctx, agentID, 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "Should respect limit parameter")

	// Verify all entries are for correct agent
	for _, entry := range entries {
		assert.Equal(t, agentID, entry.AgentID)
	}
}

func TestReasoningRepository_JSONBFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewReasoningRepository(testDB.DB())
	ctx := context.Background()

	// Complex reasoning steps with nested structures
	reasoningSteps := []map[string]interface{}{
		{
			"step":      1,
			"action":    "tool_call",
			"tool":      "get_price",
			"input":     map[string]interface{}{"symbol": "BTC/USDT"},
			"output":    map[string]interface{}{"price": 42000},
			"timestamp": time.Now().Format(time.RFC3339),
		},
		{
			"step":      2,
			"action":    "thinking",
			"content":   "Price is above key level",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	reasoningStepsJSON, _ := json.Marshal(reasoningSteps)

	decision := map[string]interface{}{
		"action":     "publish",
		"confidence": 0.85,
		"reasoning":  "Strong bullish signal",
		"levels": map[string]interface{}{
			"entry":  42000,
			"stop":   41000,
			"target": 44000,
		},
	}
	decisionJSON, _ := json.Marshal(decision)

	entry := &reasoning.LogEntry{
		ID:             uuid.New(),
		AgentID:        "test_agent",
		AgentType:      "market_analyst",
		SessionID:      uuid.New().String(),
		ReasoningSteps: reasoningStepsJSON,
		Decision:       decisionJSON,
		CreatedAt:      time.Now(),
	}

	// Create and retrieve
	err := repo.Create(ctx, entry)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, entry.ID)
	require.NoError(t, err)

	// Unmarshal and verify JSONB integrity
	var retrievedSteps []map[string]interface{}
	err = json.Unmarshal(retrieved.ReasoningSteps, &retrievedSteps)
	require.NoError(t, err)
	assert.Len(t, retrievedSteps, 2)
	assert.Equal(t, "get_price", retrievedSteps[0]["tool"])

	var retrievedDecision map[string]interface{}
	err = json.Unmarshal(retrieved.Decision, &retrievedDecision)
	require.NoError(t, err)
	assert.Equal(t, "publish", retrievedDecision["action"])
	assert.Equal(t, 0.85, retrievedDecision["confidence"])

	// Check nested structure
	levels, ok := retrievedDecision["levels"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(42000), levels["entry"])
}
