<!-- @format -->

# Prometheus — Action Plan

**Дата создания:** 26 ноября 2025  
**Цель:** Довести систему до production-ready состояния  
**Срок:** 4-6 недель

---

## 🎯 Sprint 1: Risk & Safety (Неделя 1-2)

### Цель: Сделать систему безопасной для real money trading

---

### Task 1.1: Risk Engine Core (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 2 дня  
**Файлы:** `internal/risk/engine.go`

```go
□ Создать internal/risk/engine.go
□ Реализовать RiskEngine struct:
  type RiskEngine struct {
      riskRepo risk.Repository
      posRepo  position.Repository
      redis    *redis.Client
      log      *logger.Logger
  }

□ Реализовать CanTrade(ctx, userID) (bool, error)
  - Проверка daily drawdown
  - Проверка consecutive losses
  - Проверка max exposure
  - Проверка circuit breaker state

□ Реализовать RecordTrade(ctx, userID, pnl)
  - Обновление daily stats
  - Обновление circuit breaker state
  - Сохранение в Redis cache

□ Реализовать GetUserState(ctx, userID) (*UserRiskState, error)

□ Реализовать ResetDaily(ctx) error
  - Сброс daily counters
  - Reset circuit breaker если conditions met

□ Добавить тесты:
  - TestCanTrade_WithinLimits
  - TestCanTrade_ExceedDrawdown
  - TestCanTrade_ConsecutiveLosses
  - TestRecordTrade_UpdatesState
```

**Критерий готовности:**

- ✅ CanTrade() блокирует trading при превышении лимитов
- ✅ RecordTrade() обновляет состояние
- ✅ Тесты покрывают все edge cases

---

### Task 1.2: Risk Middleware (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/tools/trading/middleware.go`

```go
□ Создать withRiskCheck() middleware:
  func withRiskCheck(engine *risk.RiskEngine, fn tradingFunc) tradingFunc {
      return func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
          userID := extractUserID(ctx)

          canTrade, err := engine.CanTrade(ctx, userID)
          if err != nil {
              return nil, fmt.Errorf("risk check failed: %w", err)
          }
          if !canTrade {
              return nil, errors.New("trading blocked by risk engine")
          }

          return fn(ctx, args)
      }
  }

□ Обернуть все trading tools:
  - place_order
  - place_bracket_order
  - place_ladder_order
  - add_to_position

□ Добавить integration tests
```

**Критерий готовности:**

- ✅ Trading tools не работают без risk check
- ✅ Circuit breaker блокирует orders

---

### Task 1.3: Kill Switch Implementation (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/risk/killswitch.go`, `internal/tools/risk/emergency_close_all.go`

```go
□ Реализовать KillSwitch struct в internal/risk/killswitch.go

□ Реализовать EmergencyCloseAll реальную логику:
  - Получить все открытые позиции пользователя
  - Закрыть все позиции market orders
  - Отменить все open orders
  - Установить circuit breaker в BLOCKED state
  - Отправить Kafka event для уведомления

□ Добавить Redis flag для kill switch state

□ Добавить /killswitch команду в Telegram bot (stub пока)

□ Тесты:
  - TestEmergencyCloseAll_ClosesAllPositions
  - TestEmergencyCloseAll_CancelsOrders
  - TestEmergencyCloseAll_BlocksTrading
```

**Критерий готовности:**

- ✅ emergency_close_all закрывает все позиции
- ✅ После kill switch trading заблокирован
- ✅ Можно разблокировать вручную через admin interface

---

### Task 1.4: Circuit Breaker Tests (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 1 день  
**Файлы:** `internal/risk/engine_test.go`

```go
□ TestCircuitBreaker_MaxDrawdown
  - User loses 10% in a day
  - Circuit breaker trips
  - Trading blocked

□ TestCircuitBreaker_ConsecutiveLosses
  - 3 consecutive losing trades
  - Circuit breaker trips

□ TestCircuitBreaker_Reset
  - After 24h circuit breaker resets
  - Trading allowed again

□ TestCircuitBreaker_ManualOverride
  - Admin can unblock manually

□ Integration test с real repositories
```

**Критерий готовности:**

- ✅ 100% coverage для risk engine
- ✅ Integration tests проходят

---

## 🤖 Sprint 2: Workers Infrastructure (Неделя 2-3)

### Цель: Система автоматически мониторит позиции и собирает данные

---

### Task 2.1: Worker Interface & Scheduler (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 2 дня  
**Файлы:** `internal/workers/worker.go`, `internal/workers/scheduler.go`

```go
□ Создать internal/workers/worker.go:

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
      log      *logger.Logger
  }

□ Создать internal/workers/scheduler.go:

  type Scheduler struct {
      workers []Worker
      ctx     context.Context
      cancel  context.CancelFunc
      wg      sync.WaitGroup
      log     *logger.Logger
  }

  func (s *Scheduler) Start(ctx context.Context) error
  func (s *Scheduler) Stop() error
  func (s *Scheduler) RegisterWorker(w Worker)

□ Graceful shutdown support

□ Тесты:
  - TestScheduler_StartStop
  - TestScheduler_GracefulShutdown
```

**Критерий готовности:**

- ✅ Scheduler запускает/останавливает workers
- ✅ Graceful shutdown работает
- ✅ Context cancellation propagates

---

### Task 2.2: Position Monitor Worker (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/workers/trading/position_monitor.go`

```go
□ Создать PositionMonitor worker (интервал: 30s):

  type PositionMonitor struct {
      BaseWorker
      posRepo    position.Repository
      exchFactory *exchanges.Factory
      riskEngine  *risk.Engine
      kafka       *kafka.Producer
  }

□ Логика Run():
  1. Получить все открытые позиции (всех пользователей)
  2. Для каждой позиции:
     - Получить текущую цену с биржи
     - Обновить unrealized PnL
     - Проверить SL/TP levels
     - Если SL/TP hit → отправить event в Kafka
  3. Обновить positions в БД

□ Добавить Kafka events:
  - position.sl_hit
  - position.tp_hit
  - position.pnl_updated

□ Тесты с mock exchange
```

**Критерий готовности:**

- ✅ Позиции мониторятся каждые 30s
- ✅ PnL обновляется
- ✅ SL/TP events генерируются

---

### Task 2.3: Order Sync Worker (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/workers/trading/order_sync.go`

```go
□ Создать OrderSync worker (интервал: 10s):

  type OrderSync struct {
      BaseWorker
      orderRepo   order.Repository
      posRepo     position.Repository
      exchFactory *exchanges.Factory
      kafka       *kafka.Producer
  }

□ Логика Run():
  1. Получить все pending/partially_filled orders
  2. Для каждого ордера:
     - Запросить status с биржи
     - Если filled → обновить order + создать/обновить position
     - Если cancelled → обновить order status
     - Если partially_filled → обновить filled_amount
  3. Отправить events в Kafka

□ Kafka events:
  - order.filled
  - order.cancelled
  - order.partially_filled

□ Тесты
```

**Критерий готовности:**

- ✅ Orders синхронизируются с биржей
- ✅ Positions создаются при fill
- ✅ Events генерируются

---

### Task 2.4: Risk Monitor Worker (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/workers/trading/risk_monitor.go`

```go
□ Создать RiskMonitor worker (интервал: 10s):

  type RiskMonitor struct {
      BaseWorker
      riskEngine *risk.Engine
      userRepo   user.Repository
      posRepo    position.Repository
      kafka      *kafka.Producer
  }

□ Логика Run():
  1. Получить всех active users
  2. Для каждого user:
     - Посчитать daily PnL
     - Проверить drawdown
     - Проверить consecutive losses
     - Если лимиты превышены → trip circuit breaker
     - Отправить alert в Kafka

□ Kafka events:
  - risk.drawdown_warning (80% of limit)
  - risk.circuit_breaker_tripped
  - risk.consecutive_losses

□ Тесты
```

**Критерий готовности:**

- ✅ Risk limits мониторятся
- ✅ Circuit breaker trips автоматически
- ✅ Alerts генерируются

---

### Task 2.5: OHLCV Collector Worker (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 1 день  
**Файлы:** `internal/workers/marketdata/ohlcv_collector.go`

```go
□ Создать OHLCVCollector worker (интервал: 1m):

  type OHLCVCollector struct {
      BaseWorker
      mdRepo      market_data.Repository
      exchFactory *exchanges.Factory
      symbols     []string  // BTC, ETH, etc.
      timeframes  []string  // 1m, 5m, 15m, 1h, 4h, 1d
  }

□ Логика Run():
  1. Для каждого symbol + timeframe:
     - Получить latest candle с биржи (central API keys)
     - Сохранить в ClickHouse
  2. Batch insert (100+ candles за раз)

□ Rate limiting (не превышать exchange limits)

□ Retry с exponential backoff

□ Тесты с mock exchange
```

**Критерий готовности:**

- ✅ OHLCV data собирается для всех pairs
- ✅ Data сохраняется в ClickHouse
- ✅ Rate limiting работает

---

### Task 2.6: Integrate Workers into main.go (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 0.5 дня  
**Файлы:** `cmd/main.go`

```go
□ Добавить в main.go:

  func initWorkers(
      cfg *config.Config,
      repos *Repositories,
      riskEngine *risk.Engine,
      exchFactory *exchanges.Factory,
      kafka *kafka.Producer,
  ) *workers.Scheduler {
      scheduler := workers.NewScheduler()

      // Trading workers
      scheduler.RegisterWorker(
          trading.NewPositionMonitor(...),
      )
      scheduler.RegisterWorker(
          trading.NewOrderSync(...),
      )
      scheduler.RegisterWorker(
          trading.NewRiskMonitor(...),
      )

      // Market data workers
      scheduler.RegisterWorker(
          marketdata.NewOHLCVCollector(...),
      )

      return scheduler
  }

□ В main():
  workers := initWorkers(...)
  if err := workers.Start(ctx); err != nil {
      log.Fatal(err)
  }
  defer workers.Stop()

□ Тесты интеграции
```

**Критерий готовности:**

- ✅ Workers запускаются при старте
- ✅ Graceful shutdown работает
- ✅ Logs показывают worker activity

---

## 🔧 Sprint 3: Tools Implementation (Неделя 3-4)

### Цель: Агенты имеют доступ к необходимым инструментам

---

### Task 3.1: Market Data Tools (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 2 дня  
**Файлы:** `internal/tools/market/*.go`

```go
□ get_funding_rate - реализовать реальную логику
□ get_open_interest - реализовать
□ get_long_short_ratio - реализовать
□ get_liquidations - интегрировать с ClickHouse

□ Каждый tool:
  - Извлечь параметры (symbol, exchange, timeframe)
  - Получить данные из ClickHouse или exchange API
  - Вернуть результат в standardized format

□ Middleware:
  - Timeout per tool (10s)
  - Retry logic (3 attempts)
  - Stats tracking

□ Тесты с mock data
```

**Критерий готовности:**

- ✅ 4/4 market data tools работают
- ✅ Возвращают real data
- ✅ Tests pass

---

### Task 3.2: Technical Indicators (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 2 дня  
**Файлы:** `internal/tools/indicators/*.go`

```go
□ Реализовать 10 indicators:

Momentum (4):
□ stochastic
□ cci
□ roc
□ williams_r

Volume (3):
□ vwap
□ obv
□ volume_profile

Volatility (2):
□ bollinger
□ keltner

Trend (1):
□ ichimoku

□ Использовать library или написать свои (простые indicators)

□ Helpers для OHLCV fetch из ClickHouse

□ Тесты с fixture data
```

**Критерий готовности:**

- ✅ 10 indicators реализованы
- ✅ Calculations correct
- ✅ Tests pass

---

### Task 3.3: Trading Execution Tools (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 2 дня  
**Файлы:** `internal/tools/trading/*.go`

```go
□ place_bracket_order - Entry + SL + TP:
  type BracketOrderArgs struct {
      Symbol    string
      Side      string
      Amount    float64
      EntryPrice float64  // optional
      StopLoss   float64
      TakeProfit float64
  }

  Logic:
  1. Place entry order (market or limit)
  2. Wait for fill (or poll order status)
  3. Place SL stop_market order
  4. Place TP limit order
  5. Link orders (parent_order_id)
  6. Return all 3 order IDs

□ place_ladder_order - Entry + multiple TPs:
  type LadderOrderArgs struct {
      Symbol     string
      Side       string
      Amount     float64
      EntryPrice float64
      StopLoss   float64
      TakeProfits []struct {
          Price   float64
          Percent float64  // % of position to close
      }
  }

  Logic:
  1. Place entry
  2. Place SL
  3. Place multiple TP orders (split amount)

□ cancel_all_orders - реализовать реально

□ close_position - реализовать реально

□ move_sl_to_breakeven - helper function

□ set_trailing_stop - если exchange supports

□ Тесты с mock exchange
```

**Критерий готовности:**

- ✅ Bracket orders работают
- ✅ Ladder orders работают
- ✅ Position management tools работают
- ✅ Integration tests с testnet

---

### Task 3.4: Memory & Evaluation Tools (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 1 день  
**Файлы:** `internal/tools/memory/*.go`, `internal/tools/evaluation/*.go`

```go
Memory tools:
□ store_memory - сохранить observation/decision
□ get_trade_history - получить past trades
□ get_market_regime - текущий regime
□ store_market_regime - сохранить regime

Evaluation tools:
□ get_strategy_stats - win rate, profit factor
□ log_trade_decision - создать journal entry
□ get_trade_journal - получить journal entries
□ evaluate_last_trades - aggregate analysis
□ get_best_strategies - top performers
□ get_worst_strategies - underperformers

□ Интеграция с repositories

□ Тесты
```

**Критерий готовности:**

- ✅ 9/9 tools реализованы
- ✅ Работают с real DB
- ✅ Tests pass

---

### Task 3.5: Update Tool Catalog (Приоритет: 🟡 СРЕДНИЙ)

**Срок:** 0.5 дня  
**Файлы:** `internal/tools/catalog.go`

```go
□ Обновить implementedTools map:

var implementedTools = map[string]toolFactory{
    // Market Data (8/8) ✅
    "get_price": market.NewGetPriceTool,
    "get_ohlcv": market.NewGetOHLCVTool,
    "get_orderbook": market.NewGetOrderBookTool,
    "get_trades": market.NewGetTradesTool,
    "get_funding_rate": market.NewGetFundingRateTool,
    "get_open_interest": market.NewGetOpenInterestTool,
    "get_long_short_ratio": market.NewGetLongShortRatioTool,
    "get_liquidations": market.NewGetLiquidationsTool,

    // Indicators (14/18)
    "rsi": indicators.NewRSITool,
    "ema": indicators.NewEMATool,
    "macd": indicators.NewMACDTool,
    "atr": indicators.NewATRTool,
    "stochastic": indicators.NewStochasticTool,
    // ... etc

    // Trading (10/14)
    "get_balance": trading.NewGetBalanceTool,
    "get_positions": trading.NewGetPositionsTool,
    "place_order": trading.NewPlaceOrderTool,
    "place_bracket_order": trading.NewPlaceBracketOrderTool,
    // ... etc

    // Memory (5/5) ✅
    // Evaluation (6/6) ✅
    // Risk (7/7) ✅
}

□ Убрать stubs для реализованных tools

□ Обновить docs/DEVELOPMENT_PLAN.md
```

**Критерий готовности:**

- ✅ implementedTools содержит 40+ tools
- ✅ Stubs остались только для advanced tools (SMC, sentiment, etc.)
- ✅ Docs updated

---

## 💬 Sprint 4: Telegram Bot (Неделя 4-5)

### Цель: Пользователи могут управлять системой через Telegram

---

### Task 4.1: Bot Infrastructure (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/adapters/telegram/bot.go`

```go
□ Создать internal/adapters/telegram/bot.go:

  type Bot struct {
      api      *tgbotapi.BotAPI
      userRepo user.Repository
      redis    *redis.Client
      handlers map[string]CommandHandler
      log      *logger.Logger
  }

  type CommandHandler func(ctx context.Context, msg *tgbotapi.Message) error

□ Методы:
  func (b *Bot) Start(ctx context.Context) error
  func (b *Bot) RegisterCommand(cmd string, handler CommandHandler)
  func (b *Bot) SendMessage(chatID int64, text string) error
  func (b *Bot) SendNotification(userID uuid.UUID, template string, data interface{}) error

□ Long polling loop с context cancellation

□ Auto-user registration (при /start создать user если не существует)

□ Graceful shutdown

□ Тесты с mock Telegram API
```

**Критерий готовности:**

- ✅ Bot запускается и принимает команды
- ✅ User registration работает
- ✅ Graceful shutdown

---

### Task 4.2: Core Commands (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 2 дня  
**Файлы:** `internal/adapters/telegram/handlers/*.go`

```go
□ /start - Welcome message + auto-registration

□ /connect <exchange> <api_key> <secret>
  - Validate credentials (test API call)
  - Encrypt keys
  - Save ExchangeAccount to DB
  - Send confirmation

□ /disconnect <exchange>
  - List connected exchanges
  - Remove selected
  - Confirmation required

□ /balance
  - Get balances from all connected exchanges
  - Format nicely with emojis

□ /positions
  - Show all open positions
  - Include PnL, SL/TP levels
  - Add quick actions (close, modify)

□ /orders
  - Show pending orders
  - Option to cancel

□ /stats
  - Today's PnL
  - Win rate
  - Total trades
  - Best/worst trades

□ /help
  - List all commands
  - Usage examples

□ Inline keyboards для interactive actions

□ Тесты
```

**Критерий готовности:**

- ✅ 8 core commands работают
- ✅ Exchange connection works
- ✅ Real data displayed

---

### Task 4.3: Trading Commands (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 1 день  
**Файлы:** `internal/adapters/telegram/handlers/trading.go`

```go
□ /open_position <symbol> <size>
  - Start conversation flow
  - Ask: Long or Short?
  - Ask: Entry type (market/limit)?
  - Ask: Stop Loss %?
  - Ask: Take Profit %?
  - Confirm details
  - Execute via agents (RiskManager + Executor)

□ /close_position <position_id>
  - Show position details
  - Confirm close
  - Execute market close

□ /modify_sl <position_id> <new_sl>
  - Update stop loss
  - Confirm via exchange API

□ /pause
  - Pause ALL trading for this user
  - Set flag in DB

□ /resume
  - Resume trading

□ /killswitch
  - Trigger emergency_close_all
  - Confirmation required (type "CONFIRM")
  - Close all positions
  - Block trading

□ Conversation state через Redis

□ Тесты
```

**Критерий готовности:**

- ✅ Trading commands работают
- ✅ Conversation flows работают
- ✅ Safety checks (confirmations)

---

### Task 4.4: Notification System (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 1 день  
**Файлы:** `internal/adapters/telegram/notifications.go`

```go
□ Создать NotificationService:

  type NotificationService struct {
      bot       *Bot
      userRepo  user.Repository
      templates *templates.Registry
      kafka     *kafka.Consumer
  }

□ Методы:
  func (n *NotificationService) SendTradeOpened(userID uuid.UUID, order *order.Order)
  func (n *NotificationService) SendTradeClosed(userID uuid.UUID, position *position.Position)
  func (n *NotificationService) SendStopLossHit(...)
  func (n *NotificationService) SendTakeProfitHit(...)
  func (n *NotificationService) SendCircuitBreakerTriggered(...)
  func (n *NotificationService) SendDailyReport(...)

□ Интеграция с templates:
  data := map[string]interface{}{
      "Symbol": order.Symbol,
      "Side": order.Side,
      "Price": order.Price,
      // ...
  }
  text, _ := templates.Render("notifications/trade_opened", data)
  bot.SendMessage(chatID, text)

□ Kafka consumer:
  - Слушать события: trades.opened, trades.closed, risk.*, position.*
  - Для каждого события → отправить уведомление user'у

□ Тесты
```

**Критерий готовности:**

- ✅ Notifications отправляются на события
- ✅ Templates используются
- ✅ Kafka integration works

---

### Task 4.5: Integrate Bot into main.go (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 0.5 дня  
**Файлы:** `cmd/main.go`

```go
□ Добавить в main.go:

  func initTelegramBot(
      cfg *config.Config,
      repos *Repositories,
      agentSystem *AgentSystem,
      kafka *kafka.Consumer,
  ) (*telegram.Bot, *telegram.NotificationService, error) {
      bot := telegram.NewBot(cfg.Telegram, repos.User, db.Redis)

      // Register command handlers
      handlers := telegram.NewHandlers(bot, repos, agentSystem)
      bot.RegisterCommand("start", handlers.Start)
      bot.RegisterCommand("connect", handlers.Connect)
      // ... etc

      notifService := telegram.NewNotificationService(bot, kafka, repos.User, templates.Get())

      return bot, notifService, nil
  }

□ В main():
  bot, notifService, err := initTelegramBot(...)
  go bot.Start(ctx)
  go notifService.Start(ctx)

□ Тесты
```

**Критерий готовности:**

- ✅ Bot запускается вместе с системой
- ✅ Commands работают
- ✅ Notifications приходят

---

## ✅ Sprint 5: Testing & Quality (Неделя 5-6)

### Цель: Довести test coverage до 60%+

---

### Task 5.1: Repository Tests (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 2 дня  
**Файлы:** `internal/repository/postgres/*_test.go`

```go
□ Для каждого repository создать *_test.go:

  func TestUserRepository_Create(t *testing.T) {
      db := testsupport.SetupPostgres(t)
      defer db.Teardown()

      repo := postgres.NewUserRepository(db.DB())
      user := &user.User{...}

      err := repo.Create(context.Background(), user)
      require.NoError(t, err)

      // Verify
      fetched, err := repo.GetByID(context.Background(), user.ID)
      require.NoError(t, err)
      assert.Equal(t, user.TelegramID, fetched.TelegramID)
  }

□ Table-driven tests для edge cases

□ Repositories to test:
  - UserRepository (5 methods)
  - ExchangeAccountRepository (7 methods)
  - TradingPairRepository (7 methods)
  - OrderRepository (10 methods)
  - PositionRepository (8 methods)
  - MemoryRepository (7 methods)
  - JournalRepository (5 methods)
  - RiskRepository (4 methods)
  - ReasoningRepository (5 methods)

□ Использовать testsupport.SetupPostgres(t)

□ Target: 80%+ coverage для repositories
```

**Критерий готовности:**

- ✅ Все repositories протестированы
- ✅ Coverage 80%+
- ✅ Tests pass

---

### Task 5.2: Service Tests (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 2 дня  
**Файлы:** `internal/domain/*/service_test.go`

```go
□ Unit tests для services с mock repositories:

  func TestJournalService_GetStrategyStats(t *testing.T) {
      mockRepo := &mockJournalRepo{
          strategyStats: []journal.StrategyStats{...},
      }
      service := journal.NewService(mockRepo)

      stats, err := service.GetStrategyStats(ctx, userID, since)
      require.NoError(t, err)
      assert.Len(t, stats, 2)
  }

□ Services to test:
  - UserService
  - ExchangeAccountService (encryption!)
  - TradingPairService
  - OrderService
  - PositionService
  - MemoryService
  - JournalService
  - RiskService

□ Mock repositories with testify/mock

□ Target: 70%+ coverage
```

**Критерий готовности:**

- ✅ All services tested
- ✅ Business logic validated
- ✅ Coverage 70%+

---

### Task 5.3: Exchange Adapter Tests (Приоритет: 🔴 КРИТИЧНЫЙ)

**Срок:** 2 дня  
**Файлы:** `internal/adapters/exchanges/*/client_test.go`

```go
□ Integration tests с testnet:

  func TestBinanceClient_PlaceOrder_Testnet(t *testing.T) {
      if testing.Short() {
          t.Skip("skipping integration test")
      }

      client := binance.NewClient(binance.Config{
          APIKey: os.Getenv("BINANCE_TESTNET_API_KEY"),
          SecretKey: os.Getenv("BINANCE_TESTNET_SECRET"),
          Testnet: true,
      })

      order, err := client.PlaceOrder(ctx, &exchanges.OrderRequest{
          Symbol: "BTCUSDT",
          Side: "buy",
          Type: "limit",
          Amount: 0.001,
          Price: 20000.0,
      })

      require.NoError(t, err)
      assert.NotEmpty(t, order.ID)

      // Cleanup: cancel order
      client.CancelOrder(ctx, "BTCUSDT", order.ID)
  }

□ Unit tests с mock HTTP client

□ Test все exchanges:
  - Binance (Spot + Futures)
  - Bybit (Spot + Futures)
  - OKX (Spot + Futures)

□ Test все operations:
  - Market data (GetTicker, GetOHLCV, etc.)
  - Trading (PlaceOrder, CancelOrder)
  - Account (GetBalance, GetPositions)
  - Futures (SetLeverage, SetMarginMode)

□ Target: 60%+ coverage
```

**Критерий готовности:**

- ✅ Integration tests с testnet pass
- ✅ All operations validated
- ✅ Coverage 60%+

---

### Task 5.4: Tool Tests (Приоритет: 🟡 ВЫСОКИЙ)

**Срок:** 1 день  
**Файлы:** `internal/tools/*/tool_test.go`

```go
□ Unit tests для tools:

  func TestRSITool(t *testing.T) {
      mockRepo := &mockMarketDataRepo{
          ohlcv: []marketdata.OHLCV{...},  // fixture data
      }
      deps := shared.Deps{MarketDataRepo: mockRepo}
      tool := indicators.NewRSITool(deps)

      result, err := tool.Execute(ctx, map[string]interface{}{
          "symbol": "BTCUSDT",
          "period": 14,
      })

      require.NoError(t, err)
      assert.InDelta(t, 65.5, result["rsi"], 0.1)
  }

□ Test categories:
  - Market data tools
  - Indicators
  - Trading tools
  - Risk tools
  - Memory tools
  - Evaluation tools

□ Mock dependencies

□ Target: 70%+ coverage
```

**Критерий готовности:**

- ✅ All implemented tools tested
- ✅ Edge cases covered
- ✅ Coverage 70%+

---

### Task 5.5: End-to-End Tests (Приоритет: 🟡 СРЕДНИЙ)

**Срок:** 1 день  
**Файлы:** `test/e2e/*_test.go`

```go
□ Создать test/e2e/ directory

□ E2E tests:

  func TestTradingFlow_EndToEnd(t *testing.T) {
      // 1. Setup: user, exchange account, trading pair
      // 2. Collect market data (OHLCV)
      // 3. Run agent analysis
      // 4. Generate trade signal
      // 5. Risk check
      // 6. Place order (mock exchange)
      // 7. Verify order in DB
      // 8. Simulate fill
      // 9. Verify position created
      // 10. Close position
      // 11. Verify journal entry
  }

  func TestCircuitBreaker_EndToEnd(t *testing.T) {
      // 1. User places 3 losing trades
      // 2. Circuit breaker trips
      // 3. Next trade attempt blocked
      // 4. Notification sent
  }

  func TestAgentPipeline_EndToEnd(t *testing.T) {
      // 1. Run MarketAnalyst agent
      // 2. Run SentimentAnalyst agent
      // 3. Run StrategyPlanner agent
      // 4. Run RiskManager agent
      // 5. Verify complete pipeline execution
  }

□ Mock external dependencies (exchanges, AI providers)

□ Use real DB (docker containers)
```

**Критерий готовности:**

- ✅ 3 E2E tests pass
- ✅ Critical paths validated
- ✅ Integration points verified

---

## 📊 Sprint 6: Advanced Features (Неделя 6+)

### Lower priority features - реализовать по необходимости

---

### Task 6.1: Data Sources Integration

```
□ CoinDesk news provider
□ Alternative.me Fear&Greed
□ Coinglass liquidations
□ Glassnode on-chain (если есть API key)
```

### Task 6.2: SMC Tools Implementation

```
□ detect_fvg (Fair Value Gaps)
□ detect_order_blocks
□ detect_liquidity_zones
□ get_market_structure (BOS, CHoCH)
```

### Task 6.3: Memory System Enhancements

```
□ Embedding generation via AI provider
□ Collective memory promotion logic
□ Memory expiration/TTL
□ Deduplication
```

### Task 6.4: Self-Evaluation Enhancements

```
□ Auto journal entry after trades
□ Strategy evaluation logic
□ AI-generated lessons
□ Auto-disable poor performers
```

### Task 6.5: Observability

```
□ OpenTelemetry tracing
□ Prometheus metrics export
□ Grafana dashboards
□ Alert rules
```

---

## ✅ Definition of Done

### MVP считается готовым когда:

1. ✅ **Risk Engine работает**

   - Circuit breaker trips на excessive losses
   - Kill switch закрывает все позиции
   - Daily reset функционирует

2. ✅ **Workers мониторят систему**

   - Позиции обновляются каждые 30s
   - Ордера синхронизируются каждые 10s
   - Risk checks каждые 10s
   - OHLCV собирается каждую минуту

3. ✅ **40+ tools реализованы**

   - Все market data tools
   - Все trading execution tools
   - 10+ indicators
   - All memory/evaluation tools

4. ✅ **Telegram Bot функционален**

   - Core commands работают
   - Trading commands работают
   - Notifications приходят
   - User может управлять системой

5. ✅ **Test coverage 60%+**

   - Repository tests
   - Service tests
   - Exchange adapter tests
   - Tool tests
   - 3+ E2E tests

6. ✅ **System runs stable**
   - No panics
   - No goroutine leaks
   - Graceful shutdown works
   - Error tracking active

---

## 🎯 Success Metrics

### Измеряемые KPIs:

1. **Test Coverage:** ≥ 60%
2. **Tools Implemented:** ≥ 40/80 (50%)
3. **Workers Running:** ≥ 5 critical workers
4. **Commands Available:** ≥ 15 bot commands
5. **Uptime:** ≥ 99% (local testing)
6. **Mean Time to Recovery:** < 5 minutes

---

**Last Updated:** 26 ноября 2025  
**Owner:** Development Team  
**Status:** 🚧 In Progress
