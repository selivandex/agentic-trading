<!-- @format -->

# Phase 2 — Workers Activation & Position Tracking (2 days)

## Objective

Activate existing workers to enable continuous portfolio review by agents. **Key:** PortfolioReviewWorker wakes up every N minutes, runs PortfolioManager agent for each user strategy, which calls expert agents (TechnicalAnalyzer, SMCAnalyzer, etc.) as tools to make decisions about each asset.

## Why After Phase 1

Phase 1 enables order execution. Now we need continuous monitoring of those positions, risk limits, and market opportunities.

## Scope & Tasks

### 1. Verify Workers Run (Day 1, 2 hours)

Workers are implemented in `internal/workers/`:

- `analysis/opportunity_finder.go` - market research every 15 min
- `trading/position_monitor.go` - PnL updates every 1 min
- `trading/risk_monitor.go` - risk checks every 2 min
- `trading/pnl_calculator.go` - equity calc every 5 min

**Action:**

- Verify all workers register in `bootstrap/workers.go`
- Check scheduler starts them (already uncommented in Phase 1)
- Add structured logging for each iteration

### 2. Position Updates from WebSocket (Day 1-2)

Update `internal/consumers/websocket_consumer.go`:

Currently has placeholder:

```go
// TODO: Implement order update handling
```

**Implement:**

- Parse `OrderUpdateEvent` from Binance/Bybit
- Call `position.Repository.UpdateFromOrderFill()`
- Publish `position.updated` event to Kafka
- Handle partial fills, cancellations

**New method in position.Repository:**

```go
UpdateFromOrderFill(ctx, orderID, filledQty, avgPrice, fee) error
```

### 3. PnL Calculation Fix (Day 2)

In `internal/workers/trading/pnl_calculator.go`, fix TODOs:

```go
// Line 196-201: Currently returns 0.0 for daily_pnl_percent, win_rate, etc.
```

Implement actual calculations:

- `daily_pnl_percent = (current_equity - start_equity) / start_equity * 100`
- `win_rate = winning_trades / total_trades`
- Use ClickHouse for historical data

### 4. Tests (Day 2)

**Worker tests (with mocks):**

- `position_monitor_test.go` - mock exchange client, verify PnL updates
- `opportunity_finder_test.go` - mock ADK workflow, verify Kafka publish

**Integration test:**

- Seed OHLCV data → OpportunityFinder runs → verify opportunity event published
- Seed open position → PositionMonitor runs → verify PnL updated

**WebSocket consumer test:**

- Mock Kafka message (order update) → verify position updated

## Deliverables

- ✅ `agents` table created (migration 009) - stores prompts, config, tools
- ✅ Agents seeded from templates (`seeds/agents.go` loads from `pkg/templates/`)
- ✅ PortfolioReviewWorker running every 30 minutes
- ✅ PortfolioManager agent calls expert agents as tools
- ✅ Expert agents (TechnicalAnalyzer, SMCAnalyzer, CorrelationAnalyzer) registered as tools
- ✅ `portfolio_decisions` table stores BOTH decisions AND expert analyses (unified)
- ✅ `agent_id` references `agents` table (FK)
- ✅ Hierarchy via `parent_decision_id`: decision → expert analyses
- ✅ All agent reasoning saved to single table (audit trail)
- ✅ Existing workers running (PnL, risk, opportunity finder)
- ✅ Tests: workflow with expert agent tools + hierarchical storage
- ✅ Structured logs (agent identifier, symbol, decision, analysis)

## Exit Criteria

1. `agents` table populated with 6 agents (portfolio_architect, portfolio_manager, technical_analyzer, smc_analyzer, correlation_analyzer, opportunity_synthesizer)
2. Agent prompts loaded from `pkg/templates/prompts/agents/*.tmpl` files
3. Can update prompt in DB: `UPDATE agents SET system_prompt = '...' WHERE identifier = 'portfolio_manager'` → agent uses new prompt immediately
4. PortfolioReviewWorker runs every 30 minutes for all active strategies
5. Logs show: "PortfolioManager calling TechnicalAnalyzer for BTC/USDT"
6. `portfolio_decisions` table contains both decisions (parent_decision_id=NULL) and analyses (parent_decision_id=decision.id)
7. Each PortfolioManager decision has 2-3 child records (expert analyses)
8. Query: `SELECT * FROM portfolio_decisions pd JOIN agents a ON pd.agent_id = a.id WHERE parent_decision_id = 'decision-123'` returns expert analyses with agent names
9. Existing workers still running (PnL updates, risk checks)
10. Manual test: create strategy → wait 30 min → see 1 decision + 3 analyses in DB

## Dependencies

Requires Phase 1 (order execution) to have real positions to monitor.

---

**Success Metric:** Position opens → PnL updates automatically → risk limits checked → opportunity signals flow
