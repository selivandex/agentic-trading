<!-- @format -->

# AGENTS — Prometheus (Agentic Trading System)

Codex-style agents read this file before working in the repo so they consistently follow our constraints and rituals per the AGENTS guidance from OpenAI Codex docs[^codex]. Keep the guidance concise, current, and actionable.

[^codex]: https://developers.openai.com/codex/guides/agents-md/

---

## 0. Role & Expectations

You are an expert in Go, microservices architecture, and clean backend development practices. Your role is to ensure code stays idiomatic, modular, testable, and aligned with modern best practices/design patterns.

### General Responsibilities

- Guide the development of maintainable, high-performance Go services.
- Enforce modular design and separation of concerns through Clean Architecture and DDD.
- Promote test-driven development, robust observability, and scalable patterns.

### Architecture Patterns

- Apply Clean Architecture layers (handlers/controllers → services/use cases → repositories/data access → domain models).
- Favor domain-driven design and interface-driven development with explicit dependency injection.
- Prefer composition over inheritance; expose small, purpose-specific interfaces.
- Require public functions to accept interfaces (not concrete types) to preserve flexibility and testability.

### Development Best Practices

- Keep functions short and focused; single responsibility only.
- Always check/handle errors explicitly, wrapping with `fmt.Errorf("context: %w", err)` for traceability.
- Avoid global state; prefer constructors to inject dependencies and propagate `context.Context`.
- Guard goroutines with sync primitives/channels; cancel work via context to prevent leaks.
- Defer resource cleanup and validate all external inputs.

### Security & Resilience

- Validate/sanitize every external input and enforce permission boundaries.
- Use secure defaults for tokens, JWTs, and configuration.
- Implement retries with exponential backoff, timeouts, circuit breakers, and rate limiting on all remote calls (Redis for distributed throttling when needed).

### Testing

- Write table-driven, parallel-friendly unit tests; separate fast unit suites from slower integration/E2E runs.
- Mock external interfaces (generated or handwritten) and ensure coverage for every exported function.
- Track coverage with `go test -cover`; ensure deterministic fixtures.

### Documentation & Standards

- Document public APIs with GoDoc comments; keep service-level READMEs concise.
- Maintain `CONTRIBUTING.md`/`ARCHITECTURE.md` style docs when behavior changes.
- Enforce consistent naming/formatting via `go fmt`, `goimports`, and `golangci-lint`.

### Observability & Telemetry

- Use OpenTelemetry (traces + metrics) and structured logging (zap) everywhere.
- Start/propagate spans across HTTP/gRPC/DB/external boundaries; include request IDs, user IDs, and error metadata.
- Export traces/metrics to OTel Collector/Jaeger/Prometheus; correlate logs with trace IDs.

### Performance & Concurrency

- Benchmark critical paths before/after changes; profile before optimizing.
- Minimize allocations in hot paths without sacrificing clarity.
- Ensure goroutine lifecycles respect context cancellation and shared state safety.

### Tooling & Dependencies

- Prefer the Go standard library; add third-party libs sparingly and version-pin via Go modules.
- Integrate linting, testing, and security scanners in CI; keep builds deterministic.

### Key Conventions

1. Prioritize readability, simplicity, and maintainability.
2. Design for change: isolate business logic, minimize framework lock-in.
3. Emphasize clear boundaries and dependency inversion.
4. Ensure all behavior is observable, testable, and documented.
5. Automate workflows for testing, building, deployment, and infra tasks.

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
├── templates/              # Prompt/template assets (Phase 5 focus)
└── Makefile                # Canonical local workflow commands
```

## 3. Environment & Secrets

1. Copy `.env.local.example` → `.env` then fill required keys listed in `docs/ENV_SETUP.md`.
2. Always keep AI/exchange API keys, encryption secrets, and Telegram tokens **out of the repo**; rely on `.env` and `doppler`/local vault if possible.
3. Generate encryption keys with `make gen-encryption-key`.
4. Never log secrets; use `pkg/logger` helpers and redact sensitive fields in structs.

## 4. Tooling & Commands

- `make deps` — download/tidy modules.
- `make docker-up` / `make docker-down` — spin up local Postgres, ClickHouse, Redis, Kafka.
- `make build` or `go build ./cmd/main.go` — compile binary to `bin/prometheus`.
- `make run` or `go run cmd/main.go` — start orchestrator.
- `make test`, `make test-coverage` — run unit tests/coverage; keep coverage artifacts out of git.
- `make lint` — `golangci-lint` must pass before opening PRs; install via `make install-tools`.
- `make fmt` — enforce `go fmt ./...`.
- `make install-tools` — install dev tooling like golangci-lint (run once per machine).
- `make generate` — run `go generate` hooks (mocks/codegen); keep generated artifacts committed.
- `make gen-encryption-key` — emit a 32-byte AES key for secrets management.
- Prefer `make` targets over ad-hoc commands to keep the workflow deterministic and documented.

## 5. Standard Workflow

1. **Plan** — confirm scope against `docs/DEVELOPMENT_PLAN.md`. Document any deviation inside PR/commit messages.
2. **Sync** — run `make deps` after pulling, ensure generated code (if any) stays in sync.
3. **Develop** — keep packages small, dependency direction `cmd` → `internal` → `pkg`; no imports from `cmd` into libraries.
4. **Testing** — unit tests via `make test`; add focused tests for new repositories/services. For adapter code hitting exchanges, mock HTTP/WebSocket clients.
5. **Static checks** — `make lint` and `make fmt` must be clean. Add go:generate directives only when reproducible.
6. **Docs** — update relevant markdown (`docs/`, `README.md`, templates) whenever behavior or config changes.

## 6. Coding Notes

- Prefer context-aware constructors returning interfaces (e.g., `internal/adapters/exchanges/factory.go`).
- Keep repository interfaces in `internal/domain/<module>/repository.go` and concrete implementations beside adapters.
- Use pgvector helpers for semantic memory operations; never bypass encryption helpers when dealing with exchange keys.
- Logging: use structured `logger` package; include `user_id`, `agent`, `exchange`, but never secrets.
- Errors: wrap with `fmt.Errorf("context: %w", err)`; expose domain errors via `pkg/errors`.
- Concurrency: guard shared state with `context.Context` + `redis` locks; avoid global vars other than DI singletons configured in `cmd/main.go`.

## 7. Testing & Verification Checklist

- **Unit tests**: `go test -v ./internal/...` for touched packages.
- **Integration smoke**: when adapters touching Redis/Postgres are modified, run targeted integration tests (see `internal/adapters/*/client_test.go` if present) against dockerized services.
- **Static analysis**: `golangci-lint run`.
- **Manual**: if Telegram bot flows or exchange adapters change, run the bot locally with sandbox API keys before merging.

## 8. Operational Guardrails

- Circuit breaker logic lives in `internal/risk`; never short-circuit it for "quick tests".
- Kafka topics are the contract between agents; coordinate schema changes through versioned structs in `internal/events` (or introduce them if missing).
- Telegram commands must degrade gracefully when AI providers are rate-limited; implement retries/backoffs at the adapter level, not inside agents.
- Feature toggles should be env-driven; avoid global flags.

## 9. Current Focus (Nov 2025)

- Phase 5+ roadmap items in `docs/DEVELOPMENT_PLAN.md`: template registry, tool registry, prompt authoring, and agent orchestration wrappers remain TODO.
- Prioritize completing template/tool registry infrastructure before expanding exchange coverage.
- Document every new tool/agent pairing in both `docs/DEVELOPMENT_PLAN.md` and inline comments within `internal/tools/registry`.

## 10. Reference Playbooks

- **Add a new agent**: define entity + config in `internal/agents/<name>`, register in `internal/agents/registry.go`, extend tool map, add prompt template placeholder (`pkg/templates`), update docs.
- **Add a new exchange adapter**: implement interface under `internal/adapters/exchanges/<provider>`, wire into factory, add env stubs + docs, create integration tests hitting mocked REST/WebSocket servers.
- **Add telemetry**: prefer ClickHouse for time-series; define schema under `migrations/clickhouse` (if missing) and wire worker collectors.

## 11. Communication

- Document design/decisions in PR descriptions and, when major, append a short section to `docs/DEVELOPMENT_PLAN.md`.
- All TODOs in code must reference an issue/phase milestone.
- Prefer English for code comments/docs; Russian acceptable for high-level spec sections already localized.

Stay disciplined: deterministic builds, strong typing, auditable reasoning logs.
