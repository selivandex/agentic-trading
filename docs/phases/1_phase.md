<!-- @format -->

# Phase 1 — Order Execution Bridge (2-3 days)

## Objective

Connect pending orders to real Binance execution. Currently `/invest` flow works perfectly until order execution - PortfolioArchitect creates orders in DB but they never reach Binance. This is THE critical blocker for MVP.

## Why This is Phase 1

The `/invest` command and PortfolioArchitect workflow already work! The agent analyzes market, designs portfolio, calls `execute_trade` tool, and creates orders in DB. But nothing happens after that - orders sit in "pending" status forever because:

1. `Order.Service.Place()` only writes to DB (never calls exchange)
2. No worker picks up pending orders and sends them to Binance
3. Workers are disabled in `bootstrap/container.go:271`

**This phase bridges the gap between AI decision (order in DB) and real trading (order on Binance).**

## Current Flow (Broken)

```
User: /invest
  ↓
Bot: Select exchange account (inline buttons) ✅
  ↓
User: selects Binance account
  ↓
Bot: Select market type (spot/futures) ✅
  ↓
User: selects Spot
  ↓
Bot: Select risk profile (conservative/moderate/aggressive) ✅
  ↓
User: selects Moderate
  ↓
Bot: Enter investment amount ✅
  ↓
User: types "1000"
  ↓
Strategy created in DB ✅
  ↓
PortfolioArchitect workflow runs ✅
  ↓
Agent calls execute_trade("BTC/USDT", $350) ✅
  ↓
Order saved to DB (status=pending) ✅
  ↓
❌ STOPS HERE - nothing sends order to Binance
  ↓
User: /portfolio → empty (no positions created)
```

## Target Flow (After Phase 1)

```
User: /invest
  ↓
Bot: Select exchange → market type → risk profile → enter amount ✅
  ↓
User: completes flow (e.g., Binance, Spot, Moderate, $1000) ✅
  ↓
Strategy created in DB ✅
  ↓
PortfolioArchitect workflow runs ✅
  ↓
Agent analyzes market, designs portfolio ✅
  ↓
Agent calls execute_trade("BTC/USDT", $350) ✅
  ↓
Order saved to DB (status=pending) ✅
  ↓
✅ NEW: OrderExecutor picks up order (polls every 5s)
  ↓
✅ NEW: Resolves Binance account, decrypts API keys
  ↓
✅ NEW: Calls binance.PlaceOrder() with real API
  ↓
✅ NEW: Order appears on Binance testnet
  ↓
✅ NEW: Updates order status (submitted → filled)
  ↓
✅ NEW: WebSocket receives fill → creates position
  ↓
User: /portfolio → sees BTC position with PnL ✅
```

## Scope & Tasks

### 1. Order Execution Worker (Day 1-2)

**MVP Scope: Binance only** (no OKX, Bybit for now)

Create `internal/workers/trading/order_executor.go`:

```go
// Polls pending orders every 5-10 seconds
// For each pending order:
//   1. Resolve exchange account + decrypt credentials
//   2. Call exchange.PlaceOrder()
//   3. Update order status (submitted/rejected)
//   4. Handle errors with retry logic
```

**Key implementation:**

- Use `order.Repository.GetPending()` to fetch orders
- Use `exchangefactory.UserExchangeFactory` to get exchange client
- Update order with `ExchangeOrderID` after successful placement
- Set status to `rejected` with error reason on failure
- Add exponential backoff for retries (max 3 attempts)

### 2. Fix ExecutionService (Day 2)

Update `internal/services/execution/service.go`:

- Resolve `ExchangeAccountID` from user preferences or strategy config
- Add validation that account exists and is active
- Remove `uuid.Nil` placeholder

### 3. Order Repository Method (Day 1)

Add to `internal/domain/order/repository.go`:

```go
GetPending(ctx context.Context, limit int) ([]*Order, error)
GetPendingByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error)
```

Implement in `internal/repository/postgres/order.go`.

### 4. Worker Registration (Day 2)

Uncomment workers in `internal/bootstrap/container.go:271`:

```go
if err := c.Background.WorkerScheduler.Start(c.Context); err != nil {
    return fmt.Errorf("failed to start workers: %w", err)
}
```

Register OrderExecutor in `internal/bootstrap/workers.go`.

### 5. Tests (Day 3)

**Repository tests (integration with testcontainers):**

- `order_repository_test.go` - test GetPending, GetPendingByUser

**Service tests (with mocks):**

- `order_executor_test.go` - mock exchange adapter, test happy path + retries

**Integration test:**

- Create pending order → OrderExecutor runs → verify exchange.PlaceOrder called

## Deliverables

- ✅ OrderExecutor worker running every 5-10 seconds
- ✅ Pending orders sent to real exchanges (testnet)
- ✅ Order statuses updated (submitted/rejected/filled)
- ✅ Tests: repository integration + worker unit tests
- ✅ Lint passes (`make lint`)

## Exit Criteria

1. Manual test: create pending order in DB → OrderExecutor picks it up → order appears on Binance testnet
2. Integration test passes without mocks
3. Error handling: invalid account → order rejected with clear reason
4. Metrics: orders_executed_total, orders_failed_total counters work

## Dependencies

None. This phase is self-contained and critical path.

---

**Success Metric:** `/invest 1000` in Telegram → orders execute on testnet → positions appear in DB
