<!-- @format -->

# Agentic Trading System - Development Plan

**Version:** 1.0  
**Date:** November 2025  
**Based on:** specs.md v1.2

---

## Phase 1: Foundation & Core Infrastructure ✅ (Weeks 1-2)

### 1.1 Project Setup ✅

- [x] Initialize Go project structure (cmd/, internal/, pkg/)
- [x] Setup go.mod with dependencies (ADK, sqlx, pgvector, clickhouse-go, redis, kafka, zap)
- [x] Configure environment management (envconfig + godotenv)
- [x] Setup .env.example with all required variables (created docs/ENV_SETUP.md)
- [x] Initialize structured logging (Zap) with error tracking interface

### 1.2 Database Setup ✅

- [x] PostgreSQL 16 with pgvector extension (migrations ready)
- [x] ClickHouse for time-series market data (schemas ready)
- [x] Redis for caching and distributed locks (client ready)
- [x] Apache Kafka for event streaming
- [x] Migration framework setup (postgres + clickhouse)

### 1.3 Core Adapters ✅

- [x] PostgreSQL client with sqlx connection pool
- [x] ClickHouse client with batch writer
- [x] Redis client with lock/cache utilities
- [x] Kafka producer and consumer setup
- [x] Error tracking adapter (Sentry interface + implementation)

### 1.4 Configuration System ✅

- [x] Load all ENV variables with validation
- [x] Separate configs: App, Postgres, ClickHouse, Redis, Kafka, Telegram, AI, Crypto, MarketData, ErrorTracking
- [x] Encryption setup (AES-256-GCM for API keys)

---

## Phase 2: Domain Layer & Repositories ✅ (Weeks 2-3)

### 2.1 Domain Entities (with UUIDs, PostgreSQL ENUMs) ✅

- [x] User entity with settings JSONB
- [x] ExchangeAccount entity with encrypted API keys
- [x] TradingPair entity with budget and risk parameters
- [x] Order entity with all order types (market, limit, stop_market, stop_limit)
- [x] Position entity with PnL tracking
- [x] Memory entity with pgvector embeddings (user + collective)
- [x] JournalEntry entity for trade reflection
- [x] CircuitBreakerState entity for risk monitoring
- [x] MarketData entities (OHLCV, Ticker, OrderBook, Trade)
- [x] MarketRegime entity for regime detection
- [x] MacroEvent entity for economic calendar
- [x] DerivativesData entities (OptionsSnapshot, OptionsFlow)
- [x] LiquidationData entities (Liquidation, LiquidationHeatmap)
- [x] Sentiment entities (News, SocialSentiment, FearGreedIndex)
- [x] Reasoning entities (LogEntry, Step)
- [x] Stats entities (ToolUsage)

### 2.2 PostgreSQL Migrations ✅

- [x] 001_init: users, exchange_accounts, trading_pairs, orders, positions
- [x] 002_add_enums: All PostgreSQL ENUM types
- [x] 003_agent_reasoning_log: Chain-of-Thought logging
- [x] 004_tool_usage_stats: Tool usage statistics
- [x] Memories tables (user + collective) with pgvector
- [x] Journal entries table
- [x] Circuit breaker state table
- [x] Risk events table

### 2.3 ClickHouse Schemas ✅

- [x] OHLCV table with ReplacingMergeTree
- [x] Tickers table with real-time updates
- [x] OrderBook snapshots table
- [x] News table with sentiment scores
- [x] Social sentiment table
- [x] Liquidations table
- [x] Materialized views for aggregations

### 2.4 Repository Implementations (sqlx pattern) ✅

- [x] UserRepository with CRUD operations
- [x] ExchangeAccountRepository (encryption handled in service layer)
- [x] TradingPairRepository with active pairs queries
- [x] OrderRepository with batch operations
- [x] PositionRepository with PnL calculations
- [x] MemoryRepository with pgvector semantic search (user + collective)
- [x] JournalRepository with strategy stats aggregation
- [x] MarketDataRepository (ClickHouse) with batch writer
- [x] SentimentRepository (ClickHouse) with news and social sentiment
- [x] RiskRepository with circuit breaker state and events
- [x] ReasoningRepository for Chain-of-Thought logs
- [x] StatsRepository for tool usage tracking
- [x] PostgreSQL client with sqlx connection pool
- [x] ClickHouse client with async inserts
- [x] Redis client with lock/cache utilities

---

## Phase 3: Exchange Integration (Week 3)

### 3.1 Exchange Interface

- [x] Unified Exchange interface (GetTicker, GetOrderBook, GetOHLCV, GetTrades)
- [x] SpotExchange and FuturesExchange interfaces
- [x] OrderRequest and OrderResponse structs

### 3.2 Exchange Adapters

- [x] Binance adapter (Spot + Futures)
- [x] Bybit adapter (Spot + Futures)
- [x] OKX adapter (Spot + Futures)
- [x] Exchange factory with client pooling
- [x] Central factory for market data collection (uses ENV API keys, not user keys)

### 3.3 Exchange Operations

- [x] Market data fetching (OHLCV, tickers, orderbook, trades)
- [x] Order placement (market, limit, stop orders)
- [x] Bracket orders (Entry + SL + TP)
- [x] Ladder orders (multiple TP levels)
- [x] Position management (get positions, close, modify SL/TP)
- [x] Leverage and margin mode settings

---

## Phase 4: AI Provider Abstraction (Week 4)

### 4.1 AI Provider Interface

- [x] Provider interface with GetModel, ListModels, SupportsStreaming, SupportsTools
- [x] ModelInfo struct with costs and capabilities
- [x] ProviderRegistry for multi-provider support

### 4.2 AI Provider Implementations

- [x] Claude provider (primary, was "Anthropic")
- [x] OpenAI provider
- [x] DeepSeek provider
- [x] Gemini provider
- [x] Provider factory with registry

### 4.3 Model Configuration

- [x] Model selection per agent type
- [x] Cost tracking per model
- [x] Token usage monitoring
- [x] Timeout configuration per provider

---

## Phase 5: Template System (Week 4)

### 5.1 Template Registry

- [ ] Template loader from filesystem
- [ ] Template caching and hot-reload
- [ ] Template ID mapping (path-based)
- [ ] Template.Render() with data binding

### 5.2 Agent Prompt Templates (with Chain-of-Thought)

- [ ] Market Analyst system prompt with CoT protocol
- [ ] SMC Analyst system prompt
- [ ] Sentiment Analyst system prompt
- [ ] On-Chain Analyst system prompt
- [ ] Correlation Analyst system prompt
- [ ] Macro Analyst system prompt
- [ ] Order Flow Analyst system prompt
- [ ] Derivatives Analyst system prompt
- [ ] Strategy Planner system prompt with synthesis
- [ ] Risk Manager system prompt with validation
- [ ] Executor system prompt
- [ ] Position Manager system prompt
- [ ] Self Evaluator system prompt

### 5.3 Notification Templates

- [ ] Trade opened notification
- [ ] Trade closed notification
- [ ] Stop loss hit notification
- [ ] Take profit hit notification
- [ ] Circuit breaker triggered notification
- [ ] Daily performance report

---

## Phase 6: Tools Registry (Weeks 5-6)

### 6.1 Tool Registry Infrastructure

- [x] Tool interface wrapper for ADK
- [x] Tool registry with name-based lookup
- [x] Tool middleware for stats tracking
- [x] Tool timeout and retry logic

### 6.2 Market Data Tools (8 tools)

- [x] get_price: Current price with bid/ask
- [x] get_ohlcv: Historical candles
- [x] get_orderbook: Order book depth
- [x] get_trades: Recent trades (tape)
- [x] get_funding_rate: Futures funding
- [x] get_open_interest: OI data
- [x] get_long_short_ratio: Long/short ratio
- [x] get_liquidations: Recent liquidations

### 6.3 Order Flow Tools (8 tools)

- [x] get_trade_imbalance: Buy vs sell pressure
- [x] get_cvd: Cumulative Volume Delta
- [x] get_whale_trades: Large transactions
- [x] get_tick_speed: Trade velocity
- [x] get_orderbook_imbalance: Bid/ask delta
- [x] detect_iceberg: Hidden liquidity
- [x] detect_spoofing: Fake orders
- [x] get_absorption_zones: Order absorption

### 6.4 Technical Indicators - Momentum (4 tools)

- [x] rsi: Relative Strength Index
- [x] stochastic: Stochastic oscillator
- [x] cci: Commodity Channel Index
- [x] roc: Rate of Change

### 6.5 Technical Indicators - Volatility (3 tools)

- [x] atr: Average True Range
- [x] bollinger: Bollinger Bands
- [x] keltner: Keltner Channels

### 6.6 Technical Indicators - Trend (7 tools)

- [x] sma: Simple Moving Average
- [x] ema: Exponential Moving Average
- [x] ema_ribbon: Multiple EMAs
- [x] macd: MACD with signal
- [x] supertrend: Supertrend indicator
- [x] ichimoku: Ichimoku Cloud
- [x] pivot_points: Pivot points

### 6.7 Technical Indicators - Volume (4 tools)

- [x] vwap: Volume Weighted Average Price
- [x] obv: On-Balance Volume
- [x] volume_profile: Volume profile
- [x] delta_volume: Buy/sell volume delta

### 6.8 Smart Money Concepts Tools (7 tools)

- [x] detect_fvg: Fair Value Gaps
- [x] detect_order_blocks: Order Blocks
- [x] detect_liquidity_zones: Liquidity pools
- [x] detect_stop_hunt: Stop run detection
- [x] get_market_structure: BOS, CHoCH
- [x] detect_imbalances: Price imbalances
- [x] get_swing_points: Swing highs/lows

### 6.9 Sentiment Tools (5 tools)

- [x] get_fear_greed: Fear & Greed Index
- [x] get_social_sentiment: Twitter/Reddit sentiment
- [x] get_news: Latest crypto news
- [x] get_trending: Trending coins/topics
- [x] get_funding_sentiment: Sentiment from funding rates

### 6.10 On-Chain Tools (9 tools)

- [x] get_whale_movements: Large wallet transfers
- [x] get_exchange_flows: Exchange inflow/outflow
- [x] get_miner_reserves: Miner holdings
- [x] get_active_addresses: Active address count
- [x] get_nvt_ratio: Network Value to Transactions
- [x] get_sopr: Spent Output Profit Ratio
- [x] get_mvrv: Market Value to Realized Value
- [x] get_realized_pnl: Net Realized Profit/Loss
- [x] get_stablecoin_flows: USDT/USDC flows

### 6.11 Macro Tools (6 tools)

- [x] get_economic_calendar: CPI, FOMC, NFP dates
- [x] get_fed_rate: Current Fed rate
- [x] get_fed_watch: Rate probabilities
- [x] get_cpi: Inflation data
- [x] get_pmi: PMI data
- [x] get_macro_impact: Historical event impact

### 6.12 Derivatives Tools (6 tools)

- [x] get_options_oi: Options open interest
- [x] get_max_pain: Max pain price
- [x] get_put_call_ratio: Put/Call ratio
- [x] get_gamma_exposure: Dealer gamma
- [x] get_options_flow: Large options trades
- [x] get_iv_surface: Implied volatility

### 6.13 Correlation Tools (7 tools)

- [x] btc_dominance: BTC market dominance
- [x] usdt_dominance: Stablecoin dominance
- [x] altcoin_correlation: Alt correlation to BTC
- [x] stock_correlation: BTC vs SPX/Nasdaq
- [x] dxy_correlation: BTC vs Dollar Index
- [x] gold_correlation: BTC vs Gold
- [x] get_session_volume: Asia/EU/US sessions

### 6.14 Trading Execution Tools (14 tools)

- [x] get_balance: Account balance
- [x] get_positions: Open positions
- [x] place_order: Market/limit/stop order
- [x] place_bracket_order: Entry + SL + TP
- [x] place_ladder_order: Multiple TP levels
- [x] place_iceberg_order: Hidden size order
- [x] cancel_order: Cancel specific order
- [x] cancel_all_orders: Cancel all orders
- [x] close_position: Close specific position
- [x] close_all_positions: Close all positions
- [x] set_leverage: Set leverage (futures)
- [x] set_margin_mode: Cross/isolated margin
- [x] move_sl_to_breakeven: Move SL to entry
- [x] set_trailing_stop: Activate trailing stop
- [x] add_to_position: DCA / averaging

### 6.15 Risk Management Tools (7 tools)

- [x] check_circuit_breaker: Trading allowed?
- [x] get_daily_pnl: Today's PnL
- [x] get_max_drawdown: Current drawdown
- [x] get_exposure: Total exposure
- [x] calculate_position_size: Kelly/fixed fraction
- [x] validate_trade: Pre-trade checks
- [x] emergency_close_all: KILL SWITCH

### 6.16 Memory Tools (5 tools)

- [x] store_memory: Save observation/decision
- [x] search_memory: Semantic search past memories
- [x] get_trade_history: Past trades with outcomes
- [x] get_market_regime: Current detected regime
- [x] store_market_regime: Save regime detection

### 6.17 Evaluation Tools (6 tools)

- [x] get_strategy_stats: Win rate per strategy
- [x] log_trade_decision: Create journal entry
- [x] get_trade_journal: Retrieve past decisions
- [x] evaluate_last_trades: Performance review
- [x] get_best_strategies: Top performers
- [x] get_worst_strategies: Underperformers

---

## Phase 7: Agent System (Weeks 6-7)

### 7.1 Agent Registry & Factory

- [x] AgentType enum with all agent types
- [x] AgentConfig with tools, prompts, limits (MaxToolCalls, MaxThinkingTokens, TimeoutPerTool, MaxCostPerRun)
- [x] Agent registry for registration and lookup
- [x] Agent factory for creating configured agents
- [x] CoT wrapper for reasoning logging

### 7.2 Analysis Agents (8 agents)

- [x] MarketAnalyst: Technical analysis (25 tool calls max)
- [x] SMCAnalyst: Smart Money Concepts (15 tool calls max)
- [x] SentimentAnalyst: News & social sentiment (10 tool calls max)
- [x] OnChainAnalyst: Blockchain metrics (10 tool calls max)
- [x] CorrelationAnalyst: Cross-market analysis (8 tool calls max)
- [x] MacroAnalyst: Economic events (8 tool calls max)
- [x] OrderFlowAnalyst: Tape reading, CVD (12 tool calls max)
- [x] DerivativesAnalyst: Options flow (10 tool calls max)

### 7.3 Synthesis & Execution Agents (5 agents)

- [x] StrategyPlanner: Synthesize analysis into trade plan (8 tool calls max, 6000 thinking tokens)
- [x] RiskManager: Validate and size positions (5 tool calls max, fast execution)
- [x] Executor: Place orders (3 tool calls max, user-specific API keys)
- [x] PositionManager: Monitor and adjust positions (5 tool calls max)
- [x] SelfEvaluator: Analyze performance (10 tool calls max)

### 7.4 Agent Orchestration

- [x] ParallelAgent wrapper for concurrent analysis
- [x] SequentialAgent wrapper for pipeline execution
- [x] CreateTradingPipeline: Full workflow (Analysis → Strategy → Risk → Executor)

### 7.5 Agent Tool Assignments

- [x] Strict tool sets per agent type (AgentToolMap)
- [x] Tool access control enforcement
- [x] Tool call limits per agent type
- [x] Cost limits per agent execution

### 7.6 Chain-of-Thought Logging

- [x] Reasoning log table (agent_reasoning_logs)
- [x] Step-by-step reasoning capture
- [x] Tool call tracking with input/output
- [x] Decision logging with confidence scores
- [x] Token usage and cost tracking per execution

---

## Phase 8: Workers & Schedulers (Weeks 7-8)

### 8.1 Worker Infrastructure

- [ ] Worker interface with Run(), Interval(), Enabled()
- [ ] BaseWorker with common fields
- [ ] Scheduler with graceful start/stop
- [ ] Worker registry

### 8.2 Market Data Workers (5 workers)

- [ ] ohlcv_collector: Collect candles (1m interval, central API keys)
- [ ] ticker_collector: Real-time tickers (5s interval)
- [ ] orderbook_collector: Order book snapshots (10s interval)
- [ ] trades_collector: Tape feed (1s interval)
- [ ] funding_collector: Funding rates (1m interval)

### 8.3 Order Flow Workers (3 workers)

- [ ] liquidation_collector: Real-time liquidations (5s interval)
- [ ] oi_collector: Open interest (1m interval)
- [ ] whale_alert_collector: Large trades (10s interval)

### 8.4 Sentiment Workers (4 workers)

- [ ] news_collector: News aggregation (5m interval)
- [ ] twitter_collector: X.com sentiment (2m interval)
- [ ] reddit_collector: Reddit sentiment (5m interval)
- [ ] feargreed_collector: Fear & Greed index (15m interval)

### 8.5 On-Chain Workers (3 workers)

- [ ] exchange_flow_collector: Exchange flows (5m interval)
- [ ] whale_collector: Whale movements (5m interval)
- [ ] metrics_collector: MVRV, SOPR, NVT (15m interval)

### 8.6 Macro Workers (3 workers)

- [ ] calendar_collector: Economic calendar (1h interval)
- [ ] fed_collector: Fed data (1h interval)
- [ ] correlation_collector: SPX, DXY, Gold (5m interval)

### 8.7 Derivatives Workers (2 workers)

- [ ] options_collector: Options data (5m interval)
- [ ] gamma_collector: Gamma exposure (15m interval)

### 8.8 Trading Workers (4 workers)

- [ ] position_monitor: Position health check (30s interval)
- [ ] order_sync: Order status sync (10s interval)
- [ ] pnl_calculator: Calculate PnL (1m interval)
- [ ] risk_monitor: Circuit breaker checks (10s interval)

### 8.9 Analysis Workers (4 workers)

- [ ] market_scanner: Run agent analysis for ALL active users (5m interval, parallel processing)
- [ ] opportunity_finder: Find trading setups (1m interval)
- [ ] regime_detector: Update market regime (5m interval)
- [ ] smc_scanner: FVG, OB detection (1m interval)

### 8.10 Evaluation Workers (3 workers)

- [ ] daily_report: Generate daily report (24h interval)
- [ ] strategy_evaluator: Evaluate strategies (6h interval)
- [ ] journal_compiler: Compile trade journal (1h interval)

---

## Phase 9: Risk Engine (Week 8)

### 9.1 Risk Engine Core

- [ ] Risk engine with user state management
- [ ] CanTrade() check (called before ANY execution)
- [ ] RecordTrade() for state updates
- [ ] Circuit breaker logic (max drawdown, consecutive losses)
- [ ] Kill switch for emergency shutdown
- [ ] Daily reset at 00:00 UTC

### 9.2 Risk Limits & Triggers

- [ ] Max daily drawdown threshold
- [ ] Max consecutive losses threshold
- [ ] Max portfolio exposure
- [ ] Max position size validation
- [ ] Rate limiting per user

### 9.3 Risk Event Publishing

- [ ] Risk event types (drawdown_warning, circuit_breaker, kill_switch, etc.)
- [ ] Kafka event publishing for notifications
- [ ] Risk event logging in database

---

## Phase 10: Memory System (Week 9)

### 10.1 User Memory (Isolated per user)

- [ ] Personal observations storage
- [ ] User-specific trade history
- [ ] Personal context and patterns
- [ ] Semantic search with pgvector

### 10.2 Collective Memory (Shared knowledge)

- [ ] Validated lessons promotion (score >= 0.8, >= 3 confirming trades)
- [ ] Market patterns that work across users
- [ ] Strategy insights shared by personality type
- [ ] Anonymous source tracking

### 10.3 Memory Service

- [ ] StoreLesson() with validation
- [ ] SearchMemory() combining user + collective
- [ ] Memory expiration (TTL for short-term)
- [ ] Memory importance scoring

---

## Phase 11: Self-Evaluation System (Week 9)

### 11.1 Strategy Evaluator

- [ ] Calculate strategy stats (win rate, profit factor, expected value)
- [ ] Per-strategy performance tracking
- [ ] Sharpe ratio calculation
- [ ] Max drawdown per strategy

### 11.2 Performance Analysis

- [ ] EvaluateStrategies() for time period
- [ ] GetUnderperformingStrategies() with criteria (win rate < 35%, profit factor < 0.8)
- [ ] DisableStrategy() for poor performers
- [ ] Strategy re-enabling logic after improvement

### 11.3 Trade Journal

- [ ] JournalEntry creation after each trade
- [ ] Lessons learned (AI-generated)
- [ ] Improvement tips generation
- [ ] Context capture (indicators, regime, confidence)

### 11.4 Daily Reports

- [ ] Daily performance summary
- [ ] Strategy breakdown
- [ ] Best/worst trades of the day
- [ ] Recommendations for improvement

---

## Phase 12: Telegram Bot (Week 10)

### 12.1 Bot Infrastructure

- [ ] Telegram bot handler with long polling
- [ ] Command registry
- [ ] Callback query handler
- [ ] Conversation state management (Redis)
- [ ] Auto-user registration

### 12.2 Core Commands

- [ ] /start: Welcome and registration
- [ ] /help: Show all commands
- [ ] /connect: Connect exchange (format: /connect bybit API_KEY API_SECRET)
- [ ] /disconnect: Remove exchange account
- [ ] /exchanges: List connected exchanges
- [ ] /open-position: Open position (format: /open-position BTC 100)
- [ ] /update-budget: Update budget for pair
- [ ] /stop-trading: Pause trading for pair
- [ ] /resume-trading: Resume trading for pair
- [ ] /list-positions: Show all active positions

### 12.3 Monitoring Commands

- [ ] /balance: Show balances
- [ ] /positions: Show open positions with PnL
- [ ] /orders: Show open orders
- [ ] /stats: Trading statistics
- [ ] /pnl: Today's PnL
- [ ] /report: Generate performance report

### 12.4 Settings Commands

- [ ] /settings: User settings menu
- [ ] /pause: Pause ALL trading
- [ ] /resume: Resume ALL trading
- [ ] /killswitch: Emergency close all positions

### 12.5 Notification System

- [ ] Trade opened notifications
- [ ] Trade closed notifications
- [ ] Stop loss hit notifications
- [ ] Take profit hit notifications
- [ ] Circuit breaker triggered alerts
- [ ] Daily report delivery

---

## Phase 13: Data Source Integration (Week 11)

### 13.1 News Sources

- [ ] CoinDesk provider
- [ ] CoinTelegraph provider
- [ ] The Block provider
- [ ] NewsSource interface standardization

### 13.2 Sentiment Sources

- [ ] Santiment provider
- [ ] LunarCrush provider
- [ ] Twitter/X.com provider
- [ ] Reddit provider
- [ ] SentimentSource interface

### 13.3 On-Chain Sources

- [ ] Glassnode provider
- [ ] Santiment on-chain provider
- [ ] OnChainSource interface

### 13.4 Derivatives Sources

- [ ] Deribit provider
- [ ] Laevitas provider
- [ ] Greeks.live provider
- [ ] DerivativesSource interface

### 13.5 Liquidation Sources

- [ ] Coinglass provider
- [ ] Hyblock provider
- [ ] LiquidationSource interface

### 13.6 Macro Sources

- [ ] Investing.com provider (economic calendar)
- [ ] FRED provider (Federal Reserve data)
- [ ] CME FedWatch provider
- [ ] MacroSource interface

---

## Phase 14: Kafka Event Streaming (Week 11)

### 14.1 Topic Definitions

- [ ] Trading events (trades.opened, trades.closed, orders.placed, orders.filled)
- [ ] Risk events (risk.alerts, risk.circuit_breaker, risk.kill_switch)
- [ ] Market data events (market.prices, market.liquidations, market.whale_alerts)
- [ ] Agent events (agents.decisions, agents.signals)
- [ ] Notifications (notifications.telegram)

### 14.2 Event Types

- [ ] TradeEvent struct
- [ ] RiskEvent struct
- [ ] WhaleAlertEvent struct
- [ ] AgentSignalEvent struct

### 14.3 Kafka Consumers

- [ ] Notification consumer → Telegram bot
- [ ] Risk event consumer → Emergency actions
- [ ] Whale alert consumer → Agent notifications

---

## Phase 15: Testing & Quality (Week 12)

### 15.1 Unit Tests

- [ ] Domain logic tests
- [ ] Tool calculation tests
- [ ] Repository tests (with test DB)
- [ ] Agent prompt rendering tests

### 15.2 Integration Tests

- [ ] Exchange adapter tests (testnet)
- [ ] Database integration tests
- [ ] Kafka producer/consumer tests
- [ ] Redis cache/lock tests

### 15.3 End-to-End Tests

- [ ] Full trading flow test (mock exchange)
- [ ] Risk engine circuit breaker test
- [ ] Agent pipeline execution test
- [ ] Telegram bot command tests

### 15.4 Load Testing

- [ ] Multi-user concurrent execution
- [ ] Worker performance under load
- [ ] Database query optimization
- [ ] Kafka throughput testing

---

## Phase 16: Deployment & DevOps (Week 12)

### 16.1 Docker Setup

- [ ] Dockerfile with multi-stage build
- [ ] docker-compose.yml with all services
- [ ] Volume management for persistence
- [ ] Network configuration

### 16.2 Services Configuration

- [ ] PostgreSQL with pgvector
- [ ] ClickHouse with custom config
- [ ] Redis with persistence
- [ ] Kafka with KRaft (no Zookeeper)
- [ ] Kafka UI for monitoring

### 16.3 Monitoring Setup

- [ ] Prometheus metrics export
- [ ] Grafana dashboards
- [ ] Error tracking (Sentry integration)
- [ ] Log aggregation

### 16.4 CI/CD Pipeline

- [ ] GitHub Actions for tests
- [ ] Docker image building
- [ ] Automated deployment
- [ ] Database migration automation

---

## Phase 17: Advanced Features (Weeks 13-14)

### 17.1 ML Integration Preparation

- [ ] ML model interface (ONNX runtime)
- [ ] Regime detection model placeholder
- [ ] Anomaly detection model placeholder
- [ ] Feature pipeline for ML models

### 17.2 Advanced Order Types

- [ ] Iceberg orders with hidden size
- [ ] TWAP/VWAP execution algorithms
- [ ] Ladder orders with dynamic TP levels
- [ ] DCA strategies

### 17.3 Portfolio Management

- [ ] Portfolio-level risk calculation
- [ ] Correlation-based position sizing
- [ ] Rebalancing strategies
- [ ] Multi-pair optimization

### 17.4 Advanced Analytics

- [ ] Monte Carlo simulation for risk
- [ ] Backtesting framework preparation
- [ ] Strategy optimization engine
- [ ] Performance attribution analysis

---

## Phase 18: Documentation & Polish (Week 14)

### 18.1 Code Documentation

- [ ] GoDoc comments for all public APIs
- [ ] Architecture decision records (ADR)
- [ ] Database schema documentation
- [ ] API endpoint documentation (if REST API added)

### 18.2 User Documentation

- [ ] README.md with quick start
- [ ] Telegram bot user guide
- [ ] Trading strategies explanation
- [ ] Risk management guide
- [ ] Troubleshooting guide

### 18.3 Deployment Documentation

- [ ] Installation guide
- [ ] Environment variables reference
- [ ] Docker deployment guide
- [ ] Production setup checklist
- [ ] Backup and recovery procedures

### 18.4 Code Quality

- [ ] Linter configuration (golangci-lint)
- [ ] Code formatting standards
- [ ] Error handling patterns
- [ ] Security audit checklist

---

## Key Milestones

| Milestone                    | Week    | Deliverable                                     |
| ---------------------------- | ------- | ----------------------------------------------- |
| **M1: Foundation Complete**  | Week 2  | Database, config, logging, adapters working     |
| **M2: Domain & Repos Ready** | Week 3  | All entities, repositories, migrations complete |
| **M3: Exchange Integration** | Week 3  | Can fetch data and place orders on testnet      |
| **M4: AI & Templates**       | Week 4  | All providers and prompts configured            |
| **M5: Tools Complete**       | Week 6  | All 80+ tools implemented and tested            |
| **M6: Agent System**         | Week 7  | All 13 agents with CoT working                  |
| **M7: Workers Running**      | Week 8  | All data collection and analysis workers active |
| **M8: Risk Engine Active**   | Week 8  | Circuit breaker and kill switch operational     |
| **M9: Memory System**        | Week 9  | User + collective memory with semantic search   |
| **M10: Self-Evaluation**     | Week 9  | Strategy analysis and auto-disable working      |
| **M11: Telegram Bot Live**   | Week 10 | Full bot with all commands functional           |
| **M12: Data Sources**        | Week 11 | All external data providers integrated          |
| **M13: Kafka Streaming**     | Week 11 | Event-driven architecture complete              |
| **M14: Testing Complete**    | Week 12 | Unit, integration, E2E tests passing            |
| **M15: Production Ready**    | Week 12 | Deployed and monitored                          |
| **M16: Advanced Features**   | Week 14 | ML prep, advanced orders, portfolio management  |
| **M17: Documentation**       | Week 14 | All docs complete, code production-ready        |

---

## Critical Success Factors

### Architecture Principles

- **LLM for reasoning** - All strategic decisions
- **Rule-based for safety** - Hard stops, circuit breakers
- **Multi-user background execution** - One agent pool serves all users
- **User data isolation** - Each user has separate memory, positions, budget
- **Fail-safe design** - System degrades gracefully, never loses data

### Performance Requirements

- Market scanner processes ALL users in < 5 minutes
- Agent execution < 30 seconds per analysis
- Risk checks < 100ms (critical path)
- Order execution < 1 second
- Database queries < 50ms (p95)

### Security Requirements

- All API keys encrypted (AES-256-GCM)
- API keys never logged
- User data isolated in database
- Rate limiting on all external APIs
- Circuit breaker protects capital

### Reliability Requirements

- Graceful shutdown with data flush
- Database transaction safety
- Kafka message delivery guarantees
- Automatic retry with exponential backoff
- Health checks for all services

---

## Risk Mitigation

### Technical Risks

- **AI costs too high** → Implement cost limits per execution, use cheaper models for non-critical tasks
- **Exchange API rate limits** → Implement request pooling, caching, and adaptive rate limiting
- **Database performance** → Use proper indexes, connection pooling, query optimization
- **Kafka lag** → Monitor consumer lag, scale consumers horizontally

### Business Risks

- **Poor trading performance** → Self-evaluation system disables bad strategies automatically
- **User losses** → Circuit breaker and kill switch protect capital
- **Data quality issues** → Multiple data sources for redundancy, anomaly detection

### Operational Risks

- **Service downtime** → Health checks, auto-restart, monitoring alerts
- **Data loss** → Regular backups, transaction safety, audit logs
- **Security breach** → Encryption, API key rotation, audit trails

---

## Post-MVP Roadmap (Phase 19+)

### Phase 19: ML Integration (Month 4)

- Train regime detection model
- Deploy anomaly detection
- Integrate price prediction models
- Implement reinforcement learning for execution

### Phase 20: Web Dashboard (Month 5)

- React dashboard for analytics
- Real-time position monitoring
- Strategy backtesting UI
- Performance visualization

### Phase 21: Advanced Features (Month 6)

- Copy trading functionality
- Strategy marketplace
- Multi-user workspaces
- Mobile app (iOS/Android)

---

**End of Development Plan**

Refer to `specs.md` for detailed implementation specifications.
