<!-- @format -->

# cmd/ — Application Entrypoint

This directory contains the main entrypoint (`main.go`) for the Prometheus trading system. Follow these architectural patterns strictly.

---

## Core Principles

### 1. Dependency Injection via `provide*()` Functions

**RULE:** Each infrastructure component MUST have its own `provide*()` function.

**❌ WRONG:**

```go
func initDatabases() (*Database, error) {
    pg := connectPostgres()
    ch := connectClickHouse()
    redis := connectRedis()
    return &Database{pg, ch, redis}, nil
}
```

**✅ CORRECT:**

```go
func providePostgres(cfg *config.Config, log *logger.Logger) (*pgclient.Client, error) {
    // Single responsibility: only Postgres
}

func provideClickHouse(cfg *config.Config, log *logger.Logger) (*chclient.Client, error) {
    // Single responsibility: only ClickHouse
}

func provideRedis(cfg *config.Config, log *logger.Logger) (*redisclient.Client, error) {
    // Single responsibility: only Redis
}

// In main():
pg, err := providePostgres(cfg, log)
ch, err := provideClickHouse(cfg, log)
redis, err := provideRedis(cfg, log)
```

**WHY:**

- Each component can be tested independently
- Clear dependency graph in `main()`
- Easy to mock/replace individual components
- No hidden coupling between unrelated systems

---

## 2. Graceful Shutdown Architecture

### Context Cancellation Flow

**CRITICAL:** Understand the difference between signaling and forcing termination.

```go
// Create application context
ctx, cancel := context.WithCancel(context.Background())

// Start all components with this context
workerScheduler.Start(ctx)
marketScanner.Start(ctx)
httpServer (uses ctx indirectly)

// Wait for OS signal
<-quit

// IMMEDIATELY cancel context - this SIGNALS all components to stop
cancel() // ← This does NOT kill anything! It's a polite "please stop" signal

// Now WAIT for graceful completion with timeouts
gracefulShutdown(...)
```

### What `cancel()` Does

When you call `cancel()`:

- `ctx.Done()` channel is closed
- All components listening to `ctx` receive the signal
- Components BEGIN their shutdown procedure
- They DO NOT terminate immediately

### Example: Worker Graceful Shutdown

```go
func (w *Worker) Run(ctx context.Context) error {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            // ✅ Received shutdown signal
            w.saveState()        // Finish current work
            w.closeConnections() // Clean up resources
            return nil           // Exit gracefully

        case <-ticker.C:
            w.doWork() // Normal operation
        }
    }
}
```

**The worker will:**

1. Finish its current iteration of `doWork()`
2. See `ctx.Done()` in the next select
3. Perform cleanup
4. Return gracefully

**The worker will NOT:**

- Be forcefully killed mid-operation
- Leave incomplete database transactions
- Leak resources

---

## 3. Graceful Shutdown Sequence

```go
// 1. Receive OS signal
sig := <-quit
log.Info("Shutdown signal received")

// 2. Cancel application context (SIGNAL to stop)
cancel()

// 3. Wait for components to finish with timeouts
gracefulShutdown(wg, httpServer, workers, kafka, db, log)
```

### Shutdown Order (IMPORTANT!)

```go
[1/7] HTTP Server (5s timeout)
    - Stop accepting new connections
    - Finish processing current requests
    - Return 503 for new requests

[2/7] Background Workers (30s timeout)
    - Workers see ctx.Done()
    - Complete current iteration
    - Exit their goroutines
    - scheduler.Stop() waits for wg.Wait()

[3/7] Auxiliary Goroutines (5s timeout)
    - Market scanner event listener
    - Other background tasks
    - wg.Wait() for completion

[4/7] Kafka Clients
    - Flush producer buffers
    - Close consumer connections

[5/7] Error Tracker (3s timeout)
    - Flush pending errors to Sentry

[6/7] Logger Sync
    - Write buffered logs to disk

[7/7] Database Connections (LAST!)
    - Close PostgreSQL pool
    - Close ClickHouse connection
    - Close Redis client
    - Why last? Other components may need DB during shutdown
```

---

## 4. sync.WaitGroup for Goroutine Tracking

**RULE:** Every long-running goroutine MUST be tracked via WaitGroup.

```go
var wg sync.WaitGroup

// Start HTTP server
wg.Add(1)
go func() {
    defer wg.Done()
    httpServer.ListenAndServe()
}()

// Start event listener
if cfg.EventDrivenMode {
    wg.Add(1)
    go func() {
        defer wg.Done()
        scanner.Start(ctx) // Uses application context!
    }()
}

// In shutdown:
wg.Wait() // Ensures all goroutines finished
```

**WHY:**

- Prevents resource leaks (closed DB while goroutine still writing)
- Ensures clean shutdown (all work completed)
- Makes goroutine lifecycle explicit

---

## 5. Common Mistakes to Avoid

### ❌ MISTAKE 1: Wrong Context in Goroutines

```go
// WRONG: Uses context.Background() - ignores shutdown signal
go func() {
    scanner.Start(context.Background())
}()
```

```go
// CORRECT: Uses application context - respects shutdown
go func() {
    scanner.Start(ctx) // Will stop when ctx is cancelled
}()
```

### ❌ MISTAKE 2: No WaitGroup for Goroutines

```go
// WRONG: No tracking - may exit before goroutine finishes
go func() {
    doWork()
}()
// main exits - goroutine killed mid-work
```

```go
// CORRECT: Track with WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    doWork()
}()
// Shutdown waits for wg.Done()
```

### ❌ MISTAKE 3: Aggregated Providers

```go
// WRONG: Returns everything at once
func initAll() (*AllDeps, error) {
    return &AllDeps{DB: db, Redis: redis, Kafka: kafka}, nil
}
```

```go
// CORRECT: Separate providers for each dependency
func provideDB(...) (*DB, error)
func provideRedis(...) (*Redis, error)
func provideKafka(...) (*Kafka, error)
```

### ❌ MISTAKE 4: defer cancel() Before Signal Handler

```go
// WRONG: cancel() called at function exit (too late!)
ctx, cancel := context.WithCancel(...)
defer cancel()

<-quit // Never reached if panic
```

```go
// CORRECT: cancel() called explicitly after signal
ctx, cancel := context.WithCancel(...)

<-quit
cancel() // Explicit control flow
```

---

## 6. Testing Graceful Shutdown

To test shutdown behavior locally:

```bash
# Start the application
./bin/prometheus

# Send SIGINT (Ctrl+C)
# OR from another terminal:
kill -SIGINT <pid>

# Expected log output:
# Received signal: interrupt
# [1/7] Stopping HTTP server...
# ✓ HTTP server stopped
# [2/7] Stopping background workers...
# ✓ Workers stopped
# [3/7] Waiting for auxiliary goroutines...
# ✓ All goroutines finished
# [4/7] Closing Kafka clients...
# ✓ Kafka clients closed
# [5/7] Flushing error tracker...
# ✓ Error tracker flushed
# [6/7] Syncing logs...
# ✓ Logs synced
# [7/7] Closing database connections...
# ✓ Database connections closed
# ✅ Graceful shutdown complete
```

**If you see:**

- `⚠ Some goroutines did not finish within timeout` → Goroutine leak! Check ctx usage
- `Workers shutdown failed` → Worker not respecting ctx.Done()
- Panic during shutdown → Resource closed while still in use

---

## 7. Adding New Components

### Template for New Provider

```go
func provideMyComponent(
    cfg *config.Config,
    log *logger.Logger,
    // ... dependencies
) (*MyComponent, error) {
    log.Info("Initializing MyComponent...")

    component, err := mycomponent.New(cfg.MyComponent)
    if err != nil {
        return nil, errors.Wrap(err, "create mycomponent")
    }

    log.Info("✓ MyComponent initialized")
    return component, nil
}
```

### Add to main()

```go
// In main():
myComponent, err := provideMyComponent(cfg, log, deps...)
if err != nil {
    log.Fatalf("failed to initialize mycomponent: %v", err)
}
```

### Add to Shutdown

```go
// In gracefulShutdown():
log.Info("[N/7] Closing MyComponent...")
if err := myComponent.Close(); err != nil {
    log.Error("MyComponent close failed", "error", err)
} else {
    log.Info("✓ MyComponent closed")
}
```

---

## 8. Quick Reference

### Checklist for New Code

- [ ] Each component has separate `provide*()` function
- [ ] All goroutines tracked with `wg.Add(1)` / `defer wg.Done()`
- [ ] All goroutines use application `ctx`, not `context.Background()`
- [ ] Worker implements proper `ctx.Done()` handling in select
- [ ] Component added to `gracefulShutdown()` in correct order
- [ ] Close/cleanup logic tested with `kill -SIGINT`

### Order of Dependencies

1. **Infrastructure**: Postgres, ClickHouse, Redis (independent)
2. **Repositories**: Depend on databases
3. **Services**: Depend on repositories
4. **External Adapters**: Kafka, Exchange factories (independent)
5. **Business Logic**: Risk engine, tool registry, agents (depend on above)
6. **Workers**: Depend on everything above
7. **HTTP Server**: For health checks / metrics

---

## Questions?

If you're modifying `main.go`:

1. Read `AGENTS.md` in project root
2. Check existing `provide*()` functions for patterns
3. Follow the shutdown sequence strictly
4. Test with `kill -SIGINT` before committing

**Remember:** `cancel()` is a signal, not a kill command. Graceful shutdown means "finish what you're doing, then exit."
