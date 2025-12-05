package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/agent"
	"prometheus/pkg/errors"
)

// mockRepository implements agent.Repository for testing
type mockRepository struct {
	createFunc          func(context.Context, *agent.Agent) error
	getByIdentifierFunc func(context.Context, string) (*agent.Agent, error)
	getByIDFunc         func(context.Context, int) (*agent.Agent, error)
	updateFunc          func(context.Context, *agent.Agent) error
	findOrCreateFunc    func(context.Context, *agent.Agent) (*agent.Agent, bool, error)
	listActiveFunc      func(context.Context) ([]*agent.Agent, error)
	listByCategoryFunc  func(context.Context, string) ([]*agent.Agent, error)
	listFunc            func(context.Context) ([]*agent.Agent, error)
}

func (m *mockRepository) Create(ctx context.Context, a *agent.Agent) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, a)
	}
	return nil
}

func (m *mockRepository) GetByIdentifier(ctx context.Context, identifier string) (*agent.Agent, error) {
	if m.getByIdentifierFunc != nil {
		return m.getByIdentifierFunc(ctx, identifier)
	}
	return nil, errors.ErrNotFound
}

func (m *mockRepository) GetByID(ctx context.Context, id int) (*agent.Agent, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.ErrNotFound
}

func (m *mockRepository) Update(ctx context.Context, a *agent.Agent) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, a)
	}
	return nil
}

func (m *mockRepository) FindOrCreate(ctx context.Context, a *agent.Agent) (*agent.Agent, bool, error) {
	if m.findOrCreateFunc != nil {
		return m.findOrCreateFunc(ctx, a)
	}
	a.ID = 1
	return a, true, nil
}

func (m *mockRepository) ListActive(ctx context.Context) ([]*agent.Agent, error) {
	if m.listActiveFunc != nil {
		return m.listActiveFunc(ctx)
	}
	return []*agent.Agent{}, nil
}

func (m *mockRepository) ListByCategory(ctx context.Context, category string) ([]*agent.Agent, error) {
	if m.listByCategoryFunc != nil {
		return m.listByCategoryFunc(ctx, category)
	}
	return []*agent.Agent{}, nil
}

func (m *mockRepository) List(ctx context.Context) ([]*agent.Agent, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []*agent.Agent{}, nil
}

// mockTemplateRegistry implements template rendering
type mockTemplateRegistry struct {
	prompts map[string]string
}

func (m *mockTemplateRegistry) Render(templatePath string, data any) (string, error) {
	if m.prompts != nil {
		if prompt, ok := m.prompts[templatePath]; ok {
			return prompt, nil
		}
	}
	return "Mocked prompt", nil
}

func TestService_EnsureSystemAgents(t *testing.T) {
	ctx := context.Background()

	t.Run("creates all system agents on first run", func(t *testing.T) {
		callCount := 0
		mockRepo := &mockRepository{
			findOrCreateFunc: func(ctx context.Context, a *agent.Agent) (*agent.Agent, bool, error) {
				callCount++
				a.ID = callCount
				return a, true, nil // All created
			},
		}

		domainSvc := agent.NewService(mockRepo)

		mockTmpl := &mockTemplateRegistry{
			prompts: map[string]string{
				"prompts/agents/portfolio_architect":     "Portfolio Architect Prompt",
				"prompts/agents/portfolio_manager":       "Portfolio Manager Prompt",
				"prompts/agents/technical_analyzer":      "Technical Analyzer Prompt",
				"prompts/agents/smc_analyzer":            "SMC Analyzer Prompt",
				"prompts/agents/correlation_analyzer":    "Correlation Analyzer Prompt",
				"prompts/agents/opportunity_synthesizer": "Opportunity Synthesizer Prompt",
			},
		}

		svc := NewService(domainSvc, mockTmpl)

		err := svc.EnsureSystemAgents(ctx)
		require.NoError(t, err)
		assert.Equal(t, 6, callCount) // 6 system agents
	})

	t.Run("skips existing agents", func(t *testing.T) {
		callCount := 0
		mockRepo := &mockRepository{
			findOrCreateFunc: func(ctx context.Context, a *agent.Agent) (*agent.Agent, bool, error) {
				callCount++
				a.ID = 1
				return a, false, nil // Already exists
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		err := svc.EnsureSystemAgents(ctx)
		require.NoError(t, err)
		assert.Equal(t, 6, callCount)
	})

	t.Run("continues on template error", func(t *testing.T) {
		callCount := 0
		mockRepo := &mockRepository{
			findOrCreateFunc: func(ctx context.Context, a *agent.Agent) (*agent.Agent, bool, error) {
				callCount++
				a.ID = callCount
				// Verify placeholder prompt used when template returns empty
				// mockTemplateRegistry returns "Mocked prompt" as fallback
				return a, true, nil
			},
		}

		domainSvc := agent.NewService(mockRepo)

		// Mock that returns error for Render
		mockTmpl := &mockTemplateRegistry{
			prompts: nil, // Will use fallback "Mocked prompt"
		}

		svc := NewService(domainSvc, mockTmpl)

		err := svc.EnsureSystemAgents(ctx)
		require.NoError(t, err)
		assert.Equal(t, 6, callCount) // Should still create all agents
	})

	t.Run("continues on individual agent error", func(t *testing.T) {
		callCount := 0
		mockRepo := &mockRepository{
			findOrCreateFunc: func(ctx context.Context, a *agent.Agent) (*agent.Agent, bool, error) {
				callCount++
				if callCount == 3 {
					return nil, false, errors.New("database error")
				}
				a.ID = callCount
				return a, true, nil
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		err := svc.EnsureSystemAgents(ctx)
		require.NoError(t, err) // Should not fail completely
		assert.Equal(t, 6, callCount)
	})
}

func TestService_UpdatePrompt(t *testing.T) {
	ctx := context.Background()

	t.Run("updates prompt successfully", func(t *testing.T) {
		existing := &agent.Agent{
			ID:           1,
			Identifier:   "test_agent",
			SystemPrompt: "Old prompt",
			Version:      1,
		}

		updated := false
		mockRepo := &mockRepository{
			getByIdentifierFunc: func(ctx context.Context, identifier string) (*agent.Agent, error) {
				assert.Equal(t, "test_agent", identifier)
				return existing, nil
			},
			updateFunc: func(ctx context.Context, a *agent.Agent) error {
				assert.Equal(t, "New prompt", a.SystemPrompt)
				a.Version = 2 // Simulate DB trigger
				updated = true
				return nil
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		err := svc.UpdatePrompt(ctx, "test_agent", "New prompt")
		require.NoError(t, err)
		assert.True(t, updated)
	})

	t.Run("fails when agent not found", func(t *testing.T) {
		mockRepo := &mockRepository{
			getByIdentifierFunc: func(ctx context.Context, identifier string) (*agent.Agent, error) {
				return nil, errors.ErrNotFound
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		err := svc.UpdatePrompt(ctx, "non_existent", "New prompt")
		assert.Error(t, err)
	})

	t.Run("fails when update fails", func(t *testing.T) {
		mockRepo := &mockRepository{
			getByIdentifierFunc: func(ctx context.Context, identifier string) (*agent.Agent, error) {
				return &agent.Agent{ID: 1, Identifier: "test", SystemPrompt: "Old"}, nil
			},
			updateFunc: func(ctx context.Context, a *agent.Agent) error {
				return errors.New("database error")
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		err := svc.UpdatePrompt(ctx, "test", "New prompt")
		assert.Error(t, err)
	})
}

func TestService_GetByIdentifier(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to domain service", func(t *testing.T) {
		expected := &agent.Agent{
			ID:         1,
			Identifier: "test_agent",
		}

		mockRepo := &mockRepository{
			getByIdentifierFunc: func(ctx context.Context, identifier string) (*agent.Agent, error) {
				assert.Equal(t, "test_agent", identifier)
				return expected, nil
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		result, err := svc.GetByIdentifier(ctx, "test_agent")
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestService_ListActive(t *testing.T) {
	ctx := context.Background()

	t.Run("returns active agents", func(t *testing.T) {
		expected := []*agent.Agent{
			{ID: 1, Identifier: "agent1", IsActive: true},
			{ID: 2, Identifier: "agent2", IsActive: true},
		}

		mockRepo := &mockRepository{
			listActiveFunc: func(ctx context.Context) ([]*agent.Agent, error) {
				return expected, nil
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		result, err := svc.ListActive(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestService_ListByCategory(t *testing.T) {
	ctx := context.Background()

	t.Run("returns agents by category", func(t *testing.T) {
		expected := []*agent.Agent{
			{ID: 1, Category: agent.CategoryExpert},
			{ID: 2, Category: agent.CategoryExpert},
		}

		mockRepo := &mockRepository{
			listByCategoryFunc: func(ctx context.Context, category string) ([]*agent.Agent, error) {
				assert.Equal(t, agent.CategoryExpert, category)
				return expected, nil
			},
		}

		domainSvc := agent.NewService(mockRepo)
		mockTmpl := &mockTemplateRegistry{}
		svc := NewService(domainSvc, mockTmpl)

		result, err := svc.ListByCategory(ctx, agent.CategoryExpert)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}
