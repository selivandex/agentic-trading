<!-- @format -->

# Adapters

Infrastructure clients for external services (databases, exchanges, AI providers, Kafka, Redis).

## Core Rules

- **Interface-first**: Define interface in package root, implement in subpackages
- **Factory pattern**: Use factory functions for initialization with config validation
- **Context-aware**: All methods accept `context.Context`, handle cancellation
- **Error wrapping**: Use `pkg/errors.Wrap(err, "context")`, never ignore errors
- **Structured logging**: Use `pkg/logger` with fields (exchange, user_id, agent), redact secrets
- **Resource cleanup**: Implement `Close()` method, cleanup in defer
- **Retries**: Exponential backoff for network calls, respect context cancellation
- **Timeouts**: Set appropriate timeouts for all external requests

## Structure

```
adapters/
├── exchanges/      # Exchange APIs (binance, bybit, okx)
├── ai/             # AI providers (OpenAI, Claude, Gemini)
├── postgres/       # Postgres client
├── redis/          # Redis with distributed locks
├── clickhouse/     # ClickHouse for analytics
├── kafka/          # Kafka producer/consumer
└── telegram/       # Telegram bot
```

## Adding New Adapter

1. Create interface in `<adapter>/interface.go` with methods returning errors
2. Implement client in `<adapter>/client.go` with struct holding config + logger
3. Constructor `NewClient(cfg Config, log *logger.Logger) (Interface, error)` validates config
4. Wire in `cmd/main.go` via `provide<Service>()` helper
5. Add integration tests with mocked responses or testcontainers
6. Update `docs/ENV_SETUP.md` with required env vars

## Best Practices

**DO:**

- Return interfaces from constructors
- Validate config in constructor, fail fast
- Use structured logging with context fields
- Implement retries at adapter level, not in business logic
- Test with mocked HTTP/WebSocket clients
- Handle rate limits gracefully

**DON'T:**

- Store business logic in adapters
- Use global state or singletons
- Ignore context cancellation
- Log secrets (API keys, passwords)
- Panic on errors
- Make synchronous calls without timeouts

## Testing

- **Unit**: Mock interfaces, test logic in isolation
- **Integration**: Use `internal/testsupport` helpers for real services
- Run with `go test -short` for unit only, `go test` for all

## Common Patterns

- **Retry**: Wrap calls in backoff loop with context cancellation check
- **Circuit breaker**: See `internal/risk/engine.go` for integration
- **Connection pooling**: Let underlying libs handle it (http.Client, sql.DB)
- **Graceful shutdown**: Implement `Close()`, call from main on signal

## References

- Exchange factory: `internal/adapters/exchanges/factory.go`
- AI registry: `internal/adapters/ai/registry.go`
- Config loading: `internal/adapters/config/config.go`
