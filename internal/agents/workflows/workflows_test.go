package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/adk/agent"

	"prometheus/internal/adapters/ai"
	"prometheus/internal/agents"
	"prometheus/internal/tools"
	"prometheus/pkg/templates"
)

func TestWorkflowFactory_CreateParallelAnalysts(t *testing.T) {
	// Create minimal setup for testing
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := tools.NewRegistry()

	// Register at least one tool so agents can be created
	// This is a placeholder - in real scenario we'd register actual tools
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

func TestWorkflowFactory_CreateTradingPipeline(t *testing.T) {
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register("test_tool", createDummyTool())

	agentFactory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	require.NoError(t, err)

	workflowFactory := NewFactory(agentFactory, "test_provider", "test_model")

	// Create trading pipeline
	pipeline, err := workflowFactory.CreateTradingPipeline()

	assert.NotNil(t, pipeline)
	assert.Equal(t, "TradingPipeline", pipeline.Name())

	// Should have 3 sub-agents: parallel analysts, strategy, risk
	subAgents := pipeline.SubAgents()
	assert.Equal(t, 3, len(subAgents), "Pipeline should have 3 steps")
}

func TestWorkflowFactory_CreateExecutionPipeline(t *testing.T) {
	aiRegistry := createTestAIRegistry(t)
	toolRegistry := tools.NewRegistry()
	toolRegistry.Register("test_tool", createDummyTool())

	agentFactory, err := agents.NewFactory(agents.FactoryDeps{
		AIRegistry:   aiRegistry,
		ToolRegistry: toolRegistry,
		Templates:    templates.Get(),
	})
	require.NoError(t, err)

	workflowFactory := NewFactory(agentFactory, "test_provider", "test_model")

	// Create full execution pipeline
	pipeline, err := workflowFactory.CreateExecutionPipeline()

	assert.NotNil(t, pipeline)
	assert.Equal(t, "FullExecutionPipeline", pipeline.Name())
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

func createDummyTool() agent.Tool {
	return &dummyTool{}
}

type dummyTool struct{}

func (d *dummyTool) Name() string        { return "test_tool" }
func (d *dummyTool) Description() string { return "Test tool" }
func (d *dummyTool) IsLongRunning() bool { return false }

func (d *dummyTool) ProcessRequest(ctx agent.Context, req *agent.LLMRequest) error {
	return nil
}

func (d *dummyTool) Run(ctx agent.Context, args map[string]any) (map[string]any, error) {
	return map[string]any{"result": "ok"}, nil
}
