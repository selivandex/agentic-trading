<!-- @format -->

# Agents

Agent definitions for Google ADK integration. Agents orchestrate tools to accomplish trading tasks.

## Architecture

- **Registry** (`registry.go`) — catalog of all agents with metadata
- **Factory** (`factory.go`) — instantiates agents with dependencies
- **Tool Assignments** (`tool_assignments.go`) — maps available tools to each agent
- **Config** (`config.go`) — agent-specific settings (model, temperature, etc.)

Agents use Google ADK for LLM orchestration and tool calling.

## Available Agents

| Agent | Purpose | Key Tools |
|-------|---------|-----------|
| Market Analysis | Analyze conditions, detect regimes | Market data, indicators |
| Risk Manager | Validate trades, manage risk | Risk validation, circuit breaker |
| Trade Executor | Execute orders | Place/cancel orders, positions |
| Portfolio Manager | Portfolio optimization | Balances, positions |

## Adding New Agent

1. Add config struct in `config.go` with Enable, Model, Temperature fields
2. Create implementation in `<agentname>/agent.go` with constructor accepting config, logger, tools
3. Register in `registry.go` with name, description, version, enabled status
4. Wire in `factory.go` switch case to instantiate with dependencies
5. Assign tools in `tool_assignments.go` map
6. Create prompt template in `pkg/templates/assets/<agent>_system.tmpl`
7. Update `docs/DEVELOPMENT_PLAN.md` with agent purpose and phase

## Core Rules

- **Single responsibility**: Keep agents focused on one domain
- **Stateless**: No state in agent struct, use services (memory, journal)
- **Log reasoning**: Write all decisions to journal service with structured metadata
- **Context handling**: Respect cancellation, pass context through tool calls
- **Risk validation**: Never bypass circuit breaker, always validate before trades
- **Error handling**: Use `pkg/errors`, don't swallow tool failures

## Agent Implementation Pattern

- Constructor accepts config, logger, tool list
- `Process(ctx, input)` method orchestrates tool calls via ADK
- Log agent name + user_id + input hash at start
- Store reasoning steps in journal service
- Store insights in memory service for retrieval
- Return structured result or error

## Tool Usage

Agents call tools via ADK runtime. Tools return structured data (maps). Agent processes results and makes decisions. All tool calls logged automatically via middleware.

## Memory & State

**Stateless agents** — use services for persistence:
- **Journal service**: Store reasoning, decisions, actions with timestamps
- **Memory service**: Store insights, learnings, context with embeddings for semantic search
- **Order/Position services**: Query current state, don't cache in agent

## Testing

- Mock tool list in unit tests
- Test decision logic with different input scenarios
- Integration tests with real tools in sandbox mode (test API keys)
- Validate journal/memory writes after agent execution

## Configuration

Load from env vars in `internal/adapters/config/config.go`:
- `<AGENT>_ENABLED` — enable/disable
- `<AGENT>_MODEL` — LLM model name
- `<AGENT>_TEMPERATURE` — creativity (0-1)
- `<AGENT>_MAX_TOKENS` — response limit

## Monitoring

Track metrics via OpenTelemetry:
- Invocation count per agent
- Success/failure rate
- Latency (p50, p95, p99)
- Token usage
- Tool call distribution

## References

- ADK adapters: `internal/adapters/adk/`
- Tool registry: `internal/tools/registry.go`
- Templates: `pkg/templates/`
- Orchestration: `cmd/main.go`
