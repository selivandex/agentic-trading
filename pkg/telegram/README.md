<!-- @format -->

# Telegram Bot Framework

**Self-contained, production-ready framework for building Telegram bots with clean architecture principles.**

No direct dependencies on `tgbotapi` - everything works through interfaces!

## Features

- 🎯 **Command Registry** - Declarative command registration with middleware support
- 📱 **Menu Navigator** - State-machine based menu system with navigation
- ⚡ **Async Messaging** - Non-blocking message sending with queue and rate limiting
- 💥 **Self-Destructing Messages** - Auto-delete messages after timeout
- ✅ **Input Validation** - Reusable validators for user input
- 🎨 **Message Builder** - Fluent API for constructing messages
- 🔄 **Middleware** - Composable request/response middleware
- 📊 **Pagination** - Built-in pagination support
- 📝 **Template Registry** - Embedded templates with rich helpers
- 💾 **Session Management** - In-memory and custom storage adapters
- 🗺️ **Menu Registry** - Automatic routing for multi-screen menus

## Architecture

```
pkg/telegram/
├── types.go              # Core interfaces (Bot, Session, TemplateRenderer)
├── command_registry.go   # Command registration and routing
├── middleware.go         # Common middleware implementations
├── async.go              # Async message queue and self-destruct
├── helpers.go            # Validators, builders, formatters
├── menu_navigator.go     # Multi-screen menu navigation framework
├── menu_registry.go      # Automatic menu routing
├── template_registry.go  # Template engine with embedded templates
├── session_memory.go     # In-memory session storage implementation
├── templates/            # Embedded .tmpl files (common, invest, etc.)
└── README.md             # This file
```

## Design Principles

1. **Interface-Based** - No direct dependency on `tgbotapi`, works through `Bot` interface
2. **Self-Contained** - All templates embedded, ready to use out of the box
3. **Testable** - All components mockable, includes in-memory implementations
4. **Type-Safe** - Generic `interface{}` for user types (cast in your app)
5. **DRY** - Screen builders eliminate boilerplate for common patterns

## Quick Start

```go
import "prometheus/pkg/telegram"

// 1. Create template registry (embedded templates)
templates, _ := telegram.NewDefaultTemplateRegistry()

// 2. Create session service
sessions := telegram.NewInMemorySessionService()

// 3. Create bot (implement telegram.Bot interface)
bot := myapp.NewTelegramBot(apiToken)

// 4. Create command registry
commands := telegram.NewCommandRegistry(bot, log)
commands.Use(telegram.LoggingMiddleware(log))
commands.Use(telegram.RecoveryMiddleware(log))

// 5. Create menu navigator
menuNav := telegram.NewMenuNavigator(sessions, bot, templates, log, 30*time.Minute)

// 6. Build your menus and register commands
// See examples below...
```

## Usage Examples

### 1. Templates

**Using embedded templates:**

```go
// Create registry with embedded templates
templates, err := telegram.NewDefaultTemplateRegistry()
if err != nil {
    log.Fatal(err)
}

// Render template
text, err := templates.Render("common/welcome", map[string]interface{}{
    "FirstName": "John",
    "UserID": "123",
})

// Templates have rich helpers:
// - {{money 1500.50}} → $1,500.50
// - {{bold "Hello"}} → *Hello*
// - {{safeText .UserInput}} → Escaped user input
// - {{checkmark}} → ✅
```

**Template helpers available:**

```
String: upper, lower, title, trim, replace, join, split
Math: add, sub, mul, div, mod
Format: money, percent, duration, date, datetime
Markdown: bold, italic, code, pre, link, escape
Emojis: checkmark, cross, warning, rocket, chart, fire
Logic: eq, ne, lt, gt, and, or, not
Collections: len, empty, first, last
```

### 2. Command Registry

**Register commands declaratively:**

```go
import "prometheus/pkg/telegram"

// Create registry
registry := telegram.NewCommandRegistry(bot, log)

// Add global middleware
registry.Use(telegram.LoggingMiddleware(log))
registry.Use(telegram.RecoveryMiddleware(log))
registry.Use(telegram.RateLimitMiddleware(10, log)) // 10 commands per minute

// Register commands
registry.Register(telegram.CommandConfig{
    Name:        "invest",
    Aliases:     []string{"i"},
    Description: "Start investment flow",
    Category:    "Trading",
    Handler: func(ctx *telegram.CommandContext) error {
        // Your handler logic
        return ctx.Bot.SendMessage(ctx.ChatID, "Starting investment flow...")
    },
})

registry.Register(telegram.CommandConfig{
    Name:        "status",
    Description: "Check portfolio status",
    Category:    "Portfolio",
    Middleware: []telegram.CommandMiddleware{
        telegram.AuthRequiredMiddleware(), // Command-specific middleware
    },
    Handler: func(ctx *telegram.CommandContext) error {
        user := ctx.User.(*user.User) // Cast to your user type
        return handleStatus(ctx, user)
    },
})

// Handle commands
registry.Handle(ctx, user, telegramID, chatID, "invest", "", rawMessage)
```

### 2. Message Builder

**Build messages with fluent API:**

```go
// Simple message
telegram.NewMessage(chatID, "Hello!").
    WithMarkdown().
    Silent().
    Send(bot)

// Message with keyboard
telegram.NewMessage(chatID, "Choose an option:").
    WithButtons(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Option 1", "opt1"),
            tgbotapi.NewInlineKeyboardButtonData("Option 2", "opt2"),
        ),
    ).
    Send(bot)

// Self-destructing message
telegram.NewMessage(chatID, "This will disappear in 5 seconds").
    SelfDestruct(5 * time.Second).
    Send(bot)

// Message with callback
telegram.NewMessage(chatID, "Processing...").
    OnSent(func(messageID int, err error) {
        if err != nil {
            log.Errorw("Failed to send", "error", err)
            return
        }
        log.Infow("Message sent", "message_id", messageID)
    }).
    Send(bot)
```

### 3. Async Message Queue

**Send messages asynchronously:**

```go
// Create queue
queue := telegram.NewAsyncMessageQueue(
    bot,
    5,                    // 5 workers
    50*time.Millisecond,  // 50ms rate limit
    log,
)

// Start workers
queue.Start()
defer queue.Stop()

// Enqueue messages
queue.Enqueue(chatID, "Notification 1", telegram.MessageOptions{}, nil)
queue.Enqueue(chatID, "Notification 2", telegram.MessageOptions{
    SelfDestruct: 10 * time.Second,
}, func(messageID int, err error) {
    log.Infow("Sent", "message_id", messageID)
})

// Check queue size
queueSize := queue.GetQueueSize()
```

### 4. Input Validation

**Validate user input:**

```go
// Validate amount
amountValidator := telegram.ValidateAmount(10.0, 10000.0)
result := amountValidator("1500")
if !result.IsValid() {
    return fmt.Errorf(result.FirstError())
}

// Chain validators
validator := telegram.ChainValidators(
    telegram.ValidateLength(3, 50),
    telegram.ValidateEmail(),
)
result = validator(userInput)

// Custom validator
customValidator := func(input string) telegram.ValidationResult {
    if strings.Contains(input, "bad") {
        return telegram.ValidationResult{
            Valid: false,
            Errors: []telegram.ValidationError{
                {Field: "input", Message: "Input contains forbidden word"},
            },
        }
    }
    return telegram.ValidationResult{Valid: true}
}
```

### 5. Keyboard Helpers

**Build common keyboard layouts:**

```go
// Yes/No keyboard
keyboard := telegram.YesNoKeyboard("confirm:yes", "confirm:no")

// Confirm/Cancel
keyboard = telegram.ConfirmCancelKeyboard("do_action", "cancel")

// Pagination
keyboard = telegram.PaginationKeyboard(
    telegram.PaginationInfo{
        CurrentPage: 2,
        TotalPages:  5,
        PageSize:    10,
        TotalItems:  47,
    },
    "list", // callback prefix
)

// Manual keyboard with helpers
keyboard = tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("Submit", "submit"),
    ),
    tgbotapi.NewInlineKeyboardRow(
        telegram.BackButton(),
        telegram.CancelButton(),
    ),
)
```

### 6. Text Formatting

**Format messages:**

```go
text := fmt.Sprintf(
    "%s Investment Summary\n\n"+
    "Amount: %s\n"+
    "Risk: %s\n\n"+
    "Confirm? %s",
    telegram.Bold("📊"),
    telegram.Code("$1,000"),
    telegram.Italic("Moderate"),
    telegram.Link("Terms", "https://example.com/terms"),
)

// Escape user input
safeText := telegram.Escape(userProvidedText)
```

### 7. Menu Navigator (DRY Screen Builder)

**Build multi-screen menus declaratively:**

```go
// Create menu navigator
menuNav := telegram.NewMenuNavigator(sessionService, bot, templates, log, 30*time.Minute)

// Option-based screen (radio buttons)
marketTypeScreen := menuNav.BuildOptionScreen(telegram.OptionScreenConfig{
    ID:           "select_market",
    Template:     "invest/select_market_type",
    NextScreenID: "select_risk",
    ParamKey:     "market_type",
    Options: func(ctx context.Context, session telegram.Session) ([]telegram.MenuOption, error) {
        return []telegram.MenuOption{
            MarketOption{Value: "spot", Label: "Spot Trading", Emoji: "📊"},
            MarketOption{Value: "futures", Label: "Futures", Emoji: "⚡"},
        }, nil
    },
})

// Dynamic list screen (from database)
exchangeScreen := menuNav.BuildListScreen(telegram.ListScreenConfig{
    ID:           "select_exchange",
    Template:     "invest/select_exchange",
    NextScreenID: "select_market",
    ParamKey:     "exchange_id",
    ItemsKey:     "Exchanges",
    Items: func(ctx context.Context, session telegram.Session) ([]telegram.ListItem, error) {
        accounts, _ := exchangeService.GetUserAccounts(ctx, userID)
        var items []telegram.ListItem
        for _, acc := range accounts {
            items = append(items, telegram.ListItem{
                ID:         acc.ID.String(),
                ButtonText: fmt.Sprintf("📊 %s", acc.Label),
                TemplateData: map[string]interface{}{
                    "Exchange": acc.Exchange,
                    "Label":    acc.Label,
                },
            })
        }
        return items, nil
    },
})

// Text input screen
amountScreen := menuNav.BuildTextInputScreen("enter_amount", "invest/enter_amount", nil)

// Start menu
screens := map[string]*telegram.Screen{
    "select_exchange": exchangeScreen,
    "select_market":   marketTypeScreen,
    "enter_amount":    amountScreen,
}

menuNav.StartMenu(ctx, telegramID, exchangeScreen, map[string]interface{}{
    "user_id": userID.String(),
})

// Handle callbacks (automatic routing!)
menuNav.HandleCallback(ctx, telegramID, messageID, callbackData, screens)
```

**Key features:**

- Auto-generates keyboards from options/items
- Auto-saves parameters to session
- Auto-adds "Back" button with navigation stack
- Supports custom keyboard builders
- Template data auto-populated

### 8. Menu Registry (Auto-Routing)

**Register menu handlers once, route automatically:**

```go
registry := telegram.NewMenuRegistry(log)

// Register invest menu (handles screens: sel, mkt, risk, amt)
registry.Register(investMenuService)

// Register settings menu (handles screens: main, profile, limits)
registry.Register(settingsMenuService)

// Auto-route callbacks
registry.RouteCallback(ctx, userID, telegramID, messageID, "sel:a=123")  // → investMenuService
registry.RouteCallback(ctx, userID, telegramID, messageID, "main")       // → settingsMenuService

// Auto-route text messages (for amount input, etc.)
routed, _ := registry.RouteTextMessage(ctx, userID, telegramID, "1000")
```

### 9. Session Management

**In-memory sessions (for development/testing):**

```go
sessionService := telegram.NewInMemorySessionService()

// Create session
session, _ := sessionService.CreateSession(ctx, telegramID, "initial_screen", data, 30*time.Minute)

// Store data
session.SetData("amount", 1000.0)
session.SetData("risk", "moderate")

// Retrieve data
amount, ok := session.GetData("amount")
risk, ok := session.GetString("risk")

// Navigation stack
session.PushScreen("screen2")
session.PushScreen("screen3")
prevScreen, ok := session.PopScreen() // → "screen3"
```

**Custom storage (Redis, PostgreSQL, etc.):**

```go
// Implement telegram.SessionService interface
type RedisSessionService struct {
    client *redis.Client
}

func (s *RedisSessionService) GetSession(ctx context.Context, telegramID int64) (telegram.Session, error) {
    // Your Redis implementation
}

func (s *RedisSessionService) SaveSession(ctx context.Context, session telegram.Session, ttl time.Duration) error {
    // Your Redis implementation
}

// Use your implementation
sessionService := &RedisSessionService{client: redisClient}
menuNav := telegram.NewMenuNavigator(sessionService, bot, templates, log, 30*time.Minute)
```

### 10. Middleware

**Custom middleware:**

```go
// Metrics middleware
metricsMiddleware := telegram.MetricsMiddleware(func(command string, success bool, duration time.Duration) {
    metrics.RecordCommand(command, success, duration)
})

// Admin-only middleware
adminMiddleware := telegram.AdminOnlyMiddleware(func(user interface{}) bool {
    u := user.(*user.User)
    return u.IsAdmin
})

// Maintenance mode
maintenanceMiddleware := telegram.MaintenanceModeMiddleware(
    func() bool { return config.MaintenanceMode },
    []int64{123456789}, // Allowed admin IDs
)

registry.Use(metricsMiddleware)
registry.Use(maintenanceMiddleware)
```

## Integration with Your Application

### Implement Bot Interface

Your bot adapter should implement `telegram.Bot` interface:

```go
type MyBot struct {
    api   *tgbotapi.BotAPI
    queue *telegram.AsyncMessageQueue
    log   *logger.Logger
}

func (b *MyBot) SendMessage(chatID int64, text string) error {
    msg := tgbotapi.NewMessage(chatID, text)
    _, err := b.api.Send(msg)
    return err
}

func (b *MyBot) SendMessageWithOptions(chatID int64, text string, opts telegram.MessageOptions) (int, error) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = opts.ParseMode
    msg.DisableWebPagePreview = opts.DisableWebPagePreview
    msg.DisableNotification = opts.DisableNotification

    if opts.Keyboard != nil {
        msg.ReplyMarkup = *opts.Keyboard
    }

    sentMsg, err := b.api.Send(msg)
    if err != nil {
        return 0, err
    }

    // Handle self-destruct
    if opts.SelfDestruct > 0 {
        b.scheduleSelfDestruct(chatID, sentMsg.MessageID, opts.SelfDestruct)
    }

    // Call OnSent callback
    if opts.OnSent != nil {
        go opts.OnSent(sentMsg.MessageID, nil)
    }

    return sentMsg.MessageID, nil
}

// ... implement other methods ...
```

### Bootstrap in main.go

```go
// Create bot
bot := telegram_adapter.NewBot(apiToken, log)

// Create command registry
cmdRegistry := telegram.NewCommandRegistry(bot, log)

// Add middleware
cmdRegistry.Use(telegram.LoggingMiddleware(log))
cmdRegistry.Use(telegram.RecoveryMiddleware(log))
cmdRegistry.Use(telegram.RateLimitMiddleware(30, log))

// Register commands
registerCommands(cmdRegistry, services)

// Create async queue
asyncQueue := telegram.NewAsyncMessageQueue(bot, 10, 50*time.Millisecond, log)
asyncQueue.Start()

// In your message handler
if msg.IsCommand() {
    return cmdRegistry.Handle(ctx, user, telegramID, chatID, msg.Command(), msg.CommandArguments(), msg.Text)
}
```

## Best Practices

1. **Use middleware for cross-cutting concerns**: Logging, metrics, rate limiting, auth
2. **Validate all user input**: Use built-in validators or create custom ones
3. **Handle errors gracefully**: Return user-friendly error messages
4. **Use async queue for bulk notifications**: Prevents rate limiting issues
5. **Self-destruct sensitive messages**: API keys, passwords, temp codes
6. **Category commands**: Group related commands for better `/help` output
7. **Provide command aliases**: Short versions like `/i` for `/invest`
8. **Test with mock implementations**: All interfaces are mockable

## Performance Tips

- Use `AsyncMessageQueue` for sending multiple messages
- Set appropriate `rateLimitDelay` (Telegram limits: 30 msgs/sec to different users)
- Buffer size for queue: `1000` is usually sufficient
- Workers count: `5-10` workers for most use cases
- Use `Silent()` for non-urgent notifications
- Cache user data to avoid repeated DB lookups

## Thread Safety

- `CommandRegistry` is thread-safe after initialization
- `AsyncMessageQueue` is fully thread-safe
- `MessageBuilder` is NOT thread-safe (use per-request)
- `Session` implementations should be thread-safe

## Summary

**Что мы сделали:**

✅ Полностью самодостаточный пакет `pkg/telegram` - не требует ничего кроме logger  
✅ Работает через интерфейсы - никакой зависимости от `tgbotapi` напрямую  
✅ Embedded templates - всё работает из коробки  
✅ DRY фреймворк для меню - строишь экраны декларативно  
✅ Command registry с middleware - как в web-фреймворках  
✅ Async queue - не блокирует при отправке  
✅ Session management - in-memory или свой адаптер  
✅ Auto-routing - регистрируешь handlers один раз

**Как использовать в проекте:**

1. В `internal/adapters/telegram` реализуй `telegram.Bot` интерфейс
2. Создай `telegram.SessionService` адаптер (Redis/PostgreSQL)
3. Регистрируй команды и меню
4. Profit! 🚀

**Migration path:**

Существующий код в `internal/adapters/telegram/` можно постепенно мигрировать:

- `menu_nav.go` → использовать `telegram.MenuNavigator` из пакета
- `menu_registry.go` → использовать `telegram.MenuRegistry` из пакета
- `handlers.go` → использовать `telegram.CommandRegistry` из пакета

Все старые экраны останутся работать, просто меняешь импорты и типы!

## License

Internal package for Prometheus trading system.
