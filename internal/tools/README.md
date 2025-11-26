<!-- @format -->

# Tools

Atomic operations exposed to ADK agents via middleware chain pattern.

## Architecture

Tools follow **middleware chain**: `Retry → Timeout → Stats → Risk → Core Tool`

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

1. Create `<category>/<tool_name>.go` with struct implementing ADK Tool interface
2. Implement `Name()`, `Description()`, `Execute(ctx, params)` methods
3. Validate parameters at start, return structured error on missing/invalid params
4. Use dependencies from `shared.Dependencies` (exchanges, services, repos)
5. Log execution with tool name + key params
6. Register in `registry.go` with appropriate middleware chain
7. Add metadata to `catalog.go` (params, return type, risk level)
8. Assign to agents in `internal/agents/tool_assignments.go`
9. Write unit tests with mocked dependencies

## Core Rules

- **Atomic operations**: One tool = one responsibility
- **Parameter validation**: Check all params upfront, fail fast with clear errors
- **Context handling**: Respect cancellation, pass context to all external calls
- **Structured returns**: Return maps with consistent schema, document in catalog
- **Error wrapping**: Use `pkg/errors.Wrap(err, "context")`
- **Logging**: Use `pkg/logger` with tool name + params (no secrets)
- **No business logic**: Keep in services, tools just orchestrate
- **No state**: Tools are instantiated once, don't store request state

## Middleware Order (Critical!)

Apply from outermost to innermost:
1. **Retry** — retry failed operations with backoff
2. **Timeout** — enforce operation deadline
3. **Stats** — track usage to ClickHouse
4. **Risk** — check circuit breaker before execution
5. **Core** — actual tool implementation

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
