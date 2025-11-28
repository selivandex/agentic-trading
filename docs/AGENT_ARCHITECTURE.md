<!-- @format -->

# Agent Architecture — AI Hedge Fund Design

**Version:** 2.0  
**Date:** November 2025  
**Last Updated:** November 28, 2025  
**Status:** ✅ Phase 1-2 Complete | ⏳ Phase 3-4 In Progress | ❌ Phase 5-6 Not Started

---

## Executive Summary

Prometheus is an **AI-powered hedge fund** where specialized AI agents replace the traditional human team structure. Users connect their exchange accounts and allocate capital (e.g., `/invest 1000`), and the system manages their portfolio like a professional fund.

This document defines the production-ready agent architecture based on hedge fund organizational patterns, event-driven design, and multi-agent collaboration principles.

### Current Implementation Status (Nov 28, 2025)

| Phase | Status | Completion | Notes |
|-------|--------|------------|-------|
| **Phase 1: Core Refactoring** | ✅ COMPLETE | 100% | All agents defined, prompts ready, schemas updated |
| **Phase 2: Event-Driven Architecture** | ✅ COMPLETE | 95% | Event system operational, WebSocket deferred |
| **Phase 3: Research Committee** | ⏳ IN PROGRESS | 60% | Agents/prompts ready, orchestration pending |
| **Phase 4: ML Regime Detection** | ⏳ IN PROGRESS | 70% | Infrastructure ready, model not trained |
| **Phase 5: Information Processing** | ❌ NOT STARTED | 20% | Data collectors exist, core layer missing |
| **Phase 6: Memory System** | ⏳ PARTIAL | 30% | Basic persistence exists, rich memory missing |
| **Phase 7: Documentation & Testing** | ⏳ IN PROGRESS | 50% | Docs exist, test coverage minimal |

**Overall Progress: ~60% complete**

### What's Working Right Now

✅ **Event-Driven Position Monitoring** (Phase 2)
- Real-time position event generation (7 event types)
- Urgency-based routing (CRITICAL → algorithmic, HIGH/MEDIUM → LLM agent)
- Position guardian agent handling complex decisions
- Critical events processed <1 second, complex events <30 seconds

✅ **Agent Infrastructure** (Phase 1)
- 12 agent types defined and registered
- 14 prompt templates ready
- Tool registry with 15+ tools
- Structured output schemas with reasoning traces

✅ **Data Collection** (Phase 4 foundation)
- OHLCV, funding rates, liquidations, orderbook data
- On-chain data collectors (whales, flows, network metrics)
- ClickHouse time-series storage
- Worker health monitoring

### Critical Gaps

❌ **Multi-Agent Research Committee** (Phase 3)
- Agents and prompts exist, but workflow orchestration not implemented
- Cannot run parallel analyst execution + synthesis yet

❌ **Information Processing Layer** (Phase 5)
- No ContextProvider interface
- No InformationClassifier, SignalExtractor, or ContextBuilder
- External data collected but not intelligently processed

❌ **Trained ML Model** (Phase 4)
- Regime detection infrastructure ready
- Python training scripts ready
- BUT: No historical data labeled, no model trained

❌ **Rich Memory System** (Phase 6)
- Basic trade persistence exists
- Missing: TradeEpisode context, ValidatedPattern storage, Working Memory

### Key Design Principles

1. **Hedge Fund Metaphor**: Agents mirror real fund roles (Research Team, Portfolio Managers, Trading Desk, Risk Committee)
2. **Event-Driven Architecture**: Real-time reaction to market events, not polling-based monitoring ✅ IMPLEMENTED
3. **Multi-Agent Collaboration**: Research decisions involve debate and consensus ⏳ INFRASTRUCTURE READY
4. **Separation of Concerns**: Clear boundaries between analysis, decision-making, execution, and monitoring ✅ IMPLEMENTED
5. **Explainable Reasoning**: All decisions include reasoning traces for audit and learning ✅ IMPLEMENTED

---

## 1. Hedge Fund Organizational Structure

Real hedge funds organize into specialized teams. Our agent architecture mirrors this:

```
┌─────────────────────────────────────────────────────────────┐
│                     FUND LEADERSHIP                          │
│  • Fund Director (Supervisor Agent)                          │
│  • Investment Committee                                      │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   RESEARCH   │    │ PORTFOLIO    │    │  OPERATIONS  │
│     TEAM     │    │ MANAGEMENT   │    │     TEAM     │
└──────────────┘    └──────────────┘    └──────────────┘
│                   │                   │
│ Analysts          │ Portfolio Mgrs    │ Risk Committee
│ Market Intel      │ Trading Desk      │ Performance
│ Quant Team        │ Position Monitors │ Compliance
└───────────────────┴───────────────────┴────────────────┘
```

### Team Responsibilities

**Research Team (Global):**

- Continuous market monitoring across all assets
- Identify trading opportunities
- Provide market intelligence and regime analysis
- **Output**: Trading ideas published to Investment Committee

**Portfolio Management (Per-Client):**

- Manage individual client portfolios
- Adapt global trading ideas to client constraints
- Make personalized entry/exit decisions
- **Output**: Client-specific trade approvals

**Operations Team (Cross-Cutting):**

- Risk monitoring and circuit breakers
- Performance analysis and learning
- Compliance and audit trails
- **Output**: System health and insights

---

## 2. Architecture Tiers

### Tier 1: Fund-Level Agents (Global, Singleton)

These agents operate at the fund level, analyzing markets and generating trading ideas for ALL clients.

#### 2.1. Research Committee (Multi-Agent Team)

**Purpose**: Collaborative market analysis with diverse viewpoints and debate-driven consensus.

**Team Composition:**

```go
type ResearchCommittee struct {
    // Specialist Analysts (parallel execution)
    TechnicalAnalyst   agent.Agent  // Technical indicators, chart patterns
    StructuralAnalyst  agent.Agent  // Smart Money Concepts (SMC/ICT)
    FlowAnalyst        agent.Agent  // Order flow, whale activity, derivatives
    MacroAnalyst       agent.Agent  // Economic context, correlations

    // Synthesis & Decision
    HeadOfResearch     agent.Agent  // Synthesizes all inputs, leads debate

    // Workflow orchestration
    workflow           agent.Agent  // ADK SequentialAgent or custom orchestrator
}
```

**Workflow:**

```
Step 1: PARALLEL ANALYSIS (4 agents, 30-45 sec)
├─ TechnicalAnalyst  → Technical analysis + signal strength
├─ StructuralAnalyst → SMC analysis + key levels
├─ FlowAnalyst       → Order flow + whale activity
└─ MacroAnalyst      → Economic context + risk factors

Step 2: SYNTHESIS & DEBATE (1 agent, 30-60 sec)
└─ HeadOfResearch:
   - Reviews all analyst reports
   - Identifies conflicts and consensus
   - Challenges weak reasoning
   - Conducts "pre-mortem" (what could go wrong?)
   - Makes final decision: PUBLISH or SKIP

Step 3: OUTPUT
└─ If PUBLISH:
   - Publishes to Kafka: "opportunity_identified"
   - Includes: all analyst reports + synthesis + reasoning trace
   - Metadata: confidence, timeframe, key levels
```

**Agent Prompts:**

Each analyst has a specialized prompt:

- **TechnicalAnalyst**: Focus on momentum, trend, volatility indicators
- **StructuralAnalyst**: Focus on market structure, order blocks, liquidity
- **FlowAnalyst**: Focus on real-time data, whale behavior, positioning
- **MacroAnalyst**: Focus on broader context, risk events, correlations

**HeadOfResearch** prompt includes:

- Synthesis instructions
- Conflict resolution framework
- Confidence calibration rules
- Decision criteria (publish vs skip)

**Cost & Performance:**

- Total time: 60-90 seconds (parallel + synthesis)
- Cost per scan: ~$0.30-0.50 (4 agents + synthesizer)
- Quality: High (diverse viewpoints, debate-driven)

**Fast-Path Alternative:**

For routine scanning, use **OpportunitySynthesizer** (single agent with tool orchestration):

- Faster: 15-30 seconds
- Cheaper: $0.05-0.10
- Use case: High-frequency scanning, low-stakes signals

**Selection Logic:**

```go
func (f *Fund) SelectResearchPath(symbol string, priority Priority) agent.Agent {
    if priority == High || f.isHighStakes(symbol) {
        // Use multi-agent research committee
        return f.researchCommittee
    }
    // Use fast-path single agent
    return f.opportunitySynthesizer
}
```

#### 2.2. Market Intelligence (Algorithmic + LLM)

**RegimeDetector**: Classifies market regime and recommends parameter adjustments.

**Architecture (Two-Tier Approach):**

**Phase 4: Market Data Based (Core)**

```
1. Feature Extraction (algorithmic) - Market Data Only
   Phase 4 features (available from exchanges):
   - Volatility: ATR, Bollinger bandwidth, historical vol
   - Trend: EMA slopes, ADX, higher highs/lows pattern
   - Volume: volume trend, volume/price divergence
   - Structure: support/resistance breaks, consolidation periods
   - Cross-asset: BTC dominance, correlation tightness
   - Derivatives: funding rates, liquidations (from exchange API)

   Phase 5+ enhancement (with external data):
   - Sentiment: news sentiment, social sentiment, Fear & Greed Index
   - On-chain: whale activity, exchange flows, MVRV ratio

2. Regime Classification (ML Model)
   - Model: Random Forest / XGBoost (not LSTM initially - simpler)
   - Input: feature vector (20-30 features in Phase 4, 40+ in Phase 5+)
   - Training: Historical data with labeled regimes (manual labeling initially)
   - Output: regime probabilities
     * Bull Trending: 0.75
     * Bull Euphoric: 0.15
     * Range-Bound: 0.10
     * Bear Trending: 0.00
     * High Volatility: 0.20
   - Confidence threshold: only act if max probability > 0.6

3. Regime Interpretation (LLM Agent)
   - Input: classified regime + feature values + context
   - Task: Explain what this regime means strategically
   - Output: Strategic recommendations
     * "Bull trending confirmed: 20 EMA > 50 EMA, ADX > 25, volume increasing"
     * "Recommended actions: Favor trend-following, increase position size 1.2x"
     * "Risk adjustments: Reduce cash reserve to 5% (from 10%)"
     * "Avoid: Counter-trend plays, mean reversion strategies"
     * Reasoning: Why these adjustments make sense given features

4. Parameter Adjustment (algorithmic)
   - Apply recommended adjustments to system config
   - Update PortfolioManager prompts with regime context
   - Publish to Kafka: "regime_changed" event
   - All agents receive updated parameters and adapt behavior
```

**Why ML for classification?**

- Regime detection is a pattern recognition task (not reasoning)
- Historical data available for supervised learning
- ML models excel at multi-class classification
- LLMs are expensive, slow, and inconsistent for this
- ML model runs every hour at pennies cost vs dollars for LLM

**Why LLM for interpretation?**

- Explaining "what this regime means" requires reasoning and context
- LLMs can articulate strategic implications in plain language
- Output is human-readable for transparency and audit
- Can incorporate qualitative factors beyond pure numbers

**Data Requirements:**

Phase 4 (market data only):

- OHLCV from exchanges (free, real-time)
- Funding rates from perpetual contracts (free)
- Liquidation data (free from most exchanges)
- Cross-asset correlation (computed from above)

Phase 5+ (external data enhancement):

- News sentiment (NewsAPI, ~$50/mo)
- Social sentiment (Twitter API or free alternatives)
- Fear & Greed Index (free from Alternative.me)
- On-chain metrics (Glassnode/Santiment or free alternatives)

**Run Schedule:**

- Hourly classification (cheap, ~$0.01 ML + $0.05 LLM)
- Publish event only on regime change (reduce noise)
- Emergency re-classification on volatility spike (>5% move in 1 hour)
- Daily validation: compare predicted vs actual regime behavior

#### 2.3. Information Processing Layer (Data Intelligence)

**Purpose**: Bridge between raw data collection and agent decision-making. Transforms unstructured data streams into structured, classified, actionable intelligence.

**Problem Statement:**

Without this layer:

```
NewsAPI → 500 articles/day → Research Agents read all → $$$, slow, noisy
Twitter → 10,000 tweets/day → Agents drown in noise
OnChain → 1,000 transactions/day → Signal buried in data
```

**Solution Architecture:**

```
Raw Data Sources
    ↓
Workers (Collect)
    ↓
Information Processing Layer ⭐
    ├─ InformationClassifier: Categorize, score, filter
    ├─ SignalExtractor: Extract trading signals
    └─ ContextBuilder: Assemble relevant context on-demand
    ↓
Research Agents (Consume structured intelligence)
```

**Components:**

**1. InformationClassifier**

Hybrid approach: Fast algorithmic filtering + LLM for complex cases.

```go
type InformationClassifier struct {
    // Fast algorithmic path (90% of data)
    keywordMatcher    *KeywordMatcher    // Regex, keyword lists
    sentimentModel    *SentimentAnalyzer // FinBERT or similar ML model

    // LLM path for complex cases (10% of data)
    llmAgent          agent.Agent
}

type ClassifiedInformation struct {
    ID              string
    RawText         string
    Source          string      // "newsapi", "twitter", "onchain"
    Category        string      // "macro_news", "asset_news", "social_sentiment"
    Sentiment       string      // "bullish", "bearish", "neutral"
    SentimentScore  float64     // -1.0 to +1.0
    AffectedAssets  []string    // ["BTC", "ETH"]
    Relevance       float64     // 0.0 to 1.0 (trading relevance)
    Credibility     float64     // 0.0 to 1.0 (source quality)
    IsSignal        bool        // Worth alerting agents?
    Timestamp       time.Time
    ProcessedAt     time.Time
}
```

**Workflow:**

```
1. Worker collects: "Federal Reserve cuts interest rates by 50bps"

2. Algorithmic classification (fast):
   - Keywords: ["federal reserve", "rates", "cut"] → macro_news
   - Sentiment model: "rate cut" → bullish for risk assets
   - Source: Reuters → credibility 0.9

3. Scoring:
   - Relevance: 0.95 (macro affects all crypto)
   - Affected: ["BTC", "ETH", "GOLD", "SPX"]
   - IsSignal: true (important)

4. If ambiguous → LLM fallback:
   - Complex jargon or sarcasm
   - Conflicting signals
   - Novel event type

5. Save classified info + Publish to Kafka if IsSignal == true
```

**2. SignalExtractor**

Extracts actionable trading signals from classified information.

```go
type SignalExtractor struct {
    rulesEngine    *RulesEngine    // Deterministic rules
    llmAgent       agent.Agent     // Complex interpretation
    signalRepo     *SignalRepository
}

type TradingSignal struct {
    Type           string      // "macro_shift", "whale_activity", "sentiment_surge"
    Direction      string      // "bullish", "bearish"
    Confidence     float64     // 0.0 to 1.0
    AffectedAssets []string
    Evidence       []string    // IDs of supporting classified info
    ExpiresAt      time.Time   // Signals have lifetime
    Priority       Priority    // Critical, High, Medium, Low
    Reasoning      string      // Why this is a signal
}
```

**Signal Rules Examples:**

```go
// Rule 1: High-credibility macro news
if info.Category == "macro_news" &&
   info.Credibility > 0.8 &&
   info.Relevance > 0.9 {
    publishSignal(TradingSignal{
        Type:     "macro_shift",
        Priority: Critical,
    })
}

// Rule 2: Whale activity (LLM interpretation)
if info.Source == "onchain" &&
   info.Category == "large_transfer" &&
   info.Amount > threshold {
    signal := llmAgent.InterpretWhaleActivity(info)
    publishSignal(signal)
}

// Rule 3: Social sentiment surge
if info.Source == "twitter" &&
   volumeIncrease > 300% &&
   sentimentAlignment > 0.8 {
    publishSignal(TradingSignal{
        Type:     "sentiment_surge",
        Priority: Medium,  // Less reliable than macro
    })
}
```

**3. ContextBuilder**

On-demand service that assembles **relevant** context when agents request analysis.

```go
type ContextBuilder struct {
    infoRepo    *ClassifiedInformationRepository
    signalRepo  *SignalRepository
    cache       *ContextCache  // Cache frequent requests
}

// Agent tool calls: "Get me context for BTC analysis"
func (cb *ContextBuilder) BuildContext(req ContextRequest) (*TradingContext, error) {
    // 1. Get recent signals (last N hours)
    signals := cb.signalRepo.GetRecentSignals(
        req.Symbol,
        req.TimeWindow,
        minRelevance=0.7,
    )

    // 2. Get top N classified items (not all 10,000!)
    topInfo := cb.infoRepo.GetTopRelevant(
        req.Symbol,
        req.TimeWindow,
        limit=20,  // Only top 20 items
        minRelevance=0.6,
    )

    // 3. Structure by type
    return &TradingContext{
        Symbol:    req.Symbol,
        Timestamp: time.Now(),

        // High-level signals
        MacroSignals:      filterByType(signals, "macro_shift"),
        OnChainSignals:    filterByType(signals, "whale_activity"),
        SentimentSignals:  filterByType(signals, "sentiment_surge"),

        // Supporting evidence (top items only)
        RelevantNews:   filterByCategory(topInfo, "news", limit=10),
        RelevantSocial: filterByCategory(topInfo, "social", limit=5),
        OnChainEvents:  filterByCategory(topInfo, "onchain", limit=5),

        // Summary metrics
        NetSentiment:    calculateNetSentiment(signals),
        BullishCount:    countBullish(signals),
        BearishCount:    countBearish(signals),
        ConfidenceScore: calculateConfidence(signals, topInfo),
    }
}
```

**Agent Integration:**

```go
// Tool: get_external_context
type ExternalContextTool struct {
    contextBuilder *ContextBuilder
}

func (t *ExternalContextTool) Execute(params map[string]interface{}) (interface{}, error) {
    symbol := params["symbol"].(string)

    ctx := t.contextBuilder.BuildContext(ContextRequest{
        Symbol:     symbol,
        TimeWindow: 24 * time.Hour,
    })

    return ctx, nil
}
```

**Benefits:**

✅ **Efficiency**: Agents read 20 items, not 10,000  
✅ **Cost**: 95% reduction in LLM tokens for data processing  
✅ **Latency**: Critical signals delivered immediately (event-driven)  
✅ **Quality**: Filtered, scored, structured → better decisions  
✅ **Scalability**: Add new data sources without overloading agents

**Cost & Performance:**

- InformationClassifier: $0.001-0.01 per item (mostly algorithmic)
- SignalExtractor: $0.01-0.05 per signal (rules + occasional LLM)
- ContextBuilder: $0.00 (query only, cached)
- Total overhead: ~$5-10/day for 1000 items processed

**Implementation Note:**

Phase 1-4: Agents work without external data (only technical/structural tools).  
Phase 5: Add Information Layer when connecting first external data source.  
Use ContextProvider interface from start so agents don't need refactoring later.

---

### Tier 2: Client-Level Agents (Per-User Instances)

Each client has dedicated agents managing their specific portfolio.

#### 2.1. PortfolioManager (renamed from StrategyPlanner)

**Role**: Portfolio Manager responsible for a specific client's account. Receives fund's trading ideas and decides whether to execute for THIS client.

**Responsibilities:**

1. Monitor client's portfolio (positions, exposure, risk)
2. Receive opportunities from Research Committee
3. Assess fit: Does this opportunity suit THIS client?
4. Calculate position size based on client's risk profile
5. Validate via pre-trade checks
6. Approve or reject trade
7. Document reasoning for audit

**Prompt Framework:**

```tmpl
# ROLE

You are a Portfolio Manager at an AI hedge fund managing a client's account.

Client Profile:
- Capital: ${{.ClientCapital}}
- Risk Tolerance: {{.RiskProfile}} (conservative/moderate/aggressive)
- Current Holdings: {{.CurrentPositions}}
- Available Capital: ${{.AvailableCapital}}

# INPUT

The fund's Research Committee has identified a trading opportunity:

Opportunity: {{.Opportunity}}
- Symbol: {{.Symbol}}
- Direction: {{.Direction}}
- Entry: {{.EntryPrice}}
- Stop Loss: {{.StopLoss}}
- Take Profit: {{.TakeProfit}}
- Confidence: {{.Confidence}}%
- Reasoning: {{.ResearchReasoning}}

# YOUR TASK

Decide if this opportunity should be executed for THIS specific client.

## Step 1: Fit Analysis

1. Risk Profile Alignment
   - Is this opportunity too aggressive for client's risk tolerance?
   - Does confidence level justify the trade?

2. Capital Availability
   - Does client have sufficient available capital?
   - Check via get_account_status tool

3. Portfolio Correlation
   - Does client already have exposure to this asset?
   - Check via get_correlation tool
   - If correlated positions exist, reduce size significantly

4. Position Limits
   - Would this position exceed client's position size limits?
   - Max single position: {{.MaxPositionSize}}%

## Step 2: Position Sizing

Base calculation:
- Use Kelly Criterion adjusted for confidence
- Formula: Size = (Confidence * (R:R)) / R:R
- Adjust for client's risk profile:
  * Conservative: base_size * 0.5
  * Moderate: base_size * 1.0
  * Aggressive: base_size * 1.5

Correlation adjustment:
- If existing correlated positions > 0:
  * Reduce size by: existing_exposure / 2

Example:
- Base size: 10%
- Client: Moderate (1.0x)
- Existing BTC: 15%
- Final size: 10% * 1.0 - (15% / 2) = 2.5%

## Step 3: Risk Validation

Call pre_trade_check tool with proposed parameters:
- symbol: {{.Symbol}}
- amount: calculated_size
- direction: {{.Direction}}
- stop_loss: {{.StopLoss}}

If validation fails, REJECT trade.

## Step 4: Decision

Output format:
{
  "decision": "APPROVE" | "MODIFY" | "REJECT",
  "reasoning": "Step-by-step explanation",
  "position_size": calculated_amount,
  "entry": entry_price,
  "stop_loss": stop_price,
  "take_profit": target_price,
  "modifications": "If MODIFY, explain changes",
  "rejection_reason": "If REJECT, explain why",
  "confidence": adjusted_confidence,
  "reasoning_trace": [
    {"step": "fit_analysis", "result": "..."},
    {"step": "sizing", "result": "..."},
    {"step": "validation", "result": "..."},
    {"step": "decision", "result": "..."}
  ]
}

# CONSTRAINTS

- NEVER exceed client's risk limits
- ALWAYS check correlation via tools
- ALWAYS validate via pre_trade_check
- NEVER execute without available capital
- ALWAYS document reasoning for audit

You are managing REAL CLIENT MONEY. Be conservative but decisive.

When in doubt, REJECT and wait for better setup.
```

**Tools Available:**

- `get_account_status`: Portfolio, positions, risk profile
- `get_correlation`: Check correlation with existing positions
- `pre_trade_check`: Validate trade against risk engine
- `search_memory`: Look for similar past setups
- `save_memory`: Document decision reasoning

**Output Schema:**

```go
type PortfolioManagerDecision struct {
    Decision         string      `json:"decision"`          // APPROVE, MODIFY, REJECT
    Reasoning        string      `json:"reasoning"`
    PositionSize     float64     `json:"position_size"`
    Entry            float64     `json:"entry"`
    StopLoss         float64     `json:"stop_loss"`
    TakeProfit       float64     `json:"take_profit"`
    Modifications    string      `json:"modifications,omitempty"`
    RejectionReason  string      `json:"rejection_reason,omitempty"`
    Confidence       float64     `json:"confidence"`
    ReasoningTrace   []ReasoningStep `json:"reasoning_trace"`
    Evidence         Evidence    `json:"evidence"`
    Alternatives     []Alternative `json:"alternatives_considered"`
}

type ReasoningStep struct {
    Step      string      `json:"step"`
    Input     interface{} `json:"input"`
    Output    interface{} `json:"output"`
    Reasoning string      `json:"reasoning"`
}

type Evidence struct {
    Sources       []string  `json:"sources"`
    Timestamp     time.Time `json:"timestamp"`
    DataQuality   float64   `json:"data_quality"`
}

type Alternative struct {
    Option       string `json:"option"`
    WhyRejected  string `json:"why_rejected"`
}
```

#### 2.2. PortfolioArchitect (Onboarding)

**Role**: Designs initial portfolio allocation when a new client allocates capital.

**Trigger**: User sends `/invest 1000`

**Responsibilities:**

1. Assess client's risk profile (via questionnaire or defaults)
2. Fetch current market conditions
3. Design diversified portfolio allocation
4. Calculate position sizes for each asset
5. Validate each position via pre_trade_check
6. Execute portfolio via execute_trade
7. Document strategy for client

**Prompt Framework:**

```tmpl
# ROLE

You are a Portfolio Architect designing initial portfolio allocation for a new client.

Client Details:
- Capital: ${{.Capital}}
- Risk Tolerance: {{.RiskProfile}}
- Preferred Assets: {{.PreferredAssets}} (optional)
- Investment Horizon: {{.TimeHorizon}}

Current Market Regime: {{.MarketRegime}}

# TASK

Design a balanced, diversified portfolio allocation.

## Step 1: Market Assessment

Use analysis tools to understand current market:
- get_technical_analysis: Overall market trend
- get_smc_analysis: Key structural levels
- get_market_analysis: Current flow and sentiment

Determine market regime suitability:
- Bull trending: Can allocate more to alts
- Bear: Increase BTC/stablecoin allocation
- High volatility: Reduce exposure, increase cash

## Step 2: Asset Selection

Select 4-6 assets based on:
- Client's risk profile
- Market regime
- Diversification (low correlation)
- Liquidity (can exit easily)

Use calc_asset_correlation tool to verify diversification.

## Step 3: Allocation Design

Guidelines by risk profile:

Conservative (15-30% annual target):
- 70% large-cap (BTC, ETH)
- 20% mid-cap (SOL, BNB)
- 10% cash reserve
- Max 15% per asset

Moderate (30-60% annual target):
- 50% large-cap
- 30% mid-cap
- 15% small-cap
- 5% cash reserve
- Max 20% per asset

Aggressive (60-150% annual target):
- 30% large-cap
- 40% mid-cap
- 25% small-cap
- 5% cash reserve
- Max 30% per asset

Market regime adjustments:
- Bull: +10% alts, -10% BTC/cash
- Bear: +15% BTC/cash, -15% alts
- High volatility: +10% cash, reduce exposure

## Step 4: Position Sizing

For each selected asset:
1. Calculate dollar amount: total_capital * allocation%
2. Verify correlation limits (use calc_asset_correlation)
3. Validate via pre_trade_check
4. If validation fails, adjust or skip

## Step 5: Execution

For each approved position:
1. Call execute_trade with parameters
2. Set stop-loss levels (conservative: -15%, moderate: -25%, aggressive: -35%)
3. Document reasoning

## Step 6: Strategy Documentation

Save portfolio strategy to memory:
- Asset allocations and reasoning
- Risk management rules
- Rebalancing triggers
- Expected timeframe

Output format:
{
  "portfolio": [
    {
      "asset": "BTC",
      "allocation_pct": 40,
      "amount_usd": 400,
      "entry_price": 43500,
      "stop_loss": 37000,
      "reasoning": "Core holding, market leader"
    },
    ...
  ],
  "strategy": {
    "risk_profile": "moderate",
    "total_invested": 900,
    "cash_reserve": 100,
    "rebalancing_frequency": "weekly",
    "portfolio_stop": -25%
  },
  "reasoning_trace": [...]
}

# CONSTRAINTS

- ALWAYS maintain cash reserve (5-10%)
- NEVER exceed correlation limits
- ALWAYS validate each position via pre_trade_check
- ALWAYS set stop-losses
- Document everything for client transparency
```

**Tools:**

- `get_technical_analysis`, `get_smc_analysis`, `get_market_analysis`: Market snapshot
- `calc_asset_correlation`: Verify diversification
- `pre_trade_check`: Validate each position
- `execute_trade`: Place orders
- `save_memory`: Document strategy

---

### Tier 3: Position-Level Monitoring (Event-Driven)

#### 3.1. PositionGuardian (Event-Driven Architecture)

**Role**: 24/7 real-time monitoring of open positions with immediate reaction to critical events.

**Current Problem:**

```go
// WRONG: Polling-based monitoring
func (w *PositionMonitorWorker) Run() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        positions := w.getOpenPositions()
        for _, pos := range positions {
            w.agent.EvaluatePosition(pos)
        }
    }
}
```

**Issues:**

- 5-minute delay between event and reaction
- Stop hit at 10:00:00 → detected at 10:05:00 → **loss amplified**
- Inefficient: checks all positions even when nothing changed

**Correct Architecture: Event-Driven**

```go
// Event types
type PositionEventType string

const (
    EventStopApproaching     PositionEventType = "stop_approaching"      // Within 2% of stop
    EventStopHit             PositionEventType = "stop_hit"              // Stop triggered
    EventTargetApproaching   PositionEventType = "target_approaching"    // Within 5% of target
    EventTargetHit           PositionEventType = "target_hit"            // Target reached
    EventThesisInvalidation  PositionEventType = "thesis_invalidation"   // Structure broken
    EventTimeDecay           PositionEventType = "time_decay"            // Position open too long
    EventProfitMilestone     PositionEventType = "profit_milestone"      // +5%, +10%, +20%
    EventCorrelationSpike    PositionEventType = "correlation_spike"     // Risk concentration
    EventRegimeChange        PositionEventType = "regime_change"         // Market regime shifted
    EventVolatilitySpike     PositionEventType = "volatility_spike"      // Sudden price swing
)

type PositionEvent struct {
    Type        PositionEventType
    Position    domain.Position
    Trigger     interface{}  // Event-specific data
    Urgency     Urgency      // Low, Medium, High, Critical
    Timestamp   time.Time
}

// Event-driven monitoring service
type PositionGuardian struct {
    // Event sources
    priceStreams    map[string]*WebSocketStream  // Real-time price feeds
    kafkaConsumer   *kafka.Consumer              // thesis_invalidation, regime_change

    // Event handlers
    criticalHandler *CriticalEventHandler  // Immediate algorithmic action
    agentHandler    *AgentEventHandler     // LLM-based decision making

    // Agent for complex decisions
    agent           agent.Agent

    // Execution service
    executionSvc    *ExecutionService
}

func (pg *PositionGuardian) Start(ctx context.Context) error {
    // Subscribe to real-time price streams for all open positions
    positions, err := pg.positionRepo.GetOpenPositions(ctx)
    if err != nil {
        return err
    }

    for _, pos := range positions {
        pg.subscribeToPriceStream(pos.Symbol, pos.Exchange)
    }

    // Subscribe to Kafka events
    pg.subscribeToKafkaEvents([]string{
        "thesis_invalidation",
        "regime_changed",
        "risk_alert",
    })

    // Event loop
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()

        case priceUpdate := <-pg.priceUpdates:
            pg.checkPriceTriggers(priceUpdate)

        case kafkaEvent := <-pg.kafkaEvents:
            pg.handleKafkaEvent(kafkaEvent)

        case timeEvent := <-pg.timeTriggers:
            pg.checkTimeBasedEvents(timeEvent)
        }
    }
}

func (pg *PositionGuardian) checkPriceTriggers(update PriceUpdate) {
    positions := pg.positionRepo.GetPositionsBySymbol(update.Symbol)

    for _, pos := range positions {
        // Check stop-loss
        if pg.isStopHit(pos, update.Price) {
            pg.handleEvent(PositionEvent{
                Type:     EventStopHit,
                Position: pos,
                Urgency:  Critical,
            })
        } else if pg.isStopApproaching(pos, update.Price) {
            pg.handleEvent(PositionEvent{
                Type:     EventStopApproaching,
                Position: pos,
                Urgency:  High,
            })
        }

        // Check take-profit
        if pg.isTargetHit(pos, update.Price) {
            pg.handleEvent(PositionEvent{
                Type:     EventTargetHit,
                Position: pos,
                Urgency:  Medium,
            })
        }

        // Check profit milestones
        if milestone := pg.checkProfitMilestone(pos, update.Price); milestone > 0 {
            pg.handleEvent(PositionEvent{
                Type:     EventProfitMilestone,
                Position: pos,
                Trigger:  milestone,
                Urgency:  Medium,
            })
        }
    }
}

func (pg *PositionGuardian) handleEvent(event PositionEvent) {
    log := pg.log.With(
        "event_type", event.Type,
        "position_id", event.Position.ID,
        "urgency", event.Urgency,
    )

    switch event.Urgency {
    case Critical:
        // IMMEDIATE algorithmic action, no LLM delay
        pg.criticalHandler.Handle(event)

    case High, Medium:
        // Queue for agent evaluation
        pg.agentHandler.Handle(event)

    case Low:
        // Batch for next scheduled review
        pg.queueForBatchProcessing(event)
    }
}

// Critical event handler: deterministic, fast, no LLM
type CriticalEventHandler struct {
    executionSvc *ExecutionService
    notifier     *Notifier
}

func (h *CriticalEventHandler) Handle(event PositionEvent) {
    switch event.Type {
    case EventStopHit:
        // Close position immediately
        h.executionSvc.ClosePosition(event.Position, "Stop-loss hit")
        h.notifier.NotifyClient(event.Position.UserID,
            fmt.Sprintf("Position %s closed at stop-loss", event.Position.Symbol))

    case EventTargetHit:
        // Close position at target
        h.executionSvc.ClosePosition(event.Position, "Take-profit hit")
        h.notifier.NotifyClient(event.Position.UserID,
            fmt.Sprintf("Position %s closed at target profit", event.Position.Symbol))
    }
}

// Agent event handler: uses LLM for complex decisions
type AgentEventHandler struct {
    agent        agent.Agent
    executionSvc *ExecutionService
}

func (h *AgentEventHandler) Handle(event PositionEvent) {
    // Prepare context for agent
    input := map[string]interface{}{
        "event_type": event.Type,
        "position":   event.Position,
        "trigger":    event.Trigger,
        "urgency":    event.Urgency,
    }

    // Ask agent to evaluate
    response, err := h.agent.Run(context.Background(), input)
    if err != nil {
        // Fallback to safe default
        h.handleAgentFailure(event, err)
        return
    }

    // Execute agent's decision
    decision := response.Output.(PositionDecision)
    h.executeDecision(decision, event.Position)
}
```

**Agent Prompt for PositionGuardian:**

```tmpl
# ROLE

You are a Position Monitor on the trading desk, responsible for managing active positions in real-time.

# EVENT

Type: {{.EventType}}
Position: {{.Position}}
Trigger: {{.Trigger}}
Urgency: {{.Urgency}}

# CONTEXT

Current Market:
- Price: {{.CurrentPrice}}
- Entry: {{.Position.EntryPrice}}
- P&L: {{.Position.PnL}}% ({{.Position.PnLUSD}})
- Stop: {{.Position.StopLoss}}
- Target: {{.Position.TakeProfit}}
- Time in position: {{.Position.Duration}}

Original Thesis:
{{.Position.EntryReasoning}}

# YOUR TASK

Evaluate the situation and decide on an action.

## Decision Framework

1. **Thesis Validation**
   - Is the original thesis still valid?
   - Has market structure changed?
   - Are we still in the same regime?

2. **Risk Assessment**
   - Current risk vs remaining reward (R:R ratio)
   - Is R:R still favorable?
   - Any new risks emerged?

3. **Action Options**
   - HOLD: Thesis intact, R:R acceptable, within limits
   - TRAIL_STOP: Position profitable, lock in gains
   - TRIM: Take partial profit, reduce exposure
   - EXIT: Thesis invalidated or poor R:R
   - ADD: Thesis strengthening, position profitable (pyramiding)

## Event-Specific Guidance

If event is **stop_approaching**:
- Reassess thesis urgently
- If thesis weakening → prepare to exit
- If thesis strong → consider if stop is too tight

If event is **target_approaching**:
- Evaluate: Take profit or let it run?
- Check if momentum strong → trail stop
- If momentum weak → take profit

If event is **thesis_invalidation**:
- Exit immediately or very soon
- Thesis invalidation is serious signal

If event is **profit_milestone** (+5%, +10%, +20%):
- ALWAYS trail stop to at least breakeven at +5%
- Consider taking partial profit at +10%
- Trail aggressively at +20%

If event is **regime_change**:
- Reassess if this position fits new regime
- Defensive regime → tighten stops or exit

## Output Format

{
  "action": "HOLD" | "TRAIL_STOP" | "TRIM" | "EXIT" | "ADD",
  "reasoning": "Clear explanation of decision",
  "parameters": {
    "new_stop": price (if trailing),
    "trim_percentage": 0-100 (if trimming),
    "exit_type": "market" | "limit" (if exiting)
  },
  "urgency": "immediate" | "within_hour" | "next_review",
  "reasoning_trace": [...]
}

# CONSTRAINTS

- NEVER hold without stops
- NEVER let winners turn into losers (trail stops!)
- NEVER average down on losers
- ALWAYS respect risk limits
- Cut losers fast, let winners run

Discipline over emotion. Systematic management.
```

**Benefits of Event-Driven Architecture:**

✅ **Immediate Reaction**: Stop hit → closed in <1 second, not 5 minutes  
✅ **Efficient**: Only processes changes, not all positions every tick  
✅ **Scalable**: Can monitor 1000s of positions without overhead  
✅ **Flexible**: Different urgency levels for different events  
✅ **Hybrid**: Algorithmic for critical, LLM for complex

---

### Tier 4: Learning & Adaptation (Periodic)

#### 4.1. PreTradeReviewer (Split from SelfEvaluator)

**Role**: Quality gate before trade execution. Red-teams the trade plan and conducts pre-mortem analysis.

**Trigger**: PortfolioManager outputs "APPROVE" decision

**Before Execution:**

1. Review the trade plan
2. Challenge assumptions
3. Play devil's advocate: What could go wrong?
4. Check data completeness and freshness
5. Score confidence honestly
6. Output: GO / HOLD / NO-GO

**Prompt:**

```tmpl
# ROLE

You are a Pre-Trade Reviewer on the Risk Committee. Your job: challenge trade plans before execution.

# INPUT

Trade Plan: {{.TradePlan}}
- Symbol: {{.Symbol}}
- Direction: {{.Direction}}
- Position Size: {{.PositionSize}}
- Entry/Stop/Target: {{.Levels}}
- Reasoning: {{.Reasoning}}
- Confidence: {{.Confidence}}%

Portfolio Manager's Analysis: {{.PMAnalysis}}

# YOUR TASK

Red-team this trade. Find flaws, challenge assumptions, conduct pre-mortem.

## Quality Checklist

1. **Data Completeness**
   - Is analysis based on fresh data (<1 hour old)?
   - Are all key indicators present?
   - Any critical data missing?

2. **Reasoning Quality**
   - Does reasoning flow logically?
   - Evidence supports conclusions?
   - Any circular logic or confirmation bias?

3. **Conflict Resolution**
   - Are there contradicting signals?
   - If yes, are they acknowledged and resolved?
   - Or are conflicts being ignored?

4. **Risk Management**
   - Is stop-loss defined?
   - Is R:R ratio favorable (>2:1)?
   - Position sizing appropriate?

5. **Pre-Mortem**
   - What's the bear case (if going long)?
   - What could invalidate this thesis?
   - What's the probability of being wrong?

## Confidence Calibration

Portfolio Manager says {{.Confidence}}%. Is this realistic?

- Overconfident if: Single source, no conflicts, extreme conviction
- Appropriate if: Multiple sources, conflicts resolved, measured confidence
- Underconfident if: Strong consensus, clear setup, hesitant confidence

Adjust confidence if needed.

## Decision Criteria

**GO** (approve execution):
- High confidence (>70%)
- Clear plan with defined entry/stop/target
- Risk managed, data complete
- Reasoning sound, conflicts resolved
- Pre-mortem conducted, risks acceptable

**HOLD** (delay execution):
- Medium confidence (50-70%)
- Conflicts exist OR missing data OR event risk ahead
- "Wait and see" approach warranted
- Could be GO later with more data

**NO-GO** (reject execution):
- Low confidence (<50%)
- Major conflicts unresolved
- Critical data gaps
- High risk, poor R:R
- Reasoning flawed

## Output Format

{
  "decision": "GO" | "HOLD" | "NO-GO",
  "adjusted_confidence": 75,
  "concerns": [
    "Overweight correlation not fully addressed",
    "Data is 2 hours old, may be stale"
  ],
  "pre_mortem_scenarios": [
    {
      "scenario": "Support breaks and triggers cascade",
      "probability": 0.25,
      "impact": "Full stop-loss"
    }
  ],
  "recommendation": "GO with reduced position size (50%)",
  "reasoning": "Setup is solid but correlation concern reduces confidence"
}

# CONSTRAINTS

- NEVER approve without stop-loss
- NEVER ignore major conflicts
- NEVER proceed with <40% confidence
- ALWAYS conduct pre-mortem
- Better to miss opportunity than take bad trade

Your job is to PREVENT mistakes, not to approve trades.
```

#### 4.2. PerformanceCommittee (Split from SelfEvaluator)

**Role**: Weekly performance review. Analyzes closed trades, extracts patterns, updates strategy playbook.

**Trigger**: Cron job, Sunday 00:00 UTC

**Responsibilities:**

1. Collect all closed trades from past week
2. Group by setup type, regime, confidence level
3. Calculate metrics (win rate, R:R, Sharpe)
4. Identify validated patterns
5. Identify failure modes
6. Extract actionable lessons
7. Save to memory for system improvement

**Prompt:**

```tmpl
# ROLE

You are the Performance Committee conducting weekly review of the fund's trading results.

# INPUT

Trade Data (Past Week):
- Total trades: {{.TotalTrades}}
- Win rate: {{.WinRate}}%
- Avg R:R: {{.AvgRR}}
- Total P&L: {{.TotalPnL}}%

Closed Trades: {{.ClosedTrades}}

# YOUR TASK

Analyze performance, extract patterns, provide actionable insights.

## Analysis Dimensions

### 1. By Setup Type
Group trades:
- Breakout: Win rate? Volume pattern correlation?
- Reversal: Success rate? Divergence effectiveness?
- Continuation: Pullback vs trend-following?
- Structure: SMC setups working?

For each setup type:
- Count trades
- Win rate
- Avg R:R
- Best conditions (when it works)
- Failure modes (when it fails)

### 2. By Market Regime
- Bull trending: Which strategies worked?
- Range-bound: Mean reversion vs breakout?
- High volatility: Performance degradation?

### 3. By Confidence Level
Agent calibration check:
- 70-80% confidence → actual win rate?
- 80-90% confidence → actual win rate?
- Are high-confidence signals actually better?
- Is the fund overconfident or underconfident?

### 4. By Timing
- Entry precision: Early vs late entries?
- Exit discipline: Hit targets or stopped out?
- Hold duration: Optimal timeframes?

## Pattern Extraction

Validated patterns (sample size ≥10):

Example:
"Breakout trades with volume >2x average: 15 trades, 73% win rate, avg R:R 3.1:1.
Failed breakouts (4 trades) all had volume <1.5x.
Lesson: Require volume confirmation >2x for breakout signals."

For each pattern:
- Setup description
- Sample size
- Win rate
- Conditions (when to apply)
- Actionable rule

## Failure Mode Analysis

Common mistakes:
- Overtrading: >3 trades/day → lower win rate?
- Sizing errors: Too large → bigger losses?
- Correlation: Concentrated exposure → correlated losses?
- Timing: Premature entries → stopped out?

## Lessons Learned

Actionable insights for system improvement:

Categories:
1. **Strategy Validation**: Which strategies work best?
2. **Agent Calibration**: Confidence accuracy, sizing appropriateness
3. **Risk Insights**: Position limits, stop placement, correlation management
4. **Market Regime**: Strategy performance by regime

## Output Format

{
  "summary": {
    "total_trades": 47,
    "win_rate": 0.68,
    "avg_rr": 2.4,
    "best_strategy": "SMC reversals in bull trending",
    "worst_strategy": "Breakouts in range-bound"
  },
  "validated_patterns": [
    {
      "pattern": "Support bounce + order block",
      "sample_size": 12,
      "win_rate": 0.75,
      "avg_rr": 2.8,
      "conditions": "4H timeframe, bull regime, RSI <70",
      "actionable_rule": "Favor support bounces with SMC confluence in bull markets"
    }
  ],
  "failure_modes": [
    {
      "issue": "Breakouts in low volume",
      "sample_size": 8,
      "win_rate": 0.25,
      "lesson": "Skip breakouts without 2x volume confirmation"
    }
  ],
  "recommendations": [
    "Increase position size for SMC reversals (proven 75% win rate)",
    "Reduce or skip breakout strategies during range-bound regime",
    "OpportunitySynthesizer is well-calibrated (75% stated vs 73% actual)"
  ]
}

# CONSTRAINTS

- NEVER conclude from <10 trades (insufficient sample)
- ALWAYS include sample size, win rate, conditions
- ALWAYS make lessons actionable (what to do, not just observations)
- Focus on PATTERNS, not individual trades

Learn from outcomes. Find patterns. Improve system.
```

**Output:**

- Save validated patterns to memory
- Update strategy playbook
- Publish performance report to clients

---

## 3. Communication & Orchestration

### 3.1. Event Bus (Kafka Topics)

All inter-agent communication via Kafka:

```
# Research → Portfolio Management
opportunity_identified:
  - symbol, exchange, direction
  - entry, stop, target
  - confidence, reasoning
  - analyst_reports (all 4 analysts)

# Market Intelligence → All Agents
regime_changed:
  - old_regime, new_regime
  - confidence, timestamp
  - parameter_adjustments

# Risk → All Agents
risk_alert:
  - type: circuit_breaker | correlation_spike | volatility_spike
  - affected_symbols
  - recommended_action

# Portfolio Management → Execution
trade_approved:
  - user_id, symbol, exchange
  - direction, amount, entry, stop, target
  - reasoning, confidence

# Execution → Position Monitoring
position_opened:
  - position_id, user_id
  - symbol, exchange, direction
  - entry_price, amount, stop, target

position_closed:
  - position_id, exit_price, pnl
  - close_reason (stop_hit | target_hit | manual | thesis_invalidation)

# Position Monitoring → Portfolio Management
thesis_invalidation:
  - position_id, symbol
  - reason (structure_broken | regime_change | time_decay)
  - recommended_action
```

### 3.2. Supervisor Agent (Fund Director)

**Optional but Recommended**: Meta-agent that orchestrates the entire system.

**Responsibilities:**

1. Receives user requests (via Telegram)
2. Determines which workflow to initiate
3. Monitors workflow execution
4. Handles conflicts between agents
5. Makes system-level decisions

**Example Flow:**

```
User: /invest 1000

SupervisorAgent:
1. Analyzes request → identifies workflow: "onboarding"
2. Creates PortfolioArchitect agent instance for this user
3. Monitors execution
4. If errors occur → decides recovery strategy
5. Confirms completion → notifies user
```

---

## 4. Memory Architecture

### Three Memory Types

```go
type MemorySystem struct {
    Episodic  *EpisodicMemory   // What happened (trade history)
    Semantic  *SemanticMemory   // Validated knowledge (patterns, rules)
    Working   *WorkingMemory    // Current state (active beliefs, pending actions)
}
```

### 4.1. Episodic Memory

**What**: Record of specific events and episodes.

**Structure:**

```go
type TradeEpisode struct {
    ID              string
    UserID          string
    Symbol          string
    Direction       string
    Entry           float64
    Exit            float64
    PnL             float64
    Duration        time.Duration

    // Context at entry
    EntryContext    EpisodeContext

    // Context at exit
    ExitContext     EpisodeContext

    // Decision trail
    Reasoning       []ReasoningStep
}

type EpisodeContext struct {
    Timestamp       time.Time
    Regime          string
    TechnicalSignals map[string]interface{}
    SMCSignals      map[string]interface{}
    FlowSignals     map[string]interface{}
    Confidence      float64
}
```

**Usage:**

- Agents query: "Show me similar past trades for BTC support bounce"
- System returns: Trade episodes with similar context
- Agents learn: "Last 3 times this setup occurred, 2 were profitable"

### 4.2. Semantic Memory

**What**: Extracted knowledge, validated patterns, general rules.

**Structure:**

```go
type ValidatedPattern struct {
    ID              string
    Name            string
    Description     string
    SampleSize      int
    WinRate         float64
    AvgRR           float64
    Conditions      []string
    ActionableRule  string
    Confidence      float64
    LastValidated   time.Time
}

type MarketRegime struct {
    Name            string
    Characteristics []string
    BestStrategies  []string
    AvoidStrategies []string
    ParameterAdj    map[string]float64
}

type UserPreference struct {
    UserID          string
    RiskProfile     string
    PreferredAssets []string
    Constraints     map[string]interface{}
}
```

**Usage:**

- Agents query: "What strategies work best in bull trending regime?"
- System returns: Validated patterns for that regime
- Agents apply: Favor those strategies in current decisions

### 4.3. Working Memory

**What**: Temporary state, active hypotheses, pending actions.

**Structure:**

```go
type WorkingMemory struct {
    // Current beliefs (per agent)
    Beliefs         map[string]Hypothesis

    // Pending actions
    PendingActions  []PlannedAction

    // Temporary context
    SessionContext  map[string]interface{}
}

type Hypothesis struct {
    Statement       string
    Confidence      float64
    Evidence        []string
    ExpiresAt       time.Time
}

type PlannedAction struct {
    Action          string
    TriggerCondition string
    Params          map[string]interface{}
}
```

**Usage:**

- Agent forms hypothesis: "BTC support at $42k will hold"
- Stores in working memory with confidence and expiry
- If hypothesis invalidated → triggers re-evaluation

---

## 5. Prompt Engineering Guidelines

### Chain-of-Thought Framework

All agent prompts should use explicit reasoning steps:

```
# REASONING FRAMEWORK

## Step 1: Evidence Gathering
[List what data/signals we have]

## Step 2: Conflict Resolution
[If signals contradict, how to resolve?]

## Step 3: Confidence Calibration
[Am I overconfident? What's the sample size?]

## Step 4: Decision with Reasoning
[Final decision with explicit reasoning chain]
```

### Self-Questioning

Agents should challenge themselves:

```
Questions to ask:
- What could I be missing?
- What's the opposite case?
- If I'm wrong, what would be the early warning sign?
- Am I overconfident given the sample size?
- Is this reasoning or rationalization?
```

### Evidence Citation

All conclusions must cite evidence:

```
Decision: BTC LONG setup
Because:
  - Technical: RSI recovery (45→58), MACD cross, EMA aligned
  - SMC: Order block support at $42.5k, FVG below
  - Flow: CVD positive (+$45M), whale accumulation
Confidence: 78% (3/3 sources aligned, clear levels, favorable R:R)
```

### Reasoning Trace

All outputs include step-by-step reasoning:

```json
"reasoning_trace": [
  {
    "step": "evidence_gathering",
    "input": {"tools": ["technical", "smc", "flow"]},
    "output": {"signals": [...]},
    "reasoning": "Called 3 analysis tools to gather evidence"
  },
  {
    "step": "synthesis",
    "input": {"signals": [...]},
    "output": {"consensus": "bullish", "conflicts": []},
    "reasoning": "All 3 sources agree on bullish direction"
  },
  {
    "step": "decision",
    "input": {"consensus": "bullish", "confidence": 0.78},
    "output": {"action": "PUBLISH"},
    "reasoning": "Meets criteria: consensus ≥2/3, confidence ≥70%, R:R ≥2:1"
  }
]
```

---

## 6. Implementation Checklist

### Phase 1: Core Refactoring (Weeks 1-2) ✅ COMPLETED

- [x] **Rename Agents**

  - [x] StrategyPlanner → PortfolioManager
  - [x] Update all references, configs, tools

- [x] **Split SelfEvaluator**

  - [x] Create PreTradeReviewer agent
  - [x] Create PerformanceCommittee agent
  - [x] Separate prompts and workflows

- [x] **Update Schemas**

  - [x] Add reasoning_trace to all output schemas
  - [x] Add evidence section
  - [x] Add alternatives_considered section

- [x] **Rewrite Prompts**

  - [x] PortfolioManager: hedge fund PM voice
  - [x] PositionGuardian: trading desk monitor voice (position_manager.tmpl)
  - [x] All agents: add Chain-of-Thought framework
  - [x] OpportunitySynthesizer: CoT with tool orchestration
  - [x] PortfolioArchitect: CoT with 6-step allocation process
  - [x] RegimeDetector: LLM interpretation focus (not classification)
  - [x] PerformanceAnalyzer: CoT with pattern extraction
  - [x] PreTradeReviewer: Quality gate with pre-mortem analysis
  - [x] PerformanceCommittee: Weekly review with statistical rigor

- [x] **Future-Proof Interfaces**
  - [x] Create ContextProvider interface in tools (Phase 5 preparation)
  - [x] Implement NoOpContextProvider (returns empty, no external data yet)
  - [x] Agent prompts include optional {{if .ExternalContext}} blocks
  - [x] Ready for Phase 5: swap NoOp → IntelligentContextProvider without agent changes

### Phase 2: Event-Driven Architecture (Weeks 2-3) ✅ COMPLETED

- [x] **Event System**

  - [x] Define PositionEvent types (7 new events: StopApproaching, TargetApproaching, ThesisInvalidation, TimeDecay, ProfitMilestone, CorrelationSpike, VolatilitySpike)
  - [x] Implement event publisher/subscriber
  - [x] Add Kafka topics for position events
  - [x] EventUrgency enum (LOW, MEDIUM, HIGH, CRITICAL)

- [x] **PositionGuardian Refactor**

  - [x] Replace polling worker with event-driven service (PositionEventGenerator in `internal/workers/trading/position_event_generator.go`)
  - [ ] Subscribe to price streams (WebSocket) - SKIPPED (polling retained for simplicity)
  - [x] Subscribe to Kafka events (PositionGuardianConsumer in `internal/consumers/position_guardian_consumer.go`)
  - [x] Implement CriticalEventHandler (algorithmic in `internal/services/position/critical_handler.go`)
  - [x] Implement AgentEventHandler (LLM-based in `internal/services/position/agent_handler.go`)

- [ ] **Price Stream Integration** - DEFERRED
  - [ ] WebSocket connections to exchanges
  - [ ] Real-time price monitoring (currently polling every 30-60s)
  - [ ] Event generation on triggers (implemented, but triggered by polling, not WebSocket)

**Implementation Notes:**

- ✅ Polling retained (30-60s intervals) for price updates instead of WebSocket
- ✅ Event-driven architecture for event handling (not price fetching)
- ✅ Critical events (stop/target hit) handled algorithmically (<1s)
- ✅ HIGH/MEDIUM events (stop/target approaching, profit milestones) handled via LLM agent
- ✅ Event routing by urgency: CRITICAL → CriticalEventHandler, HIGH/MEDIUM → AgentEventHandler
- ✅ All 7 position event types implemented and integrated
- ✅ Fallback logic: if agent fails on ThesisInvalidation → close position automatically

### Phase 3: Multi-Agent Research Committee (Weeks 3-4) ⏳ IN PROGRESS

- [x] **Agent Creation** (agents defined, prompts ready)

  - [x] TechnicalAnalyst agent + prompt (`pkg/templates/prompts/agents/technical_analyst.tmpl`)
  - [x] StructuralAnalyst agent + prompt (`pkg/templates/prompts/agents/structural_analyst.tmpl`)
  - [x] FlowAnalyst agent + prompt (`pkg/templates/prompts/agents/flow_analyst.tmpl`)
  - [x] MacroAnalyst agent + prompt (`pkg/templates/prompts/agents/macro_analyst.tmpl`)
  - [x] HeadOfResearch agent + prompt (`pkg/templates/prompts/agents/head_of_research.tmpl`)
  - [x] Agent types registered in `internal/agents/types.go`

- [ ] **Workflow Orchestration** ⚠️ NOT IMPLEMENTED

  - [ ] Implement ResearchCommittee workflow (file exists: `internal/agents/workflows/research_committee.go` but needs implementation)
  - [ ] Parallel execution of 4 analysts (coordination logic needed)
  - [ ] Sequential synthesis by HeadOfResearch
  - [ ] Debate and consensus logic
  - [ ] Integration with Kafka (publish opportunity_identified events)

- [x] **Fast-Path Alternative**
  - [x] OpportunitySynthesizer implemented (`internal/agents/workflows/opportunity_synthesizer_test.go`)
  - [ ] Path selection logic (decide between fast-path vs multi-agent committee)
  - [ ] Benchmark: speed, cost, quality

**Status:**

- **Completed:** Agent definitions, prompts, type registration
- **Blocked:** Workflow orchestration needs ADK multi-agent coordination patterns
- **Next:** Implement parallel analyst execution + synthesis workflow

### Phase 4: ML-Based Regime Detection (Week 4) ⏳ IN PROGRESS

- [x] **Data Foundation** (partially complete)

  - [x] Exchange data collectors implemented (`internal/workers/marketdata/`)
    - [x] OHLCV collector
    - [x] Funding rates collector
    - [x] Liquidation collector
    - [x] Ticker, trades, orderbook collectors
  - [x] Feature extraction worker (`internal/workers/analysis/feature_extractor.go`)
  - [x] ClickHouse schema for time-series features (in `migrations/clickhouse/`)
  - [ ] Historical data backfill (6-12 months minimum) ⚠️ NOT DONE

- [ ] **ML Model (Market Data Based)** ⚠️ INFRASTRUCTURE READY, MODEL NOT TRAINED

  - [x] Feature engineering implemented in `internal/workers/analysis/feature_extractor.go`
  - [x] Python training scripts ready:
    - [x] `scripts/ml/regime/train_model.py` - Random Forest/XGBoost training
    - [x] `scripts/ml/regime/label_data.py` - manual labeling tool
  - [x] ONNX model integration (`internal/ml/regime/classifier.go`)
  - [ ] Train model on historical regimes ⚠️ NO LABELED DATA YET
  - [ ] Backtest on historical data, validate accuracy
  - [ ] Deploy trained model file

- [x] **LLM Integration** (worker ready, needs trained model)
  - [x] RegimeDetectorML worker (`internal/workers/analysis/regime_detector_ml.go`)
  - [x] RegimeInterpreter agent defined (`internal/agents/types.go`)
  - [x] RegimeInterpreter prompt (`pkg/templates/prompts/agents/regime_interpreter.tmpl`)
  - [x] Parameter adjustment logic in worker
  - [x] Kafka "regime_changed" event publishing
  - [ ] End-to-end testing with real model

**Status:**
- **Completed:** Infrastructure (workers, collectors, schemas, agent definitions)
- **Blocked:** Need to collect 6-12 months historical data → label regimes → train model
- **Next:** Run historical backfill → manual regime labeling → train Random Forest model

### Phase 5: Information Processing Layer (Week 5) ❌ NOT STARTED

**Purpose**: Bridge raw external data and agent intelligence. Enhances both Research agents (external context) and RegimeDetector (sentiment features).

- [ ] **Data Source Integration** ⚠️ PARTIAL (on-chain collectors exist)

  - [ ] Connect NewsAPI or similar news aggregator
  - [ ] Twitter/Reddit API integration (optional, or use free alternatives)
  - [x] On-chain data collectors implemented (`internal/workers/onchain/`)
    - [x] Whale movement collector
    - [x] Exchange flow collector
    - [x] Network metrics collector
    - [x] Miner metrics collector
  - [ ] Fear & Greed Index collector (free from Alternative.me)
  - [x] Sentiment collectors implemented (`internal/workers/sentiment/`)
    - [x] News collector skeleton
    - [x] Twitter/Reddit collector skeletons
    - [x] Fear & Greed collector skeleton
  - [ ] Raw data → ClickHouse storage (schema exists, but workers not fully integrated)

- [ ] **InformationClassifier (Hybrid: Algorithmic + LLM)** ⚠️ NOT IMPLEMENTED

  - [ ] Keyword matcher + sentiment ML model (FinBERT or similar)
  - [ ] LLM agent for complex/ambiguous texts (fallback, ~10% of items)
  - [ ] Classification schema: category, sentiment, relevance, credibility
  - [ ] Repository for classified information (PostgreSQL + pgvector for semantic search)
  - [ ] Batch processing: classify collected data every 5-15 minutes

- [ ] **SignalExtractor (Event-Driven)** ⚠️ NOT IMPLEMENTED

  - [ ] Rules engine for deterministic signal extraction
  - [ ] Signal types: macro_shift, whale_activity, sentiment_surge, onchain_anomaly
  - [ ] Kafka topic: "signal_detected" with priority routing
  - [ ] Priority routing (critical/high/medium/low)
  - [ ] Aggregation: combine similar signals to reduce noise

- [ ] **ContextProvider (On-Demand Service)** ⚠️ NOT IMPLEMENTED

  - [ ] ContextProvider interface definition (NOT FOUND in codebase)
  - [ ] Fetch top N relevant items (not all raw data - limit to 10-20 items)
  - [ ] Structure context by type (macro/news/social/onchain)
  - [ ] Caching for frequently requested contexts (Redis, TTL 5-10 min)
  - [ ] Time-decay scoring: recent items weighted higher

- [ ] **Agent Integration**

  - [ ] Update Research agents to consume structured context via ContextProvider
  - [ ] Add external context to agent prompts (optional blocks NOT FOUND in templates)
  - [ ] Wire real ContextProvider into tool registry
  - [ ] Enhance RegimeDetector: add sentiment features to ML model input
  - [ ] Benchmark: latency, cost, quality improvement vs Phase 4 baseline

- [ ] **Testing & Validation**
  - [ ] A/B test: Research agents with vs without external context
  - [ ] Measure: decision quality, confidence calibration, cost
  - [ ] RegimeDetector accuracy: with vs without sentiment features
  - [ ] Monitor: classification latency (<5 sec target), cost per item (<$0.01)

**Status:**
- **Completed:** Data collector skeletons (on-chain, sentiment workers exist)
- **Blocked:** Core Information Processing Layer components not started
- **Next:** Design & implement ContextProvider interface → InformationClassifier → SignalExtractor → ContextBuilder

### Phase 6: Memory System (Week 6) ⏳ PARTIAL

- [x] **Episodic Memory** (basic structure exists)

  - [x] Schema for trades exists in PostgreSQL (`migrations/postgres/`)
  - [x] Closed trades saved via position repository
  - [ ] TradeEpisode domain model with full context ⚠️ NOT IMPLEMENTED
  - [ ] Query interface for similarity search (pgvector exists but not used)
  - [ ] Context capture at entry/exit (only basic fields saved)

- [ ] **Semantic Memory** ⚠️ NOT IMPLEMENTED

  - [ ] Schema for ValidatedPattern
  - [ ] PerformanceCommittee writes patterns (agent exists but no pattern persistence)
  - [ ] Query interface for pattern lookup
  - [ ] Integration with `search_memory` tool

- [ ] **Working Memory** ⚠️ NOT IMPLEMENTED
  - [ ] In-memory store for active hypotheses
  - [ ] Expiry and invalidation logic
  - [ ] Agent read/write interface

**Status:**
- **Completed:** Basic trade persistence (positions, orders, PnL)
- **Blocked:** Rich episodic memory (TradeEpisode with full context), semantic memory (patterns), working memory
- **Existing Tools:** `save_memory` and `search_memory` tools exist in `internal/tools/memory/` but need rich memory schemas
- **Next:** Define TradeEpisode, ValidatedPattern, Hypothesis schemas → implement repositories → integrate with agents

### Phase 7: Documentation & Testing (Week 7) ⏳ IN PROGRESS

- [x] **Architecture Docs**

  - [x] This document (`docs/AGENT_ARCHITECTURE.md`)
  - [x] Agent-specific README files exist:
    - [x] `internal/agents/README.md`
    - [x] `internal/tools/README.md`
    - [x] `internal/workers/README.md`
    - [x] `pkg/templates/prompts/agents/README.md`
  - [ ] Workflow diagrams (need to create visual diagrams)
  - [ ] Event flow diagrams (need to create visual diagrams)

- [ ] **Testing** ⚠️ MINIMAL COVERAGE

  - [ ] Unit tests for all agents (very few exist)
  - [x] Some worker tests exist (`*_test.go` files in `internal/workers/`)
  - [x] Some schema tests exist (`internal/agents/schemas/*_test.go`)
  - [ ] Integration tests for workflows
  - [ ] Event-driven system tests (critical - need to test PositionGuardian flow)
  - [ ] End-to-end scenarios

- [x] **Monitoring & Observability** (infrastructure exists)
  - [x] AI usage tracking via AIUsageEvent (protobuf schema)
  - [x] Worker health monitoring (`internal/workers/registry.go`)
  - [x] Event publishing for analytics (`internal/events/worker_publisher.go`)
  - [x] ClickHouse for telemetry storage
  - [ ] Agent performance metrics dashboard
  - [ ] Event processing latency monitoring
  - [ ] LLM cost tracking per agent (event exists, needs aggregation)
  - [ ] Decision audit trails (reasoning_trace exists in schemas, needs UI)

**Status:**
- **Completed:** Basic documentation structure, observability infrastructure
- **Blocked:** Testing coverage is minimal, needs significant work
- **Next:** Create comprehensive test suite for event-driven PositionGuardian flow → add workflow integration tests → build monitoring dashboard

---

## 7. Cost & Performance Benchmarks

### Research Committee (Multi-Agent)

**Composition:** 4 analysts + 1 synthesizer

**Per Scan:**

- Time: 60-90 seconds (parallel + synthesis)
- Cost: $0.30-0.50
- Quality: High (diverse viewpoints, debate)

**Use Cases:**

- High-stakes decisions
- Complex market conditions
- When confidence is critical

### OpportunitySynthesizer (Single Agent)

**Per Scan:**

- Time: 15-30 seconds
- Cost: $0.05-0.10
- Quality: Good (tool orchestration)

**Use Cases:**

- Routine scanning
- High-frequency checks
- Low-stakes signals

### PortfolioManager (Per-Client)

**Per Decision:**

- Time: 20-40 seconds
- Cost: $0.05-0.08
- Tools: 3-5 calls (account_status, correlation, pre_trade_check)

### PositionGuardian (Event-Driven)

**Critical Events (algorithmic):**

- Time: <1 second
- Cost: $0 (no LLM)
- Example: Stop hit → immediate close

**Complex Events (LLM):**

- Time: 10-20 seconds
- Cost: $0.02-0.05
- Example: Thesis invalidation → agent evaluation

### PerformanceCommittee (Weekly)

**Per Review:**

- Time: 2-5 minutes
- Cost: $0.20-0.40
- Frequency: Weekly
- ROI: High (system improvement)

---

## 8. Migration Strategy

### Step 1: Parallel Running (Week 1)

- Deploy new agents alongside existing
- Route 10% of traffic to new architecture
- Compare: decisions, performance, cost
- Monitor: errors, latency, quality

### Step 2: Gradual Rollout (Week 2-3)

- Increase to 25% → 50% → 75%
- Monitor key metrics at each stage
- Rollback plan if issues detected

### Step 3: Full Migration (Week 4)

- Route 100% to new architecture
- Deprecate old agents
- Update documentation
- Train support team

### Step 4: Optimization (Week 5-6)

- Fine-tune prompts based on production data
- Optimize event processing
- Reduce latency and cost
- A/B test prompt variations

---

## 9. Success Metrics

### System Performance

- **Decision Latency**: <30 sec for routine, <90 sec for complex
- **Event Reaction Time**: <1 sec for critical, <30 sec for complex
- **Uptime**: 99.9% for position monitoring
- **Cost per User per Month**: <$50

### Agent Quality

- **Confidence Calibration**: Stated confidence ±5% of actual win rate
- **Win Rate**: ≥60% for published opportunities
- **Risk-Adjusted Returns**: Sharpe ratio ≥1.5
- **Pattern Validation**: ≥80% accuracy for validated patterns

### User Experience

- **Response Time**: <5 sec for Telegram commands
- **Transparency**: 100% of decisions include reasoning
- **Audit Trail**: 100% of trades logged with context
- **Customer Satisfaction**: ≥4.5/5 rating

---

## 10. Risks & Mitigations

### Risk 1: LLM Hallucination

**Mitigation:**

- Structured output schemas with validation
- Pre-trade reviewer as quality gate
- Reasoning trace for audit
- Human-in-the-loop for high-stakes decisions

### Risk 2: Event Processing Delay

**Mitigation:**

- Dedicated event processors per priority level
- Critical events bypass queue (algorithmic handling)
- WebSocket for real-time price feeds
- Monitoring and alerting on latency

### Risk 3: Multi-Agent Disagreement

**Mitigation:**

- HeadOfResearch has final say
- Explicit conflict resolution framework in prompt
- Fall back to conservative decision when uncertain
- Log all debates for review

### Risk 4: Cost Overrun

**Mitigation:**

- Fast-path for routine scanning
- Caching for repeated queries
- Budget limits per user
- Cost monitoring and alerts

### Risk 5: Agent Drift (Performance Degradation)

**Mitigation:**

- Weekly PerformanceCommittee review
- A/B testing of prompt variations
- Continuous monitoring of key metrics
- Automated rollback on quality drop

---

## 11. Next Steps

1. **Review & Approval**: Share this document with team for feedback
2. **Detailed Design**: Create per-agent detailed specs
3. **Prototyping**: Build ResearchCommittee prototype
4. **Testing**: Validate multi-agent collaboration
5. **Deployment**: Follow migration strategy
6. **Iteration**: Refine based on production data

---

## Appendix A: Agent Responsibilities Matrix

| Agent                  | Scope        | Frequency     | Decision Type              | Cost   | Criticality |
| ---------------------- | ------------ | ------------- | -------------------------- | ------ | ----------- |
| ResearchCommittee      | Global       | Continuous    | Opportunity Identification | High   | Medium      |
| OpportunitySynthesizer | Global       | High-Freq     | Quick Scan                 | Low    | Low         |
| RegimeInterpreter      | Global       | Hourly        | Market Classification      | Low    | High        |
| PortfolioManager       | Per-Client   | On-Demand     | Trade Approval             | Medium | High        |
| PortfolioArchitect     | Per-Client   | Onboarding    | Initial Allocation         | Medium | High        |
| PositionGuardian       | Per-Position | Real-Time     | Position Management        | Low    | Critical    |
| PreTradeReviewer       | Per-Trade    | Pre-Execution | Quality Gate               | Low    | High        |
| PerformanceCommittee   | Global       | Weekly        | Learning & Adaptation      | Medium | Medium      |

---

## Appendix B: Workflow Diagrams

### Opportunity Discovery → Execution Flow

```
1. ResearchCommittee (60-90s)
   ↓ publishes: opportunity_identified

2. PortfolioManager (per client, 20-40s)
   ↓ publishes: trade_approved

3. PreTradeReviewer (10-20s)
   ↓ publishes: trade_validated

4. ExecutionDesk (5-10s)
   ↓ publishes: position_opened

5. PositionGuardian (real-time)
   → monitors until closed
```

### Event-Driven Position Monitoring

```
WebSocket Price Feed
   ↓
Position Event Generator
   ↓
Event Router
   ├─ Critical → Immediate Action (algorithmic)
   ├─ High     → Agent Evaluation (LLM)
   └─ Low      → Batch Processing

Agent Decision
   ↓
Execution Service
   ↓
Client Notification
```

---

**Document Status**: Draft for Review  
**Next Review**: After team feedback  
**Owner**: Architecture Team
