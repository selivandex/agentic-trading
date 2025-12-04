<!-- @format -->

# Database Entities Architecture

## Overview

This document describes the database entity structure for an **AI-driven autonomous hedge fund** system. The architecture is designed to support:

- **Multi-user portfolio management** where each user operates their own hedge fund
- **AI agents** acting as trading desk analysts (10-20 agents per user)
- **Real-time trading** across multiple exchanges (Binance, Bybit, OKX)
- **Risk management** with circuit breakers and position monitoring
- **Memory systems** for agent learning and pattern recognition
- **Event-driven architecture** with Kafka for async processing

The system follows **Clean Architecture** principles with strict separation between domain entities, business logic, and infrastructure.

---

## Core Entity Structure

### 1. User Management

#### `User`

Represents a hedge fund participant (client). Each user operates their own autonomous portfolio managed by AI agents.

**Key Fields:**

- `id`: UUID (primary key)
- `telegram_id`: int64 (Telegram bot integration)
- `telegram_username`, `first_name`, `last_name`: User profile data
- `is_active`: bool (account status)
- `is_premium`: bool (subscription tier)
- `limit_profile_id`: UUID (FK to `LimitProfile` - determines usage limits)
- `settings`: JSONB (risk parameters, AI preferences, circuit breaker config)

**Settings JSONB Structure:**

```json
{
  "default_ai_provider": "claude",
  "default_ai_model": "claude-sonnet-4",
  "risk_level": "moderate",
  "max_positions": 3,
  "max_portfolio_risk": 10.0,
  "max_daily_drawdown": 5.0,
  "max_consecutive_loss": 3,
  "notifications_on": true,
  "circuit_breaker_on": true,
  "max_position_size_usd": 1000.0,
  "max_total_exposure_usd": 5000.0,
  "min_position_size_usd": 10.0,
  "max_leverage_multiple": 1.0,
  "allowed_exchanges": ["binance", "bybit"]
}
```

**Relationships:**

- 1 User → N ExchangeAccounts (multi-exchange support)
- 1 User → N Strategies (portfolios)
- 1 User → N Positions (open trades)
- 1 User → N Orders (trade history)
- 1 User → 1 CircuitBreakerState (risk monitoring)
- 1 User → 1 LimitProfile (tier limits)

---

#### `LimitProfile`

Defines usage limits based on subscription tier (Free, Basic, Premium, Enterprise).

**Key Fields:**

- `id`: UUID
- `name`: string ("free", "basic", "premium", "enterprise")
- `description`: string
- `limits`: JSONB (dynamic limit configuration)
- `is_active`: bool

**Limits JSONB Structure:**

```json
{
  "exchanges_count": 2,
  "active_positions": 5,
  "daily_trades_count": 20,
  "monthly_trades_count": 300,
  "trading_pairs_count": 20,
  "monthly_ai_requests": 1000,
  "max_agents_count": 5,
  "advanced_agents_access": true,
  "max_leverage": 3.0,
  "max_position_size_usd": 1000.0,
  "max_total_exposure_usd": 5000.0,
  "backtesting_allowed": true,
  "live_trading_allowed": true,
  "realtime_data_access": true,
  "priority_support": true
}
```

**Purpose in Hedge Fund:**

- Enforces **capital limits** per tier (prevents over-leveraging)
- Controls **AI agent usage** (cost management)
- Manages **data access** (real-time vs delayed market data)
- Defines **risk parameters** per subscription level

---

### 2. Exchange Integration

#### `ExchangeAccount`

Represents a user's connection to a cryptocurrency exchange. Each user can connect multiple exchanges with different trading accounts.

**Key Fields:**

- `id`: UUID
- `user_id`: UUID (FK to User)
- `exchange`: enum (binance, bybit, okx, kucoin, gate)
- `label`: string ("Main Binance", "Futures Bybit")
- `api_key_encrypted`: bytea (AES-256 encrypted credentials)
- `secret_encrypted`: bytea
- `passphrase`: bytea (for OKX)
- `is_testnet`: bool (paper trading mode)
- `permissions`: string[] (["spot", "futures", "read", "trade"])
- `is_active`: bool (can be deactivated if API keys fail)
- `last_sync_at`: timestamp (last balance/position sync)
- `listen_key_encrypted`: bytea (WebSocket User Data Stream key)
- `listen_key_expires_at`: timestamp (auto-renewal tracking)

**Security:**

- All credentials stored **encrypted at rest**
- Uses `pkg/crypto` encryption helpers
- Never logs or exposes plaintext keys
- WebSocket listen keys auto-renewed via background worker

**Relationships:**

- 1 ExchangeAccount → N Positions
- 1 ExchangeAccount → N Orders
- 1 ExchangeAccount → N TradingPairs

**Hedge Fund Specifics:**

- Users can segregate capital across multiple exchanges
- Each exchange account operates independently (risk isolation)
- Automated health checks deactivate accounts with invalid API keys
- Listen keys enable **real-time order/position updates** via WebSocket

---

### 3. Portfolio & Strategy Management

#### `Strategy` (Portfolio)

Represents a **trading portfolio** (hedge fund strategy). Each user can have multiple strategies with different risk profiles and asset allocations.

**Key Fields:**

- `id`: UUID
- `user_id`: UUID (FK to User)
- `name`: string ("Conservative Growth", "Momentum Trading")
- `description`: string
- `status`: enum (active, paused, closed)
- `allocated_capital`: decimal (initial capital - **NEVER changes**)
- `current_equity`: decimal (cash + positions market value - **updated real-time**)
- `cash_reserve`: decimal (unallocated cash available for new positions)
- `market_type`: enum (spot, futures)
- `risk_tolerance`: enum (conservative, moderate, aggressive)
- `rebalance_frequency`: enum (daily, weekly, monthly, never)
- `target_allocations`: JSONB ({"BTC/USDT": 0.5, "ETH/USDT": 0.3, "SOL/USDT": 0.2})
- `total_pnl`: decimal (CurrentEquity - AllocatedCapital)
- `total_pnl_percent`: decimal ((CurrentEquity/AllocatedCapital - 1) \* 100)
- `sharpe_ratio`: decimal (risk-adjusted returns)
- `max_drawdown`: decimal (peak-to-trough decline)
- `win_rate`: decimal (winning trades %)
- `reasoning_log`: JSONB (AI reasoning for portfolio creation - **explainability**)

**Target Allocations Example:**

```json
{
  "BTC/USDT": 0.4,
  "ETH/USDT": 0.3,
  "SOL/USDT": 0.15,
  "AVAX/USDT": 0.1,
  "USDT": 0.05
}
```

**Business Logic:**

```go
// Capital allocation flow
CurrentEquity = CashReserve + Sum(PositionsMarketValue)
TotalPnL = CurrentEquity - AllocatedCapital
TotalPnLPercent = (CurrentEquity / AllocatedCapital - 1) * 100

// When opening position:
strategy.AllocateCash(amount) -> reduces CashReserve

// When closing position:
strategy.ReturnCash(positionValue) -> increases CashReserve
strategy.UpdateEquity(positionsValue) -> recalculates metrics
```

**Relationships:**

- 1 Strategy → N Positions (portfolio holdings)
- 1 Strategy → N TradingPairs (watchlist/allocation targets)

**Hedge Fund Specifics:**

- Tracks **allocated capital separately** from current equity (essential for PnL)
- Supports **portfolio rebalancing** based on target allocations
- Stores **AI reasoning** for regulatory compliance and explainability
- Multiple strategies allow **risk diversification** (conservative + aggressive)

---

### 4. Trading Execution

#### `Position`

Represents an **open trading position** (long or short). This is the core entity for active trades.

**Key Fields:**

- `id`: UUID
- `user_id`: UUID (FK to User)
- `trading_pair_id`: UUID (FK to TradingPair)
- `exchange_account_id`: UUID (FK to ExchangeAccount)
- `strategy_id`: UUID (FK to Strategy - **NEW: links position to portfolio**)
- `symbol`: string (BTC/USDT)
- `market_type`: enum (spot, futures)
- `side`: enum (long, short)
- `size`: decimal (position size in base asset)
- `entry_price`: decimal (average entry price)
- `current_price`: decimal (updated real-time from WebSocket)
- `liquidation_price`: decimal (for futures - critical for risk)
- `leverage`: int (1-125x for futures)
- `margin_mode`: enum (cross, isolated)
- `unrealized_pnl`: decimal (real-time P&L calculation)
- `unrealized_pnl_pct`: decimal (P&L percentage)
- `realized_pnl`: decimal (locked-in profits from partial closes)
- `stop_loss_price`: decimal (risk management)
- `take_profit_price`: decimal (profit target)
- `trailing_stop_pct`: decimal (dynamic SL adjustment)
- `stop_loss_order_id`: UUID (FK to Order - tracks SL order)
- `take_profit_order_id`: UUID (FK to Order - tracks TP order)
- `open_reasoning`: text (AI reasoning for opening position)
- `status`: enum (open, closed, liquidated)
- `opened_at`: timestamp
- `closed_at`: timestamp (NULL if open)

**Real-time P&L Calculation:**

```go
// For LONG position:
UnrealizedPnL = (CurrentPrice - EntryPrice) * Size
UnrealizedPnLPct = ((CurrentPrice / EntryPrice) - 1) * 100 * Leverage

// For SHORT position:
UnrealizedPnL = (EntryPrice - CurrentPrice) * Size
UnrealizedPnLPct = ((EntryPrice / CurrentPrice) - 1) * 100 * Leverage
```

**Relationships:**

- 1 Position → 1 Strategy (portfolio allocation)
- 1 Position → 1 TradingPair (market configuration)
- 1 Position → 1 ExchangeAccount (execution venue)
- 1 Position → N Orders (entry, SL, TP orders)

**Hedge Fund Specifics:**

- **Real-time price updates** via WebSocket (sub-second latency)
- **Automatic SL/TP order placement** on position open
- **Liquidation monitoring** for futures positions (risk alert system)
- **Strategy linkage** enables portfolio-level risk aggregation
- Stores **AI reasoning** for trade journal and post-mortem analysis

---

#### `Order`

Represents a **trading order** (buy/sell instruction sent to exchange).

**Key Fields:**

- `id`: UUID (internal ID)
- `exchange_order_id`: string (exchange's order ID - for tracking)
- `user_id`: UUID (FK to User)
- `trading_pair_id`: UUID (FK to TradingPair)
- `exchange_account_id`: UUID (FK to ExchangeAccount)
- `symbol`: string (BTC/USDT)
- `market_type`: enum (spot, futures)
- `side`: enum (buy, sell)
- `type`: enum (market, limit, stop_market, stop_limit)
- `status`: enum (pending, open, filled, partial, canceled, rejected, expired)
- `price`: decimal (limit price, NULL for market orders)
- `amount`: decimal (order size)
- `filled_amount`: decimal (executed quantity)
- `avg_fill_price`: decimal (average execution price)
- `stop_price`: decimal (trigger price for stop orders)
- `reduce_only`: bool (futures - can only close position, not open)
- `agent_id`: string (which AI agent created the order)
- `reasoning`: text (AI reasoning for trade decision)
- `parent_order_id`: UUID (FK to Order - for SL/TP orders linked to main order)
- `fee`: decimal (trading fee paid)
- `fee_currency`: string (usually USDT)
- `created_at`: timestamp
- `filled_at`: timestamp (NULL if not filled yet)

**Order Lifecycle:**

```
pending -> open -> partial -> filled
                    ↓
                 canceled / rejected / expired
```

**Relationships:**

- 1 Order → 0..1 Position (entry order creates position)
- 1 Order → 0..1 ParentOrder (SL/TP linked to main order)

**Hedge Fund Specifics:**

- **Agent attribution** (`agent_id`) for performance tracking
- **Reasoning storage** for compliance and explainability
- **Parent-child relationships** for SL/TP order management
- **Status tracking** for order fill monitoring and retries
- Integration with exchange WebSocket for **real-time order updates**

---

#### `TradingPair`

Represents a **configured trading pair** (watchlist item) for a user.

**Key Fields:**

- `id`: UUID
- `user_id`: UUID (FK to User)
- `strategy_id`: UUID (FK to Strategy - links to portfolio)
- `exchange_account_id`: UUID (FK to ExchangeAccount)
- `symbol`: string (BTC/USDT)
- `market_type`: enum (spot, futures)
- `budget`: decimal (USDT allocated to this pair)
- `max_position_size`: decimal (max single trade size)
- `max_leverage`: int (for futures)
- `stop_loss_percent`: decimal (default SL %)
- `take_profit_percent`: decimal (default TP %)
- `ai_provider`: string (claude, openai, deepseek)
- `strategy_mode`: enum (auto, semi_auto, signals)
- `timeframes`: string[] (["1h", "4h", "1d"])
- `is_active`: bool
- `is_paused`: bool
- `paused_reason`: string (e.g., "high volatility", "circuit breaker")

**Strategy Modes:**

- **auto**: Fully autonomous trading (no human approval)
- **semi_auto**: Requires user confirmation before execution
- **signals**: Only sends trade signals, no execution

**Relationships:**

- 1 TradingPair → N Positions (historical positions on this pair)
- 1 TradingPair → 1 Strategy (portfolio allocation)

**Hedge Fund Specifics:**

- Links **market configuration** to **portfolio strategy**
- Enables **per-pair risk limits** (max size, leverage)
- Supports **pausing individual pairs** without closing positions
- AI agents use this as **trading watchlist** with execution parameters

---

### 5. Risk Management

#### `CircuitBreakerState`

Tracks **risk metrics** and circuit breaker status per user. Auto-halts trading when risk thresholds are breached.

**Key Fields:**

- `user_id`: UUID (FK to User - primary key)
- `is_triggered`: bool (circuit breaker active?)
- `triggered_at`: timestamp
- `trigger_reason`: string ("daily drawdown exceeded", "consecutive losses")
- `daily_pnl`: decimal (today's P&L)
- `daily_pnl_percent`: decimal (today's P&L %)
- `daily_trade_count`: int
- `daily_wins`: int
- `daily_losses`: int
- `consecutive_losses`: int (counter for consecutive losing trades)
- `max_daily_drawdown`: decimal (threshold from user settings)
- `max_consecutive_loss`: int (threshold from user settings)
- `reset_at`: timestamp (next day 00:00 UTC - auto-reset)

**Circuit Breaker Logic:**

```go
// Trigger conditions (any of):
1. DailyPnLPercent <= -MaxDailyDrawdown
2. ConsecutiveLosses >= MaxConsecutiveLoss
3. User manually activated kill switch

// When triggered:
- Halt all new position openings
- Allow only position closes
- Send emergency notification to user
- Auto-reset at next trading day (00:00 UTC)
```

**Relationships:**

- 1 CircuitBreakerState → 1 User (one-to-one)

**Hedge Fund Specifics:**

- **Automatic risk protection** prevents catastrophic losses
- **Daily reset** allows fresh start each trading day
- **Consecutive loss tracking** detects strategy breakdown
- Integrated with order execution pipeline (blocks new orders when triggered)

---

#### `RiskEvent`

Historical log of **risk alerts and circuit breaker triggers**.

**Key Fields:**

- `id`: UUID
- `user_id`: UUID (FK to User)
- `timestamp`: timestamp
- `event_type`: enum (drawdown_warning, consecutive_loss, circuit_breaker_triggered, max_exposure_reached, kill_switch_activated, anomaly_detected)
- `severity`: enum (warning, critical)
- `message`: string ("Daily drawdown -6.2% exceeds limit -5.0%")
- `data`: JSONB (detailed event context)
- `acknowledged`: bool (user has seen the alert)

**Event Types:**

- `drawdown_warning`: Approaching drawdown limit (80% threshold)
- `consecutive_loss`: Multiple losing trades in a row
- `circuit_breaker_triggered`: Trading halted
- `max_exposure_reached`: Portfolio exposure limit hit
- `kill_switch_activated`: User manually stopped trading
- `anomaly_detected`: ML-based anomaly detection

**Hedge Fund Specifics:**

- **Audit trail** for risk events (regulatory compliance)
- **Real-time alerts** to user via Telegram
- **ML anomaly detection** for unusual trading patterns

---

### 6. AI Agent Memory & Learning

#### `Memory`

Unified memory system for AI agents. Supports **episodic** (user-specific) and **semantic** (collective) memory.

**Key Fields:**

- `id`: UUID
- `scope`: enum (user, collective, working)
- `user_id`: UUID (NULL for collective memories)
- `agent_id`: string (which agent created the memory)
- `session_id`: string (ADK session ID)
- `type`: enum (observation, decision, trade, lesson, regime, pattern)
- `content`: text (memory content)
- `embedding`: vector(1536) (pgvector embedding for semantic search)
- `embedding_model`: string (text-embedding-3-large)
- `embedding_dimensions`: int (1536)
- `validation_score`: float64 (for collective memories - how accurate?)
- `validation_trades`: int (number of trades validating this memory)
- `symbol`: string (BTC/USDT - context)
- `timeframe`: string (1h, 4h, 1d)
- `importance`: float64 (0.0-1.0 - memory weight)
- `tags`: string[] (["breakout", "volume_spike", "bullish"])
- `metadata`: JSONB (flexible additional data)
- `personality`: string (for collective memories - "aggressive", "conservative")
- `source_user_id`: UUID (for collective memories - who discovered this pattern?)
- `source_trade_id`: UUID (trade that validated this memory)
- `expires_at`: timestamp (NULL for permanent memories)

**Memory Scopes:**

- **user**: Agent-level episodic memory for specific user (e.g., "I opened BTC long at $42k")
- **collective**: Agent-level shared knowledge across all users (e.g., "BTC breakouts above $45k succeed 73% of the time")
- **working**: Temporary scratch space (e.g., current analysis context)

**Memory Types:**

- **observation**: Market pattern observed ("volume spike detected")
- **decision**: Trade decision reasoning ("entering long due to bullish divergence")
- **trade**: Trade outcome ("closed position with +5.2% profit")
- **lesson**: Pattern learned from trade outcome ("MACD crossovers in range markets are unreliable")
- **regime**: Market regime classification ("transitioning from trend to range")
- **pattern**: Detected recurring pattern ("cup and handle formation")

**Semantic Search:**

```sql
-- Find similar memories using cosine similarity
SELECT *, 1 - (embedding <=> query_embedding) AS similarity
FROM memories
WHERE scope = 'collective'
  AND type = 'lesson'
ORDER BY similarity DESC
LIMIT 10;
```

**Collective Memory Validation:**

- Memories start with `validation_score = 0.0`
- Each successful trade using the memory increases validation
- Memories with high validation scores are prioritized
- Low-scoring memories auto-expire after N days

**Relationships:**

- N Memories → 1 User (episodic memories)
- N Memories → 1 Agent (agent attribution)
- N Memories → 1 SourceTrade (memory provenance)

**Hedge Fund Specifics:**

- **Semantic search** retrieves relevant past experiences for current decision
- **Collective learning** allows agents to learn from all users' trades
- **Validation system** ensures only high-quality patterns are retained
- **Personality filtering** enables risk-profile-specific memories
- **Explainability** via memory provenance (which trade/user discovered this?)

---

#### `Session`

Tracks **ADK (Agent Development Kit) agent sessions** for conversation history.

**Key Fields:**

- `id`: UUID
- `app_name`: string ("prometheus-agents")
- `user_id`: string (maps to User.ID)
- `session_id`: string (ADK session ID)
- `state`: JSONB (session state variables)
- `events`: JSONB array (conversation history: messages, tool calls, responses)
- `updated_at`: timestamp
- `created_at`: timestamp

**Event Structure:**

```json
{
  "event_id": "evt_abc123",
  "author": "market_analyst",
  "content": { "type": "text", "text": "Analyzing BTC momentum..." },
  "timestamp": "2024-01-15T10:30:00Z",
  "actions": {
    "transfer_to_agent": "risk_manager",
    "state_delta": { "analysis_complete": true }
  },
  "usage_metadata": {
    "prompt_tokens": 1500,
    "completion_tokens": 300,
    "total_tokens": 1800
  }
}
```

**State Management:**

- `_app_` prefix: Application-level state (all users)
- `_user_` prefix: User-level state (across sessions)
- `_temp_` prefix: Session-specific temporary state

**Hedge Fund Specifics:**

- **Multi-agent conversations** (10-20 agents per session)
- **Token usage tracking** for cost management
- **Agent handoffs** via `transfer_to_agent`
- **State persistence** across sessions

---

### 7. Performance Analytics

#### `JournalEntry`

**Trade journal** with post-trade analysis and lessons learned.

**Key Fields:**

- `id`: UUID
- `user_id`: UUID (FK to User)
- `trade_id`: UUID (FK to Position)
- `symbol`: string
- `side`: enum (long, short)
- `entry_price`, `exit_price`, `size`: decimal
- `pnl`, `pnl_percent`: decimal
- `strategy_used`: string ("momentum breakout")
- `timeframe`: string (1h, 4h, 1d)
- `setup_type`: enum (breakout, reversal, trend_follow)
- `market_regime`: enum (trend, range, volatile)
- `entry_reasoning`: text (AI reasoning)
- `exit_reasoning`: text (AI reasoning)
- `confidence_score`: float64 (0.0-1.0)
- `rsi_at_entry`, `atr_at_entry`, `volume_at_entry`: float64
- `was_correct_entry`, `was_correct_exit`: bool
- `max_drawdown`, `max_profit`: decimal (intra-trade metrics)
- `hold_duration`: int64 (seconds)
- `lessons_learned`: text (AI-generated post-mortem)
- `improvement_tips`: text (AI suggestions)

**Business Logic:**

```go
// Post-trade analysis (AI-generated):
1. Was entry timing optimal? (check if max_profit > exit_price)
2. Was exit premature? (check price action after close)
3. Did market regime change during trade?
4. What indicators worked/failed?
5. Extract lessons for future trades
```

**Hedge Fund Specifics:**

- **Automated trade journal** generation (no manual input)
- **AI-powered post-mortem** analysis
- **Pattern extraction** (feeds into collective memory)
- **Strategy performance tracking** by setup type

---

#### `AIUsageLog` (ClickHouse)

Tracks **AI model usage and costs** (stored in ClickHouse for analytics).

**Key Fields:**

- `timestamp`: DateTime
- `event_id`: string
- `user_id`: string
- `session_id`: string
- `agent_name`: string (market_analyst, risk_manager, etc.)
- `provider`: enum (anthropic, openai, google, deepseek)
- `model_id`: string (claude-sonnet-4, gpt-4o)
- `prompt_tokens`, `completion_tokens`, `total_tokens`: uint32
- `input_cost_usd`, `output_cost_usd`, `total_cost_usd`: float64
- `tool_calls_count`: uint16
- `is_cached`, `cache_hit`: bool (prompt caching)
- `latency_ms`: uint32
- `reasoning_step`: uint16 (step in reasoning chain)
- `workflow_name`: string (MarketResearchWorkflow)

**Analytics Queries:**

```sql
-- Cost per user per day
SELECT user_id, toDate(timestamp) as date, sum(total_cost_usd) as cost
FROM ai_usage_logs
GROUP BY user_id, date
ORDER BY date DESC;

-- Most expensive agents
SELECT agent_name, sum(total_cost_usd) as total_cost, count() as requests
FROM ai_usage_logs
GROUP BY agent_name
ORDER BY total_cost DESC;
```

**Hedge Fund Specifics:**

- **Cost attribution** per user and agent
- **Prompt caching analytics** (cost savings)
- **Performance monitoring** (latency tracking)
- **Billing basis** for subscription tiers

---

### 8. Market Data & Regime Detection

#### Market Data Tables (ClickHouse)

High-frequency market data stored in **ClickHouse** for fast analytics.

**Tables:**

- `market_data_tick`: Real-time tick data (WebSocket)
- `market_data_kline`: Candlestick data (1m, 5m, 15m, 1h, 4h, 1d)
- `market_data_orderbook`: Order book snapshots
- `market_data_trades`: Executed trades

**Example Schema (market_data_kline):**

```sql
CREATE TABLE market_data_kline (
    timestamp DateTime64(3),
    symbol String,
    exchange String,
    timeframe String,
    open Decimal(18, 8),
    high Decimal(18, 8),
    low Decimal(18, 8),
    close Decimal(18, 8),
    volume Decimal(18, 8),
    quote_volume Decimal(18, 8),
    trades_count UInt32
) ENGINE = MergeTree()
ORDER BY (symbol, exchange, timeframe, timestamp);
```

**Hedge Fund Specifics:**

- **Sub-second latency** for real-time data
- **Historical backtesting** data (years of candles)
- **Cross-exchange aggregation** (multi-venue analytics)

---

#### `RegimeClassification` (PostgreSQL)

ML-based **market regime detection** (trend, range, volatile).

**Key Fields:**

- `id`: UUID
- `symbol`: string (BTC/USDT)
- `timeframe`: string (1h, 4h, 1d)
- `timestamp`: timestamp
- `regime`: enum (trend_up, trend_down, range, volatile, transition)
- `confidence`: float64 (0.0-1.0)
- `indicators`: JSONB (ADX, ATR, Bollinger Bands width, etc.)
- `model_version`: string (regime_model_v1.2)

**Business Logic:**

```
Trend Up:    ADX > 25, Price > MA(50), ATR stable
Trend Down:  ADX > 25, Price < MA(50), ATR stable
Range:       ADX < 20, Bollinger Bands narrow
Volatile:    ATR spike, Bollinger Bands wide
Transition:  Regime change in progress
```

**Hedge Fund Specifics:**

- **Regime-aware trading** (different strategies per regime)
- **ML model predictions** updated every 15 minutes
- **Agent context** for decision-making

---

## Entity Relationship Diagram

```
User ──┬─→ ExchangeAccount ──→ Position ──→ Order
       │                       ↓
       ├─→ Strategy ───────────┘
       │                       ↓
       ├─→ TradingPair ────────┘
       │
       ├─→ CircuitBreakerState
       ├─→ LimitProfile
       ├─→ Memory (episodic)
       ├─→ Session (agent conversations)
       ├─→ JournalEntry (trade analysis)
       └─→ RiskEvent (alerts)

Memory (collective) ─→ SourceUser
                    ─→ SourceTrade

Order ──→ ParentOrder (SL/TP linkage)
Position ──→ StopLossOrder
         ──→ TakeProfitOrder
```

---

## Key Design Patterns for Hedge Funds

### 1. **Capital Tracking Pattern**

```
Strategy.AllocatedCapital = Initial capital (IMMUTABLE)
Strategy.CurrentEquity = CashReserve + Sum(Position.MarketValue)
Strategy.TotalPnL = CurrentEquity - AllocatedCapital
```

**Why?** Essential for accurate P&L calculation and ROI metrics.

---

### 2. **Position-Strategy Linkage**

Every `Position` has `strategy_id` linking it to a portfolio. This enables:

- Portfolio-level risk aggregation
- Per-strategy performance analytics
- Multi-strategy accounts (conservative + aggressive)

---

### 3. **Real-time Price Updates**

`Position.current_price` updated via WebSocket → triggers `unrealized_pnl` recalculation → updates `Strategy.current_equity`.

---

### 4. **Circuit Breaker Integration**

Before opening position:

```go
1. Check CircuitBreakerState.is_triggered
2. If triggered → reject order
3. If not → validate risk limits (max exposure, max leverage)
4. Execute order
5. Update CircuitBreakerState metrics
```

---

### 5. **Memory-Driven Trading**

Before making decision:

```go
1. Query relevant memories (semantic search)
2. Filter by market regime, symbol, timeframe
3. Prioritize high validation_score memories
4. Use memories as context for AI reasoning
5. Store new decision in memory
6. Validate memory based on trade outcome
```

---

### 6. **AI Cost Management**

```go
1. Track all AI calls in AIUsageLog (ClickHouse)
2. Check LimitProfile.monthly_ai_requests
3. Use prompt caching to reduce costs
4. Auto-throttle if approaching limit
5. Send billing alerts to user
```

---

### 7. **Order-Position Lifecycle**

```
1. Agent decides to open position
2. Create Order (status=pending)
3. Submit to exchange → exchange_order_id assigned
4. Order fills → Create Position
5. Create SL/TP Orders (parent_order_id = entry order)
6. Update Position.stop_loss_order_id, take_profit_order_id
7. Monitor Position via WebSocket
8. SL/TP triggers → Close Position
9. Create JournalEntry for analysis
10. Extract lessons → Store in Memory (collective)
```

---

## Database Technology Stack

- **PostgreSQL 16** (transactional data, pgvector for embeddings)
- **ClickHouse** (time-series analytics, market data, AI usage logs)
- **Redis** (caching, distributed locks, circuit breaker state)
- **Kafka** (event-driven architecture, async processing)

---

## Compliance & Explainability

Every trade has **full audit trail**:

1. `Position.open_reasoning` (why opened)
2. `Order.reasoning` (why this order type/price)
3. `Memory` (historical context used)
4. `Session.events` (agent conversation)
5. `JournalEntry.lessons_learned` (post-mortem)

This enables:

- Regulatory compliance (MiFID II, SEC requirements)
- User transparency (explain any trade)
- ML model validation (verify decision quality)

---

## Conclusion

This database architecture supports a **fully autonomous AI-driven hedge fund** with:

- ✅ Multi-user portfolio management
- ✅ Real-time risk monitoring
- ✅ AI agent memory and learning
- ✅ Multi-exchange execution
- ✅ Regulatory compliance
- ✅ Cost tracking and billing
- ✅ Advanced analytics

The design follows **Clean Architecture** principles with strict separation of concerns, enabling easy testing, maintenance, and scalability.



