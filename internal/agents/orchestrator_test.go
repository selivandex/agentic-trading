package agents

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/adapters/config"
)

func TestOrchestrator_Creation(t *testing.T) {
	// This is a basic test to ensure orchestrator can be created
	// Full integration test requires real ADK agents which need AI provider connections

	runtimeConfig := config.AgentsConfig{
		MaxTokens:              10000,
		ExecutionTimeout:       1 * time.Minute,
		MaxToolCalls:           10,
		EnableCompression:      true,
		EnableMemory:           false, // Disable for test
		SelfReflectionInterval: 30 * time.Second,
	}

	costTracker := NewCostTracker()

	// Factory, memoryService, and reasoningRepo are nil for this basic test
	orchestrator := NewOrchestrator(nil, runtimeConfig, costTracker, nil, nil)

	if orchestrator == nil {
		t.Fatal("NewOrchestrator returned nil")
	}

	if orchestrator.costTracker != costTracker {
		t.Error("Cost tracker not set correctly")
	}
}

func TestTradePlan_Structure(t *testing.T) {
	// Test that TradePlan structure is properly defined
	userID := uuid.New()

	plan := &TradePlan{
		UserID:     userID,
		Symbol:     "BTC/USDT",
		MarketType: "spot",
		StrategyOutput: &ExecutionOutput{
			AgentType:  AgentStrategyPlanner,
			Confidence: 0.75,
		},
		AnalysisInputs: make(map[AgentType]*ExecutionOutput),
		RiskApproved:   false,
		CreatedAt:      time.Now(),
	}

	if plan.UserID != userID {
		t.Error("UserID not set correctly")
	}

	if plan.Symbol != "BTC/USDT" {
		t.Error("Symbol not set correctly")
	}

	if plan.StrategyOutput.AgentType != AgentStrategyPlanner {
		t.Error("Strategy output agent type incorrect")
	}
}

// TestOrchestrator_RunParallel_EmptyRequests tests error handling
func TestOrchestrator_RunParallel_EmptyRequests(t *testing.T) {
	runtimeConfig := config.AgentsConfig{
		MaxTokens:        10000,
		ExecutionTimeout: 1 * time.Minute,
	}

	orchestrator := NewOrchestrator(nil, runtimeConfig, NewCostTracker(), nil, nil)

	ctx := context.Background()

	// Empty requests should return error
	_, err := orchestrator.RunParallel(ctx, []AgentRequest{}, "claude", "claude-sonnet-4")

	if err == nil {
		t.Error("RunParallel should return error for empty requests")
	}
}

// TestAgentRequest_Structure validates AgentRequest structure
func TestAgentRequest_Structure(t *testing.T) {
	userID := uuid.New()

	req := AgentRequest{
		AgentType: AgentMarketAnalyst,
		Input: ExecutionInput{
			UserID:     userID,
			Symbol:     "BTC/USDT",
			MarketType: "spot",
			Context: map[string]interface{}{
				"timeframe": "1h",
			},
		},
	}

	if req.AgentType != AgentMarketAnalyst {
		t.Error("Agent type not set correctly")
	}

	if req.Input.Symbol != "BTC/USDT" {
		t.Error("Symbol not set correctly")
	}

	if req.Input.Context["timeframe"] != "1h" {
		t.Error("Context not set correctly")
	}
}

// Note: Full integration test with real ADK agents requires:
// 1. AI provider API keys (Claude, OpenAI, etc.)
// 2. Tool registry with real tools
// 3. Database connections for memory/reasoning
// 4. Longer execution time
//
// Such tests should be in a separate integration test suite with build tag:
// // +build integration
//
// Example integration test structure:
// func TestFullAgentPipeline_Integration(t *testing.T) {
//     if testing.Short() {
//         t.Skip("Skipping integration test")
//     }
//
//     // Setup: real AI providers, tools, databases
//     // Run: full analysis pipeline
//     // Assert: results are valid, reasoning logged, costs tracked
// }
