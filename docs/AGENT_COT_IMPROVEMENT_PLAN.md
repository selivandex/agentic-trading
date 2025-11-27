<!-- @format -->

# Agent Chain-of-Thought (CoT) Improvement Plan

**Status:** Draft  
**Priority:** Critical  
**Estimated effort:** 2-3 weeks  
**Created:** 2025-11-27

---

## Executive Summary

Current agent implementation correctly uses ADK workflow primitives (parallelagent, sequentialagent) and has comprehensive callbacks, but **fails to leverage structured Chain-of-Thought reasoning**. This creates critical audit and debugging issues, especially for OpportunitySynthesizer which makes trading decisions.

**Key Issues:**

1. ❌ CoT described in prompts but not enforced via OutputSchema
2. ❌ No structured reasoning trace logged or persisted
3. ❌ Sequential workflows don't pass reasoning context between agents
4. ❌ InstructionProvider (dynamic prompts) completely unused
5. ❌ State management doesn't store reasoning history

**Impact:**

- Cannot audit why OpportunitySynthesizer published/skipped signals
- Cannot debug bad trading decisions (no reasoning trace)
- Cannot replay or validate agent logic
- Missing opportunity for self-correction mechanisms

---

## Phase 1: Critical — OpportunitySynthesizer Structured CoT

**Priority:** 🔥 CRITICAL  
**Timeline:** Week 1  
**Blockers:** None

### 1.1 Define OpportunitySynthesizer OutputSchema

**File:** `internal/agents/schemas/opportunity_synthesizer.go`

**Create structured output schema:**

```go
OpportunitySynthesizerOutputSchema = &genai.Schema{
    Type: genai.TypeObject,
    Properties: {
        "synthesis_steps": TypeArray of reasoning steps
        "decision": TypeObject with action, consensus, confidence
        "conflicts": TypeArray of identified conflicts
        "resolution": TypeObject explaining conflict resolution
    }
}
```

**Required fields:**

- `synthesis_steps[]`:
  - `step_name`: "review_inputs" | "count_consensus" | "calculate_confidence" | "identify_conflicts" | "assess_signal" | "define_parameters" | "make_decision"
  - `input_data`: structured data for this step
  - `observation`: what was observed/analyzed
  - `calculation`: if applicable (e.g., weighted confidence formula)
  - `conclusion`: what was concluded from this step
- `decision`:

  - `action`: "publish" | "skip"
  - `consensus_count`: integer (how many analysts agreed)
  - `weighted_confidence`: float 0-1
  - `conflicts_resolved`: boolean
  - `rationale`: 2-3 sentence summary

- `conflicts[]` (if any):
  - `conflicting_analysts`: array of analyst names
  - `conflict_type`: "direction" | "confidence" | "timing"
  - `resolution_method`: which tie-breaker was used

**Why this structure:**

- Each synthesis step is auditable
- Calculations are transparent (weighted confidence formula shown)
- Conflicts are explicitly documented with resolution logic
- Decision rationale directly references synthesis steps

### 1.2 Update OpportunitySynthesizer Prompt Template

**File:** `pkg/templates/prompts/agents/opportunity_synthesizer.tmpl`

**Changes required:**

1. Add explicit OUTPUT FORMAT section:

   ```markdown
   # OUTPUT FORMAT (REQUIRED)

   You MUST structure your synthesis as follows:

   {
   "synthesis_steps": [
   {
   "step_name": "review_inputs",
   "input_data": { /* analyst outputs */ },
   "observation": "Received 8 analyst inputs...",
   "conclusion": "Proceeding to consensus calculation"
   },
   {
   "step_name": "count_consensus",
   "input_data": { "buy": 5, "sell": 1, "neutral": 2 },
   "observation": "5/8 analysts signal BUY...",
   "calculation": null,
   "conclusion": "Strong bullish consensus"
   },
   {
   "step_name": "calculate_confidence",
   "input_data": { /* weighted inputs */ },
   "observation": "Applying weights...",
   "calculation": "0.78×0.30 + 0.65×0.20 + ...",
   "conclusion": "Weighted confidence = 65.4%"
   },
   // ... remaining steps
   ],
   "decision": {
   "action": "publish",
   "consensus_count": 5,
   "weighted_confidence": 0.654,
   "conflicts_resolved": true,
   "rationale": "Strong bullish consensus (5/8), confidence 65.4% meets threshold, macro risk noted but secondary"
   },
   "conflicts": [
   {
   "conflicting_analysts": ["macro_analyst"],
   "conflict_type": "direction",
   "resolution_method": "HTF technical consensus dominates single macro dissent"
   }
   ]
   }
   ```

2. Update REASONING APPROACH section to match structured output:

   - Map each reasoning step to output field
   - Show examples with actual structured output
   - Emphasize: "Each step MUST be documented in synthesis_steps array"

3. Update EXAMPLES section:
   - Replace prose examples with structured JSON examples
   - Show both "publish" and "skip" scenarios in structured format

### 1.3 Register OutputSchema in Factory

**File:** `internal/agents/factory.go`

**Update `getSchemaForAgent()`:**

```go
func getSchemaForAgent(agentType AgentType) (input, output *genai.Schema) {
    switch agentType {
    case AgentOpportunitySynthesizer:
        return nil, schemas.OpportunitySynthesizerOutputSchema
    // ... existing cases
    }
}
```

### 1.4 Create Reasoning Logger Callback

**File:** `internal/agents/callbacks/reasoning.go` (new file)

**Purpose:** Extract structured reasoning from agent output and save to database

**Create callback:**

```go
func SaveStructuredReasoningCallback(reasoningRepo ReasoningRepository) agent.AfterAgentCallback
```

**Responsibilities:**

- Parse agent output (JSON with synthesis_steps, decision, etc.)
- Extract reasoning trace
- Save to `reasoning_logs` table with:
  - `session_id`
  - `agent_type`
  - `reasoning_steps` (JSONB)
  - `final_decision` (JSONB)
  - `created_at`

**Integration:**

- Add to `buildAgentCallbacks()` in factory.go
- Register for all agents that use structured CoT

### 1.5 Add Reasoning Logs Table Migration

**File:** `migrations/postgres/XXX_add_reasoning_logs.up.sql` (new)

**Schema:**

```sql
CREATE TABLE reasoning_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    user_id UUID,
    agent_type TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    reasoning_steps JSONB NOT NULL,
    final_output JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reasoning_logs_session ON reasoning_logs(session_id);
CREATE INDEX idx_reasoning_logs_agent_type ON reasoning_logs(agent_type);
CREATE INDEX idx_reasoning_logs_created ON reasoning_logs(created_at DESC);
```

### 1.6 Extract Hardcoded Prompt from OpportunityFinder Worker

**Priority:** 🔥 CRITICAL  
**Problem:** `internal/workers/analysis/opportunity_finder.go` has hardcoded prompt in `buildResearchPrompt()` method

**File:** `internal/workers/analysis/opportunity_finder.go`

**Current issue:**

```go
func (of *OpportunityFinder) buildResearchPrompt(symbol string) string {
    return fmt.Sprintf(`Conduct comprehensive market research for %s on %s exchange.

**Your task:**
1. Analyze ALL dimensions (technical, SMC, sentiment, orderflow, derivatives, macro, onchain, correlation)
2. Synthesize findings into a coherent market view
3. Determine if there is a high-confidence (>65%%) trading opportunity
4. If YES → publish via publish_opportunity tool
5. If NO → explain why the signal is not strong enough
...`, symbol, of.exchange)
}
```

**Problems:**

- Prompt is not in template system (hard to maintain)
- Doesn't specify structured output requirements
- Inconsistent with agent-level prompts (analysts have detailed templates)
- Can't be versioned/tested independently

**Solution:**

#### 1.6.1 Create Workflow Input Template

**File:** `pkg/templates/prompts/workflows/market_research_input.tmpl` (new)

**Content:**

````markdown
# MARKET RESEARCH WORKFLOW EXECUTION

**Symbol:** {{.Symbol}}  
**Exchange:** {{.Exchange}}  
**Timestamp:** {{.Timestamp}}

---

## MISSION

Conduct comprehensive multi-dimensional market analysis to identify high-quality trading opportunities.

You are running a sophisticated workflow:

1. **8 Parallel Analysts** analyze different market dimensions simultaneously
2. **OpportunitySynthesizer** synthesizes findings and decides whether to publish

---

## ANALYSIS DIMENSIONS (8 Analysts)

Each analyst will independently analyze:

1. **MarketAnalyst:** Technical indicators, patterns, support/resistance
2. **SMCAnalyst:** Smart Money Concepts, liquidity, order blocks
3. **SentimentAnalyst:** News, social media, fear & greed
4. **OrderFlowAnalyst:** Order book, CVD, delta, tape
5. **DerivativesAnalyst:** Funding rates, open interest, options
6. **MacroAnalyst:** Economic events, correlations, risk regime
7. **OnChainAnalyst:** Whale movements, exchange flows, supply dynamics
8. **CorrelationAnalyst:** Cross-asset correlations, regime detection

---

## STRUCTURED OUTPUT REQUIREMENT

⚠️ **CRITICAL:** Each agent MUST return structured JSON with reasoning_trace and final_analysis.

**Expected format for each analyst:**

```json
{
  "reasoning_trace": [
    {
      "step_number": 1,
      "step_name": "assess_context",
      "observation": "...",
      "tool_used": "get_price",
      "conclusion": "..."
    }
    // ... more steps
  ],
  "final_analysis": {
    "direction": "bullish|bearish|neutral",
    "confidence": 0.75,
    "key_findings": ["...", "...", "..."],
    "rationale": "..."
  }
}
```
````

---

## SYNTHESIS REQUIREMENTS

After all analysts complete, **OpportunitySynthesizer** MUST:

1. **Review all 8 analyst outputs** (structured reasoning traces)
2. **Count consensus:** How many agree on direction?
3. **Calculate weighted confidence:** Apply analyst weights
4. **Identify conflicts:** Document any disagreements
5. **Resolve conflicts:** Apply tie-breaker rules
6. **Make decision:** Publish or skip?
7. **Output structured synthesis** (synthesis_steps + decision)

**Decision criteria:**

- ✅ Publish if: 5+ analysts agree, weighted confidence >65%, clear levels, R:R >2:1
- ❌ Skip if: <5 consensus, confidence <65%, conflicts unresolved, unclear levels

**Output format:**

```json
{
  "synthesis_steps": [
    {
      "step_name": "review_inputs",
      "input_data": {
        /* all 8 analysts */
      },
      "conclusion": "..."
    },
    {
      "step_name": "count_consensus",
      "input_data": { "buy": 5, "sell": 1, "neutral": 2 },
      "conclusion": "5/8 agree - strong consensus"
    },
    {
      "step_name": "calculate_confidence",
      "calculation": "0.78×0.30 + 0.65×0.20 + ... = 0.654",
      "conclusion": "Weighted confidence 65.4%"
    }
    // ... more steps
  ],
  "decision": {
    "action": "publish",
    "consensus_count": 5,
    "weighted_confidence": 0.654,
    "conflicts_resolved": true,
    "rationale": "..."
  }
}
```

---

## QUALITY PRINCIPLES

**Every analyst must:**

- ✅ Use tools iteratively (don't guess, gather data)
- ✅ Think step-by-step (document each reasoning step)
- ✅ Return structured output (JSON with reasoning_trace)
- ✅ Express confidence honestly (0-1 scale)

**OpportunitySynthesizer must:**

- ✅ Review ALL 8 analyst reasoning traces
- ✅ Calculate consensus and weighted confidence transparently
- ✅ Document conflicts and how they were resolved
- ✅ Only publish high-quality signals (5+ consensus, >65% confidence)

---

## EXECUTION

Begin comprehensive analysis. Each analyst should work independently and thoroughly.
OpportunitySynthesizer will synthesize after all analysts complete.

**Think comprehensively. Be rigorous. Only publish high-quality opportunities.**

````

#### 1.6.2 Update OpportunityFinder to Use Template

**File:** `internal/workers/analysis/opportunity_finder.go`

**Changes:**

```go
type OpportunityFinder struct {
    *workers.BaseWorker
    workflow       agent.Agent
    runner         *runner.Runner
    sessionService session.Service
    templates      *templates.Registry  // ADD: Template registry
    symbols        []string
    exchange       string
    log            *logger.Logger
}

func NewOpportunityFinder(
    workflowFactory *workflows.Factory,
    sessionService session.Service,
    templates *templates.Registry,  // ADD: Inject templates
    symbols []string,
    exchange string,
    interval time.Duration,
    enabled bool,
) (*OpportunityFinder, error) {
    // ... existing code ...

    return &OpportunityFinder{
        BaseWorker:     workers.NewBaseWorker("opportunity_finder", interval, enabled),
        workflow:       workflow,
        runner:         runnerInstance,
        sessionService: sessionService,
        templates:      templates,  // ADD
        symbols:        symbols,
        exchange:       exchange,
        log:            log,
    }, nil
}

// REPLACE buildResearchPrompt
func (of *OpportunityFinder) buildResearchPrompt(symbol string) string {
    // Use template system instead of hardcoded string
    prompt, err := of.templates.Render("workflows/market_research_input", map[string]interface{}{
        "Symbol":    symbol,
        "Exchange":  of.exchange,
        "Timestamp": time.Now().Format(time.RFC3339),
    })

    if err != nil {
        of.log.Errorf("Failed to render workflow input template: %v", err)
        // Fallback to basic prompt
        return fmt.Sprintf("Analyze %s on %s exchange for trading opportunities.", symbol, of.exchange)
    }

    return prompt
}
````

#### 1.6.3 Update Worker Registration in main.go

**File:** `cmd/main.go` (lines ~918-929)

**Current code:**

```go
opportunityFinder, err := analysis.NewOpportunityFinder(
    workflowFactory,
    adkSessionService,
    defaultSymbols,
    "binance", // Primary exchange
    cfg.Workers.OpportunityFinderInterval,
    true, // enabled
)
```

**Updated code:**

```go
opportunityFinder, err := analysis.NewOpportunityFinder(
    workflowFactory,
    adkSessionService,
    templates.Get(),  // ADD: Pass template registry
    defaultSymbols,
    "binance", // Primary exchange
    cfg.Workers.OpportunityFinderInterval,
    true, // enabled
)
```

**Note:** `templates.Get()` is already imported and used elsewhere in main.go (line 64), so no additional imports needed.

**Benefits of this change:**

- ✅ Centralized prompt management (all prompts in templates/)
- ✅ Consistent with other agent prompts
- ✅ Can version and test prompts independently
- ✅ Explicitly requires structured output from all agents
- ✅ Clear documentation of workflow expectations
- ✅ Easier to update when implementing structured CoT

---

### 1.7 Testing & Validation

**Create test:** `internal/agents/workflows/opportunity_synthesizer_test.go`

**Test scenarios:**

1. All 8 analysts agree (BUY) → verify structured output contains all synthesis steps
2. Split decision (5 BUY, 3 SELL) → verify conflict resolution documented
3. Below confidence threshold → verify "skip" decision with rationale
4. Macro conflict → verify resolution method documented

**Validation criteria:**

- OutputSchema enforced by ADK (invalid output causes error)
- All required fields present
- Synthesis steps in logical order
- Decision rationale references specific steps
- Reasoning saved to database correctly

**Additional validation for OpportunityFinder:**

- Workflow input prompt renders correctly from template
- Template variables (Symbol, Exchange, Timestamp) injected properly
- Fallback prompt works if template fails

---

## Phase 2: High Priority — All Analyst Agents Structured CoT

**Priority:** 🔴 HIGH  
**Timeline:** Week 2  
**Depends on:** Phase 1 complete

### 2.1 Define Analyst OutputSchemas

**File:** `internal/agents/schemas/analysts.go` (new)

**Create schemas for each analyst type:**

#### Template Structure (apply to all 8 analysts):

```go
MarketAnalystOutputSchema = &genai.Schema{
    Type: genai.TypeObject,
    Properties: {
        "reasoning_trace": TypeArray of reasoning steps
        "final_analysis": TypeObject with direction, confidence, levels, rationale
        "tool_calls_summary": TypeArray of tools used and results
    }
}
```

**Reasoning trace fields:**

- `step_number`: integer (1, 2, 3, ...)
- `step_name`: string from prompt (e.g., "assess_context", "identify_gaps", "gather_data", "build_hypothesis", "validate", "conclude")
- `observation`: what was observed at this step
- `tool_used`: string | null (tool name if called)
- `tool_result_summary`: string | null (brief summary of tool output)
- `conclusion`: what was concluded, what's next

**Final analysis fields (varies by analyst):**

- `direction`: "bullish" | "bearish" | "neutral"
- `confidence`: float 0-1
- `key_findings`: array of strings (3-5 key points)
- `supporting_factors`: array of strings
- `risk_factors`: array of strings
- `rationale`: 2-3 sentence summary

**Schemas to create:**

1. `MarketAnalystOutputSchema`
2. `SMCAnalystOutputSchema`
3. `SentimentAnalystOutputSchema`
4. `OrderFlowAnalystOutputSchema`
5. `DerivativesAnalystOutputSchema`
6. `MacroAnalystOutputSchema`
7. `OnChainAnalystOutputSchema`
8. `CorrelationAnalystOutputSchema`

**Customizations per analyst:**

- MarketAnalyst: add `key_levels` array (support/resistance)
- SMCAnalyst: add `market_structure` (HH/HL/LH/LL), `fvg_zones` array
- SentimentAnalyst: add `sentiment_score` (-1 to 1), `fear_greed_index`
- OrderFlowAnalyst: add `flow_bias` (bullish/bearish/neutral), `cvd_trend`
- DerivativesAnalyst: add `funding_rate`, `oi_change`, `options_flow`
- MacroAnalyst: add `regime` (risk-on/risk-off), `catalysts` array
- OnChainAnalyst: add `whale_activity`, `exchange_flows`
- CorrelationAnalyst: add `correlation_regime`, `btc_correlation`

### 2.2 Update All Analyst Prompt Templates

**Files:** `pkg/templates/prompts/agents/*.tmpl` (8 files)

**For each analyst prompt:**

1. **Replace "HOW TO COMPLETE YOUR ANALYSIS" section** with structured output requirements

2. **Add OUTPUT FORMAT section** (similar to OpportunitySynthesizer):

   ```markdown
   # OUTPUT FORMAT (REQUIRED)

   You MUST structure your analysis as follows:

   {
   "reasoning_trace": [
   {
   "step_number": 1,
   "step_name": "assess_context",
   "observation": "I have symbol BTC/USDT and timeframe 4h",
   "tool_used": null,
   "tool_result_summary": null,
   "conclusion": "Need price data to start analysis"
   },
   {
   "step_number": 2,
   "step_name": "gather_price_data",
   "observation": "Current price $42,000",
   "tool_used": "get_price",
   "tool_result_summary": "BTC/USDT: $42,000, 24h change +2.3%",
   "conclusion": "Price stable, need HTF trend context"
   },
   // ... continue for all reasoning steps
   ],
   "final_analysis": {
   "direction": "bullish",
   "confidence": 0.75,
   "key_findings": [
   "HTF uptrend intact (higher highs since October)",
   "Support test at $40k zone (3 touches)",
   "RSI recovery from oversold"
   ],
   "supporting_factors": ["HTF structure", "tested support", "momentum recovery"],
   "risk_factors": ["Macro uncertainty", "Resistance at $44k"],
   "rationale": "Strong HTF uptrend with LTF pullback to tested support. Confluence at $40k. Risk:reward favorable for long entry."
   },
   "tool_calls_summary": [
   {"tool": "get_price", "result": "success"},
   {"tool": "get_ohlcv", "result": "success"},
   {"tool": "calculate_rsi", "result": "RSI=45"}
   ]
   }
   ```

3. **Update REASONING APPROACH section:**

   - Map each step (1-7 from prompt) to reasoning_trace entry
   - Show that each step must be documented
   - Emphasize: "Think aloud IN YOUR OUTPUT, not just in your head"

4. **Update EXAMPLES:**

   - Show structured reasoning_trace examples
   - Demonstrate good vs bad reasoning structure

5. **Remove `save_analysis` tool instructions** (replaced by structured output)

### 2.3 Register All Analyst Schemas

**File:** `internal/agents/factory.go`

**Update `getSchemaForAgent()`:**

```go
case AgentMarketAnalyst:
    return nil, schemas.MarketAnalystOutputSchema
case AgentSMCAnalyst:
    return nil, schemas.SMCAnalystOutputSchema
// ... etc for all 8 analysts
```

### 2.4 Update Reasoning Callback for Analysts

**File:** `internal/agents/callbacks/reasoning.go`

**Extend `SaveStructuredReasoningCallback()`:**

- Handle analyst-specific output formats
- Extract reasoning_trace and final_analysis
- Save to reasoning_logs table with proper agent_type

### 2.5 Testing

**Create tests:** `internal/agents/workflows/analysts_cot_test.go`

**For each analyst:**

- Mock tool responses
- Run agent
- Verify structured output contains:
  - reasoning_trace with 3+ steps
  - Each step has observation, conclusion
  - Tools used are documented
  - final_analysis has all required fields
  - Output conforms to schema

---

## Phase 3: Medium Priority — Strategy/Risk/Executor Structured CoT

**Priority:** 🟡 MEDIUM  
**Timeline:** Week 3  
**Depends on:** Phase 2 complete

### 3.1 Define Schemas for Trading Pipeline Agents

**File:** `internal/agents/schemas/trading_pipeline.go` (new)

**Create schemas:**

#### StrategyPlannerOutputSchema

```go
{
    "synthesis_process": [
        {
            "step": "review_analyst_inputs",
            "analyst_summaries": { /* all 8 analysts */ },
            "conclusion": "..."
        },
        {
            "step": "count_consensus",
            "consensus_data": { "aligned": 5, "conflicting": 1, "neutral": 2 },
            "conclusion": "..."
        },
        {
            "step": "resolve_conflicts",
            "conflicts": [ /* details */ ],
            "resolution": "..."
        },
        {
            "step": "identify_edge",
            "edge_factors": [ /* ... */ ],
            "conclusion": "..."
        },
        {
            "step": "design_strategy",
            "strategy_components": { /* entry, sizing, stops, targets */ },
            "conclusion": "..."
        },
        {
            "step": "stress_test",
            "scenarios": [ /* base, bear, shock */ ],
            "conclusion": "..."
        }
    ],
    "trade_plan": {
        "recommendation": "enter_long" | "enter_short" | "hold_cash" | "adjust",
        "confidence": 0-1,
        "direction": "long" | "short",
        "entry": { "zone": [min, max], "trigger": "..." },
        "sizing": { "position_pct": 6, "rationale": "..." },
        "stop_loss": { "price": 39000, "rationale": "..." },
        "targets": [ { "price": 41000, "scale_out_pct": 50 } ],
        "timeframe": "3-7 days",
        "monitoring": { /* validation, invalidation criteria */ },
        "contingencies": [ /* what-if plans */ ]
    }
}
```

#### RiskManagerOutputSchema

```go
{
    "risk_evaluation_steps": [
        {
            "step": "parse_proposal",
            "proposal_summary": { /* ... */ },
            "conclusion": "..."
        },
        {
            "step": "check_hard_limits",
            "limits_checked": { /* ... */ },
            "result": "pass" | "fail",
            "conclusion": "..."
        },
        {
            "step": "analyze_concentration",
            "concentration": { /* by asset, exchange, correlation */ },
            "status": "acceptable" | "concerning" | "breach",
            "conclusion": "..."
        },
        {
            "step": "stress_test",
            "scenarios": { /* -10%, -20%, -30% */ },
            "max_drawdown_risk": "...",
            "conclusion": "..."
        },
        {
            "step": "liquidity_assessment",
            "liquidity_data": { /* ... */ },
            "conclusion": "..."
        },
        {
            "step": "form_decision",
            "decision_rationale": "...",
            "conclusion": "..."
        }
    ],
    "risk_decision": {
        "decision": "approved" | "approved_with_conditions" | "rejected",
        "confidence": 0-1,
        "hard_limits_check": "passed" | "failed",
        "concentration_status": "acceptable" | "concerning" | "breach",
        "conditions": [ /* if approved_with_conditions */ ],
        "breaches": [ /* if rejected */ ],
        "kill_switch_triggers": [ /* ... */ ],
        "monitoring_plan": [ /* ... */ ],
        "rationale": "..."
    }
}
```

#### ExecutorOutputSchema

```go
{
    "execution_planning_steps": [
        {
            "step": "parse_objective",
            "objective": { /* buy/sell, size, urgency */ },
            "conclusion": "..."
        },
        {
            "step": "venue_analysis",
            "venues_analyzed": [ /* binance, bybit, etc */ ],
            "liquidity_data": { /* ... */ },
            "conclusion": "..."
        },
        {
            "step": "select_venue",
            "chosen_venue": "binance",
            "rationale": "...",
            "conclusion": "..."
        },
        {
            "step": "choose_order_type",
            "order_type": "twap_limit_post_only",
            "rationale": "...",
            "conclusion": "..."
        },
        {
            "step": "plan_slicing",
            "slicing_strategy": { /* ... */ },
            "conclusion": "..."
        },
        {
            "step": "define_failsafes",
            "failsafes": { /* ... */ },
            "conclusion": "..."
        }
    ],
    "execution_plan": {
        "objective": "buy 10 BTC over 60 mins",
        "venue": "binance",
        "order_strategy": {
            "type": "twap_limit_post_only",
            "slicing": "60 orders × 0.167 BTC",
            "interval_seconds": 60,
            "price_logic": "best_bid, step_up_after_2min"
        },
        "risk_controls": {
            "max_slippage_bps": 5,
            "order_expiry_seconds": 120,
            "kill_switch_conditions": [ /* ... */ ]
        },
        "contingencies": [ /* ... */ ],
        "estimated_completion_minutes": 60
    }
}
```

### 3.2 Update Prompt Templates

**Files to update:**

- `pkg/templates/prompts/agents/strategy_planner.tmpl`
- `pkg/templates/prompts/agents/risk_manager.tmpl`
- `pkg/templates/prompts/agents/executor.tmpl`

**For each:**

1. Add OUTPUT FORMAT section with structured schema
2. Map reasoning steps to output structure
3. Update examples to show structured output
4. Remove `save_analysis` tool instructions

### 3.3 Register Schemas & Test

Similar to Phase 2 process.

---

## Phase 4: State-Based Reasoning Context Passing

**Priority:** 🟡 MEDIUM  
**Timeline:** Week 3 (parallel with Phase 3)  
**Depends on:** Phase 2 complete

### 4.1 Design State Schema for Reasoning Context

**File:** `internal/agents/state/reasoning.go` (new)

**Purpose:** Store and retrieve reasoning traces in ADK state for cross-agent context

**Functions to create:**

```go
// SetAgentReasoning stores reasoning trace in state
func SetAgentReasoning(state session.State, agentType string, reasoning ReasoningTrace)

// GetAgentReasoning retrieves reasoning trace from state
func GetAgentReasoning(state session.ReadonlyState, agentType string) (ReasoningTrace, bool)

// GetAllReasoningContext retrieves all agent reasoning for workflow
func GetAllReasoningContext(state session.ReadonlyState) map[string]ReasoningTrace
```

**Types:**

```go
type ReasoningTrace struct {
    AgentType   string
    Steps       []ReasoningStep
    Conclusion  interface{} // varies by agent
    Confidence  float64
    Timestamp   time.Time
}

type ReasoningStep struct {
    StepNumber  int
    StepName    string
    Observation string
    Conclusion  string
    ToolUsed    *string
}
```

### 4.2 Create Reasoning State Callback

**File:** `internal/agents/callbacks/reasoning.go`

**Add new callback:**

```go
func SaveReasoningToStateCallback() agent.AfterAgentCallback {
    return func(ctx agent.CallbackContext) (*genai.Content, error) {
        // Extract agent output (reasoning trace)
        // Parse structured output
        // Save to state using state.SetAgentReasoning()
        // Return nil (non-blocking)
    }
}
```

**Integration:**

- Add to `buildAgentCallbacks()` after SaveStructuredReasoningCallback
- Only for agents in workflows (analysts, strategy, risk, executor)

### 4.3 Update Workflow Factory to Inject Reasoning Context

**File:** `internal/agents/workflows/factory.go`

**Problem:** Sequential workflows need to pass reasoning from previous agents to next agent

**Solution:** Use InstructionProvider to inject reasoning context dynamically

**For each workflow:**

```go
// Example: PersonalTradingWorkflow
func (f *Factory) CreatePersonalTradingWorkflow() (agent.Agent, error) {
    // Step 1: Strategy Planner with InstructionProvider
    strategyAgent := createAgentWithReasoningContext(
        AgentStrategyPlanner,
        []string{"market_analyst", "smc_analyst", /* ... all 8 */}
    )

    // Step 2: Risk Manager with strategy reasoning context
    riskAgent := createAgentWithReasoningContext(
        AgentRiskManager,
        []string{"strategy_planner"}
    )

    // Step 3: Executor with risk + strategy context
    executorAgent := createAgentWithReasoningContext(
        AgentExecutor,
        []string{"strategy_planner", "risk_manager"}
    )

    // Create workflow
    workflow := sequentialagent.New(...)
}
```

### 4.4 Implement InstructionProvider for Context Injection

**File:** `internal/agents/factory.go`

**Add method:**

```go
func (f *Factory) createAgentWithReasoningContext(
    agentType AgentType,
    previousAgentTypes []string,
) (agent.Agent, error) {
    cfg := DefaultAgentConfigs[agentType]

    // Build InstructionProvider that injects reasoning context
    cfg.InstructionProvider = func(ctx agent.CallbackContext) (string, error) {
        // 1. Render base template
        basePrompt := f.templates.Render(cfg.SystemPromptTemplate, ...)

        // 2. Retrieve reasoning from previous agents
        reasoningContext := buildReasoningContext(ctx.ReadonlyState(), previousAgentTypes)

        // 3. Append to prompt
        contextSection := formatReasoningContext(reasoningContext)

        return basePrompt + "\n\n" + contextSection, nil
    }

    return f.CreateAgent(cfg)
}

func buildReasoningContext(state session.ReadonlyState, agentTypes []string) string {
    var context strings.Builder
    context.WriteString("# PREVIOUS AGENT REASONING CONTEXT\n\n")

    for _, agentType := range agentTypes {
        reasoning, ok := state.GetAgentReasoning(agentType)
        if !ok {
            continue
        }

        context.WriteString(fmt.Sprintf("## %s Analysis:\n", agentType))
        context.WriteString(fmt.Sprintf("**Conclusion:** %v\n", reasoning.Conclusion))
        context.WriteString(fmt.Sprintf("**Confidence:** %.2f\n\n", reasoning.Confidence))

        context.WriteString("**Reasoning Steps:**\n")
        for _, step := range reasoning.Steps {
            context.WriteString(fmt.Sprintf("- **%s:** %s → %s\n",
                step.StepName, step.Observation, step.Conclusion))
        }
        context.WriteString("\n")
    }

    return context.String()
}
```

### 4.5 Update Prompt Templates to Reference Context

**Files:** All agent prompts that receive context

**Add section at top:**

```markdown
{{if .PreviousAgentReasoning}}

# PREVIOUS AGENT REASONING CONTEXT

You have access to reasoning from previous agents in the workflow.
Use this context to inform your decisions, but form your own independent analysis.

{{.PreviousAgentReasoning}}
{{end}}
```

### 4.6 Testing

**Test:** Sequential workflow passes reasoning correctly

**Scenarios:**

1. MarketResearchWorkflow: analysts → synthesizer
   - Verify synthesizer receives all 8 analyst reasoning traces
2. PersonalTradingWorkflow: strategy → risk → executor
   - Verify risk manager sees strategy reasoning
   - Verify executor sees both strategy + risk reasoning
3. Reasoning context is summarized (not full dump)
   - Each agent gets relevant info, not overwhelming detail

---

## Phase 5: Optional Enhancements

**Priority:** ✅ NICE TO HAVE  
**Timeline:** Future iterations

### 5.1 Self-Correction Mechanism

**Idea:** Agent can revisit previous reasoning steps if new information contradicts

**Implementation:**

- Add `revision_history` field to reasoning trace
- Allow agent to call `revise_step(step_number, new_conclusion)` tool
- Log all revisions for audit

### 5.2 Reasoning Quality Metrics

**Idea:** Automated evaluation of reasoning quality

**Metrics:**

- **Completeness:** All required steps present?
- **Coherence:** Conclusions logically follow observations?
- **Tool usage:** Appropriate tools called?
- **Confidence calibration:** Stated confidence matches actual outcomes?

**Implementation:**

- Create `internal/evaluation/reasoning_quality.go`
- Run post-analysis evaluation
- Store quality scores in reasoning_logs table
- Alert on low-quality reasoning

### 5.3 Reasoning Replay & Debugging UI

**Idea:** Web UI to visualize agent reasoning step-by-step

**Features:**

- Timeline view of reasoning steps
- Tool calls with input/output
- Confidence evolution over time
- Highlight conflicts and resolutions
- Compare reasoning across similar scenarios

**Tech stack:**

- Backend: Add API endpoints to query reasoning_logs
- Frontend: React visualization component
- Integration with existing Prometheus UI

### 5.4 Multi-Turn Reasoning for Complex Decisions

**Idea:** Allow agents to iterate on decisions with human-in-the-loop

**Use case:** OpportunitySynthesizer uncertain → ask for clarification

**Implementation:**

- Add `requires_clarification` flag to decision output
- Pause workflow, request input
- Continue with additional context
- Log multi-turn reasoning separately

---

## Implementation Order & Dependencies

```
Phase 1: OpportunitySynthesizer CoT (Week 1)
├─ 1.1 Define OutputSchema
├─ 1.2 Update prompt template
├─ 1.3 Register schema in factory
├─ 1.4 Create reasoning logger callback
├─ 1.5 Add database migration
├─ 1.6 Extract hardcoded prompt from OpportunityFinder
│  ├─ 1.6.1 Create workflow input template
│  ├─ 1.6.2 Update OpportunityFinder to use template
│  └─ 1.6.3 Update worker registration
└─ 1.7 Testing & validation

Phase 2: Analyst Agents CoT (Week 2)
├─ Depends on: Phase 1 complete
├─ 2.1 Define 8 analyst OutputSchemas
├─ 2.2 Update 8 analyst prompts
├─ 2.3 Register schemas
├─ 2.4 Extend reasoning callback
└─ 2.5 Testing

Phase 3: Trading Pipeline CoT (Week 3)
├─ Depends on: Phase 2 complete
├─ 3.1 Define Strategy/Risk/Executor schemas
├─ 3.2 Update prompts
└─ 3.3 Testing

Phase 4: State-Based Context Passing (Week 3, parallel)
├─ Depends on: Phase 2 complete
├─ 4.1 Design state schema
├─ 4.2 Create state callback
├─ 4.3 Update workflow factory
├─ 4.4 Implement InstructionProvider
├─ 4.5 Update prompt templates
└─ 4.6 Testing

Phase 5: Optional Enhancements (Future)
├─ Depends on: Phases 1-4 complete
├─ 5.1 Self-correction
├─ 5.2 Quality metrics
├─ 5.3 Debugging UI
└─ 5.4 Multi-turn reasoning
```

---

## Success Criteria

### Phase 1 Success:

- ✅ OpportunitySynthesizer returns structured JSON conforming to schema
- ✅ All synthesis steps logged with observations and conclusions
- ✅ Conflicts explicitly documented with resolution method
- ✅ Decision rationale references specific reasoning steps
- ✅ Reasoning saved to database and queryable
- ✅ Can replay any opportunity decision and understand why it was made

### Phase 2 Success:

- ✅ All 8 analysts return structured reasoning traces
- ✅ Each reasoning step documents observation → tool use → conclusion
- ✅ Final analysis contains direction, confidence, key findings
- ✅ Can debug any analyst output by reviewing step-by-step reasoning
- ✅ Tool calls are documented and traceable

### Phase 3 Success:

- ✅ Strategy, Risk, Executor agents use structured CoT
- ✅ Trading decisions fully auditable end-to-end
- ✅ Can trace any trade from market signal → strategy → risk approval → execution plan

### Phase 4 Success:

- ✅ Sequential workflows pass reasoning context via state
- ✅ Strategy planner sees all 8 analyst reasoning summaries
- ✅ Risk manager sees strategy reasoning when evaluating
- ✅ Executor sees strategy + risk context when planning execution
- ✅ InstructionProvider dynamically builds prompts with relevant context

---

## Risks & Mitigations

### Risk 1: LLM struggles with strict JSON output

**Mitigation:**

- Use ADK OutputSchema validation (fails fast if invalid)
- Provide clear examples in prompts
- Use o1/o3 models or Gemini with structured output support
- Fallback: Parse free-form output and structure it post-hoc

### Risk 2: Verbose output increases token costs

**Mitigation:**

- Limit reasoning steps to essential ones (3-7 steps typical)
- Use summarization for context passing (don't dump full traces)
- Monitor token usage via existing callbacks
- Adjust verbosity based on cost/quality tradeoff

### Risk 3: State size grows large with reasoning traces

**Mitigation:**

- Store only essential info in state (summaries, not full traces)
- Full traces go to database, state gets lightweight version
- Clean up temp state after workflow completes
- Set state TTL/size limits

### Risk 4: Schema changes break existing agents

**Mitigation:**

- Phase 1 first (OpportunitySynthesizer) validates approach
- Gradual rollout (1 agent type at a time)
- Backward compatibility: optional OutputSchema at first
- Extensive testing before production deployment

---

## Metrics to Track

### During Implementation:

- Number of agents with structured CoT: 0 → 15 (goal)
- Schema validation pass rate (should be >95%)
- Reasoning logs saved per day
- Average reasoning trace length (steps per agent)

### Post-Implementation:

- **Audit success rate:** Can we explain 100% of decisions?
- **Debug time reduction:** Time to diagnose bad decisions (should decrease)
- **Decision quality:** Does structured reasoning correlate with better outcomes?
- **Token usage:** Cost increase from verbose reasoning (acceptable range?)

### Production Monitoring:

- Reasoning logs ingestion rate
- Schema validation errors
- State size per session
- Context injection latency

---

## Open Questions

1. **Reasoning granularity:** How many steps is "enough"? 3? 7? 10?

   - Recommendation: Start with 3-7, adjust based on usefulness

2. **Context window limits:** Passing 8 analyst traces to synthesizer → large prompt

   - Solution: Summarize reasoning (key points only, not full steps)

3. **Real-time vs batch:** Save reasoning synchronously or async?

   - Recommendation: Async logging (don't block agent execution)

4. **Retention policy:** How long to keep reasoning logs?

   - Recommendation: 90 days full detail, summarize/archive older data

5. **Human review:** Should high-stakes decisions require human approval?

   - Recommendation: Phase 5 enhancement (multi-turn reasoning)

6. **Prompt location policy:** Should ALL prompts be in templates, or are some OK in code?
   - **Found:** `OpportunityFinder.buildResearchPrompt()` has hardcoded prompt
   - **Recommendation:** ALL prompts in templates for consistency, versioning, testing
   - **Action:** Phase 1.6 extracts this to `workflows/market_research_input.tmpl`
   - **Policy:** No hardcoded prompts in Go code (exception: fallback error messages only)

---

## Related Documents

- `DEVELOPMENT_PLAN.md` — Overall project roadmap
- `ERROR_HANDLING.md` — Error handling patterns
- `AGENTS.md` — Agent design guidelines
- ADK Go docs: https://google.github.io/adk-docs/
- ADK source code adk-go - in the root dir

---

## Change Log

| Date       | Author       | Changes                                  |
| ---------- | ------------ | ---------------------------------------- |
| 2025-11-27 | AI Assistant | Initial draft based on codebase analysis |
