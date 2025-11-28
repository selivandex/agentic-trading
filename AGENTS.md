<!-- @format -->

We’re building a project where AI agents act like a full trading desk — 10–20 analysts working inside a hedge fund.
Each agent understands the market, correlations, technical analysis, macro signals, liquidity zones, volatility regimes — the whole package.

A user simply connects their exchange account through Telegram and invests with a command like /invest 1000.
From that moment, the agents take the user’s exchange account into circulation:
they design the portfolio, allocate capital, monitor market conditions, react to volatility, rebalance positions, buy more, close trades, and manage risk with stop-losses and other controls.

In other words, we’re building an AI-driven fund.
The agents do the exact same work professional traders and analysts would do — but continuously, automatically, and at scale.

⸻

Never create .md docs files without explicit consent !!!!!!!!!!!!
Never create .txt examples files without explicit consent
Never use sed function to mass-replace code. It's very dangerous

# AGENTS — Prometheus (Agentic Trading System)

Guidance for AI agents working in this codebase. Follow idiomatic Go, Clean Architecture, and DDD principles. Keep code modular, testable, and observable.

---

## 0. Core Principles

- **Architecture**: Clean layers (handlers → services → repositories → domain), interface-driven DI, composition over inheritance
- **Error Handling**: Wrap with `fmt.Errorf("context: %w", err)`, never ignore errors
- **Concurrency**: Guard with `context.Context`, prevent goroutine leaks, use Redis locks for distributed state
- **Testing**: Table-driven unit tests, mock interfaces, separate integration/E2E suites
- **Observability**: OpenTelemetry spans + structured logging (zap), include user_id/agent/exchange but never secrets
- **Tooling**: Run `make lint` and `make fmt` before committing; see Makefile for all commands

## 1. Mission & Context

- Build **Prometheus**, an autonomous multi-agent crypto trading stack with Telegram-first control, centralized memory, and multi-exchange execution.
- Core principles: fail-safe execution (risk engine + circuit breaker), explainable LLM reasoning (logs, journals), and deterministic migrations/config.
- Primary docs: `README.md` (overview), `docs/ENV_SETUP.md` (env vars), `docs/DEVELOPMENT_PLAN.md` (roadmap), `docs/specs.md` (full spec). Always consult them before guessing.

## 2. Architecture Snapshot

- Language/runtime: Go 1.22+ with Google ADK Go for agents.
- Data layer: PostgreSQL 16 + pgvector (state, memories), ClickHouse (market & telemetry), Redis (locks/cache), Kafka (events).
- Adapter layout: `internal/adapters/*` for infra clients (db, AI providers, exchanges, Telegram bot, Kafka, redis).
- Business layer: `internal/domain/*` entities + repositories; encryption handled in services.
- Agent/tool layer: `internal/agents`, `internal/tools`, `internal/workers`, `internal/risk`.
- Templates/prompts live under `pkg/templates` (Phase 5 TODO).
- ADK integration: agents implement Google ADK interfaces inside `internal/agents/<name>`, are registered in `internal/agents/registry.go`, and are instantiated via `internal/agents/factory.go` so orchestration stays declarative.

### Repository Layout

```
.
├── cmd/                     # Application entrypoints (ADK bootstrap)
├── internal/
│   ├── adapters/           # Infra clients (db, AI, exchanges, Telegram, Kafka, redis)
│   ├── agents/             # ADK agent definitions + registry/factory/orchestrators
│   ├── domain/             # Entities, repositories, services per bounded context
│   ├── tools/              # Tool implementations exposed to ADK agents
│   ├── workers/            # Schedulers + collectors
│   ├── risk/               # Circuit breaker & kill switch logic
│   └── events/             # Kafka topic contracts (add/update schemas here)
├── pkg/                    # Shared utilities (logger, templates, errors, etc.)
├── docs/                   # Specs, environment/setup docs, roadmap
├── migrations/             # Postgres & ClickHouse migrations
├── templates/              # Prompt/template prompts (Phase 5 focus)
└── Makefile                # Canonical local workflow commands
```

## 3. Environment & Commands

- **Setup**: Copy `.env.local.example` → `.env`, fill keys per `docs/ENV_SETUP.md`. Never commit secrets. Generate encryption keys: `make gen-encryption-key`
- **Key commands**: `make docker-up`, `make deps`, `make build`, `make run`, `make test`, `make lint`, `make fmt` — see Makefile for full list
- **Workflow**: Plan → Sync (`make deps`) → Develop (`cmd` → `internal` → `pkg` direction) → Test → Lint/Fmt → Update docs

## 4. Prometheus-Specific Coding Rules

- **Constructors**: Context-aware, return interfaces (see `internal/adapters/exchanges/factory.go`)
- **Repositories**: Interfaces in `internal/domain/<module>/repository.go`, implementations in `internal/repository/*`
- **Encryption**: Always use encryption helpers for exchange keys; use pgvector for semantic memory
- **Logging**: Use `pkg/logger` with structured fields (`user_id`/`agent`/`exchange`), never log secrets, write meaningful messages with context
- **Errors**: Always use `pkg/errors` for error wrapping/tracking, never return raw errors from stdlib
- **DI**: Inject deps in `cmd/main.go` via constructor helpers (`provideDB()`, `provideRedis()`), no global state
- **Testing**: Mock exchanges/HTTP, run integration tests against dockerized services for adapter changes

## 5. Operational Guardrails

- **Risk**: Never bypass circuit breaker logic (`internal/risk`) — fail-safe execution is mandatory
- **Kafka**: Topics = agent contracts; version schemas in `internal/events` before changes
- **Telegram**: Degrade gracefully on AI rate-limits; retries/backoffs in adapters, not agents
- **Config**: Feature toggles via env vars only

## 6. Current Focus (Nov 2025)

- **Phase 5+ TODO**: Template registry, tool registry, prompt authoring, agent orchestration wrappers
- **Priority**: Complete template/tool registry infrastructure before expanding exchange coverage
- **Documentation**: Update `docs/DEVELOPMENT_PLAN.md` + inline comments in `internal/tools/registry` for new tools/agents

## 7. Quick Reference

- **New agent**: Define in `internal/agents/<name>`, register in `registry.go`, extend tool map, add template (`pkg/templates`), update docs
- **New exchange**: Implement interface in `internal/adapters/exchanges/<provider>`, wire factory, add env + docs, create integration tests
- **New telemetry**: ClickHouse schema in `migrations/clickhouse`, wire worker collectors

## 8. Documentation Rules

- Major design changes → update `docs/DEVELOPMENT_PLAN.md` + PR description
- All code TODOs → reference issue/phase milestone
- Code/comments in English; high-level specs accept Russian

---

**Stay disciplined: deterministic builds, strong typing, auditable reasoning logs.**
