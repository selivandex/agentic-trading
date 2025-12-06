<!-- @format -->

# Database Seeder

Command-line tool for applying database seeds using Go-based builder pattern.

## Features

✅ **Type-safe** - Go code with compile-time validation
✅ **Builder pattern** - uses `internal/testsupport/seeds` builders
✅ **Environment-specific** - separate seeds for dev/test/staging
✅ **Idempotent** - safe to run multiple times
✅ **No duplication** - same builders as tests

## Usage

```bash
# Apply dev seeds (default)
make db-seed

# Apply seeds for specific environment
make db-seed ENV=dev
make db-seed ENV=test
make db-seed ENV=staging

# Validate without applying (dry-run)
make db-seed-validate
make db-seed-validate ENV=test
```

## Direct usage

```bash
# Run with go
go run ./cmd/seeder --env dev
go run ./cmd/seeder --env test --dry-run

# Or build and run
go build -o bin/seeder ./cmd/seeder
./bin/seeder --env dev
```

## Structure

```
cmd/seeder/
├── main.go           # Entry point with environment router
└── README.md         # This file

internal/seeds/
├── dev/              # Development seeds
│   ├── 01_limit_profiles.go
│   └── 02_users.go
├── test/             # Test seeds (minimal)
│   ├── 01_limit_profiles.go
│   └── 02_users.go
├── staging/          # Staging seeds (production-like)
│   └── 01_limit_profiles.go
└── README.md         # Detailed seed development guide
```

## How It Works

1. **Seed files** in `internal/seeds/{env}/` export functions
2. **main.go** imports seed packages and calls functions in order
3. **Builders** from `internal/testsupport/seeds` create entities
4. **Repositories** handle database insertion

## Adding New Seeds

### 1. Create seed file

```go
// internal/seeds/dev/03_positions.go
package dev

import (
	"context"
	"prometheus/internal/testsupport/seeds"
)

func SeedPositions(ctx context.Context, s *seeds.Seeder) error {
	// Your seed logic here
	return nil
}
```

### 2. Register in main.go

```go
func getSeedFunctions(env string) []func(...) error {
	switch env {
	case "dev":
		return []func(...) error{
			devseeds.SeedLimitProfiles,
			devseeds.SeedUsers,
			devseeds.SeedPositions,  // Add here
		}
	}
}
```

### 3. Test

```bash
make db-seed-validate
make db-seed
```

## Best Practices

1. **Order matters** - list seeds in dependency order
2. **Check before create** - make seeds idempotent
3. **Use MustInsert()** - for dev/test (panics on error)
4. **Use Insert()** - for staging (returns error)
5. **Log progress** - use structured logging
6. **One file per entity type** - keep files focused

## Troubleshooting

### Import errors

- Ensure seed packages are in `internal/seeds/{env}/`
- Check imports in `cmd/seeder/main.go`

### Seeds not found

- Check function is exported (capitalized)
- Verify it's added to `getSeedFunctions()`

### Database errors

- Ensure `make db-migrate` ran first
- Check foreign key dependencies

## Development Workflow

```bash
# Full reset
make db-reset        # Drop, create, migrate
make db-seed         # Apply seeds

# Iterative development
# 1. Edit seed file in internal/seeds/dev/
# 2. Run make db-seed
# 3. Check database
# 4. Repeat
```

