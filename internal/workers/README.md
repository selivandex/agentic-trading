<!-- @format -->

# Workers

Background workers for scheduled tasks (market data collection, analysis, trading operations).

## Architecture

Workers run on intervals via `Scheduler`. Each worker implements `Worker` interface with `Start()`, `Stop()`, graceful shutdown.

## Structure

```
workers/
├── scheduler.go           # Main scheduler orchestrating all workers
├── worker.go              # Worker interface
├── marketdata/            # Market data collection (tickers, OHLCV)
├── analysis/              # Market scanning, regime detection, opportunities
├── trading/               # Order sync, PnL calculation, position management
├── evaluation/            # Performance metrics, strategy evaluation
└── sentiment/             # News collection, sentiment analysis
```

## Worker Categories

| Category | Purpose | Interval |
|----------|---------|----------|
| **Market Data** | Collect tickers, OHLCV | 1-5 min |
| **Analysis** | Scan markets, detect regimes | 5-15 min |
| **Trading** | Sync orders, calculate PnL | 1-5 min |
| **Evaluation** | Calculate performance metrics | 1 hour |
| **Sentiment** | Collect news, analyze sentiment | 15 min |

## Adding New Worker

1. Create `<category>/<worker_name>.go` with struct implementing `Worker` interface
2. Add dependencies (repos, adapters, services) to struct
3. Implement `Start(ctx)` method with ticker loop respecting context cancellation
4. Implement `Stop()` for cleanup (close channels, flush buffers)
5. Add config in `internal/adapters/config/config.go` (enabled, interval)
6. Register in scheduler (`internal/workers/scheduler.go`)
7. Load config and instantiate in `cmd/main.go`
8. Add tests with mocked dependencies

## Core Rules

- **Context respect**: Always check `<-ctx.Done()` in loops, exit gracefully
- **Graceful shutdown**: Implement `Stop()`, finish in-progress work, cleanup resources
- **Error handling**: Log errors, don't crash worker, implement retry with backoff
- **Idempotency**: Workers may run twice, ensure operations are idempotent
- **Rate limiting**: Respect exchange/API rate limits, use backoff
- **Logging**: Structured logs with worker name, iteration count, duration
- **Metrics**: Track execution count, duration, errors to ClickHouse

## Worker Pattern

Constructor accepts config, dependencies (repos, services, adapters). Start() runs ticker loop with select on ticker.C and ctx.Done(). On each tick, execute work, log result, handle errors. On context cancellation, cleanup and return.

## Configuration

Load from env in `internal/adapters/config/config.go`:
- `<WORKER>_ENABLED` — enable/disable
- `<WORKER>_INTERVAL` — run interval (e.g., "5m", "1h")
- Category-specific settings (symbols, exchanges, limits)

## Scheduler Integration

Scheduler starts all enabled workers in goroutines, manages lifecycle, handles signals (SIGINT/SIGTERM), gracefully stops workers on shutdown.

Register worker: `scheduler.Register("worker_name", workerInstance)`

## Error Handling

- **Transient errors**: Log warning, continue to next iteration
- **Permanent errors**: Log error, optionally disable worker, alert
- **Rate limit errors**: Backoff exponentially, reduce frequency
- Never panic, always recover and log

## Best Practices

**DO:**
- Use ticker for intervals, not sleep loops
- Check context cancellation in every iteration
- Log start/stop events with timestamps
- Track metrics for monitoring
- Implement health checks if exposed via API
- Test with short intervals in tests

**DON'T:**
- Block indefinitely without context check
- Store large state in memory
- Ignore errors silently
- Make unbounded loops
- Start goroutines without tracking
- Skip cleanup in Stop()

## Performance

- Batch database operations (bulk inserts)
- Use connection pooling from adapters
- Limit concurrent operations (semaphore/worker pool)
- Process in chunks for large datasets
- Monitor memory usage, avoid leaks

## Testing

- Mock dependencies (repos, adapters)
- Use short intervals (100ms) in tests
- Test context cancellation and graceful shutdown
- Verify idempotency with duplicate runs
- Check error handling paths

## Monitoring

Track via ClickHouse:
- Execution count per worker
- Success/failure rate
- Duration per run
- Records processed
- Errors by type

Log structured events:
- Worker start/stop
- Iteration complete (duration, records)
- Errors with context

## References

- Scheduler: `internal/workers/scheduler.go`
- Worker interface: `internal/workers/worker.go`
- Config: `internal/adapters/config/config.go`
- Main orchestration: `cmd/main.go`

