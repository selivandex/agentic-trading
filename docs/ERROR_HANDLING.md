<!-- @format -->

# Error Handling Guidelines

## Overview

Prometheus uses a layered error handling strategy that combines standard Go error wrapping with domain-specific errors for business logic.

## Error Types

### 1. **Standard Wrapping** (`fmt.Errorf`)

Use for **infrastructure/technical errors**:

```go
// ✅ Good: Wrapping database errors
if err := repo.Save(ctx, user); err != nil {
    return fmt.Errorf("failed to save user: %w", err)
}

// ✅ Good: Adding context to errors
if err := exchange.PlaceOrder(ctx, req); err != nil {
    return fmt.Errorf("place order on %s: %w", exchange.Name(), err)
}
```

### 2. **Domain Errors** (`pkg/errors`)

Use for **business logic errors** that callers need to handle differently:

```go
import "prometheus/pkg/errors"

// ✅ Good: Return specific domain error
if engine.IsCircuitTripped(userID) {
    return errors.ErrCircuitBreakerTripped
}

// ✅ Good: Wrap domain error with context
if drawdownExceeded {
    return errors.Wrap(errors.ErrDrawdownExceeded, "daily PnL check")
}
```

### 3. **Helper Functions**

Use `pkg/errors` helpers for cleaner code:

```go
// Instead of:
return fmt.Errorf("context: %w", err)

// Use:
return errors.Wrap(err, "context")

// With formatting:
return errors.Wrapf(err, "failed to process symbol %s", symbol)
```

## Available Domain Errors

### General

- `ErrNotFound` - Resource not found
- `ErrAlreadyExists` - Resource already exists
- `ErrInvalidInput` - Invalid input parameters
- `ErrUnauthorized` - Insufficient permissions
- `ErrForbidden` - Action is forbidden
- `ErrInternal` - Internal server error
- `ErrTimeout` - Operation timeout
- `ErrUnavailable` - Service unavailable

### Risk Management

- `ErrTradingBlocked` - Trading blocked by risk engine
- `ErrCircuitBreakerTripped` - Circuit breaker is active
- `ErrDrawdownExceeded` - Daily drawdown limit exceeded
- `ErrConsecutiveLosses` - Consecutive losses limit exceeded
- `ErrMaxExposure` - Maximum exposure limit reached
- `ErrKillSwitchActive` - Kill switch is active

### Exchange Operations

- `ErrExchangeUnavailable` - Exchange API unavailable
- `ErrInsufficientBalance` - Insufficient account balance
- `ErrInvalidSymbol` - Invalid trading symbol
- `ErrOrderRejected` - Order rejected by exchange
- `ErrPositionNotFound` - Position not found
- `ErrRateLimitExceeded` - API rate limit exceeded

## Usage Patterns

### Pattern 1: Repository Layer

```go
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Entity, error) {
    var entity Entity
    err := r.db.GetContext(ctx, &entity, query, id)
    if err == sql.ErrNoRows {
        return nil, errors.ErrNotFound  // Domain error
    }
    if err != nil {
        return nil, fmt.Errorf("get entity: %w", err)  // Wrap technical error
    }
    return &entity, nil
}
```

### Pattern 2: Service Layer

```go
func (s *Service) ValidateAndCreate(ctx context.Context, input Input) (*Entity, error) {
    if input.Amount <= 0 {
        return nil, errors.NewValidationError("amount", "must be positive", input.Amount)
    }
    
    entity, err := s.repo.Create(ctx, input)
    if err != nil {
        if errors.Is(err, errors.ErrAlreadyExists) {
            return nil, err  // Pass domain error up
        }
        return nil, fmt.Errorf("create entity: %w", err)
    }
    
    return entity, nil
}
```

### Pattern 3: API/Tool Layer

```go
func (t *Tool) Execute(ctx context.Context, args Args) (Response, error) {
    canTrade, err := riskEngine.CanTrade(ctx, userID)
    if err != nil {
        // Check for specific domain errors
        if errors.Is(err, errors.ErrCircuitBreakerTripped) {
            return nil, errors.NewDomainError(
                "CIRCUIT_BREAKER_ACTIVE",
                "Trading is blocked. Circuit breaker is active due to risk limits.",
                err,
            )
        }
        if errors.Is(err, errors.ErrDrawdownExceeded) {
            return nil, errors.NewDomainError(
                "DRAWDOWN_EXCEEDED",
                "Trading is blocked. Daily drawdown limit exceeded.",
                err,
            )
        }
        return nil, errors.Wrap(err, "risk check failed")
    }
    
    if !canTrade {
        return nil, errors.ErrTradingBlocked
    }
    
    // ... continue with trade execution
}
```

## Error Checking

### Using `errors.Is` for Sentinel Errors

```go
if errors.Is(err, errors.ErrCircuitBreakerTripped) {
    // Handle circuit breaker specifically
    log.Warn("Circuit breaker active", "user_id", userID)
    sendNotification(userID, "circuit_breaker_alert")
    return
}
```

### Using `errors.As` for Type Assertions

```go
var domainErr *errors.DomainError
if errors.As(err, &domainErr) {
    // Handle domain error with code
    log.Error("Domain error", "code", domainErr.Code, "message", domainErr.Message)
    return domainErr.Code, domainErr.Message
}
```

## Best Practices

### ✅ DO

1. **Return domain errors** for business logic violations
2. **Wrap infrastructure errors** with context using `fmt.Errorf` or `errors.Wrap`
3. **Use sentinel errors** for well-known failure modes
4. **Check errors** with `errors.Is` and `errors.As`
5. **Add context** at each layer

### ❌ DON'T

1. **Don't create new error strings** for known errors - use sentinel errors
2. **Don't lose error context** - always wrap with `%w` or `errors.Wrap`
3. **Don't ignore errors** - check every error explicitly
4. **Don't log and return** - either log OR return, not both (unless critical)
5. **Don't expose internal errors** to external APIs - wrap with domain errors

## Example: Risk Engine

```go
// ✅ Good: Combination of domain errors and wrapping
func (e *RiskEngine) CanTrade(ctx context.Context, userID uuid.UUID) (bool, error) {
    state, err := e.riskRepo.GetState(ctx, userID)
    if err != nil {
        // Wrap technical error
        return false, errors.Wrap(err, "failed to get risk state")
    }
    
    if state.IsTriggered {
        // Return domain error
        return false, errors.ErrCircuitBreakerTripped
    }
    
    if e.isDrawdownExceeded(state) {
        e.tripCircuitBreaker(ctx, state, "drawdown exceeded")
        // Return specific domain error
        return false, errors.ErrDrawdownExceeded
    }
    
    return true, nil
}
```

## Migration Strategy

When updating existing code:

1. **Identify business logic errors** → convert to domain errors
2. **Keep wrapping infrastructure errors** with `fmt.Errorf("%w")`
3. **Update tests** to check for domain errors with `errors.Is`
4. **Update error handlers** to provide user-friendly messages

---

**Last Updated:** 26 ноября 2025  
**Related:** `pkg/errors/errors.go`, `AGENTS.md` (Error handling section)



