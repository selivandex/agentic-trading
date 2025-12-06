# Database Seeds

Go-based seed data for different environments using the builder pattern.

## Why Go instead of YAML?

✅ **Type-safe** - compile-time validation
✅ **IDE support** - autocomplete, refactoring, go-to-definition
✅ **Reusable** - same builders as in tests
✅ **No duplication** - uses `internal/testsupport/seeds` directly
✅ **Flexible** - full Go power for complex scenarios

## Structure

```
internal/seeds/
├── dev/              # Development environment
│   ├── 01_limit_profiles.go
│   └── 02_users.go
├── test/             # Test environment
│   ├── 01_limit_profiles.go
│   └── 02_users.go
└── staging/          # Staging environment
    └── 01_limit_profiles.go
```

## Usage

```bash
make db-seed              # Apply dev seeds (default)
make db-seed ENV=test     # Apply test seeds
make db-seed ENV=staging  # Apply staging seeds
make db-seed-validate     # Validate (dry-run)
```

## Writing Seeds

Each seed file exports a function that receives `*seeds.Seeder`:

```go
package dev

import (
	"context"
	"prometheus/internal/testsupport/seeds"
	"prometheus/pkg/logger"
)

func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	log := logger.Get()

	// Create user with builder pattern
	user := s.User().
		WithTelegramID(123456789).
		WithUsername("dev_user").
		WithFirstName("John").
		WithLastName("Developer").
		WithActive(true).
		WithPremium(true).
		MustInsert()

	log.Infow("Created user", "id", user.ID)

	// Create exchange account for user
	s.ExchangeAccount().
		WithUserID(user.ID).
		WithBinance().
		WithLabel("My Binance").
		WithTestnet(true).
		WithActive(true).
		MustInsert()

	// Create strategy for user
	s.Strategy().
		WithUserID(user.ID).
		WithName("BTC Strategy").
		WithActive().
		WithSpot().
		WithModerateRisk().
		MustInsert()

	return nil
}
```

## Available Builders

See `internal/testsupport/seeds/` for all builders:

- `s.User()` - User builder
- `s.LimitProfile()` - Limit profile builder
- `s.ExchangeAccount()` - Exchange account builder
- `s.Strategy()` - Strategy builder
- `s.Position()` - Position builder
- `s.Order()` - Order builder
- `s.Memory()` - Memory builder

## Execution Order

Seeds are executed in the order defined in `cmd/seeder/main.go`:

```go
func getSeedFunctions(env string) []func(...) error {
	switch env {
	case "dev":
		return []func(...) error{
			devseeds.SeedLimitProfiles,  // 1. No dependencies
			devseeds.SeedUsers,           // 2. May reference limit profiles
		}
	}
}
```

## Best Practices

1. **Idempotency** - Seeds should be safe to run multiple times
2. **Check before create** - Use repository to check if entity exists
3. **Clean separation** - One file per entity type
4. **Minimal test data** - Keep test seeds minimal
5. **Realistic staging** - Staging should mirror production

## Idempotency Pattern

```go
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	// Try to find existing user
	existing, err := userRepo.GetByTelegramID(ctx, 123456789)
	if err == nil && existing != nil {
		log.Info("User already exists, skipping")
		return nil
	}

	// Create only if doesn't exist
	s.User().
		WithTelegramID(123456789).
		// ...
		MustInsert()

	return nil
}
```

## Adding New Seeds

1. Create new file in appropriate environment directory
2. Export seed function: `func SeedXXX(ctx, s) error`
3. Register in `cmd/seeder/main.go` in `getSeedFunctions()`
4. Run `make db-seed ENV=xxx` to test

## Development Workflow

```bash
# 1. Reset database
make db-reset

# 2. Apply seeds
make db-seed

# 3. Validate without applying
make db-seed-validate

# 4. Test different environments
make db-seed ENV=test
make db-seed ENV=staging
```

## Examples

### Minimal User (test)

```go
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	s.User().
		WithTelegramID(111111111).
		WithUsername("test").
		WithFirstName("Test").
		WithLastName("User").
		MustInsert()

	return nil
}
```

### Complex User with Nested Entities (dev)

```go
func SeedUsers(ctx context.Context, s *seeds.Seeder) error {
	log := logger.Get()

	// Create user
	user := s.User().
		WithTelegramID(123456789).
		WithUsername("trader").
		WithPremium(true).
		MustInsert()

	// User's exchange accounts
	binance := s.ExchangeAccount().
		WithUserID(user.ID).
		WithBinance().
		WithLabel("Binance Testnet").
		WithTestnet(true).
		MustInsert()

	// User's strategy
	strategy := s.Strategy().
		WithUserID(user.ID).
		WithName("BTC Strategy").
		WithActive().
		WithSpot().
		MustInsert()

	// Strategy's position
	s.Position().
		WithUserID(user.ID).
		WithStrategyID(strategy.ID).
		WithExchangeAccountID(binance.ID).
		WithSymbol("BTC/USDT").
		WithLongSide().
		MustInsert()

	log.Infow("Created full user setup", "user_id", user.ID)
	return nil
}
```

## Notes

- Seeds use builders from `internal/testsupport/seeds`
- No YAML parsing overhead
- Type-safe at compile time
- Same code in tests and seeds
- Refactoring tools work perfectly
