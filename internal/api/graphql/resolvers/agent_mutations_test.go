package resolvers_test

import (
	"context"
	"testing"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/api/graphql/generated"
	"prometheus/internal/domain/agent"
	"prometheus/internal/testsupport/seeds"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAgentMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		input       generated.CreateAgentInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *agent.Agent)
	}{
		{
			name: "successful creation",
			input: generated.CreateAgentInput{
				Identifier:    "test_agent_" + randomString(8),
				Name:          "Test Agent",
				Description:   "Test Description",
				Category:      agent.CategoryExpert,
				SystemPrompt:  "You are a test agent",
				ModelProvider: string(ai.ProviderNameDeepSeek),
				ModelName:     string(ai.ModelDeepSeekReasoner),
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.NotZero(t, result.ID)
				assert.Equal(t, "Test Agent", result.Name)
				assert.Equal(t, agent.CategoryExpert, result.Category)
				assert.True(t, result.IsActive) // default is true
				assert.Equal(t, 1.0, result.Temperature)
				assert.Equal(t, 4000, result.MaxTokens)
			},
		},
		{
			name: "with custom settings",
			input: generated.CreateAgentInput{
				Identifier:     "custom_agent_" + randomString(8),
				Name:           "Custom Agent",
				Description:    "Custom settings",
				Category:       agent.CategoryCoordinator,
				SystemPrompt:   "Custom prompt",
				Instructions:   strPtr("Custom instructions"),
				ModelProvider:  string(ai.ProviderNameDeepSeek),
				ModelName:      string(ai.ModelDeepSeekReasoner),
				Temperature:    float64Ptr(0.5),
				MaxTokens:      intPtr(8000),
				MaxCostPerRun:  float64Ptr(1.0),
				TimeoutSeconds: intPtr(120),
				AvailableTools: []string{"tool1", "tool2"},
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.Equal(t, 0.5, result.Temperature)
				assert.Equal(t, 8000, result.MaxTokens)
				assert.Equal(t, 1.0, result.MaxCostPerRun)
				assert.Equal(t, 120, result.TimeoutSeconds)
				tools, err := result.GetAvailableTools()
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"tool1", "tool2"}, tools)
			},
		},
		{
			name: "inactive agent",
			input: generated.CreateAgentInput{
				Identifier:    "inactive_agent_" + randomString(8),
				Name:          "Inactive Agent",
				Description:   "Inactive",
				Category:      agent.CategorySpecialist,
				SystemPrompt:  "Inactive prompt",
				ModelProvider: string(ai.ProviderNameDeepSeek),
				ModelName:     string(ai.ModelDeepSeekReasoner),
				IsActive:      boolPtr(false),
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.False(t, result.IsActive)
			},
		},
		{
			name: "missing required identifier",
			input: generated.CreateAgentInput{
				Name:          "No Identifier",
				Description:   "Missing identifier",
				Category:      agent.CategoryExpert,
				SystemPrompt:  "Prompt",
				ModelProvider: string(ai.ProviderNameDeepSeek),
				ModelName:     string(ai.ModelDeepSeekReasoner),
			},
			wantErr:     true,
			errContains: "identifier is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().CreateAgent(ctx, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestUpdateAgentMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test agent using seeds
	seeder := seeds.New(setup.pg.DB())
	testAgent := seeder.Agent().
		WithIdentifier("test_update_" + randomString(8)).
		WithName("Original Name").
		WithDescription("Original Description").
		MustInsert()

	tests := []struct {
		name        string
		agentID     int
		input       generated.UpdateAgentInput
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result *agent.Agent)
	}{
		{
			name:    "update name only",
			agentID: testAgent.ID,
			input: generated.UpdateAgentInput{
				Name: strPtr("Updated Name"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.Equal(t, "Updated Name", result.Name)
				assert.Equal(t, "Original Description", result.Description) // unchanged
			},
		},
		{
			name:    "update multiple fields",
			agentID: testAgent.ID,
			input: generated.UpdateAgentInput{
				Name:        strPtr("Multi Update"),
				Description: strPtr("New Description"),
				Category:    strPtr(agent.CategoryCoordinator),
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.Equal(t, "Multi Update", result.Name)
				assert.Equal(t, "New Description", result.Description)
				assert.Equal(t, agent.CategoryCoordinator, result.Category)
			},
		},
		{
			name:    "update model settings",
			agentID: testAgent.ID,
			input: generated.UpdateAgentInput{
				Temperature:   float64Ptr(0.7),
				MaxTokens:     intPtr(6000),
				MaxCostPerRun: float64Ptr(0.50),
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.Equal(t, 0.7, result.Temperature)
				assert.Equal(t, 6000, result.MaxTokens)
				assert.Equal(t, 0.50, result.MaxCostPerRun)
			},
		},
		{
			name:    "update available tools",
			agentID: testAgent.ID,
			input: generated.UpdateAgentInput{
				AvailableTools: []string{"new_tool1", "new_tool2", "new_tool3"},
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				tools, err := result.GetAvailableTools()
				require.NoError(t, err)
				assert.ElementsMatch(t, []string{"new_tool1", "new_tool2", "new_tool3"}, tools)
			},
		},
		{
			name:    "update system prompt",
			agentID: testAgent.ID,
			input: generated.UpdateAgentInput{
				SystemPrompt: strPtr("New system prompt for testing"),
			},
			wantErr: false,
			validate: func(t *testing.T, result *agent.Agent) {
				assert.Equal(t, "New system prompt for testing", result.SystemPrompt)
			},
		},
		{
			name:        "non-existent agent",
			agentID:     99999,
			input:       generated.UpdateAgentInput{Name: strPtr("Test")},
			wantErr:     true,
			errContains: "failed to get agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().UpdateAgent(ctx, tt.agentID, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.agentID, result.ID)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestDeleteAgentMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create agent to delete
	seeder := seeds.New(setup.pg.DB())
	agentToDelete := seeder.Agent().
		WithIdentifier("delete_test_" + randomString(8)).
		MustInsert()

	tests := []struct {
		name        string
		agentID     int
		wantErr     bool
		errContains string
	}{
		{
			name:    "delete existing agent",
			agentID: agentToDelete.ID,
			wantErr: false,
		},
		{
			name:        "delete non-existent agent",
			agentID:     99999,
			wantErr:     true,
			errContains: "failed to get agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().DeleteAgent(ctx, tt.agentID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, result)

			// Verify agent is actually deleted
			_, err = setup.resolver.Query().Agent(ctx, tt.agentID)
			assert.Error(t, err) // Should not find deleted agent
		})
	}
}

func TestSetAgentActiveMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test agent
	seeder := seeds.New(setup.pg.DB())
	testAgent := seeder.Agent().
		WithIdentifier("active_test_" + randomString(8)).
		WithIsActive(true).
		MustInsert()

	tests := []struct {
		name        string
		agentID     int
		isActive    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "deactivate agent",
			agentID:  testAgent.ID,
			isActive: false,
			wantErr:  false,
		},
		{
			name:     "activate agent",
			agentID:  testAgent.ID,
			isActive: true,
			wantErr:  false,
		},
		{
			name:        "non-existent agent",
			agentID:     99999,
			isActive:    true,
			wantErr:     true,
			errContains: "failed to get agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().SetAgentActive(ctx, tt.agentID, tt.isActive)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.agentID, result.ID)
			assert.Equal(t, tt.isActive, result.IsActive)
		})
	}
}

func TestUpdateAgentPromptMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Create test agent
	seeder := seeds.New(setup.pg.DB())
	testAgent := seeder.Agent().
		WithIdentifier("prompt_test_" + randomString(8)).
		WithSystemPrompt("Original prompt").
		MustInsert()

	originalVersion := testAgent.Version

	tests := []struct {
		name         string
		agentID      int
		systemPrompt string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "update prompt successfully",
			agentID:      testAgent.ID,
			systemPrompt: "Updated prompt for testing agent behavior",
			wantErr:      false,
		},
		{
			name:         "non-existent agent",
			agentID:      99999,
			systemPrompt: "Test prompt",
			wantErr:      true,
			errContains:  "failed to get agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := setup.resolver.Mutation().UpdateAgentPrompt(ctx, tt.agentID, tt.systemPrompt)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.agentID, result.ID)
			assert.Equal(t, tt.systemPrompt, result.SystemPrompt)
			// Version should be incremented by DB trigger
			assert.Greater(t, result.Version, originalVersion)
		})
	}
}

func TestEnsureSystemAgentsMutation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	t.Run("ensure system agents", func(t *testing.T) {
		result, err := setup.resolver.Mutation().EnsureSystemAgents(ctx)
		require.NoError(t, err)
		assert.True(t, result)

		// Verify that system agents were created
		agents, err := setup.resolver.Query().Agents(ctx, nil, nil, intPtr(100), nil, nil, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, agents.Edges)

		// Check for well-known agents
		agentIdentifiers := make(map[string]bool)
		for _, edge := range agents.Edges {
			agentIdentifiers[edge.Node.Identifier] = true
		}

		// At least portfolio_architect should exist
		assert.True(t, agentIdentifiers[agent.IdentifierPortfolioArchitect] ||
			agentIdentifiers[agent.IdentifierPortfolioManager] ||
			agentIdentifiers[agent.IdentifierTechnicalAnalyzer],
			"At least one system agent should be created")
	})

	t.Run("ensure is idempotent", func(t *testing.T) {
		// Call again - should not create duplicates
		result, err := setup.resolver.Mutation().EnsureSystemAgents(ctx)
		require.NoError(t, err)
		assert.True(t, result)
	})
}

func TestAgentMutationsWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	setup := setupResolverTest(t)
	defer setup.pg.Close()

	ctx := context.Background()

	// Full lifecycle test: create → update → update prompt → deactivate → activate → delete
	t.Run("complete agent lifecycle", func(t *testing.T) {
		// 1. Create agent
		createInput := generated.CreateAgentInput{
			Identifier:    "lifecycle_agent_" + randomString(8),
			Name:          "Lifecycle Test Agent",
			Description:   "Testing full lifecycle",
			Category:      agent.CategoryExpert,
			SystemPrompt:  "Initial prompt",
			ModelProvider: string(ai.ProviderNameDeepSeek),
			ModelName:     string(ai.ModelDeepSeekReasoner),
			Temperature:   float64Ptr(1.0),
		}

		created, err := setup.resolver.Mutation().CreateAgent(ctx, createInput)
		require.NoError(t, err)
		assert.True(t, created.IsActive)
		assert.Equal(t, "Lifecycle Test Agent", created.Name)

		// 2. Update agent
		updateInput := generated.UpdateAgentInput{
			Name:        strPtr("Updated Lifecycle Agent"),
			Description: strPtr("Updated description"),
			Temperature: float64Ptr(0.5),
		}

		updated, err := setup.resolver.Mutation().UpdateAgent(ctx, created.ID, updateInput)
		require.NoError(t, err)
		assert.Equal(t, "Updated Lifecycle Agent", updated.Name)
		assert.Equal(t, 0.5, updated.Temperature)

		// 3. Update prompt
		newPrompt := "Updated system prompt for better performance"
		withNewPrompt, err := setup.resolver.Mutation().UpdateAgentPrompt(ctx, created.ID, newPrompt)
		require.NoError(t, err)
		assert.Equal(t, newPrompt, withNewPrompt.SystemPrompt)

		// 4. Deactivate agent
		deactivated, err := setup.resolver.Mutation().SetAgentActive(ctx, created.ID, false)
		require.NoError(t, err)
		assert.False(t, deactivated.IsActive)

		// 5. Activate agent
		activated, err := setup.resolver.Mutation().SetAgentActive(ctx, created.ID, true)
		require.NoError(t, err)
		assert.True(t, activated.IsActive)

		// 6. Delete agent
		deleted, err := setup.resolver.Mutation().DeleteAgent(ctx, created.ID)
		require.NoError(t, err)
		assert.True(t, deleted)
	})
}

// Helper to generate random string for unique identifiers
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
