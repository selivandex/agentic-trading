<!-- @format -->

# Tools

Atomic operations exposed to ADK agents via middleware chain pattern.

## Architecture

Tools use **fluent ToolBuilder API** with layered middleware: `Stats → Timeout → Retry → Core Tool`

```go
shared.NewToolBuilder("tool_name", "description", toolFunc, deps).
    WithRetry(3, 500*time.Millisecond).
    WithTimeout(10*time.Second).
    WithStats().
    Build()
```

Each tool is wrapped with middleware for resilience, observability, and safety.

## Structure

```
tools/
├── registry.go         # Central registry with middleware wrapping
├── catalog.go          # Metadata (params, return types, risk levels)
├── middleware/         # Retry, timeout, stats, risk wrappers
├── market/             # Market data (price, orderbook, OHLCV, trades)
├── trading/            # Execution (place/cancel orders, positions, balance)
├── risk/               # Validation (trade validation, circuit breaker, emergency close)
├── memory/             # Knowledge (search memory, store insights)
├── indicators/         # Technical analysis (RSI, EMA, MACD, ATR)
└── shared/             # Deps container, validation helpers
```

## Tool Categories

- **Market**: Fetch market data (low risk, fast)
- **Trading**: Execute trades (high risk, slow)
- **Risk**: Validate operations (critical, fast)
- **Memory**: Search knowledge base (medium risk, medium speed)
- **Indicators**: Calculate technical indicators (low risk, fast)

## Adding New Tool

1. Create `<category>/<tool_name>.go` with NewXXXTool constructor:

```go
func NewMyTool(deps shared.Deps) tool.Tool {
    return shared.NewToolBuilder(
        "my_tool",
        "Tool description",
        func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
            // 1. Validate parameters
            symbol, ok := args["symbol"].(string)
            if !ok || symbol == "" {
                return nil, errors.New("missing required parameter: symbol")
            }

            // 2. Execute business logic using deps
            result, err := deps.MarketDataRepo.GetLatestTicker(ctx, "binance", symbol)
            if err != nil {
                return nil, errors.Wrap(err, "failed to fetch ticker")
            }

            // 3. Return structured result
            return map[string]interface{}{
                "price": result.Price,
                "timestamp": result.Timestamp,
            }, nil
        },
        deps,
    ).
        WithTimeout(10*time.Second).
        WithRetry(3, 500*time.Millisecond).
        WithStats().
        Build()
}
```

2. Register in `register.go`: `registry.Register("my_tool", category.NewMyTool(deps))`
3. Add metadata to `catalog.go` (params, return type, risk level)
4. Assign to agents in `internal/agents/tool_assignments.go`
5. Write unit tests with mocked dependencies

## Core Rules

- **Atomic operations**: One tool = one responsibility
- **Parameter validation**: Check all params upfront, fail fast with clear errors
- **Context handling**: Respect cancellation, pass context to all external calls
- **Structured returns**: Return maps with consistent schema, document in catalog
- **Error wrapping**: Use `pkg/errors.Wrap(err, "context")`
- **Logging**: Use `pkg/logger` with tool name + params (no secrets)
- **No business logic**: Keep in services, tools just orchestrate
- **No state**: Tools are instantiated once, don't store request state

## Middleware Order (Automatic via ToolBuilder!)

ToolBuilder applies middleware in correct order automatically:

1. **Timeout** (outer) — enforces deadline on tool + retries
2. **Retry** (inner) — retries core tool on failure
3. **Core** — actual tool implementation

**Stats tracking** is handled by ADK callbacks (see `internal/agents/callbacks/tool.go`), not middleware.

Configure via fluent API:

```go
.WithRetry(attempts, backoff)  // Enable retry with N attempts
.WithTimeout(duration)          // Enable timeout
```

Never bypass middleware unless explicitly required for specific tool.

## Parameter Validation

Use `shared.ValidateParams(params, []string{"exchange", "symbol"})` helper. Return clear error messages for missing/invalid params.

## Testing

- **Unit**: Mock dependencies via interfaces, test with various param combinations
- **Integration**: Test with real adapters in sandbox mode
- Table-driven tests with valid/invalid cases

## Timeouts by Category

- Market data: 10s
- Trading: 30s
- Risk validation: 5s
- Memory search: 15s
- Indicators: 5s

Adjust in middleware wrapping based on tool type.

## Observability

Stats middleware automatically tracks to ClickHouse:

- Execution count
- Success/failure rate
- Latency percentiles
- Error types

Log at tool level with structured fields (tool, exchange, symbol, user_id).

## Error Handling

- **Validation errors**: Return `errors.New("missing required parameter: X")`
- **External errors**: Wrap with `errors.Wrap(err, "failed to fetch from exchange")`
- **Risk errors**: Include metadata with `errors.WithMetadata(err, map[...])`

Never return raw errors from stdlib/external libs.

## Best Practices

**DO:**

- Validate params first
- Handle context cancellation
- Return consistent data structures
- Log meaningful events
- Keep tools pure and deterministic where possible
- Use dependency injection

**DON'T:**

- Store state in tool struct
- Implement complex business logic
- Make calls without timeouts
- Ignore middleware
- Log sensitive data
- Bypass risk validation

## Performance

- Cache expensive calculations in Redis with TTL
- Batch similar requests when possible
- Respect exchange rate limits
- Use connection pooling from adapters

## References

- Middleware: `internal/tools/middleware/`
- Dependencies: `internal/tools/shared/deps.go`
- Risk engine: `internal/risk/engine.go`
- Agent assignments: `internal/agents/tool_assignments.go`
