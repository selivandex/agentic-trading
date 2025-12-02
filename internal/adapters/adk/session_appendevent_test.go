package adk_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/session"

	"prometheus/internal/adapters/adk"
	domainsession "prometheus/internal/domain/session"
	"prometheus/internal/repository/postgres"
	"prometheus/internal/testsupport"
)

// TestADKSessionService_AppendEvent tests that AppendEvent correctly handles foreign key
func TestADKSessionService_AppendEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := postgres.NewSessionRepository(testDB.Tx())
	domainService := domainsession.NewService(repo)
	adkService := adk.NewSessionService(domainService)

	ctx := context.Background()

	// Step 1: Create session via ADK service
	createResp, err := adkService.Create(ctx, &session.CreateRequest{
		AppName:   "test_app",
		UserID:    "user123",
		SessionID: "", // Auto-generate
		State:     nil,
	})
	require.NoError(t, err)
	require.NotNil(t, createResp.Session)

	adkSession := createResp.Session

	t.Logf("✅ Session created: AppName=%s, UserID=%s, SessionID=%s",
		adkSession.AppName(),
		adkSession.UserID(),
		adkSession.ID(),
	)

	// Step 2: Create event with unique ID
	adkEvent := &session.Event{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Branch:    "main",
		Author:    "user",
	}
	adkEvent.LLMResponse.Content = nil
	adkEvent.LLMResponse.Partial = false
	adkEvent.TurnComplete = true

	// Step 3: Append event (THIS IS WHERE THE BUG WAS!)
	err = adkService.AppendEvent(ctx, adkSession, adkEvent)
	require.NoError(t, err, "AppendEvent should work without foreign key violation")

	t.Logf("✅ Event appended successfully")

	// Step 4: Verify event was stored
	getResp, err := adkService.Get(ctx, &session.GetRequest{
		AppName:         adkSession.AppName(),
		UserID:          adkSession.UserID(),
		SessionID:       adkSession.ID(),
		NumRecentEvents: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, getResp.Session.Events().Len(), "Should have 1 event")
}

// TestADKSessionService_MultipleEvents tests multiple event appends
func TestADKSessionService_MultipleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := postgres.NewSessionRepository(testDB.Tx())
	domainService := domainsession.NewService(repo)
	adkService := adk.NewSessionService(domainService)

	ctx := context.Background()

	// Create session
	createResp, err := adkService.Create(ctx, &session.CreateRequest{
		AppName:   "test_workflow",
		UserID:    "user456",
		SessionID: "",
		State:     nil,
	})
	require.NoError(t, err)

	adkSession := createResp.Session

	// Append multiple events
	for i := 0; i < 5; i++ {
		adkEvent := &session.Event{
			ID:        fmt.Sprintf("event_%d_%d", time.Now().UnixNano(), i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Branch:    "main",
			Author:    "agent",
		}
		adkEvent.LLMResponse.Partial = false
		adkEvent.TurnComplete = true

		err = adkService.AppendEvent(ctx, adkSession, adkEvent)
		require.NoError(t, err, "Event %d should append successfully", i)
	}

	t.Logf("✅ All 5 events appended successfully")

	// Verify all events stored
	getResp, err := adkService.Get(ctx, &session.GetRequest{
		AppName:         adkSession.AppName(),
		UserID:          adkSession.UserID(),
		SessionID:       adkSession.ID(),
		NumRecentEvents: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, getResp.Session.Events().Len(), "Should have 5 events")
}
