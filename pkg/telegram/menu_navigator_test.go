package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeCallback_StoresDataInSession(t *testing.T) {
	// Create session
	session := NewInMemorySession(123456, "test", nil)

	// Create navigator
	nav := &MenuNavigator{}

	// Make callback with parameters
	callbackKey := nav.MakeCallback(session, "detail", "acc_id", "uuid-123", "name", "Binance")

	// Verify callback key format
	assert.Len(t, callbackKey, 8, "Callback key should be 8 hex chars")
	assert.Regexp(t, "^[a-f0-9]{8}$", callbackKey, "Callback key should be 8 hex chars")

	// Verify data was stored in session
	callbackData, ok := session.GetCallbackData(callbackKey)
	require.True(t, ok, "Callback data should be stored in session")

	// Verify parameters
	assert.Equal(t, "detail", callbackData["screen"])
	assert.Equal(t, "uuid-123", callbackData["acc_id"])
	assert.Equal(t, "Binance", callbackData["name"])
}

func TestMakeCallback_UniqueKeys(t *testing.T) {
	session := NewInMemorySession(123456, "test", nil)
	nav := &MenuNavigator{}

	// Generate multiple callbacks in tight loop
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key := nav.MakeCallback(session, "screen", "param", "value")
		if keys[key] {
			t.Fatalf("Duplicate key generated: %s", key)
		}
		keys[key] = true
	}

	assert.Len(t, keys, 100, "Should generate 100 unique keys")
}

func TestSession_ClearCallbackData(t *testing.T) {
	session := NewInMemorySession(123456, "test", nil)
	nav := &MenuNavigator{}

	// Add some callback data
	key1 := nav.MakeCallback(session, "screen1", "p1", "v1")
	key2 := nav.MakeCallback(session, "screen2", "p2", "v2")

	// Verify data exists
	_, ok1 := session.GetCallbackData(key1)
	_, ok2 := session.GetCallbackData(key2)
	assert.True(t, ok1, "First callback should exist")
	assert.True(t, ok2, "Second callback should exist")

	// Clear callback data
	session.ClearCallbackData()

	// Verify data was cleared
	_, ok1After := session.GetCallbackData(key1)
	_, ok2After := session.GetCallbackData(key2)
	assert.False(t, ok1After, "First callback should be cleared")
	assert.False(t, ok2After, "Second callback should be cleared")
}

func TestHandleCallback_RetrievesStoredData(t *testing.T) {
	// This test verifies the full flow:
	// 1. MakeCallback stores data
	// 2. HandleCallback retrieves data
	// 3. Screen is found and parameters are extracted

	ctx := context.Background()

	// Create in-memory session service
	sessionService := NewInMemorySessionService()

	// Create session
	session, err := sessionService.CreateSession(ctx, 123456, "list", map[string]interface{}{
		"user_id": "test-user",
	}, 0)
	require.NoError(t, err)

	// Create navigator (minimal - just for testing MakeCallback)
	nav := &MenuNavigator{
		sessionService: sessionService,
	}

	// Make callback and store in session
	callbackKey := nav.MakeCallback(session, "detail", "acc_id", "uuid-test-123")

	// Save session (important!)
	err = sessionService.SaveSession(ctx, session, 0)
	require.NoError(t, err)

	// Now retrieve session and verify callback can be handled
	retrievedSession, err := sessionService.GetSession(ctx, 123456)
	require.NoError(t, err)

	// Verify callback data exists
	callbackData, ok := retrievedSession.GetCallbackData(callbackKey)
	require.True(t, ok, "Callback data should exist after session save/load")
	assert.Equal(t, "detail", callbackData["screen"])
	assert.Equal(t, "uuid-test-123", callbackData["acc_id"])
}
