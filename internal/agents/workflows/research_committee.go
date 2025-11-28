package workflows

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/workflowagents/parallelagent"
	"google.golang.org/adk/agent/workflowagents/sequentialagent"

	"prometheus/internal/agents"
	"prometheus/pkg/errors"
)

// CreateResearchCommitteeWorkflow creates the multi-agent Research Committee workflow.
//
// Architecture:
// Stage 1 (Parallel): 4 specialist analysts run concurrently (30-45s)
//   - TechnicalAnalyst: momentum, trend, volatility
//   - StructuralAnalyst: SMC, market structure
//   - FlowAnalyst: order flow, whale activity
//   - MacroAnalyst: economic context, correlations
//
// Stage 2 (Sequential): HeadOfResearch synthesizes all inputs (30-60s)
//   - Reviews all analyst reports
//   - Identifies consensus and conflicts
//   - Resolves disagreements
//   - Conducts pre-mortem
//   - Makes final decision: PUBLISH or SKIP
//
// Total time: 60-90 seconds
// Total cost: ~$0.30-0.50 (4 analysts × $0.03 + head × $0.08)
//
// Input: {symbol: "BTC/USDT", exchange: "binance", ...}
// Output: HeadOfResearch decision with full reasoning trace
func (f *Factory) CreateResearchCommitteeWorkflow() (agent.Agent, error) {
	f.log.Info("Creating Research Committee workflow (Phase 3: multi-agent)")

	// Create 4 specialist analyst agents
	technicalAnalyst, err := f.createAgent(agents.AgentTechnicalAnalyst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TechnicalAnalyst")
	}

	structuralAnalyst, err := f.createAgent(agents.AgentStructuralAnalyst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create StructuralAnalyst")
	}

	flowAnalyst, err := f.createAgent(agents.AgentFlowAnalyst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create FlowAnalyst")
	}

	macroAnalyst, err := f.createAgent(agents.AgentMacroAnalyst)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create MacroAnalyst")
	}

	// Create HeadOfResearch synthesizer
	headOfResearch, err := f.createAgent(agents.AgentHeadOfResearch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HeadOfResearch")
	}

	// Stage 1: Parallel execution of 4 analysts
	// All analysts run concurrently, analyzing the same market from different perspectives
	parallelAnalysts, err := parallelagent.New(parallelagent.Config{
		AgentConfig: agent.Config{
			Name:        "ParallelAnalysts",
			Description: "Four specialist analysts running in parallel: Technical, Structural, Flow, Macro",
			SubAgents: []agent.Agent{
				technicalAnalyst,
				structuralAnalyst,
				flowAnalyst,
				macroAnalyst,
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ParallelAnalysts workflow")
	}

	// Stage 2: Sequential workflow: ParallelAnalysts → HeadOfResearch
	// HeadOfResearch receives all 4 analyst reports and synthesizes them
	researchCommittee, err := sequentialagent.New(sequentialagent.Config{
		AgentConfig: agent.Config{
			Name:        "ResearchCommittee",
			Description: "Multi-agent research team: 4 parallel analysts followed by synthesis and decision",
			SubAgents: []agent.Agent{
				parallelAnalysts, // Stage 1: All analysts run in parallel
				headOfResearch,   // Stage 2: Synthesizer receives all reports
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ResearchCommittee workflow")
	}

	f.log.Info("Research Committee workflow created successfully (4 analysts + 1 synthesizer)")
	return researchCommittee, nil
}

