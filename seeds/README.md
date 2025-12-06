# Database Seeds

This directory contains seed data for different environments.

## Structure

```
seeds/
├── dev/              # Development environment (local)
│   ├── 01_limit_profiles.yaml
│   ├── 02_users.yaml
│   ├── 03_exchange_accounts.yaml
│   └── 04_strategies.yaml
├── test/             # Test environment (CI/integration tests)
│   ├── 01_limit_profiles.yaml
│   └── 02_users.yaml
└── staging/          # Staging environment (pre-production)
    └── 01_limit_profiles.yaml
```

## File Naming Convention

Files are named with a numeric prefix to ensure execution order:

- `01_` - Entities with no dependencies (limit_profiles)
- `02_` - Entities depending on 01 (users)
- `03_` - Entities depending on 02 (exchange_accounts)
- `04_` - Entities depending on 03 (strategies)
- `05_` - Entities depending on 04 (positions, orders)

## Usage

### Apply seeds for development:
```bash
make db-seed                    # Uses dev environment by default
make db-seed ENV=dev            # Explicit dev environment
```

### Apply seeds for test environment:
```bash
make db-seed ENV=test
```

### Apply seeds for staging:
```bash
make db-seed ENV=staging
```

### Clean database before seeding:
```bash
make db-seed-clean              # Clean + seed dev
make db-seed-clean ENV=staging  # Clean + seed staging
```

### Validate seeds without applying:
```bash
make db-seed-validate           # Validate dev seeds
make db-seed-validate ENV=test  # Validate test seeds
```

## YAML Structure

Each seed file can contain multiple entity types:

```yaml
# Limit Profiles
limit_profiles:
  - id: "uuid"
    name: "Conservative"
    max_daily_loss: 100.0
    max_positions: 3
    max_position_size: 1000.0
    max_leverage: 2.0
    is_default: true

# Users
users:
  - id: "uuid"
    telegram_id: 123456789
    telegram_username: "user123"
    first_name: "John"
    last_name: "Doe"
    language_code: "en"
    is_active: true
    is_premium: false
    limit_profile_id: "uuid"  # References limit profile
    settings:
      risk_level: "moderate"
      max_positions: 5
      max_position_size: 5000.0
      max_daily_loss: 500.0
      enable_notifications: true
      notify_on_trade: true
      notify_on_risk: true
      preferred_assets:
        - "BTC/USDT"
        - "ETH/USDT"

# Exchange Accounts
exchange_accounts:
  - id: "uuid"
    user_id: "uuid"  # References user
    exchange: "binance"
    api_key: "test_key"
    is_active: true
    is_testnet: true
    account_ref: "dev_account_1"

# Strategies
strategies:
  - id: "uuid"
    user_id: "uuid"  # References user
    name: "My Strategy"
    description: "Strategy description"
    type: "momentum"
    status: "active"
    initial_capital: 10000.0
    current_capital: 10500.0
    target_assets:
      - "BTC/USDT"
      - "ETH/USDT"
    risk_level: "moderate"
    max_positions: 3
    exchange_account_id: "uuid"  # References exchange account

# Positions
positions:
  - id: "uuid"
    user_id: "uuid"
    strategy_id: "uuid"
    exchange_account_id: "uuid"
    symbol: "BTC/USDT"
    side: "long"
    entry_price: 45000.0
    quantity: 0.1
    current_price: 46000.0
    stop_loss: 44000.0
    take_profit: 48000.0
    status: "open"
    opened_at: "2024-01-01T00:00:00Z"

# Orders
orders:
  - id: "uuid"
    user_id: "uuid"
    strategy_id: "uuid"
    exchange_account_id: "uuid"
    position_id: "uuid"  # Optional
    symbol: "BTC/USDT"
    side: "buy"
    type: "limit"
    quantity: 0.1
    price: 45000.0
    status: "filled"
    placed_at: "2024-01-01T00:00:00Z"

# Memories
memories:
  - id: "uuid"
    user_id: "uuid"
    agent_name: "market_analyst"
    memory_key: "last_analysis"
    content: "Market is bullish..."
    metadata:
      timestamp: "2024-01-01T00:00:00Z"
      confidence: 0.85
```

## Best Practices

1. **Use UUIDs for IDs** - Generate consistent UUIDs for entities that need to reference each other
2. **Respect dependencies** - Use file numbering to ensure correct order
3. **Environment separation** - Keep dev/test/staging data separate
4. **Testnet credentials** - Always use testnet API keys in seeds
5. **No production data** - NEVER put real user data or production API keys in seeds
6. **Validate before applying** - Use `--dry-run` flag to validate YAML structure
7. **Document relationships** - Add comments explaining entity relationships

## Generating UUIDs

You can generate UUIDs using:

```bash
# macOS/Linux
uuidgen | tr '[:upper:]' '[:lower:]'

# Python
python3 -c "import uuid; print(uuid.uuid4())"

# Go
go run -c "fmt.Println(uuid.New())"
```

## Adding New Entity Types

1. Add the entity struct to `cmd/seeder/parser.go`
2. Add the builder method to the parser
3. Update the `Parse()` method to handle the new entity type
4. Add validation in the `Validate()` method
5. Create example YAML files in each environment directory
6. Update this README

## Troubleshooting

### "No seed files found"
- Check that the environment directory exists (`seeds/dev`, `seeds/test`, etc.)
- Ensure files have `.yaml` or `.yml` extension

### "Failed to create entity"
- Check UUID references are correct
- Verify dependencies are seeded first (follow numbering)
- Check required fields are present

### "Validation failed"
- Check YAML syntax is correct
- Verify UUIDs are in correct format
- Ensure all required fields are present

### Foreign key violations
- Ensure files are numbered correctly (01_, 02_, etc.)
- Verify referenced entities exist in previous files
- Check that you're using correct UUIDs for references
