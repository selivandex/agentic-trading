<!-- @format -->

# MVP Development Plan — Agentic Trading System

## Executive Summary

**Goal:** Working AI trading system where users invest via Telegram and agents autonomously trade on their behalf.

**Timeline:** 10-14 days (8 phases)

**Current Status:** 70% complete - core infrastructure exists but critical execution gaps remain.

---

## ✅ Phase 1 Complete!

Orders now execute on Binance. OrderExecutor worker polls pending orders and sends to exchange.

## Current Status: Starting Phase 2

### What's Working:
- ✅ `/invest` flow (Telegram interactive menu)
- ✅ PortfolioArchitect creates initial portfolio
- ✅ Orders execute on Binance testnet
- ✅ Positions tracked in DB

### What's Next (Phase 2):
- 🔨 PortfolioReviewWorker - reviews portfolio every 30 min
- 🔨 Expert agents as tools (TechnicalAnalyzer, SMCAnalyzer)
- 🔨 Save decisions + expert analyses to DB

### 🚨 3. ExecutionService Missing Account Resolution (P1)

**Problem:** `ExchangeAccountID: uuid.Nil` hardcoded.

**Impact:** Orders created with invalid account.

**Fix:** Phase 1 - Resolve from user + exchange preference.

---

## Phase Breakdown

### **Phase 1: Order Execution Bridge (2-3 days)** ⚡ CRITICAL

**Deliverables:**

- OrderExecutor worker polls pending orders → sends to exchange
- Workers uncommented and running
- ExecutionService resolves exchange accounts
- Tests: integration + unit

**Exit Criteria:**

- Manual order in DB → appears on Binance testnet within 10 seconds
- All 4 workers running (logs show iterations)

**Estimated Effort:** 2-3 dev days

---

### **Phase 2: Workers Activation & Position Tracking (2 days)**

**Deliverables:**

- OpportunityFinder publishes market signals
- PositionMonitor updates PnL from WebSocket
- WebSocket consumer handles order fills
- PnL calculations fixed (no more 0.0 placeholders)

**Exit Criteria:**

- OpportunityFinder publishes ≥1 signal per hour
- Position PnL updates within 1 minute of fill
- WebSocket order updates flow to positions

**Estimated Effort:** 2 dev days

---

### **Phase 3: Telegram MVP Flow (2-3 days)**

**Deliverables:**

- `/invest` flow tested end-to-end
- Commands: `/portfolio`, `/status`, `/pause`, `/resume`, `/killswitch`
- GraphQL resolvers: `pauseStrategy`, `resumeStrategy`, `closeStrategy`
- User-friendly error messages

**Exit Criteria:**

- New user: `/invest 1000` → positions appear in `/portfolio`
- `/pause` stops new trades, `/resume` restarts
- Workflow timeout → user gets progress update

**Estimated Effort:** 2-3 dev days

---

### **Phase 4: Risk Management & Circuit Breakers (1-2 days)**

**Deliverables:**

- RiskEngine blocks orders exceeding limits
- Circuit breakers: user-triggered, auto drawdown, emergency
- Position sizing with Kelly Criterion
- All risk events logged to ClickHouse

**Exit Criteria:**

- Order exceeding exposure → rejected with reason
- Simulate 6% loss → circuit breaker triggers
- Position size respects max 10% per trade

**Estimated Effort:** 1-2 dev days

---

### **Phase 5: Exchange Account Lifecycle (1-2 days)**

**Deliverables:**

- Account validation on creation (test API keys)
- Listen key refresh (prevent disconnects)
- Account health monitoring
- Error handling for all exchange failures

**Exit Criteria:**

- Invalid API key → validation fails immediately
- Listen key refreshes every 30 minutes
- Bad account → deactivated within 5 minutes

**Estimated Effort:** 1-2 dev days

---

### **Phase 6: Observability & Alerting (1 day)**

**Deliverables:**

- Prometheus metrics (orders, positions, workers, risk)
- Enhanced `/health` endpoint
- Telegram admin alerts (critical + warning)
- Basic Grafana dashboards (optional)

**Exit Criteria:**

- Circuit breaker → admin alert within 30 seconds
- Health check shows all component statuses
- Metrics scrape successfully

**Estimated Effort:** 1 dev day

---

### **Phase 7: Production Readiness (2-3 days)**

**Deliverables:**

- Integration test suite (5+ end-to-end scenarios)
- Idempotent operations (orders, Kafka consumers)
- Data consistency validation
- Deployment automation (Docker Compose)
- Load testing (optional)

**Exit Criteria:**

- All integration tests pass
- Worker crash → restart → no data loss
- Load test: 100 orders/min → system stable
- 48 hours on testnet without crashes

**Estimated Effort:** 2-3 dev days

---

### **Phase 8: Production Launch (1-2 days)**

**Deliverables:**

- Production infrastructure deployed
- Phased rollout: internal → limited → open beta
- 24/7 monitoring with alerts
- Incident response runbook
- User support channel

**Exit Criteria:**

- Internal beta: 24 hours with no critical issues
- Limited beta: 5+ users trading successfully
- Open beta: >95% order success rate
- System uptime >99.5%

**Estimated Effort:** 1-2 dev days

---

## Timeline Summary

| Phase   | Duration   | Dev Days | Type      |
| ------- | ---------- | -------- | --------- |
| Phase 1 | Days 1-3   | 2-3      | Critical  |
| Phase 2 | Days 4-5   | 2        | Critical  |
| Phase 3 | Days 6-8   | 2-3      | Critical  |
| Phase 4 | Days 9-10  | 1-2      | Important |
| Phase 5 | Days 11-12 | 1-2      | Important |
| Phase 6 | Day 13     | 1        | Important |
| Phase 7 | Days 14-16 | 2-3      | Critical  |
| Phase 8 | Days 17-18 | 1-2      | Launch    |

**Total:** 10-14 working days (2-3 weeks calendar time)

---

## Key Architectural Decisions

### ✅ What's Working

1. **Clean Architecture** - layers properly separated (domain, services, adapters)
2. **ADK Agents** - workflows implemented (MarketResearch, PortfolioInit, PersonalTrading)
3. **Exchange Adapters** - Binance/Bybit/OKX clients with `PlaceOrder()` ready
4. **Telegram Bot** - `/invest` flow fully implemented
5. **WebSocket Streams** - market data + user data connected
6. **Kafka Infrastructure** - consumers for opportunities, risk, notifications
7. **Risk Engine** - limits and circuit breakers defined

### 🚨 What's Broken

1. **Order Execution Gap** - orders never sent to exchanges (Phase 1 fixes)
2. **Workers Disabled** - market research and monitoring not running (Phase 1 fixes)
3. **Missing Account Resolution** - can't identify which exchange account to use (Phase 1 fixes)
4. **Incomplete GraphQL** - pause/resume/close resolvers return "not implemented" (Phase 3 fixes)

### 🎯 What's NOT in MVP

- ❌ Web UI (Next.js) - Telegram is sufficient
- ❌ ML Regime Detection - optimization, not core functionality
- ❌ Advanced Memory System - basic memory works
- ❌ Multiple Exchanges - start with Binance testnet
- ❌ Social Features - focus on core trading

---

## Testing Strategy

### Repository Tests (Integration)

**Use testcontainers:**

- Real Postgres, Redis, ClickHouse
- Seed data, call methods, assert DB state
- No mocks for DB layer

**Example:**

```go
// internal/repository/postgres/order_test.go
func TestOrderRepository_GetPending(t *testing.T) {
    db := testsupport.NewTestPostgres(t)
    repo := postgres.NewOrderRepository(db)

    // Seed pending orders
    // Call GetPending()
    // Assert correct orders returned
}
```

### Service Tests (Unit)

**Mock repositories:**

- Use gomock or manual mocks
- Test business logic without DB
- Fast execution (<100ms per test)

**Example:**

```go
// internal/services/execution/service_test.go
func TestExecutionService_Execute(t *testing.T) {
    mockOrderService := mocks.NewMockOrderService()
    svc := execution.NewExecutionService(mockOrderService)

    // Test execution logic
    // Verify order created with correct params
}
```

### Worker Tests (Unit + Integration)

**Unit:** Mock all dependencies (repos, exchanges, ADK)
**Integration:** Use testcontainers + mock exchange API

### E2E Tests (Integration)

**Full stack:**

- Real DB (testcontainers)
- Mock Kafka
- Mock exchange API
- Test complete user flows

**Example scenarios:**

- `/invest 1000` → orders execute → positions created
- Circuit breaker triggers → trading paused
- Invalid account → order rejected

---

## Code Quality Requirements

### ✅ Linters (Must Pass)

```bash
make lint
```

**Tools:**

- `golangci-lint` - comprehensive Go linter
- `gofmt` - code formatting
- `govet` - suspicious constructs

**No exceptions:** Fix all linter errors before commit.

### ✅ Test Coverage

**Minimum requirements:**

- Domain layer: 100% (pure business logic, easy to test)
- Services: 80% (mock repos)
- Repositories: 80% (integration tests)
- Workers: 70% (critical paths)
- Adapters: 60% (external dependencies, harder to test)

**Run coverage:**

```bash
make test-coverage
```

### ✅ Clean Architecture Violations

**Forbidden:**

- Domain importing infrastructure (e.g., `domain/` → `repository/postgres`)
- Services importing API layer (`services/` → `api/`)
- Circular dependencies
- Global state or singletons (except logger)

**Verify:**

```bash
go mod graph | grep prometheus/internal/domain | grep postgres
# Should return nothing
```

---

## Risk Mitigation

### High Risk Areas

**1. Order Execution Reliability**

- Mitigation: Idempotent order placement with `client_order_id`
- Mitigation: Retry logic with exponential backoff
- Mitigation: Dead letter queue for permanently failed orders

**2. WebSocket Disconnections**

- Mitigation: Auto-reconnect with exponential backoff
- Mitigation: Listen key refresh every 30 minutes
- Mitigation: Fallback to REST API if WebSocket fails

**3. Race Conditions**

- Mitigation: Database transactions for multi-step operations
- Mitigation: Redis locks for distributed state
- Mitigation: Optimistic locking for concurrent updates

**4. LLM API Failures**

- Mitigation: Retry with different provider (OpenAI → Claude fallback)
- Mitigation: Circuit breaker for AI providers
- Mitigation: Cache common responses

**5. Exchange API Rate Limits**

- Mitigation: Respect rate limits in adapter clients
- Mitigation: Exponential backoff on 429 errors
- Mitigation: Distribute requests across accounts (if multiple)

---

## Success Metrics

### MVP Launch (Phase 8 Complete)

**Technical:**

- ✅ Order success rate >95%
- ✅ System uptime >99.5%
- ✅ Worker crash recovery <30 seconds
- ✅ Order execution latency <10 seconds (pending → exchange)
- ✅ Zero data loss (orders, positions, balances)

**Business:**

- ✅ 10+ users trading real money
- ✅ 100+ orders executed
- ✅ Average user PnL >0% (or break-even acceptable for beta)
- ✅ No P0 incidents in first week
- ✅ User feedback positive (no major complaints)

**Operational:**

- ✅ All alerts working (test by triggering)
- ✅ Health checks respond <100ms
- ✅ Logs parseable and searchable
- ✅ Dashboards show real-time data
- ✅ Deployment takes <10 minutes

---

## Resources

**Required Infrastructure:**

- Postgres 16 (managed or self-hosted)
- ClickHouse (cloud or self-hosted)
- Redis 7 (with persistence)
- Kafka (3+ brokers for HA)
- Application servers (2+ for redundancy)

**External APIs:**

- Binance testnet (free)
- OpenAI/Anthropic (budget ~$50/day for MVP)
- Telegram Bot API (free)

**Team:**

- 1-2 backend developers (Go)
- DevOps support (deployment, monitoring)
- QA/testing (manual smoke tests)

---

## Next Steps After MVP

**Month 2:**

- Add more exchanges (OKX, Bybit production)
- Web UI (if Telegram isn't enough)
- Advanced risk features (correlation limits)
- Performance optimization (reduce latency)

**Month 3:**

- ML regime detection (market conditions)
- Social features (leaderboard, copy trading)
- Mobile app (React Native)
- Advanced strategies (mean reversion, arbitrage)

**Long-term:**

- Multi-asset support (stocks, forex, commodities)
- Institutional features (API for funds)
- White-label solution
- Decentralized execution (on-chain)

---

## Questions & Clarifications

**Q: Why Phase 1 is Order Execution, not Auth/GraphQL?**

A: Auth/GraphQL mostly works. The critical blocker is orders not executing on exchanges. Without this, nothing else matters.

**Q: Why no Web UI in MVP?**

A: Telegram bot provides all essential functionality. Web UI is polish, not core. Can add later if demand exists.

**Q: Why Phase 7 (Testing) comes before Phase 8 (Launch)?**

A: You can't launch to production without comprehensive tests. Phase 7 ensures system is bulletproof.

**Q: Can phases run in parallel?**

A: Partially. Phase 6 (Observability) can run parallel to Phase 4-5. But Phase 1-3 must be sequential (each depends on previous).

**Q: What if timeline slips?**

A: Each phase has buffer. If Phase 1 takes 4 days instead of 3, adjust timeline. Don't skip testing to save time.

---

## Contact & Support

**Questions about this plan:**

- Review with team before starting
- Adjust estimates based on team velocity
- Break phases into smaller tickets (use Jira/Linear)

**During implementation:**

- Daily standups to track progress
- Unblock issues immediately (don't wait)
- Update this doc if scope changes

**Blockers:**

- External API issues (exchange, LLM)
- Infrastructure problems (DB, Kafka)
- Team capacity constraints

---

**Last Updated:** 2024-12-04
**Version:** 1.0 (MVP Plan)
**Status:** Ready for Implementation
