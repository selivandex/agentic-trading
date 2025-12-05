<!-- @format -->

# Phase 4 — Risk Management & Circuit Breakers (1-2 days)

## Objective

Harden risk engine with real enforcement: position limits, drawdown protection, circuit breakers. Prevent catastrophic losses in production.

## Why After Phase 3

Phase 3 enables users to invest and trade. Now we need safeguards before scaling to real money.

## Scope & Tasks

### 1. Risk Engine Enforcement (Day 1)

Currently `internal/services/risk/risk_engine.go` exists but needs verification:

**Test all checks work:**

- `CanTrade()` - called before every order
- Max position size per symbol
- Max total exposure per user
- Max daily loss limit
- Max drawdown from peak

**Add missing:**

- Correlation checks (don't open 5 correlated positions)
- Leverage limits for futures
- Minimum capital requirements

**Integration points:**

- ExecutionService calls `RiskEngine.CanTrade()` before placing order
- If rejected → log reason, return error to agent
- Publish `risk.trade_rejected` event to Kafka

### 2. Circuit Breaker Implementation (Day 1-2)

Three levels of circuit breakers:

**Level 1 - Soft Pause (user-triggered):**

- User: `/pause` → set `user.settings.circuit_breaker_on = true`
- Effect: no new positions, existing remain open
- Resume: `/resume`

**Level 2 - Drawdown Protection (automatic):**

- Trigger: daily loss > 5% OR total drawdown > 15%
- Effect: close all positions, pause for 24h
- Notify user via Telegram + email (if configured)
- Log to ClickHouse for analysis

**Level 3 - Emergency (admin-triggered):**

- Trigger: system anomaly detected (too many failed orders, API issues)
- Effect: global pause for all users
- Admin notification via Telegram + PagerDuty

**Implementation:**

- Update `internal/services/risk/circuit_breaker.go`
- Store state in Redis (fast lookup)
- Persist events in Postgres (audit trail)

### 3. Position Size Validation (Day 2)

In `internal/services/risk/position_sizer.go`:

Currently has TODOs. Implement Kelly Criterion + safety limits:

```go
maxPositionSize := min(
    strategy.AllocatedCapital * 0.1,  // Max 10% per position
    kellyFraction(winRate, avgWin, avgLoss) * capital,
    account.AvailableBalance * 0.15,   // Max 15% of available
)
```

Add to ExecutionService before order placement.

### 4. Tests (Day 2)

**Risk engine tests (unit with mocks):**

- `CanTrade()` returns false when exposure > limit
- Circuit breaker triggers on 5% daily loss
- Position size calculation with various scenarios

**Integration tests:**

- Place order that exceeds limit → verify rejected
- Trigger drawdown → verify all positions closed
- User pauses → verify no new orders execute

**Stress test (optional):**

- Simulate 100 orders in 1 second → verify rate limits work

## Deliverables

- ✅ Risk engine blocks orders exceeding limits
- ✅ Circuit breakers work (user-triggered, auto drawdown, emergency)
- ✅ Position sizing with Kelly Criterion
- ✅ All risk checks logged to ClickHouse
- ✅ Tests: unit + integration for risk scenarios
- ✅ Metrics: trades_rejected_total, circuit_breaker_triggered_total

## Exit Criteria

1. Place order exceeding exposure → rejected with clear reason
2. Simulate 6% daily loss → circuit breaker triggers, positions close
3. User `/pause` → no new orders execute (test with OpportunityConsumer)
4. Position size respects Kelly + hard limits (max 10% per position)
5. All risk events visible in logs and metrics

## Dependencies

Requires Phase 1 (execution) and Phase 2 (monitoring) for position data.

---

**Success Metric:** System safely rejects risky trades and protects capital from drawdowns
