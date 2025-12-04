<!-- @format -->

# Tool Validation Architecture

## Separation of Concerns

Our tool validation follows a clear separation of responsibilities:

### 1. Tool Implementation (e.g., `trading/place_order.go`)

**Responsibility**: Parameter validation and business logic

```go
// ✅ Tool validates its own parameters
if symbol == "" || sideStr == "" || typeStr == "" || amountStr == "" {
    return nil, errors.ErrInvalidInput
}

// ✅ Tool validates business rules
amount, err := decimal.NewFromString(amountStr)
if err != nil {
    return nil, errors.Wrap(err, "invalid amount format")
}
```

**What tools should validate:**

- ✅ Required vs optional parameters
- ✅ Parameter types and formats
- ✅ Business rules (e.g., amount > 0, valid price ranges)
- ✅ Domain-specific constraints

### 2. Tool Callbacks (`internal/agents/callbacks/tool.go`)

**Responsibility**: Cross-cutting concerns (risk, security, observability)

```go
// ✅ Callback checks risk level from metadata
switch meta.RiskLevel {
case tools.RiskLevelCritical:
    // Check permissions, require confirmation
case tools.RiskLevelHigh:
    // Enhanced validation, audit logging
}
```

**What callbacks should validate:**

- ✅ Risk level enforcement
- ✅ Rate limiting
- ✅ Permission/authorization checks
- ✅ Circuit breaker status
- ✅ Audit logging
- ✅ Cost tracking

### 3. Tool Catalog (`internal/tools/catalog.go`)

**Responsibility**: Declarative metadata

```go
{
    Name:         "place_order",
    Category:     string(CategoryExecution),
    RiskLevel:    RiskLevelMedium,  // ✅ Declared once
    RequiresAuth: true,
    RateLimit:    30,
}
```

**What catalog defines:**

- ✅ Tool metadata (name, category, description)
- ✅ Risk levels (none, low, medium, high, critical)
- ✅ Auth requirements
- ✅ Rate limits
- ✅ Input/output schemas (future)

## Why This Architecture?

### ❌ Anti-pattern: Hardcoding in Callbacks

```go
// DON'T do this - hardcoded tool-specific logic in callback
switch toolName {
case "place_order":
    if args["symbol"] == nil || args["side"] == nil {
        return nil, errors.New("missing params")
    }
case "cancel_order":
    if args["order_id"] == nil {
        return nil, errors.New("missing order_id")
    }
}
```

**Problems:**

- 🔴 Callback becomes a dumping ground for all tool logic
- 🔴 Violates Single Responsibility Principle
- 🔴 Hard to maintain as tools grow (50+ tools)
- 🔴 Duplicates validation already in tool implementation
- 🔴 Tight coupling between callback and tool details

### ✅ Correct Pattern: Metadata-Driven Validation

```go
// DO this - generic risk-based validation
meta, ok := registry.GetMetadata(toolName)
switch meta.RiskLevel {
case tools.RiskLevelCritical:
    // Generic critical-risk handling
    log.Warn("Critical operation requires confirmation")
}
```

**Benefits:**

- ✅ Single source of truth (catalog)
- ✅ Scales to hundreds of tools
- ✅ Clear separation of concerns
- ✅ Easy to add new tools without changing callbacks
- ✅ Declarative and maintainable

## Adding a New Tool

### Step 1: Define in Catalog

```go
// internal/tools/catalog.go
{
    Name:         "my_new_tool",
    Category:     string(CategoryExecution),
    RiskLevel:    RiskLevelMedium,
    RequiresAuth: true,
    RateLimit:    60,
}
```

### Step 2: Implement Tool with Validation

```go
// internal/tools/mypackage/my_new_tool.go
func NewMyNewTool(deps shared.Deps) tool.Tool {
    return shared.NewToolBuilder(
        "my_new_tool",
        "Does something useful",
        func(ctx tool.Context, args map[string]interface{}) (map[string]interface{}, error) {
            // ✅ Validate parameters here
            param1, ok := args["param1"].(string)
            if !ok || param1 == "" {
                return nil, errors.ErrInvalidInput
            }

            // Tool logic...
            return result, nil
        },
        deps,
    ).Build()
}
```

### Step 3: Register

```go
// internal/tools/register.go
registry.Register("my_new_tool", mypackage.NewMyNewTool(deps))
```

**That's it!** Callbacks automatically:

- Check risk level
- Enforce rate limits
- Log audit trail
- Track costs

## Risk Levels

| Level      | Use Case                        | Examples                      |
| ---------- | ------------------------------- | ----------------------------- |
| `None`     | Read-only, no side effects      | `get_price`, `rsi`, `macd`    |
| `Low`      | Read user data, write memories  | `get_balance`, `save_insight` |
| `Medium`   | Single trade execution          | `place_order`, `cancel_order` |
| `High`     | Bulk operations                 | `cancel_all_orders`           |
| `Critical` | Emergency actions, irreversible | `emergency_close_all`         |

## Future Enhancements

### Schema-Based Validation

We can extend catalog to include JSON schemas:

```go
InputSchema: map[string]interface{}{
    "type": "object",
    "required": ["symbol", "side", "amount"],
    "properties": map[string]interface{}{
        "symbol": map[string]interface{}{"type": "string"},
        "side": map[string]interface{}{"enum": []string{"buy", "sell"}},
        "amount": map[string]interface{}{"type": "number", "minimum": 0},
    },
}
```

Then add a generic schema validator in callbacks. But even with this, domain-specific business rules should stay in tool implementations.




