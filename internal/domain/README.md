<!-- @format -->

# Domain

Business entities, repository interfaces, and domain services following DDD principles.

## Architecture

Each bounded context has:
- `entity.go` — domain entities with business logic
- `repository.go` — repository interface (data access contract)
- `service.go` — domain service (optional, for complex operations)

Implementations live in `internal/repository/<store>/` (postgres, clickhouse).

## Bounded Contexts

| Context | Purpose | Storage |
|---------|---------|---------|
| **user** | User accounts, profiles | Postgres |
| **exchange_account** | Exchange credentials (encrypted) | Postgres |
| **trading_pair** | Trading pairs, configs | Postgres |
| **order** | Order lifecycle, history | Postgres |
| **position** | Position tracking | Postgres |
| **memory** | Agent memory with embeddings | Postgres (pgvector) |
| **journal** | Agent reasoning logs | Postgres |
| **risk** | Risk limits, rules | Postgres |
| **market_data** | OHLCV, tickers | ClickHouse |
| **sentiment** | News, sentiment scores | ClickHouse |
| **regime** | Market regime detection | ClickHouse |
| **stats** | Performance metrics | ClickHouse |

## Adding New Domain

1. Create `internal/domain/<context>/` directory
2. Define entity in `entity.go` with struct, validation methods, business logic
3. Define repository interface in `repository.go` with CRUD + query methods
4. Implement repository in `internal/repository/<store>/<context>.go`
5. Create service in `service.go` if complex operations span multiple entities
6. Add migrations in `migrations/<store>/`
7. Wire in `cmd/main.go` via provider helpers

## Core Rules

- **Entity encapsulation**: Business logic in entities, not services
- **Rich domain model**: Methods on entities for validation, state transitions
- **Repository interface**: Define in domain, implement in infra layer
- **Service layer**: Only for operations spanning multiple aggregates/repos
- **Dependency direction**: Services depend on repos, repos depend on entities
- **Encryption**: Use `pkg/crypto` for sensitive fields (API keys, secrets)
- **Validation**: Validate in entity constructors, return errors for invalid state

## Entity Pattern

Struct with fields, constructor `New<Entity>(...)` validating inputs, methods for state transitions, validation methods returning errors, no setters (immutability preferred).

## Repository Interface

Methods: `Create(ctx, entity)`, `Update(ctx, entity)`, `GetByID(ctx, id)`, `Delete(ctx, id)`, domain-specific queries `FindByUserID(ctx, userID)`, pagination support for lists.

Return `entity, error` or `[]entity, error`. Use `pkg/errors` for wrapping.

## Service Pattern

Constructor accepts repos, logger, other deps. Methods orchestrate operations across repos, handle transactions, enforce business rules. Return domain errors, not infra errors.

Example: `exchange_account.Service` encrypts/decrypts keys, validates exchange configs.

## Persistence

- **Postgres**: Transactional data (users, orders, positions, memory)
- **ClickHouse**: Analytics, time-series (market data, stats, sentiment)

Use appropriate storage based on access patterns and consistency requirements.

## Encryption

Sensitive fields (exchange API keys, secrets) must be encrypted at service layer before storing. Use `pkg/crypto` helpers. Never log encrypted fields in plaintext.

## Testing

- **Entity tests**: Validate business logic, state transitions, validation rules
- **Repository tests**: Integration tests against real DB (use `internal/testsupport`)
- **Service tests**: Mock repos, test orchestration logic

## Validation

Entities validate themselves. Return structured errors for invalid state. Use Go validator tags where appropriate. Validate on construction and before state changes.

## Transactions

Services handle transactions when operations span multiple repos. Use `BeginTx(ctx)` from Postgres client, pass tx context to repo methods, commit/rollback in service.

## Error Handling

- **Domain errors**: Define custom error types in `pkg/errors` (e.g., `ErrNotFound`, `ErrInvalidState`)
- **Wrapping**: Repos wrap storage errors with context
- **Services**: Return domain errors to callers

## Best Practices

**DO:**
- Keep entities rich with behavior
- Define clear aggregate boundaries
- Use value objects for complex types
- Validate in constructors
- Return immutable entities where possible
- Document business rules in comments

**DON'T:**
- Put business logic in repos (only data access)
- Expose database details in domain
- Use database types in entities (use primitives)
- Skip validation
- Return nil without error
- Cross aggregate boundaries in transactions

## Naming Conventions

- Entity: `Order`, `Position`, `User` (singular, noun)
- Repository: `OrderRepository`, `PositionRepository` (interface)
- Service: `OrderService`, `PositionService`
- Methods: `CreateOrder`, `GetOrderByID`, `UpdateOrderStatus`

## References

- Repository implementations: `internal/repository/`
- Migrations: `migrations/postgres/`, `migrations/clickhouse/`
- Crypto helpers: `pkg/crypto/`
- Error types: `pkg/errors/`

