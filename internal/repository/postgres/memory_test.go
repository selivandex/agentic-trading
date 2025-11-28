package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/memory"
	"prometheus/internal/testsupport"
)

// createTestEmbedding creates a dummy embedding vector for testing
func createTestEmbedding(dimensions int) pgvector.Vector {
	slice := make([]float32, dimensions)
	for i := range slice {
		slice[i] = float32(i) * 0.1
	}
	return pgvector.NewVector(slice)
}

func TestMemoryRepository_StoreUserMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	embedding := createTestEmbedding(1536)

	mem := &memory.Memory{
		ID:                  uuid.New(),
		Scope:               memory.MemoryScopeUser,
		UserID:              &userID,
		AgentID:             "test_agent",
		SessionID:           uuid.New().String(),
		Type:                memory.MemoryObservation,
		Content:             "BTC showing strong bullish momentum",
		Embedding:           embedding,
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: 1536,
		Symbol:              "BTC/USDT",
		Timeframe:           "1h",
		Importance:          0.8,
		Tags:                []string{"bullish", "momentum"},
		Metadata: map[string]interface{}{
			"confidence": 0.85,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: nil,
	}

	err := repo.Store(ctx, mem)
	require.NoError(t, err)

	// Verify we can retrieve it
	retrieved, err := repo.GetByID(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, mem.ID, retrieved.ID)
	assert.Equal(t, memory.MemoryScopeUser, retrieved.Scope)
	assert.Equal(t, userID, *retrieved.UserID)
	assert.Equal(t, mem.Content, retrieved.Content)
}

func TestMemoryRepository_StoreCollectiveMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	sourceUserID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	validatedAt := time.Now()
	embedding := createTestEmbedding(1536)

	mem := &memory.Memory{
		ID:                  uuid.New(),
		Scope:               memory.MemoryScopeCollective,
		UserID:              nil, // Collective has no user
		AgentID:             "opportunity_synthesizer",
		Type:                memory.MemoryLesson,
		Content:             "Momentum breakouts work best with volume >2x average",
		Embedding:           embedding,
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: 1536,
		ValidationScore:     87.5,
		ValidationTrades:    15,
		ValidatedAt:         &validatedAt,
		Symbol:              "",
		Timeframe:           "",
		Importance:          0.85,
		Tags:                []string{"momentum", "breakout", "volume"},
		Personality:         nil,
		SourceUserID:        &sourceUserID,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	err := repo.Store(ctx, mem)
	require.NoError(t, err)

	// Verify
	retrieved, err := repo.GetByID(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, memory.MemoryScopeCollective, retrieved.Scope)
	assert.Nil(t, retrieved.UserID)
	assert.Equal(t, 87.5, retrieved.ValidationScore)
	assert.Equal(t, 15, retrieved.ValidationTrades)
}

func TestMemoryRepository_SearchSimilarUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	agentID := "test_agent"

	// Create test memories
	for i := 0; i < 5; i++ {
		mem := &memory.Memory{
			ID:                  uuid.New(),
			Scope:               memory.MemoryScopeUser,
			UserID:              &userID,
			AgentID:             agentID,
			Type:                memory.MemoryObservation,
			Content:             "Test memory " + string(rune(i+'0')),
			Embedding:           createTestEmbedding(1536),
			EmbeddingModel:      "text-embedding-3-small",
			EmbeddingDimensions: 1536,
			Importance:          0.5,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := repo.Store(ctx, mem)
		require.NoError(t, err)
	}

	// Search
	queryEmbedding := createTestEmbedding(1536)
	results, err := repo.SearchSimilar(ctx, userID, agentID, queryEmbedding, 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)

	// Verify results are for correct user and agent
	for _, result := range results {
		assert.Equal(t, memory.MemoryScopeUser, result.Scope)
		assert.Equal(t, userID, *result.UserID)
		assert.Equal(t, agentID, result.AgentID)
	}
}

func TestMemoryRepository_SearchCollectiveSimilar(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	agentID := "portfolio_manager"
	validatedAt := time.Now()

	// Create collective memories
	for i := 0; i < 4; i++ {
		mem := &memory.Memory{
			ID:                  uuid.New(),
			Scope:               memory.MemoryScopeCollective,
			UserID:              nil,
			AgentID:             agentID,
			Type:                memory.MemoryLesson,
			Content:             "Collective lesson " + string(rune(i+'0')),
			Embedding:           createTestEmbedding(1536),
			EmbeddingModel:      "text-embedding-3-small",
			EmbeddingDimensions: 1536,
			ValidationScore:     float64(75 + i*5),
			ValidationTrades:    10 + i,
			ValidatedAt:         &validatedAt,
			Importance:          0.8,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		err := repo.Store(ctx, mem)
		require.NoError(t, err)
	}

	// Search
	queryEmbedding := createTestEmbedding(1536)
	results, err := repo.SearchCollectiveSimilar(ctx, queryEmbedding, agentID, 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 3)

	// Verify
	for _, result := range results {
		assert.Equal(t, memory.MemoryScopeCollective, result.Scope)
		assert.Equal(t, agentID, result.AgentID)
		assert.Nil(t, result.UserID)
	}
}

func TestMemoryRepository_GetValidated(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	agentID := "opportunity_synthesizer"
	validatedAt := time.Now()

	// Create memories with different validation scores
	scores := []float64{95.0, 85.0, 75.0, 65.0}
	for _, score := range scores {
		mem := &memory.Memory{
			ID:               uuid.New(),
			Scope:            memory.MemoryScopeCollective,
			AgentID:          agentID,
			Type:             memory.MemoryLesson,
			Content:          "Lesson with score " + string(rune(int(score))),
			Embedding:        createTestEmbedding(1536),
			ValidationScore:  score,
			ValidationTrades: 10,
			ValidatedAt:      &validatedAt,
			Importance:       0.8,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		err := repo.Store(ctx, mem)
		require.NoError(t, err)
	}

	// Get validated with min score 80
	results, err := repo.GetValidated(ctx, agentID, 80.0, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2, "Should return at least 2 memories with score >= 80")

	// Verify all scores are >= 80
	for _, result := range results {
		assert.GreaterOrEqual(t, result.ValidationScore, 80.0)
		assert.NotNil(t, result.ValidatedAt)
		assert.Equal(t, agentID, result.AgentID)
	}
}

func TestMemoryRepository_UpdateValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	validatedAt := time.Now()

	mem := &memory.Memory{
		ID:               uuid.New(),
		Scope:            memory.MemoryScopeCollective,
		AgentID:          "test_agent",
		Type:             memory.MemoryLesson,
		Content:          "Initial lesson",
		Embedding:        createTestEmbedding(1536),
		ValidationScore:  70.0,
		ValidationTrades: 5,
		ValidatedAt:      &validatedAt,
		Importance:       0.7,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Store(ctx, mem)
	require.NoError(t, err)

	// Update validation
	err = repo.UpdateValidation(ctx, mem.ID, 85.0, 12)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, 85.0, retrieved.ValidationScore)
	assert.Equal(t, 12, retrieved.ValidationTrades)
}

func TestMemoryRepository_PromoteToCollective(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	// Create user memory
	userMemory := &memory.Memory{
		ID:                  uuid.New(),
		Scope:               memory.MemoryScopeUser,
		UserID:              &userID,
		AgentID:             "opportunity_synthesizer",
		Type:                memory.MemoryLesson,
		Content:             "RSI divergence signals trend reversal",
		Embedding:           createTestEmbedding(1536),
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: 1536,
		Symbol:              "BTC/USDT",
		Timeframe:           "4h",
		Importance:          0.9,
		Tags:                []string{"rsi", "divergence"},
		Metadata: map[string]interface{}{
			"confidence": 0.85,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := repo.Store(ctx, userMemory)
	require.NoError(t, err)

	// Promote to collective
	collective, err := repo.PromoteToCollective(ctx, userMemory, 88.5, 12)
	require.NoError(t, err)
	require.NotNil(t, collective)

	// Verify promoted memory
	assert.NotEqual(t, userMemory.ID, collective.ID)
	assert.Equal(t, memory.MemoryScopeCollective, collective.Scope)
	assert.Nil(t, collective.UserID)
	assert.Equal(t, userMemory.AgentID, collective.AgentID)
	assert.Equal(t, userMemory.Content, collective.Content)
	assert.Equal(t, 88.5, collective.ValidationScore)
	assert.Equal(t, 12, collective.ValidationTrades)
	assert.NotNil(t, collective.ValidatedAt)
	assert.Equal(t, &userID, collective.SourceUserID)
}

func TestMemoryRepository_DeleteExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	// Create expired memory
	pastTime := time.Now().Add(-24 * time.Hour)
	expiredMem := &memory.Memory{
		ID:        uuid.New(),
		Scope:     memory.MemoryScopeUser,
		UserID:    &userID,
		AgentID:   "test_agent",
		Type:      memory.MemoryObservation,
		Content:   "Expired memory",
		Embedding: createTestEmbedding(1536),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: &pastTime,
	}

	err := repo.Store(ctx, expiredMem)
	require.NoError(t, err)

	// Create non-expired memory
	validMem := &memory.Memory{
		ID:        uuid.New(),
		Scope:     memory.MemoryScopeUser,
		UserID:    &userID,
		AgentID:   "test_agent",
		Type:      memory.MemoryObservation,
		Content:   "Valid memory",
		Embedding: createTestEmbedding(1536),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: nil,
	}

	err = repo.Store(ctx, validMem)
	require.NoError(t, err)

	// Delete expired
	deleted, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify expired is gone
	_, err = repo.GetByID(ctx, expiredMem.ID)
	assert.Error(t, err)

	// Verify valid still exists
	retrieved, err := repo.GetByID(ctx, validMem.ID)
	require.NoError(t, err)
	assert.Equal(t, validMem.ID, retrieved.ID)
}

func TestMemoryRepository_GetByAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	agent1 := "agent_1"
	agent2 := "agent_2"

	// Create memories for agent 1
	for i := 0; i < 3; i++ {
		mem := &memory.Memory{
			ID:        uuid.New(),
			Scope:     memory.MemoryScopeUser,
			UserID:    &userID,
			AgentID:   agent1,
			Type:      memory.MemoryObservation,
			Content:   "Memory from agent 1",
			Embedding: createTestEmbedding(1536),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Store(ctx, mem)
		require.NoError(t, err)
	}

	// Create memory for agent 2
	mem := &memory.Memory{
		ID:        uuid.New(),
		Scope:     memory.MemoryScopeUser,
		UserID:    &userID,
		AgentID:   agent2,
		Type:      memory.MemoryObservation,
		Content:   "Memory from agent 2",
		Embedding: createTestEmbedding(1536),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Store(ctx, mem)
	require.NoError(t, err)

	// Get by agent 1
	results, err := repo.GetByAgent(ctx, userID, agent1, 10)
	require.NoError(t, err)
	assert.Equal(t, 3, len(results))

	for _, result := range results {
		assert.Equal(t, agent1, result.AgentID)
	}
}

func TestMemoryRepository_GetByType(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	fixtures := NewTestFixtures(t, testDB.DB())
	userID := fixtures.CreateUser()

	repo := NewMemoryRepository(testDB.DB())
	ctx := context.Background()

	// Create memories of different types
	types := []memory.MemoryType{
		memory.MemoryObservation,
		memory.MemoryLesson,
		memory.MemoryObservation,
	}

	for _, memType := range types {
		mem := &memory.Memory{
			ID:        uuid.New(),
			Scope:     memory.MemoryScopeUser,
			UserID:    &userID,
			AgentID:   "test_agent",
			Type:      memType,
			Content:   "Memory of type " + memType.String(),
			Embedding: createTestEmbedding(1536),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := repo.Store(ctx, mem)
		require.NoError(t, err)
	}

	// Get observations
	results, err := repo.GetByType(ctx, userID, memory.MemoryObservation, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))

	for _, result := range results {
		assert.Equal(t, memory.MemoryObservation, result.Type)
	}
}
