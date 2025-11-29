<!-- @format -->

# Bootstrap Package

**Purpose**: Organized dependency injection and application lifecycle management for Prometheus trading system.

## Overview

The `bootstrap` package provides a clean, maintainable way to initialize and manage the application lifecycle without the complexity of frameworks like Uber FX. This approach gives us **explicit control** over initialization order and shutdown sequence—critical for a trading system.

## Architecture

```
bootstrap/
├── container.go       # Dependency container with all components
├── lifecycle.go       # Graceful shutdown manager
├── providers.go       # Provider functions for all dependencies
└── workers.go         # Background workers initialization
```

## Key Components

### Container

Central dependency injection container organized in logical groups:

```go
type Container struct {
    // Core
    Config       *config.Config
    Log          *logger.Logger
    ErrorTracker errors.Tracker

    // Infrastructure
    PG    *pgclient.Client
    CH    *chclient.Client
    Redis *redisclient.Client

    // Domain
    Repos    *Repositories
    Services *Services

    // Business
    Business *Business

    // Application
    Application *Application

    // Background
    Background *Background

    // Lifecycle
    Lifecycle *Lifecycle
    WG        *sync.WaitGroup
    Context   context.Context
    Cancel    context.CancelFunc
}
```

### Initialization Phases

Components are initialized in **8 explicit phases** for clarity and correctness:

1. **Config & Logging** (`MustInitConfig`)

   - Load configuration from environment
   - Initialize structured logger
   - Set up error tracker (Sentry)

2. **Infrastructure** (`MustInitInfrastructure`)

   - PostgreSQL connection
   - ClickHouse connection
   - Redis connection

3. **Repositories** (`MustInitRepositories`)

   - All domain repositories (17 total)
   - Postgres: users, orders, positions, sessions, etc.
   - ClickHouse: market data, analytics, telemetry

4. **Adapters** (`MustInitAdapters`)

   - Kafka producer & consumers (7 topics)
   - Exchange factories (Binance, Bybit, OKX)
   - Embedding provider (OpenAI)
   - Encryption utilities

5. **Services** (`MustInitServices`)

   - Domain services for business logic
   - ADK session service adapter

6. **Business Logic** (`MustInitBusiness`)

   - Risk engine
   - Tool registry (30+ tools)
   - Agent factory & registry
   - Workflow factory

7. **Application** (`MustInitApplication`)

   - HTTP server (health checks, metrics)
   - Telegram bot & webhook
   - Notification services

8. **Background** (`MustInitBackground`)
   - Worker scheduler (30+ workers)
   - Event consumers (8 topics)
   - Position guardian (algorithmic + LLM)

### Lifecycle Management

**Critical for hedge fund systems**: The shutdown sequence is **explicit and ordered** to prevent data loss or corruption.

```go
// 8-step graceful shutdown (order matters!)
1. Stop HTTP Server        → No new requests
2. Stop Workers             → Finish current jobs
3. Close Kafka Consumers    → Unblock ReadMessage()
4. Wait for Goroutines      → Let them finish cleanly
5. Close Kafka Producer     → After consumers done
6. Flush Error Tracker      → Send pending errors to Sentry
7. Sync Logs                → Flush all log entries
8. Close Databases          → Last (components may need them)
```

Each step has logging and timeouts for observability.

## Usage

### main.go (Simple & Clean)

```go
func main() {
    // 1. Create container
    container := bootstrap.NewContainer()

    // 2. Initialize all phases
    container.MustInit()

    // 3. Start background systems
    if err := container.Start(); err != nil {
        log.Fatalf("Failed to start: %v", err)
    }

    // 4. Wait for signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // 5. Graceful shutdown
    container.Shutdown()
}
```

**58 lines** instead of 1500! All complexity moved to organized packages.

### Adding New Components

#### 1. Add to Container

```go
// container.go
type Business struct {
    RiskEngine   *riskservice.RiskEngine
    NewComponent *MyComponent  // ← Add here
}
```

#### 2. Create Provider

```go
// providers.go
func (c *Container) MustInitBusiness() {
    // ... existing components ...

    c.Business.NewComponent = provideNewComponent(c.Config, c.Log)
}

func provideNewComponent(cfg *config.Config, log *logger.Logger) *MyComponent {
    comp, err := NewMyComponent(cfg.MyComponentConfig)
    if err != nil {
        log.Fatalf("failed to init my component: %v", err)
    }
    log.Info("✓ My component initialized")
    return comp
}
```

#### 3. Add to Shutdown (if needed)

```go
// lifecycle.go - only if component needs graceful cleanup
func (l *Lifecycle) Shutdown(...) {
    // Step 9: Close My Component
    if myComponent != nil {
        myComponent.Close()
    }
}
```

## Advantages Over Uber FX

| Aspect             | Bootstrap                      | Uber FX                              |
| ------------------ | ------------------------------ | ------------------------------------ |
| **Shutdown Order** | ✅ Explicit 8-step sequence    | ⚠️ Graph-based (harder to guarantee) |
| **Debugging**      | ✅ Linear flow, easy to trace  | ⚠️ Reflection-based, harder to debug |
| **Observability**  | ✅ Step-by-step logs (`[3/8]`) | ⚠️ Less visibility                   |
| **Testability**    | ✅ Easy to mock phases         | ✅ Good with modules                 |
| **Complexity**     | ✅ ~500 lines, readable        | ⚠️ Learning curve + magic            |
| **Control**        | ✅ **Full control**            | ⚠️ Framework dictates flow           |
| **Performance**    | ✅ No reflection overhead      | ⚠️ Reflection at startup             |

For a **hedge fund trading system**, explicit control and predictability are more important than framework elegance.

## Testing Strategy

### Unit Tests

```go
func TestContainerInit(t *testing.T) {
    container := bootstrap.NewContainer()
    container.MustInitConfig()

    assert.NotNil(t, container.Config)
    assert.NotNil(t, container.Log)
}
```

### Integration Tests

```go
func TestFullLifecycle(t *testing.T) {
    container := bootstrap.NewContainer()
    container.MustInit()

    // Start & immediately shutdown
    go container.Start()
    time.Sleep(100 * time.Millisecond)
    container.Shutdown()

    // Verify clean shutdown
    assert.Eventually(t, func() bool {
        return container.Context.Err() != nil
    }, 5*time.Second, 100*time.Millisecond)
}
```

## Metrics

Access system metrics via container:

```go
metrics := container.GetMetrics()
// map[string]interface{}{
//     "tools": 30,
//     "agents": 9,
// }
```

## Troubleshooting

### Startup Issues

- Check logs: `"Starting Prometheus in production mode"`
- Each phase logs completion: `"✓ PostgreSQL connected"`
- Fatal errors panic with context: `"failed to connect postgres: ..."`

### Shutdown Hangs

- Watch step logs: `"[3/8] Closing Kafka consumers..."`
- Check WaitGroup timeout warnings: `"⚠ Some goroutines did not finish"`
- Increase shutdown timeout in `lifecycle.go` if needed (default 150s)

## Future Improvements

1. **Health Checks**: Add `container.HealthCheck()` method
2. **Metrics**: Export initialization times per phase
3. **Hot Reload**: Support config reloading without restart
4. **Graceful Drain**: HTTP connection draining before shutdown

## References

- Original implementation: `cmd/main.go` (1500 lines → 58 lines)
- Clean Architecture: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
- Go DI patterns: https://github.com/google/wire
