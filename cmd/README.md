<!-- @format -->

# cmd/ — Application Entrypoint

**Post-Bootstrap Refactoring** ✅

This directory contains the main entrypoint (`main.go`) for the Prometheus trading system.

---

## New Architecture

The application has been **refactored from 1500+ lines to 58 lines** by moving all initialization logic to `internal/bootstrap`.

### Before (Old Approach)

```go
func main() {
    // 1500+ lines of:
    // - Infrastructure initialization
    // - Repository creation
    // - Service wiring
    // - Worker setup
    // - Consumer management
    // - Complex shutdown logic
}
```

**Problems:**

- ❌ Hard to test
- ❌ Difficult to understand flow
- ❌ No clear separation of concerns
- ❌ Main function doing too much

### After (Bootstrap Approach)

```go
func main() {
    // 1. Create container
    container := bootstrap.NewContainer()

    // 2. Initialize all phases
    container.MustInit()

    // 3. Start background systems
    container.Start()

    // 4. Wait for signal & shutdown
    <-quit
    container.Shutdown()
}
```

**Improvements:**

- ✅ **58 lines** instead of 1500
- ✅ Clear, linear flow
- ✅ Easy to test each phase
- ✅ Full shutdown control
- ✅ Organized by concern

---

## Bootstrap Package Structure

All initialization logic moved to `internal/bootstrap/`:

```
internal/bootstrap/
├── container.go       # Dependency container
├── lifecycle.go       # Graceful shutdown (8 steps)
├── providers.go       # All provider functions
├── workers.go         # Worker initialization
└── README.md          # Detailed documentation
```

### Initialization Phases

The container initializes components in **8 explicit phases**:

1. **Config & Logging**

   - Load environment variables
   - Initialize structured logger
   - Set up error tracker (Sentry)

2. **Infrastructure**

   - PostgreSQL connection
   - ClickHouse connection
   - Redis connection

3. **Repositories**

   - 17 domain repositories
   - Postgres: users, orders, positions
   - ClickHouse: market data, analytics

4. **Adapters**

   - Kafka (producer + 7 consumers)
   - Exchange factories
   - Embedding provider
   - Encryption

5. **Services**

   - Domain services (user, position, etc.)
   - ADK session service

6. **Business Logic**

   - Risk engine
   - Tool registry (30+ tools)
   - Agent factory (9 agents)
   - Workflow factory

7. **Application**

   - HTTP server
   - Telegram bot
   - Notification services

8. **Background**
   - Worker scheduler (30+ workers)
   - Event consumers
   - Position guardian

### Graceful Shutdown Sequence

**Critical for trading systems** — shutdown is **explicit and ordered**:

```
1. Stop HTTP Server       [5s timeout]
2. Stop Workers           [30s timeout]
3. Close Kafka Consumers  [immediate]
4. Wait for Goroutines    [5s timeout]
5. Close Kafka Producer   [immediate]
6. Flush Error Tracker    [3s timeout]
7. Sync Logs              [immediate]
8. Close Databases        [immediate]
```

Each step is logged: `"[3/8] Closing Kafka consumers..."` for observability.

---

## Running the Application

### Development

```bash
# Run with default config
make run

# Run with custom env
ENV=staging make run

# Run with debug logging
LOG_LEVEL=debug make run
```

### Production

```bash
# Build binary
make build

# Run binary
./bin/prometheus
```

### Docker

```bash
# Build Docker image
docker build -t prometheus:latest .

# Run container
docker run --env-file .env prometheus:latest
```

---

## Configuration

All configuration loaded from environment variables via `internal/adapters/config`.

See `docs/ENV_SETUP.md` for full list of variables.

### Required Variables

```bash
# Application
APP_NAME=prometheus
APP_ENV=production

# Databases
POSTGRES_DSN=postgres://...
CLICKHOUSE_DSN=tcp://...
REDIS_URL=redis://...

# Kafka
KAFKA_BROKERS=localhost:9092

# AI Providers
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Telegram
TELEGRAM_BOT_TOKEN=...
```

---

## Signals & Lifecycle

### Startup

```
2024-11-28 10:00:00 INFO Starting Prometheus in production mode
2024-11-28 10:00:01 INFO ✓ PostgreSQL connected
2024-11-28 10:00:01 INFO ✓ ClickHouse connected
2024-11-28 10:00:01 INFO ✓ Redis connected
2024-11-28 10:00:02 INFO ✓ Repositories initialized
...
2024-11-28 10:00:05 INFO ✓ All systems operational
```

### Shutdown (SIGINT/SIGTERM)

```
2024-11-28 15:30:00 INFO 📡 Received signal: interrupt
2024-11-28 15:30:00 INFO [1/8] Stopping HTTP server...
2024-11-28 15:30:00 INFO ✓ HTTP server stopped
2024-11-28 15:30:01 INFO [2/8] Stopping workers...
2024-11-28 15:30:01 INFO ✓ Workers stopped
...
2024-11-28 15:30:05 INFO [8/8] Closing databases...
2024-11-28 15:30:05 INFO ✓ Database connections closed
2024-11-28 15:30:05 INFO ✅ Graceful shutdown complete
2024-11-28 15:30:05 INFO 👋 Shutdown complete. Goodbye!
```

---

## Testing

### Unit Tests

```bash
# Test container initialization
go test ./internal/bootstrap/... -v

# Test specific phase
go test ./internal/bootstrap/... -run TestMustInitInfrastructure
```

### Integration Tests

```bash
# Full lifecycle test
go test ./cmd/... -tags=integration -v
```

---

## Advantages vs Uber FX

| Aspect               | Bootstrap (Our Approach)    | Uber FX               |
| -------------------- | --------------------------- | --------------------- |
| **Lines of Code**    | 58 in main.go               | ~100+                 |
| **Shutdown Control** | ✅ Explicit 8-step sequence | ⚠️ Graph-based        |
| **Debugging**        | ✅ Linear, easy to trace    | ⚠️ Reflection magic   |
| **Observability**    | ✅ Step-by-step logs        | ⚠️ Less visible       |
| **Testability**      | ✅ Easy to mock phases      | ✅ Good               |
| **Complexity**       | ✅ ~500 lines total         | ⚠️ Framework overhead |
| **Control**          | ✅ **Full control**         | ⚠️ Framework dictates |

**For a hedge fund trading system**, explicit control and predictability are more important than framework magic.

---

## Troubleshooting

### Startup Failures

**Problem:** Application crashes on startup

**Solution:**

1. Check config: `make check-config`
2. Check logs: Look for first error
3. Check dependencies: `make deps`
4. Check services: `make docker-up`

### Shutdown Hangs

**Problem:** Application doesn't exit cleanly

**Solution:**

1. Check logs for which step is hanging: `"[3/8] Closing Kafka..."`
2. Increase timeout in `lifecycle.go` if needed
3. Check for goroutine leaks with `pprof`

### Memory Leaks

**Problem:** Memory grows unbounded

**Solution:**

1. Check worker cleanup in shutdown
2. Verify Kafka consumers are closing
3. Check database connection pools
4. Use `pprof` for heap analysis

---

## Adding New Components

See `internal/bootstrap/README.md` for detailed guide.

**Quick Start:**

1. Add to `Container` struct
2. Create `provide*()` function
3. Call in appropriate `MustInit*()` phase
4. Add cleanup to `Shutdown()` if needed

---

## Migration Notes

This refactoring was completed on **Nov 28, 2024**.

**Changes:**

- ✅ All provider functions moved to `internal/bootstrap/providers.go`
- ✅ Lifecycle management moved to `internal/bootstrap/lifecycle.go`
- ✅ Worker setup moved to `internal/bootstrap/workers.go`
- ✅ Main.go reduced from 1500 lines → 58 lines
- ✅ Zero functional changes (behavior identical)
- ✅ All tests passing
- ✅ Build successful

**Old Code:**

- Preserved in git history
- Reference: commit before refactoring
- Can rollback if needed (unlikely)

---

## References

- **Bootstrap Package**: `internal/bootstrap/README.md`
- **Environment Setup**: `docs/ENV_SETUP.md`
- **Architecture**: `docs/AGENT_ARCHITECTURE.md`
- **Development Plan**: `docs/DEVELOPMENT_PLAN.md`
- **Specs**: `docs/specs.md`

---

**Last Updated:** Nov 28, 2024  
**Author:** System refactoring with AI assistance  
**Status:** ✅ Production-ready
