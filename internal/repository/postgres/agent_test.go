package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/agent"
	"prometheus/internal/testsupport"
	"prometheus/pkg/errors"
)

func TestAgentRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("creates agent successfully", func(t *testing.T) {
		a := &agent.Agent{
			Identifier:     "test_agent",
			Name:           "Test Agent",
			Description:    "Test agent description",
			Category:       agent.CategoryExpert,
			SystemPrompt:   "You are a test agent",
			ModelProvider:  "anthropic",
			ModelName:      "claude-3-5-haiku-20241022",
			Temperature:    0.5,
			MaxTokens:      1000,
			MaxCostPerRun:  0.10,
			TimeoutSeconds: 60,
			IsActive:       true,
		}

		err := a.SetAvailableTools([]string{"tool1", "tool2"})
		require.NoError(t, err)

		err = repo.Create(ctx, a)
		require.NoError(t, err)
		assert.NotZero(t, a.ID)
		assert.Equal(t, 1, a.Version)
		assert.NotZero(t, a.CreatedAt)
		assert.NotZero(t, a.UpdatedAt)
	})

	t.Run("fails on duplicate identifier", func(t *testing.T) {
		a1 := &agent.Agent{
			Identifier:    "duplicate_test",
			Name:          "Agent 1",
			SystemPrompt:  "Prompt 1",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
		}
		err := repo.Create(ctx, a1)
		require.NoError(t, err)

		a2 := &agent.Agent{
			Identifier:    "duplicate_test", // Same identifier
			Name:          "Agent 2",
			SystemPrompt:  "Prompt 2",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
		}
		err = repo.Create(ctx, a2)
		assert.Error(t, err)
	})
}

func TestAgentRepository_GetByIdentifier(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("retrieves existing agent", func(t *testing.T) {
		// Create
		created := &agent.Agent{
			Identifier:    "get_test",
			Name:          "Get Test Agent",
			SystemPrompt:  "Test prompt",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
		}
		err := repo.Create(ctx, created)
		require.NoError(t, err)

		// Retrieve
		retrieved, err := repo.GetByIdentifier(ctx, "get_test")
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, "get_test", retrieved.Identifier)
		assert.Equal(t, "Get Test Agent", retrieved.Name)
		assert.Equal(t, "Test prompt", retrieved.SystemPrompt)
	})

	t.Run("returns not found for non-existent agent", func(t *testing.T) {
		_, err := repo.GetByIdentifier(ctx, "non_existent")
		assert.ErrorIs(t, err, errors.ErrNotFound)
	})
}

func TestAgentRepository_FindOrCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("creates new agent when not found", func(t *testing.T) {
		a := &agent.Agent{
			Identifier:    "find_or_create_new",
			Name:          "New Agent",
			SystemPrompt:  "New prompt",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
		}

		result, wasCreated, err := repo.FindOrCreate(ctx, a)
		require.NoError(t, err)
		assert.True(t, wasCreated)
		assert.NotZero(t, result.ID)
		assert.Equal(t, "find_or_create_new", result.Identifier)
	})

	t.Run("returns existing agent when found", func(t *testing.T) {
		// Create first time
		a1 := &agent.Agent{
			Identifier:    "find_or_create_existing",
			Name:          "Existing Agent",
			SystemPrompt:  "Original prompt",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
		}
		result1, wasCreated1, err := repo.FindOrCreate(ctx, a1)
		require.NoError(t, err)
		assert.True(t, wasCreated1)
		originalID := result1.ID

		// Try to create again - should return existing
		a2 := &agent.Agent{
			Identifier:    "find_or_create_existing", // Same identifier
			Name:          "Different Name",          // Different data
			SystemPrompt:  "Different prompt",
			ModelProvider: "openai",
			ModelName:     "gpt-4",
		}
		result2, wasCreated2, err := repo.FindOrCreate(ctx, a2)
		require.NoError(t, err)
		assert.False(t, wasCreated2)
		assert.Equal(t, originalID, result2.ID)
		assert.Equal(t, "Existing Agent", result2.Name) // Original name, not changed
		assert.Equal(t, "Original prompt", result2.SystemPrompt)
	})
}

func TestAgentRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("updates agent and increments version", func(t *testing.T) {
		// Create
		a := &agent.Agent{
			Identifier:    "update_test",
			Name:          "Original Name",
			SystemPrompt:  "Original prompt",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
			Temperature:   0.5,
			IsActive:      true,
		}
		err := repo.Create(ctx, a)
		require.NoError(t, err)
		assert.Equal(t, 1, a.Version)
		originalUpdatedAt := a.UpdatedAt

		// Update
		a.Name = "Updated Name"
		a.SystemPrompt = "Updated prompt"
		a.Temperature = 0.7

		err = repo.Update(ctx, a)
		require.NoError(t, err)
		assert.Equal(t, 2, a.Version) // Version auto-incremented by trigger
		assert.True(t, a.UpdatedAt.After(originalUpdatedAt))

		// Verify
		retrieved, err := repo.GetByID(ctx, a.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, "Updated prompt", retrieved.SystemPrompt)
		assert.Equal(t, 0.7, retrieved.Temperature)
		assert.Equal(t, 2, retrieved.Version)
	})
}

func TestAgentRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("lists all agents", func(t *testing.T) {
		// Create multiple agents
		agents := []*agent.Agent{
			{
				Identifier:    "list_test_1",
				Name:          "Agent 1",
				Category:      agent.CategoryCoordinator,
				SystemPrompt:  "Prompt 1",
				ModelProvider: "anthropic",
				ModelName:     "claude-3-5-sonnet-20241022",
			},
			{
				Identifier:    "list_test_2",
				Name:          "Agent 2",
				Category:      agent.CategoryExpert,
				SystemPrompt:  "Prompt 2",
				ModelProvider: "anthropic",
				ModelName:     "claude-3-5-haiku-20241022",
			},
		}

		for _, a := range agents {
			err := repo.Create(ctx, a)
			require.NoError(t, err)
		}

		// List all
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)
	})
}

func TestAgentRepository_ListActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("lists only active agents", func(t *testing.T) {
		// Create active agent
		active := &agent.Agent{
			Identifier:    "active_test",
			Name:          "Active Agent",
			SystemPrompt:  "Active",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
			IsActive:      true,
		}
		err := repo.Create(ctx, active)
		require.NoError(t, err)

		// Create inactive agent
		inactive := &agent.Agent{
			Identifier:    "inactive_test",
			Name:          "Inactive Agent",
			SystemPrompt:  "Inactive",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
			IsActive:      false,
		}
		err = repo.Create(ctx, inactive)
		require.NoError(t, err)

		// List active
		list, err := repo.ListActive(ctx)
		require.NoError(t, err)

		// Check only active returned
		for _, a := range list {
			if a.Identifier == "inactive_test" {
				t.Error("Inactive agent should not be in ListActive result")
			}
		}
	})
}

func TestAgentRepository_ListByCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := testsupport.NewTestPostgres(t)
	defer testDB.Close()

	repo := NewAgentRepository(testDB.Tx())
	ctx := context.Background()

	t.Run("lists agents by category", func(t *testing.T) {
		// Create expert agent
		expert := &agent.Agent{
			Identifier:    "expert_category_test",
			Name:          "Expert",
			Category:      agent.CategoryExpert,
			SystemPrompt:  "Expert",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-haiku-20241022",
			IsActive:      true,
		}
		err := repo.Create(ctx, expert)
		require.NoError(t, err)

		// Create coordinator agent
		coordinator := &agent.Agent{
			Identifier:    "coordinator_category_test",
			Name:          "Coordinator",
			Category:      agent.CategoryCoordinator,
			SystemPrompt:  "Coordinator",
			ModelProvider: "anthropic",
			ModelName:     "claude-3-5-sonnet-20241022",
			IsActive:      true,
		}
		err = repo.Create(ctx, coordinator)
		require.NoError(t, err)

		// List experts only
		experts, err := repo.ListByCategory(ctx, agent.CategoryExpert)
		require.NoError(t, err)

		// Verify only experts returned
		for _, a := range experts {
			assert.Equal(t, agent.CategoryExpert, a.Category)
		}
	})
}
