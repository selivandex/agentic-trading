package workflows

import (
	"context"
	"fmt"
	"iter"
	"testing"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// createMockAgent creates a simple mock agent for testing
func createMockAgent(name string) agent.Agent {
	ag, err := agent.New(agent.Config{
		Name: name,
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				yield(&session.Event{
					LLMResponse: model.LLMResponse{
						Content: genai.NewContentFromText(fmt.Sprintf("Response from %s", name), genai.RoleModel),
					},
				}, nil)
			}
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create mock agent: %v", err))
	}
	return ag
}

func TestPathSelector_PriorityRouting(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		ForceCommitteeOnHigh:  true,
		HighStakesThreshold:   10.0,
		VolatilityThreshold:   5.0,
		CommitteeUsagePercent: 100.0, // Disable cost control for testing
	})

	tests := []struct {
		name          string
		request       AnalysisRequest
		expectedAgent string
		description   string
	}{
		{
			name: "High priority routes to committee",
			request: AnalysisRequest{
				Symbol:   "BTC/USDT",
				Exchange: "binance",
				Priority: PriorityHigh,
			},
			expectedAgent: "Committee",
			description:   "High priority should always use committee when ForceCommitteeOnHigh=true",
		},
		{
			name: "Critical priority routes to committee",
			request: AnalysisRequest{
				Symbol:   "BTC/USDT",
				Exchange: "binance",
				Priority: PriorityCritical,
			},
			expectedAgent: "Committee",
			description:   "Critical priority should always use committee",
		},
		{
			name: "Low priority routes to fast-path",
			request: AnalysisRequest{
				Symbol:   "DOGE/USDT",
				Exchange: "binance",
				Priority: PriorityLow,
			},
			expectedAgent: "FastPath",
			description:   "Low priority should use fast-path by default",
		},
		{
			name: "Medium priority routes to fast-path",
			request: AnalysisRequest{
				Symbol:   "ETH/USDT",
				Exchange: "binance",
				Priority: PriorityMedium,
			},
			expectedAgent: "FastPath",
			description:   "Medium priority should use fast-path by default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			selected := selector.SelectPath(ctx, tt.request)

			agentName := selected.Name()
			if agentName != tt.expectedAgent {
				t.Errorf("%s: expected %s, got %s", tt.description, tt.expectedAgent, agentName)
			}
		})
	}
}

func TestPathSelector_HighStakesRouting(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		HighStakesThreshold:   10.0, // 10% threshold
		VolatilityThreshold:   5.0,
		ForceCommitteeOnHigh:  true,
		CommitteeUsagePercent: 100.0, // Disable cost control
	})

	tests := []struct {
		name            string
		positionSizePct float64
		expectedAgent   string
		description     string
	}{
		{
			name:            "Small position uses fast-path",
			positionSizePct: 5.0,
			expectedAgent:   "FastPath",
			description:     "5% position is below 10% threshold",
		},
		{
			name:            "Threshold position uses committee",
			positionSizePct: 10.0,
			expectedAgent:   "Committee",
			description:     "10% position equals threshold, should use committee",
		},
		{
			name:            "Large position uses committee",
			positionSizePct: 15.0,
			expectedAgent:   "Committee",
			description:     "15% position exceeds threshold, should use committee",
		},
		{
			name:            "Very large position uses committee",
			positionSizePct: 25.0,
			expectedAgent:   "Committee",
			description:     "25% position well above threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := AnalysisRequest{
				Symbol:          "BTC/USDT",
				Exchange:        "binance",
				Priority:        PriorityMedium,
				PositionSizePct: tt.positionSizePct,
			}

			selected := selector.SelectPath(ctx, req)
			agentName := selected.Name()

			if agentName != tt.expectedAgent {
				t.Errorf("%s: expected %s, got %s", tt.description, tt.expectedAgent, agentName)
			}
		})
	}
}

func TestPathSelector_VolatilityRouting(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		HighStakesThreshold:   10.0,
		VolatilityThreshold:   5.0, // 5% threshold
		ForceCommitteeOnHigh:  true,
		CommitteeUsagePercent: 100.0,
	})

	tests := []struct {
		name          string
		volatility    float64
		expectedAgent string
		description   string
	}{
		{
			name:          "Low volatility uses fast-path",
			volatility:    2.0,
			expectedAgent: "FastPath",
			description:   "2% volatility is below 5% threshold",
		},
		{
			name:          "Threshold volatility uses committee",
			volatility:    5.0,
			expectedAgent: "Committee",
			description:   "5% volatility equals threshold",
		},
		{
			name:          "High volatility uses committee",
			volatility:    8.0,
			expectedAgent: "Committee",
			description:   "8% volatility exceeds threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := AnalysisRequest{
				Symbol:     "BTC/USDT",
				Exchange:   "binance",
				Priority:   PriorityMedium,
				Volatility: tt.volatility,
			}

			selected := selector.SelectPath(ctx, req)
			agentName := selected.Name()

			if agentName != tt.expectedAgent {
				t.Errorf("%s: expected %s, got %s", tt.description, tt.expectedAgent, agentName)
			}
		})
	}
}

func TestPathSelector_ConflictingSignalsRouting(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		HighStakesThreshold:   10.0,
		VolatilityThreshold:   5.0,
		ForceCommitteeOnHigh:  true,
		CommitteeUsagePercent: 100.0,
	})

	t.Run("Conflicting signals escalate to committee", func(t *testing.T) {
		ctx := context.Background()
		req := AnalysisRequest{
			Symbol:             "BTC/USDT",
			Exchange:           "binance",
			Priority:           PriorityMedium,
			ConflictingSignals: true,
		}

		selected := selector.SelectPath(ctx, req)
		if selected.Name() != "Committee" {
			t.Errorf("Conflicting signals should escalate to committee, got %s", selected.Name())
		}
	})

	t.Run("No conflicts use fast-path", func(t *testing.T) {
		ctx := context.Background()
		req := AnalysisRequest{
			Symbol:             "BTC/USDT",
			Exchange:           "binance",
			Priority:           PriorityMedium,
			ConflictingSignals: false,
		}

		selected := selector.SelectPath(ctx, req)
		if selected.Name() != "FastPath" {
			t.Errorf("No conflicts should use fast-path, got %s", selected.Name())
		}
	})
}

func TestPathSelector_CostControl(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		HighStakesThreshold:   10.0,
		VolatilityThreshold:   5.0,
		ForceCommitteeOnHigh:  true,
		CommitteeUsagePercent: 20.0, // 20% cost control
	})

	ctx := context.Background()

	// Send 10 high-priority requests
	committeeCount := 0
	for i := 0; i < 10; i++ {
		req := AnalysisRequest{
			Symbol:   "BTC/USDT",
			Exchange: "binance",
			Priority: PriorityHigh,
		}

		selected := selector.SelectPath(ctx, req)
		if selected.Name() == "Committee" {
			committeeCount++
		}
	}

	// Should have routed ~2-3 to committee (20%), rest downgraded to fast-path
	// Allow some tolerance
	if committeeCount < 1 || committeeCount > 4 {
		t.Errorf("Cost control failed: expected 2-3 committee requests (20%%), got %d", committeeCount)
	}

	metrics := selector.GetMetrics()
	usage := metrics["committee_usage_pct"].(float64)

	if usage < 10.0 || usage > 40.0 {
		t.Errorf("Committee usage outside expected range: got %.1f%%, expected 10-40%%", usage)
	}
}

func TestPathSelector_Metrics(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		HighStakesThreshold:   10.0,
		VolatilityThreshold:   5.0,
		ForceCommitteeOnHigh:  true,
		CommitteeUsagePercent: 100.0,
	})

	ctx := context.Background()

	// Send mix of requests
	requests := []AnalysisRequest{
		{Priority: PriorityHigh},     // Committee
		{Priority: PriorityMedium},   // Fast-path
		{Priority: PriorityLow},      // Fast-path
		{Priority: PriorityCritical}, // Committee
		{Volatility: 6.0},            // Committee
	}

	for _, req := range requests {
		req.Symbol = "BTC/USDT"
		req.Exchange = "binance"
		selector.SelectPath(ctx, req)
	}

	metrics := selector.GetMetrics()

	if metrics["total_requests"].(int) != 5 {
		t.Errorf("Expected 5 total requests, got %d", metrics["total_requests"])
	}

	if metrics["committee_requests"].(int) != 3 {
		t.Errorf("Expected 3 committee requests, got %d", metrics["committee_requests"])
	}

	usage := metrics["committee_usage_pct"].(float64)
	// Expected ~60% (3/5)
	if usage < 59.0 || usage > 61.0 {
		t.Errorf("Expected ~60%% committee usage, got %.1f%%", usage)
	}
}

func TestPathSelector_ResetMetrics(t *testing.T) {
	fastPath := createMockAgent("FastPath")
	committePath := createMockAgent("Committee")

	selector := NewPathSelector(PathSelectorConfig{
		FastPath:              fastPath,
		CommitteePath:         committePath,
		HighStakesThreshold:   10.0,
		VolatilityThreshold:   5.0,
		ForceCommitteeOnHigh:  true,
		CommitteeUsagePercent: 100.0,
	})

	ctx := context.Background()

	// Send some requests
	for i := 0; i < 5; i++ {
		selector.SelectPath(ctx, AnalysisRequest{
			Symbol:   "BTC/USDT",
			Exchange: "binance",
			Priority: PriorityHigh,
		})
	}

	// Verify metrics exist
	metrics := selector.GetMetrics()
	if metrics["total_requests"].(int) != 5 {
		t.Errorf("Expected 5 requests before reset, got %d", metrics["total_requests"])
	}

	// Reset
	selector.ResetMetrics()

	// Verify reset
	metrics = selector.GetMetrics()
	if metrics["total_requests"].(int) != 0 {
		t.Errorf("Expected 0 requests after reset, got %d", metrics["total_requests"])
	}
	if metrics["committee_requests"].(int) != 0 {
		t.Errorf("Expected 0 committee requests after reset, got %d", metrics["committee_requests"])
	}
}
