<!-- @format -->

# Phase 6 — Observability & Alerting (1 day)

## Objective

Add essential monitoring: metrics, logs, alerts. Enable ops team to detect issues before users complain.

## Why After Phase 5

Phase 1-5 build complete trading system. Now add visibility to operate it reliably.

## Scope & Tasks

### 1. Reasoning Traces Visibility (4 hours)

**Make agent decisions transparent and debuggable.**

**Telegram Command:**

```go
// /reasoning <session_id>
func (bot *Bot) HandleReasoningCommand(ctx context.Context, sessionID string) error {
    // Get session events from Postgres
    events := sessionRepo.GetEventsBySessionID(ctx, sessionID)

    var output strings.Builder
    output.WriteString("💭 Agent Reasoning Trace\n\n")

    stepNum := 1
    for _, event := range events {
        if event.Content == nil {
            continue
        }

        // Show thinking text
        for _, part := range event.Content.Parts {
            if part.Text != "" {
                output.WriteString(fmt.Sprintf("%d. %s\n\n", stepNum, part.Text))
                stepNum++
            }

            // Show tool calls
            if part.FunctionCall != nil {
                output.WriteString(fmt.Sprintf("🔧 Tool: %s\n", part.FunctionCall.Name))
                output.WriteString(fmt.Sprintf("   Args: %v\n\n", part.FunctionCall.Args))
            }
        }

        // Show token usage
        if event.UsageMetadata != nil {
            output.WriteString(fmt.Sprintf("📊 Tokens: %d\n", event.UsageMetadata.TotalTokenCount))
        }
    }

    return bot.SendMessage(output.String())
}
```

**Implementation:**
1. Add command to telegram handler registry
2. Query session_events table by session_id
3. Format for readability (collapse tool responses, highlight decisions)
4. Add pagination for long sessions (>10 steps)

**Optional but recommended:**
```go
// /last_decision <strategy_id>
// Shows most recent PortfolioManager reasoning for this strategy
func (bot *Bot) HandleLastDecisionCommand(strategyID string) {
    strategy := strategyRepo.GetByID(strategyID)
    reasoning := strategy.ReasoningLog // Latest reasoning

    bot.SendMessage("💭 Latest Decision:\n\n" + reasoning)
}
```

### 2. Prometheus Metrics (3 hours)

Already exists: `internal/metrics/prometheus.go`

**Verify all critical metrics exposed:**

**Order metrics:**

- `orders_created_total{status, exchange, symbol}`
- `orders_executed_total{exchange, market_type}`
- `orders_failed_total{reason, exchange}`
- `order_execution_duration_seconds{exchange}`

**Position metrics:**

- `positions_open{user_id, exchange, symbol}`
- `positions_pnl_usd{user_id, strategy_id}`
- `position_duration_seconds{symbol, side}`

**Worker metrics:**

- `worker_runs_total{worker_name, status}`
- `worker_errors_total{worker_name, error_type}`
- `worker_duration_seconds{worker_name}`

**Risk metrics:**

- `trades_rejected_total{reason}`
- `circuit_breakers_triggered_total{level, user_id}`
- `user_exposure_percent{user_id}`

**Agent metrics:**

- `agent_sessions_total{agent_name, status}`
- `llm_requests_total{provider, model}`
- `llm_cost_usd_total{provider, model}`

### 2. Structured Logging (2 hours)

Already using `pkg/logger` with zap. Verify consistency:

**All log entries must include:**

- `component` - worker name, service name, consumer name
- `user_id` - when user-specific action
- `order_id` / `position_id` / `strategy_id` - when applicable
- `exchange` - when exchange interaction

**Log levels:**

- `Debug` - detailed execution flow (disabled in prod)
- `Info` - normal operations (order placed, position updated)
- `Warn` - recoverable errors (rate limit, retry successful)
- `Error` - action failed (order rejected, workflow error)

**Sample structured log:**

```go
log.Infow("Order executed successfully",
    "component", "order_executor",
    "user_id", userID,
    "order_id", orderID,
    "exchange", "binance",
    "symbol", "BTC/USDT",
    "duration_ms", duration.Milliseconds(),
)
```

### 3. Health Checks (2 hours)

Already exists: `internal/api/health/handler.go`

**Enhance `/health` endpoint:**

```json
{
  "status": "healthy",
  "timestamp": "2024-12-04T10:30:00Z",
  "components": {
    "postgres": { "status": "up", "latency_ms": 5 },
    "clickhouse": { "status": "up", "latency_ms": 12 },
    "redis": { "status": "up", "latency_ms": 2 },
    "kafka": { "status": "up", "lag": 0 },
    "workers": {
      "opportunity_finder": { "status": "running", "last_run": "2m ago" },
      "position_monitor": { "status": "running", "last_run": "30s ago" },
      "order_executor": { "status": "running", "last_run": "10s ago" }
    },
    "exchanges": {
      "binance": { "status": "connected", "websocket": "active" }
    }
  }
}
```

### 4. Telegram Alerting (4 hours)

Create admin notification channel:

**Critical alerts (immediate):**

- Worker crashed (stopped running for >5 minutes)
- Database connection lost
- Kafka consumer lag >1000 messages
- Circuit breaker triggered (user or system)
- Exchange API rate limit exceeded

**Warning alerts (can wait):**

- High order rejection rate (>10%)
- Exchange account degraded
- LLM API cost spike (>$10/hour)
- Unusual position count (>50 open positions)

**Implementation:**

```go
// internal/workers/alerting/telegram_notifier.go
func (tn *TelegramNotifier) SendAdminAlert(level, title, details string) {
    msg := fmt.Sprintf("🚨 %s\n\n%s\n\nTime: %s", title, details, time.Now())
    tn.bot.SendMessage(adminChatID, msg)
}
```

### 5. Basic Dashboards (2 hours)

If using Grafana (optional for MVP, but recommended):

**Dashboard 1: Trading Overview**

- Orders per minute (line chart)
- Order success rate (gauge)
- Open positions (table)
- Total PnL (line chart)

**Dashboard 2: System Health**

- Worker status (status panel)
- Kafka consumer lag (line chart)
- Database connection pool (gauge)
- API latency (heatmap)

**Dashboard 3: Risk & Limits**

- Circuit breaker triggers (counter)
- Rejected trades by reason (pie chart)
- User exposure distribution (histogram)

## Deliverables

- ✅ Reasoning traces visible via `/reasoning <session_id>` command
- ✅ `/last_decision <strategy_id>` shows agent's current thinking
- ✅ All critical metrics exposed on `/metrics`
- ✅ Structured logging with consistent fields
- ✅ Enhanced `/health` endpoint with component details
- ✅ Telegram alerts for critical events
- ✅ Basic Grafana dashboards (optional)
- ✅ Runbook: how to investigate alerts

## Exit Criteria

1. User can run `/reasoning <session_id>` and see agent's thinking step-by-step
2. `/portfolio` shows why agent is holding positions (inline reasoning)
3. Prometheus scrapes `/metrics` successfully (test with `curl`)
4. Trigger circuit breaker → admin receives Telegram alert within 30 seconds
5. Stop worker → health check shows degraded status
6. All logs parseable by log aggregator (structured JSON)
7. Dashboard shows real-time order flow and system health (optional)

## Dependencies

None. Can be done in parallel with other phases.

---

**Success Metric:** Ops team can detect and diagnose issues using metrics, logs, and alerts
