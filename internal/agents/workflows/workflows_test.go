package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/agents"
	toolspkg "prometheus/internal/tools"
	"prometheus/pkg/templates"
)

func TestWorkflowFactory_CreateParallelAnalysts(t *testing.T) {
	// Create minimal setup for testing
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := toolspkg.NewRegistry()

	// Register at least one tool so agents can be created
	toolRegistry.Register("test_tool", createDummyTool())

	agentFactory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	require.NoError(t, err)

	// Create workflow factory
	workflowFactory := NewFactory(agentFactory, "test_provider", "test_model")

	// Create parallel analysts workflow
	parallelAgent, err := workflowFactory.CreateParallelAnalysts()

	// Should not error even if some analysts fail to create
	assert.NotNil(t, parallelAgent)
	assert.Equal(t, "ParallelAnalysts", parallelAgent.Name())

	// Should have sub-agents
	subAgents := parallelAgent.SubAgents()
	assert.Greater(t, len(subAgents), 0, "Should have at least one analyst")
}

func TestWorkflowFactory_CreateMarketResearchWorkflow(t *testing.T) {
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := toolspkg.NewRegistry()
	toolRegistry.Register("test_tool", createDummyTool())

	agentFactory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	require.NoError(t, err)

	workflowFactory := NewFactory(agentFactory, "test_provider", "test_model")

	// Create market research workflow
	workflow, err := workflowFactory.CreateMarketResearchWorkflow()

	assert.NotNil(t, workflow)
	assert.Equal(t, "MarketResearchWorkflow", workflow.Name())

	// Should have 2 sub-agents: parallel analysts, synthesizer
	subAgents := workflow.SubAgents()
	assert.Equal(t, 2, len(subAgents), "Workflow should have 2 steps")
}

func TestWorkflowFactory_CreatePersonalTradingWorkflow(t *testing.T) {
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := toolspkg.NewRegistry()
	toolRegistry.Register("test_tool", createDummyTool())

	agentFactory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	require.NoError(t, err)

	workflowFactory := NewFactory(agentFactory, "test_provider", "test_model")

	// Create personal trading workflow
	workflow, err := workflowFactory.CreatePersonalTradingWorkflow()

	assert.NotNil(t, workflow)
	assert.Equal(t, "PersonalTradingWorkflow", workflow.Name())

	// Should have 3 sub-agents: strategy, risk, executor
	subAgents := workflow.SubAgents()
	assert.Equal(t, 3, len(subAgents), "Workflow should have 3 steps")
}

func TestWorkflowFactory_CreatePortfolioInitializationWorkflow(t *testing.T) {
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := toolspkg.NewRegistry()
	toolRegistry.Register("test_tool", createDummyTool())

	agentFactory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	require.NoError(t, err)

	workflowFactory := NewFactory(agentFactory, "test_provider", "test_model")

	// Create portfolio initialization workflow
	workflow, err := workflowFactory.CreatePortfolioInitializationWorkflow()

	assert.NotNil(t, workflow)
	assert.Equal(t, "PortfolioInitializationWorkflow", workflow.Name())

	// Should have 4 sub-agents: market, architect, risk, executor
	subAgents := workflow.SubAgents()
	assert.Equal(t, 4, len(subAgents), "Workflow should have 4 steps")
}

// Helper functions

func createTestAIRegistry(t *testing.T) *ai.ProviderRegistry {
	registry := ai.NewProviderRegistry()

	// Register a test provider
	provider := &testProvider{}
	registry.Register(provider)

	return registry
}

type testProvider struct{}

func (p *testProvider) Name() string { return "test_provider" }

func (p *testProvider) GetModel(ctx context.Context, model string) (ai.ModelInfo, error) {
	return ai.ModelInfo{
		Name:              "test_model",
		Family:            "test",
		MaxTokens:         4096,
		InputCostPer1K:    0.001,
		OutputCostPer1K:   0.002,
		SupportsTools:     true,
		SupportsStreaming: false,
	}, nil
}

func (p *testProvider) ListModels(ctx context.Context) ([]ai.ModelInfo, error) {
	return []ai.ModelInfo{}, nil
}

func (p *testProvider) SupportsStreaming() bool { return false }
func (p *testProvider) SupportsTools() bool     { return true }

func (p *testProvider) Chat(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	return &ai.ChatResponse{
		Choices: []ai.Choice{
			{
				Message: ai.Message{
					Role:    ai.RoleAssistant,
					Content: "Test response",
				},
				FinishReason: ai.FinishReasonStop,
			},
		},
		Usage: ai.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (p *testProvider) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatStreamChunk, <-chan error) {
	errCh := make(chan error, 1)
	close(errCh)
	return nil, errCh
}

// createDummyTool creates a minimal tool for testing using ADK functiontool
func createDummyTool() tool.Tool {
	return functiontool.New(functiontool.Config{
		Name:        "test_tool",
		Description: "Test tool for workflow testing",
		Func: func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	})
}
