<!-- @format -->

# Phase 5 — Exchange Account Lifecycle (1-2 days)

## Objective

Complete exchange account management: validation, error handling, deactivation, listen key refresh. Ensure accounts remain healthy and secure.

## Why After Phase 4

Phase 1-4 enable core trading. Now harden account management to prevent failed orders due to invalid keys or expired sessions.

## Scope & Tasks

### 1. Account Validation on Creation (Day 1)

Update `internal/services/exchange/service.go`:

When user adds exchange account:

**Validation steps:**

1. Decrypt credentials
2. Test connection (call exchange.GetAccountInfo())
3. Verify permissions (spot trading, futures if needed)
4. Check IP whitelist (if configured)
5. Store `last_validated_at` timestamp

**Add to GraphQL mutation:**

```graphql
mutation CreateExchangeAccount(
  exchange: String!
  apiKey: String!
  apiSecret: String!
  passphrase: String  # For OKX
  testnet: Boolean
) {
  # Returns validation result + account ID
}
```

### 2. Listen Key Management (Day 1-2)

Binance/Bybit require listen key refresh every 30-60 minutes for user data streams.

**Current state:** Partial implementation in websocket clients.

**Complete:**

- Background job in OrderExecutor or separate worker
- Refresh every 30 minutes
- On failure: mark account as `degraded`, notify user
- After 3 failures: deactivate account, close streams

**Add to `internal/adapters/exchanges/*/client.go`:**

```go
RefreshListenKey(ctx, listenKey) (newKey, error)
```

### 3. Account Health Monitoring (Day 2)

Add worker `internal/workers/trading/account_health_monitor.go`:

**Every 5 minutes:**

- Check all active exchange accounts
- Test connection with lightweight call (ping or account info)
- Update `last_checked_at`, `status` in DB
- If 3 consecutive failures → deactivate account
- Notify user via Telegram

**Account statuses:**

- `active` - healthy, can trade
- `degraded` - issues detected, warn user
- `inactive` - deactivated, cannot trade (user can reactivate)
- `suspended` - exchange suspended account (admin action needed)

### 4. Error Handling for Failed Orders (Day 2)

When OrderExecutor fails to place order:

**Parse exchange error codes:**

- Invalid API key → deactivate account immediately
- Insufficient balance → circuit breaker (don't retry)
- Rate limit → exponential backoff
- Invalid symbol → log, skip
- Network timeout → retry (max 3 attempts)

**Update order status:**

```go
order.Status = "rejected"
order.ErrorReason = "Exchange: Invalid API signature"
order.RejectedAt = time.Now()
```

Publish `order.rejected` event to Kafka.

### 5. Tests (Day 2)

**Service tests (unit with mocks):**

- Create account with invalid key → validation fails
- Listen key refresh fails → account marked degraded
- 3 consecutive health checks fail → account deactivated

**Integration tests:**

- Create account → validate on testnet → verify active
- Simulate expired key → verify OrderExecutor rejects orders

**Error scenario tests:**

- Test all exchange error codes with fixtures
- Verify correct status updates and notifications

## Deliverables

- ✅ Account validation on creation (test API keys)
- ✅ Listen key refresh working (no stream disconnects)
- ✅ Account health monitoring (deactivate bad accounts)
- ✅ Error handling for all exchange failure modes
- ✅ Tests: validation + health checks + error parsing
- ✅ Metrics: accounts_active, accounts_degraded, validation_failures_total

## Exit Criteria

1. Add invalid API key → validation fails with clear error
2. Valid key → account activated, listen key refreshes automatically
3. Revoke API key on exchange → account deactivated within 5 minutes
4. Failed order with "insufficient balance" → circuit breaker (don't spam exchange)
5. All exchange errors logged with proper categorization

## Dependencies

Requires Phase 1 (OrderExecutor) to test error handling.

---

**Success Metric:** Exchange accounts remain healthy, invalid keys detected immediately, no silent failures
