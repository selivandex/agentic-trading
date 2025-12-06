# Database Seeds

Seed data for different environments using YAML files.

Seeds are stored in `cmd/seeder/seeds/` and use domain entities directly with YAML tags.

## Structure

```
cmd/seeder/seeds/
├── dev/              # Development environment (local)
│   ├── 01_limit_profiles.yaml
│   └── 02_users.yaml
├── test/             # Test environment (CI/integration tests)
│   ├── 01_limit_profiles.yaml
│   └── 02_users.yaml
└── staging/          # Staging environment (pre-production)
    └── 01_limit_profiles.yaml
```

## Key Features

✅ **Idempotent** - can be run multiple times safely (find or create)
✅ **Type-safe** - uses domain entities directly with YAML tags
✅ **Nested structure** - users contain their exchange accounts, strategies, etc.
✅ **No ID management** - IDs auto-generated, reference by unique fields
✅ **Environment separation** - dev/test/staging have separate data
✅ **Validation** - dry-run mode validates without inserting

## File Naming Convention

Files are numbered to ensure correct execution order:

- `01_limit_profiles.yaml` - No dependencies
- `02_users.yaml` - Depends on limit profiles

## Usage

### Apply seeds:
```bash
make db-seed              # Apply dev seeds (default)
make db-seed ENV=test     # Apply test seeds
make db-seed ENV=staging  # Apply staging seeds
```

### Validate without applying:
```bash
make db-seed-validate           # Validate dev seeds
make db-seed-validate ENV=test  # Validate test seeds
```

### Full database reset + seeds:
```bash
make db-reset      # Drop, create, migrate
make db-seed       # Apply seeds
```

## YAML Structure

### Limit Profiles (01_limit_profiles.yaml)

```yaml
limit_profiles:
  - name: "Conservative"                    # Unique identifier
    description: "Conservative limits"
    max_positions: 3
    max_position_size: 1000
    max_leverage: 2
    max_total_exposure: 3000
    monthly_ai_requests: 100
    is_active: true
```

### Users with Nested Entities (02_users.yaml)

```yaml
users:
  - telegram_id: 123456789                  # Unique identifier (or use email)
    telegram_username: "user123"
    first_name: "John"
    last_name: "Doe"
    language_code: "en"
    is_active: true
    is_premium: false
    limit_profile_name: "Conservative"      # Reference by name
    settings:
      risk_level: "moderate"
      max_positions: 5

    # User's exchange accounts
    exchange_accounts:
      - exchange: "binance"                 # binance, bybit, okx
        label: "Main Binance"               # Unique per user
        is_testnet: true
        is_active: true

      - exchange: "bybit"
        label: "Bybit Testnet"
        is_testnet: true
        is_active: true

    # User's strategies
    strategies:
      - name: "BTC Strategy"                # Unique per user
        description: "Bitcoin trading"
        status: "active"                    # active, paused, closed
        allocated_capital: "10000"
        current_equity: "10500"
        cash_reserve: "5000"
        market_type: "spot"                 # spot, futures
        risk_tolerance: "moderate"          # conservative, moderate, aggressive
        target_allocations:
          "BTC/USDT": 0.6
          "ETH/USDT": 0.4

        # Strategy's positions (optional)
        positions:
          - symbol: "BTC/USDT"
            side: "long"                    # long, short
            entry_price: "45000"
            size: "0.1"
            stop_loss_price: "44000"
            take_profit_price: "48000"
            status: "open"                  # open, closed

        # Strategy's orders (optional)
        orders:
          - symbol: "BTC/USDT"
            side: "buy"                     # buy, sell
            type: "limit"                   # market, limit
            amount: "0.1"
            price: "45000"
            status: "filled"                # pending, filled, canceled

    # User's memories (optional)
    memories:
      - agent_id: "market_analyst"
        session_id: "session_123"
        type: "observation"                 # observation, decision, trade
        content: "Market is bullish"
        symbol: "BTC/USDT"
        metadata:
          confidence: 0.85
```

## Referencing Entities

Instead of UUIDs, use natural identifiers:

- **Limit Profile** → reference by `name`
- **User** → reference by `telegram_id` or `email`
- **Exchange Account** → nested under user, unique by `label`
- **Strategy** → nested under user, unique by `name`
- **Position** → nested under strategy
- **Order** → nested under strategy
- **Memory** → nested under user

## Idempotency

Seeds are idempotent - running multiple times won't create duplicates:

- **Limit Profiles** - found by `name`
- **Users** - found by `telegram_id` or `email`
- **Exchange Accounts** - found by user + `label`
- **Strategies** - found by user + `name`
- **Positions** - found by strategy + symbol + side
- **Orders** - always created (not idempotent by nature)
- **Memories** - always created

## Field Types

### Decimal Fields
Use string format for decimal fields to avoid floating point precision issues:
```yaml
allocated_capital: "10000"      # ✅ Correct
allocated_capital: 10000        # ⚠️ May lose precision
```

### Enums
Use string values that match domain constants:
```yaml
status: "active"              # StrategyActive
market_type: "spot"           # MarketSpot
risk_tolerance: "moderate"    # RiskModerate
side: "long"                  # PositionLong
```

### Optional Fields
Use `omitempty` - if not specified, defaults are used:
```yaml
# These are optional:
permissions: ["spot", "trade"]
leverage: 1
margin_mode: "cross"
```

## Best Practices

1. **Environment isolation** - Never share seeds between dev/test/staging
2. **Testnet only** - Always use `is_testnet: true` in dev/test
3. **No secrets** - Never put real API keys or production credentials
4. **Descriptive names** - Use clear, descriptive names for strategies/accounts
5. **Validate first** - Run with `--dry-run` before applying
6. **Version control** - Commit seed files to track changes
7. **Minimal test data** - Keep test seeds minimal, detailed in dev

## Examples

### Minimal User (test environment)

```yaml
users:
  - telegram_id: 123456789
    telegram_username: "test"
    first_name: "Test"
    last_name: "User"
    language_code: "en"
    is_active: true
    is_premium: false
    limit_profile_name: "Test Default"
```

### Complex User (dev environment)

```yaml
users:
  - telegram_id: 123456789
    telegram_username: "dev_trader"
    first_name: "Developer"
    last_name: "Trader"
    language_code: "en"
    is_active: true
    is_premium: true
    limit_profile_name: "Aggressive"
    settings:
      risk_level: "aggressive"
      max_positions: 10

    exchange_accounts:
      - exchange: "binance"
        label: "Binance Futures"
        is_testnet: true
        is_active: true

      - exchange: "bybit"
        label: "Bybit Derivatives"
        is_testnet: true
        is_active: true

    strategies:
      - name: "Scalping Strategy"
        description: "High-frequency scalping"
        status: "active"
        allocated_capital: "50000"
        current_equity: "52000"
        cash_reserve: "10000"
        market_type: "futures"
        risk_tolerance: "aggressive"
        target_allocations:
          "BTC/USDT": 0.5
          "ETH/USDT": 0.3
          "SOL/USDT": 0.2

        positions:
          - symbol: "BTC/USDT"
            side: "long"
            entry_price: "45000"
            size: "1.0"
            leverage: 5
            stop_loss_price: "44000"
            take_profit_price: "48000"
            status: "open"
```

## Troubleshooting

### "No seed files found"
- Check that environment directory exists (`seeds/dev`, `seeds/test`, etc.)
- Ensure files have `.yaml` or `.yml` extension

### "Limit profile not found"
- Ensure limit profile is defined in `01_limit_profiles.yaml`
- Check that `limit_profile_name` matches exactly

### "Failed to create entity"
- Run with `--dry-run` to validate YAML syntax
- Check that enum values match domain constants
- Verify decimal fields use string format

### "Validation failed"
- Check YAML syntax is correct
- Ensure required fields are present
- Verify nested structure is valid

## Advanced Usage

### Custom seed directory
```bash
go run ./cmd/seeder --dir custom/seeds --env dev
```

### Dry run (validation only)
```bash
go run ./cmd/seeder --env dev --dry-run
```

## Development Workflow

1. **Start fresh database:**
   ```bash
   make db-reset
   ```

2. **Apply seeds:**
   ```bash
   make db-seed
   ```

3. **Validate changes before committing:**
   ```bash
   make db-seed-validate
   ```

4. **Test different environments:**
   ```bash
   make db-seed ENV=test
   make db-seed ENV=staging
   ```

## Notes

- Seeds use **find or create** pattern for idempotency
- IDs are auto-generated, never specify in YAML
- Use natural keys for references (name, telegram_id, label)
- Nested structure ensures data consistency
- Domain entities are used directly (no duplication)
