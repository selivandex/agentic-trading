<!-- @format -->

# Phase 3 — Telegram MVP Flow (2-3 days)

## Objective

Complete end-to-end Telegram user journey: `/invest` → portfolio creation → trades execute → user sees status. Add essential commands for portfolio management.

## Why After Phase 2

Phase 1-2 enabled execution + monitoring backend. Now expose it to users through Telegram (no UI needed for MVP).

## Scope & Tasks

### 1. Test & Fix /invest Flow (Day 1)

Flow already implemented in `internal/adapters/telegram/invest_menu.go`:

**Test end-to-end:**

1. User: `/invest` (no arguments)
2. Bot: Select exchange account (inline buttons)
3. User: Selects Binance account
4. Bot: Select market type (spot/futures)
5. User: Selects Spot
6. Bot: Select risk profile (conservative/moderate/aggressive)
7. User: Selects Moderate
8. Bot: Enter investment amount
9. User: Types "1000"
10. Bot: Shows confirmation
11. Onboarding service creates strategy
12. Portfolio initialization workflow runs
13. Orders execute via OrderExecutor (Phase 1)
14. User receives notifications

**Fix issues found:**

- Exchange account validation (must be active)
- Capital limits (min/max)
- Error messages (user-friendly)
- Timeout handling (workflow can take 2-3 minutes)

### 2. Agent Reasoning Storage - Per-Asset Decisions (Day 1-2)

**Critical for user trust:** Users need to understand WHY agents make decisions about EACH asset separately.

**New structured storage:** `portfolio_decisions` table (migration 010 already created)

Instead of bulk JSONB, store each decision as a separate record:

```sql
-- Each row = one decision about one asset
INSERT INTO portfolio_decisions (
    strategy_id, user_id, agent_id,
    symbol, action, decision_reason,
    market_regime, position_after, allocation_after,
    confidence_level, decision_timestamp
) VALUES (
    'abc-123', 'user-456', 'PortfolioArchitect',
    'BTC/USDT', 'buy', 'Market leader, defensive anchor in moderate risk portfolio. Strong support at $43k.',
    'bull_trending', 0.015, 0.35,
    0.90, NOW()
);
```

**Implement domain layer:**

```go
// internal/domain/portfolio_decision/entity.go
type PortfolioDecision struct {
    ID                uuid.UUID
    StrategyID        uuid.UUID
    UserID            uuid.UUID
    AgentID           string // "PortfolioArchitect", "PortfolioManager"
    Symbol            string // "BTC/USDT"
    Action            DecisionAction // buy, sell, hold, rebalance, close
    DecisionReason    string // Human-readable
    MarketRegime      string
    PositionBefore    decimal.Decimal
    PositionAfter     decimal.Decimal
    AllocationBefore  decimal.Decimal // 0.35 = 35%
    AllocationAfter   decimal.Decimal
    ConfidenceLevel   decimal.Decimal // 0-1
    OrderID           *uuid.UUID
    DecisionTimestamp time.Time
}

// internal/domain/portfolio_decision/repository.go
type Repository interface {
    Create(ctx, *PortfolioDecision) error
    GetByStrategy(ctx, strategyID, limit) ([]*PortfolioDecision, error)
    GetBySymbol(ctx, strategyID, symbol, limit) ([]*PortfolioDecision, error)
    GetLatest(ctx, strategyID, symbol) (*PortfolioDecision, error)
}
```

**Extract decisions from workflow:**

```go
// internal/services/onboarding/service.go
func (s *Service) StartOnboarding(...) {
    for event := range runnerInstance.Run(...) {
        // Parse reasoning text for asset-specific decisions
        decisions := extractAssetDecisions(event, strategy.ID)

        for _, decision := range decisions {
            portfolioDecisionRepo.Create(ctx, decision)
        }
    }
}

// Extract per-asset decisions from agent output
func extractAssetDecisions(event *session.Event, strategyID uuid.UUID) []*PortfolioDecision {
    var decisions []*PortfolioDecision

    // Agent output contains structured decisions per asset
    // Example: "BTC (35%): Market leader, strong support at $43k"
    //          "ETH (25%): L2 growth, moderate risk"

    text := getReasoningText(event)

    // Parse asset blocks (regex or simple string parsing)
    assetBlocks := parseAssetBlocks(text)

    for _, block := range assetBlocks {
        decision := &PortfolioDecision{
            StrategyID:       strategyID,
            AgentID:          "PortfolioArchitect",
            Symbol:           block.Symbol,      // "BTC/USDT"
            Action:           ActionBuy,         // or ActionHold
            DecisionReason:   block.Reasoning,   // "Market leader..."
            AllocationAfter:  block.Allocation,  // 0.35
            ConfidenceLevel:  decimal.NewFromFloat(0.9),
            DecisionTimestamp: time.Now(),
        }
        decisions = append(decisions, decision)
    }

    return decisions
}
```

### 3. Essential Commands (Day 2)

Implement in `internal/adapters/telegram/`:

**`/portfolio`** - Show current positions with per-asset reasoning

```
📊 Your Portfolio

💼 Strategy: Portfolio 2024-12-04
💰 Capital: $1,000.00
📈 PnL: +$45.23 (+4.52%)

Positions:

🪙 BTC/USDT: 0.015 BTC ($630) +2.3%
💭 Holding (35% allocation)
> Strong support at $43k, bull market structure intact.
> Next review at $48k resistance or $41k support break.
Last decision: 2 hours ago

🪙 ETH/USDT: 0.3 ETH ($370) +6.8%
💭 Holding (25% allocation)
> L2 ecosystem growth continues, momentum indicators bullish.
> Watching $2,500 breakout level.
Last decision: 2 hours ago

💰 Cash Reserve: $0 (0%)
💭 Fully invested per moderate risk profile
> Will rebuild reserve on next rebalance or profit taking.

Risk: 🟢 Safe (60% exposure, 40% in stable majors)
Next rebalance: in 10 days
```

**`/status`** - Strategy status

```
⚙️ Strategy Status

Name: Portfolio 2024-12-04
Status: 🟢 Active
Capital: $1,000
Risk: Moderate
Trades: 8 executed

Last activity: 2 min ago
Next rebalance: ~10 min
```

**`/pause`** - Pause trading (set circuit breaker)

```
⏸️ Trading Paused

No new positions will be opened.
Existing positions remain active.

Use /resume to continue trading.
```

**`/resume`** - Resume trading

**`/killswitch`** - Emergency stop (close all positions)

**`/explain <strategy_id> [symbol]`** - Show decision history timeline

Without symbol - all decisions across all assets:

```
💭 Portfolio Decision History

📅 Strategy: Portfolio 2024-12-04
🎯 Risk Profile: Moderate
📊 Total Decisions: 12

Recent Decisions (last 7 days):

🪙 BTC/USDT
├─ 2 hours ago: HOLD (35%)
│  💭 Strong support at $43k, bull structure intact
├─ 1 day ago: HOLD (35%)
│  💭 Consolidating above key level, no action needed
└─ 3 days ago: BUY (35%)
   💭 Initial allocation - market leader, defensive anchor
   📈 Entry: $42,850 | Order: #abc123

🪙 ETH/USDT
├─ 2 hours ago: HOLD (25%)
│  💭 L2 momentum continues, watching $2.5k breakout
├─ 1 day ago: HOLD (25%)
│  💭 Trending with strength, position sized correctly
└─ 3 days ago: BUY (25%)
   💭 Initial allocation - ecosystem growth story
   📈 Entry: $2,280 | Order: #def456

🪙 SOL/USDT
├─ 1 day ago: SELL (0%)
│  💭 Took profit at 15% gain, reducing small-cap exposure
│  📉 Exit: $108.50 | Gain: +15.2%
└─ 3 days ago: BUY (20%)
   💭 Initial allocation - high beta bull market play
   📈 Entry: $94.20 | Order: #ghi789
```

With symbol - detailed history for specific asset:

```
💭 BTC/USDT Decision History

Current Position: 0.015 BTC ($630)
Entry Price: $42,850
Current Price: $43,200
PnL: +$5.25 (+0.8%)

📊 Decision Timeline:

┌─ 2 hours ago: HOLD
│  Action: Keep 35% allocation (0.015 BTC)
│  Reason: Strong support at $43k, bull structure intact.
│          No rebalancing needed - within ±5% allocation band.
│  Market: Bull trending, RSI 62
│  Confidence: 85%
│  Agent: PortfolioManager
│
├─ 1 day ago: HOLD
│  Action: Keep 35% allocation
│  Reason: Consolidating above key level. Watching $44k
│          breakout or $41k support test.
│  Market: Bull trending
│  Confidence: 80%
│  Agent: PortfolioManager
│
└─ 3 days ago: BUY
   Action: Open 35% allocation (0.015 BTC @ $42,850)
   Reason: Initial portfolio allocation. Market leader,
           defensive anchor in moderate risk portfolio.
           Strong support at $43k level.
   Market: Bull trending
   Confidence: 90%
   Agent: PortfolioArchitect
   Order: #abc123 (filled)
```

### 4. Notification Templates (Day 2)

Already exist in `pkg/telegram/templates/`, verify:

- ✅ `investment_accepted.tmpl`
- ✅ `portfolio_created.tmpl`
- ✅ `trade_executed.tmpl`
- ✅ `position_closed.tmpl`
- ✅ `risk_alert.tmpl`

Add if missing:

- `portfolio_status.tmpl`
- `strategy_paused.tmpl`

### 5. GraphQL Resolvers (Day 3)

**Strategy mutations** in `internal/api/graphql/resolvers/strategy.resolvers.go`:

```go
// Currently return "not implemented"
UpdateStrategy(ctx, id, input) - update allocated capital, risk profile
PauseStrategy(ctx, id) - set status=paused, trigger circuit breaker
ResumeStrategy(ctx, id) - set status=active, clear circuit breaker
CloseStrategy(ctx, id) - close all positions, set status=closed
```

**New query for decisions** in `internal/api/graphql/schema/portfolio_decision.graphql`:

```graphql
type PortfolioDecision {
  id: ID!
  strategyId: ID!
  symbol: String!
  action: DecisionAction!
  decisionReason: String!
  marketRegime: String
  positionBefore: Float
  positionAfter: Float
  allocationBefore: Float # 0.35 = 35%
  allocationAfter: Float
  confidenceLevel: Float # 0-1
  orderId: ID
  decisionTimestamp: Time!
}

enum DecisionAction {
  BUY
  SELL
  HOLD
  REBALANCE
  CLOSE
}

extend type Query {
  # Get all decisions for strategy, optionally filtered by symbol
  portfolioDecisions(
    strategyId: ID!
    symbol: String
    limit: Int = 50
  ): [PortfolioDecision!]!

  # Get latest decision for specific asset
  latestDecision(strategyId: ID!, symbol: String!): PortfolioDecision
}
```

Connect to `internal/services/portfolio_decision/service.go` methods.

### 6. Tests (Day 3)

**Telegram handler tests (unit with mocks):**

- `/invest` flow with mock onboarding service
- `/portfolio` with mock position repository
- `/pause` with mock strategy service

**GraphQL resolver tests (integration):**

- `pauseStrategy` mutation → verify DB status changed
- `resumeStrategy` → verify circuit breaker cleared

**E2E smoke test (manual for now):**

- `/invest 100` → wait 3 min → `/portfolio` shows positions
- `/pause` → verify no new trades
- `/resume` → verify trading resumes

## Deliverables

- ✅ Complete `/invest` flow tested and working
- ✅ `agents` table with prompts (migration 009, seeded from templates)
- ✅ `portfolio_decisions` table created (migration 010, references agents)
- ✅ PortfolioDecision domain entity & repository
- ✅ Agent reasoning extraction per asset (not bulk JSONB)
- ✅ Can update agent prompts via DB without deployment
- ✅ Essential commands: /portfolio, /status, /pause, /resume, /killswitch, /explain
- ✅ GraphQL queries: `portfolioDecisions(strategyID, symbol?)`
- ✅ GraphQL resolvers implemented (pause/resume/close strategy)
- ✅ User-friendly error messages and timeouts
- ✅ Tests: decision repository + telegram commands + GraphQL
- ✅ Notification templates for all user actions

## Exit Criteria

1. User can `/invest` → complete flow → see positions in `/portfolio`
2. `/portfolio` shows per-asset reasoning inline (BTC: why holding, ETH: why holding)
3. `/explain <strategy_id>` shows timeline of all decisions across all assets
4. `/explain <strategy_id> BTC` shows detailed decision history for BTC only
5. Each decision stored as separate record in `portfolio_decisions` table
6. Can query decision history via GraphQL: `portfolioDecisions(strategyID: "...", symbol: "BTC/USDT")`
7. `/pause` stops new trades, `/resume` restarts them
8. All GraphQL mutations work (test via GraphQL playground)
9. Error handling: invalid amount → clear message, retry possible
10. Timeout: workflow >5min → user gets "processing" update
11. Reasoning is human-readable (not technical logs or raw JSON)

## Dependencies

Requires Phase 1 (execution) and Phase 2 (monitoring) for complete flow.

---

**Success Metric:** New user can invest via Telegram and see their portfolio without touching UI or DB
