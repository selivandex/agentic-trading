<!-- @format -->

# Agentic Trading System — Technical Specification

**Version:** 1.0  
**Date:** November 2025  
**Codename:** Prometheus

---

## 1. Executive Summary

Автономная мульти-агентная торговая система для криптовалютных рынков с управлением через Telegram. Система собирает данные из множества источников, анализирует рынок с помощью специализированных AI-агентов и исполняет торговые операции на spot/futures рынках множества бирж.

### 1.1 Ключевые характеристики

- **Multi-Agent Architecture**: Специализированные агенты для анализа, риск-менеджмента и исполнения
- **Multi-Provider AI**: Claude, OpenAI, DeepSeek, Gemini через единый интерфейс
- **Multi-Exchange**: Binance, Bybit, OKX, и другие через адаптеры
- **Real-time Data Pipeline**: Сбор market data, sentiment, news, macro, derivatives в реальном времени
- **Telegram-first UX**: Полное управление системой через Telegram бота
- **Risk Engine**: Независимый сервис с circuit breaker и kill switch
- **Self-Evaluation**: Автоматический анализ эффективности стратегий
- **SMC/ICT Tools**: Fair Value Gaps, Order Blocks, Liquidity Zones detection
- **ML-Ready Architecture**: Подготовка к интеграции ML моделей (Phase 2+)

### 1.2 Архитектурные принципы

- **LLM для reasoning** — принятие решений, анализ контекста
- **Rule-based engine** — жёсткие стопы, circuit breakers
- **ML-ready** — архитектура готова к добавлению ML моделей
- **Fail-safe** — при любой ошибке система переходит в безопасный режим

---

## 2. Technology Stack

| Component       | Technology               | Purpose                                     |
| --------------- | ------------------------ | ------------------------------------------- |
| Language        | Go 1.22+                 | Core application                            |
| Agent Framework | Google ADK Go            | Multi-agent orchestration                   |
| Primary DB      | PostgreSQL 16 + pgvector | Users, orders, memory, embeddings           |
| Time-series DB  | ClickHouse               | Market data, OHLCV, indicators              |
| Cache/Locks     | Redis 7                  | Session state, distributed locks, cache     |
| Message Queue   | Apache Kafka             | Event-driven communication, high throughput |
| Bot Interface   | Telegram Bot API         | User interaction                            |
| Config          | envconfig + godotenv     | Environment configuration                   |
| Logging         | Zap                      | Structured logging                          |
| SQL Toolkit     | sqlx                     | Database queries with struct scanning       |
| ML Runtime      | ONNX Runtime             | ML model inference (Phase 2)                |

---

## 2.1 Data Sources

| Source       | Data Type                         | Provider Options                      |
| ------------ | --------------------------------- | ------------------------------------- |
| Market Data  | OHLCV, Tickers, OrderBook         | Binance, Bybit, OKX WebSocket         |
| Liquidations | Real-time liquidations, heatmaps  | Coinglass, Hyblock                    |
| On-Chain     | Flows, MVRV, SOPR, whale tracking | Glassnode, Santiment                  |
| Sentiment    | Social, Fear&Greed                | LunarCrush, Santiment, Alternative.me |
| News         | Crypto news                       | CoinDesk, CoinTelegraph, The Block    |
| Macro        | Economic calendar, Fed rates      | Investing.com, FRED, CME FedWatch     |
| Derivatives  | Options OI, max pain, gamma       | Deribit, Laevitas, Greeks.live        |
| Correlations | BTC vs SPX, DXY, Gold             | Yahoo Finance, TradingView            |

---

## 3. Project Structure

```
prometheus/
├── cmd/
│   └── main.go                      # Single entrypoint
├── internal/
│   ├── adapters/
│   │   ├── config/
│   │   │   └── config.go            # envconfig + dotenv
│   │   ├── postgres/
│   │   │   ├── client.go            # sqlx connection pool
│   │   │   └── migrations/          # SQL migrations
│   │   ├── clickhouse/
│   │   │   └── client.go            # CH connection
│   │   ├── redis/
│   │   │   └── client.go            # Redis client (cache, locks)
│   │   ├── kafka/
│   │   │   ├── producer.go          # Kafka producer
│   │   │   ├── consumer.go          # Kafka consumer
│   │   │   └── topics.go            # Topic definitions
│   │   ├── telegram/
│   │   │   ├── bot.go               # Telegram bot handler
│   │   │   ├── handlers/            # Command handlers
│   │   │   └── keyboards/           # Inline keyboards
│   │   ├── exchanges/
│   │   │   ├── interface.go         # Exchange interface
│   │   │   ├── binance/
│   │   │   │   ├── spot.go
│   │   │   │   └── futures.go
│   │   │   ├── bybit/
│   │   │   │   ├── spot.go
│   │   │   │   └── futures.go
│   │   │   └── factory.go           # Exchange factory
│   │   ├── ai/
│   │   │   ├── interface.go         # AI provider interface
│   │   │   ├── claude/
│   │   │   ├── openai/
│   │   │   ├── deepseek/
│   │   │   ├── gemini/
│   │   │   └── registry.go          # Provider registry
│   │   ├── errors/
│   │   │   ├── interface.go         # Error tracker interface
│   │   │   ├── sentry/
│   │   │   │   └── tracker.go       # Sentry implementation
│   │   │   └── noop/
│   │   │       └── tracker.go       # No-op implementation for testing
│   │   └── datasources/
│   │       ├── news/                # News sources
│   │       │   ├── interface.go     # NewsSource interface
│   │       │   ├── coindesk/
│   │       │   │   └── provider.go
│   │       │   ├── cointelegraph/
│   │       │   │   └── provider.go
│   │       │   └── theblock/
│   │       │       └── provider.go
│   │       ├── sentiment/            # Sentiment sources
│   │       │   ├── interface.go     # SentimentSource interface
│   │       │   ├── santiment/
│   │       │   │   └── provider.go
│   │       │   ├── lunarcrush/
│   │       │   │   └── provider.go
│   │       │   ├── twitter/
│   │       │   │   └── provider.go
│   │       │   └── reddit/
│   │       │       └── provider.go
│   │       ├── onchain/             # On-chain sources
│   │       │   ├── interface.go     # OnChainSource interface
│   │       │   ├── glassnode/
│   │       │   │   └── provider.go
│   │       │   └── santiment/
│   │       │       └── provider.go
│   │       ├── derivatives/         # Derivatives sources
│   │       │   ├── interface.go     # DerivativesSource interface
│   │       │   ├── deribit/
│   │       │   │   └── provider.go
│   │       │   ├── laevitas/
│   │       │   │   └── provider.go
│   │       │   └── greeks_live/
│   │       │       └── provider.go
│   │       ├── liquidations/         # Liquidation sources
│   │       │   ├── interface.go     # LiquidationSource interface
│   │       │   ├── coinglass/
│   │       │   │   └── provider.go
│   │       │   └── hyblock/
│   │       │       └── provider.go
│   │       └── macro/               # Macro sources
│   │           ├── interface.go     # MacroSource interface
│   │           ├── investing/       # Economic calendar
│   │           │   └── provider.go
│   │           ├── fred/            # Federal Reserve data
│   │           │   └── provider.go
│   │           └── fedwatch/        # CME FedWatch
│   │               └── provider.go
│   ├── domain/
│   │   ├── user/
│   │   │   ├── entity.go            # User entity
│   │   │   ├── repository.go        # Repository interface
│   │   │   └── service.go           # Business logic
│   │   ├── exchange_account/
│   │   │   ├── entity.go            # Exchange account entity
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├── trading_pair/
│   │   │   ├── entity.go            # Trading pair config
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├── order/
│   │   │   ├── entity.go            # Order entity
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├── position/
│   │   │   ├── entity.go            # Position entity
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├── market_data/
│   │   │   ├── entity.go            # OHLCV, ticks
│   │   │   ├── repository.go        # ClickHouse repo
│   │   │   └── service.go
│   │   ├── sentiment/
│   │   │   ├── entity.go            # Sentiment data
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── memory/
│   │       ├── entity.go            # Agent memory
│   │       ├── repository.go        # pgvector repo
│   │       └── service.go
│   ├── agents/
│   │   ├── registry.go              # Agent registry
│   │   ├── factory.go               # Agent factory
│   │   ├── market_analyst/
│   │   │   └── agent.go             # Technical analysis agent
│   │   ├── sentiment_analyst/
│   │   │   └── agent.go             # News & social sentiment
│   │   ├── onchain_analyst/
│   │   │   └── agent.go             # On-chain metrics
│   │   ├── correlation_analyst/
│   │   │   └── agent.go             # Cross-market correlation
│   │   ├── macro_analyst/
│   │   │   └── agent.go             # Macro events & Fed analysis
│   │   ├── derivatives_analyst/
│   │   │   └── agent.go             # Options flow & gamma
│   │   ├── orderflow_analyst/
│   │   │   └── agent.go             # Tape reading, CVD, imbalance
│   │   ├── smc_analyst/
│   │   │   └── agent.go             # SMC/ICT: FVG, OB, liquidity
│   │   ├── risk_manager/
│   │   │   └── agent.go             # Risk assessment
│   │   ├── strategy_planner/
│   │   │   └── agent.go             # Trade planning
│   │   ├── executor/
│   │   │   └── agent.go             # Order execution
│   │   ├── position_manager/
│   │   │   └── agent.go             # Position monitoring
│   │   ├── self_evaluator/
│   │   │   └── agent.go             # Strategy performance analysis
│   │   └── orchestrator/
│   │       └── agent.go             # Master coordinator
│   ├── tools/
│   │   ├── registry.go              # Tool registry
│   │   ├── market/
│   │   │   ├── get_price.go
│   │   │   ├── get_ohlcv.go
│   │   │   ├── get_orderbook.go
│   │   │   ├── get_trades.go
│   │   │   ├── get_funding_rate.go
│   │   │   ├── get_open_interest.go
│   │   │   ├── get_long_short_ratio.go
│   │   │   └── get_liquidations.go
│   │   ├── orderflow/
│   │   │   ├── get_trade_imbalance.go    # Buy vs sell pressure
│   │   │   ├── get_cvd.go                # Cumulative Volume Delta
│   │   │   ├── get_whale_trades.go       # Large transactions
│   │   │   ├── get_tick_speed.go         # Trade velocity
│   │   │   ├── get_orderbook_imbalance.go # Bid/ask delta
│   │   │   ├── detect_iceberg.go         # Hidden liquidity
│   │   │   ├── detect_spoofing.go        # Fake orders
│   │   │   └── get_absorption_zones.go   # Order absorption
│   │   ├── indicators/
│   │   │   ├── rsi.go
│   │   │   ├── stochastic.go
│   │   │   ├── cci.go
│   │   │   ├── roc.go
│   │   │   ├── macd.go
│   │   │   ├── bollinger.go
│   │   │   ├── keltner.go
│   │   │   ├── atr.go
│   │   │   ├── ema.go
│   │   │   ├── sma.go
│   │   │   ├── ema_ribbon.go
│   │   │   ├── supertrend.go
│   │   │   ├── vwap.go
│   │   │   ├── obv.go
│   │   │   ├── volume_profile.go
│   │   │   ├── delta_volume.go
│   │   │   ├── fibonacci.go
│   │   │   ├── ichimoku.go
│   │   │   └── pivot_points.go
│   │   ├── smc/                          # Smart Money Concepts
│   │   │   ├── detect_fvg.go             # Fair Value Gaps
│   │   │   ├── detect_order_blocks.go    # Order Blocks
│   │   │   ├── detect_liquidity_zones.go # Liquidity pools
│   │   │   ├── detect_stop_hunt.go       # Stop run detection
│   │   │   ├── get_market_structure.go   # BOS, CHoCH
│   │   │   ├── detect_imbalances.go      # Price imbalances
│   │   │   └── get_swing_points.go       # Swing highs/lows
│   │   ├── sentiment/
│   │   │   ├── get_fear_greed.go
│   │   │   ├── get_social_sentiment.go
│   │   │   ├── get_news.go
│   │   │   ├── get_trending.go
│   │   │   └── get_funding_sentiment.go
│   │   ├── onchain/
│   │   │   ├── get_whale_movements.go
│   │   │   ├── get_exchange_flows.go
│   │   │   ├── get_miner_reserves.go
│   │   │   ├── get_active_addresses.go
│   │   │   ├── get_nvt_ratio.go
│   │   │   ├── get_sopr.go
│   │   │   ├── get_mvrv.go
│   │   │   ├── get_realized_pnl.go
│   │   │   └── get_stablecoin_flows.go
│   │   ├── macro/
│   │   │   ├── get_economic_calendar.go  # CPI, FOMC, NFP dates
│   │   │   ├── get_fed_rate.go           # Current Fed rate
│   │   │   ├── get_fed_watch.go          # Rate probabilities
│   │   │   ├── get_cpi.go                # Inflation data
│   │   │   ├── get_pmi.go                # PMI data
│   │   │   └── get_macro_impact.go       # Historical event impact
│   │   ├── derivatives/
│   │   │   ├── get_options_oi.go         # Options open interest
│   │   │   ├── get_max_pain.go           # Max pain price
│   │   │   ├── get_put_call_ratio.go     # Put/Call ratio
│   │   │   ├── get_gamma_exposure.go     # Dealer gamma
│   │   │   ├── get_options_flow.go       # Large options trades
│   │   │   └── get_iv_surface.go         # Implied volatility
│   │   ├── correlation/
│   │   │   ├── btc_dominance.go
│   │   │   ├── usdt_dominance.go
│   │   │   ├── altcoin_correlation.go
│   │   │   ├── stock_correlation.go      # SPX, Nasdaq
│   │   │   ├── dxy_correlation.go
│   │   │   ├── gold_correlation.go
│   │   │   └── get_session_volume.go     # Asia/EU/US sessions
│   │   ├── trading/
│   │   │   ├── get_balance.go
│   │   │   ├── get_positions.go
│   │   │   ├── place_order.go
│   │   │   ├── place_bracket_order.go    # Entry + SL + TP
│   │   │   ├── place_ladder_order.go     # Multiple TP levels
│   │   │   ├── place_iceberg_order.go    # Hidden size
│   │   │   ├── cancel_order.go
│   │   │   ├── cancel_all_orders.go
│   │   │   ├── close_position.go
│   │   │   ├── close_all_positions.go
│   │   │   ├── set_leverage.go
│   │   │   ├── set_margin_mode.go
│   │   │   ├── move_sl_to_breakeven.go
│   │   │   ├── set_trailing_stop.go
│   │   │   └── add_to_position.go        # DCA / averaging
│   │   ├── risk/
│   │   │   ├── check_circuit_breaker.go  # Should we stop trading?
│   │   │   ├── get_daily_pnl.go
│   │   │   ├── get_max_drawdown.go
│   │   │   ├── get_exposure.go           # Current exposure
│   │   │   ├── calculate_position_size.go
│   │   │   ├── validate_trade.go         # Pre-trade checks
│   │   │   └── emergency_close_all.go    # Kill switch
│   │   ├── memory/
│   │   │   ├── store_memory.go
│   │   │   ├── search_memory.go
│   │   │   ├── get_trade_history.go
│   │   │   ├── get_market_regime.go      # Current regime from memory
│   │   │   └── store_market_regime.go
│   │   └── evaluation/
│   │       ├── get_strategy_stats.go     # Win rate per strategy
│   │       ├── log_trade_decision.go     # Journal entry
│   │       ├── get_trade_journal.go      # Past decisions
│   │       ├── evaluate_last_trades.go   # Performance review
│   │       ├── get_best_strategies.go    # Top performers
│   │       └── get_worst_strategies.go   # Underperformers
│   ├── workers/
│   │   ├── registry.go              # Worker registry
│   │   ├── scheduler.go             # Cron-like scheduler
│   │   ├── market_data/
│   │   │   ├── ohlcv_collector.go   # OHLCV collection
│   │   │   ├── ticker_collector.go  # Real-time tickers
│   │   │   ├── orderbook_collector.go
│   │   │   ├── trades_collector.go  # Tape / trades feed
│   │   │   └── funding_collector.go # Funding rates
│   │   ├── orderflow/
│   │   │   ├── liquidation_collector.go  # Liquidations stream
│   │   │   ├── oi_collector.go           # Open interest
│   │   │   └── whale_alert_collector.go  # Large trades
│   │   ├── sentiment/
│   │   │   ├── news_collector.go    # News aggregation
│   │   │   ├── twitter_collector.go # X.com sentiment
│   │   │   ├── reddit_collector.go  # Reddit sentiment
│   │   │   └── feargreed_collector.go
│   │   ├── onchain/
│   │   │   ├── exchange_flow_collector.go
│   │   │   ├── whale_collector.go
│   │   │   └── metrics_collector.go # MVRV, SOPR, NVT
│   │   ├── macro/
│   │   │   ├── calendar_collector.go    # Economic events
│   │   │   ├── fed_collector.go         # Fed decisions
│   │   │   └── correlation_collector.go # SPX, DXY, Gold
│   │   ├── derivatives/
│   │   │   ├── options_collector.go     # Options data
│   │   │   └── gamma_collector.go       # Gamma exposure
│   │   ├── trading/
│   │   │   ├── position_monitor.go  # Position health check
│   │   │   ├── order_sync.go        # Order status sync
│   │   │   ├── pnl_calculator.go    # PnL calculation
│   │   │   └── risk_monitor.go      # Circuit breaker checks
│   │   ├── analysis/
│   │   │   ├── market_scanner.go    # Periodic market scan
│   │   │   ├── opportunity_finder.go # Trading opportunities
│   │   │   ├── regime_detector.go   # Market regime updates
│   │   │   └── smc_scanner.go       # FVG, OB detection
│   │   └── evaluation/
│   │       ├── daily_report.go      # Daily performance
│   │       ├── strategy_evaluator.go # Strategy analysis
│   │       └── journal_compiler.go  # Trade journal
│   ├── risk/
│   │   ├── engine.go                # Risk engine core
│   │   ├── circuit_breaker.go       # Trading halt logic
│   │   ├── position_limits.go       # Position size limits
│   │   ├── drawdown_monitor.go      # Drawdown tracking
│   │   └── kill_switch.go           # Emergency shutdown
│   ├── ml/                          # ML components (Phase 2)
│   │   ├── regime/
│   │   │   ├── detector.go          # Market regime detection
│   │   │   └── model.onnx           # Trained model
│   │   ├── anomaly/
│   │   │   ├── detector.go          # Anomaly detection
│   │   │   └── model.onnx
│   │   └── inference/
│   │       └── onnx_runtime.go      # ONNX inference wrapper
│   └── repository/
│       ├── postgres/
│       │   ├── user.go
│       │   ├── exchange_account.go
│       │   ├── trading_pair.go
│       │   ├── order.go
│       │   ├── position.go
│       │   └── memory.go
│       └── clickhouse/
│           ├── market_data.go
│           └── sentiment.go
├── pkg/
│   ├── logger/
│   │   └── logger.go                # Zap logger wrapper with error tracking
│   ├── templates/
│   │   ├── loader.go                # Template loader
│   │   ├── registry.go              # Template registry
│   │   └── prompts/                  # .tmpl files
│   │       ├── agents/
│   │       │   ├── market_analyst/
│   │       │   │   ├── system.tmpl
│   │       │   │   └── analysis.tmpl
│   │       │   ├── sentiment_analyst/
│   │       │   │   ├── system.tmpl
│   │       │   │   └── analysis.tmpl
│   │       │   ├── risk_manager/
│   │       │   │   ├── system.tmpl
│   │       │   │   └── assessment.tmpl
│   │       │   ├── executor/
│   │       │   │   ├── system.tmpl
│   │       │   │   └── execution.tmpl
│   │       │   └── orchestrator/
│   │       │       ├── system.tmpl
│   │       │       └── coordination.tmpl
│   │       └── notifications/
│   │           ├── trade_opened.tmpl
│   │           ├── trade_closed.tmpl
│   │           ├── stop_loss_hit.tmpl
│   │           └── daily_report.tmpl
│   ├── crypto/
│   │   └── encryption.go            # AES-256-GCM encryption for API keys
│   ├── clickhouse/
│   │   └── batcher.go                # Batch writer with graceful shutdown
│   ├── decimal/
│   │   └── decimal.go               # Decimal wrapper
│   └── errors/
│       ├── tracker.go                # Error tracking interface
│       └── errors.go                # Custom errors
├── migrations/
│   ├── postgres/
│   │   ├── 001_init.up.sql
│   │   ├── 001_init.down.sql
│   │   └── ...
│   └── clickhouse/
│       ├── 001_init.up.sql
│       └── ...
├── templates/                        # Symlink or copy to pkg/templates/prompts
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
├── go.mod
└── README.md
```

### 3.1 Agent layering with ADK

- **`internal/adapters/adk`** — vendored copy of the Google ADK Go interfaces and reference agents to avoid framework drift. It supplies the minimal agent, tool, and model contracts plus workflow helpers without any business logic.
- **`internal/agents`** — domain-level orchestration that uses the ADK types directly. This layer wires prompts, tool registries, model selection, and middleware, then instantiates concrete analysts/execution agents through factories while relying on the in-repo ADK copy under `internal/adapters/adk`.

---

## 4. Domain Entities

### 4.1 User

```go
// internal/domain/user/entity.go
package user

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID              uuid.UUID  `db:"id"`
    TelegramID      int64      `db:"telegram_id"`
    TelegramUsername string    `db:"telegram_username"`
    FirstName       string     `db:"first_name"`
    LastName        string     `db:"last_name"`
    LanguageCode    string     `db:"language_code"`
    IsActive        bool       `db:"is_active"`
    IsPremium       bool       `db:"is_premium"`
    Settings        Settings   `db:"settings"`
    CreatedAt       time.Time  `db:"created_at"`
    UpdatedAt       time.Time  `db:"updated_at"`
}

type Settings struct {
    DefaultAIProvider  string  `json:"default_ai_provider"`  // claude, openai, etc.
    RiskLevel          string  `json:"risk_level"`           // conservative, moderate, aggressive
    MaxPositions       int     `json:"max_positions"`
    MaxPortfolioRisk   float64 `json:"max_portfolio_risk"`   // percentage
    MaxDailyDrawdown   float64 `json:"max_daily_drawdown"`   // percentage, circuit breaker
    MaxConsecutiveLoss int     `json:"max_consecutive_loss"` // circuit breaker trigger
    NotificationsOn    bool    `json:"notifications_on"`
    DailyReportTime    string  `json:"daily_report_time"`    // HH:MM
    Timezone           string  `json:"timezone"`
    CircuitBreakerOn   bool    `json:"circuit_breaker_on"`
}
```

### 4.2 Exchange Account

```go
// internal/domain/exchange_account/entity.go
package exchange_account

import (
    "time"
    "github.com/google/uuid"
)

type ExchangeAccount struct {
    ID              uuid.UUID     `db:"id"`
    UserID          uuid.UUID     `db:"user_id"`
    Exchange        ExchangeType  `db:"exchange"`        // binance, bybit, okx
    Label           string        `db:"label"`           // "Main Binance", etc.
    APIKeyEncrypted []byte        `db:"api_key_encrypted"`
    SecretEncrypted []byte        `db:"secret_encrypted"`
    Passphrase      []byte        `db:"passphrase"`      // For OKX
    IsTestnet       bool          `db:"is_testnet"`
    Permissions     []string      `db:"permissions"`     // spot, futures, read, trade
    IsActive        bool          `db:"is_active"`
    LastSyncAt      *time.Time    `db:"last_sync_at"`
    CreatedAt       time.Time     `db:"created_at"`
    UpdatedAt       time.Time     `db:"updated_at"`
}

type ExchangeType string

const (
    ExchangeBinance ExchangeType = "binance"
    ExchangeBybit   ExchangeType = "bybit"
    ExchangeOKX     ExchangeType = "okx"
)
```

### 4.3 Trading Pair Configuration

```go
// internal/domain/trading_pair/entity.go
package trading_pair

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type TradingPair struct {
    ID                uuid.UUID       `db:"id"`
    UserID            uuid.UUID       `db:"user_id"`
    ExchangeAccountID uuid.UUID       `db:"exchange_account_id"`
    Symbol            string          `db:"symbol"`           // BTC/USDT
    MarketType        MarketType      `db:"market_type"`      // spot, futures

    // Budget & Risk
    Budget            decimal.Decimal `db:"budget"`           // USDT allocated
    MaxPositionSize   decimal.Decimal `db:"max_position_size"`
    MaxLeverage       int             `db:"max_leverage"`     // For futures
    StopLossPercent   decimal.Decimal `db:"stop_loss_percent"`
    TakeProfitPercent decimal.Decimal `db:"take_profit_percent"`

    // Strategy
    AIProvider        string          `db:"ai_provider"`
    StrategyMode      StrategyMode    `db:"strategy_mode"`    // auto, semi-auto, signals-only
    Timeframes        []string        `db:"timeframes"`       // ["1h", "4h", "1d"]

    // State
    IsActive          bool            `db:"is_active"`
    IsPaused          bool            `db:"is_paused"`
    PausedReason      string          `db:"paused_reason"`

    CreatedAt         time.Time       `db:"created_at"`
    UpdatedAt         time.Time       `db:"updated_at"`
}

type MarketType string
const (
    MarketSpot    MarketType = "spot"
    MarketFutures MarketType = "futures"
)

type StrategyMode string
const (
    StrategyAuto       StrategyMode = "auto"        // Full automation
    StrategySemiAuto   StrategyMode = "semi_auto"   // Needs confirmation
    StrategySignals    StrategyMode = "signals"     // Signals only, no execution
)
```

### 4.4 Order

```go
// internal/domain/order/entity.go
package order

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type Order struct {
    ID                uuid.UUID       `db:"id"`
    UserID            uuid.UUID       `db:"user_id"`
    TradingPairID     uuid.UUID       `db:"trading_pair_id"`
    ExchangeAccountID uuid.UUID       `db:"exchange_account_id"`

    // Exchange data
    ExchangeOrderID   string          `db:"exchange_order_id"`
    Symbol            string          `db:"symbol"`
    MarketType        string          `db:"market_type"`

    // Order details
    Side              OrderSide       `db:"side"`             // buy, sell
    Type              OrderType       `db:"type"`             // market, limit, stop_market, stop_limit
    Status            OrderStatus     `db:"status"`

    // Prices & amounts
    Price             decimal.Decimal `db:"price"`
    Amount            decimal.Decimal `db:"amount"`
    FilledAmount      decimal.Decimal `db:"filled_amount"`
    AvgFillPrice      decimal.Decimal `db:"avg_fill_price"`

    // Stop orders
    StopPrice         decimal.Decimal `db:"stop_price"`

    // Futures specific
    ReduceOnly        bool            `db:"reduce_only"`

    // Metadata
    AgentID           string          `db:"agent_id"`         // Which agent created
    Reasoning         string          `db:"reasoning"`        // AI reasoning
    ParentOrderID     *uuid.UUID      `db:"parent_order_id"`  // For SL/TP orders

    // Fees
    Fee               decimal.Decimal `db:"fee"`
    FeeCurrency       string          `db:"fee_currency"`

    // Timestamps
    CreatedAt         time.Time       `db:"created_at"`
    UpdatedAt         time.Time       `db:"updated_at"`
    FilledAt          *time.Time      `db:"filled_at"`
}

type OrderSide string
const (
    OrderSideBuy  OrderSide = "buy"
    OrderSideSell OrderSide = "sell"
)

type OrderType string
const (
    OrderTypeMarket     OrderType = "market"
    OrderTypeLimit      OrderType = "limit"
    OrderTypeStopMarket OrderType = "stop_market"
    OrderTypeStopLimit  OrderType = "stop_limit"
)

type OrderStatus string
const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusOpen      OrderStatus = "open"
    OrderStatusFilled    OrderStatus = "filled"
    OrderStatusPartial   OrderStatus = "partial"
    OrderStatusCanceled  OrderStatus = "canceled"
    OrderStatusRejected  OrderStatus = "rejected"
    OrderStatusExpired   OrderStatus = "expired"
)
```

### 4.5 Position

```go
// internal/domain/position/entity.go
package position

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type Position struct {
    ID                uuid.UUID       `db:"id"`
    UserID            uuid.UUID       `db:"user_id"`
    TradingPairID     uuid.UUID       `db:"trading_pair_id"`
    ExchangeAccountID uuid.UUID       `db:"exchange_account_id"`

    Symbol            string          `db:"symbol"`
    MarketType        string          `db:"market_type"`
    Side              PositionSide    `db:"side"`             // long, short

    // Size & prices
    Size              decimal.Decimal `db:"size"`
    EntryPrice        decimal.Decimal `db:"entry_price"`
    CurrentPrice      decimal.Decimal `db:"current_price"`
    LiquidationPrice  decimal.Decimal `db:"liquidation_price"` // Futures

    // Leverage (futures)
    Leverage          int             `db:"leverage"`
    MarginMode        string          `db:"margin_mode"`      // cross, isolated

    // PnL
    UnrealizedPnL     decimal.Decimal `db:"unrealized_pnl"`
    UnrealizedPnLPct  decimal.Decimal `db:"unrealized_pnl_pct"`
    RealizedPnL       decimal.Decimal `db:"realized_pnl"`

    // Risk management
    StopLossPrice     decimal.Decimal `db:"stop_loss_price"`
    TakeProfitPrice   decimal.Decimal `db:"take_profit_price"`
    TrailingStopPct   decimal.Decimal `db:"trailing_stop_pct"`

    // Orders
    StopLossOrderID   *uuid.UUID      `db:"stop_loss_order_id"`
    TakeProfitOrderID *uuid.UUID      `db:"take_profit_order_id"`

    // Metadata
    OpenReasoning     string          `db:"open_reasoning"`

    Status            PositionStatus  `db:"status"`
    OpenedAt          time.Time       `db:"opened_at"`
    ClosedAt          *time.Time      `db:"closed_at"`
    UpdatedAt         time.Time       `db:"updated_at"`
}

type PositionSide string
const (
    PositionLong  PositionSide = "long"
    PositionShort PositionSide = "short"
)

type PositionStatus string
const (
    PositionOpen     PositionStatus = "open"
    PositionClosed   PositionStatus = "closed"
    PositionLiquidated PositionStatus = "liquidated"
)
```

### 4.6 Market Data (ClickHouse)

```go
// internal/domain/market_data/entity.go
package market_data

import "time"

type OHLCV struct {
    Exchange   string    `ch:"exchange"`
    Symbol     string    `ch:"symbol"`
    Timeframe  string    `ch:"timeframe"`   // 1m, 5m, 15m, 1h, 4h, 1d
    OpenTime   time.Time `ch:"open_time"`
    Open       float64   `ch:"open"`
    High       float64   `ch:"high"`
    Low        float64   `ch:"low"`
    Close      float64   `ch:"close"`
    Volume     float64   `ch:"volume"`
    QuoteVolume float64  `ch:"quote_volume"`
    Trades     int64     `ch:"trades"`
}

type Ticker struct {
    Exchange      string    `ch:"exchange"`
    Symbol        string    `ch:"symbol"`
    Timestamp     time.Time `ch:"timestamp"`
    Price         float64   `ch:"price"`
    Bid           float64   `ch:"bid"`
    Ask           float64   `ch:"ask"`
    Volume24h     float64   `ch:"volume_24h"`
    Change24h     float64   `ch:"change_24h"`
    High24h       float64   `ch:"high_24h"`
    Low24h        float64   `ch:"low_24h"`
    FundingRate   float64   `ch:"funding_rate"`    // Futures
    OpenInterest  float64   `ch:"open_interest"`   // Futures
}

type OrderBookSnapshot struct {
    Exchange   string    `ch:"exchange"`
    Symbol     string    `ch:"symbol"`
    Timestamp  time.Time `ch:"timestamp"`
    Bids       string    `ch:"bids"`        // JSON array
    Asks       string    `ch:"asks"`        // JSON array
    BidDepth   float64   `ch:"bid_depth"`   // Total bid volume
    AskDepth   float64   `ch:"ask_depth"`   // Total ask volume
}
```

### 4.7 Agent Memory (pgvector)

```go
// internal/domain/memory/entity.go
package memory

import (
    "time"
    "github.com/google/uuid"
    "github.com/pgvector/pgvector-go"
)

type Memory struct {
    ID          uuid.UUID        `db:"id"`
    UserID      uuid.UUID        `db:"user_id"`
    AgentID     string           `db:"agent_id"`
    SessionID   string           `db:"session_id"`

    Type        MemoryType       `db:"type"`          // observation, decision, trade, lesson
    Content     string           `db:"content"`
    Embedding   pgvector.Vector  `db:"embedding"`

    // Metadata
    Symbol      string           `db:"symbol"`
    Timeframe   string           `db:"timeframe"`
    Importance  float64          `db:"importance"`    // 0-1, for retrieval ranking

    // References
    RelatedIDs  []uuid.UUID      `db:"related_ids"`   // Related memories
    TradeID     *uuid.UUID       `db:"trade_id"`      // If trade-related

    CreatedAt   time.Time        `db:"created_at"`
    ExpiresAt   *time.Time       `db:"expires_at"`    // TTL for short-term
}

type MemoryType string
const (
    MemoryObservation MemoryType = "observation"  // Market observation
    MemoryDecision    MemoryType = "decision"     // Trade decision reasoning
    MemoryTrade       MemoryType = "trade"        // Trade outcome
    MemoryLesson      MemoryType = "lesson"       // Learned pattern
    MemoryRegime      MemoryType = "regime"       // Market regime
    MemoryPattern     MemoryType = "pattern"      // Detected pattern
)
```

### 4.8 Trade Journal Entry

```go
// internal/domain/journal/entity.go
package journal

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type JournalEntry struct {
    ID              uuid.UUID       `db:"id"`
    UserID          uuid.UUID       `db:"user_id"`
    TradeID         uuid.UUID       `db:"trade_id"`

    // Trade details
    Symbol          string          `db:"symbol"`
    Side            string          `db:"side"`
    EntryPrice      decimal.Decimal `db:"entry_price"`
    ExitPrice       decimal.Decimal `db:"exit_price"`
    Size            decimal.Decimal `db:"size"`
    PnL             decimal.Decimal `db:"pnl"`
    PnLPercent      decimal.Decimal `db:"pnl_percent"`

    // Strategy info
    StrategyUsed    string          `db:"strategy_used"`
    Timeframe       string          `db:"timeframe"`
    SetupType       string          `db:"setup_type"`       // breakout, reversal, trend_follow

    // Decision context
    MarketRegime    string          `db:"market_regime"`    // trend, range, volatile
    EntryReasoning  string          `db:"entry_reasoning"`
    ExitReasoning   string          `db:"exit_reasoning"`
    ConfidenceScore float64         `db:"confidence_score"`

    // Indicators at entry
    RSIAtEntry      float64         `db:"rsi_at_entry"`
    ATRAtEntry      float64         `db:"atr_at_entry"`
    VolumeAtEntry   float64         `db:"volume_at_entry"`

    // Outcome analysis
    WasCorrectEntry bool            `db:"was_correct_entry"`
    WasCorrectExit  bool            `db:"was_correct_exit"`
    MaxDrawdown     decimal.Decimal `db:"max_drawdown"`
    MaxProfit       decimal.Decimal `db:"max_profit"`
    HoldDuration    time.Duration   `db:"hold_duration"`

    // Lessons learned (AI generated)
    LessonsLearned  string          `db:"lessons_learned"`
    ImprovementTips string          `db:"improvement_tips"`

    CreatedAt       time.Time       `db:"created_at"`
}

// Strategy performance aggregation
type StrategyStats struct {
    StrategyName    string          `db:"strategy_name"`
    TotalTrades     int             `db:"total_trades"`
    WinningTrades   int             `db:"winning_trades"`
    LosingTrades    int             `db:"losing_trades"`
    WinRate         float64         `db:"win_rate"`
    AvgWin          decimal.Decimal `db:"avg_win"`
    AvgLoss         decimal.Decimal `db:"avg_loss"`
    ProfitFactor    decimal.Decimal `db:"profit_factor"`
    ExpectedValue   decimal.Decimal `db:"expected_value"`
    MaxDrawdown     decimal.Decimal `db:"max_drawdown"`
    SharpeRatio     float64         `db:"sharpe_ratio"`
    IsActive        bool            `db:"is_active"`     // Can be disabled if underperforming
    LastUpdated     time.Time       `db:"last_updated"`
}
```

### 4.9 Market Regime

```go
// internal/domain/regime/entity.go
package regime

import (
    "time"
)

type MarketRegime struct {
    Symbol       string      `db:"symbol"`
    Timestamp    time.Time   `db:"timestamp"`
    Regime       RegimeType  `db:"regime"`
    Confidence   float64     `db:"confidence"`
    Volatility   VolLevel    `db:"volatility"`
    Trend        TrendType   `db:"trend"`

    // Supporting metrics
    ATR14        float64     `db:"atr_14"`
    ADX          float64     `db:"adx"`
    BBWidth      float64     `db:"bb_width"`
    Volume24h    float64     `db:"volume_24h"`
    VolumeChange float64     `db:"volume_change"`
}

type RegimeType string
const (
    RegimeTrendUp      RegimeType = "trend_up"
    RegimeTrendDown    RegimeType = "trend_down"
    RegimeRange        RegimeType = "range"
    RegimeBreakout     RegimeType = "breakout"
    RegimeVolatile     RegimeType = "volatile"
    RegimeAccumulation RegimeType = "accumulation"
    RegimeDistribution RegimeType = "distribution"
)

type VolLevel string
const (
    VolLow    VolLevel = "low"
    VolMedium VolLevel = "medium"
    VolHigh   VolLevel = "high"
    VolExtreme VolLevel = "extreme"
)

type TrendType string
const (
    TrendBullish  TrendType = "bullish"
    TrendBearish  TrendType = "bearish"
    TrendNeutral  TrendType = "neutral"
)
```

### 4.10 Macro Event

```go
// internal/domain/macro/entity.go
package macro

import (
    "time"
)

type MacroEvent struct {
    ID           string      `db:"id"`
    EventType    EventType   `db:"event_type"`
    Title        string      `db:"title"`
    Country      string      `db:"country"`
    Currency     string      `db:"currency"`

    // Timing
    EventTime    time.Time   `db:"event_time"`

    // Values
    Previous     string      `db:"previous"`
    Forecast     string      `db:"forecast"`
    Actual       string      `db:"actual"`

    // Impact
    Impact       ImpactLevel `db:"impact"`

    // Historical price reaction
    BTCReaction1h  float64   `db:"btc_reaction_1h"`  // % change 1h after
    BTCReaction24h float64   `db:"btc_reaction_24h"` // % change 24h after

    CollectedAt  time.Time   `db:"collected_at"`
}

type EventType string
const (
    EventCPI          EventType = "cpi"
    EventPPI          EventType = "ppi"
    EventFOMC         EventType = "fomc"
    EventNFP          EventType = "nfp"
    EventGDP          EventType = "gdp"
    EventPMI          EventType = "pmi"
    EventUnemployment EventType = "unemployment"
    EventRetailSales  EventType = "retail_sales"
    EventFedSpeech    EventType = "fed_speech"
)

type ImpactLevel string
const (
    ImpactLow    ImpactLevel = "low"
    ImpactMedium ImpactLevel = "medium"
    ImpactHigh   ImpactLevel = "high"
)
```

### 4.11 Derivatives Data

```go
// internal/domain/derivatives/entity.go
package derivatives

import (
    "time"
)

type OptionsSnapshot struct {
    Symbol        string    `ch:"symbol"`
    Timestamp     time.Time `ch:"timestamp"`

    // Open Interest
    CallOI        float64   `ch:"call_oi"`
    PutOI         float64   `ch:"put_oi"`
    TotalOI       float64   `ch:"total_oi"`
    PutCallRatio  float64   `ch:"put_call_ratio"`

    // Max Pain
    MaxPainPrice  float64   `ch:"max_pain_price"`
    MaxPainDelta  float64   `ch:"max_pain_delta"`  // % from current price

    // Gamma
    GammaExposure float64   `ch:"gamma_exposure"`
    GammaFlip     float64   `ch:"gamma_flip"`      // Price where gamma flips

    // IV
    IVIndex       float64   `ch:"iv_index"`
    IV25dCall     float64   `ch:"iv_25d_call"`
    IV25dPut      float64   `ch:"iv_25d_put"`
    IVSkew        float64   `ch:"iv_skew"`

    // Large trades
    LargeCallVol  float64   `ch:"large_call_vol"`  // $1M+ trades
    LargePutVol   float64   `ch:"large_put_vol"`
}

type OptionsFlow struct {
    ID          string    `ch:"id"`
    Timestamp   time.Time `ch:"timestamp"`
    Symbol      string    `ch:"symbol"`

    Side        string    `ch:"side"`         // call, put
    Strike      float64   `ch:"strike"`
    Expiry      time.Time `ch:"expiry"`
    Premium     float64   `ch:"premium"`      // USD value
    Size        int       `ch:"size"`         // contracts
    Spot        float64   `ch:"spot"`         // Spot price at time

    TradeType   string    `ch:"trade_type"`   // buy, sell
    Sentiment   string    `ch:"sentiment"`    // bullish, bearish
}
```

### 4.12 Liquidation Data

```go
// internal/domain/liquidation/entity.go
package liquidation

import (
    "time"
)

type Liquidation struct {
    Exchange    string    `ch:"exchange"`
    Symbol      string    `ch:"symbol"`
    Timestamp   time.Time `ch:"timestamp"`

    Side        string    `ch:"side"`     // long, short
    Price       float64   `ch:"price"`
    Quantity    float64   `ch:"quantity"`
    Value       float64   `ch:"value"`    // USD
}

type LiquidationHeatmap struct {
    Symbol       string             `json:"symbol"`
    Timestamp    time.Time          `json:"timestamp"`
    Levels       []LiquidationLevel `json:"levels"`
}

type LiquidationLevel struct {
    Price         float64 `json:"price"`
    LongLiqValue  float64 `json:"long_liq_value"`   // USD to be liquidated
    ShortLiqValue float64 `json:"short_liq_value"`
    Leverage      int     `json:"leverage"`         // Estimated leverage
}
```

### 4.13 Circuit Breaker State

```go
// internal/domain/risk/entity.go
package risk

import (
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type CircuitBreakerState struct {
    UserID             uuid.UUID       `db:"user_id"`

    // Current state
    IsTriggered        bool            `db:"is_triggered"`
    TriggeredAt        *time.Time      `db:"triggered_at"`
    TriggerReason      string          `db:"trigger_reason"`

    // Daily stats
    DailyPnL           decimal.Decimal `db:"daily_pnl"`
    DailyPnLPercent    decimal.Decimal `db:"daily_pnl_percent"`
    DailyTradeCount    int             `db:"daily_trade_count"`
    DailyWins          int             `db:"daily_wins"`
    DailyLosses        int             `db:"daily_losses"`
    ConsecutiveLosses  int             `db:"consecutive_losses"`

    // Thresholds (from user settings)
    MaxDailyDrawdown   decimal.Decimal `db:"max_daily_drawdown"`
    MaxConsecutiveLoss int             `db:"max_consecutive_loss"`

    // Auto-reset
    ResetAt            time.Time       `db:"reset_at"`  // Next day 00:00 UTC

    UpdatedAt          time.Time       `db:"updated_at"`
}

type RiskEvent struct {
    ID        uuid.UUID `db:"id"`
    UserID    uuid.UUID `db:"user_id"`
    Timestamp time.Time `db:"timestamp"`

    EventType RiskEventType `db:"event_type"`
    Severity  string        `db:"severity"`  // warning, critical
    Message   string        `db:"message"`
    Data      string        `db:"data"`      // JSON with details

    Acknowledged bool      `db:"acknowledged"`
}

type RiskEventType string
const (
    RiskEventDrawdown       RiskEventType = "drawdown_warning"
    RiskEventConsecutiveLoss RiskEventType = "consecutive_loss"
    RiskEventCircuitBreaker RiskEventType = "circuit_breaker_triggered"
    RiskEventMaxExposure    RiskEventType = "max_exposure_reached"
    RiskEventKillSwitch     RiskEventType = "kill_switch_activated"
    RiskEventAnomalyDetected RiskEventType = "anomaly_detected"
)
```

---

## 5. Configuration

### 5.1 Environment Configuration

```go
// internal/adapters/config/config.go
package config

import (
    "fmt"
    "github.com/joho/godotenv"
    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    App         AppConfig
    Postgres    PostgresConfig
    ClickHouse  ClickHouseConfig
    Redis       RedisConfig
    Telegram    TelegramConfig
    AI          AIConfig
    Crypto      CryptoConfig
    MarketData  MarketDataConfig
    ErrorTracking ErrorTrackingConfig
}

type AppConfig struct {
    Name        string `envconfig:"APP_NAME" default:"prometheus"`
    Env         string `envconfig:"APP_ENV" default:"development"`
    LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
    Debug       bool   `envconfig:"DEBUG" default:"false"`
}

type PostgresConfig struct {
    Host     string `envconfig:"POSTGRES_HOST" required:"true"`
    Port     int    `envconfig:"POSTGRES_PORT" default:"5432"`
    User     string `envconfig:"POSTGRES_USER" required:"true"`
    Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
    Database string `envconfig:"POSTGRES_DB" required:"true"`
    SSLMode  string `envconfig:"POSTGRES_SSL_MODE" default:"disable"`
    MaxConns int    `envconfig:"POSTGRES_MAX_CONNS" default:"25"`
}

func (c PostgresConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
    )
}

type ClickHouseConfig struct {
    Host     string `envconfig:"CLICKHOUSE_HOST" required:"true"`
    Port     int    `envconfig:"CLICKHOUSE_PORT" default:"9000"`
    User     string `envconfig:"CLICKHOUSE_USER" default:"default"`
    Password string `envconfig:"CLICKHOUSE_PASSWORD"`
    Database string `envconfig:"CLICKHOUSE_DB" default:"trading"`
}

type RedisConfig struct {
    Host     string `envconfig:"REDIS_HOST" required:"true"`
    Port     int    `envconfig:"REDIS_PORT" default:"6379"`
    Password string `envconfig:"REDIS_PASSWORD"`
    DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func (c RedisConfig) Addr() string {
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type KafkaConfig struct {
    Brokers []string `envconfig:"KAFKA_BROKERS" required:"true"`
    GroupID string   `envconfig:"KAFKA_GROUP_ID" default:"prometheus"`
}

type TelegramConfig struct {
    BotToken string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`
    WebhookURL string `envconfig:"TELEGRAM_WEBHOOK_URL"`
    AdminIDs []int64 `envconfig:"TELEGRAM_ADMIN_IDS"`
}

type AIConfig struct {
    ClaudeKey string `envconfig:"CLAUDE_API_KEY"`
    OpenAIKey    string `envconfig:"OPENAI_API_KEY"`
    DeepSeekKey  string `envconfig:"DEEPSEEK_API_KEY"`
    GeminiKey    string `envconfig:"GEMINI_API_KEY"`
    DefaultProvider string `envconfig:"DEFAULT_AI_PROVIDER" default:"claude"`
}

type CryptoConfig struct {
    EncryptionKey string `envconfig:"ENCRYPTION_KEY" required:"true"` // 32 bytes for AES-256
}

// Exchange-specific market data credentials
type BinanceMarketDataConfig struct {
    APIKey string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
    Secret string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
}

type BybitMarketDataConfig struct {
    APIKey string `envconfig:"BYBIT_MARKET_DATA_API_KEY"`
    Secret string `envconfig:"BYBIT_MARKET_DATA_SECRET"`
}

type OKXMarketDataConfig struct {
    APIKey     string `envconfig:"OKX_MARKET_DATA_API_KEY"`
    Secret     string `envconfig:"OKX_MARKET_DATA_SECRET"`
    Passphrase string `envconfig:"OKX_MARKET_DATA_PASSPHRASE"`
}

type MarketDataConfig struct {
    // Central API keys for market data collection (not user-specific)
    Binance BinanceMarketDataConfig
    Bybit   BybitMarketDataConfig
    OKX     OKXMarketDataConfig
}

type ErrorTrackingConfig struct {
    Enabled     bool   `envconfig:"ERROR_TRACKING_ENABLED" default:"true"`
    Provider    string `envconfig:"ERROR_TRACKING_PROVIDER" default:"sentry"`
    SentryDSN   string `envconfig:"SENTRY_DSN"`
    Environment string `envconfig:"SENTRY_ENVIRONMENT" default:"production"`
}

func Load() (*Config, error) {
    // Load .env for local development (ignore error if not exists)
    _ = godotenv.Load()

    var cfg Config
    if err := envconfig.Process("", &cfg); err != nil {
        return nil, fmt.Errorf("failed to process env config: %w", err)
    }

    return &cfg, nil
}
```

### 5.2 Example .env

```bash
# .env.example

# App
APP_NAME=prometheus
APP_ENV=development
LOG_LEVEL=debug
DEBUG=true

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=trading
POSTGRES_PASSWORD=secret
POSTGRES_DB=prometheus
POSTGRES_SSL_MODE=disable
POSTGRES_MAX_CONNS=25

# ClickHouse
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=
CLICKHOUSE_DB=trading

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=prometheus

# Telegram
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_WEBHOOK_URL=https://your-domain.com/webhook
TELEGRAM_ADMIN_IDS=123456789,987654321

# AI Providers
CLAUDE_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
DEEPSEEK_API_KEY=sk-...
GEMINI_API_KEY=...
DEFAULT_AI_PROVIDER=claude

# Encryption (32 bytes base64 encoded)
ENCRYPTION_KEY=your-32-byte-encryption-key-here

# Central Market Data API Keys (for data collection, not user trading)
BINANCE_MARKET_DATA_API_KEY=your_binance_api_key
BINANCE_MARKET_DATA_SECRET=your_binance_secret
BYBIT_MARKET_DATA_API_KEY=your_bybit_api_key
BYBIT_MARKET_DATA_SECRET=your_bybit_secret
OKX_MARKET_DATA_API_KEY=your_okx_api_key
OKX_MARKET_DATA_SECRET=your_okx_secret
OKX_MARKET_DATA_PASSPHRASE=your_okx_passphrase

# Error Tracking
ERROR_TRACKING_ENABLED=true
ERROR_TRACKING_PROVIDER=sentry
SENTRY_DSN=https://your-sentry-dsn@sentry.io/project-id
SENTRY_ENVIRONMENT=production
```

---

## 6. Core Packages

### 6.1 Logger

```go
// pkg/logger/logger.go
package logger

import (
    "os"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var globalLogger *zap.SugaredLogger

type Logger struct {
    *zap.SugaredLogger
}

func Init(level string, env string) error {
    var config zap.Config

    if env == "production" {
        config = zap.NewProductionConfig()
    } else {
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    }

    // Parse level
    var zapLevel zapcore.Level
    if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
        zapLevel = zapcore.InfoLevel
    }
    config.Level = zap.NewAtomicLevelAt(zapLevel)

    logger, err := config.Build(
        zap.AddCallerSkip(1),
        zap.AddStacktrace(zapcore.ErrorLevel),
    )
    if err != nil {
        return err
    }

    globalLogger = logger.Sugar()
    return nil
}

func Get() *Logger {
    if globalLogger == nil {
        // Fallback to basic logger
        logger, _ := zap.NewDevelopment()
        globalLogger = logger.Sugar()
    }
    return &Logger{globalLogger}
}

func (l *Logger) With(args ...interface{}) *Logger {
    return &Logger{l.SugaredLogger.With(args...)}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
    args := make([]interface{}, 0, len(fields)*2)
    for k, v := range fields {
        args = append(args, k, v)
    }
    return &Logger{l.SugaredLogger.With(args...)}
}

// Convenience functions
func Debug(args ...interface{})              { Get().Debug(args...) }
func Debugf(tpl string, args ...interface{}) { Get().Debugf(tpl, args...) }
func Info(args ...interface{})               { Get().Info(args...) }
func Infof(tpl string, args ...interface{})  { Get().Infof(tpl, args...) }
func Warn(args ...interface{})               { Get().Warn(args...) }
func Warnf(tpl string, args ...interface{})  { Get().Warnf(tpl, args...) }
func Error(args ...interface{})              { Get().Error(args...) }
func Errorf(tpl string, args ...interface{}) { Get().Errorf(tpl, args...) }
func Fatal(args ...interface{})              { Get().Fatal(args...) }
func Fatalf(tpl string, args ...interface{}) { Get().Fatalf(tpl, args...) }

func Sync() error {
    if globalLogger != nil {
        return globalLogger.Sync()
    }
    return nil
}
```

### 6.2 Template System

```go
// pkg/templates/loader.go
package templates

import (
    "bytes"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "text/template"
)

type Registry struct {
    templates map[string]*template.Template
    mu        sync.RWMutex
    basePath  string
}

var globalRegistry *Registry

// Init loads all templates from the given directory.
// Must be called in main.go before any other initialization.
// Panics if templates cannot be loaded.
func Init(basePath string) {
    registry, err := loadTemplates(basePath)
    if err != nil {
        panic(fmt.Sprintf("failed to load templates: %v", err))
    }
    globalRegistry = registry
}

// Get returns the global template registry.
// Panics if Init() was not called.
func Get() *Registry {
    if globalRegistry == nil {
        panic("templates not initialized: call templates.Init() in main.go")
    }
    return globalRegistry
}

func loadTemplates(basePath string) (*Registry, error) {
    registry := &Registry{
        templates: make(map[string]*template.Template),
        basePath:  basePath,
    }

    err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() || !strings.HasSuffix(path, ".tmpl") {
            return nil
        }

        // Create template ID from relative path without extension
        relPath, err := filepath.Rel(basePath, path)
        if err != nil {
            return err
        }

        id := strings.TrimSuffix(relPath, ".tmpl")
        id = strings.ReplaceAll(id, string(os.PathSeparator), "/")

        content, err := os.ReadFile(path)
        if err != nil {
            return fmt.Errorf("failed to read template %s: %w", path, err)
        }

        tmpl, err := template.New(id).Parse(string(content))
        if err != nil {
            return fmt.Errorf("failed to parse template %s: %w", path, err)
        }

        registry.templates[id] = tmpl
        return nil
    })

    if err != nil {
        return nil, err
    }

    if len(registry.templates) == 0 {
        return nil, fmt.Errorf("no templates found in %s", basePath)
    }

    return registry, nil
}

// Lookup returns a Template wrapper for the given ID.
// Returns nil if template not found.
func (r *Registry) Lookup(id string) *Template {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tmpl, ok := r.templates[id]
    if !ok {
        return nil
    }
    return &Template{tmpl: tmpl, id: id}
}

// MustLookup returns a Template wrapper or panics if not found.
func (r *Registry) MustLookup(id string) *Template {
    t := r.Lookup(id)
    if t == nil {
        panic(fmt.Sprintf("template not found: %s", id))
    }
    return t
}

// List returns all registered template IDs.
func (r *Registry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    ids := make([]string, 0, len(r.templates))
    for id := range r.templates {
        ids = append(ids, id)
    }
    return ids
}

// Template wrapper for convenient rendering
type Template struct {
    tmpl *template.Template
    id   string
}

// Render executes the template with the given data and returns the result.
func (t *Template) Render(data interface{}) (string, error) {
    var buf bytes.Buffer
    if err := t.tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to render template %s: %w", t.id, err)
    }
    return buf.String(), nil
}

// MustRender renders the template or panics on error.
func (t *Template) MustRender(data interface{}) string {
    result, err := t.Render(data)
    if err != nil {
        panic(err)
    }
    return result
}
```

### 6.3 Example Prompt Templates

```tmpl
{{/* pkg/templates/prompts/agents/market_analyst/system.tmpl */}}

You are an expert cryptocurrency market analyst specializing in {{.Symbol}} trading.
Your role is to provide thorough technical and fundamental analysis using Chain of Thought reasoning.

## Your Capabilities:
- Technical analysis across multiple timeframes ({{range .Timeframes}}{{.}}, {{end}})
- Order book analysis for support/resistance and liquidity
- Indicator interpretation (RSI, MACD, Bollinger Bands, etc.)
- Volume analysis and market structure identification

## Analysis Framework:

**STEP 1 - DATA GATHERING:**
Use available tools to collect:
- Current price, volume, 24h change
- Order book depth and liquidity zones
- Technical indicators for each timeframe
- Funding rates and open interest (futures)

**STEP 2 - TECHNICAL ANALYSIS:**
For each timeframe, identify:
- Trend direction and strength
- Key support/resistance levels
- Chart patterns
- Divergences

**STEP 3 - SYNTHESIS:**
Combine all data into actionable insight:
- Primary trend direction
- Confidence level (0-100%)
- Recommended action: {{.AllowedActions}}
- Key levels to watch

## Output Format:
Always respond with valid JSON matching the MarketAnalysis schema.

## Risk Profile: {{.RiskLevel}}
{{if eq .RiskLevel "conservative"}}
- Only high-confidence setups (>80%)
- Prefer trend-following over reversals
{{else if eq .RiskLevel "aggressive"}}
- Accept moderate confidence (>60%)
- Consider counter-trend opportunities
{{end}}
```

```tmpl
{{/* pkg/templates/prompts/agents/risk_manager/assessment.tmpl */}}

Evaluate the following trade proposal:

## Market Analysis:
{{.MarketAnalysis}}

## Current Portfolio State:
- Open Positions: {{len .OpenPositions}}
- Total Exposure: {{.TotalExposure}} USDT
- Available Balance: {{.AvailableBalance}} USDT
- Current Portfolio Risk: {{.PortfolioRisk}}%

## User Risk Parameters:
- Max Position Size: {{.MaxPositionSize}} USDT
- Max Portfolio Risk: {{.MaxPortfolioRisk}}%
- Max Concurrent Positions: {{.MaxPositions}}
- Stop Loss Requirement: {{.StopLossPercent}}%

## Validation Checklist:
1. Market Analysis Confidence >= {{.MinConfidence}}%?
2. Risk/Reward Ratio >= {{.MinRiskReward}}?
3. Stop Loss <= {{.MaxStopLoss}}%?
4. Would not exceed position limits?
5. Would not exceed portfolio risk limits?

Provide your risk assessment as JSON with:
- approved: boolean
- position_size: recommended size in USDT
- stop_loss: price level
- take_profit: array of price levels
- risk_reward_ratio: calculated R:R
- reasoning: detailed explanation
```

```tmpl
{{/* pkg/templates/prompts/notifications/trade_opened.tmpl */}}

🟢 *Trade Opened*

*Symbol:* `{{.Symbol}}`
*Side:* {{if eq .Side "long"}}🟢 LONG{{else}}🔴 SHORT{{end}}
*Type:* {{.MarketType}}

*Entry Price:* `{{.EntryPrice}}`
*Position Size:* `{{.PositionSize}} {{.BaseCurrency}}`
*Value:* `{{.PositionValue}} USDT`

*Stop Loss:* `{{.StopLoss}}` ({{.StopLossPercent}}%)
*Take Profit:* `{{.TakeProfit}}` ({{.TakeProfitPercent}}%)
*R:R Ratio:* `{{.RiskRewardRatio}}`

{{if .Leverage}}*Leverage:* `{{.Leverage}}x`{{end}}

📊 *Analysis Summary:*
{{.Reasoning}}

_Opened at {{.Timestamp}}_
```

---

## 7. Agent Architecture

### 7.1 Agent Roles & Responsibilities

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ORCHESTRATOR                                    │
│        Coordinates all agents, manages workflow, makes final decisions       │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
          ┌───────────────────────────┼───────────────────────────┐
          │                           │                           │
          ▼                           ▼                           ▼
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│ PARALLEL BLOCK 1│         │ PARALLEL BLOCK 2│         │ PARALLEL BLOCK 3│
│ Market Analysis │         │ External Factors│         │  Flow Analysis  │
└─────────────────┘         └─────────────────┘         └─────────────────┘
         │                           │                           │
    ┌────┴────┐                 ┌────┴────┐                 ┌────┴────┐
    ▼         ▼                 ▼         ▼                 ▼         ▼
┌───────┐ ┌───────┐       ┌───────┐ ┌───────┐       ┌───────┐ ┌───────┐
│MARKET │ │  SMC  │       │SENTMNT│ │ MACRO │       │ORDERFL│ │DERIVAT│
│ANALYST│ │ANALYST│       │ANALYST│ │ANALYST│       │ANALYST│ │ANALYST│
└───────┘ └───────┘       └───────┘ └───────┘       └───────┘ └───────┘
    │         │                 │         │                 │         │
    └────┬────┘                 └────┬────┘                 └────┬────┘
         │                           │                           │
         └───────────────────────────┼───────────────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │   CORRELATION     │
                          │     ANALYST       │
                          │                   │
                          │ - BTC dominance   │
                          │ - DXY, SPX, Gold  │
                          │ - Session analysis│
                          └───────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │    ONCHAIN        │
                          │    ANALYST        │
                          │                   │
                          │ - Whale moves     │
                          │ - Exchange flows  │
                          │ - MVRV, SOPR      │
                          └───────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │    STRATEGY       │
                          │     PLANNER       │
                          │                   │
                          │ - Synthesize all  │
                          │ - Trade thesis    │
                          │ - Entry/exit plan │
                          └───────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │      RISK         │
                          │    MANAGER        │
                          │                   │
                          │ - Position sizing │
                          │ - Validate trade  │
                          │ - Check limits    │
                          └───────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │     EXECUTOR      │
                          │                   │
                          │ - Place orders    │
                          │ - Set SL/TP       │
                          │ - Bracket orders  │
                          └───────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │    POSITION       │
                          │    MANAGER        │
                          │                   │
                          │ - Monitor PnL     │
                          │ - Trailing stops  │
                          │ - Scale out       │
                          └───────────────────┘
                                     │
                                     ▼
                          ┌───────────────────┐
                          │  SELF EVALUATOR   │
                          │  (Daily/Weekly)   │
                          │                   │
                          │ - Analyze trades  │
                          │ - Strategy stats  │
                          │ - Disable bad     │
                          └───────────────────┘
```

### 7.2 Agent Definitions

| Agent                   | Role                                     | Key Tools                                                                                                 | Output                 |
| ----------------------- | ---------------------------------------- | --------------------------------------------------------------------------------------------------------- | ---------------------- |
| **Market Analyst**      | Technical analysis, indicators, patterns | `get_ohlcv`, `rsi`, `macd`, `bollinger`, `atr`, `volume_profile`, `get_swing_points`                      | `market_analysis`      |
| **SMC Analyst**         | Smart Money Concepts, liquidity          | `detect_fvg`, `detect_order_blocks`, `detect_liquidity_zones`, `detect_stop_hunt`, `get_market_structure` | `smc_analysis`         |
| **Sentiment Analyst**   | News, social, Fear&Greed                 | `get_news`, `get_social_sentiment`, `get_fear_greed`, `get_trending`                                      | `sentiment_analysis`   |
| **Macro Analyst**       | Economic events, Fed, correlations       | `get_economic_calendar`, `get_fed_watch`, `get_cpi`, `get_macro_impact`                                   | `macro_analysis`       |
| **Order Flow Analyst**  | Tape reading, CVD, imbalance             | `get_cvd`, `get_trade_imbalance`, `get_whale_trades`, `get_orderbook_imbalance`, `detect_iceberg`         | `orderflow_analysis`   |
| **Derivatives Analyst** | Options, gamma, max pain                 | `get_options_oi`, `get_max_pain`, `get_gamma_exposure`, `get_put_call_ratio`, `get_options_flow`          | `derivatives_analysis` |
| **Correlation Analyst** | Cross-market analysis                    | `btc_dominance`, `stock_correlation`, `dxy_correlation`, `gold_correlation`, `get_session_volume`         | `correlation_analysis` |
| **On-Chain Analyst**    | Blockchain metrics                       | `get_whale_movements`, `get_exchange_flows`, `get_mvrv`, `get_sopr`, `get_stablecoin_flows`               | `onchain_analysis`     |
| **Strategy Planner**    | Synthesize all, create plan              | `search_memory`, `get_trade_history`, `get_market_regime`                                                 | `trade_plan`           |
| **Risk Manager**        | Validate, size positions                 | `get_positions`, `get_balance`, `calculate_position_size`, `validate_trade`, `check_circuit_breaker`      | `risk_assessment`      |
| **Executor**            | Execute trades                           | `place_bracket_order`, `place_ladder_order`, `set_leverage`, `set_trailing_stop`                          | `execution_result`     |
| **Position Manager**    | Manage open positions                    | `get_positions`, `move_sl_to_breakeven`, `set_trailing_stop`, `close_position`, `add_to_position`         | `position_update`      |
| **Self Evaluator**      | Analyze performance                      | `get_strategy_stats`, `get_trade_journal`, `evaluate_last_trades`, `get_worst_strategies`                 | `evaluation_report`    |

```go
// internal/agents/registry.go
package agents

import (
    "context"
    "google.golang.org/adk/agent"
)

type AgentType string

const (
    // Personal trading workflow agents (per-user)
    AgentStrategyPlanner AgentType = "strategy_planner"
    AgentRiskManager     AgentType = "risk_manager"
    AgentExecutor        AgentType = "executor"
    AgentPositionManager AgentType = "position_manager"
    AgentSelfEvaluator   AgentType = "self_evaluator"

    // Market research workflow agent (global)
    AgentOpportunitySynthesizer AgentType = "opportunity_synthesizer"

    // Portfolio management agent (onboarding)
    AgentPortfolioArchitect AgentType = "portfolio_architect"
)

type AgentConfig struct {
    Type        AgentType
    Name        string
    Description string
    AIProvider  string // claude, openai, etc.
    Model       string // claude-3-opus, gpt-4, etc.
    Tools       []string
    OutputKey   string
    SystemPromptTemplate string
}

type Registry struct {
    agents map[AgentType]agent.Agent
    configs map[AgentType]AgentConfig
}

func NewRegistry() *Registry {
    return &Registry{
        agents:  make(map[AgentType]agent.Agent),
        configs: make(map[AgentType]AgentConfig),
    }
}

func (r *Registry) Register(agentType AgentType, a agent.Agent, cfg AgentConfig) {
    r.agents[agentType] = a
    r.configs[agentType] = cfg
}

func (r *Registry) Get(agentType AgentType) (agent.Agent, bool) {
    a, ok := r.agents[agentType]
    return a, ok
}

func (r *Registry) MustGet(agentType AgentType) agent.Agent {
    a, ok := r.Get(agentType)
    if !ok {
        panic("agent not found: " + string(agentType))
    }
    return a
}
```

### 7.3 Agent Factory

````go
// internal/agents/factory.go
package agents

import (
    "fmt"

    "google.golang.org/adk/agent"
    "google.golang.org/adk/agent/llmagent"
    "google.golang.org/adk/agent/workflowagents/parallelagent"
    "google.golang.org/adk/agent/workflowagents/sequentialagent"
    "google.golang.org/adk/tool"

    "prometheus/internal/adapters/ai"
    "prometheus/internal/tools"
    "prometheus/pkg/templates"
)

type Factory struct {
    aiRegistry   ai.ProviderRegistry
    toolRegistry *tools.Registry
    templates    *templates.Registry
}

type FactoryDeps struct {
    AIRegistry   ai.ProviderRegistry
    ToolRegistry *tools.Registry
    Templates    *templates.Registry
}

func NewFactory(deps FactoryDeps) *Factory {
    return &Factory{
        aiRegistry:   deps.AIRegistry,
        toolRegistry: deps.ToolRegistry,
        templates:    deps.Templates,
    }
}

func (f *Factory) CreateAgent(cfg AgentConfig) (agent.Agent, error) {
    // Get AI model
    model, err := f.aiRegistry.GetModel(cfg.AIProvider, cfg.Model)
    if err != nil {
        return nil, fmt.Errorf("failed to get model: %w", err)
    }

    // Get tools
    agentTools := make([]tool.Tool, 0, len(cfg.Tools))
    for _, toolName := range cfg.Tools {
        t, ok := f.toolRegistry.Get(toolName)
        if !ok {
            return nil, fmt.Errorf("tool not found: %s", toolName)
        }
        agentTools = append(agentTools, t)
    }

    // Render system prompt
    systemPrompt := ""
    if cfg.SystemPromptTemplate != "" {
        tmpl := f.templates.Lookup(cfg.SystemPromptTemplate)
        if tmpl == nil {
            return nil, fmt.Errorf("template not found: %s", cfg.SystemPromptTemplate)
        }
        // Render with default data, can be overridden at runtime
        systemPrompt, _ = tmpl.Render(map[string]interface{}{})
    }

    // Create LLM agent
    return llmagent.New(llmagent.Config{
        Name:        cfg.Name,
        Description: cfg.Description,
        Model:       model,
        Tools:       agentTools,
        Instruction: systemPrompt,
        OutputKey:   cfg.OutputKey,
    })
}

// CreateMarketResearchWorkflow creates the simplified market research workflow
// This workflow uses OpportunitySynthesizer which calls algorithmic analysis tools directly
func (f *Factory) CreateMarketResearchWorkflow() (agent.Agent, error) {
    // Single agent: OpportunitySynthesizer calls comprehensive analysis tools directly
    // No intermediate analyst agents needed - tools return algorithmic analysis
    synthesizerAgent, _ := f.CreateAgent(AgentConfig{
        Type:        AgentOpportunitySynthesizer,
        Name:        "OpportunitySynthesizer",
        AIProvider:  "openai",
        Model:       "gpt-4o-mini",
        Tools:       []string{"get_technical_analysis", "get_smc_analysis", "get_market_analysis", "publish_opportunity", "save_memory"},
        OutputKey:   "opportunity_decision",
        SystemPromptTemplate: "agents/opportunity_synthesizer",
        MaxToolCalls: 10,
    })

    return synthesizerAgent, nil
}

// CreatePersonalTradingWorkflow creates the per-user trading workflow
func (f *Factory) CreatePersonalTradingWorkflow(userCfg UserAgentConfig) (agent.Agent, error) {
    // Strategy planner
    strategyPlanner, _ := f.CreateAgent(AgentConfig{
        Type:        AgentStrategyPlanner,
        Name:        "StrategyPlanner",
        AIProvider:  userCfg.AIProvider,
        Model:       userCfg.Model,
        Tools:       []string{"get_trade_history", "search_memory"},
        OutputKey:   "trade_plan",
        SystemPromptTemplate: "agents/strategy_planner/system",
    })

    // Risk manager
    riskManager, _ := f.CreateAgent(AgentConfig{
        Type:        AgentRiskManager,
        Name:        "RiskManager",
        AIProvider:  userCfg.AIProvider,
        Model:       userCfg.Model,
        Tools:       []string{"get_positions", "get_balance"},
        OutputKey:   "risk_assessment",
        SystemPromptTemplate: "agents/risk_manager/system",
    })

    // Executor
    executor, _ := f.CreateAgent(AgentConfig{
        Type:        AgentExecutor,
        Name:        "Executor",
        AIProvider:  userCfg.AIProvider,
        Model:       userCfg.Model,
        Tools:       []string{"place_order", "cancel_order", "set_leverage", "set_margin_mode"},
        OutputKey:   "execution_result",
        SystemPromptTemplate: "agents/executor/system",
    })

    // Full pipeline: ParallelAnalysis -> Correlation -> Strategy -> Risk -> Executor
    fullPipeline, err := sequentialagent.New(sequentialagent.Config{
        AgentConfig: agent.Config{
            Name:        "TradingPipeline",
            Description: "Complete trading analysis and execution workflow",
            SubAgents: []agent.Agent{
                parallelAnalysis,
                correlationAnalyst,
                strategyPlanner,
                riskManager,
                executor,
            },
        },
    })

    return fullPipeline, err
}

type UserAgentConfig struct {
    UserID     string
    AIProvider string
    Model      string
    Symbol     string
    MarketType string
    RiskLevel  string
}

---

## 8. Tools Registry

### 8.1 Tool Interface & Registry

```go
// internal/tools/registry.go
package tools

import (
    "sync"
    "google.golang.org/adk/tool"
)

type Registry struct {
    tools map[string]tool.Tool
    mu    sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]tool.Tool),
    }
}

func (r *Registry) Register(name string, t tool.Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[name] = t
}

func (r *Registry) Get(name string) (tool.Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    t, ok := r.tools[name]
    return t, ok
}

func (r *Registry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    names := make([]string, 0, len(r.tools))
    for name := range r.tools {
        names = append(names, name)
    }
    return names
}
````

### 8.2 Complete Tool List

#### Market Data Tools

| Tool Name              | Description                        |
| ---------------------- | ---------------------------------- |
| `get_price`            | Current price, bid/ask spread      |
| `get_ohlcv`            | OHLCV candles for any timeframe    |
| `get_orderbook`        | Order book with configurable depth |
| `get_trades`           | Recent trades / tape               |
| `get_funding_rate`     | Futures funding rate               |
| `get_open_interest`    | Open interest data                 |
| `get_long_short_ratio` | Long/short account ratio           |
| `get_liquidations`     | Recent liquidations                |

#### Order Flow Tools

| Tool Name                 | Description                  |
| ------------------------- | ---------------------------- |
| `get_trade_imbalance`     | Buy vs sell pressure (delta) |
| `get_cvd`                 | Cumulative Volume Delta      |
| `get_whale_trades`        | Large transactions (>$100k)  |
| `get_tick_speed`          | Trade velocity / momentum    |
| `get_orderbook_imbalance` | Bid/ask volume delta         |
| `detect_iceberg`          | Hidden liquidity detection   |
| `detect_spoofing`         | Fake order detection         |
| `get_absorption_zones`    | Where orders get absorbed    |

#### Technical Indicators - Momentum

| Tool Name    | Description             |
| ------------ | ----------------------- |
| `rsi`        | Relative Strength Index |
| `stochastic` | Stochastic oscillator   |
| `cci`        | Commodity Channel Index |
| `roc`        | Rate of Change          |

#### Technical Indicators - Volatility

| Tool Name   | Description        |
| ----------- | ------------------ |
| `atr`       | Average True Range |
| `bollinger` | Bollinger Bands    |
| `keltner`   | Keltner Channels   |

#### Technical Indicators - Trend

| Tool Name    | Description                  |
| ------------ | ---------------------------- |
| `sma`        | Simple Moving Average        |
| `ema`        | Exponential Moving Average   |
| `ema_ribbon` | EMA ribbon (multiple EMAs)   |
| `macd`       | MACD with signal & histogram |
| `supertrend` | Supertrend indicator         |
| `ichimoku`   | Ichimoku Cloud               |

#### Technical Indicators - Volume

| Tool Name        | Description                   |
| ---------------- | ----------------------------- |
| `vwap`           | Volume Weighted Average Price |
| `obv`            | On-Balance Volume             |
| `volume_profile` | Volume profile / POC          |
| `delta_volume`   | Buy/sell volume delta         |

#### Smart Money Concepts (SMC/ICT)

| Tool Name                | Description                         |
| ------------------------ | ----------------------------------- |
| `detect_fvg`             | Fair Value Gaps                     |
| `detect_order_blocks`    | Order Blocks (bullish/bearish)      |
| `detect_liquidity_zones` | Liquidity pools (stop clusters)     |
| `detect_stop_hunt`       | Stop run / liquidity grab detection |
| `get_market_structure`   | Break of Structure (BOS), CHoCH     |
| `detect_imbalances`      | Price imbalances                    |
| `get_swing_points`       | Swing highs and lows                |

#### Sentiment Tools

| Tool Name               | Description                       |
| ----------------------- | --------------------------------- |
| `get_fear_greed`        | Fear & Greed Index                |
| `get_social_sentiment`  | Twitter/Reddit/Telegram sentiment |
| `get_news`              | Latest crypto news                |
| `get_trending`          | Trending coins/topics             |
| `get_funding_sentiment` | Sentiment from funding rates      |

#### On-Chain Tools

| Tool Name              | Description                    |
| ---------------------- | ------------------------------ |
| `get_whale_movements`  | Large wallet transfers         |
| `get_exchange_flows`   | Exchange inflow/outflow        |
| `get_miner_reserves`   | Miner holdings                 |
| `get_active_addresses` | Active address count           |
| `get_nvt_ratio`        | Network Value to Transactions  |
| `get_sopr`             | Spent Output Profit Ratio      |
| `get_mvrv`             | Market Value to Realized Value |
| `get_realized_pnl`     | Net Realized Profit/Loss       |
| `get_stablecoin_flows` | USDT/USDC flows to exchanges   |

#### Macro Tools

| Tool Name               | Description                    |
| ----------------------- | ------------------------------ |
| `get_economic_calendar` | Upcoming CPI, FOMC, NFP, etc.  |
| `get_fed_rate`          | Current Fed funds rate         |
| `get_fed_watch`         | Rate change probabilities      |
| `get_cpi`               | CPI data (actual vs forecast)  |
| `get_pmi`               | PMI data                       |
| `get_macro_impact`      | Historical event impact on BTC |

#### Derivatives Tools

| Tool Name            | Description                |
| -------------------- | -------------------------- |
| `get_options_oi`     | Options open interest      |
| `get_max_pain`       | Max pain strike price      |
| `get_put_call_ratio` | Put/call ratio             |
| `get_gamma_exposure` | Dealer gamma exposure      |
| `get_options_flow`   | Large options trades       |
| `get_iv_surface`     | Implied volatility surface |

#### Correlation Tools

| Tool Name             | Description                   |
| --------------------- | ----------------------------- |
| `btc_dominance`       | BTC market dominance %        |
| `usdt_dominance`      | Stablecoin dominance %        |
| `altcoin_correlation` | Alt correlation to BTC        |
| `stock_correlation`   | BTC vs SPX/Nasdaq correlation |
| `dxy_correlation`     | BTC vs Dollar Index           |
| `gold_correlation`    | BTC vs Gold                   |
| `get_session_volume`  | Asia/EU/US session volumes    |

#### Trading Execution Tools

| Tool Name              | Description                 |
| ---------------------- | --------------------------- |
| `get_balance`          | Account balance             |
| `get_positions`        | Open positions              |
| `place_order`          | Market/limit/stop order     |
| `place_bracket_order`  | Entry + SL + TP in one      |
| `place_ladder_order`   | Multiple TP levels          |
| `place_iceberg_order`  | Hidden size order           |
| `cancel_order`         | Cancel specific order       |
| `cancel_all_orders`    | Cancel all open orders      |
| `close_position`       | Close specific position     |
| `close_all_positions`  | Close all positions         |
| `set_leverage`         | Set leverage (futures)      |
| `set_margin_mode`      | Set cross/isolated margin   |
| `move_sl_to_breakeven` | Move SL to entry price      |
| `set_trailing_stop`    | Activate trailing stop      |
| `add_to_position`      | DCA / average into position |

#### Risk Management Tools

| Tool Name                 | Description                 |
| ------------------------- | --------------------------- |
| `check_circuit_breaker`   | Check if trading allowed    |
| `get_daily_pnl`           | Today's PnL                 |
| `get_max_drawdown`        | Current drawdown            |
| `get_exposure`            | Current total exposure      |
| `calculate_position_size` | Kelly/fixed fraction sizing |
| `validate_trade`          | Pre-trade risk checks       |
| `emergency_close_all`     | KILL SWITCH                 |

#### Memory Tools

| Tool Name             | Description                   |
| --------------------- | ----------------------------- |
| `store_memory`        | Save observation/decision     |
| `search_memory`       | Semantic search past memories |
| `get_trade_history`   | Past trades with outcomes     |
| `get_market_regime`   | Current detected regime       |
| `store_market_regime` | Save regime detection         |

#### Evaluation Tools

| Tool Name              | Description                |
| ---------------------- | -------------------------- |
| `get_strategy_stats`   | Win rate per strategy      |
| `log_trade_decision`   | Create journal entry       |
| `get_trade_journal`    | Retrieve past decisions    |
| `evaluate_last_trades` | Performance review         |
| `get_best_strategies`  | Top performing strategies  |
| `get_worst_strategies` | Underperforming strategies |

### 8.3 Tool Implementation Example

```go
// internal/tools/indicators/rsi.go
package indicators

import (
    "context"
    "fmt"

    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/functiontool"

    "prometheus/internal/domain/market_data"
)

type RSIArgs struct {
    Symbol    string `json:"symbol" jsonschema:"description=Trading pair symbol (e.g. BTC/USDT),required"`
    Timeframe string `json:"timeframe" jsonschema:"description=Timeframe (1m/5m/15m/1h/4h/1d),required"`
    Period    int    `json:"period" jsonschema:"description=RSI period (default 14)"`
}

type RSIResult struct {
    Status    string  `json:"status"`
    RSI       float64 `json:"rsi"`
    Signal    string  `json:"signal"`     // overbought, oversold, neutral
    Trend     string  `json:"trend"`      // bullish_divergence, bearish_divergence, none
    PrevRSI   float64 `json:"prev_rsi"`
    Error     string  `json:"error,omitempty"`
}

type RSITool struct {
    marketDataRepo market_data.Repository
}

func NewRSITool(repo market_data.Repository) tool.Tool {
    t := &RSITool{marketDataRepo: repo}
    return functiontool.New(
        "rsi",
        "Calculate Relative Strength Index (RSI) for a trading pair. RSI > 70 indicates overbought, RSI < 30 indicates oversold.",
        t.Calculate,
    )
}

func (t *RSITool) Calculate(ctx tool.Context, args RSIArgs) (RSIResult, error) {
    period := args.Period
    if period == 0 {
        period = 14
    }

    // Fetch OHLCV data
    candles, err := t.marketDataRepo.GetOHLCV(context.Background(), market_data.OHLCVQuery{
        Symbol:    args.Symbol,
        Timeframe: args.Timeframe,
        Limit:     period + 50, // Extra for calculation
    })
    if err != nil {
        return RSIResult{Status: "error", Error: err.Error()}, nil
    }

    if len(candles) < period+1 {
        return RSIResult{Status: "error", Error: "insufficient data"}, nil
    }

    // Calculate RSI
    closes := make([]float64, len(candles))
    for i, c := range candles {
        closes[i] = c.Close
    }

    rsi := calculateRSI(closes, period)
    prevRSI := calculateRSI(closes[:len(closes)-1], period)

    // Determine signal
    signal := "neutral"
    if rsi > 70 {
        signal = "overbought"
    } else if rsi < 30 {
        signal = "oversold"
    }

    // Check for divergences (simplified)
    trend := "none"
    // ... divergence detection logic

    return RSIResult{
        Status:  "success",
        RSI:     rsi,
        Signal:  signal,
        Trend:   trend,
        PrevRSI: prevRSI,
    }, nil
}

func calculateRSI(closes []float64, period int) float64 {
    if len(closes) < period+1 {
        return 50 // Default neutral
    }

    gains := 0.0
    losses := 0.0

    // Initial average
    for i := 1; i <= period; i++ {
        change := closes[len(closes)-period-1+i] - closes[len(closes)-period-2+i]
        if change > 0 {
            gains += change
        } else {
            losses -= change
        }
    }

    avgGain := gains / float64(period)
    avgLoss := losses / float64(period)

    if avgLoss == 0 {
        return 100
    }

    rs := avgGain / avgLoss
    return 100 - (100 / (1 + rs))
}
```

---

## 9. Workers & Schedulers

### 9.1 Worker Interface

```go
// internal/workers/worker.go
package workers

import (
    "context"
    "time"
)

type Worker interface {
    Name() string
    Run(ctx context.Context) error
    Interval() time.Duration
    Enabled() bool
}

type BaseWorker struct {
    name     string
    interval time.Duration
    enabled  bool
}

func (w *BaseWorker) Name() string           { return w.name }
func (w *BaseWorker) Interval() time.Duration { return w.interval }
func (w *BaseWorker) Enabled() bool          { return w.enabled }
```

### 9.2 Scheduler

```go
// internal/workers/scheduler.go
package workers

import (
    "context"
    "sync"
    "time"

    "prometheus/pkg/logger"
)

type Scheduler struct {
    workers []Worker
    wg      sync.WaitGroup
    log     *logger.Logger
}

func NewScheduler(workers []Worker) *Scheduler {
    return &Scheduler{
        workers: workers,
        log:     logger.Get().With("component", "scheduler"),
    }
}

func (s *Scheduler) Start(ctx context.Context) {
    for _, w := range s.workers {
        if !w.Enabled() {
            s.log.Infof("Worker %s is disabled, skipping", w.Name())
            continue
        }

        s.wg.Add(1)
        go s.runWorker(ctx, w)
    }
}

func (s *Scheduler) runWorker(ctx context.Context, w Worker) {
    defer s.wg.Done()

    log := s.log.With("worker", w.Name())
    log.Infof("Starting worker with interval %s", w.Interval())

    ticker := time.NewTicker(w.Interval())
    defer ticker.Stop()

    // Run immediately on start
    s.executeWorker(ctx, w, log)

    for {
        select {
        case <-ctx.Done():
            log.Info("Worker stopped")
            return
        case <-ticker.C:
            s.executeWorker(ctx, w, log)
        }
    }
}

func (s *Scheduler) executeWorker(ctx context.Context, w Worker, log *logger.Logger) {
    start := time.Now()

    if err := w.Run(ctx); err != nil {
        log.Errorf("Worker execution failed: %v", err)
        return
    }

    log.Debugf("Worker completed in %s", time.Since(start))
}

func (s *Scheduler) Wait() {
    s.wg.Wait()
}
```

### 9.3 Worker Implementations

```go
// internal/workers/market_data/ohlcv_collector.go
package market_data

import (
    "context"
    "time"

    "prometheus/internal/adapters/exchanges"
    "prometheus/internal/domain/market_data"
    "prometheus/internal/workers"
    "prometheus/pkg/logger"
)

type OHLCVCollector struct {
    workers.BaseWorker
    exchangeFactory exchanges.Factory
    repo            market_data.Repository
    symbols         []string
    timeframes      []string
    log             *logger.Logger
}

type OHLCVCollectorConfig struct {
    Symbols    []string
    Timeframes []string
    Interval   time.Duration
    Enabled    bool
}

func NewOHLCVCollector(
    factory exchanges.Factory,
    repo market_data.Repository,
    cfg OHLCVCollectorConfig,
) *OHLCVCollector {
    return &OHLCVCollector{
        BaseWorker: workers.BaseWorker{
            name:     "ohlcv_collector",
            interval: cfg.Interval,
            enabled:  cfg.Enabled,
        },
        exchangeFactory: factory,
        repo:            repo,
        symbols:         cfg.Symbols,
        timeframes:      cfg.Timeframes,
        log:             logger.Get().With("worker", "ohlcv_collector"),
    }
}

func (w *OHLCVCollector) Run(ctx context.Context) error {
    for _, exchange := range w.exchangeFactory.ListExchanges() {
        client, err := w.exchangeFactory.GetClient(exchange)
        if err != nil {
            w.log.Warnf("Failed to get %s client: %v", exchange, err)
            continue
        }

        for _, symbol := range w.symbols {
            for _, tf := range w.timeframes {
                candles, err := client.GetOHLCV(ctx, symbol, tf, 100)
                if err != nil {
                    w.log.Warnf("Failed to fetch %s %s %s: %v", exchange, symbol, tf, err)
                    continue
                }

                if err := w.repo.BatchInsert(ctx, candles); err != nil {
                    w.log.Errorf("Failed to insert candles: %v", err)
                }
            }
        }
    }

    return nil
}
```

```go
// internal/workers/sentiment/news_collector.go
package sentiment

import (
    "context"
    "time"

    "prometheus/internal/adapters/datasources"
    "prometheus/internal/domain/sentiment"
    "prometheus/internal/workers"
    "prometheus/pkg/logger"
)

type NewsCollector struct {
    workers.BaseWorker
    sources []datasources.NewsSource
    repo    sentiment.Repository
    log     *logger.Logger
}

func NewNewsCollector(
    sources []datasources.NewsSource,
    repo sentiment.Repository,
    interval time.Duration,
) *NewsCollector {
    return &NewsCollector{
        BaseWorker: workers.BaseWorker{
            name:     "news_collector",
            interval: interval,
            enabled:  true,
        },
        sources: sources,
        repo:    repo,
        log:     logger.Get().With("worker", "news_collector"),
    }
}

func (w *NewsCollector) Run(ctx context.Context) error {
    for _, source := range w.sources {
        articles, err := source.FetchLatest(ctx, 50)
        if err != nil {
            w.log.Warnf("Failed to fetch from %s: %v", source.Name(), err)
            continue
        }

        for _, article := range articles {
            // Analyze sentiment
            sentimentScore := analyzeSentiment(article.Content)

            news := &sentiment.News{
                Source:    source.Name(),
                Title:     article.Title,
                Content:   article.Content,
                URL:       article.URL,
                Sentiment: sentimentScore,
                Symbols:   extractSymbols(article.Content),
                PublishedAt: article.PublishedAt,
            }

            if err := w.repo.InsertNews(ctx, news); err != nil {
                w.log.Errorf("Failed to insert news: %v", err)
            }
        }
    }

    return nil
}
```

```go
// internal/workers/analysis/market_scanner.go
package analysis

import (
    "context"
    "time"

    "prometheus/internal/agents"
    "prometheus/internal/domain/trading_pair"
    "prometheus/internal/workers"
    "prometheus/pkg/logger"
)

type MarketScanner struct {
    workers.BaseWorker
    tradingPairRepo trading_pair.Repository
    agentFactory    *agents.Factory
    log             *logger.Logger
}

func NewMarketScanner(
    repo trading_pair.Repository,
    factory *agents.Factory,
    interval time.Duration,
) *MarketScanner {
    return &MarketScanner{
        BaseWorker: workers.BaseWorker{
            name:     "market_scanner",
            interval: interval,
            enabled:  true,
        },
        tradingPairRepo: repo,
        agentFactory:    factory,
        log:             logger.Get().With("worker", "market_scanner"),
    }
}

func (w *MarketScanner) Run(ctx context.Context) error {
    // Get all active trading pairs
    pairs, err := w.tradingPairRepo.FindActive(ctx)
    if err != nil {
        return err
    }

    for _, pair := range pairs {
        if pair.IsPaused {
            continue
        }

        // Create pipeline for this pair
        pipeline, err := w.agentFactory.CreateTradingPipeline(agents.UserAgentConfig{
            UserID:     pair.UserID.String(),
            AIProvider: pair.AIProvider,
            Symbol:     pair.Symbol,
            MarketType: string(pair.MarketType),
        })
        if err != nil {
            w.log.Errorf("Failed to create pipeline for %s: %v", pair.Symbol, err)
            continue
        }

        // Run analysis
        w.log.Infof("Scanning %s for user %s", pair.Symbol, pair.UserID)

        // Execute pipeline...
        // This would trigger the full agent workflow
    }

    return nil
}
```

### 9.4 Worker Schedule Summary

| Worker                    | Interval | Description               |
| ------------------------- | -------- | ------------------------- |
| **Market Data**           |          |                           |
| `ohlcv_collector`         | 1m       | Collect OHLCV candles     |
| `ticker_collector`        | 5s       | Real-time price tickers   |
| `orderbook_collector`     | 10s      | Order book snapshots      |
| `trades_collector`        | 1s       | Tape / trades feed        |
| `funding_collector`       | 1m       | Funding rates             |
| **Order Flow**            |          |                           |
| `liquidation_collector`   | 5s       | Real-time liquidations    |
| `oi_collector`            | 1m       | Open interest             |
| `whale_alert_collector`   | 10s      | Large trades alerts       |
| **Sentiment**             |          |                           |
| `news_collector`          | 5m       | News aggregation          |
| `twitter_collector`       | 2m       | X.com sentiment           |
| `reddit_collector`        | 5m       | Reddit sentiment          |
| `feargreed_collector`     | 15m      | Fear & Greed index        |
| **On-Chain**              |          |                           |
| `exchange_flow_collector` | 5m       | Exchange inflows/outflows |
| `whale_collector`         | 5m       | Whale movements           |
| `metrics_collector`       | 15m      | MVRV, SOPR, NVT           |
| **Macro**                 |          |                           |
| `calendar_collector`      | 1h       | Economic calendar         |
| `fed_collector`           | 1h       | Fed data                  |
| `correlation_collector`   | 5m       | SPX, DXY, Gold            |
| **Derivatives**           |          |                           |
| `options_collector`       | 5m       | Options OI, max pain      |
| `gamma_collector`         | 15m      | Gamma exposure            |
| **Trading**               |          |                           |
| `position_monitor`        | 30s      | Position health check     |
| `order_sync`              | 10s      | Sync order statuses       |
| `pnl_calculator`          | 1m       | Calculate PnL             |
| `risk_monitor`            | 10s      | Circuit breaker checks    |
| **Analysis**              |          |                           |
| `market_scanner`          | 5m       | Run agent analysis        |
| `opportunity_finder`      | 1m       | Find trading setups       |
| `regime_detector`         | 5m       | Update market regime      |
| `smc_scanner`             | 1m       | FVG, OB detection         |
| **Evaluation**            |          |                           |
| `daily_report`            | 24h      | Generate daily report     |
| `strategy_evaluator`      | 6h       | Evaluate strategies       |
| `journal_compiler`        | 1h       | Compile trade journal     |

---

## 9.5 Risk Engine

Risk Engine — **независимый сервис**, который может остановить любую торговлю.

```go
// internal/risk/engine.go
package risk

import (
    "context"
    "sync"
    "time"

    "github.com/shopspring/decimal"

    "prometheus/internal/domain/risk"
    "prometheus/internal/domain/user"
    "prometheus/pkg/logger"
)

type Engine struct {
    stateRepo    risk.StateRepository
    userRepo     user.Repository
    eventPublisher EventPublisher

    // In-memory state for fast checks
    states       map[string]*CircuitBreakerState
    mu           sync.RWMutex

    log          *logger.Logger
}

type EngineConfig struct {
    CheckInterval time.Duration  // How often to check (e.g., 1s)
}

func NewEngine(
    stateRepo risk.StateRepository,
    userRepo user.Repository,
    publisher EventPublisher,
) *Engine {
    return &Engine{
        stateRepo:      stateRepo,
        userRepo:       userRepo,
        eventPublisher: publisher,
        states:         make(map[string]*CircuitBreakerState),
        log:            logger.Get().With("component", "risk_engine"),
    }
}

// CanTrade checks if trading is allowed for a user
// This is called BEFORE any trade execution
func (e *Engine) CanTrade(ctx context.Context, userID string) (bool, string) {
    e.mu.RLock()
    state, ok := e.states[userID]
    e.mu.RUnlock()

    if !ok {
        // Load from DB
        state, _ = e.loadState(ctx, userID)
    }

    if state == nil {
        return true, ""  // No state = no restrictions
    }

    if state.IsTriggered {
        return false, state.TriggerReason
    }

    return true, ""
}

// RecordTrade updates risk state after a trade
func (e *Engine) RecordTrade(ctx context.Context, userID string, pnl decimal.Decimal, isWin bool) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    state, ok := e.states[userID]
    if !ok {
        state = e.newState(userID)
    }

    // Update stats
    state.DailyPnL = state.DailyPnL.Add(pnl)
    state.DailyTradeCount++

    if isWin {
        state.DailyWins++
        state.ConsecutiveLosses = 0
    } else {
        state.DailyLosses++
        state.ConsecutiveLosses++
    }

    // Check triggers
    e.checkTriggers(ctx, state)

    // Persist
    return e.stateRepo.Save(ctx, state)
}

func (e *Engine) checkTriggers(ctx context.Context, state *CircuitBreakerState) {
    // Check max daily drawdown
    if state.DailyPnLPercent.LessThan(state.MaxDailyDrawdown.Neg()) {
        e.triggerCircuitBreaker(ctx, state, "max_daily_drawdown",
            "Daily drawdown exceeded limit")
        return
    }

    // Check consecutive losses
    if state.ConsecutiveLosses >= state.MaxConsecutiveLoss {
        e.triggerCircuitBreaker(ctx, state, "consecutive_losses",
            "Too many consecutive losses")
        return
    }
}

func (e *Engine) triggerCircuitBreaker(ctx context.Context, state *CircuitBreakerState, reason, message string) {
    now := time.Now()
    state.IsTriggered = true
    state.TriggeredAt = &now
    state.TriggerReason = reason

    e.log.Warnf("Circuit breaker triggered for user %s: %s", state.UserID, message)

    // Publish event for notifications
    e.eventPublisher.Publish(ctx, risk.RiskEvent{
        UserID:    state.UserID,
        EventType: risk.RiskEventCircuitBreaker,
        Severity:  "critical",
        Message:   message,
    })
}

// KillSwitch - emergency shutdown, closes ALL positions
func (e *Engine) KillSwitch(ctx context.Context, userID string) error {
    e.log.Errorf("KILL SWITCH ACTIVATED for user %s", userID)

    // 1. Trigger circuit breaker
    e.mu.Lock()
    state := e.states[userID]
    if state != nil {
        now := time.Now()
        state.IsTriggered = true
        state.TriggeredAt = &now
        state.TriggerReason = "kill_switch"
    }
    e.mu.Unlock()

    // 2. Publish kill switch event (will trigger position closing)
    return e.eventPublisher.Publish(ctx, risk.RiskEvent{
        UserID:    userID,
        EventType: risk.RiskEventKillSwitch,
        Severity:  "critical",
        Message:   "Emergency kill switch activated",
    })
}

// ResetDaily resets daily stats (called at 00:00 UTC)
func (e *Engine) ResetDaily(ctx context.Context) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    for _, state := range e.states {
        state.DailyPnL = decimal.Zero
        state.DailyPnLPercent = decimal.Zero
        state.DailyTradeCount = 0
        state.DailyWins = 0
        state.DailyLosses = 0
        state.ConsecutiveLosses = 0
        state.IsTriggered = false
        state.TriggeredAt = nil
        state.TriggerReason = ""
    }

    e.log.Info("Daily risk stats reset")
    return nil
}
```

---

## 9.6 Self-Evaluation System

````go
// internal/evaluation/evaluator.go
package evaluation

import (
    "context"
    "time"

    "github.com/shopspring/decimal"

    "prometheus/internal/domain/journal"
    "prometheus/internal/domain/trading_pair"
    "prometheus/pkg/logger"
)

type Evaluator struct {
    journalRepo      journal.Repository
    tradingPairRepo  trading_pair.Repository
    log              *logger.Logger
}

func NewEvaluator(
    journalRepo journal.Repository,
    tradingPairRepo trading_pair.Repository,
) *Evaluator {
    return &Evaluator{
        journalRepo:     journalRepo,
        tradingPairRepo: tradingPairRepo,
        log:             logger.Get().With("component", "evaluator"),
    }
}

// EvaluateStrategies analyzes all strategies and returns stats
func (e *Evaluator) EvaluateStrategies(ctx context.Context, userID string, period time.Duration) ([]journal.StrategyStats, error) {
    since := time.Now().Add(-period)

    entries, err := e.journalRepo.GetEntriesSince(ctx, userID, since)
    if err != nil {
        return nil, err
    }

    // Group by strategy
    strategyTrades := make(map[string][]journal.JournalEntry)
    for _, entry := range entries {
        strategyTrades[entry.StrategyUsed] = append(strategyTrades[entry.StrategyUsed], entry)
    }

    // Calculate stats for each strategy
    var stats []journal.StrategyStats
    for strategy, trades := range strategyTrades {
        stat := e.calculateStats(strategy, trades)
        stats = append(stats, stat)
    }

    return stats, nil
}

func (e *Evaluator) calculateStats(strategy string, trades []journal.JournalEntry) journal.StrategyStats {
    stat := journal.StrategyStats{
        StrategyName: strategy,
        TotalTrades:  len(trades),
        IsActive:     true,
        LastUpdated:  time.Now(),
    }

    var totalWin, totalLoss decimal.Decimal
    var winCount, lossCount int
    var maxDD decimal.Decimal
    var runningPnL decimal.Decimal
    var peak decimal.Decimal

    for _, trade := range trades {
        runningPnL = runningPnL.Add(trade.PnL)

        if runningPnL.GreaterThan(peak) {
            peak = runningPnL
        }

        dd := peak.Sub(runningPnL)
        if dd.GreaterThan(maxDD) {
            maxDD = dd
        }

        if trade.PnL.GreaterThan(decimal.Zero) {
            winCount++
            totalWin = totalWin.Add(trade.PnL)
        } else {
            lossCount++
            totalLoss = totalLoss.Add(trade.PnL.Abs())
        }
    }

    stat.WinningTrades = winCount
    stat.LosingTrades = lossCount
    stat.MaxDrawdown = maxDD

    if stat.TotalTrades > 0 {
        stat.WinRate = float64(winCount) / float64(stat.TotalTrades) * 100
    }

    if winCount > 0 {
        stat.AvgWin = totalWin.Div(decimal.NewFromInt(int64(winCount)))
    }

    if lossCount > 0 {
        stat.AvgLoss = totalLoss.Div(decimal.NewFromInt(int64(lossCount)))
    }

    if !totalLoss.IsZero() {
        stat.ProfitFactor = totalWin.Div(totalLoss)
    }

    // Expected Value = (WinRate * AvgWin) - (LossRate * AvgLoss)
    winRateDec := decimal.NewFromFloat(stat.WinRate / 100)
    lossRateDec := decimal.NewFromFloat(1 - stat.WinRate/100)
    stat.ExpectedValue = winRateDec.Mul(stat.AvgWin).Sub(lossRateDec.Mul(stat.AvgLoss))

    return stat
}

// GetUnderperformingStrategies returns strategies that should be disabled
func (e *Evaluator) GetUnderperformingStrategies(ctx context.Context, userID string) ([]string, error) {
    stats, err := e.EvaluateStrategies(ctx, userID, 30*24*time.Hour) // Last 30 days
    if err != nil {
        return nil, err
    }

    var underperforming []string
    for _, stat := range stats {
        // Criteria for underperforming:
        // 1. Win rate < 35%
        // 2. Profit factor < 0.8
        // 3. Expected value negative
        // 4. At least 10 trades (statistical significance)

        if stat.TotalTrades < 10 {
            continue
        }

        isUnderperforming := stat.WinRate < 35 ||
            stat.ProfitFactor.LessThan(decimal.NewFromFloat(0.8)) ||
            stat.ExpectedValue.LessThan(decimal.Zero)

        if isUnderperforming {
            underperforming = append(underperforming, stat.StrategyName)
            e.log.Warnf("Strategy %s is underperforming: WR=%.1f%%, PF=%s, EV=%s",
                stat.StrategyName, stat.WinRate, stat.ProfitFactor, stat.ExpectedValue)
        }
    }

    return underperforming, nil
}

// DisableStrategy marks a strategy as inactive
func (e *Evaluator) DisableStrategy(ctx context.Context, userID, strategy string) error {
    e.log.Warnf("Disabling strategy %s for user %s due to poor performance", strategy, userID)
    // Store in user preferences or strategy settings
    return nil
}

// GenerateDailyReport creates a performance summary
func (e *Evaluator) GenerateDailyReport(ctx context.Context, userID string) (*DailyReport, error) {
    today := time.Now().Truncate(24 * time.Hour)

    entries, err := e.journalRepo.GetEntriesSince(ctx, userID, today)
    if err != nil {
        return nil, err
    }

    report := &DailyReport{
        Date:        today,
        TotalTrades: len(entries),
    }

    for _, entry := range entries {
        report.TotalPnL = report.TotalPnL.Add(entry.PnL)
        if entry.PnL.GreaterThan(decimal.Zero) {
            report.Wins++
        } else {
            report.Losses++
        }
    }

    if report.TotalTrades > 0 {
        report.WinRate = float64(report.Wins) / float64(report.TotalTrades) * 100
    }

    // Get strategy breakdown
    stats, _ := e.EvaluateStrategies(ctx, userID, 24*time.Hour)
    report.StrategyBreakdown = stats

    return report, nil
}

type DailyReport struct {
    Date              time.Time
    TotalTrades       int
    Wins              int
    Losses            int
    WinRate           float64
    TotalPnL          decimal.Decimal
    BestTrade         *journal.JournalEntry
    WorstTrade        *journal.JournalEntry
    StrategyBreakdown []journal.StrategyStats
}

---

## 10. AI Provider Abstraction

### 10.1 Provider Interface

```go
// internal/adapters/ai/interface.go
package ai

import (
    "context"

    "google.golang.org/adk/model"
)

type Provider interface {
    Name() string
    GetModel(modelID string) (model.Model, error)
    ListModels() []ModelInfo
    SupportsStreaming() bool
    SupportsTools() bool
}

type ModelInfo struct {
    ID          string
    Name        string
    MaxTokens   int
    InputCost   float64  // Per 1M tokens
    OutputCost  float64  // Per 1M tokens
    SupportsVision bool
}

type ProviderRegistry interface {
    Register(provider Provider)
    Get(name string) (Provider, bool)
    GetModel(providerName, modelID string) (model.Model, error)
    ListProviders() []string
}
````

### 10.2 Provider Registry Implementation

```go
// internal/adapters/ai/registry.go
package ai

import (
    "fmt"
    "sync"

    "google.golang.org/adk/model"
)

type registry struct {
    providers map[string]Provider
    mu        sync.RWMutex
}

func NewRegistry() ProviderRegistry {
    return &registry{
        providers: make(map[string]Provider),
    }
}

func (r *registry) Register(provider Provider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[provider.Name()] = provider
}

func (r *registry) Get(name string) (Provider, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    p, ok := r.providers[name]
    return p, ok
}

func (r *registry) GetModel(providerName, modelID string) (model.Model, error) {
    provider, ok := r.Get(providerName)
    if !ok {
        return nil, fmt.Errorf("provider not found: %s", providerName)
    }
    return provider.GetModel(modelID)
}

func (r *registry) ListProviders() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    names := make([]string, 0, len(r.providers))
    for name := range r.providers {
        names = append(names, name)
    }
    return names
}
```

### 10.3 Claude Provider

```go
// internal/adapters/ai/claude/provider.go
package claude

import (
    "context"
    "fmt"

    "google.golang.org/adk/model"
    "google.golang.org/genai"

    "prometheus/internal/adapters/ai"
)

type Provider struct {
    apiKey string
    models map[string]*modelWrapper
}

func New(apiKey string) *Provider {
    return &Provider{
        apiKey: apiKey,
        models: make(map[string]*modelWrapper),
    }
}

func (p *Provider) Name() string {
    return "claude"
}

func (p *Provider) GetModel(modelID string) (model.Model, error) {
    if m, ok := p.models[modelID]; ok {
        return m, nil
    }

    // Create new model wrapper
    m, err := newModelWrapper(p.apiKey, modelID)
    if err != nil {
        return nil, err
    }

    p.models[modelID] = m
    return m, nil
}

func (p *Provider) ListModels() []ai.ModelInfo {
    return []ai.ModelInfo{
        {ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", MaxTokens: 64000},
        {ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", MaxTokens: 200000},
        {ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", MaxTokens: 200000},
    }
}

func (p *Provider) SupportsStreaming() bool { return true }
func (p *Provider) SupportsTools() bool     { return true }
```

### 10.4 Provider Factory

```go
// internal/adapters/ai/factory.go
package ai

import (
    "prometheus/internal/adapters/ai/claude"
    "prometheus/internal/adapters/ai/openai"
    "prometheus/internal/adapters/ai/deepseek"
    "prometheus/internal/adapters/ai/gemini"
    "prometheus/internal/adapters/config"
)

func SetupProviders(cfg config.AIConfig) ProviderRegistry {
    registry := NewRegistry()

    if cfg.ClaudeKey != "" {
        registry.Register(claude.New(cfg.ClaudeKey))
    }

    if cfg.OpenAIKey != "" {
        registry.Register(openai.New(cfg.OpenAIKey))
    }

    if cfg.DeepSeekKey != "" {
        registry.Register(deepseek.New(cfg.DeepSeekKey))
    }

    if cfg.GeminiKey != "" {
        registry.Register(gemini.New(cfg.GeminiKey))
    }

    return registry
}
```

---

## 11. Exchange Abstraction

### 11.1 Exchange Interface

```go
// internal/adapters/exchanges/interface.go
package exchanges

import (
    "context"

    "github.com/shopspring/decimal"
)

type Exchange interface {
    Name() string

    // Market Data
    GetTicker(ctx context.Context, symbol string) (*Ticker, error)
    GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error)
    GetOHLCV(ctx context.Context, symbol string, timeframe string, limit int) ([]OHLCV, error)
    GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error)

    // Futures specific
    GetFundingRate(ctx context.Context, symbol string) (*FundingRate, error)
    GetOpenInterest(ctx context.Context, symbol string) (*OpenInterest, error)

    // Account
    GetBalance(ctx context.Context) (*Balance, error)
    GetPositions(ctx context.Context) ([]Position, error)
    GetOpenOrders(ctx context.Context, symbol string) ([]Order, error)

    // Trading
    PlaceOrder(ctx context.Context, req *OrderRequest) (*Order, error)
    CancelOrder(ctx context.Context, symbol, orderID string) error

    // Futures
    SetLeverage(ctx context.Context, symbol string, leverage int) error
    SetMarginMode(ctx context.Context, symbol string, mode MarginMode) error
}

type SpotExchange interface {
    Exchange
}

type FuturesExchange interface {
    Exchange
    GetFundingRate(ctx context.Context, symbol string) (*FundingRate, error)
    SetLeverage(ctx context.Context, symbol string, leverage int) error
    SetMarginMode(ctx context.Context, symbol string, mode MarginMode) error
}

type MarginMode string
const (
    MarginCross    MarginMode = "cross"
    MarginIsolated MarginMode = "isolated"
)
```

### 11.2 Exchange Factory

```go
// internal/adapters/exchanges/factory.go
package exchanges

import (
    "fmt"
    "sync"

    "prometheus/internal/adapters/exchanges/binance"
    "prometheus/internal/adapters/exchanges/bybit"
    "prometheus/internal/domain/exchange_account"
)

type Factory interface {
    CreateClient(account *exchange_account.ExchangeAccount) (Exchange, error)
    GetClient(exchangeType exchange_account.ExchangeType) (Exchange, error)
    ListExchanges() []exchange_account.ExchangeType
}

// CentralFactory uses central API keys from ENV for market data collection
type CentralFactory interface {
    GetClient(exchange string) (Exchange, error) // Uses central API keys, not user-specific
    ListExchanges() []string
}

type factory struct {
    clients map[string]Exchange
    mu      sync.RWMutex
}

func NewFactory() Factory {
    return &factory{
        clients: make(map[string]Exchange),
    }
}

func (f *factory) CreateClient(account *exchange_account.ExchangeAccount, encryptor *crypto.Encryptor) (Exchange, error) {
    key := fmt.Sprintf("%s:%s", account.Exchange, account.ID)

    f.mu.Lock()
    defer f.mu.Unlock()

    if client, ok := f.clients[key]; ok {
        return client, nil
    }

    // Decrypt API keys
    apiKey, err := encryptor.Decrypt(account.APIKeyEncrypted)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt API key: %w", err)
    }

    secret, err := encryptor.Decrypt(account.SecretEncrypted)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt secret: %w", err)
    }

    var client Exchange

    switch account.Exchange {
    case exchange_account.ExchangeBinance:
        client, err = binance.NewClient(binance.Config{
            APIKey:    apiKey,
            SecretKey: secret,
            Testnet:   account.IsTestnet,
        })
    case exchange_account.ExchangeBybit:
        client, err = bybit.NewClient(bybit.Config{
            APIKey:    apiKey,
            SecretKey: secret,
            Testnet:   account.IsTestnet,
        })
    default:
        return nil, fmt.Errorf("unsupported exchange: %s", account.Exchange)
    }

    if err != nil {
        return nil, err
    }

    f.clients[key] = client
    return client, nil
}

// CentralFactory for market data collection (uses central API keys from ENV)
type centralFactory struct {
    cfg     config.MarketDataConfig
    clients map[string]Exchange
    mu      sync.RWMutex
}

func NewCentralFactory(cfg config.MarketDataConfig) CentralFactory {
    return &centralFactory{
        cfg:     cfg,
        clients: make(map[string]Exchange),
    }
}

func (f *centralFactory) GetClient(exchange string) (Exchange, error) {
    f.mu.RLock()
    if client, ok := f.clients[exchange]; ok {
        f.mu.RUnlock()
        return client, nil
    }
    f.mu.RUnlock()

    f.mu.Lock()
    defer f.mu.Unlock()

    // Double check
    if client, ok := f.clients[exchange]; ok {
        return client, nil
    }

    var client Exchange
    var err error

    switch exchange {
    case "binance":
        client, err = binance.NewClient(binance.Config{
            APIKey:    f.cfg.BinanceAPIKey,
            SecretKey: f.cfg.BinanceSecret,
            Testnet:   false,
        })
    case "bybit":
        client, err = bybit.NewClient(bybit.Config{
            APIKey:    f.cfg.BybitAPIKey,
            SecretKey: f.cfg.BybitSecret,
            Testnet:   false,
        })
    case "okx":
        client, err = okx.NewClient(okx.Config{
            APIKey:    f.cfg.OKXAPIKey,
            SecretKey: f.cfg.OKXSecret,
            Passphrase: f.cfg.OKXPassphrase,
            Testnet:   false,
        })
    default:
        return nil, fmt.Errorf("unsupported exchange: %s", exchange)
    }

    if err != nil {
        return nil, err
    }

    f.clients[exchange] = client
    return client, nil
}

func (f *centralFactory) ListExchanges() []string {
    return []string{"binance", "bybit", "okx"}
}
```

---

## 12. Telegram Bot

### 12.1 Bot Handler

```go
// internal/adapters/telegram/bot.go
package telegram

import (
    "context"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

    "prometheus/internal/adapters/telegram/handlers"
    "prometheus/internal/domain/user"
    "prometheus/pkg/logger"
)

type Bot struct {
    api         *tgbotapi.BotAPI
    userService user.Service
    handlers    *handlers.Registry
    log         *logger.Logger
}

type BotDeps struct {
    Token       string
    UserService user.Service
    Handlers    *handlers.Registry
}

func NewBot(deps BotDeps) (*Bot, error) {
    api, err := tgbotapi.NewBotAPI(deps.Token)
    if err != nil {
        return nil, err
    }

    return &Bot{
        api:         api,
        userService: deps.UserService,
        handlers:    deps.Handlers,
        log:         logger.Get().With("component", "telegram"),
    }, nil
}

func (b *Bot) Start(ctx context.Context) error {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := b.api.GetUpdatesChan(u)

    for {
        select {
        case <-ctx.Done():
            return nil
        case update := <-updates:
            go b.handleUpdate(ctx, update)
        }
    }
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
    // Ensure user exists
    var telegramUser *tgbotapi.User
    if update.Message != nil {
        telegramUser = update.Message.From
    } else if update.CallbackQuery != nil {
        telegramUser = update.CallbackQuery.From
    }

    if telegramUser == nil {
        return
    }

    // Auto-register user
    u, err := b.userService.GetOrCreate(ctx, user.CreateParams{
        TelegramID:       telegramUser.ID,
        TelegramUsername: telegramUser.UserName,
        FirstName:        telegramUser.FirstName,
        LastName:         telegramUser.LastName,
        LanguageCode:     telegramUser.LanguageCode,
    })
    if err != nil {
        b.log.Errorf("Failed to get/create user: %v", err)
        return
    }

    // Route to handler
    b.handlers.Handle(ctx, b.api, update, u)
}
```

### 12.2 Command Handlers

```go
// internal/adapters/telegram/handlers/registry.go
package handlers

import (
    "context"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

    "prometheus/internal/domain/user"
)

type Handler interface {
    Handle(ctx context.Context, api *tgbotapi.BotAPI, update tgbotapi.Update, u *user.User)
}

type CommandHandler interface {
    Command() string
    Handler
}

type CallbackHandler interface {
    Prefix() string
    Handler
}

type Registry struct {
    commands  map[string]CommandHandler
    callbacks map[string]CallbackHandler
}

func NewRegistry() *Registry {
    return &Registry{
        commands:  make(map[string]CommandHandler),
        callbacks: make(map[string]CallbackHandler),
    }
}

func (r *Registry) RegisterCommand(h CommandHandler) {
    r.commands[h.Command()] = h
}

func (r *Registry) RegisterCallback(h CallbackHandler) {
    r.callbacks[h.Prefix()] = h
}

func (r *Registry) Handle(ctx context.Context, api *tgbotapi.BotAPI, update tgbotapi.Update, u *user.User) {
    if update.Message != nil && update.Message.IsCommand() {
        cmd := update.Message.Command()
        if h, ok := r.commands[cmd]; ok {
            h.Handle(ctx, api, update, u)
        }
    }

    if update.CallbackQuery != nil {
        // Parse callback data prefix
        // Format: "prefix:data"
        // Route to appropriate handler
    }
}
```

### 12.3 Bot Commands

| Command       | Description              |
| ------------- | ------------------------ |
| `/start`      | Welcome, auto-register   |
| `/help`       | Show all commands        |
| `/connect`    | Connect exchange account |
| `/disconnect` | Remove exchange account  |
| `/exchanges`  | List connected exchanges |
| `/addpair`    | Add trading pair         |
| `/removepair` | Remove trading pair      |
| `/pairs`      | List trading pairs       |
| `/balance`    | Show balances            |
| `/positions`  | Show open positions      |
| `/orders`     | Show open orders         |
| `/stats`      | Trading statistics       |
| `/pnl`        | Today's PnL              |
| `/settings`   | User settings            |
| `/pause`      | Pause trading            |
| `/resume`     | Resume trading           |
| `/report`     | Generate report          |

### 12.4 Conversation Flows

```
/connect Flow:
┌────────────────────┐
│ Select Exchange    │
│ [Binance] [Bybit]  │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Enter API Key      │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Enter Secret       │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Test Connection    │
│ ✅ Connected!      │
└────────────────────┘

/addpair Flow:
┌────────────────────┐
│ Select Exchange    │
│ [Binance Account]  │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Enter Symbol       │
│ e.g., BTC/USDT     │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Select Market Type │
│ [Spot] [Futures]   │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Enter Budget (USDT)│
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Select AI Provider │
│ [Claude] [GPT-4]   │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ Select Risk Level  │
│ [Low][Med][High]   │
└────────────────────┘
         │
         ▼
┌────────────────────┐
│ ✅ Pair Added!     │
│ Trading will begin │
└────────────────────┘
```

---

## 13. Database Schemas

### 13.1 PostgreSQL Migrations

```sql
-- migrations/postgres/001_init.up.sql

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    telegram_id BIGINT UNIQUE NOT NULL,
    telegram_username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    language_code VARCHAR(10) DEFAULT 'en',
    is_active BOOLEAN DEFAULT true,
    is_premium BOOLEAN DEFAULT false,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);

-- Exchange Accounts
CREATE TABLE exchange_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange VARCHAR(50) NOT NULL,
    label VARCHAR(255),
    api_key_encrypted BYTEA NOT NULL,
    secret_encrypted BYTEA NOT NULL,
    passphrase BYTEA,
    is_testnet BOOLEAN DEFAULT false,
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_user_exchange_label UNIQUE(user_id, exchange, label)
);

CREATE INDEX idx_exchange_accounts_user ON exchange_accounts(user_id);

-- Trading Pairs
CREATE TABLE trading_pairs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id) ON DELETE CASCADE,
    symbol VARCHAR(50) NOT NULL,
    market_type VARCHAR(20) NOT NULL,

    -- Budget & Risk
    budget DECIMAL(20,8) NOT NULL,
    max_position_size DECIMAL(20,8),
    max_leverage INTEGER DEFAULT 1,
    stop_loss_percent DECIMAL(5,2) DEFAULT 2.0,
    take_profit_percent DECIMAL(5,2) DEFAULT 4.0,

    -- Strategy
    ai_provider VARCHAR(50) DEFAULT 'claude',
    strategy_mode VARCHAR(20) DEFAULT 'signals',
    timeframes TEXT[] DEFAULT '{"1h","4h","1d"}',

    -- State
    is_active BOOLEAN DEFAULT true,
    is_paused BOOLEAN DEFAULT false,
    paused_reason TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_pair_per_account UNIQUE(exchange_account_id, symbol, market_type)
);

CREATE INDEX idx_trading_pairs_user ON trading_pairs(user_id);
CREATE INDEX idx_trading_pairs_active ON trading_pairs(is_active, is_paused);

-- Orders
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    trading_pair_id UUID REFERENCES trading_pairs(id),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),

    exchange_order_id VARCHAR(255),
    symbol VARCHAR(50) NOT NULL,
    market_type VARCHAR(20) NOT NULL,

    side VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    price DECIMAL(20,8),
    amount DECIMAL(20,8) NOT NULL,
    filled_amount DECIMAL(20,8) DEFAULT 0,
    avg_fill_price DECIMAL(20,8),

    stop_price DECIMAL(20,8),
    reduce_only BOOLEAN DEFAULT false,

    agent_id VARCHAR(100),
    reasoning TEXT,
    parent_order_id UUID REFERENCES orders(id),

    fee DECIMAL(20,8) DEFAULT 0,
    fee_currency VARCHAR(20),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    filled_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_exchange ON orders(exchange_order_id);

-- Positions
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    trading_pair_id UUID REFERENCES trading_pairs(id),
    exchange_account_id UUID NOT NULL REFERENCES exchange_accounts(id),

    symbol VARCHAR(50) NOT NULL,
    market_type VARCHAR(20) NOT NULL,
    side VARCHAR(10) NOT NULL,

    size DECIMAL(20,8) NOT NULL,
    entry_price DECIMAL(20,8) NOT NULL,
    current_price DECIMAL(20,8),
    liquidation_price DECIMAL(20,8),

    leverage INTEGER DEFAULT 1,
    margin_mode VARCHAR(20) DEFAULT 'cross',

    unrealized_pnl DECIMAL(20,8) DEFAULT 0,
    unrealized_pnl_pct DECIMAL(10,4) DEFAULT 0,
    realized_pnl DECIMAL(20,8) DEFAULT 0,

    stop_loss_price DECIMAL(20,8),
    take_profit_price DECIMAL(20,8),
    trailing_stop_pct DECIMAL(5,2),

    stop_loss_order_id UUID REFERENCES orders(id),
    take_profit_order_id UUID REFERENCES orders(id),

    open_reasoning TEXT,

    status VARCHAR(20) NOT NULL DEFAULT 'open',
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_positions_user ON positions(user_id);
CREATE INDEX idx_positions_status ON positions(status);

-- Agent Memory (with vector embeddings)
CREATE TABLE memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    agent_id VARCHAR(100) NOT NULL,
    session_id VARCHAR(255),

    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),  -- OpenAI ada-002 dimension

    symbol VARCHAR(50),
    timeframe VARCHAR(10),
    importance DECIMAL(3,2) DEFAULT 0.5,

    related_ids UUID[] DEFAULT '{}',
    trade_id UUID REFERENCES orders(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_memories_user ON memories(user_id);
CREATE INDEX idx_memories_agent ON memories(agent_id);
CREATE INDEX idx_memories_embedding ON memories USING ivfflat (embedding vector_cosine_ops);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER exchange_accounts_updated_at BEFORE UPDATE ON exchange_accounts
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trading_pairs_updated_at BEFORE UPDATE ON trading_pairs
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders
FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER positions_updated_at BEFORE UPDATE ON positions
FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

### 13.2 ClickHouse Schemas

```sql
-- migrations/clickhouse/001_init.up.sql

-- OHLCV Data
CREATE TABLE IF NOT EXISTS ohlcv (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timeframe LowCardinality(String),
    open_time DateTime64(3),
    open Float64,
    high Float64,
    low Float64,
    close Float64,
    volume Float64,
    quote_volume Float64,
    trades UInt64
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(open_time)
ORDER BY (exchange, symbol, timeframe, open_time)
TTL open_time + INTERVAL 2 YEAR;

-- Tickers
CREATE TABLE IF NOT EXISTS tickers (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    price Float64,
    bid Float64,
    ask Float64,
    volume_24h Float64,
    change_24h Float64,
    high_24h Float64,
    low_24h Float64,
    funding_rate Nullable(Float64),
    open_interest Nullable(Float64)
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (exchange, symbol, timestamp)
TTL timestamp + INTERVAL 30 DAY;

-- Order Book Snapshots
CREATE TABLE IF NOT EXISTS orderbook_snapshots (
    exchange LowCardinality(String),
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    bids String,  -- JSON array
    asks String,  -- JSON array
    bid_depth Float64,
    ask_depth Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (exchange, symbol, timestamp)
TTL timestamp + INTERVAL 7 DAY;

-- News & Sentiment
CREATE TABLE IF NOT EXISTS news (
    id UUID DEFAULT generateUUIDv4(),
    source LowCardinality(String),
    title String,
    content String,
    url String,
    sentiment Float32,
    symbols Array(LowCardinality(String)),
    published_at DateTime64(3),
    collected_at DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(published_at)
ORDER BY (published_at, source)
TTL published_at + INTERVAL 1 YEAR;

-- Social Sentiment
CREATE TABLE IF NOT EXISTS social_sentiment (
    platform LowCardinality(String),  -- twitter, reddit
    symbol LowCardinality(String),
    timestamp DateTime64(3),
    mentions UInt32,
    sentiment_score Float32,
    positive_count UInt32,
    negative_count UInt32,
    neutral_count UInt32,
    influencer_sentiment Float32,
    trending_rank Nullable(UInt16)
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (platform, symbol, timestamp)
TTL timestamp + INTERVAL 90 DAY;

-- Materialized view for sentiment aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS sentiment_hourly_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (symbol, hour)
AS SELECT
    symbol,
    toStartOfHour(timestamp) as hour,
    avg(sentiment_score) as avg_sentiment,
    sum(mentions) as total_mentions,
    sum(positive_count) as total_positive,
    sum(negative_count) as total_negative
FROM social_sentiment
GROUP BY symbol, hour;
```

---

## 14. Redis Usage

### 14.1 Key Patterns

| Pattern                            | Purpose             | TTL |
| ---------------------------------- | ------------------- | --- |
| `session:{user_id}:{session_id}`   | Agent session state | 24h |
| `lock:{resource}`                  | Distributed locks   | 30s |
| `rate:{user_id}:{action}`          | Rate limiting       | 1m  |
| `cache:ticker:{exchange}:{symbol}` | Price cache         | 5s  |
| `cache:balance:{account_id}`       | Balance cache       | 30s |
| `cache:regime:{symbol}`            | Market regime cache | 5m  |

### 14.2 Redis Client with sqlx-style patterns

```go
// internal/adapters/redis/client.go
package redis

import (
    "context"
    "time"

    "github.com/redis/go-redis/v9"
)

type Client struct {
    rdb *redis.Client
}

func NewClient(cfg Config) (*Client, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr(),
        Password: cfg.Password,
        DB:       cfg.DB,
    })

    if err := rdb.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }

    return &Client{rdb: rdb}, nil
}

// Distributed lock
func (c *Client) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
    return c.rdb.SetNX(ctx, "lock:"+key, "1", ttl).Result()
}

func (c *Client) ReleaseLock(ctx context.Context, key string) error {
    return c.rdb.Del(ctx, "lock:"+key).Err()
}

// Cache with generics
func (c *Client) GetCached(ctx context.Context, key string, dest interface{}) error {
    data, err := c.rdb.Get(ctx, key).Bytes()
    if err != nil {
        return err
    }
    return json.Unmarshal(data, dest)
}

func (c *Client) SetCached(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.rdb.Set(ctx, key, data, ttl).Err()
}
```

---

## 15. Kafka Event Streaming

### 15.1 Topic Definitions

```go
// internal/adapters/kafka/topics.go
package kafka

const (
    // Trading events
    TopicTradeOpened    = "trades.opened"
    TopicTradeClosed    = "trades.closed"
    TopicOrderPlaced    = "orders.placed"
    TopicOrderFilled    = "orders.filled"
    TopicOrderCanceled  = "orders.canceled"

    // Risk events
    TopicRiskAlert      = "risk.alerts"
    TopicCircuitBreaker = "risk.circuit_breaker"
    TopicKillSwitch     = "risk.kill_switch"

    // Market data events
    TopicPriceUpdate    = "market.prices"
    TopicLiquidation    = "market.liquidations"
    TopicWhaleAlert     = "market.whale_alerts"

    // Agent events
    TopicAgentDecision  = "agents.decisions"
    TopicAgentSignal    = "agents.signals"

    // Notifications
    TopicNotifications  = "notifications.telegram"
)
```

### 15.2 Kafka Producer

```go
// internal/adapters/kafka/producer.go
package kafka

import (
    "context"
    "encoding/json"

    "github.com/segmentio/kafka-go"

    "prometheus/pkg/logger"
)

type Producer struct {
    writers map[string]*kafka.Writer
    log     *logger.Logger
}

type ProducerConfig struct {
    Brokers []string
    Async   bool
}

func NewProducer(cfg ProducerConfig) *Producer {
    return &Producer{
        writers: make(map[string]*kafka.Writer),
        log:     logger.Get().With("component", "kafka_producer"),
    }
}

func (p *Producer) getWriter(topic string) *kafka.Writer {
    if w, ok := p.writers[topic]; ok {
        return w
    }

    w := &kafka.Writer{
        Addr:     kafka.TCP(p.cfg.Brokers...),
        Topic:    topic,
        Balancer: &kafka.LeastBytes{},
        Async:    p.cfg.Async,
    }

    p.writers[topic] = w
    return w
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, event interface{}) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }

    msg := kafka.Message{
        Key:   []byte(key),
        Value: data,
    }

    if err := p.getWriter(topic).WriteMessages(ctx, msg); err != nil {
        p.log.Errorf("Failed to publish to %s: %v", topic, err)
        return err
    }

    p.log.Debugf("Published to %s: %s", topic, key)
    return nil
}

func (p *Producer) Close() error {
    for _, w := range p.writers {
        if err := w.Close(); err != nil {
            return err
        }
    }
    return nil
}
```

### 15.3 Kafka Consumer

```go
// internal/adapters/kafka/consumer.go
package kafka

import (
    "context"

    "github.com/segmentio/kafka-go"

    "prometheus/pkg/logger"
)

type Consumer struct {
    reader *kafka.Reader
    log    *logger.Logger
}

type ConsumerConfig struct {
    Brokers  []string
    GroupID  string
    Topic    string
    MinBytes int
    MaxBytes int
}

func NewConsumer(cfg ConsumerConfig) *Consumer {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  cfg.Brokers,
        GroupID:  cfg.GroupID,
        Topic:    cfg.Topic,
        MinBytes: cfg.MinBytes,
        MaxBytes: cfg.MaxBytes,
    })

    return &Consumer{
        reader: reader,
        log:    logger.Get().With("component", "kafka_consumer", "topic", cfg.Topic),
    }
}

type MessageHandler func(ctx context.Context, msg kafka.Message) error

func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            msg, err := c.reader.ReadMessage(ctx)
            if err != nil {
                c.log.Errorf("Failed to read message: %v", err)
                continue
            }

            if err := handler(ctx, msg); err != nil {
                c.log.Errorf("Failed to handle message: %v", err)
                // Could implement retry logic here
            }
        }
    }
}

func (c *Consumer) Close() error {
    return c.reader.Close()
}
```

### 15.4 Event Types

```go
// internal/adapters/kafka/events.go
package kafka

import (
    "time"

    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type TradeEvent struct {
    ID          uuid.UUID       `json:"id"`
    UserID      uuid.UUID       `json:"user_id"`
    Type        string          `json:"type"`  // opened, closed, sl_hit, tp_hit
    Symbol      string          `json:"symbol"`
    Side        string          `json:"side"`
    EntryPrice  decimal.Decimal `json:"entry_price"`
    ExitPrice   decimal.Decimal `json:"exit_price,omitempty"`
    Size        decimal.Decimal `json:"size"`
    PnL         decimal.Decimal `json:"pnl,omitempty"`
    PnLPercent  decimal.Decimal `json:"pnl_percent,omitempty"`
    Reasoning   string          `json:"reasoning"`
    Timestamp   time.Time       `json:"timestamp"`
}

type RiskEvent struct {
    ID        uuid.UUID `json:"id"`
    UserID    uuid.UUID `json:"user_id"`
    EventType string    `json:"event_type"`
    Severity  string    `json:"severity"`
    Message   string    `json:"message"`
    Data      string    `json:"data"`
    Timestamp time.Time `json:"timestamp"`
}

type WhaleAlertEvent struct {
    Exchange  string          `json:"exchange"`
    Symbol    string          `json:"symbol"`
    Side      string          `json:"side"`
    Price     decimal.Decimal `json:"price"`
    Size      decimal.Decimal `json:"size"`
    ValueUSD  decimal.Decimal `json:"value_usd"`
    Timestamp time.Time       `json:"timestamp"`
}

type AgentSignalEvent struct {
    AgentID    string          `json:"agent_id"`
    UserID     uuid.UUID       `json:"user_id"`
    Symbol     string          `json:"symbol"`
    Signal     string          `json:"signal"`  // buy, sell, hold
    Confidence float64         `json:"confidence"`
    Reasoning  string          `json:"reasoning"`
    Timestamp  time.Time       `json:"timestamp"`
}
```

### 15.5 Kafka in Docker Compose

```yaml
# docker-compose.yml additions
services:
  kafka:
    image: bitnami/kafka:3.6
    ports:
      - "9092:9092"
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
    volumes:
      - kafka_data:/bitnami/kafka

  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    ports:
      - "8080:8080"
    environment:
      - KAFKA_CLUSTERS_0_NAME=local
      - KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092
    depends_on:
      - kafka

volumes:
  kafka_data:
```

---

## 16. PostgreSQL with sqlx

### 16.1 Database Client

```go
// internal/adapters/postgres/client.go
package postgres

import (
    "context"
    "fmt"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"

    "prometheus/internal/adapters/config"
)

type Client struct {
    db *sqlx.DB
}

func NewClient(cfg config.PostgresConfig) (*Client, error) {
    db, err := sqlx.Connect("postgres", cfg.DSN())
    if err != nil {
        return nil, fmt.Errorf("failed to connect to postgres: %w", err)
    }

    db.SetMaxOpenConns(cfg.MaxConns)
    db.SetMaxIdleConns(cfg.MaxConns / 2)

    return &Client{db: db}, nil
}

func (c *Client) DB() *sqlx.DB {
    return c.db
}

func (c *Client) Close() error {
    return c.db.Close()
}
```

### 16.2 Repository Interface Pattern

```go
// internal/domain/user/repository.go
package user

import (
    "context"

    "github.com/google/uuid"
)

// Repository interface - defined in domain layer
type Repository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    GetByTelegramID(ctx context.Context, telegramID int64) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, limit, offset int) ([]*User, error)
}
```

### 16.3 Repository Implementation with sqlx

```go
// internal/repository/postgres/user.go
package postgres

import (
    "context"
    "database/sql"
    "errors"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"

    "prometheus/internal/domain/user"
)

// Compile-time check that we implement the interface
var _ user.Repository = (*UserRepository)(nil)

type UserRepository struct {
    db *sqlx.DB
}

// NewUserRepository returns struct, receives interface (via *sqlx.DB)
func NewUserRepository(db *sqlx.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
    query := `
        INSERT INTO users (
            id, telegram_id, telegram_username, first_name, last_name,
            language_code, is_active, is_premium, settings, created_at, updated_at
        ) VALUES (
            :id, :telegram_id, :telegram_username, :first_name, :last_name,
            :language_code, :is_active, :is_premium, :settings, :created_at, :updated_at
        )`

    _, err := r.db.NamedExecContext(ctx, query, u)
    return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
    var u user.User

    query := `SELECT * FROM users WHERE id = $1`

    if err := r.db.GetContext(ctx, &u, query, id); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }

    return &u, nil
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
    var u user.User

    query := `SELECT * FROM users WHERE telegram_id = $1`

    if err := r.db.GetContext(ctx, &u, query, telegramID); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }

    return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
    query := `
        UPDATE users SET
            telegram_username = :telegram_username,
            first_name = :first_name,
            last_name = :last_name,
            language_code = :language_code,
            is_active = :is_active,
            is_premium = :is_premium,
            settings = :settings,
            updated_at = NOW()
        WHERE id = :id`

    _, err := r.db.NamedExecContext(ctx, query, u)
    return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
    query := `DELETE FROM users WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*user.User, error) {
    var users []*user.User

    query := `SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

    if err := r.db.SelectContext(ctx, &users, query, limit, offset); err != nil {
        return nil, err
    }

    return users, nil
}
```

### 16.4 Order Repository Example

```go
// internal/repository/postgres/order.go
package postgres

import (
    "context"
    "database/sql"
    "errors"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"

    "prometheus/internal/domain/order"
)

var _ order.Repository = (*OrderRepository)(nil)

type OrderRepository struct {
    db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
    return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
    query := `
        INSERT INTO orders (
            id, user_id, trading_pair_id, exchange_account_id,
            exchange_order_id, symbol, market_type,
            side, type, status,
            price, amount, filled_amount, avg_fill_price,
            stop_price, reduce_only,
            agent_id, reasoning, parent_order_id,
            fee, fee_currency,
            created_at, updated_at
        ) VALUES (
            :id, :user_id, :trading_pair_id, :exchange_account_id,
            :exchange_order_id, :symbol, :market_type,
            :side, :type, :status,
            :price, :amount, :filled_amount, :avg_fill_price,
            :stop_price, :reduce_only,
            :agent_id, :reasoning, :parent_order_id,
            :fee, :fee_currency,
            :created_at, :updated_at
        )`

    _, err := r.db.NamedExecContext(ctx, query, o)
    return err
}

func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
    var o order.Order

    query := `SELECT * FROM orders WHERE id = $1`

    if err := r.db.GetContext(ctx, &o, query, id); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }

    return &o, nil
}

func (r *OrderRepository) GetOpenByUser(ctx context.Context, userID uuid.UUID) ([]*order.Order, error) {
    var orders []*order.Order

    query := `
        SELECT * FROM orders
        WHERE user_id = $1 AND status IN ('pending', 'open', 'partial')
        ORDER BY created_at DESC`

    if err := r.db.SelectContext(ctx, &orders, query, userID); err != nil {
        return nil, err
    }

    return orders, nil
}

func (r *OrderRepository) GetByExchangeOrderID(ctx context.Context, exchangeOrderID string) (*order.Order, error) {
    var o order.Order

    query := `SELECT * FROM orders WHERE exchange_order_id = $1`

    if err := r.db.GetContext(ctx, &o, query, exchangeOrderID); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }

    return &o, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus, filledAmount, avgPrice decimal.Decimal) error {
    query := `
        UPDATE orders SET
            status = $2,
            filled_amount = $3,
            avg_fill_price = $4,
            updated_at = NOW(),
            filled_at = CASE WHEN $2 = 'filled' THEN NOW() ELSE filled_at END
        WHERE id = $1`

    _, err := r.db.ExecContext(ctx, query, id, status, filledAmount, avgPrice)
    return err
}

// Batch operations with transaction
func (r *OrderRepository) CreateBatch(ctx context.Context, orders []*order.Order) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    query := `
        INSERT INTO orders (
            id, user_id, trading_pair_id, exchange_account_id,
            symbol, side, type, status, price, amount, created_at, updated_at
        ) VALUES (
            :id, :user_id, :trading_pair_id, :exchange_account_id,
            :symbol, :side, :type, :status, :price, :amount, :created_at, :updated_at
        )`

    for _, o := range orders {
        if _, err := tx.NamedExecContext(ctx, query, o); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### 16.5 Memory Repository with pgvector

```go
// internal/repository/postgres/memory.go
package postgres

import (
    "context"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/pgvector/pgvector-go"

    "prometheus/internal/domain/memory"
)

var _ memory.Repository = (*MemoryRepository)(nil)

type MemoryRepository struct {
    db *sqlx.DB
}

func NewMemoryRepository(db *sqlx.DB) *MemoryRepository {
    return &MemoryRepository{db: db}
}

func (r *MemoryRepository) Store(ctx context.Context, m *memory.Memory) error {
    query := `
        INSERT INTO memories (
            id, user_id, agent_id, session_id, type, content, embedding,
            symbol, timeframe, importance, related_ids, trade_id, created_at, expires_at
        ) VALUES (
            :id, :user_id, :agent_id, :session_id, :type, :content, :embedding,
            :symbol, :timeframe, :importance, :related_ids, :trade_id, :created_at, :expires_at
        )`

    _, err := r.db.NamedExecContext(ctx, query, m)
    return err
}

func (r *MemoryRepository) SearchSimilar(ctx context.Context, userID uuid.UUID, embedding pgvector.Vector, limit int) ([]*memory.Memory, error) {
    var memories []*memory.Memory

    // Cosine similarity search using pgvector
    query := `
        SELECT *, 1 - (embedding <=> $2) as similarity
        FROM memories
        WHERE user_id = $1
          AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY embedding <=> $2
        LIMIT $3`

    if err := r.db.SelectContext(ctx, &memories, query, userID, embedding, limit); err != nil {
        return nil, err
    }

    return memories, nil
}

func (r *MemoryRepository) GetByAgent(ctx context.Context, userID uuid.UUID, agentID string, limit int) ([]*memory.Memory, error) {
    var memories []*memory.Memory

    query := `
        SELECT * FROM memories
        WHERE user_id = $1 AND agent_id = $2
          AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY created_at DESC
        LIMIT $3`

    if err := r.db.SelectContext(ctx, &memories, query, userID, agentID, limit); err != nil {
        return nil, err
    }

    return memories, nil
}

func (r *MemoryRepository) GetByType(ctx context.Context, userID uuid.UUID, memType memory.MemoryType, limit int) ([]*memory.Memory, error) {
    var memories []*memory.Memory

    query := `
        SELECT * FROM memories
        WHERE user_id = $1 AND type = $2
          AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY importance DESC, created_at DESC
        LIMIT $3`

    if err := r.db.SelectContext(ctx, &memories, query, userID, memType, limit); err != nil {
        return nil, err
    }

    return memories, nil
}

func (r *MemoryRepository) DeleteExpired(ctx context.Context) (int64, error) {
    result, err := r.db.ExecContext(ctx, `DELETE FROM memories WHERE expires_at < NOW()`)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}
```

### 16.6 Journal Repository

```go
// internal/repository/postgres/journal.go
package postgres

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"

    "prometheus/internal/domain/journal"
)

var _ journal.Repository = (*JournalRepository)(nil)

type JournalRepository struct {
    db *sqlx.DB
}

func NewJournalRepository(db *sqlx.DB) *JournalRepository {
    return &JournalRepository{db: db}
}

func (r *JournalRepository) Create(ctx context.Context, entry *journal.JournalEntry) error {
    query := `
        INSERT INTO journal_entries (
            id, user_id, trade_id, symbol, side,
            entry_price, exit_price, size, pnl, pnl_percent,
            strategy_used, timeframe, setup_type,
            market_regime, entry_reasoning, exit_reasoning, confidence_score,
            rsi_at_entry, atr_at_entry, volume_at_entry,
            was_correct_entry, was_correct_exit, max_drawdown, max_profit, hold_duration,
            lessons_learned, improvement_tips, created_at
        ) VALUES (
            :id, :user_id, :trade_id, :symbol, :side,
            :entry_price, :exit_price, :size, :pnl, :pnl_percent,
            :strategy_used, :timeframe, :setup_type,
            :market_regime, :entry_reasoning, :exit_reasoning, :confidence_score,
            :rsi_at_entry, :atr_at_entry, :volume_at_entry,
            :was_correct_entry, :was_correct_exit, :max_drawdown, :max_profit, :hold_duration,
            :lessons_learned, :improvement_tips, :created_at
        )`

    _, err := r.db.NamedExecContext(ctx, query, entry)
    return err
}

func (r *JournalRepository) GetEntriesSince(ctx context.Context, userID uuid.UUID, since time.Time) ([]journal.JournalEntry, error) {
    var entries []journal.JournalEntry

    query := `
        SELECT * FROM journal_entries
        WHERE user_id = $1 AND created_at >= $2
        ORDER BY created_at DESC`

    if err := r.db.SelectContext(ctx, &entries, query, userID, since); err != nil {
        return nil, err
    }

    return entries, nil
}

func (r *JournalRepository) GetStrategyStats(ctx context.Context, userID uuid.UUID, since time.Time) ([]journal.StrategyStats, error) {
    var stats []journal.StrategyStats

    query := `
        SELECT
            strategy_used as strategy_name,
            COUNT(*) as total_trades,
            SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
            SUM(CASE WHEN pnl <= 0 THEN 1 ELSE 0 END) as losing_trades,
            ROUND(100.0 * SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) / COUNT(*), 2) as win_rate,
            COALESCE(AVG(CASE WHEN pnl > 0 THEN pnl END), 0) as avg_win,
            COALESCE(AVG(CASE WHEN pnl < 0 THEN ABS(pnl) END), 0) as avg_loss,
            CASE
                WHEN SUM(CASE WHEN pnl < 0 THEN ABS(pnl) END) > 0
                THEN SUM(CASE WHEN pnl > 0 THEN pnl END) / SUM(CASE WHEN pnl < 0 THEN ABS(pnl) END)
                ELSE 0
            END as profit_factor,
            MAX(created_at) as last_updated
        FROM journal_entries
        WHERE user_id = $1 AND created_at >= $2
        GROUP BY strategy_used
        ORDER BY total_trades DESC`

    if err := r.db.SelectContext(ctx, &stats, query, userID, since); err != nil {
        return nil, err
    }

    return stats, nil
}
```

---

## 17. Main Entry Point

```go
// cmd/main.go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "prometheus/internal/adapters/ai"
    "prometheus/internal/adapters/clickhouse"
    "prometheus/internal/adapters/config"
    "prometheus/internal/adapters/exchanges"
    "prometheus/internal/adapters/kafka"
    "prometheus/internal/adapters/postgres"
    "prometheus/internal/adapters/redis"
    "prometheus/internal/adapters/telegram"
    "prometheus/internal/adapters/telegram/handlers"
    "prometheus/internal/agents"
    "prometheus/internal/domain/user"
    "prometheus/internal/evaluation"
    "prometheus/internal/repository"
    "prometheus/internal/risk"
    "prometheus/internal/tools"
    "prometheus/internal/workers"
    "prometheus/pkg/logger"
    "prometheus/pkg/templates"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        panic("failed to load config: " + err.Error())
    }

    // Initialize logger
    if err := logger.Init(cfg.App.LogLevel, cfg.App.Env); err != nil {
        panic("failed to init logger: " + err.Error())
    }
    defer logger.Sync()

    log := logger.Get()
    log.Info("Starting Prometheus Trading System")

    // Initialize templates - MUST be before anything else
    templates.Init("pkg/templates/prompts")
    log.Infof("Loaded %d templates", len(templates.Get().List()))

    // Initialize database connections
    pgClient, err := postgres.NewClient(cfg.Postgres)
    if err != nil {
        log.Fatalf("Failed to connect to PostgreSQL: %v", err)
    }
    defer pgClient.Close()

    chClient, err := clickhouse.NewClient(cfg.ClickHouse)
    if err != nil {
        log.Fatalf("Failed to connect to ClickHouse: %v", err)
    }
    defer chClient.Close()

    redisClient, err := redis.NewClient(cfg.Redis)
    if err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }
    defer redisClient.Close()

    // Initialize Kafka
    kafkaProducer := kafka.NewProducer(kafka.ProducerConfig{
        Brokers: cfg.Kafka.Brokers,
        Async:   true,
    })
    defer kafkaProducer.Close()

    // Initialize repositories (using sqlx)
    db := pgClient.DB()
    userRepo := repository.NewUserRepository(db)
    exchangeAccountRepo := repository.NewExchangeAccountRepository(db)
    tradingPairRepo := repository.NewTradingPairRepository(db)
    orderRepo := repository.NewOrderRepository(db)
    positionRepo := repository.NewPositionRepository(db)
    memoryRepo := repository.NewMemoryRepository(db)
    journalRepo := repository.NewJournalRepository(db)
    riskStateRepo := repository.NewRiskStateRepository(db)
    marketDataRepo := repository.NewMarketDataRepository(chClient)
    sentimentRepo := repository.NewSentimentRepository(chClient)

    // Initialize services
    userService := user.NewService(userRepo)

    // Initialize Risk Engine
    riskEngine := risk.NewEngine(riskStateRepo, userRepo, kafkaProducer)

    // Initialize Self-Evaluator
    evaluator := evaluation.NewEvaluator(journalRepo, tradingPairRepo)

    // Initialize AI providers
    aiRegistry := ai.SetupProviders(cfg.AI)
    log.Infof("Registered AI providers: %v", aiRegistry.ListProviders())

    // Initialize exchange factory
    exchangeFactory := exchanges.NewFactory()

    // Initialize tools registry
    toolRegistry := tools.NewRegistry()
    tools.RegisterAllTools(toolRegistry, tools.ToolDeps{
        MarketDataRepo:  marketDataRepo,
        SentimentRepo:   sentimentRepo,
        ExchangeFactory: exchangeFactory,
        MemoryRepo:      memoryRepo,
        OrderRepo:       orderRepo,
        PositionRepo:    positionRepo,
        RiskEngine:      riskEngine,
        JournalRepo:     journalRepo,
        KafkaProducer:   kafkaProducer,
    })
    log.Infof("Registered %d tools", len(toolRegistry.List()))

    // Initialize agent factory
    agentFactory := agents.NewFactory(agents.FactoryDeps{
        AIRegistry:   aiRegistry,
        ToolRegistry: toolRegistry,
        Templates:    templates.Get(),
    })

    // Initialize workers
    workerList := []workers.Worker{
        // Market Data
        workers.NewOHLCVCollector(exchangeFactory, marketDataRepo, workers.OHLCVCollectorConfig{
            Symbols:    []string{"BTC/USDT", "ETH/USDT"},
            Timeframes: []string{"1m", "5m", "15m", "1h", "4h", "1d"},
            Interval:   time.Minute,
            Enabled:    true,
        }),
        workers.NewTickerCollector(exchangeFactory, marketDataRepo, 5*time.Second),
        workers.NewOrderbookCollector(exchangeFactory, marketDataRepo, 10*time.Second),
        workers.NewTradesCollector(exchangeFactory, marketDataRepo, time.Second),

        // Order Flow
        workers.NewLiquidationCollector(marketDataRepo, 5*time.Second),
        workers.NewWhaleAlertCollector(marketDataRepo, kafkaProducer, 10*time.Second),

        // Sentiment
        workers.NewNewsCollector(sentimentRepo, 5*time.Minute),
        workers.NewTwitterCollector(sentimentRepo, 2*time.Minute),
        workers.NewFearGreedCollector(sentimentRepo, 15*time.Minute),

        // Macro
        workers.NewCalendarCollector(sentimentRepo, time.Hour),
        workers.NewCorrelationCollector(marketDataRepo, 5*time.Minute),

        // Derivatives
        workers.NewOptionsCollector(marketDataRepo, 5*time.Minute),

        // Trading
        workers.NewPositionMonitor(positionRepo, exchangeFactory, riskEngine, 30*time.Second),
        workers.NewOrderSync(orderRepo, exchangeFactory, kafkaProducer, 10*time.Second),
        workers.NewPnLCalculator(positionRepo, 1*time.Minute),
        workers.NewRiskMonitor(riskEngine, 10*time.Second),

        // Analysis
        workers.NewMarketScanner(tradingPairRepo, agentFactory, riskEngine, 5*time.Minute),
        workers.NewOpportunityFinder(agentFactory, kafkaProducer, 1*time.Minute),
        workers.NewRegimeDetector(marketDataRepo, memoryRepo, 5*time.Minute),
        workers.NewSMCScanner(marketDataRepo, 1*time.Minute),

        // Evaluation
        workers.NewDailyReportWorker(evaluator, kafkaProducer, 24*time.Hour),
        workers.NewStrategyEvaluator(evaluator, 6*time.Hour),
    }

    scheduler := workers.NewScheduler(workerList)

    // Initialize Kafka consumers
    notificationConsumer := kafka.NewConsumer(kafka.ConsumerConfig{
        Brokers: cfg.Kafka.Brokers,
        GroupID: cfg.Kafka.GroupID,
        Topic:   kafka.TopicNotifications,
    })
    defer notificationConsumer.Close()

    riskConsumer := kafka.NewConsumer(kafka.ConsumerConfig{
        Brokers: cfg.Kafka.Brokers,
        GroupID: cfg.Kafka.GroupID,
        Topic:   kafka.TopicKillSwitch,
    })
    defer riskConsumer.Close()

    // Initialize Telegram bot handlers
    handlerRegistry := handlers.NewRegistry()
    handlers.RegisterAllHandlers(handlerRegistry, handlers.HandlerDeps{
        UserService:         userService,
        ExchangeAccountRepo: exchangeAccountRepo,
        TradingPairRepo:     tradingPairRepo,
        ExchangeFactory:     exchangeFactory,
        RiskEngine:          riskEngine,
        Evaluator:           evaluator,
        Templates:           templates.Get(),
    })

    // Initialize Telegram bot
    bot, err := telegram.NewBot(telegram.BotDeps{
        Token:       cfg.Telegram.BotToken,
        UserService: userService,
        Handlers:    handlerRegistry,
    })
    if err != nil {
        log.Fatalf("Failed to create Telegram bot: %v", err)
    }

    // Create context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Start workers
    scheduler.Start(ctx)
    log.Info("Workers started")

    // Start Kafka consumers
    go notificationConsumer.Consume(ctx, handleNotification(bot))
    go riskConsumer.Consume(ctx, handleKillSwitch(exchangeFactory, positionRepo))
    log.Info("Kafka consumers started")

    // Start Telegram bot
    go func() {
        if err := bot.Start(ctx); err != nil {
            log.Errorf("Telegram bot error: %v", err)
        }
    }()
    log.Info("Telegram bot started")

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    <-quit
    log.Info("Shutting down...")

    cancel()
    scheduler.Wait()

    log.Info("Shutdown complete")
}

// Kafka message handlers
func handleNotification(bot *telegram.Bot) kafka.MessageHandler {
    return func(ctx context.Context, msg kafka.Message) error {
        // Parse notification and send via Telegram
        return bot.SendNotification(ctx, msg.Value)
    }
}

func handleKillSwitch(factory exchanges.Factory, positionRepo position.Repository) kafka.MessageHandler {
    return func(ctx context.Context, msg kafka.Message) error {
        var event kafka.RiskEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            return err
        }

        // Close all positions for user
        positions, _ := positionRepo.GetOpenByUser(ctx, event.UserID)
        for _, pos := range positions {
            client, _ := factory.GetClientByAccountID(pos.ExchangeAccountID)
            client.ClosePosition(ctx, pos.Symbol, 100)
        }

        return nil
    }
}
```

---

## 18. Docker & Deployment

### 16.1 Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /prometheus ./cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /prometheus .

# Copy templates
COPY pkg/templates/prompts ./pkg/templates/prompts

# Copy migrations
COPY migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./prometheus"]
```

### 17.2 Docker Compose

```yaml
# docker-compose.yml
version: "3.8"

services:
  app:
    build: .
    env_file: .env
    depends_on:
      - postgres
      - clickhouse
      - redis
      - kafka
    restart: unless-stopped
    ports:
      - "8081:8081"

  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: trading
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: prometheus
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  clickhouse:
    image: clickhouse/clickhouse-server:24.1
    environment:
      CLICKHOUSE_DB: trading
    volumes:
      - clickhouse_data:/var/lib/clickhouse
    ports:
      - "8123:8123"
      - "9000:9000"

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

  kafka:
    image: bitnami/kafka:3.6
    ports:
      - "9092:9092"
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
    volumes:
      - kafka_data:/bitnami/kafka

  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    ports:
      - "8080:8080"
    environment:
      - KAFKA_CLUSTERS_0_NAME=local
      - KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092
    depends_on:
      - kafka

volumes:
  postgres_data:
  clickhouse_data:
  redis_data:
  kafka_data:
```

---

## 19. Testing Strategy

### 17.1 Test Structure

```
tests/
├── unit/
│   ├── tools/
│   ├── agents/
│   └── domain/
├── integration/
│   ├── repository/
│   ├── exchanges/
│   └── ai/
├── e2e/
│   └── trading_flow_test.go
└── fixtures/
    ├── market_data.json
    └── orders.json
```

### 17.2 Testing Approach

- **Unit Tests**: All business logic, tools, indicators
- **Integration Tests**: Repository, exchange adapters (with testnet)
- **E2E Tests**: Full trading flow with mock exchange
- **Agent Tests**: ADK evaluation framework

---

## 20. Monitoring & Observability

### 18.1 Metrics (Prometheus)

- `trading_orders_total{exchange, symbol, side, status}`
- `trading_pnl_total{user_id, symbol}`
- `agent_execution_duration_seconds{agent}`
- `agent_tool_calls_total{agent, tool}`
- `exchange_api_latency_seconds{exchange, endpoint}`
- `worker_execution_duration_seconds{worker}`

### 18.2 Logging

- Structured JSON logs (Zap)
- Correlation IDs for request tracing
- Agent session logging
- Trade audit log

### 18.3 Alerting

- Position liquidation risk
- Large drawdown
- API errors
- System health

---

## 21. Security Considerations

### 19.1 API Key Storage

- AES-256-GCM encryption for exchange API keys
- Keys never logged
- Passphrase stored separately

### 19.2 Access Control

- Telegram ID authentication
- Rate limiting per user
- Admin commands restricted

### 19.3 Trading Safety

- Max position size limits
- Mandatory stop-loss
- Daily loss limits
- Kill switch

---

## 22. Future Enhancements

1. **Web Dashboard** - React dashboard for analytics
2. **Backtesting Engine** - Historical strategy testing
3. **Strategy Marketplace** - Share/sell strategies
4. **Copy Trading** - Follow successful traders
5. **Advanced Analytics** - ML-based predictions
6. **Mobile App** - Native iOS/Android
7. **Multi-user Workspaces** - Team trading

---

## 23. ML Integration Roadmap

### 21.1 Architecture Philosophy

```
┌─────────────────────────────────────────────────────────────────┐
│                     DECISION LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│  LLM Agents (Reasoning, Context, Synthesis)                     │
│  - Understands market context                                    │
│  - Makes nuanced decisions                                       │
│  - Explains reasoning                                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SIGNAL LAYER (ML)                            │
├─────────────────────────────────────────────────────────────────┤
│  ML Models (Pattern Recognition, Prediction)                     │
│  - Market regime detection                                       │
│  - Anomaly detection                                             │
│  - Price prediction signals                                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     RULE LAYER                                   │
├─────────────────────────────────────────────────────────────────┤
│  Hard-coded Rules (Safety, Compliance)                          │
│  - Circuit breakers                                              │
│  - Position limits                                               │
│  - Kill switch                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### 21.2 Phase 1: LLM-Only (MVP)

**Timeline**: Month 1-2

```go
// No ML, pure LLM reasoning
type Phase1System struct {
    agents     *agents.Registry      // LLM agents
    tools      *tools.Registry       // Data tools
    riskEngine *risk.Engine          // Rule-based
}
```

**Capabilities**:

- LLM analyzes data from tools
- Rule-based risk management
- Template-based pattern matching
- Human-readable reasoning

**Limitations**:

- No real-time pattern discovery
- Manual regime detection
- Limited to patterns LLM knows

### 21.3 Phase 2: LLM + Simple ML

**Timeline**: Month 3-4

```go
// Add lightweight ML models
type Phase2System struct {
    agents       *agents.Registry
    tools        *tools.Registry
    riskEngine   *risk.Engine
    regimeModel  *ml.RegimeDetector    // NEW
    anomalyModel *ml.AnomalyDetector   // NEW
}
```

**New ML Components**:

```go
// internal/ml/regime/detector.go
package regime

import (
    "github.com/owulveryck/onnx-go"
)

type RegimeDetector struct {
    model *onnx.Model
}

type RegimeInput struct {
    ATR14      float64
    ADX        float64
    BBWidth    float64
    RSI        float64
    Volume24h  float64
    PriceChange24h float64
}

type RegimeOutput struct {
    Regime     string   // trend_up, trend_down, range, volatile
    Confidence float64
    Features   map[string]float64
}

func (d *RegimeDetector) Predict(input RegimeInput) (*RegimeOutput, error) {
    // Run ONNX inference
    // Model trained offline with sklearn/XGBoost -> exported to ONNX
}
```

**Models to add**:

| Model                    | Type             | Input                     | Output              | Training       |
| ------------------------ | ---------------- | ------------------------- | ------------------- | -------------- |
| **Regime Detector**      | XGBoost/RF       | ATR, ADX, BB, RSI, Volume | regime + confidence | Sklearn → ONNX |
| **Anomaly Detector**     | Isolation Forest | Volume, price change, OI  | is_anomaly + score  | Sklearn → ONNX |
| **Volatility Predictor** | GARCH            | Price history             | vol_forecast        | Arch (Python)  |

**Integration with Agents**:

```go
// Tool that uses ML model
func NewGetMarketRegimeTool(detector *ml.RegimeDetector, repo market_data.Repository) tool.Tool {
    return functiontool.New(
        "get_market_regime",
        "Detect current market regime using ML model",
        func(ctx tool.Context, args GetRegimeArgs) (GetRegimeResult, error) {
            // 1. Fetch recent data
            candles, _ := repo.GetOHLCV(ctx, args.Symbol, "1h", 100)

            // 2. Calculate features
            input := calculateFeatures(candles)

            // 3. Run ML inference
            output, err := detector.Predict(input)

            return GetRegimeResult{
                Regime:     output.Regime,
                Confidence: output.Confidence,
            }, err
        },
    )
}
```

### 21.4 Phase 3: Advanced ML

**Timeline**: Month 5-6+

**New Components**:

| Model                 | Purpose                         | Architecture                  |
| --------------------- | ------------------------------- | ----------------------------- |
| **Price Predictor**   | 1h/4h price direction           | Transformer (Temporal Fusion) |
| **Pattern Detector**  | Chart patterns (H&S, triangles) | CNN on candlestick images     |
| **Optimal Execution** | Minimize slippage               | RL (PPO/DDPG)                 |
| **Strategy Selector** | Choose best strategy            | Contextual Bandit             |

**Infrastructure**:

```yaml
# ML service (separate from main app)
ml-service:
  image: prometheus-ml
  volumes:
    - ./models:/models
  environment:
    - MODEL_PATH=/models
    - INFERENCE_DEVICE=cpu # or cuda
  ports:
    - "8000:8000" # gRPC for fast inference
```

```go
// internal/ml/client.go
type MLClient struct {
    conn *grpc.ClientConn
}

func (c *MLClient) PredictRegime(ctx context.Context, features []float64) (*RegimeOutput, error) {
    // Call ML service via gRPC
}

func (c *MLClient) PredictPrice(ctx context.Context, candles []OHLCV) (*PriceOutput, error) {
    // Call ML service via gRPC
}
```

### 21.5 ML Training Pipeline

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ ClickHouse  │───▶│  Feature    │───▶│   Train     │───▶│   Export    │
│ (Raw Data)  │    │  Pipeline   │    │   Model     │    │   ONNX      │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                               │
                                                               ▼
                                                      ┌─────────────┐
                                                      │   Deploy    │
                                                      │   to App    │
                                                      └─────────────┘
```

**Training scripts** (Python, run offline):

```python
# scripts/train_regime_model.py
import pandas as pd
from sklearn.ensemble import RandomForestClassifier
import onnxmltools

# Load data from ClickHouse
df = load_training_data()

# Features
features = ['atr_14', 'adx', 'bb_width', 'rsi', 'volume_change']
X = df[features]
y = df['regime_label']

# Train
model = RandomForestClassifier(n_estimators=100)
model.fit(X, y)

# Export to ONNX
onnx_model = onnxmltools.convert_sklearn(model)
onnxmltools.utils.save_model(onnx_model, 'models/regime_detector.onnx')
```

### 21.6 When to Use What

| Task                       | LLM               | Rule-based         | ML         |
| -------------------------- | ----------------- | ------------------ | ---------- |
| **Reasoning about market** | ✅ Primary        | ❌                 | ❌         |
| **Explaining decisions**   | ✅ Primary        | ❌                 | ❌         |
| **Risk limits**            | ❌                | ✅ Primary         | ❌         |
| **Circuit breakers**       | ❌                | ✅ Primary         | ❌         |
| **Pattern discovery**      | ⚠️ Limited        | ❌                 | ✅ Primary |
| **Regime detection**       | ⚠️ Can help       | ⚠️ Possible        | ✅ Primary |
| **Price prediction**       | ❌ Bad at numbers | ❌                 | ✅ Primary |
| **Anomaly detection**      | ⚠️ Slow           | ⚠️ Threshold-based | ✅ Primary |
| **Execution optimization** | ❌                | ⚠️ TWAP/VWAP       | ✅ RL best |

### 21.7 Recommended MVP Approach

**Start without ML. Add later based on needs.**

```go
// Phase 1: Market regime as tool with rules
func detectRegimeRuleBased(candles []OHLCV) string {
    atr := calculateATR(candles, 14)
    adx := calculateADX(candles, 14)

    // Simple rules
    if adx > 25 {
        if candles[len(candles)-1].Close > candles[len(candles)-20].Close {
            return "trend_up"
        }
        return "trend_down"
    }

    if atr < avgATR * 0.7 {
        return "low_volatility"
    }

    return "range"
}
```

Then replace with ML when you have:

1. Enough labeled data (3+ months)
2. Clear performance baseline
3. Identified patterns LLM misses

---

## 24. Development Phases

### Phase 1: Core MVP (Weeks 1-4)

- [ ] Project structure, config, logger
- [ ] PostgreSQL + ClickHouse setup
- [ ] User registration via Telegram
- [ ] Exchange connection (Binance, Bybit)
- [ ] Basic market data collection
- [ ] Core indicators (RSI, MACD, BB)
- [ ] 3 core agents (Market, Risk, Executor)
- [ ] Basic trading flow

### Phase 2: Full Agent System (Weeks 5-8)

- [ ] All 13 agents
- [ ] Full tool set (80+ tools)
- [ ] SMC/ICT tools
- [ ] Order flow analysis
- [ ] Memory system with pgvector
- [ ] Template system
- [ ] Risk Engine with circuit breaker

### Phase 3: Advanced Features (Weeks 9-12)

- [ ] Macro data integration
- [ ] Derivatives data
- [ ] Self-evaluation system
- [ ] Trade journal
- [ ] Daily reports
- [ ] Kill switch
- [ ] Advanced order types

### Phase 4: ML Integration (Weeks 13-16)

- [ ] Regime detection model
- [ ] Anomaly detection
- [ ] Training pipeline
- [ ] Model serving
- [ ] Continuous evaluation

---

## Appendix A: Environment Variables Reference

| Variable                      | Required | Default     | Description                             |
| ----------------------------- | -------- | ----------- | --------------------------------------- |
| `APP_ENV`                     | No       | development | Environment                             |
| `LOG_LEVEL`                   | No       | info        | Log level                               |
| `POSTGRES_HOST`               | Yes      | -           | PostgreSQL host                         |
| `POSTGRES_PORT`               | No       | 5432        | PostgreSQL port                         |
| `POSTGRES_USER`               | Yes      | -           | PostgreSQL user                         |
| `POSTGRES_PASSWORD`           | Yes      | -           | PostgreSQL password                     |
| `POSTGRES_DB`                 | Yes      | -           | PostgreSQL database                     |
| `CLICKHOUSE_HOST`             | Yes      | -           | ClickHouse host                         |
| `CLICKHOUSE_PORT`             | No       | 9000        | ClickHouse port                         |
| `REDIS_HOST`                  | Yes      | -           | Redis host                              |
| `REDIS_PORT`                  | No       | 6379        | Redis port                              |
| `TELEGRAM_BOT_TOKEN`          | Yes      | -           | Telegram bot token                      |
| `CLAUDE_API_KEY`              | No       | -           | Claude API key                          |
| `CLAUDE_API_KEY`              | No       | -           | Claude API key                          |
| `OPENAI_API_KEY`              | No       | -           | OpenAI API key                          |
| `DEEPSEEK_API_KEY`            | No       | -           | DeepSeek API key                        |
| `GEMINI_API_KEY`              | No       | -           | Gemini API key                          |
| `ENCRYPTION_KEY`              | Yes      | -           | 32-byte encryption key                  |
| `BINANCE_MARKET_DATA_API_KEY` | No       | -           | Central Binance API key for market data |
| `BINANCE_MARKET_DATA_SECRET`  | No       | -           | Central Binance secret for market data  |
| `BYBIT_MARKET_DATA_API_KEY`   | No       | -           | Central Bybit API key for market data   |
| `BYBIT_MARKET_DATA_SECRET`    | No       | -           | Central Bybit secret for market data    |
| `OKX_MARKET_DATA_API_KEY`     | No       | -           | Central OKX API key for market data     |
| `OKX_MARKET_DATA_SECRET`      | No       | -           | Central OKX secret for market data      |
| `OKX_MARKET_DATA_PASSPHRASE`  | No       | -           | Central OKX passphrase for market data  |
| `ERROR_TRACKING_ENABLED`      | No       | true        | Enable error tracking                   |
| `ERROR_TRACKING_PROVIDER`     | No       | sentry      | Error tracking provider                 |
| `SENTRY_DSN`                  | No       | -           | Sentry DSN for error tracking           |
| `SENTRY_ENVIRONMENT`          | No       | production  | Sentry environment name                 |

---

## 25. Architecture Refinements

### 25.1 Multi-User Background Agent System

**Принцип работы:**

- Агенты работают в фоне как единый пул, обслуживая всех пользователей одновременно
- Каждый пользователь имеет изолированные данные (бюджет, позиции, память)
- Агенты обрабатывают запросы для всех активных пользователей в параллельном режиме

```go
// internal/workers/analysis/market_scanner.go
type MarketScanner struct {
    workers.BaseWorker
    tradingPairRepo trading_pair.Repository
    agentFactory    *agents.Factory
    log             *logger.Logger
}

func (w *MarketScanner) Run(ctx context.Context) error {
    // Get ALL active trading pairs from ALL users
    pairs, err := w.tradingPairRepo.FindAllActive(ctx)
    if err != nil {
        return err
    }

    // Process in parallel batches
    semaphore := make(chan struct{}, 10) // Max 10 concurrent analyses
    var wg sync.WaitGroup

    for _, pair := range pairs {
        if pair.IsPaused {
            continue
        }

        wg.Add(1)
        go func(pair *trading_pair.TradingPair) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            // Create pipeline for this user's pair
            pipeline, err := w.agentFactory.CreateTradingPipeline(agents.UserAgentConfig{
                UserID:     pair.UserID.String(),
                AIProvider: pair.AIProvider,
                Symbol:     pair.Symbol,
                MarketType: string(pair.MarketType),
            })
            if err != nil {
                w.log.Errorf("Failed to create pipeline for user %s, pair %s: %v",
                    pair.UserID, pair.Symbol, err)
                return
            }

            // Execute analysis
            w.log.Infof("Analyzing %s for user %s", pair.Symbol, pair.UserID)
            // ... execute pipeline
        }(pair)
    }

    wg.Wait()
    return nil
}
```

### 25.2 Simplified Telegram Commands

**Новая структура команд:**

```go
// internal/adapters/telegram/handlers/connect.go
type ConnectHandler struct {
    exchangeAccountRepo exchange_account.Repository
    exchangeFactory     exchanges.Factory
}

func (h *ConnectHandler) Command() string {
    return "connect"
}

func (h *ConnectHandler) Handle(ctx context.Context, api *tgbotapi.BotAPI, update tgbotapi.Update, u *user.User) {
    // Parse: /connect bybit {API_KEY} {API_SECRET}
    // Or: /connect bybit (then ask for keys in separate messages)

    args := strings.Fields(update.Message.Text)
    if len(args) < 2 {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            "Usage: /connect <exchange> <api_key> <api_secret>\nExample: /connect bybit abc123 xyz789"))
        return
    }

    exchangeName := args[1]
    var apiKey, apiSecret string

    if len(args) >= 4 {
        apiKey = args[2]
        apiSecret = args[3]
    } else {
        // Start conversation flow to get keys securely
        // Store conversation state in Redis
        // Ask for keys in separate messages
    }

    // Create exchange account
    account := &exchange_account.ExchangeAccount{
        UserID:          u.ID,
        Exchange:        exchange_account.ExchangeType(exchangeName),
        APIKeyEncrypted: encrypt(apiKey),
        SecretEncrypted: encrypt(apiSecret),
        IsActive:        true,
    }

    // Test connection
    client, err := h.exchangeFactory.CreateClient(account)
    if err != nil {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            fmt.Sprintf("❌ Failed to connect: %v", err)))
        return
    }

    // Verify connection
    _, err = client.GetBalance(ctx)
    if err != nil {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            fmt.Sprintf("❌ Connection test failed: %v", err)))
        return
    }

    // Save to DB
    if err := h.exchangeAccountRepo.Create(ctx, account); err != nil {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            fmt.Sprintf("❌ Failed to save: %v", err)))
        return
    }

    api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
        fmt.Sprintf("✅ Connected to %s successfully!", exchangeName)))
}
```

```go
// internal/adapters/telegram/handlers/open_position.go
type OpenPositionHandler struct {
    tradingPairRepo trading_pair.Repository
    exchangeAccountRepo exchange_account.Repository
}

func (h *OpenPositionHandler) Command() string {
    return "open-position" // or "add-budget"
}

func (h *OpenPositionHandler) Handle(ctx context.Context, api *tgbotapi.BotAPI, update tgbotapi.Update, u *user.User) {
    // Parse: /open-position BTC 100
    // Format: /open-position <COIN> <AMOUNT_USD>

    args := strings.Fields(update.Message.Text)
    if len(args) < 3 {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            "Usage: /open-position <COIN> <AMOUNT_USD>\nExample: /open-position BTC 100"))
        return
    }

    coin := strings.ToUpper(args[1])
    amountStr := args[2]

    amount, err := decimal.NewFromString(amountStr)
    if err != nil {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            fmt.Sprintf("❌ Invalid amount: %s", amountStr)))
        return
    }

    // Get user's default exchange account
    accounts, err := h.exchangeAccountRepo.GetByUser(ctx, u.ID)
    if err != nil || len(accounts) == 0 {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            "❌ No exchange accounts found. Use /connect first."))
        return
    }

    // Use first active account (or let user choose)
    account := accounts[0]
    symbol := fmt.Sprintf("%s/USDT", coin)

    // Create trading pair
    pair := &trading_pair.TradingPair{
        UserID:            u.ID,
        ExchangeAccountID: account.ID,
        Symbol:            symbol,
        MarketType:        trading_pair.MarketSpot, // or futures
        Budget:            amount,
        IsActive:          true,
        IsPaused:          false,
    }

    if err := h.tradingPairRepo.Create(ctx, pair); err != nil {
        api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
            fmt.Sprintf("❌ Failed to create position: %v", err)))
        return
    }

    api.Send(tgbotapi.NewMessage(update.Message.Chat.ID,
        fmt.Sprintf("✅ Position opened: %s with budget $%s\nAgents will start trading automatically.",
            symbol, amount.String())))
}
```

**Дополнительные команды:**

```go
// /update-budget BTC 200 - обновить бюджет
// /stop-trading BTC - остановить трейдинг для пары
// /resume-trading BTC - возобновить трейдинг
// /list-positions - показать все активные позиции
```

### 25.3 Database ENUMs

**PostgreSQL ENUM types:**

```sql
-- migrations/postgres/002_add_enums.up.sql

-- Exchange types
CREATE TYPE exchange_type AS ENUM ('binance', 'bybit', 'okx', 'kucoin', 'gate');

-- Market types
CREATE TYPE market_type AS ENUM ('spot', 'futures');

-- Order sides
CREATE TYPE order_side AS ENUM ('buy', 'sell');

-- Order types
CREATE TYPE order_type AS ENUM ('market', 'limit', 'stop_market', 'stop_limit');

-- Order statuses
CREATE TYPE order_status AS ENUM (
    'pending',
    'open',
    'filled',
    'partial',
    'canceled',
    'rejected',
    'expired'
);

-- Position sides
CREATE TYPE position_side AS ENUM ('long', 'short');

-- Position statuses
CREATE TYPE position_status AS ENUM ('open', 'closed', 'liquidated');

-- Memory types
CREATE TYPE memory_type AS ENUM (
    'observation',
    'decision',
    'trade',
    'lesson',
    'regime',
    'pattern'
);

-- Risk event types
CREATE TYPE risk_event_type AS ENUM (
    'drawdown_warning',
    'consecutive_loss',
    'circuit_breaker_triggered',
    'max_exposure_reached',
    'kill_switch_activated',
    'anomaly_detected'
);

-- Update existing tables to use ENUMs
ALTER TABLE exchange_accounts
    ALTER COLUMN exchange TYPE exchange_type USING exchange::exchange_type;

ALTER TABLE trading_pairs
    ALTER COLUMN market_type TYPE market_type USING market_type::market_type;

ALTER TABLE orders
    ALTER COLUMN side TYPE order_side USING side::order_side,
    ALTER COLUMN type TYPE order_type USING type::order_type,
    ALTER COLUMN status TYPE order_status USING status::order_status,
    ALTER COLUMN market_type TYPE market_type USING market_type::market_type;

ALTER TABLE positions
    ALTER COLUMN side TYPE position_side USING side::position_side,
    ALTER COLUMN status TYPE position_status USING status::position_status,
    ALTER COLUMN market_type TYPE market_type USING market_type::market_type;

ALTER TABLE memories
    ALTER COLUMN type TYPE memory_type USING type::memory_type;

ALTER TABLE risk_events
    ALTER COLUMN event_type TYPE risk_event_type USING event_type::risk_event_type;
```

**Go code with ENUMs:**

```go
// internal/domain/order/entity.go
type OrderSide string

const (
    OrderSideBuy  OrderSide = "buy"
    OrderSideSell OrderSide = "sell"
)

func (s OrderSide) String() string { return string(s) }

func (s OrderSide) Valid() bool {
    return s == OrderSideBuy || s == OrderSideSell
}

// Repository uses ENUM in queries
func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
    query := `
        INSERT INTO orders (
            id, user_id, symbol, side, type, status, amount, price, created_at
        ) VALUES (
            :id, :user_id, :symbol, :side::order_side, :type::order_type,
            :status::order_status, :amount, :price, :created_at
        )`
    // ...
}
```

### 25.4 Agent Reasoning Log

**Таблица для логирования размышлений агентов:**

```sql
-- migrations/postgres/003_agent_reasoning_log.up.sql

CREATE TABLE agent_reasoning_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id VARCHAR(100) NOT NULL,
    session_id VARCHAR(255),

    -- Context
    symbol VARCHAR(50),
    trading_pair_id UUID REFERENCES trading_pairs(id),

    -- Reasoning steps
    reasoning_steps JSONB NOT NULL, -- Array of reasoning steps
    /*
    [
        {
            "step": 1,
            "action": "tool_call",
            "tool": "get_price",
            "input": {"symbol": "BTC/USDT"},
            "output": {"price": 45000},
            "timestamp": "2025-11-25T10:00:00Z"
        },
        {
            "step": 2,
            "action": "reasoning",
            "thought": "Price is above 200 EMA, bullish signal",
            "timestamp": "2025-11-25T10:00:01Z"
        },
        {
            "step": 3,
            "action": "tool_call",
            "tool": "rsi",
            "input": {"symbol": "BTC/USDT", "period": 14},
            "output": {"rsi": 65, "signal": "neutral"},
            "timestamp": "2025-11-25T10:00:02Z"
        }
    ]
    */

    -- Final decision
    decision JSONB, -- Final decision made by agent
    confidence DECIMAL(5,2), -- 0-100

    -- Performance
    tokens_used INTEGER,
    cost_usd DECIMAL(10,6),
    duration_ms INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reasoning_logs_user ON agent_reasoning_logs(user_id);
CREATE INDEX idx_reasoning_logs_agent ON agent_reasoning_logs(agent_id);
CREATE INDEX idx_reasoning_logs_session ON agent_reasoning_logs(session_id);
CREATE INDEX idx_reasoning_logs_symbol ON agent_reasoning_logs(symbol);
CREATE INDEX idx_reasoning_logs_created ON agent_reasoning_logs(created_at DESC);
```

**Интеграция в агентов:**

```go
// internal/agents/reasoning_logger.go
package agents

import (
    "context"
    "encoding/json"
    "time"

    "github.com/google/uuid"
    "prometheus/internal/domain/reasoning"
    "prometheus/pkg/logger"
)

type ReasoningLogger struct {
    repo reasoning.Repository
    log  *logger.Logger
}

type ReasoningStep struct {
    Step      int                    `json:"step"`
    Action    string                 `json:"action"` // "tool_call", "reasoning", "decision"
    Tool      string                 `json:"tool,omitempty"`
    Input     map[string]interface{} `json:"input,omitempty"`
    Output    map[string]interface{} `json:"output,omitempty"`
    Thought   string                 `json:"thought,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

func (l *ReasoningLogger) LogStep(ctx context.Context, entry *reasoning.LogEntry) error {
    return l.repo.Create(ctx, entry)
}

// Wrapper for agent execution
func (l *ReasoningLogger) WrapAgentExecution(
    ctx context.Context,
    agentID string,
    userID uuid.UUID,
    symbol string,
    fn func() (interface{}, error),
) (interface{}, error) {
    sessionID := uuid.New().String()
    steps := []ReasoningStep{}
    startTime := time.Now()

    // Intercept tool calls and reasoning
    // This would be integrated with ADK agent callbacks

    result, err := fn()

    duration := time.Since(startTime)

    // Save log
    stepsJSON, _ := json.Marshal(steps)
    entry := &reasoning.LogEntry{
        UserID:        userID,
        AgentID:       agentID,
        SessionID:     sessionID,
        Symbol:        symbol,
        ReasoningSteps: stepsJSON,
        DurationMs:    int(duration.Milliseconds()),
    }

    if err := l.LogStep(ctx, entry); err != nil {
        l.log.Errorf("Failed to log reasoning: %v", err)
    }

    return result, err
}
```

### 25.5 Tool Usage Statistics

**Таблица статистики использования инструментов:**

```sql
-- migrations/postgres/004_tool_usage_stats.up.sql

CREATE TABLE tool_usage_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id VARCHAR(100) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,

    -- Metrics
    call_count INTEGER DEFAULT 0,
    total_duration_ms INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,

    -- Time window
    time_window TIMESTAMPTZ NOT NULL, -- Hour or day bucket

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, agent_id, tool_name, time_window)
);

CREATE INDEX idx_tool_stats_user ON tool_usage_stats(user_id);
CREATE INDEX idx_tool_stats_agent ON tool_usage_stats(agent_id);
CREATE INDEX idx_tool_stats_tool ON tool_usage_stats(tool_name);
CREATE INDEX idx_tool_stats_window ON tool_usage_stats(time_window DESC);
```

**Интеграция в tools:**

```go
// internal/tools/middleware/stats.go
package middleware

import (
    "context"
    "time"

    "google.golang.org/adk/tool"
    "prometheus/internal/domain/stats"
)

type StatsMiddleware struct {
    statsRepo stats.Repository
}

func (m *StatsMiddleware) WrapTool(t tool.Tool, userID, agentID string) tool.Tool {
    return tool.New(
        t.Name(),
        t.Description(),
        func(ctx tool.Context, args interface{}) (interface{}, error) {
            start := time.Now()

            // Execute tool
            result, err := t.Execute(ctx, args)

            duration := time.Since(start)

            // Record stats
            go m.recordStats(context.Background(), stats.ToolUsage{
                UserID:      userID,
                AgentID:     agentID,
                ToolName:    t.Name(),
                DurationMs:  int(duration.Milliseconds()),
                Success:     err == nil,
                TimeWindow:  time.Now().Truncate(time.Hour), // Hourly buckets
            })

            return result, err
        },
    )
}

func (m *StatsMiddleware) recordStats(ctx context.Context, usage stats.ToolUsage) {
    // Upsert stats
    m.statsRepo.Increment(ctx, usage)
}
```

### 25.6 Batch Order Execution Layer

**Слой массового исполнения ордеров:**

```go
// internal/execution/batch_executor.go
package execution

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"
    "prometheus/internal/domain/order"
    "prometheus/internal/adapters/exchanges"
    "prometheus/pkg/logger"
)

type BatchExecutor struct {
    exchangeFactory exchanges.Factory
    orderRepo      order.Repository
    log            *logger.Logger

    // Batching configuration
    batchSize      int
    batchInterval time.Duration
    queue         chan *order.Order
}

func NewBatchExecutor(
    exchangeFactory exchanges.Factory,
    orderRepo order.Repository,
    batchSize int,
    batchInterval time.Duration,
) *BatchExecutor {
    ex := &BatchExecutor{
        exchangeFactory: exchangeFactory,
        orderRepo:      orderRepo,
        log:            logger.Get().With("component", "batch_executor"),
        batchSize:      batchSize,
        batchInterval:  batchInterval,
        queue:          make(chan *order.Order, 1000),
    }

    go ex.processBatches(context.Background())
    return ex
}

func (e *BatchExecutor) Submit(ctx context.Context, o *order.Order) error {
    select {
    case e.queue <- o:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (e *BatchExecutor) processBatches(ctx context.Context) {
    ticker := time.NewTicker(e.batchInterval)
    defer ticker.Stop()

    batch := make([]*order.Order, 0, e.batchSize)

    for {
        select {
        case <-ctx.Done():
            // Execute remaining batch
            if len(batch) > 0 {
                e.executeBatch(ctx, batch)
            }
            return

        case order := <-e.queue:
            batch = append(batch, order)
            if len(batch) >= e.batchSize {
                e.executeBatch(ctx, batch)
                batch = batch[:0]
            }

        case <-ticker.C:
            if len(batch) > 0 {
                e.executeBatch(ctx, batch)
                batch = batch[:0]
            }
        }
    }
}

func (e *BatchExecutor) executeBatch(ctx context.Context, orders []*order.Order) {
    // Group by exchange account
    byAccount := make(map[uuid.UUID][]*order.Order)
    for _, o := range orders {
        byAccount[o.ExchangeAccountID] = append(byAccount[o.ExchangeAccountID], o)
    }

    var wg sync.WaitGroup
    for accountID, accountOrders := range byAccount {
        wg.Add(1)
        go func(accID uuid.UUID, ords []*order.Order) {
            defer wg.Done()
            e.executeForAccount(ctx, accID, ords)
        }(accountID, accountOrders)
    }

    wg.Wait()
}

func (e *BatchExecutor) executeForAccount(ctx context.Context, accountID uuid.UUID, orders []*order.Order) {
    // Get exchange client
    account, _ := e.orderRepo.GetExchangeAccount(ctx, accountID)
    client, err := e.exchangeFactory.CreateClient(account)
    if err != nil {
        e.log.Errorf("Failed to get client for account %s: %v", accountID, err)
        return
    }

    // Execute orders in parallel (with rate limiting)
    semaphore := make(chan struct{}, 5) // Max 5 concurrent orders per account
    var wg sync.WaitGroup

    for _, o := range orders {
        wg.Add(1)
        go func(order *order.Order) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            // Execute order
            result, err := client.PlaceOrder(ctx, &exchanges.OrderRequest{
                Symbol: order.Symbol,
                Side:   string(order.Side),
                Type:   string(order.Type),
                Amount: order.Amount,
                Price:  order.Price,
            })

            if err != nil {
                e.log.Errorf("Failed to execute order %s: %v", order.ID, err)
                order.Status = order.OrderStatusRejected
            } else {
                order.ExchangeOrderID = result.ID
                order.Status = order.OrderStatusOpen
            }

            // Update in DB
            e.orderRepo.Update(ctx, order)
        }(o)
    }

    wg.Wait()
}
```

### 25.7 Agent Memory Architecture

**Структура памяти агентов:**

```
┌─────────────────────────────────────────────────────────┐
│              MEMORY ARCHITECTURE                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────────┐    ┌──────────────────┐          │
│  │  USER MEMORY     │    │ COLLECTIVE MEMORY│          │
│  │  (Isolated)      │    │  (Shared)        │          │
│  ├──────────────────┤    ├──────────────────┤          │
│  │ - User-specific  │    │ - Validated      │          │
│  │   observations   │    │   lessons        │          │
│  │ - Personal       │    │ - Market patterns│          │
│  │   trade history  │    │ - Strategy       │          │
│  │ - User context   │    │   insights       │          │
│  │                  │    │ - Cross-user     │          │
│  │ User A only      │    │   knowledge      │          │
│  └──────────────────┘    └──────────────────┘          │
│         │                        │                     │
│         └────────┬────────────────┘                     │
│                  │                                      │
│         ┌────────▼────────┐                            │
│         │  AGENT MEMORY   │                            │
│         │  (Per Agent)    │                            │
│         ├─────────────────┤                            │
│         │ - Agent-specific│                            │
│         │   patterns      │                            │
│         │ - Specialized    │                            │
│         │   knowledge     │                            │
│         └─────────────────┘                            │
└─────────────────────────────────────────────────────────┘
```

**База данных:**

```sql
-- migrations/postgres/005_memory_architecture.up.sql

-- User-specific memory (already exists, but clarify)
-- memories table: user_id + agent_id = isolated per user

-- NEW: Collective memory table
CREATE TABLE collective_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope
    agent_type VARCHAR(100) NOT NULL, -- "market_analyst", "risk_manager", etc.
    personality VARCHAR(50), -- "conservative", "aggressive", "balanced"

    -- Content
    type memory_type NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),

    -- Validation
    validation_score DECIMAL(3,2), -- How well this lesson performed
    validation_trades INTEGER, -- Number of trades that validated this
    validated_at TIMESTAMPTZ,

    -- Metadata
    symbol VARCHAR(50),
    timeframe VARCHAR(10),
    importance DECIMAL(3,2) DEFAULT 0.5,

    -- Source
    source_user_id UUID REFERENCES users(id), -- Who contributed this
    source_trade_id UUID, -- Which trade led to this lesson

    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_collective_memories_agent ON collective_memories(agent_type);
CREATE INDEX idx_collective_memories_personality ON collective_memories(personality);
CREATE INDEX idx_collective_memories_embedding ON collective_memories USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_collective_memories_validated ON collective_memories(validation_score DESC) WHERE validated_at IS NOT NULL;
```

**Логика работы:**

```go
// internal/domain/memory/service.go
type MemoryService struct {
    userMemoryRepo    memory.Repository      // User-specific
    collectiveRepo    memory.CollectiveRepo  // Shared
}

func (s *MemoryService) StoreLesson(ctx context.Context, userID uuid.UUID, lesson *memory.Memory) error {
    // 1. Store in user memory
    if err := s.userMemoryRepo.Store(ctx, lesson); err != nil {
        return err
    }

    // 2. If lesson is validated (e.g., from successful trade), promote to collective
    if lesson.ValidationScore >= 0.8 && lesson.ValidationTrades >= 3 {
        collective := &memory.CollectiveMemory{
            AgentType:       lesson.AgentID,
            Type:            lesson.Type,
            Content:         lesson.Content,
            Embedding:       lesson.Embedding,
            ValidationScore: lesson.ValidationScore,
            SourceUserID:    userID,
        }
        return s.collectiveRepo.Store(ctx, collective)
    }

    return nil
}

func (s *MemoryService) SearchMemory(ctx context.Context, userID uuid.UUID, query string, limit int) ([]*memory.Memory, error) {
    // 1. Search user-specific memory
    userMemories, _ := s.userMemoryRepo.SearchSimilar(ctx, userID, query, limit/2)

    // 2. Search collective memory (shared knowledge)
    collectiveMemories, _ := s.collectiveRepo.SearchSimilar(ctx, query, limit/2)

    // 3. Merge and rank
    return mergeMemories(userMemories, collectiveMemories), nil
}
```

**Ответ на вопрос о shared памяти:**

✅ **ДА, shared память нужна**, но с ограничениями:

1. **User Memory** — изолированная память каждого пользователя:

   - Личные наблюдения
   - История его сделок
   - Контекст его торговли

2. **Collective Memory** — общая память для всех агентов одного типа:

   - Валидированные уроки (только после успешных сделок)
   - Паттерны рынка, которые работают
   - Стратегические инсайты
   - Доступна всем пользователям, но только проверенная информация

3. **Преимущества:**

   - Новые пользователи получают проверенные знания сразу
   - Система учится быстрее (коллективный разум)
   - Изоляция пользовательских данных сохраняется

4. **Защита:**
   - В коллективную память попадают только валидированные уроки (score >= 0.8, >= 3 подтверждающих сделок)
   - Анонимизация источника (не храним детали чужих сделок)
   - Персонализация через personality (conservative/aggressive)

---

## 25.8 Chain-of-Thought & Autonomous Tool Selection

### 25.8.1 CoT Reasoning Architecture

Каждый агент использует **explicit Chain-of-Thought reasoning** с **autonomous tool selection**:

```
┌─────────────────────────────────────────────────────────┐
│                    AGENT EXECUTION                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Input: "Analyze BTC market"                             │
│                                                          │
│  ┌────────────────────────────────────────┐             │
│  │ Step 1: THINKING                       │             │
│  │ "I need current price and trend data"  │             │
│  └────────────────────────────────────────┘             │
│                  ↓                                       │
│  ┌────────────────────────────────────────┐             │
│  │ Step 2: TOOL SELECTION                 │             │
│  │ Agent decides: use get_price, get_ohlcv│             │
│  └────────────────────────────────────────┘             │
│                  ↓                                       │
│  ┌────────────────────────────────────────┐             │
│  │ Step 3: TOOL EXECUTION                 │             │
│  │ Calls tools autonomously               │             │
│  └────────────────────────────────────────┘             │
│                  ↓                                       │
│  ┌────────────────────────────────────────┐             │
│  │ Step 4: RESULT ANALYSIS                │             │
│  │ "Price is $45k, uptrend confirmed"     │             │
│  └────────────────────────────────────────┘             │
│                  ↓                                       │
│  ┌────────────────────────────────────────┐             │
│  │ Step 5: DECISION (may call more tools) │             │
│  │ Agent decides if needs more data       │             │
│  └────────────────────────────────────────┘             │
│                  ↓                                       │
│  ┌────────────────────────────────────────┐             │
│  │ Step N: FINAL SYNTHESIS                │             │
│  │ Returns structured decision            │             │
│  └────────────────────────────────────────┘             │
└─────────────────────────────────────────────────────────┘
```

### 25.8.2 Agent Tool Assignments

Каждый агент имеет **свой набор инструментов**, доступ к которым строго ограничен:

```go
// internal/agents/tool_assignments.go
package agents

import "prometheus/internal/tools"

// Tool categories per agent type (resolved dynamically from the global catalog)
var AgentToolCategories = map[AgentType][]string{
    // Personal trading workflow agents (per-user decision making)
    AgentStrategyPlanner: {"market_data", "account", "risk", "memory"},
    AgentRiskManager:     {"account", "risk", "memory"},
    AgentExecutor:        {"account", "execution", "memory"},
    AgentPositionManager: {"account", "execution", "memory"},
    AgentSelfEvaluator:   {"evaluation", "memory"},

    // Market research agent (global opportunity identification)
    AgentOpportunitySynthesizer: {"market_data", "smc", "memory"}, // Direct access to comprehensive analysis tools

    // Portfolio management agent (onboarding)
    AgentPortfolioArchitect: {"market_data", "momentum", "correlation", "account", "memory"},
}

var AgentToolMap = buildAgentToolMap()

func buildAgentToolMap() map[AgentType][]string {
    result := make(map[AgentType][]string, len(AgentToolCategories))
    defs := tools.Definitions()

    for agentType, categories := range AgentToolCategories {
        categorySet := make(map[string]struct{}, len(categories))
        for _, category := range categories {
            categorySet[category] = struct{}{}
        }

        for _, def := range defs {
            if _, ok := categorySet[def.Category]; ok {
                result[agentType] = append(result[agentType], def.Name)
            }
        }
    }

    return result
}
```

### 25.8.3 CoT Prompt Templates

**Обновленные промпты с явным CoT:**

```tmpl
{{/* pkg/templates/prompts/agents/market_analyst/system.tmpl */}}

You are an expert cryptocurrency market analyst with Chain-of-Thought reasoning capabilities.

## Your Mission
Analyze {{.Symbol}} market using available tools and provide a thorough technical analysis.

## Available Tools
You have access to the following tools (use them autonomously as needed):
{{range .Tools}}
- {{.Name}}: {{.Description}}
{{end}}

## Chain-of-Thought Process

You MUST think step-by-step and show your reasoning:

**STEP 1 - PROBLEM UNDERSTANDING:**
<thinking>
Explain what you need to analyze and why.
</thinking>

**STEP 2 - DATA COLLECTION PLAN:**
<thinking>
Decide which tools to use and in what order.
Explain your reasoning for each tool choice.
</thinking>

**STEP 3 - TOOL EXECUTION:**
Call tools autonomously. After each tool call:
<thinking>
Analyze the result. What does it tell you?
Do you need more data? Which tool next?
</thinking>

**STEP 4 - SYNTHESIS:**
<thinking>
Combine all gathered data.
Identify patterns and signals.
What is the market telling you?
</thinking>

**STEP 5 - DECISION:**
<thinking>
Based on ALL analysis, what is your recommendation?
What is your confidence level and why?
</thinking>

## Important Rules
- Show ALL your thinking in <thinking> tags
- Call tools autonomously - you decide which and when
- You can call tools multiple times if needed
- Explain WHY you're calling each tool
- Analyze EACH result before moving forward
- Maximum {{.MaxToolCalls}} tool calls per analysis

## Output Format
Return JSON with:
{
  "signal": "buy|sell|hold",
  "confidence": 0-100,
  "reasoning": "Your complete reasoning chain",
  "key_levels": {
    "support": [...],
    "resistance": [...]
  },
  "indicators": { /* all indicator values */ },
  "tool_calls_made": 12
}

## Risk Profile: {{.RiskLevel}}
{{if eq .RiskLevel "conservative"}}
- Require high confidence (>75%)
- Prefer trend-following
- Use more confirmation tools
{{else if eq .RiskLevel "aggressive"}}
- Accept moderate confidence (>60%)
- Consider counter-trend
- Faster decision making
{{end}}
```

### 25.8.4 CoT Execution Wrapper

**Wrapper для логирования CoT:**

```go
// internal/agents/cot_wrapper.go
package agents

import (
    "context"
    "encoding/json"
    "time"

    "google.golang.org/adk/agent"
    "prometheus/internal/domain/reasoning"
)

type CoTWrapper struct {
    agent          agent.Agent
    reasoningRepo  reasoning.Repository
    agentID        string
}

func NewCoTWrapper(
    agent agent.Agent,
    agentID string,
    reasoningRepo reasoning.Repository,
) *CoTWrapper {
    return &CoTWrapper{
        agent:         agent,
        agentID:       agentID,
        reasoningRepo: reasoningRepo,
    }
}

func (w *CoTWrapper) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
    sessionID := uuid.New().String()
    startTime := time.Now()

    steps := []reasoning.Step{}
    toolCalls := 0

    // Intercept agent execution
    result, err := w.agent.Run(ctx, agent.Request{
        Input: req.Input,

        // Callbacks for logging
        OnThinking: func(thought string) {
            steps = append(steps, reasoning.Step{
                Step:      len(steps) + 1,
                Action:    "thinking",
                Content:   thought,
                Timestamp: time.Now(),
            })
        },

        OnToolCall: func(toolName string, input, output interface{}) {
            toolCalls++
            inputJSON, _ := json.Marshal(input)
            outputJSON, _ := json.Marshal(output)

            steps = append(steps, reasoning.Step{
                Step:      len(steps) + 1,
                Action:    "tool_call",
                Tool:      toolName,
                Input:     inputJSON,
                Output:    outputJSON,
                Timestamp: time.Now(),
            })
        },

        OnDecision: func(decision interface{}) {
            decisionJSON, _ := json.Marshal(decision)
            steps = append(steps, reasoning.Step{
                Step:      len(steps) + 1,
                Action:    "decision",
                Content:   string(decisionJSON),
                Timestamp: time.Now(),
            })
        },
    })

    duration := time.Since(startTime)

    // Save reasoning log
    stepsJSON, _ := json.Marshal(steps)
    decisionJSON, _ := json.Marshal(result)

    logEntry := &reasoning.LogEntry{
        UserID:         req.UserID,
        AgentID:        w.agentID,
        SessionID:      sessionID,
        Symbol:         req.Symbol,
        ReasoningSteps: stepsJSON,
        Decision:       decisionJSON,
        TokensUsed:     result.TokensUsed,
        CostUSD:        result.CostUSD,
        DurationMs:     int(duration.Milliseconds()),
        CreatedAt:      time.Now(),
    }

    // Async logging (don't block execution)
    go w.reasoningRepo.Create(context.Background(), logEntry)

    return result, err
}
```

### 25.8.5 Agent-Specific Tool Sets

**Строгое разделение инструментов по агентам:**

```go
// internal/agents/factory.go
// Simplified agent creation - no separate analyst agents needed
// Analysis is now done through algorithmic tools called by OpportunitySynthesizer

func (f *Factory) CreateMarketResearchAgent() (agent.Agent, error) {
    // Single agent that calls comprehensive analysis tools directly
    return f.CreateAgent(AgentConfig{
        Type:  AgentOpportunitySynthesizer,
        Name:  "OpportunitySynthesizer",
        Tools: AgentToolMap[AgentOpportunitySynthesizer], // Includes technical, SMC, market analysis tools
        SystemPromptTemplate: "agents/opportunity_synthesizer",
        MaxToolCalls: 10,
    })
}
```

### 25.8.6 Prompt Examples with CoT

**Market Analyst CoT Prompt:**

```tmpl
{{/* pkg/templates/prompts/agents/market_analyst/system.tmpl */}}

You are a master technical analyst with 20 years of experience in cryptocurrency markets.

## Your Specialized Tools
{{range .Tools}}
- **{{.Name}}**: {{.Description}}
{{end}}

## Chain-of-Thought Analysis Protocol

### Phase 1: Initial Assessment
<thinking>
What do I need to know about {{.Symbol}} right now?
- Current price and recent movement
- Trend direction and strength
- Key technical levels

I will start with: get_price and get_ohlcv
</thinking>

[Call tools autonomously]

### Phase 2: Technical Analysis
<thinking>
Based on price data:
- What is the trend? Analyze the results
- Are we in a key zone? Check support/resistance
- What do momentum indicators show?

I need to check: RSI, MACD, and volume
</thinking>

[Call more tools based on initial findings]

### Phase 3: Multi-Timeframe Analysis
<thinking>
Current timeframe shows X, but I need confluence.
Let me check higher timeframes for confirmation.

I should analyze: 4h and 1d charts
</thinking>

[Call tools for different timeframes]

### Phase 4: Pattern Recognition
<thinking>
Looking at the chart structure:
- Any chart patterns forming?
- Swing points and market structure
- Fibonacci levels alignment

Tools needed: get_swing_points, fibonacci
</thinking>

[Final tool calls]

### Phase 5: Synthesis
<thinking>
Combining all analysis:
1. Trend: [your conclusion]
2. Momentum: [your conclusion]
3. Volume: [your conclusion]
4. Levels: [your conclusion]

Overall signal: [BUY/SELL/HOLD]
Confidence: [0-100] because [reasoning]
</thinking>

## Output Requirements
Return structured JSON:
{
  "signal": "buy|sell|hold",
  "confidence": 75,
  "trend": "bullish",
  "key_levels": {
    "support": [44500, 43800, 42000],
    "resistance": [46000, 47500, 50000]
  },
  "indicators": {
    "rsi_4h": 65,
    "macd_signal": "bullish",
    "volume_trend": "increasing"
  },
  "reasoning_summary": "Bullish trend confirmed by...",
  "tools_used": ["get_price", "get_ohlcv", "rsi", "macd", "volume_profile"],
  "tool_call_count": 7
}

## Constraints
- Maximum {{.MaxToolCalls}} tool calls
- Show ALL thinking steps
- Explain EVERY tool choice
- Must use at least 3 different indicators for confirmation
```

**Strategy Planner CoT Prompt:**

```tmpl
{{/* pkg/templates/prompts/agents/strategy_planner/system.tmpl */}}

You are a master trading strategist synthesizing market analysis into actionable trade plans.

## Your Tools
{{range .Tools}}
- **{{.Name}}**: {{.Description}}
{{end}}

## Input Data
You receive analysis from specialized agents:
- Market Analysis: {{.MarketAnalysis}}
- SMC Analysis: {{.SMCAnalysis}}
- Sentiment: {{.SentimentAnalysis}}
- On-Chain: {{.OnChainAnalysis}}
- Order Flow: {{.OrderFlowAnalysis}}

## Chain-of-Thought Strategy Formation

### Phase 1: Confluence Analysis
<thinking>
Let me check if all signals align:
- Technical: {{.MarketAnalysis.Signal}}
- SMC: {{.SMCAnalysis.Signal}}
- Sentiment: {{.SentimentAnalysis.Signal}}
- On-Chain: {{.OnChainAnalysis.Signal}}

Do they agree? If not, which carries more weight given current market regime?

I need to check: get_market_regime to understand current conditions
</thinking>

[Calls get_market_regime]

### Phase 2: Historical Context
<thinking>
How has this setup performed historically for this user?

I should check: get_trade_history and search_memory for similar setups
</thinking>

[Calls get_trade_history, search_memory]

### Phase 3: Strategy Selection
<thinking>
Based on:
- Market regime: [regime from tool]
- Historical performance: [from trade_history]
- User's risk profile: {{.RiskLevel}}
- Current confluence: [your analysis]

Which strategy fits best?
Options: trend_follow, reversal, breakout, range_trade

I'll check: get_strategy_stats to see which works best in current regime
</thinking>

[Calls get_strategy_stats]

### Phase 4: Entry/Exit Planning
<thinking>
Strategy selected: [strategy name]

Now plan execution:
- Entry zone: Where exactly?
- Stop loss: Based on ATR or structure?
- Take profit: Single or ladder?
- Position size: Based on risk per trade

Final plan: [detailed plan]
</thinking>

## Output
{
  "action": "buy|sell|hold",
  "strategy": "trend_follow",
  "confidence": 80,
  "entry": {
    "zone": [45000, 45200],
    "type": "limit"
  },
  "stop_loss": 43500,
  "take_profit": [47000, 48500, 50000],
  "reasoning": "Bullish confluence across all timeframes...",
  "similar_trades_winrate": 72,
  "tools_used": ["get_market_regime", "get_trade_history", "search_memory"]
}
```

### 25.8.7 Tool Call Limits & Safety

```go
// internal/agents/factory.go
type AgentConfig struct {
    Type        AgentType
    Name        string
    Tools       []string

    // CoT configuration
    MaxToolCalls      int           // Max tools per execution
    MaxThinkingTokens int           // Max tokens for reasoning
    TimeoutPerTool    time.Duration // Timeout for each tool
    TotalTimeout      time.Duration // Total execution timeout

    // Cost limits
    MaxCostPerRun     float64       // Max $ per execution
}

var DefaultAgentConfigs = map[AgentType]AgentConfig{
    AgentMarketAnalyst: {
        MaxToolCalls:      25,  // Может вызвать много индикаторов
        MaxThinkingTokens: 4000,
        TimeoutPerTool:    10 * time.Second,
        TotalTimeout:      2 * time.Minute,
        MaxCostPerRun:     0.10, // $0.10 max
    },

    AgentSentimentAnalyst: {
        MaxToolCalls:      10,  // Меньше инструментов
        MaxThinkingTokens: 2000,
        TimeoutPerTool:    15 * time.Second, // API calls slower
        TotalTimeout:      1 * time.Minute,
        MaxCostPerRun:     0.05,
    },

    AgentStrategyPlanner: {
        MaxToolCalls:      8,   // Только синтез + память
        MaxThinkingTokens: 6000, // Больше рассуждений
        TimeoutPerTool:    5 * time.Second,
        TotalTimeout:      1 * time.Minute,
        MaxCostPerRun:     0.15,
    },

    AgentRiskManager: {
        MaxToolCalls:      5,   // Быстрая валидация
        MaxThinkingTokens: 2000,
        TimeoutPerTool:    3 * time.Second,
        TotalTimeout:      30 * time.Second,
        MaxCostPerRun:     0.03,
    },
}
```

---

## 26. Security & Infrastructure Improvements

### 26.1 API Key Encryption

**Пакет для шифрования/дешифрования API ключей:**

```go
// pkg/crypto/encryption.go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

type Encryptor struct {
    key []byte // 32 bytes for AES-256
}

func NewEncryptor(key string) (*Encryptor, error) {
    keyBytes := []byte(key)
    if len(keyBytes) != 32 {
        return nil, errors.New("encryption key must be 32 bytes")
    }
    return &Encryptor{key: keyBytes}, nil
}

func (e *Encryptor) Encrypt(plaintext string) ([]byte, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }

    // Create GCM
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    // Create nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    // Encrypt
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return ciphertext, nil
}

func (e *Encryptor) Decrypt(ciphertext []byte) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    if len(ciphertext) < gcm.NonceSize() {
        return "", errors.New("ciphertext too short")
    }

    // Extract nonce
    nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

    // Decrypt
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

**Использование в репозитории:**

```go
// internal/repository/postgres/exchange_account.go
package postgres

import (
    "context"
    "prometheus/internal/domain/exchange_account"
    "prometheus/pkg/crypto"
)

type ExchangeAccountRepository struct {
    db        *sqlx.DB
    encryptor *crypto.Encryptor
}

func NewExchangeAccountRepository(db *sqlx.DB, encryptor *crypto.Encryptor) *ExchangeAccountRepository {
    return &ExchangeAccountRepository{
        db:        db,
        encryptor: encryptor,
    }
}

func (r *ExchangeAccountRepository) Create(ctx context.Context, account *exchange_account.ExchangeAccount) error {
    // Encrypt API keys before saving
    encryptedKey, err := r.encryptor.Encrypt(string(account.APIKeyEncrypted))
    if err != nil {
        return fmt.Errorf("failed to encrypt API key: %w", err)
    }

    encryptedSecret, err := r.encryptor.Encrypt(string(account.SecretEncrypted))
    if err != nil {
        return fmt.Errorf("failed to encrypt secret: %w", err)
    }

    account.APIKeyEncrypted = encryptedKey
    account.SecretEncrypted = encryptedSecret

    query := `
        INSERT INTO exchange_accounts (
            id, user_id, exchange, label, api_key_encrypted, secret_encrypted,
            passphrase, is_testnet, permissions, is_active, created_at, updated_at
        ) VALUES (
            :id, :user_id, :exchange, :label, :api_key_encrypted, :secret_encrypted,
            :passphrase, :is_testnet, :permissions, :is_active, :created_at, :updated_at
        )`

    _, err = r.db.NamedExecContext(ctx, query, account)
    return err
}

func (r *ExchangeAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*exchange_account.ExchangeAccount, error) {
    var account exchange_account.ExchangeAccount

    query := `SELECT * FROM exchange_accounts WHERE id = $1`
    if err := r.db.GetContext(ctx, &account, query, id); err != nil {
        return nil, err
    }

    // Decrypt API keys
    apiKey, err := r.encryptor.Decrypt(account.APIKeyEncrypted)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt API key: %w", err)
    }

    secret, err := r.encryptor.Decrypt(account.SecretEncrypted)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt secret: %w", err)
    }

    account.APIKeyEncrypted = []byte(apiKey)
    account.SecretEncrypted = []byte(secret)

    return &account, nil
}
```

### 26.2 Central Market Data API Keys

**Market data собирается по центральным API ключам из ENV, не по юзерским:**

```go
// internal/adapters/config/config.go
type Config struct {
    // ... existing fields

    // Central API keys for market data collection
    MarketData MarketDataConfig
}

type MarketDataConfig struct {
    // Exchange API keys for market data collection (not user-specific)
    Binance BinanceMarketDataConfig
    Bybit   BybitMarketDataConfig
    OKX     OKXMarketDataConfig
}

type BinanceMarketDataConfig struct {
    APIKey string `envconfig:"BINANCE_MARKET_DATA_API_KEY"`
    Secret string `envconfig:"BINANCE_MARKET_DATA_SECRET"`
}

type BybitMarketDataConfig struct {
    APIKey string `envconfig:"BYBIT_MARKET_DATA_API_KEY"`
    Secret string `envconfig:"BYBIT_MARKET_DATA_SECRET"`
}

type OKXMarketDataConfig struct {
    APIKey     string `envconfig:"OKX_MARKET_DATA_API_KEY"`
    Secret     string `envconfig:"OKX_MARKET_DATA_SECRET"`
    Passphrase string `envconfig:"OKX_MARKET_DATA_PASSPHRASE"`
}
```

```go
// internal/workers/market_data/ohlcv_collector.go
type OHLCVCollector struct {
    workers.BaseWorker
    centralExchangeFactory exchanges.CentralFactory // Uses central API keys
    repo                   market_data.Repository
    symbols                []string
    timeframes             []string
    log                    *logger.Logger
}

func NewOHLCVCollector(
    centralFactory exchanges.CentralFactory, // Central API keys from ENV
    repo market_data.Repository,
    cfg OHLCVCollectorConfig,
) *OHLCVCollector {
    return &OHLCVCollector{
        BaseWorker: workers.BaseWorker{
            name:     "ohlcv_collector",
            interval: cfg.Interval,
            enabled:  cfg.Enabled,
        },
        centralExchangeFactory: centralFactory,
        repo:                   repo,
        symbols:                cfg.Symbols,
        timeframes:             cfg.Timeframes,
        log:                    logger.Get().With("worker", "ohlcv_collector"),
    }
}

func (w *OHLCVCollector) Run(ctx context.Context) error {
    // Use central API keys, not user-specific
    for _, exchange := range []string{"binance", "bybit", "okx"} {
        client, err := w.centralExchangeFactory.GetClient(exchange)
        if err != nil {
            w.log.Warnf("Failed to get central %s client: %v", exchange, err)
            continue
        }

        for _, symbol := range w.symbols {
            for _, tf := range w.timeframes {
                candles, err := client.GetOHLCV(ctx, symbol, tf, 100)
                if err != nil {
                    w.log.Warnf("Failed to fetch %s %s %s: %v", exchange, symbol, tf, err)
                    continue
                }

                if err := w.repo.BatchInsert(ctx, candles); err != nil {
                    w.log.Errorf("Failed to insert candles: %v", err)
                }
            }
        }
    }

    return nil
}
```

### 26.3 Refactored Data Sources Architecture

**Новая структура datasources по типу данных с интерфейсами:**

```
internal/adapters/datasources/
├── news/
│   ├── interface.go           # NewsSource interface
│   ├── coindesk/
│   │   └── provider.go       # CoinDesk implementation
│   ├── cointelegraph/
│   │   └── provider.go       # CoinTelegraph implementation
│   └── theblock/
│       └── provider.go       # The Block implementation
├── sentiment/
│   ├── interface.go          # SentimentSource interface
│   ├── santiment/
│   │   └── provider.go
│   ├── lunarcrush/
│   │   └── provider.go
│   └── twitter/
│       └── provider.go
├── onchain/
│   ├── interface.go          # OnChainSource interface
│   ├── glassnode/
│   │   └── provider.go
│   └── santiment/
│       └── provider.go
├── derivatives/
│   ├── interface.go          # DerivativesSource interface
│   ├── deribit/
│   │   └── provider.go
│   ├── laevitas/
│   │   └── provider.go
│   └── greeks_live/
│       └── provider.go
├── liquidations/
│   ├── interface.go          # LiquidationSource interface
│   ├── coinglass/
│   │   └── provider.go
│   └── hyblock/
│       └── provider.go
└── macro/
    ├── interface.go          # MacroSource interface
    ├── investing/
    │   └── provider.go      # Economic calendar
    ├── fred/
    │   └── provider.go      # Federal Reserve data
    └── fedwatch/
        └── provider.go      # CME FedWatch
```

**Пример интерфейса:**

```go
// internal/adapters/datasources/news/interface.go
package news

import (
    "context"
    "time"
)

type Article struct {
    Title       string
    Content     string
    URL         string
    PublishedAt time.Time
    Symbols     []string // Extracted symbols (BTC, ETH, etc.)
}

type NewsSource interface {
    Name() string
    FetchLatest(ctx context.Context, limit int) ([]Article, error)
    FetchSince(ctx context.Context, since time.Time) ([]Article, error)
}
```

```go
// internal/adapters/datasources/news/coindesk/provider.go
package coindesk

import (
    "context"
    "prometheus/internal/adapters/datasources/news"
)

type Provider struct {
    apiKey string
}

func New(apiKey string) news.NewsSource {
    return &Provider{apiKey: apiKey}
}

func (p *Provider) Name() string {
    return "coindesk"
}

func (p *Provider) FetchLatest(ctx context.Context, limit int) ([]news.Article, error) {
    // Implementation
}
```

**Использование:**

```go
// internal/workers/sentiment/news_collector.go
type NewsCollector struct {
    workers.BaseWorker
    sources []news.NewsSource // Interface, легко заменить провайдера
    repo    sentiment.Repository
}

func NewNewsCollector(
    sources []news.NewsSource, // Можно передать любую реализацию
    repo sentiment.Repository,
) *NewsCollector {
    return &NewsCollector{
        sources: sources,
        repo:    repo,
    }
}
```

### 26.4 Rename Anthropic to Claude

**Обновление всех упоминаний:**

```go
// internal/adapters/ai/claude/provider.go (было anthropic)
package claude

import (
    "context"
    "fmt"

    "google.golang.org/adk/model"
    "google.golang.org/genai"

    "prometheus/internal/adapters/ai"
)

type Provider struct {
    apiKey string
    models map[string]*modelWrapper
}

func New(apiKey string) *Provider {
    return &Provider{
        apiKey: apiKey,
        models: make(map[string]*modelWrapper),
    }
}

func (p *Provider) Name() string {
    return "claude" // было "anthropic"
}
```

**Обновление конфигурации:**

```go
// internal/adapters/config/config.go
type AIConfig struct {
    ClaudeKey      string `envconfig:"CLAUDE_API_KEY"`      // было AnthropicKey
    OpenAIKey      string `envconfig:"OPENAI_API_KEY"`
    DeepSeekKey    string `envconfig:"DEEPSEEK_API_KEY"`
    GeminiKey      string `envconfig:"GEMINI_API_KEY"`
    DefaultProvider string `envconfig:"DEFAULT_AI_PROVIDER" default:"claude"` // было "anthropic"
}
```

```go
// internal/adapters/ai/factory.go
func SetupProviders(cfg config.AIConfig) ProviderRegistry {
    registry := NewRegistry()

    if cfg.ClaudeKey != "" { // было AnthropicKey
        registry.Register(claude.New(cfg.ClaudeKey)) // было anthropic.New
    }
    // ...
}
```

### 26.5 ClickHouse Batch Writer

**Пакет для батчинга записей в ClickHouse с graceful shutdown:**

```go
// pkg/clickhouse/batcher.go
package clickhouse

import (
    "context"
    "sync"
    "time"

    "github.com/ClickHouse/clickhouse-go/v2"
    "prometheus/pkg/logger"
)

type Batcher struct {
    conn        clickhouse.Conn
    table       string
    buffer      []interface{}
    bufferMu    sync.Mutex
    batchSize   int
    flushInterval time.Duration
    log         *logger.Logger

    // Graceful shutdown
    shutdown    chan struct{}
    wg          sync.WaitGroup
}

type BatcherConfig struct {
    Conn          clickhouse.Conn
    Table         string
    BatchSize     int           // Flush when buffer reaches this size
    FlushInterval time.Duration // Flush every N seconds
}

func NewBatcher(cfg BatcherConfig) *Batcher {
    b := &Batcher{
        conn:          cfg.Conn,
        table:         cfg.Table,
        buffer:        make([]interface{}, 0, cfg.BatchSize),
        batchSize:     cfg.BatchSize,
        flushInterval: cfg.FlushInterval,
        log:           logger.Get().With("component", "clickhouse_batcher", "table", cfg.Table),
        shutdown:      make(chan struct{}),
    }

    // Start background flusher
    b.wg.Add(1)
    go b.flushLoop()

    return b
}

func (b *Batcher) Write(ctx context.Context, data interface{}) error {
    b.bufferMu.Lock()
    defer b.bufferMu.Unlock()

    b.buffer = append(b.buffer, data)

    // Flush if buffer is full
    if len(b.buffer) >= b.batchSize {
        return b.flushLocked(ctx)
    }

    return nil
}

func (b *Batcher) flushLoop() {
    defer b.wg.Done()

    ticker := time.NewTicker(b.flushInterval)
    defer ticker.Stop()

    for {
        select {
        case <-b.shutdown:
            // Final flush on shutdown
            b.bufferMu.Lock()
            if len(b.buffer) > 0 {
                b.flushLocked(context.Background())
            }
            b.bufferMu.Unlock()
            return

        case <-ticker.C:
            b.bufferMu.Lock()
            if len(b.buffer) > 0 {
                b.flushLocked(context.Background())
            }
            b.bufferMu.Unlock()
        }
    }
}

func (b *Batcher) flushLocked(ctx context.Context) error {
    if len(b.buffer) == 0 {
        return nil
    }

    batch := b.buffer
    b.buffer = b.buffer[:0] // Clear buffer

    // Execute batch insert
    batchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    err := b.conn.AsyncInsert(batchCtx, b.table, batch, clickhouse.WithAsyncInsertData(batch))
    if err != nil {
        b.log.Errorf("Failed to flush batch to %s: %v", b.table, err)
        // Optionally restore buffer on error
        b.buffer = append(b.buffer, batch...)
        return err
    }

    b.log.Debugf("Flushed %d records to %s", len(batch), b.table)
    return nil
}

// Flush manually
func (b *Batcher) Flush(ctx context.Context) error {
    b.bufferMu.Lock()
    defer b.bufferMu.Unlock()
    return b.flushLocked(ctx)
}

// Graceful shutdown - flush all pending data
func (b *Batcher) Shutdown(ctx context.Context) error {
    close(b.shutdown)

    // Wait for flush loop to finish
    done := make(chan struct{})
    go func() {
        b.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Использование:**

```go
// internal/repository/clickhouse/market_data.go
type MarketDataRepository struct {
    ohlcvBatcher *clickhouse.Batcher
    tickerBatcher *clickhouse.Batcher
}

func NewMarketDataRepository(conn clickhouse.Conn) *MarketDataRepository {
    return &MarketDataRepository{
        ohlcvBatcher: clickhouse.NewBatcher(clickhouse.BatcherConfig{
            Conn:          conn,
            Table:         "ohlcv",
            BatchSize:     1000,
            FlushInterval: 5 * time.Second,
        }),
        tickerBatcher: clickhouse.NewBatcher(clickhouse.BatcherConfig{
            Conn:          conn,
            Table:         "tickers",
            BatchSize:     500,
            FlushInterval: 2 * time.Second,
        }),
    }
}

func (r *MarketDataRepository) BatchInsertOHLCV(ctx context.Context, candles []market_data.OHLCV) error {
    for _, candle := range candles {
        if err := r.ohlcvBatcher.Write(ctx, candle); err != nil {
            return err
        }
    }
    return nil
}
```

### 26.6 Executor Agent Architecture

**Два подхода к архитектуре Executor агента:**

#### Вариант 1: Один Executor на всех пользователей (рекомендуется)

```go
// internal/agents/executor/agent.go
package executor

import (
    "context"
    "prometheus/internal/domain/order"
    "prometheus/internal/adapters/exchanges"
)

type ExecutorAgent struct {
    exchangeFactory exchanges.Factory
    orderRepo      order.Repository
    batchExecutor  *execution.BatchExecutor
}

// Executor принимает user context и выполняет от его имени
func (a *ExecutorAgent) ExecuteTrade(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
    // req содержит:
    // - UserID
    // - ExchangeAccountID (для получения API ключей)
    // - Order details (symbol, side, amount, price, etc.)

    // 1. Получить exchange account с расшифрованными ключами
    account, err := a.orderRepo.GetExchangeAccount(ctx, req.ExchangeAccountID)
    if err != nil {
        return nil, err
    }

    // 2. Создать exchange client с юзерскими ключами
    client, err := a.exchangeFactory.CreateClient(account)
    if err != nil {
        return nil, err
    }

    // 3. Выполнить ордер
    exchangeOrder, err := client.PlaceOrder(ctx, &exchanges.OrderRequest{
        Symbol: req.Symbol,
        Side:   string(req.Side),
        Type:   string(req.Type),
        Amount: req.Amount,
        Price:  req.Price,
    })

    // 4. Сохранить в БД
    order := &order.Order{
        UserID:          req.UserID,
        ExchangeAccountID: req.ExchangeAccountID,
        ExchangeOrderID: exchangeOrder.ID,
        Symbol:          req.Symbol,
        Side:            req.Side,
        Type:            req.Type,
        Amount:          req.Amount,
        Price:           req.Price,
        Status:          order.OrderStatusOpen,
    }

    return &ExecutionResult{Order: order}, a.orderRepo.Create(ctx, order)
}

type ExecutionRequest struct {
    UserID            uuid.UUID
    ExchangeAccountID uuid.UUID
    Symbol            string
    Side              order.OrderSide
    Type              order.OrderType
    Amount            decimal.Decimal
    Price             decimal.Decimal
    StopLoss          decimal.Decimal
    TakeProfit        decimal.Decimal
}
```

**Преимущества:**

- Один агент обслуживает всех пользователей
- Проще управление и мониторинг
- Меньше overhead на создание агентов

#### Вариант 2: Executor per user (альтернатива)

```go
// internal/agents/executor/user_executor.go
type UserExecutor struct {
    userID          uuid.UUID
    exchangeFactory exchanges.Factory
    orderRepo       order.Repository
    commands        chan *ExecutionCommand
}

type ExecutorPool struct {
    executors map[uuid.UUID]*UserExecutor
    mu        sync.RWMutex
}

func (p *ExecutorPool) GetOrCreate(userID uuid.UUID) *UserExecutor {
    p.mu.RLock()
    if executor, ok := p.executors[userID]; ok {
        p.mu.RUnlock()
        return executor
    }
    p.mu.RUnlock()

    p.mu.Lock()
    defer p.mu.Unlock()

    // Double check
    if executor, ok := p.executors[userID]; ok {
        return executor
    }

    executor := NewUserExecutor(userID, p.exchangeFactory, p.orderRepo)
    p.executors[userID] = executor
    go executor.Run(context.Background())

    return executor
}

func (p *ExecutorPool) Execute(userID uuid.UUID, cmd *ExecutionCommand) error {
    executor := p.GetOrCreate(userID)
    return executor.Execute(cmd)
}
```

**Рекомендация:** Использовать Вариант 1 (один Executor), так как:

- Проще архитектура
- Меньше ресурсов
- Executor stateless - просто выполняет команды с контекстом пользователя

### 26.7 Error Tracking

**Интерфейс для error tracking:**

```go
// pkg/errors/tracker.go
package errors

import (
    "context"
    "errors"
)

type Tracker interface {
    CaptureError(ctx context.Context, err error, tags map[string]string) error
    CaptureMessage(ctx context.Context, message string, level Level, tags map[string]string) error
    SetUser(ctx context.Context, userID string, email string, username string)
    AddBreadcrumb(ctx context.Context, message string, category string, level Level, data map[string]interface{})
    Flush(ctx context.Context) error
}

type Level string

const (
    LevelDebug   Level = "debug"
    LevelInfo    Level = "info"
    LevelWarning Level = "warning"
    LevelError   Level = "error"
    LevelFatal   Level = "fatal"
)
```

**Реализация для Sentry:**

```go
// internal/adapters/errors/sentry/tracker.go
package sentry

import (
    "context"
    "github.com/getsentry/sentry-go"
    "prometheus/pkg/errors"
)

type Tracker struct {
    hub *sentry.Hub
}

func New(dsn string, environment string) (*Tracker, error) {
    err := sentry.Init(sentry.ClientOptions{
        Dsn:         dsn,
        Environment: environment,
    })
    if err != nil {
        return nil, err
    }

    return &Tracker{
        hub: sentry.CurrentHub(),
    }, nil
}

func (t *Tracker) CaptureError(ctx context.Context, err error, tags map[string]string) error {
    hub := t.hub.Clone()

    // Add tags
    for k, v := range tags {
        hub.SetTag(k, v)
    }

    // Add context
    if userID := ctx.Value("user_id"); userID != nil {
        hub.SetUser(sentry.User{ID: userID.(string)})
    }

    eventID := hub.CaptureException(err)
    return eventID
}

func (t *Tracker) CaptureMessage(ctx context.Context, message string, level errors.Level, tags map[string]string) error {
    hub := t.hub.Clone()

    for k, v := range tags {
        hub.SetTag(k, v)
    }

    sentryLevel := sentry.LevelInfo
    switch level {
    case errors.LevelDebug:
        sentryLevel = sentry.LevelDebug
    case errors.LevelWarning:
        sentryLevel = sentry.LevelWarning
    case errors.LevelError:
        sentryLevel = sentry.LevelError
    case errors.LevelFatal:
        sentryLevel = sentry.LevelFatal
    }

    eventID := hub.CaptureMessage(message, &sentry.EventHint{}, sentryLevel)
    return eventID
}

func (t *Tracker) SetUser(ctx context.Context, userID string, email string, username string) {
    t.hub.ConfigureScope(func(scope *sentry.Scope) {
        scope.SetUser(sentry.User{
            ID:       userID,
            Email:    email,
            Username: username,
        })
    })
}

func (t *Tracker) AddBreadcrumb(ctx context.Context, message string, category string, level errors.Level, data map[string]interface{}) {
    sentryLevel := sentry.LevelInfo
    switch level {
    case errors.LevelError:
        sentryLevel = sentry.LevelError
    case errors.LevelWarning:
        sentryLevel = sentry.LevelWarning
    }

    t.hub.AddBreadcrumb(&sentry.Breadcrumb{
        Message:  message,
        Category: category,
        Level:    sentryLevel,
        Data:     data,
    }, &sentry.BreadcrumbHint{})
}

func (t *Tracker) Flush(ctx context.Context) error {
    return sentry.Flush(2 * time.Second)
}
```

**Интеграция в logger:**

```go
// pkg/logger/logger.go
type Logger struct {
    *zap.SugaredLogger
    errorTracker errors.Tracker // Optional
}

func (l *Logger) Error(args ...interface{}) {
    l.SugaredLogger.Error(args...)

    // Auto-track errors if tracker is set
    if l.errorTracker != nil {
        err := errors.New(fmt.Sprint(args...))
        l.errorTracker.CaptureError(context.Background(), err, map[string]string{
            "component": "logger",
        })
    }
}

func (l *Logger) Errorf(tpl string, args ...interface{}) {
    l.SugaredLogger.Errorf(tpl, args...)

    if l.errorTracker != nil {
        err := fmt.Errorf(tpl, args...)
        l.errorTracker.CaptureError(context.Background(), err, map[string]string{
            "component": "logger",
        })
    }
}
```

**Явное использование (рекомендуется для критичных ошибок):**

```go
// internal/agents/executor/agent.go
func (a *ExecutorAgent) ExecuteTrade(ctx context.Context, req *ExecutionRequest) error {
    account, err := a.orderRepo.GetExchangeAccount(ctx, req.ExchangeAccountID)
    if err != nil {
        // Явно отправляем в error tracker
        a.errorTracker.CaptureError(ctx, err, map[string]string{
            "component":    "executor",
            "user_id":      req.UserID.String(),
            "exchange_id":  req.ExchangeAccountID.String(),
            "operation":    "get_exchange_account",
        })
        return err
    }

    // ...
}
```

**Конфигурация:**

```go
// internal/adapters/config/config.go
type Config struct {
    // ...
    ErrorTracking ErrorTrackingConfig
}

type ErrorTrackingConfig struct {
    Enabled   bool   `envconfig:"ERROR_TRACKING_ENABLED" default:"true"`
    Provider  string `envconfig:"ERROR_TRACKING_PROVIDER" default:"sentry"` // sentry, datadog, etc.
    SentryDSN string `envconfig:"SENTRY_DSN"`
    Environment string `envconfig:"SENTRY_ENVIRONMENT" default:"production"`
}
```

**Рекомендация:**

- Использовать **явное указание** для критичных ошибок (execution, risk events)
- **Автоматический tracking** в logger для остальных ошибок
- Это дает контроль над тем, что отправляется в bug tracker

---

**Document Version:** 1.2  
**Last Updated:** November 2025  
**Author:** Claude (Anthropic)
