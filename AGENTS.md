<!-- @format -->

AGENTS.md — Prometheus Coding Rules

Project Context:
AI agents act as a full trading desk (10-20 analysts). User connects exchange via Telegram → /invest 1000 → agents manage portfolio: design allocation, monitor markets, rebalance positions, manage risk, execute trades. Autonomous AI-driven hedge fund operating 24/7 at scale.

CRITICAL: Design Principles

Before AND after writing code, ask yourself:

1. Am I following CLEAN architecture principles? (separation of concerns, dependency rule)
2. Am I following SOLID principles? (SRP, OCP, LSP, ISP, DIP)
3. Am I following DRY? (no code duplication, single source of truth)

If answer is NO to any — STOP and refactor.

Architecture

Clean layers (strict dependency rule):

- `cmd/` → `internal/bootstrap/` → `internal/api/`
- `internal/api/` → `internal/services/`
- `internal/services/` → `internal/domain/` + `internal/repository/`
- `internal/adapters/` ← injected via DI
- No circular dependencies. No global state.

Structure:

- `cmd/` — entrypoints
- `internal/adapters/` — db, AI, exchanges, Telegram, Kafka, Redis clients
- `internal/agents/` — ADK agent definitions + registry/factory
- `internal/domain/` — entities + repository interfaces
- `internal/repository/` — repository implementations
- `internal/services/` — business logic
- `internal/tools/` — ADK tools
- `internal/workers/` — background jobs
- `pkg/` — shared utilities

Mandatory Rules

Error Handling:

- Use `pkg/errors` for wrapping: `fmt.Errorf("context: %w", err)`
- Never ignore errors
- Never return raw stdlib errors

Logging:

- Use `pkg/logger` with structured fields
- Include: `user_id`, `agent`, `exchange`
- NEVER log secrets/keys/tokens

Dependency Injection:

- Inject via constructors in `cmd/main.go` or `internal/bootstrap/providers.go`
- Return interfaces, accept interfaces
- No global variables

Concurrency:

- Always use `context.Context`
- Prevent goroutine leaks
- Use Redis locks for distributed state

Testing:

- Table-driven tests
- Mock all interfaces
- Integration tests in separate suites

Repository Pattern:

- Interface in `internal/domain/<module>/repository.go`
- Implementation in `internal/repository/postgres/<module>.go` or `internal/repository/clickhouse/<module>.go`

Encryption:

- Use encryption helpers for exchange credentials
- Never store plaintext secrets

Tech Stack

- Go 1.22+, Google ADK Go
- PostgreSQL 16 + pgvector, ClickHouse, Redis, Kafka
- OpenTelemetry, Prometheus metrics

Commands

- `make lint && make fmt` — before commit
- `make test` — run tests
- `make docker-up` — start services
- Full list in `Makefile`

Quick Actions

New agent: `internal/agents/<name>` → register in `registry.go` → add tools → template in `pkg/templates/`

New exchange: implement interface in `internal/adapters/exchanges/<provider>` → wire in factory → env vars → tests

New worker: implement in `internal/workers/<category>/` → register in `registry.go` → wire in bootstrap

Forbidden

- NEVER create .md/.txt files without explicit consent
- NEVER use sed for mass-replace
- NEVER bypass circuit breaker (`internal/risk/`)
- NEVER commit secrets
- NEVER use global state
- NEVER ignore linter errors

Self-Check Before Commit

1. ✓ Clean Architecture layers respected?
2. ✓ SOLID principles followed?
3. ✓ No code duplication (DRY)?
4. ✓ All errors handled?
5. ✓ Interfaces used for DI?
6. ✓ Tests written?
7. ✓ `make lint && make fmt` passed?
8. ✓ No secrets in code?
