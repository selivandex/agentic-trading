<!-- @format -->

# Phase 7 — Production Readiness & Stabilization (2-3 days)

## Objective

Harden system for production: comprehensive testing, error recovery, data consistency, deployment automation. Ensure system can run 24/7 without manual intervention.

## Why After Phase 6

Phase 1-6 build complete system with observability. Now make it bulletproof for production.

## Scope & Tasks

### 1. Integration Test Suite (Day 1)

Create `internal/testsupport/integration/` with end-to-end scenarios:

**Test 1: Complete Trading Flow**

```go
// Seed: user, exchange account, capital
// Action: trigger OpportunityFinder → workflow runs → order placed → position opened
// Assert: position in DB, order on exchange, PnL tracked
```

**Test 2: Risk Rejection**

```go
// Seed: user with 90% exposure
// Action: try to place new order
// Assert: order rejected by RiskEngine, event published
```

**Test 3: Circuit Breaker**

```go
// Seed: open position with -6% loss
// Action: PositionMonitor runs
// Assert: circuit breaker triggered, positions closed, user notified
```

**Test 4: Account Deactivation**

```go
// Seed: account with invalid API key
// Action: OrderExecutor tries to place order
// Assert: order rejected, account marked inactive, user notified
```

**Test 5: Worker Recovery**

```go
// Seed: pending order
// Action: stop OrderExecutor → restart after 1 min
// Assert: order eventually executed (no data loss)
```

Run with testcontainers (Postgres, Redis, ClickHouse) + mock Kafka.

### 2. Error Recovery & Idempotency (Day 2)

**Idempotent order placement:**

- Use `client_order_id` (UUID) to prevent duplicate orders
- If OrderExecutor crashes mid-execution → on restart, check exchange for existing order
- Update DB with exchange order ID

**Kafka consumer idempotency:**

- Store `last_processed_offset` in Postgres
- On restart, resume from last offset (not reprocess same messages)
- Use transaction: process message + update offset atomically

**Worker crash recovery:**

- Workers should be stateless (use DB/Redis for state)
- On crash → scheduler restarts worker
- Verify no data loss (pending orders still processed)

**Database transactions:**

- All multi-step operations in single transaction
- Example: create strategy + allocations + publish event → all or nothing

### 3. Data Consistency Checks (Day 2)

**Add validation worker `data_integrity_checker.go`:**

Every hour, verify:

- Positions match orders (no orphaned positions)
- Order statuses consistent (filled orders have positions)
- PnL calculations add up (sum of position PnL = strategy equity change)
- No stuck orders (pending >10 minutes → investigate)

**On inconsistency:**

- Log error with full context
- Alert admin via Telegram
- Attempt auto-fix (safe cases only)

### 4. Deployment Automation (Day 3)

**Docker Compose Production:**

- Update `docker-compose.yml` for production settings
- Separate services: app, workers, consumers
- Health checks for each container
- Resource limits (CPU, memory)

**Migration Strategy:**

- Postgres migrations with `migrate` tool
- ClickHouse migrations separate
- Zero-downtime deployment (blue-green or rolling)

**Environment Management:**

- `.env.production` with all required vars
- Secret management (use vault or encrypted env)
- Config validation on startup (fail fast if missing vars)

**Backup & Recovery:**

- Postgres backups (pg_dump) daily
- ClickHouse backups weekly (raw data less critical)
- Redis persistence (AOF enabled)
- Strategy: restore from backup + replay Kafka (if needed)

### 5. Load Testing (Day 3, optional)

Simulate production load:

**Scenario 1: High Order Volume**

- 100 orders per minute
- Verify OrderExecutor keeps up (lag <10 seconds)
- Check database connection pool (no exhaustion)

**Scenario 2: Many Concurrent Users**

- 50 users with active strategies
- OpportunityConsumer triggers workflows for each
- Verify system doesn't crash (memory, CPU usage)

**Scenario 3: Exchange Latency Spike**

- Mock exchange with 5-second delays
- Verify timeouts and retries work
- Check no goroutine leaks

Use `k6` or custom Go load test.

### 6. Final Testing Checklist (Day 3)

**Manual smoke tests on testnet:**

- [ ] New user: `/invest 1000` → portfolio created → positions opened
- [ ] Position monitoring: PnL updates every minute
- [ ] Risk limits: try to exceed exposure → blocked
- [ ] Circuit breaker: simulate loss → trading paused
- [ ] Account issues: revoke API key → detected and deactivated
- [ ] Worker restart: stop worker → restart → resumes correctly
- [ ] Telegram commands: all work (`/portfolio`, `/pause`, `/resume`, `/status`)

**Production readiness checklist:**

- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] No TODOs in critical path code
- [ ] All environment variables documented
- [ ] Health checks respond correctly
- [ ] Metrics exported and dashboards work
- [ ] Logs structured and parseable
- [ ] Alerting tested (trigger alert, verify received)
- [ ] Backups configured and tested (restore from backup)

## Deliverables

- ✅ Comprehensive integration test suite
- ✅ Idempotent operations (orders, consumers)
- ✅ Data consistency validation
- ✅ Production deployment automation
- ✅ Load testing (optional but recommended)
- ✅ Smoke test suite passed on testnet
- ✅ Production readiness checklist completed

## Exit Criteria

1. All integration tests pass without flakiness
2. Kill worker mid-execution → restart → no data loss
3. Kafka consumer restart → resumes from correct offset (no duplicates)
4. Load test: 100 orders/min → system stable (CPU <70%, memory <80%)
5. Deployment: deploy to testnet → health check green → no errors in logs
6. Manual smoke test: complete trading flow works end-to-end

## Dependencies

Requires all previous phases (1-6) to be complete.

---

**Success Metric:** System runs 24/7 on testnet without crashes or data loss for 48 hours
